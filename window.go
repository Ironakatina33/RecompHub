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

// RecompHub's Edge app window has its native non-client caption removed. The
// remaining WS_THICKFRAME keeps the normal Windows resize and snap mechanics.
// All window operations are restricted to the current unique app-mode title.
var appWindow atomic.Uintptr
var enumWindows = user32.NewProc("EnumWindows")
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
var isWindow = user32.NewProc("IsWindow")

const wsCaption = uintptr(0x00C00000)
const wsThickFrame = uintptr(0x00040000)
const wsSysMenu = uintptr(0x00080000)
const wsPopup = uintptr(0x80000000)
const swpFrameChanged = uintptr(0x0020)
const swpNoMove = uintptr(0x0002)
const swpNoSize = uintptr(0x0001)
const swpNoZOrder = uintptr(0x0004)
const swpNoActivate = uintptr(0x0010)

func attachWindow(){
 for i:=0;i<150;i++{
  if shuttingDown.Load(){return}
  var target uintptr
  cb:=syscall.NewCallback(func(hwnd,_ uintptr)uintptr{
   n,_,_:=getWindowTextLen.Call(hwnd);if n<1||n>300{return 1}
   var title[310]uint16;getWindowText.Call(hwnd,uintptr(unsafe.Pointer(&title[0])),310)
   t:=syscall.UTF16ToString(title[:]);if !strings.HasPrefix(t,"RecompHub v4"){return 1}
   var class[160]uint16;getClassName.Call(hwnd,uintptr(unsafe.Pointer(&class[0])),160)
   if syscall.UTF16ToString(class[:])!="Chrome_WidgetWin_1"{return 1}
   target=hwnd;return 0
  })
  enumWindows.Call(cb,0)
  if target!=0{
   idx:=int32(-16)
   old,_,_:=getWindowStyle.Call(target,uintptr(idx))
   // Chrome responds more consistently when the caption and system menu are
   // both omitted while WS_THICKFRAME and WS_POPUP remain. Keep the normal
   // min/max flags, if any, for Windows keyboard shortcuts and snapping.
   style:=(old &^ (wsCaption|wsSysMenu)) | wsThickFrame | wsPopup
   if old!=style {
    setWindowStyle.Call(target,uintptr(idx),style)
    setWindowPos.Call(target,0,0,0,0,0,swpFrameChanged|swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate)
   }
   current,_,_:=getWindowStyle.Call(target,uintptr(idx))
   if current&wsCaption==0 {appWindow.Store(target)}
   return
  }
  time.Sleep(130*time.Millisecond)
 }
}
func windowHandle()uintptr{hwnd:=appWindow.Load();if hwnd!=0{valid,_,_:=isWindow.Call(hwnd);if valid!=0{return hwnd}};return 0}
func apiWindow(w http.ResponseWriter,r *http.Request){
 hwnd:=windowHandle()
 if r.URL.Path=="/api/window/status"{
  z:=false;if hwnd!=0{x,_,_:=isZoomed.Call(hwnd);z=x!=0}
  jsonOut(w,map[string]any{"ok":true,"custom":hwnd!=0,"maximized":z});return
 }
 if r.Method!="POST"{fail(w,405,"POST requis");return}
 if hwnd==0{fail(w,409,"La barre personnalisée n’est pas disponible. Utilise les contrôles Windows.");return}
 var body struct {Action string `json:"action"`}
 if err:=reqBody(r,&body);err!=nil{fail(w,400,"Requête invalide");return}
 switch body.Action{
 case "minimize":showWindow.Call(hwnd,6)
 case "maximize":z,_,_:=isZoomed.Call(hwnd);if z!=0{showWindow.Call(hwnd,9)}else{showWindow.Call(hwnd,3)}
 case "close":postMessage.Call(hwnd,0x0010,0,0)
 case "drag":go func(){releaseCapture.Call();sendMessage.Call(hwnd,0x00A1,2,0)}()
 default:fail(w,400,"Action inconnue");return
 }
 jsonOut(w,map[string]any{"ok":true})
}
