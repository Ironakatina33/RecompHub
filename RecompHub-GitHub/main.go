//go:build windows
// RecompHub 1.3.0. Independent native Win32 interface: no PowerShell/.NET/WebView.
package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "sort"
 "sync"
 "strconv"
 "strings"
 "syscall"
 "time"
 "unicode/utf16"
 "unsafe"
)

var releaseRepo = "" // set by GitHub Actions: -X main.releaseRepo=owner/repo
var version = "1.3.0" // injected by GitHub Actions as well
const (
 WS_OVERLAPPEDWINDOW=0x00CF0000; WS_VISIBLE=0x10000000; WS_CHILD=0x40000000; WS_VSCROLL=0x00200000; WS_BORDER=0x00800000; WS_TABSTOP=0x00010000
 ES_AUTOHSCROLL=0x0080; LBS_OWNERDRAWFIXED=0x0010; LBS_HASSTRINGS=0x0040; LBS_NOTIFY=0x0001; BS_OWNERDRAW=0x000B
 WM_CREATE=1; WM_DESTROY=2; WM_SIZE=5; WM_CLOSE=16; WM_ERASEBKGND=20; WM_PAINT=15; WM_COMMAND=0x111; WM_DRAWITEM=0x2B; WM_MEASUREITEM=0x2C; WM_CTLCOLOREDIT=0x133; WM_CTLCOLORLISTBOX=0x134; WM_DROPFILES=0x233; WM_APP_UPDATE=0x8001
 LB_ADDSTRING=0x0180; LB_RESETCONTENT=0x0184; LB_GETCURSEL=0x0188; LB_SETCURSEL=0x0186; LB_SETITEMHEIGHT=0x01A0; LB_ERR=^uintptr(0)
 EM_SETSEL=0x00B1
 SW_SHOW=5; SW_HIDE=0; HWND_DESKTOP=0; GWLP_USERDATA=-21
 MB_ICONINFORMATION=0x40; MB_ICONERROR=0x10; MB_YESNO=4; MB_ICONQUESTION=0x20; IDYES=6
 COLOR_WINDOW=5; DT_SINGLELINE=0x20; DT_VCENTER=4; DT_END_ELLIPSIS=0x8000; DT_LEFT=0; DT_CENTER=1; TRANSPARENT=1
 ID_ALL=1001; ID_FAV=1002; ID_RECOMP=1003; ID_PORT=1004; ID_LAUNCHER=1005; ID_OTHER=1006
 ID_ADD=1010; ID_SCAN=1011; ID_SEARCH=1012; ID_LIST=1013; ID_PLAY=1014; ID_REMOVE=1015; ID_FAVORITE=1016; ID_CHANGE=1017; ID_DIR=1018; ID_UPDATE=1019
)

type rect struct { Left,Top,Right,Bottom int32 }
type point struct { X,Y int32 }
type msg struct { Hwnd uintptr; Message uint32; _ uint32; WParam,LParam uintptr; Time uint32; Pt point; LPrivate uint32 }
type wndClass struct { Style uint32; _ uint32; WndProc uintptr; ClsExtra,WndExtra int32; Instance,Icon,Cursor,Background uintptr; MenuName,ClassName *uint16 }
type paintStruct struct { HDC uintptr; Erase int32; RcPaint rect; Restore,IncUpdate int32; Reserved [32]byte }
type drawItem struct { CtlType,CtlID,ItemID,ItemAction,ItemState uint32; HwndItem,HDC uintptr; RcItem rect; ItemData uintptr }
type measureItem struct { CtlType,CtlID,ItemID,ItemWidth,ItemHeight uint32; ItemData uintptr }
type openFileName struct { StructSize uint32; _ uint32; HwndOwner,Instance uintptr; Filter,CustomFilter *uint16; MaxCustFilter,FilterIndex uint32; File *uint16; MaxFile uint32; FileTitle *uint16; MaxFileTitle uint32; InitialDir,Title *uint16; Flags uint32; FileOffset,FileExtension uint16; DefExt *uint16; CustData,Hook uintptr; TemplateName *uint16; Reserved uintptr; DwReserved,FlagsEx uint32 }
type browseInfo struct { HwndOwner,PidlRoot uintptr; DisplayName,Title *uint16; Flags uint32; _ uint32; Callback,LParam uintptr; Image int32; _ uint32 }

type Game struct { ID string `json:"id"`; Name string `json:"name"`; Path string `json:"path"`; Args string `json:"args"`; Category string `json:"category"`; Cover string `json:"cover"`; Notes string `json:"notes"`; Favorite bool `json:"favorite"`; Added string `json:"added"`; LastPlayed string `json:"lastPlayed"`; PlayCount int `json:"playCount"` }
type Settings struct { LastFolder string `json:"LastFolder"`; Sort string `json:"Sort"`; UpdateFeed string `json:"UpdateFeed"`; AutoCheck bool `json:"AutoCheck"`; LastUpdateCheck string `json:"LastUpdateCheck"` }
type Library struct { Version int `json:"version"`; Settings Settings `json:"settings"`; Games []Game `json:"games"` }
type Manifest struct { Version string `json:"version"`; URL string `json:"url"`; SHA256 string `json:"sha256"`; Notes string `json:"notes"` }
type UpdateResult struct { Info string; Manifest *Manifest; Err error }

