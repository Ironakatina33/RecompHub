//go:build windows

// RecompHub v2 - Windows launcher with five console-inspired themes.
// Native Go backend, local-only API, and Edge app-mode window.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

//go:embed assets/*
var assets embed.FS

var version = "2.0.0"
var releaseRepo = "Ironakatina33/RecompHub"

const maxSize = 100 << 20

type Game struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Args       string `json:"args"`
	Category   string `json:"category"`
	Cover      string `json:"cover"`
	Notes      string `json:"notes"`
	Favorite   bool   `json:"favorite"`
	Added      string `json:"added"`
	LastPlayed string `json:"lastPlayed"`
	PlayCount  int    `json:"playCount"`
}
type Settings struct {
	LastFolder      string `json:"LastFolder"`
	Sort            string `json:"Sort"`
	UpdateFeed      string `json:"UpdateFeed"`
	AutoCheck       bool   `json:"AutoCheck"`
	LastUpdateCheck string `json:"LastUpdateCheck"`
	Theme           string `json:"Theme"`
}
type Library struct {
	Version  int      `json:"version"`
	Settings Settings `json:"settings"`
	Games    []Game   `json:"games"`
}
type Manifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Notes   string `json:"notes"`
}

var state struct {
	sync.Mutex
	lib    Library
	dir    string
	token  string
	origin string
}
var logFile *os.File
var browserProcess *os.Process
var shuttingDown atomic.Bool

func defaults() Library {
	return Library{Version: 1, Settings: Settings{Sort: "Nom (A-Z)", AutoCheck: true, Theme: "neon"}, Games: []Game{}}
}
func saveLocked() error {
	state.lib.Version = 1
	if err := os.MkdirAll(state.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state.lib, "", "  ")
	if err != nil {
		return err
	}
	target := filepath.Join(state.dir, "library.json")
	temp := target + ".tmp"
	if err := os.WriteFile(temp, data, 0600); err != nil {
		return err
	}
	if _, err := os.Stat(target); err == nil { // Keep a backup, but never delete the only working library before staging a copy.
		original, _ := os.ReadFile(target)
		if len(original) > 0 {
			_ = os.WriteFile(target+".old", original, 0600)
		}
	}
	return os.Rename(temp, target)
}
func loadLibrary() {
	state.lib = defaults()
	data, err := os.ReadFile(filepath.Join(state.dir, "library.json"))
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &state.lib); err != nil {
		_ = os.WriteFile(filepath.Join(state.dir, "library-corrupt-backup.json"), data, 0600)
		state.lib = defaults()
		return
	}
	if state.lib.Games == nil {
		state.lib.Games = []Game{}
	}
	if !validTheme(state.lib.Settings.Theme) {
		state.lib.Settings.Theme = "neon"
	}
	for i := range state.lib.Games {
		if state.lib.Games[i].Category == "" {
			state.lib.Games[i].Category = "Port PC"
		}
	}
}
func validTheme(s string) bool {
	switch s {
	case "ps3", "ps4", "wii", "nintendo", "neon":
		return true
	}
	return false
}
func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	jsonOut(w, map[string]any{"ok": false, "error": msg})
}
func reqBody(r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 64<<10)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(out)
}
func idForGame() string { var b [10]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }
func niceName(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	folder := filepath.Base(filepath.Dir(path))
	clean := strings.ToLower(folder)
	for _, s := range []string{"release", "bin", "build", "builds", "debug", "windows", "win64", "x64", "dist", "output"} {
		if clean == s {
			folder = filepath.Base(filepath.Dir(filepath.Dir(path)))
			break
		}
	}
	if strings.EqualFold(name, "game") || strings.EqualFold(name, "launcher") || strings.EqualFold(name, "start") {
		name = folder
	}
	name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
	return strings.TrimSpace(name)
}
func gameCategory(path string) string {
	low := strings.ToLower(path)
	if strings.Contains(low, "recomp") || strings.Contains(low, "savagewind") {
		return "Recompilation"
	}
	if strings.Contains(low, "launcher") || strings.Contains(low, "retroarch") || strings.Contains(low, "playnite") {
		return "Launcher"
	}
	return "Port PC"
}
func addGameLocked(path string) bool {
	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	if !isGame(path) {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	for _, g := range state.lib.Games {
		if strings.EqualFold(g.Path, path) {
			return false
		}
	}
	state.lib.Games = append(state.lib.Games, Game{ID: idForGame(), Name: niceName(path), Path: path, Category: gameCategory(path), Added: time.Now().Format(time.RFC3339)})
	return true
}
func isGame(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".bat", ".cmd", ".lnk", ".jar":
		return true
	}
	return false
}
func discoverGames(root string) (int, int) {
	ignore := map[string]bool{".git": true, "node_modules": true, "_commonredist": true, "redist": true, "__pycache__": true, ".venv": true, "cache": true, "logs": true, "temp": true}
	added, visited := 0, 0
	var walk func(string, int)
	walk = func(dir string, depth int) {
		if depth > 5 || visited >= 1800 {
			return
		}
		visited++
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		candidate := ""
		score := -999
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".exe") {
				continue
			}
			name := strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
			if strings.Contains(name, "unins") || strings.Contains(name, "setup") || strings.Contains(name, "install") || strings.Contains(name, "update") || strings.Contains(name, "crash") || strings.Contains(name, "vcredist") || strings.Contains(name, "redist") || strings.Contains(name, "config") || strings.Contains(name, "helper") || strings.Contains(name, "server") {
				continue
			}
			s := 0
			if strings.EqualFold(name, filepath.Base(dir)) {
				s += 12
			}
			if strings.Contains(name, "launcher") || strings.Contains(name, "game") {
				s += 3
			}
			if strings.Contains(name, "win64") {
				s += 1
			}
			if s > score {
				score = s
				candidate = filepath.Join(dir, e.Name())
			}
		}
		if candidate != "" {
			state.Lock()
			if addGameLocked(candidate) {
				added++
			}
			state.Unlock()
		}
		for _, e := range entries {
			if e.IsDir() && !ignore[strings.ToLower(e.Name())] {
				walk(filepath.Join(dir, e.Name()), depth+1)
			}
		}
	}
	walk(root, 0)
	return added, visited
}

