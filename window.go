//go:build windows

package main

import (
	"net/http"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var appWindow atomic.Uintptr
var findWindowProc = user32.NewProc("EnumWindows")
var getWindowText = user32.NewProc("GetWindowTextW")
var getWindowTextLen = user32.NewProc("GetWindowTextLengthW")
var getClassName = user32.NewProc("GetClassNameW")
var getWindowStyle = user32.NewProc("GetWindowLongPtrW")
var setWindowStyle = user32.NewProc("SetWindowLongPtrW")
var setWindowPos = user32.NewProc("SetWindowPos")
var showWindow = user32.NewProc("ShowWindow")
var isZoomed = user32.NewProc("IsZoomed")
var releaseCapture = user32.NewProc("ReleaseCapture")
var sendMessage = user32.NewProc("SendMessageW")
var postMessage = user32.NewProc("PostMessageW")

func attachWindow() {
	for attempt := 0; attempt < 70; attempt++ {
		if shuttingDown.Load() {
			return
		}
		var target uintptr
		cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
			var title [280]uint16
			n, _, _ := getWindowTextLen.Call(hwnd)
			if n == 0 || n >= 280 {
				return 1
			}
			_, _, _ = getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&title[0])), 280)
			t := syscall.UTF16ToString(title[:])
			if !strings.HasPrefix(t, "RecompHub v3") {
				return 1
			}
			var class [120]uint16
			_, _, _ = getClassName.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), 120)
			if syscall.UTF16ToString(class[:]) != "Chrome_WidgetWin_1" {
				return 1
			}
			target = hwnd
			return 0
		})
		findWindowProc.Call(cb, 0)
		if target != 0 {
			index := int32(-16)
			old, _, _ := getWindowStyle.Call(target, uintptr(index))
			// Remove Edge's native caption while preserving the resizable frame.
			style := old &^ uintptr(0x00C00000)
			if style != old {
				setWindowStyle.Call(target, uintptr(index), style)
				setWindowPos.Call(target, 0, 0, 0, 0, 0, 0x0027)
			}
			appWindow.Store(target)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
func apiWindow(w http.ResponseWriter, r *http.Request) {
	hwnd := appWindow.Load()
	if r.URL.Path == "/api/window/status" {
		zoom := false
		if hwnd != 0 {
			v, _, _ := isZoomed.Call(hwnd)
			zoom = v != 0
		}
		jsonOut(w, map[string]any{"ok": true, "custom": hwnd != 0, "maximized": zoom})
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "POST requis")
		return
	}
	if hwnd == 0 {
		fail(w, 409, "Barre de fenêtre personnalisée non disponible : utilisez les contrôles Windows")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := reqBody(r, &body); err != nil {
		fail(w, 400, "Requête invalide")
		return
	}
	switch body.Action {
	case "minimize":
		showWindow.Call(hwnd, 6)
	case "maximize":
		v, _, _ := isZoomed.Call(hwnd)
		if v != 0 {
			showWindow.Call(hwnd, 9)
		} else {
			showWindow.Call(hwnd, 3)
		}
	case "close":
		postMessage.Call(hwnd, 0x0010, 0, 0)
	case "drag":
		go func() { releaseCapture.Call(); sendMessage.Call(hwnd, 0x00A1, 2, 0) }()
	default:
		fail(w, 400, "Action inconnue")
		return
	}
	jsonOut(w, map[string]any{"ok": true})
}