var (
 user32=syscall.NewLazyDLL("user32.dll"); gdi32=syscall.NewLazyDLL("gdi32.dll"); kernel32=syscall.NewLazyDLL("kernel32.dll"); shell32=syscall.NewLazyDLL("shell32.dll"); comdlg32=syscall.NewLazyDLL("comdlg32.dll"); ole32=syscall.NewLazyDLL("ole32.dll"); dwmapi=syscall.NewLazyDLL("dwmapi.dll")
 pRegisterClass=user32.NewProc("RegisterClassW"); pCreateWindow=user32.NewProc("CreateWindowExW"); pDefWndProc=user32.NewProc("DefWindowProcW"); pShowWindow=user32.NewProc("ShowWindow"); pUpdateWindow=user32.NewProc("UpdateWindow"); pGetMessage=user32.NewProc("GetMessageW"); pTranslate=user32.NewProc("TranslateMessage"); pDispatch=user32.NewProc("DispatchMessageW"); pPostQuit=user32.NewProc("PostQuitMessage"); pSendMessage=user32.NewProc("SendMessageW"); pPostMessage=user32.NewProc("PostMessageW"); pMoveWindow=user32.NewProc("MoveWindow"); pInvalidate=user32.NewProc("InvalidateRect"); pGetClientRect=user32.NewProc("GetClientRect"); pMessageBox=user32.NewProc("MessageBoxW"); pBeginPaint=user32.NewProc("BeginPaint"); pEndPaint=user32.NewProc("EndPaint"); pDrawText=user32.NewProc("DrawTextW"); pSetWindowText=user32.NewProc("SetWindowTextW"); pGetWindowText=user32.NewProc("GetWindowTextW"); pGetWindowTextLength=user32.NewProc("GetWindowTextLengthW"); pSetFocus=user32.NewProc("SetFocus"); pEnableWindow=user32.NewProc("EnableWindow"); pSetBkMode=gdi32.NewProc("SetBkMode"); pSetTextColor=gdi32.NewProc("SetTextColor"); pFillRect=user32.NewProc("FillRect"); pSetBkColor=gdi32.NewProc("SetBkColor"); pCreateBrush=gdi32.NewProc("CreateSolidBrush"); pDeleteObject=gdi32.NewProc("DeleteObject"); pSelectObject=gdi32.NewProc("SelectObject"); pCreateFont=gdi32.NewProc("CreateFontW"); pGetModuleHandle=kernel32.NewProc("GetModuleHandleW"); pLoadCursor=user32.NewProc("LoadCursorW"); pSetClassLong=user32.NewProc("SetClassLongPtrW"); pShellExecute=shell32.NewProc("ShellExecuteW"); pGetOpenFile=comdlg32.NewProc("GetOpenFileNameW"); pBrowse=shell32.NewProc("SHBrowseForFolderW"); pPidlPath=shell32.NewProc("SHGetPathFromIDListW"); pCoTaskFree=ole32.NewProc("CoTaskMemFree"); pDragAccept=shell32.NewProc("DragAcceptFiles"); pDragQuery=shell32.NewProc("DragQueryFileW"); pDragFinish=shell32.NewProc("DragFinish"); pDwmSet=dwmapi.NewProc("DwmSetWindowAttribute")
 appHwnd,editHwnd,listHwnd uintptr
 controls=map[int]uintptr{}
 font,smallFont,largeFont uintptr
 brushDark,brushEdit uintptr
 pageWidth int32=1100; pageHeight int32=720
 lib Library; visible []int; filter="Tous les jeux"; search=""; appDir string
 errorLog string
 updateChan=make(chan UpdateResult,3)
 utfCache=map[string][]uint16{}; utfMu sync.Mutex
 checking bool; updating bool
)