// Windows-only native file/folder pickers, avoiding PowerShell, Python, .NET and a separate helper.
var shell32 = syscall.NewLazyDLL("shell32.dll")
var user32 = syscall.NewLazyDLL("user32.dll")
var comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
var ole32 = syscall.NewLazyDLL("ole32.dll")
var procShellExecute = shell32.NewProc("ShellExecuteW")

type openFileName struct {
	Size                   uint32
	Padding                uint32
	Owner, Instance        uintptr
	Filter, CustomFilter   *uint16
	MaxCustom, FilterIndex uint32
	File                   *uint16
	MaxFile                uint32
	FileTitle              *uint16
	MaxFileTitle           uint32
	InitialDir, Title      *uint16
	Flags                  uint32
	FileOffset, Extension  uint16
	DefExt                 *uint16
	CustomData, Hook       uintptr
	Template               *uint16
	Reserved               uintptr
	ReservedEx, FlagsEx    uint32
}
type browseInfo struct {
	Owner, Root         uintptr
	Display, Title      *uint16
	Flags               uint32
	Pad                 uint32
	Callback, Parameter uintptr
	Image               int32
	Pad2                uint32
}

func utf(s string) *uint16 { v, _ := syscall.UTF16PtrFromString(s); return v }
func pickFile(cover bool) string {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, _, _ = ole32.NewProc("CoInitializeEx").Call(0, 2)
	defer ole32.NewProc("CoUninitialize").Call()
	var buffer [32768]uint16
	filt := "Jeux et raccourcis\x00*.exe;*.bat;*.cmd;*.lnk;*.jar\x00Tous les fichiers\x00*.*\x00\x00"
	title := "Sélectionne le programme à ajouter"
	if cover {
		filt = "Images\x00*.png;*.jpg;*.jpeg;*.webp;*.bmp\x00Tous les fichiers\x00*.*\x00\x00"
		title = "Sélectionne une couverture"
	}
	pattern := utf16.Encode([]rune(filt))
	ofn := openFileName{Size: uint32(unsafe.Sizeof(openFileName{})), File: &buffer[0], MaxFile: uint32(len(buffer)), Filter: &pattern[0], Title: utf(title), Flags: 0x00001000 | 0x00080000 | 0x00000004}
	if r, _, _ := comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&ofn))); r != 0 {
		return syscall.UTF16ToString(buffer[:])
	}
	return ""
}
func pickFolder() string {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, _, _ = ole32.NewProc("CoInitializeEx").Call(0, 2)
	defer ole32.NewProc("CoUninitialize").Call()
	var display [260]uint16
	bi := browseInfo{Display: &display[0], Title: utf("Choisis le dossier de tes jeux"), Flags: 0x40 | 1}
	p, _, _ := shell32.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&bi)))
	if p == 0 {
		return ""
	}
	defer ole32.NewProc("CoTaskMemFree").Call(p)
	var folder [32768]uint16
	r, _, _ := shell32.NewProc("SHGetPathFromIDListW").Call(p, uintptr(unsafe.Pointer(&folder[0])))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(folder[:])
}
func shellOpen(path, args string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	p := filepath.Dir(path)
	r, _, _ := procShellExecute.Call(0, uintptr(unsafe.Pointer(utf("open"))), uintptr(unsafe.Pointer(utf(path))), uintptr(unsafe.Pointer(utf(args))), uintptr(unsafe.Pointer(utf(p))), 5)
	if r <= 32 {
		return fmt.Errorf("Windows n'a pas pu ouvrir le fichier (code %d)", r)
	}
	return nil
}
func showError(msg string) {
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(utf(msg))), uintptr(unsafe.Pointer(utf("RecompHub"))), 0x10)
}