func w(s string)*uint16 { p,_:=syscall.UTF16PtrFromString(s); return p }
func ptr(s string) uintptr { utfMu.Lock(); defer utfMu.Unlock(); b,ok:=utfCache[s]; if !ok { b=utf16.Encode([]rune(s)); b=append(b,0); utfCache[s]=b }; return uintptr(unsafe.Pointer(&b[0])) }
func rgb(r,g,b int)uintptr{return uintptr(uint32(r)|uint32(g)<<8|uint32(b)<<16)}
func call(p *syscall.LazyProc, a ...uintptr)uintptr{r,_,_:=p.Call(a...);return r}
func showMsg(title,body string,flags uintptr) bool { return call(pMessageBox,appHwnd,ptr(body),ptr(title),flags)==IDYES }
func fatal(err error){ _=os.MkdirAll(appDir,0755); _=os.WriteFile(errorLog,[]byte(time.Now().Format(time.RFC3339)+" "+err.Error()+"\n"),0644); showMsg("RecompHub - erreur",err.Error()+"\n\nJournal : "+errorLog,MB_ICONERROR) }
func create(class,title string,style uintptr,x,y,width,height int32,id int, parent uintptr)uintptr{
 h:=call(pCreateWindow,0,ptr(class),ptr(title),style,uintptr(x),uintptr(y),uintptr(width),uintptr(height),parent,uintptr(id),0,0)
 if id!=0 {controls[id]=h}; return h
}
func fontSet(h uintptr, f uintptr){call(pSendMessage,h,0x30,f,1)}
func btn(id int,title string){ h:=create("BUTTON",title,WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW,0,0,130,42,id,appHwnd);fontSet(h,font) }
func drawText(hdc uintptr,s string,r rect,sizeFont uintptr,col uintptr,flags uintptr){ old:=call(pSelectObject,hdc,sizeFont);call(pSetBkMode,hdc,TRANSPARENT);call(pSetTextColor,hdc,col);call(pDrawText,hdc,ptr(s),^uintptr(0),uintptr(unsafe.Pointer(&r)),flags); call(pSelectObject,hdc,old) }
func fill(hdc uintptr,r rect,color uintptr){b:=call(pCreateBrush,color);call(pFillRect,hdc,uintptr(unsafe.Pointer(&r)),b);call(pDeleteObject,b)}
func reposition(id int,x,y,wd,ht int32){if h:=controls[id];h!=0{call(pMoveWindow,h,uintptr(x),uintptr(y),uintptr(wd),uintptr(ht),1)}}
func layout(){
 w:=pageWidth;h:=pageHeight;if w<830 {w=830}; if h<520{h=520}
 sidebar:=int32(222);left:=sidebar+24; right:=w-24; listY:=int32(183);listBottom:=h-100
 for i,id:=range []int{ID_ALL,ID_FAV,ID_RECOMP,ID_PORT,ID_LAUNCHER,ID_OTHER}{reposition(id,18,139+int32(i)*56,188,43)}
 reposition(ID_SEARCH,left,98,right-left-345,39)
 reposition(ID_ADD,right-323,98,156,39);reposition(ID_SCAN,right-159,98,159,39)
 reposition(ID_LIST,left,listY,right-left,listBottom-listY)
 y:=h-79;gap:=int32(10); widths:=[]int32{114,108,118,130,118,148};ids:=[]int{ID_PLAY,ID_FAVORITE,ID_CHANGE,ID_DIR,ID_REMOVE,ID_UPDATE};x:=left
 for i,id:=range ids {reposition(id,x,y,widths[i],46);x+=widths[i]+gap}
 call(pInvalidate,appHwnd,0,1)
}
func paintBackground(hdc uintptr){
 fill(hdc,rect{0,0,pageWidth,pageHeight},rgb(13,19,32));fill(hdc,rect{0,0,223,pageHeight},rgb(19,30,50));fill(hdc,rect{223,0,225,pageHeight},rgb(34,50,76));
 drawText(hdc,"R  /  RECOMPHUB",rect{28,29,240,69},largeFont,rgb(126,172,255),DT_SINGLELINE|DT_VCENTER)
 drawText(hdc,"BIBLIOTHÈQUE",rect{27,103,202,133},smallFont,rgb(127,146,179),DT_SINGLELINE|DT_VCENTER)
 drawText(hdc,"JEUX  /  PORTS  /  RECOMPS",rect{246,29,pageWidth-50,67},largeFont,rgb(234,242,255),DT_SINGLELINE|DT_VCENTER)
 drawText(hdc,"Un seul endroit pour lancer tes jeux",rect{247,61,pageWidth-30,87},smallFont,rgb(146,161,185),DT_SINGLELINE|DT_VCENTER)
 drawText(hdc,fmt.Sprintf("%d jeux  •  %d affichés  •  v%s",len(lib.Games),len(visible),version),rect{245,pageHeight-113,pageWidth-24,pageHeight-90},smallFont,rgb(142,161,191),DT_SINGLELINE|DT_VCENTER)
 drawText(hdc,"Les jeux restent installés dans leurs dossiers d'origine",rect{26,pageHeight-120,210,pageHeight-46},smallFont,rgb(112,136,172),DT_LEFT)
}
func drawButton(d *drawItem){
 id:=int(d.CtlID);selected:=(id==ID_ALL&&filter=="Tous les jeux")||(id==ID_FAV&&filter=="Mes favoris")||(id==ID_RECOMP&&filter=="Recompilation")||(id==ID_PORT&&filter=="Port PC")||(id==ID_LAUNCHER&&filter=="Launcher")||(id==ID_OTHER&&filter=="Autre")
 bg:=rgb(35,51,75);fg:=rgb(223,232,250)
 if selected{bg=rgb(37,85,159);fg=rgb(255,255,255)}
 if id==ID_PLAY||id==ID_ADD||id==ID_SCAN {bg=rgb(35,97,202);fg=rgb(255,255,255)}
 if id==ID_REMOVE {bg=rgb(93,46,61)}
 if id==ID_UPDATE {bg=rgb(34,74,111)}
 if (d.ItemState&1)!=0 {bg=rgb(58,103,170)}
 fill(d.HDC,d.RcItem,bg)
 caption:=map[int]string{ID_ALL:"Tous les jeux",ID_FAV:"★  Favoris",ID_RECOMP:"Recompilations",ID_PORT:"Ports PC",ID_LAUNCHER:"Launchers",ID_OTHER:"Autres",ID_ADD:"＋  Ajouter un jeu",ID_SCAN:"▣  Scanner un dossier",ID_PLAY:"▶  Jouer",ID_FAVORITE:"☆  Favori",ID_CHANGE:"✎  Changer l'EXE",ID_DIR:"Ouvrir dossier",ID_REMOVE:"Retirer",ID_UPDATE:"↻  Mises a jour"}[id]
 if caption==""{caption="Action"}
 drawText(d.HDC,caption,d.RcItem,font,fg,DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_END_ELLIPSIS)
}
func drawRow(d *drawItem){
 i:=int(d.ItemID);if i<0||i>=len(visible){return};g:=lib.Games[visible[i]];r:=d.RcItem
 selected:=(d.ItemState&1)!=0
 if selected{fill(d.HDC,r,rgb(40,71,118))}else{fill(d.HDC,r,rgb(23,36,56))}
 left:=r.Left+15; right:=r.Right-12
 drawText(d.HDC,g.Name,rect{left,r.Top+7,right-170,r.Top+33},font,rgb(239,244,255),DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS)
 category:=g.Category;if category==""{category="Port PC"};if g.Favorite{category="★  "+category}
 drawText(d.HDC,category,rect{right-170,r.Top+7,right-7,r.Top+31},smallFont,rgb(120,180,251),DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS)
 path:=g.Path;if len(path)>200{path=path[:200]}
 col:=rgb(152,170,194);if _,err:=os.Stat(g.Path);err!=nil{col=rgb(255,158,139);path="Fichier introuvable  •  "+path}
 drawText(d.HDC,path,rect{left,r.Top+33,right-8,r.Top+59},smallFont,col,DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS)
 fill(d.HDC,rect{r.Left,r.Bottom-1,r.Right,r.Bottom},rgb(37,51,76))
}
func wndProc(hwnd uintptr,uMsg uint32,wparam,lparam uintptr)uintptr{
 switch uMsg {
 case WM_CREATE:
  appHwnd=hwnd
  attr:=int32(20);val:=int32(1);call(pDwmSet,hwnd,uintptr(attr),uintptr(unsafe.Pointer(&val)),4)
  font=call(pCreateFont,^uintptr(16),0,0,0,uintptr(400),0,0,0,1,0,0,5,0,ptr("Segoe UI"))
  smallFont=call(pCreateFont,^uintptr(14),0,0,0,uintptr(400),0,0,0,1,0,0,5,0,ptr("Segoe UI"))
  largeFont=call(pCreateFont,^uintptr(21),0,0,0,uintptr(700),0,0,0,1,0,0,5,0,ptr("Segoe UI"))
  brushDark=call(pCreateBrush,rgb(23,36,56));brushEdit=call(pCreateBrush,rgb(27,42,66))
  for _,b:=range []struct{id int;name string}{{ID_ALL,"Tous"},{ID_FAV,"Favoris"},{ID_RECOMP,"Recomp"},{ID_PORT,"Port"},{ID_LAUNCHER,"Launcher"},{ID_OTHER,"Autres"},{ID_ADD,"Ajouter"},{ID_SCAN,"Scanner"},{ID_PLAY,"Jouer"},{ID_FAVORITE,"Favori"},{ID_CHANGE,"Changer"},{ID_DIR,"Dossier"},{ID_REMOVE,"Retirer"},{ID_UPDATE,"Mise a jour"}}{btn(b.id,b.name)}
  editHwnd=create("EDIT","",WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL,0,0,300,38,ID_SEARCH,hwnd);fontSet(editHwnd,font)
  listHwnd=create("LISTBOX","",WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS|LBS_NOTIFY,0,0,400,300,ID_LIST,hwnd);fontSet(listHwnd,font);call(pSendMessage,listHwnd,LB_SETITEMHEIGHT,0,uintptr(66))
  call(pDragAccept,hwnd,1);layout();refreshList();return 0
 case WM_SIZE:
  pageWidth=int32(uint16(lparam));pageHeight=int32(uint16(lparam>>16));layout();return 0
 case WM_ERASEBKGND:return 1
 case WM_PAINT:
  var ps paintStruct;dc:=call(pBeginPaint,hwnd,uintptr(unsafe.Pointer(&ps)));paintBackground(dc);call(pEndPaint,hwnd,uintptr(unsafe.Pointer(&ps)));return 0
 case WM_MEASUREITEM:
  mi:=(*measureItem)(unsafe.Pointer(lparam));if mi.CtlID==ID_LIST{mi.ItemHeight=66;return 1}
 case WM_DRAWITEM:
  di:=(*drawItem)(unsafe.Pointer(lparam));if di.CtlID==ID_LIST{drawRow(di)}else{drawButton(di)};return 1
 case WM_CTLCOLOREDIT:
  call(pSetTextColor,wparam,rgb(239,244,255));call(pSetBkColor,wparam,rgb(27,42,66));return brushEdit
 case WM_CTLCOLORLISTBOX:
  call(pSetTextColor,wparam,rgb(239,244,255));call(pSetBkColor,wparam,rgb(23,36,56));return brushDark
 case WM_COMMAND:
  id:=int(wparam&0xFFFF);notification:=int((wparam>>16)&0xFFFF)
  if id==ID_SEARCH && notification==0x300 {search=getText(editHwnd);refreshList();return 0}
  if id==ID_LIST && notification==2 {launchSelected();return 0}
  if notification==0 || notification==5 || notification==1 {handleButton(id);return 0}
 case WM_DROPFILES:
  count:=call(pDragQuery,wparam,0xFFFFFFFF,0,0);num:=0
  for i:=uintptr(0);i<count;i++ { var buf [32768]uint16;call(pDragQuery,wparam,i,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)));path:=syscall.UTF16ToString(buf[:]);if isGameFile(path){if addGame(path){num++}} }
  call(pDragFinish,wparam);if num>0{saveLibrary();refreshList()};return 0
 case WM_APP_UPDATE:
  select {case result:=<-updateChan:
    checking=false;updating=false
    if result.Err!=nil{showMsg("Mises a jour",result.Err.Error(),MB_ICONINFORMATION)} else if result.Manifest!=nil {handleUpdate(result.Manifest)}else if result.Info!=""{showMsg("Mises a jour",result.Info,MB_ICONINFORMATION)}
  default:};return 0
 case WM_CLOSE:saveLibrary();call(user32.NewProc("DestroyWindow"),hwnd);return 0
 case WM_DESTROY:
  for _,h:=range []uintptr{font,smallFont,largeFont,brushDark,brushEdit}{if h!=0{call(pDeleteObject,h)}};call(pPostQuit,0);return 0
 }
 return call(pDefWndProc,hwnd,uintptr(uMsg),wparam,lparam)
}
func getText(h uintptr)string{n:=int(call(pGetWindowTextLength,h));if n>20000 {n=20000};buf:=make([]uint16,n+1);call(pGetWindowText,h,uintptr(unsafe.Pointer(&buf[0])),uintptr(n+1));return syscall.UTF16ToString(buf)}
func selectedIndex()int{i:=call(pSendMessage,listHwnd,LB_GETCURSEL,0,0);if i==LB_ERR||int(i)>=len(visible){return -1};return visible[int(i)]}
func selectedGame()*Game{i:=selectedIndex();if i<0{return nil};return &lib.Games[i]}
func refreshList(){
 if listHwnd==0{return};prev:="";if g:=selectedGame();g!=nil{prev=g.ID};visible=nil;call(pSendMessage,listHwnd,LB_RESETCONTENT,0,0)
 q:=strings.ToLower(strings.TrimSpace(search))
 for i,g:=range lib.Games {if filter!="Tous les jeux" {if filter=="Mes favoris"&&!g.Favorite{continue};if filter!="Mes favoris"&&g.Category!=filter{continue}}
  if q!=""&&!strings.Contains(strings.ToLower(g.Name+" "+g.Path+" "+g.Category),q){continue}; visible=append(visible,i)
 }
 sort.SliceStable(visible,func(i,j int)bool{return strings.ToLower(lib.Games[visible[i]].Name)<strings.ToLower(lib.Games[visible[j]].Name)})
 for i,ix:=range visible {call(pSendMessage,listHwnd,LB_ADDSTRING,0,ptr(lib.Games[ix].Name));if lib.Games[ix].ID==prev{call(pSendMessage,listHwnd,LB_SETCURSEL,uintptr(i),0)}}
 call(pInvalidate,appHwnd,0,0)
}
func setFilter(s string){filter=s;refreshList();for _,id:=range []int{ID_ALL,ID_FAV,ID_RECOMP,ID_PORT,ID_LAUNCHER,ID_OTHER}{call(pInvalidate,controls[id],0,1)}}
func handleButton(id int){
 switch id {
 case ID_ALL:setFilter("Tous les jeux")
 case ID_FAV:setFilter("Mes favoris")
 case ID_RECOMP:setFilter("Recompilation")
 case ID_PORT:setFilter("Port PC")
 case ID_LAUNCHER:setFilter("Launcher")
 case ID_OTHER:setFilter("Autre")
 case ID_ADD:if p:=pickFile();p!=""{if addGame(p){saveLibrary();refreshList()}}
 case ID_SCAN:scanFolder()
 case ID_PLAY:launchSelected()
 case ID_REMOVE:if i:=selectedIndex();i>=0&&showMsg("Retirer ce jeu ?","Retirer « "+lib.Games[i].Name+" » de la liste ?\nLe jeu ne sera pas supprimé du disque.",MB_YESNO|MB_ICONQUESTION){lib.Games=append(lib.Games[:i],lib.Games[i+1:]...);saveLibrary();refreshList()}
 case ID_FAVORITE:if g:=selectedGame();g!=nil{g.Favorite=!g.Favorite;saveLibrary();refreshList()}
 case ID_CHANGE:if g:=selectedGame();g!=nil{if p:=pickFile();p!=""{g.Path=p;saveLibrary();refreshList()}}
 case ID_DIR:if g:=selectedGame();g!=nil{openShell(filepath.Dir(g.Path),"")}
 case ID_UPDATE:checkForUpdates(true)
 }
}
func isGameFile(path string)bool {switch strings.ToLower(filepath.Ext(path)){case ".exe",".bat",".cmd",".lnk",".jar":return true};return false}
func pretty(path string)string {folder:=filepath.Base(filepath.Dir(path));low:=strings.ToLower(folder);switch low{case "release","releases","bin","build","builds","debug","windows","win64","x64","dist","output":folder=filepath.Base(filepath.Dir(filepath.Dir(path)))};if folder=="."||folder==""{folder=strings.TrimSuffix(filepath.Base(path),filepath.Ext(path))};return strings.ReplaceAll(strings.ReplaceAll(folder,"_"," "),"-"," ")}
func category(path string)string{p:=strings.ToLower(path);switch{case strings.Contains(p,"recomp")||strings.Contains(p,"savagewind"):return "Recompilation";case strings.Contains(p,"launcher")||strings.Contains(p,"retroarch")||strings.Contains(p,"playnite"):return "Launcher";default:return "Port PC"}}
func addGame(path string)bool{path,err:=filepath.Abs(path);if err!=nil{return false};for _,g:=range lib.Games{if strings.EqualFold(g.Path,path){return false}};lib.Games=append(lib.Games,Game{ID:fmt.Sprintf("%d-%d",time.Now().UnixNano(),len(lib.Games)),Name:pretty(path),Path:path,Category:category(path),Added:time.Now().Format(time.RFC3339)});return true}
func openShell(path,args string)error {p:=filepath.Dir(path);if _,err:=os.Stat(path);err!=nil{return err};result:=call(pShellExecute,appHwnd,ptr("open"),ptr(path),ptr(args),ptr(p),SW_SHOW);if result<=32{return fmt.Errorf("Windows n'a pas pu ouvrir ce fichier (code %d)",result)};return nil}
func launchSelected(){g:=selectedGame();if g==nil{showMsg("Choisis un jeu","Sélectionne un jeu dans la liste puis clique sur Jouer.",MB_ICONINFORMATION);return};if err:=openShell(g.Path,g.Args);err!=nil{showMsg("Impossible de lancer le jeu",err.Error()+"\n\n"+g.Path,MB_ICONERROR);return};g.PlayCount++;g.LastPlayed=time.Now().Format(time.RFC3339);saveLibrary()}
func pickFile()string{
 var buffer [32768]uint16
 pattern:=utf16.Encode([]rune("Jeux et raccourcis\x00*.exe;*.bat;*.cmd;*.lnk;*.jar\x00Tous les fichiers\x00*.*\x00\x00"))
 title:=w("Sélectionne le programme à ajouter")
 ofn:=openFileName{StructSize:uint32(unsafe.Sizeof(openFileName{})),HwndOwner:appHwnd,Filter:&pattern[0],File:&buffer[0],MaxFile:uint32(len(buffer)),Title:title,Flags:0x00001000|0x00080000|0x00000004}
 if call(pGetOpenFile,uintptr(unsafe.Pointer(&ofn)))!=0{return syscall.UTF16ToString(buffer[:])};return ""
}
func browseFolder()string{
 var display [260]uint16;title:=w("Choisis le dossier contenant tes ports et recompilations")
 bi:=browseInfo{HwndOwner:appHwnd,DisplayName:&display[0],Title:title,Flags:0x00000040|0x00000001}
 pidl:=call(pBrowse,uintptr(unsafe.Pointer(&bi)));if pidl==0{return ""};defer call(pCoTaskFree,pidl)
 var folder [32768]uint16;if call(pPidlPath,pidl,uintptr(unsafe.Pointer(&folder[0])))==0{return ""};return syscall.UTF16ToString(folder[:])
}
func scanFolder(){root:=browseFolder();if root==""{return};lib.Settings.LastFolder=root;count:=0;visited:=0
 ignoreDirs:=map[string]bool{".git":true,"node_modules":true,"_commonredist":true,"redist":true,"__pycache__":true,".venv":true,"cache":true,"logs":true}
 var walk func(string,int)
 walk=func(dir string,depth int){if depth>5||visited>1800{return};visited++;entries,err:=os.ReadDir(dir);if err!=nil{return};var candidate string
  for _,e:=range entries {if e.IsDir(){continue};file:=e.Name();if !strings.EqualFold(filepath.Ext(file),".exe"){continue};lower:=strings.ToLower(strings.TrimSuffix(file,filepath.Ext(file)));if strings.Contains(lower,"unins")||strings.Contains(lower,"setup")||strings.Contains(lower,"install")||strings.Contains(lower,"update")||strings.Contains(lower,"crash")||strings.Contains(lower,"vcredist")||strings.Contains(lower,"redist"){continue};candidate=filepath.Join(dir,file);break}
  if candidate!=""&&addGame(candidate){count++}
  for _,e:=range entries{if e.IsDir()&&!ignoreDirs[strings.ToLower(e.Name())]{walk(filepath.Join(dir,e.Name()),depth+1)}}
 }
 walk(root,0);saveLibrary();refreshList();showMsg("Scan terminé",fmt.Sprintf("%d jeux ajoutés à ta bibliothèque.\n%d dossiers examinés.",count,visited),MB_ICONINFORMATION)
}
func loadLibrary(){lib=Library{Version:1,Games:[]Game{},Settings:Settings{Sort:"Nom (A-Z)",AutoCheck:true}};path:=filepath.Join(appDir,"library.json");bytes,err:=os.ReadFile(path);if err!=nil{return};if err:=json.Unmarshal(bytes,&lib);err!=nil{_ =os.WriteFile(filepath.Join(appDir,"library-backup-invalid.json"),bytes,0644);lib=Library{Version:1,Games:[]Game{}}};if lib.Games==nil{lib.Games=[]Game{}};for i:=range lib.Games{if lib.Games[i].Category==""{lib.Games[i].Category="Port PC"}}}
func saveLibrary(){if appDir==""{return};_ =os.MkdirAll(appDir,0755);lib.Version=1;bytes,err:=json.MarshalIndent(lib,"","  ");if err!=nil{return};tmp:=filepath.Join(appDir,"library.json.tmp");if err:=os.WriteFile(tmp,bytes,0644);err!=nil{return};target:=filepath.Join(appDir,"library.json");_ =os.Remove(target+".old");if _,err:=os.Stat(target);err==nil{_ =os.Rename(target,target+".old")};if err:=os.Rename(tmp,target);err!=nil{_ =os.Rename(target+".old",target)}}
func feedURL()string {if releaseRepo!=""&&strings.Count(releaseRepo,"/")==1{return "https://github.com/"+releaseRepo+"/releases/latest/download/latest.json"};return strings.TrimSpace(lib.Settings.UpdateFeed)}
func checkForUpdates(manual bool){if checking||updating{return};feed:=feedURL();if feed==""{if manual{showMsg("Mises a jour","Le programme est prêt pour les mises à jour, mais aucune adresse de publication GitHub n'est encore configurée.\n\nL'application fonctionne sans GitHub. Les mises à jour en ligne seront activées dès sa première publication officielle.",MB_ICONINFORMATION)};return}
 checking=true;go func(){cl:=http.Client{Timeout:18*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{if req.URL.Scheme!="https"{return errors.New("redirection non sécurisée")};if len(via)>5{return errors.New("trop de redirections")};return nil}};if !strings.HasPrefix(feed,"https://"){updateChan<-UpdateResult{Err:errors.New("Adresse non HTTPS")};call(pPostMessage,appHwnd,WM_APP_UPDATE,0,0);return};res,err:=cl.Get(feed);if err!=nil{updateChan<-UpdateResult{Err:err};call(pPostMessage,appHwnd,WM_APP_UPDATE,0,0);return};defer res.Body.Close();if res.StatusCode!=200{err=fmt.Errorf("GitHub n'a pas renvoyé de version (HTTP %d)",res.StatusCode)};var man Manifest;if err==nil{data,e:=io.ReadAll(io.LimitReader(res.Body,65537));if e!=nil{err=e}else if len(data)>65536{err=errors.New("manifeste trop volumineux")}else{err=json.Unmarshal(data,&man)}};result:=UpdateResult{};if err!=nil{result.Err=err}else if compareVersion(man.Version,version)>0{result.Manifest=&man}else{result.Info="Tu utilises déjà la dernière version ("+version+")."};if !manual&&result.Manifest==nil{return};updateChan<-result;call(pPostMessage,appHwnd,WM_APP_UPDATE,0,0)}()
}
func compareVersion(a,b string)int{aa:=strings.Split(a,".");bb:=strings.Split(b,".");for i:=0;i<3;i++{x,y:=0,0;if i<len(aa){x,_=strconv.Atoi(aa[i])};if i<len(bb){y,_=strconv.Atoi(bb[i])};if x>y{return 1};if x<y{return -1}};return 0}
func handleUpdate(man *Manifest){if !showMsg("Nouvelle version disponible","RecompHub "+man.Version+" est disponible.\n\nTélécharger et installer automatiquement ?\nTa bibliothèque sera conservée.",MB_YESNO|MB_ICONQUESTION){return};if len(man.SHA256)!=64||!strings.HasPrefix(man.URL,"https://"){showMsg("Mise a jour refusée","Le manifeste de mise à jour n'est pas valide.",MB_ICONERROR);return};updating=true;go func(){err:=downloadAndApply(*man);if err!=nil{_ =os.WriteFile(filepath.Join(appDir,"update-error.log"),[]byte(err.Error()),0644);updateChan<-UpdateResult{Err:err};call(pPostMessage,appHwnd,WM_APP_UPDATE,0,0)}}()}
func downloadAndApply(man Manifest)error {cl:=http.Client{Timeout:3*time.Minute};res,err:=cl.Get(man.URL);if err!=nil{return err};defer res.Body.Close();if res.StatusCode!=200{return fmt.Errorf("telechargement HTTP %d",res.StatusCode)};if res.Request.URL.Scheme!="https"{return errors.New("redirection non sécurisée")};if res.ContentLength>100<<20{return errors.New("fichier anormalement volumineux")}
 dir:=filepath.Join(appDir,"updates");if err:=os.MkdirAll(dir,0755);err!=nil{return err};staged:=filepath.Join(dir,"RecompHub-next.exe");f,err:=os.Create(staged);if err!=nil{return err};hash:=sha256.New();_,err=io.Copy(io.MultiWriter(f,hash),io.LimitReader(res.Body,100<<20));f.Close();if err!=nil{return err};if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)),man.SHA256){return errors.New("SHA-256 incorrect : mise à jour annulée")};actual,err:=os.Executable();if err!=nil{return err};updater:=filepath.Join(dir,"RecompHub-Updater.exe");exeData,err:=os.ReadFile(actual);if err!=nil{return err};if err=os.WriteFile(updater,exeData,0755);err!=nil{return err};cmd:=exec.Command(updater,"--apply",staged,actual,strconv.Itoa(os.Getpid()));if err=cmd.Start();err!=nil{return err};_ =cmd.Process.Release();call(pPostMessage,appHwnd,WM_CLOSE,0,0);return nil}