func findGameLocked(id string) *Game {
	for i := range state.lib.Games {
		if state.lib.Games[i].ID == id {
			return &state.lib.Games[i]
		}
	}
	return nil
}
func api(w http.ResponseWriter, r *http.Request) {
	// Host, token and Origin protect the loopback API against unrelated web pages.
	if r.Host != strings.TrimPrefix(state.origin, "http://") {
		fail(w, 403, "Hôte interdit")
		return
	}
	if r.Header.Get("X-RecompHub-Token") != state.token && !(r.URL.Path == "/api/cover" && r.URL.Query().Get("token") == state.token) {
		fail(w, 403, "Requête non autorisée")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != state.origin {
		fail(w, 403, "Origine interdite")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		fail(w, 405, "Méthode non autorisée")
		return
	}
	switch r.URL.Path {
	case "/api/library":
		state.Lock()
		v := state.lib
		v.Games = append([]Game{}, state.lib.Games...)
		state.Unlock()
		jsonOut(w, map[string]any{"ok": true, "library": v, "version": version, "repo": releaseRepo})
	case "/api/add":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		path := pickFile(false)
		if path == "" {
			jsonOut(w, map[string]any{"ok": true, "cancelled": true})
			return
		}
		state.Lock()
		added := addGameLocked(path)
		err := saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true, "added": added})
	case "/api/scan":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		folder := pickFolder()
		if folder == "" {
			jsonOut(w, map[string]any{"ok": true, "cancelled": true})
			return
		}
		added, visited := discoverGames(folder)
		state.Lock()
		state.lib.Settings.LastFolder = folder
		err := saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true, "added": added, "visited": visited})
	case "/api/play":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		state.Lock()
		g := findGameLocked(body.ID)
		if g == nil {
			state.Unlock()
			fail(w, 404, "Jeu introuvable")
			return
		}
		path, args := g.Path, g.Args
		state.Unlock()
		if err := shellOpen(path, args); err != nil {
			fail(w, 400, err.Error())
			return
		}
		state.Lock()
		if g = findGameLocked(body.ID); g != nil {
			g.PlayCount++
			g.LastPlayed = time.Now().Format(time.RFC3339)
			_ = saveLocked()
		}
		state.Unlock()
		jsonOut(w, map[string]any{"ok": true})
	case "/api/favorite":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		state.Lock()
		g := findGameLocked(body.ID)
		if g == nil {
			state.Unlock()
			fail(w, 404, "Jeu introuvable")
			return
		}
		g.Favorite = !g.Favorite
		err := saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/remove":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		state.Lock()
		removed := false
		for i, g := range state.lib.Games {
			if g.ID == body.ID {
				state.lib.Games = append(state.lib.Games[:i], state.lib.Games[i+1:]...)
				removed = true
				break
			}
		}
		err := saveLocked()
		state.Unlock()
		if !removed {
			fail(w, 404, "Jeu introuvable")
			return
		}
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/edit":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct{ ID, Name, Category, Args, Notes string }
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		if len(body.Name) > 140 || len(body.Args) > 2048 || len(body.Notes) > 4096 || strings.TrimSpace(body.Name) == "" {
			fail(w, 400, "Nom ou arguments incorrects")
			return
		}
		switch body.Category {
		case "Recompilation", "Port PC", "Launcher", "Autre":
		default:
			fail(w, 400, "Catégorie inconnue")
			return
		}
		state.Lock()
		g := findGameLocked(body.ID)
		if g == nil {
			state.Unlock()
			fail(w, 404, "Jeu introuvable")
			return
		}
		g.Name = strings.TrimSpace(body.Name)
		g.Category = body.Category
		g.Args = body.Args
		g.Notes = body.Notes
		err := saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/cover/pick":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		path := pickFile(true)
		if path == "" {
			jsonOut(w, map[string]any{"ok": true, "cancelled": true})
			return
		}
		stat, err := os.Stat(path)
		if err != nil || stat.Size() > 12<<20 {
			fail(w, 400, "Image invalide (maximum 12 Mo)")
			return
		}
		state.Lock()
		g := findGameLocked(body.ID)
		if g == nil {
			state.Unlock()
			fail(w, 404, "Jeu introuvable")
			return
		}
		g.Cover = path
		err = saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/cover":
		id := r.URL.Query().Get("id")
		state.Lock()
		g := findGameLocked(id)
		path := ""
		if g != nil {
			path = g.Cover
		}
		state.Unlock()
		ext := strings.ToLower(filepath.Ext(path))
		if path == "" || !(ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".bmp") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		http.ServeFile(w, r, path)
	case "/api/reveal":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		state.Lock()
		g := findGameLocked(body.ID)
		if g == nil {
			state.Unlock()
			fail(w, 404, "Jeu introuvable")
			return
		}
		p := g.Path
		state.Unlock()
		if err := exec.Command("explorer.exe", "/select,", p).Start(); err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/settings":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body struct {
			Theme     string `json:"theme"`
			AutoCheck *bool  `json:"autoCheck"`
		}
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		state.Lock()
		if body.Theme != "" {
			if !validTheme(body.Theme) {
				state.Unlock()
				fail(w, 400, "Thème inconnu")
				return
			}
			state.lib.Settings.Theme = body.Theme
		}
		if body.AutoCheck != nil {
			state.lib.Settings.AutoCheck = *body.AutoCheck
		}
		err := saveLocked()
		state.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true})
	case "/api/update/check":
		checkUpdates(w)
	case "/api/update/apply":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var body Manifest
		if err := reqBody(r, &body); err != nil {
			fail(w, 400, "Données invalides")
			return
		}
		if err := verifyManifest(body); err != nil {
			fail(w, 400, err.Error())
			return
		}
		if err := downloadAndStage(body); err != nil {
			fail(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true, "restarting": true})
		go func() {
			time.Sleep(time.Second)
			shuttingDown.Store(true)
			if browserProcess != nil {
				_ = browserProcess.Kill()
			}
			os.Exit(0)
		}()
	default:
		fail(w, 404, "Introuvable")
	}
}
func latestURL() string {
	return "https://github.com/" + releaseRepo + "/releases/latest/download/latest.json"
}
func secureClient() *http.Client {
	return &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return errors.New("redirection HTTPS requise")
		}
		if len(via) > 8 {
			return errors.New("trop de redirections")
		}
		return nil
	}}
}
func checkUpdates(w http.ResponseWriter) {
	if releaseRepo == "" {
		jsonOut(w, map[string]any{"ok": true, "available": false, "info": "Mises à jour non configurées"})
		return
	}
	res, err := secureClient().Get(latestURL())
	if err != nil {
		fail(w, 502, "GitHub inaccessible : "+err.Error())
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fail(w, 502, fmt.Sprintf("Aucune version GitHub publiée (HTTP %d)", res.StatusCode))
		return
	}
	var m Manifest
	if err := json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&m); err != nil {
		fail(w, 502, "Manifeste GitHub incorrect")
		return
	}
	if err := verifyManifest(m); err != nil {
		fail(w, 502, err.Error())
		return
	}
	jsonOut(w, map[string]any{"ok": true, "available": compareVersion(m.Version, version) > 0, "manifest": m, "current": version})
}
func compareVersion(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		v, w := 0, 0
		if i < len(as) {
			v, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			w, _ = strconv.Atoi(bs[i])
		}
		if v > w {
			return 1
		}
		if v < w {
			return -1
		}
	}
	return 0
}
func verifyManifest(m Manifest) error {
	if m.Version == "" || len(m.SHA256) != 64 {
		return errors.New("Informations de mise à jour incomplètes")
	}
	if _, err := hex.DecodeString(m.SHA256); err != nil {
		return errors.New("SHA256 invalide")
	}
	prefix := "https://github.com/" + releaseRepo + "/releases/download/"
	if !strings.HasPrefix(m.URL, prefix) || !strings.HasSuffix(strings.ToLower(m.URL), "/recomphub.exe") {
		return errors.New("URL de mise à jour non autorisée")
	}
	return nil
}
func downloadAndStage(m Manifest) error {
	client := secureClient()
	client.Timeout = 3 * time.Minute
	res, err := client.Get(m.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("téléchargement échoué (HTTP %d)", res.StatusCode)
	}
	if res.ContentLength > maxSize {
		return errors.New("mise à jour trop volumineuse")
	}
	dir := filepath.Join(state.dir, "updates")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	target := filepath.Join(dir, "RecompHub-next.exe")
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(res.Body, maxSize+1))
	_ = f.Close()
	if err != nil || n > maxSize {
		_ = os.Remove(target)
		return errors.New("téléchargement incomplet ou trop volumineux")
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), m.SHA256) {
		_ = os.Remove(target)
		return errors.New("Empreinte SHA256 incorrecte : mise à jour annulée")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	updateExe := filepath.Join(dir, "RecompHub-updater.exe")
	self, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	if err := os.WriteFile(updateExe, self, 0700); err != nil {
		return err
	}
	cmd := exec.Command(updateExe, "--apply", target, exe, strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
func applyUpdate(args []string) {
	if len(args) < 5 {
		return
	}
	source, destination := args[2], args[3]
	backup := destination + ".backup"
	var err error
	for i := 0; i < 100; i++ {
		time.Sleep(250 * time.Millisecond)
		_ = os.Remove(backup)
		err = os.Rename(destination, backup)
		if err == nil {
			err = os.Rename(source, destination)
			if err != nil {
				_ = os.Rename(backup, destination)
			}
			break
		}
	}
	if err != nil {
		showError("La mise à jour a échoué : " + err.Error())
		return
	}
	_ = exec.Command(destination).Start()
}

func locateEdge() (string, error) {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath("msedge.exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("chrome.exe"); err == nil {
		return p, nil
	}
	return "", errors.New("Microsoft Edge est nécessaire pour afficher l'interface moderne. Installe ou répare Microsoft Edge, puis relance RecompHub.")
}
func generateToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func run() error {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	state.dir = filepath.Join(appData, "RecompHub")
	if err := os.MkdirAll(state.dir, 0700); err != nil {
		return err
	}
	var err error
	logFile, err = os.OpenFile(filepath.Join(state.dir, "v2.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	log.Printf("RecompHub %s launched", version)
	state.token = generateToken()
	loadLibrary()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	state.origin = "http://" + listener.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		src, err := assets.ReadFile("assets/index.html")
		if err != nil {
			http.Error(w, "UI absente", 500)
			return
		}
		token, _ := json.Marshal(state.token)
		html := strings.ReplaceAll(string(src), "__BOOT_TOKEN__", string(token))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.WriteString(w, html)
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		data, err := assets.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(path) {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/", api)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %s", err)
		}
	}()
	edge, err := locateEdge()
	if err != nil {
		return err
	}
	url := state.origin + "/"
	profile := filepath.Join(state.dir, "web-profile")
	cmd := exec.Command(edge, "--app="+url, "--user-data-dir="+profile, "--window-size=1440,900", "--no-first-run", "--no-default-browser-check")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Microsoft Edge n'a pas démarré : %w", err)
	}
	browserProcess = cmd.Process
	log.Printf("ui started with %s", edge)
	err = cmd.Wait()
	if shuttingDown.Load() {
		return nil
	}
	return err
}
func main() {
	defer func() {
		if v := recover(); v != nil {
			showError(fmt.Sprintf("Échec du lancement : %v", v))
		}
	}()
	if len(os.Args) > 1 && os.Args[1] == "--apply" {
		applyUpdate(os.Args)
		return
	}
	if err := run(); err != nil {
		log.Printf("ERROR: %s", err)
		showError("Impossible d'ouvrir RecompHub :\n" + err.Error() + "\n\nJournal : %LOCALAPPDATA%\\RecompHub\\v2.log")
	}
}