func applyUpdate(args []string){if len(args)<5{return};staged,target:=args[2],args[3];pid,_:=strconv.Atoi(args[4]);_ =pid;backup:=target+".backup";var err error;for i:=0;i<75;i++{time.Sleep(400*time.Millisecond);_ =os.Remove(backup);err=os.Rename(target,backup);if err==nil{err=os.Rename(staged,target);if err!=nil{_ =os.Rename(backup,target)};break}};if err!=nil{showMsg("Mise a jour","Installation échouée : "+err.Error(),MB_ICONERROR);return};_ =exec.Command(target).Start()}
func main(){runtime.LockOSThread();appDir=filepath.Join(os.Getenv("LOCALAPPDATA"),"RecompHub");if os.Getenv("LOCALAPPDATA")==""{appDir=filepath.Join(os.TempDir(),"RecompHub")};errorLog=filepath.Join(appDir,"startup-error.log");defer func(){if x:=recover();x!=nil{fatal(fmt.Errorf("plantage inattendu : %v",x))}}();if len(os.Args)>1&&os.Args[1]=="--apply"{applyUpdate(os.Args);return};if err:=os.MkdirAll(appDir,0755);err!=nil{fatal(err);return};_ = os.WriteFile(filepath.Join(appDir,"startup.log"),[]byte(time.Now().Format(time.RFC3339)+" : processus natif RecompHub lance\n"),0644);loadLibrary();call(ole32.NewProc("CoInitializeEx"),0,2);defer call(ole32.NewProc("CoUninitialize"))
 clsName:=w("RecompHubNativeApp13");hInstance:=call(pGetModuleHandle,0)
 wc:=wndClass{Style:3,WndProc:syscall.NewCallback(wndProc),Instance:hInstance,Cursor:call(pLoadCursor,0,32512),ClassName:clsName}
 if call(pRegisterClass,uintptr(unsafe.Pointer(&wc)))==0{fatal(errors.New("Windows n'a pas pu créer la fenêtre de RecompHub"));return}
 hwnd:=call(pCreateWindow,0,uintptr(unsafe.Pointer(clsName)),ptr("RecompHub — Bibliothèque de jeux"),WS_OVERLAPPEDWINDOW|WS_VISIBLE,100,80,1160,760,0,0,hInstance,0)
 if hwnd==0{fatal(errors.New("La fenêtre principale n'a pas pu s'ouvrir"));return};appHwnd=hwnd
 _ = os.WriteFile(filepath.Join(appDir,"startup.log"),[]byte(time.Now().Format(time.RFC3339)+" : fenetre creee correctement\n"),0644);call(pShowWindow,hwnd,SW_SHOW);call(pUpdateWindow,hwnd)
 if feedURL()!=""&&lib.Settings.AutoCheck{go func(){time.Sleep(3*time.Second);checkForUpdates(false)}()}
 var m msg;for {ret:=int32(call(pGetMessage,uintptr(unsafe.Pointer(&m)),0,0,0));if ret<=0{break};call(pTranslate,uintptr(unsafe.Pointer(&m)));call(pDispatch,uintptr(unsafe.Pointer(&m)))}
}
