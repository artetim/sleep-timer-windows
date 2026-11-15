//go:build windows

// Sleep Timer for Windows.
// Pure Win32 GUI. No background services, administrator privileges, or external runtime required.
package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const (
	CW_USEDEFAULT     = 0x80000000
	WS_OVERLAPPED     = 0x00000000
	WS_CAPTION        = 0x00C00000
	WS_SYSMENU        = 0x00080000
	WS_MINIMIZEBOX    = 0x00020000
	WS_CHILD          = 0x40000000
	WS_VISIBLE        = 0x10000000
	WS_TABSTOP        = 0x00010000
	WS_CLIPCHILDREN   = 0x02000000
	ES_CENTER         = 0x0001
	ES_NUMBER         = 0x2000
	ES_AUTOHSCROLL    = 0x0080
	WM_CREATE         = 0x0001
	WM_COMMAND        = 0x0111
	WM_DRAWITEM       = 0x002B
	WM_DESTROY        = 0x0002
	WM_PAINT          = 0x000F
	WM_CLOSE          = 0x0010
	WM_ERASEBKGND     = 0x0014
	WM_CTLCOLOREDIT   = 0x0133
	WM_CTLCOLORSTATIC = 0x0138
	WM_TIMER          = 0x0113
	WM_SETFONT        = 0x0030
	WM_SETICON        = 0x0080
	BS_OWNERDRAW      = 0x000B
	ODS_SELECTED      = 0x0001
	ODS_DISABLED      = 0x0004
	ODS_FOCUS         = 0x0010
	ODS_HOTLIGHT      = 0x0040
	DT_LEFT           = 0x0000
	DT_CENTER         = 0x0001
	DT_VCENTER        = 0x0004
	DT_SINGLELINE     = 0x0020
	TRANSPARENT       = 1
	OPAQUE            = 2
	SW_SHOW           = 5
	MB_ICONERROR      = 0x10
	MB_OK             = 0x0
	COLOR_WINDOW      = 5
	IDC_ARROW         = 32512
	IMAGE_ICON        = 1
	LR_DEFAULTCOLOR   = 0
	DEFAULT_CHARSET   = 1
	CLEARTYPE_QUALITY = 5
	PS_SOLID          = 0
	SRCCOPY           = 0x00CC0020
	ICON_SMALL        = 0
	ICON_BIG          = 1
	WM_USER           = 0x0400
	EM_SETLIMITTEXT   = 0x00C5
	GWLP_USERDATA     = -21
	timerID           = 1
	ctrlMinutes       = 1001
	clientW           = 600
	clientH           = 570
)

type RECT struct{ Left, Top, Right, Bottom int32 }
type POINT struct{ X, Y int32 }
type PAINTSTRUCT struct {
	Hdc       uintptr
	Erase     int32
	Paint     RECT
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}
type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	Hdc        uintptr
	RcItem     RECT
	ItemData   uintptr
}
type MSG struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	Private        uint32
}
type WNDCLASSEX struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type box struct{ x, y, w, h int }

func (r box) contains(x, y int) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type button struct {
	id    int
	area  box
	label string
}

var presets = [...]button{
	{10, box{196, 329, 82, 48}, "10 min"},
	{30, box{287, 329, 82, 48}, "30 min"},
	{60, box{378, 329, 82, 48}, "60 min"},
	{120, box{469, 329, 102, 48}, "120 min"},
}
var startBtn = button{100, box{29, 406, 334, 55}, "Start timer"}
var cancelBtn = button{101, box{376, 406, 195, 55}, "Cancel"}

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	ntdll                   = syscall.NewLazyDLL("ntdll.dll")
	pRtlMoveMemory          = ntdll.NewProc("RtlMoveMemory")
	pRegisterClassEx        = user32.NewProc("RegisterClassExW")
	pCreateWindowEx         = user32.NewProc("CreateWindowExW")
	pDefWindowProc          = user32.NewProc("DefWindowProcW")
	pShowWindow             = user32.NewProc("ShowWindow")
	pUpdateWindow           = user32.NewProc("UpdateWindow")
	pGetMessage             = user32.NewProc("GetMessageW")
	pTranslateMessage       = user32.NewProc("TranslateMessage")
	pDispatchMessage        = user32.NewProc("DispatchMessageW")
	pIsDialogMessage        = user32.NewProc("IsDialogMessageW")
	pDrawFocusRect          = user32.NewProc("DrawFocusRect")
	pPostQuitMessage        = user32.NewProc("PostQuitMessage")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pInvalidateRect         = user32.NewProc("InvalidateRect")
	pSetTimer               = user32.NewProc("SetTimer")
	pKillTimer              = user32.NewProc("KillTimer")
	pLoadCursor             = user32.NewProc("LoadCursorW")
	pLoadImage              = user32.NewProc("LoadImageW")
	pSendMessage            = user32.NewProc("SendMessageW")
	pGetWindowText          = user32.NewProc("GetWindowTextW")
	pSetWindowText          = user32.NewProc("SetWindowTextW")
	pEnableWindow           = user32.NewProc("EnableWindow")
	pDestroyWindow          = user32.NewProc("DestroyWindow")
	pAdjustWindowRect       = user32.NewProc("AdjustWindowRectEx")
	pSetCapture             = user32.NewProc("SetCapture")
	pReleaseCapture         = user32.NewProc("ReleaseCapture")
	pMessageBox             = user32.NewProc("MessageBoxW")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pCreateFont             = gdi32.NewProc("CreateFontW")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pFillRect               = user32.NewProc("FillRect")
	pDrawText               = user32.NewProc("DrawTextW")
	pSetProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
	pGetDC                  = user32.NewProc("GetDC")
	pReleaseDC              = user32.NewProc("ReleaseDC")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	pGetModuleHandle        = kernel32.NewProc("GetModuleHandleW")
)

// All UI dimensions are authored at 96 DPI and scaled up for high-DPI desktops.
var (
	dpiScale      = 1.0
	mainWindow    uintptr
	appInstance   uintptr
	minutesEdit   uintptr
	buttonWindows = map[int]uintptr{}
	editBrush     uintptr
	fonts         = map[string]uintptr{}
	running       bool
	deadline      time.Time
	countdown     = "00:00:00"
	eta           = "Ready to start"
	status        = "Choose a duration and press Start timer."
	note          = "Your PC will sleep when the countdown reaches zero."
	state         = "idle"
)

func u16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func scale(n int) int32    { return int32(float64(n)*dpiScale + 0.5) }
func rect(b box) RECT      { return RECT{scale(b.x), scale(b.y), scale(b.x + b.w), scale(b.y + b.h)} }
func color(hex uint32) uintptr { // RGB input in conventional RRGGBB order, Windows COLORREF is BGR.
	return uintptr(((hex & 0x0000ff) << 16) | (hex & 0x00ff00) | ((hex & 0xff0000) >> 16))
}

const (
	bgColor       uint32 = 0x0b1423
	panelColor    uint32 = 0x192c45
	inputColor    uint32 = 0x233750
	mainText      uint32 = 0xf3f7ff
	mutedText     uint32 = 0xa9bad0
	softText      uint32 = 0x8196b0
	accent        uint32 = 0x65e4c7
	accentHover   uint32 = 0x91efd8
	darkAccent    uint32 = 0x082920
	buttonColor   uint32 = 0x233852
	buttonHover   uint32 = 0x33516f
	borderColor   uint32 = 0x3c5471
	disabledColor uint32 = 0x304158
	disabledText  uint32 = 0x8292a9
)

func setDPI() {
	if err := pSetProcessDPIAware.Find(); err == nil {
		pSetProcessDPIAware.Call()
	}
	dc, _, _ := pGetDC.Call(0)
	if dc != 0 {
		v, _, _ := pGetDeviceCaps.Call(dc, 88) // LOGPIXELSX
		if v >= 96 && v <= 480 {
			dpiScale = float64(v) / 96
		}
		pReleaseDC.Call(0, dc)
	}
}
func makeFont(key string, size, weight int) uintptr {
	h, _, _ := pCreateFont.Call(uintptr(-scale(size)), 0, 0, 0, uintptr(weight), 0, 0, 0, DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0, uintptr(unsafe.Pointer(u16("Segoe UI"))))
	fonts[key] = h
	return h
}
func setupFonts() {
	makeFont("title", 32, 700)
	makeFont("subtitle", 16, 400)
	makeFont("heading", 13, 700)
	makeFont("time", 49, 700)
	makeFont("minor", 14, 400)
	makeFont("input", 19, 700)
	makeFont("button", 16, 700)
	makeFont("body", 15, 400)
	makeFont("notice", 13, 400)
}
func setText(h uintptr, txt string) { pSetWindowText.Call(h, uintptr(unsafe.Pointer(u16(txt)))) }
func repaint() {
	if mainWindow != 0 {
		pInvalidateRect.Call(mainWindow, 0, 0)
	}
}

// Only the countdown and its explanatory caption need repainting on each second.
// Repainting the entire parent 4 times per second caused distracting flicker.
func repaintClock() {
	if mainWindow != 0 {
		r := rect(box{48, 163, 505, 101})
		pInvalidateRect.Call(mainWindow, uintptr(unsafe.Pointer(&r)), 0)
	}
}
func getMinutes() (int, bool) {
	var buf [16]uint16
	pGetWindowText.Call(minutesEdit, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	n, err := strconv.Atoi(syscall.UTF16ToString(buf[:]))
	return n, err == nil && validMinutes(n)
}
func controlsEnabled(active bool) {
	setEnabled := func(hwnd uintptr, enabled bool) {
		if hwnd == 0 {
			return
		}
		if enabled {
			pEnableWindow.Call(hwnd, 1)
		} else {
			pEnableWindow.Call(hwnd, 0)
		}
	}
	setEnabled(minutesEdit, !active)
	for _, btn := range presets {
		setEnabled(buttonWindows[btn.id], !active)
	}
	setEnabled(buttonWindows[startBtn.id], !active)
	setEnabled(buttonWindows[cancelBtn.id], active)
}
func sleepCommand() error {
	// Preserve the exact command and semantics from the original sleep-timer.bat:
	// rundll32.exe powrprof.dll,SetSuspendState Sleep
	return exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "Sleep").Start()
}
func startTimer() {
	if running {
		return
	}
	n, ok := getMinutes()
	if !ok {
		pMessageBox.Call(mainWindow, uintptr(unsafe.Pointer(u16("Enter a whole number of minutes from 1 to 1440."))), uintptr(unsafe.Pointer(u16("Sleep Timer"))), MB_OK|MB_ICONERROR)
		return
	}
	deadline = time.Now().Add(time.Duration(n) * time.Minute)
	running = true
	state = "running"
	countdown = formatSeconds(n * 60)
	eta = "Sleep scheduled for " + deadline.Format("15:04")
	status = "Timer running. You can cancel it any time."
	note = "Keep this window open. Closing it cancels the timer."
	controlsEnabled(true)
	pSetTimer.Call(mainWindow, timerID, 250, 0)
	repaint()
}
func cancelTimer() {
	if !running {
		return
	}
	pKillTimer.Call(mainWindow, timerID)
	running = false
	state = "canceled"
	countdown = "00:00:00"
	eta = "Timer canceled"
	status = "Timer canceled. Your PC will stay awake."
	note = "Choose a duration to start again."
	controlsEnabled(false)
	repaint()
}
func tickTimer() {
	if !running {
		return
	}
	s := secondsRemaining(time.Now(), deadline)
	previousCountdown := countdown
	countdown = formatSeconds(s)
	if s == 0 {
		pKillTimer.Call(mainWindow, timerID)
		running = false
		state = "done"
		eta = "Timer completed"
		status = "Sending sleep command to Windows..."
		note = "The countdown has finished."
		controlsEnabled(false)
		repaint()
		if err := sleepCommand(); err != nil {
			state = "failed"
			status = "Could not send the sleep command."
			note = "Check your Windows power settings and try again."
			pMessageBox.Call(mainWindow, uintptr(unsafe.Pointer(u16("Windows sleep command failed: "+err.Error()))), uintptr(unsafe.Pointer(u16("Sleep Timer"))), MB_OK|MB_ICONERROR)
		} else {
			status = "Sleep command sent to Windows."
		}
		repaint()
		return
	}
	if countdownChanged(previousCountdown, countdown) {
		repaintClock()
	}
}
func presetButton(id int) {
	if running {
		return
	}
	for _, btn := range presets {
		if btn.id == id {
			setText(minutesEdit, strconv.Itoa(btn.id))
			repaint()
			return
		}
	}
}
func fillRect(dc uintptr, b box, clr uint32) {
	r := rect(b)
	brush, _, _ := pCreateSolidBrush.Call(color(clr))
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteObject.Call(brush)
}
func roundRect(dc uintptr, b box, radius int, back, border uint32) {
	brush, _, _ := pCreateSolidBrush.Call(color(back))
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(scale(1)), color(border))
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, uintptr(scale(b.x)), uintptr(scale(b.y)), uintptr(scale(b.x+b.w)), uintptr(scale(b.y+b.h)), uintptr(scale(radius*2)), uintptr(scale(radius*2)))
	pSelectObject.Call(dc, oldPen)
	pSelectObject.Call(dc, oldBrush)
	pDeleteObject.Call(pen)
	pDeleteObject.Call(brush)
}
func drawText(dc uintptr, txt string, b box, fontName string, clr uint32, alignment uint32) {
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, color(clr))
	prev, _, _ := pSelectObject.Call(dc, fonts[fontName])
	r := rect(b)
	units := syscall.StringToUTF16(txt)
	pDrawText.Call(dc, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)-1), uintptr(unsafe.Pointer(&r)), uintptr(alignment|DT_SINGLELINE|DT_VCENTER))
	pSelectObject.Call(dc, prev)
}
func paintButton(dc uintptr, b button, enabled bool, primary bool, selected bool, focused bool) {
	back := buttonColor
	fg := mainText
	stroke := buttonColor
	if primary {
		back = accent
		fg = darkAccent
		stroke = accent
	}
	if !enabled {
		back = disabledColor
		fg = disabledText
		stroke = disabledColor
	}
	if selected && enabled {
		if primary {
			back = 0x52ceb3
			stroke = back
		} else {
			back = buttonHover
			stroke = back
		}
	}
	// Erase the rectangular owner-drawn control behind its rounded shape.
	fillRect(dc, b.area, bgColor)
	roundRect(dc, b.area, 12, back, stroke)
	drawText(dc, b.label, b.area, "button", fg, DT_CENTER)
	if focused {
		r := rect(box{b.area.x + 6, b.area.y + 6, b.area.w - 12, b.area.h - 12})
		pDrawFocusRect.Call(dc, uintptr(unsafe.Pointer(&r)))
	}
}
func createButton(hwnd uintptr, b button) uintptr {
	h, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16(b.label))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, uintptr(scale(b.area.x)), uintptr(scale(b.area.y)), uintptr(scale(b.area.w)), uintptr(scale(b.area.h)), hwnd, uintptr(b.id), appInstance, 0)
	if h != 0 {
		buttonWindows[b.id] = h
	}
	return h
}
func buttonByID(id int) (button, bool) {
	if id == startBtn.id {
		return startBtn, true
	}
	if id == cancelBtn.id {
		return cancelBtn, true
	}
	for _, b := range presets {
		if id == b.id {
			return b, true
		}
	}
	return button{}, false
}
func paintWindow(dc uintptr) {
	fillRect(dc, box{0, 0, clientW, clientH}, bgColor)
	drawText(dc, "Sleep Timer", box{29, 24, 540, 47}, "title", mainText, DT_LEFT)
	drawText(dc, "Choose when your computer should go to sleep.", box{31, 72, 538, 24}, "subtitle", mutedText, DT_LEFT)
	roundRect(dc, box{29, 117, 542, 166}, 18, panelColor, panelColor)
	drawText(dc, "TIME REMAINING", box{54, 132, 490, 28}, "heading", mutedText, DT_LEFT)
	drawText(dc, countdown, box{48, 163, 505, 74}, "time", accent, DT_CENTER)
	drawText(dc, eta, box{48, 239, 505, 25}, "minor", mutedText, DT_CENTER)
	drawText(dc, "Timer duration (minutes)", box{30, 298, 490, 25}, "heading", mainText, DT_LEFT)
	// Native EDIT control is inside a rounded background. This keeps typing, selection, and accessibility native.
	roundRect(dc, box{29, 329, 150, 48}, 11, inputColor, borderColor)
	drawText(dc, status, box{30, 484, 544, 28}, "body", mutedText, DT_LEFT)
	drawText(dc, note, box{30, 516, 544, 24}, "notice", softText, DT_LEFT)
}
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		minutesEdit, _, _ = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(u16("EDIT"))), uintptr(unsafe.Pointer(u16("30"))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_CENTER|ES_NUMBER|ES_AUTOHSCROLL, uintptr(scale(44)), uintptr(scale(337)), uintptr(scale(121)), uintptr(scale(31)), hwnd, ctrlMinutes, appInstance, 0)
		if minutesEdit == 0 {
			return ^uintptr(0)
		}
		pSendMessage.Call(minutesEdit, WM_SETFONT, fonts["input"], 1)
		pSendMessage.Call(minutesEdit, EM_SETLIMITTEXT, 4, 0)
		for _, b := range presets {
			if createButton(hwnd, b) == 0 {
				return ^uintptr(0)
			}
		}
		if createButton(hwnd, startBtn) == 0 {
			return ^uintptr(0)
		}
		if createButton(hwnd, cancelBtn) == 0 {
			return ^uintptr(0)
		}
		controlsEnabled(false)
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_CTLCOLOREDIT, WM_CTLCOLORSTATIC:
		pSetTextColor.Call(wParam, color(mainText))
		pSetBkColor.Call(wParam, color(inputColor))
		pSetBkMode.Call(wParam, OPAQUE)
		return editBrush
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			// Paint completely offscreen, then copy the finished pixels in one step.
			// This avoids the clear-then-draw flash during countdown updates.
			memDC, _, _ := pCreateCompatibleDC.Call(hdc)
			if memDC != 0 {
				bitmap, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(scale(clientW)), uintptr(scale(clientH)))
				if bitmap != 0 {
					oldBitmap, _, _ := pSelectObject.Call(memDC, bitmap)
					paintWindow(memDC)
					area := ps.Paint
					if area.Right > area.Left && area.Bottom > area.Top {
						pBitBlt.Call(hdc, uintptr(area.Left), uintptr(area.Top), uintptr(area.Right-area.Left), uintptr(area.Bottom-area.Top), memDC, uintptr(area.Left), uintptr(area.Top), SRCCOPY)
					}
					pSelectObject.Call(memDC, oldBitmap)
					pDeleteObject.Call(bitmap)
				} else {
					paintWindow(hdc) // Graceful fallback if offscreen allocation fails.
				}
				pDeleteDC.Call(memDC)
			} else {
				paintWindow(hdc)
			}
		}
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_COMMAND:
		id := int(wParam & 0xffff)
		notification := (wParam >> 16) & 0xffff
		if notification == 0 { // BN_CLICKED, dispatched by genuine native controls.
			if id == startBtn.id {
				startTimer()
			} else if id == cancelBtn.id {
				cancelTimer()
			} else {
				presetButton(id)
			}
		}
		return 0
	case WM_DRAWITEM:
		if lParam == 0 {
			return 0
		}
		var drawInfo DRAWITEMSTRUCT
		pRtlMoveMemory.Call(uintptr(unsafe.Pointer(&drawInfo)), lParam, unsafe.Sizeof(drawInfo))
		dis := &drawInfo
		b, ok := buttonByID(int(dis.CtlID))
		if !ok {
			break
		}
		b.area = box{0, 0, int(float64(dis.RcItem.Right-dis.RcItem.Left) / dpiScale), int(float64(dis.RcItem.Bottom-dis.RcItem.Top) / dpiScale)}
		paintButton(dis.Hdc, b, (dis.ItemState&ODS_DISABLED) == 0, b.id == startBtn.id, (dis.ItemState&ODS_SELECTED) != 0, (dis.ItemState&ODS_FOCUS) != 0)
		return 1
	case WM_TIMER:
		if wParam == timerID {
			tickTimer()
			return 0
		}
	case WM_CLOSE:
		if running {
			pKillTimer.Call(hwnd, timerID)
			running = false
		}
		pDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func main() {
	runtime.LockOSThread()
	setDPI()
	setupFonts()
	editBrush, _, _ = pCreateSolidBrush.Call(color(inputColor))
	hInst, _, _ := pGetModuleHandle.Call(0)
	appInstance = hInst
	if hInst == 0 {
		panic("GetModuleHandle failed")
	}
	// Group icon resource ID 1 is embedded in this EXE (ICO sizes 16 through 256).
	bigIcon, _, _ := pLoadImage.Call(hInst, 1, IMAGE_ICON, uintptr(scale(32)), uintptr(scale(32)), LR_DEFAULTCOLOR)
	smallIcon, _, _ := pLoadImage.Call(hInst, 1, IMAGE_ICON, uintptr(scale(16)), uintptr(scale(16)), LR_DEFAULTCOLOR)
	cursor, _, _ := pLoadCursor.Call(0, IDC_ARROW)
	cls := u16("SleepTimerPolishedWin32")
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(wndProc), Instance: hInst, Icon: bigIcon, IconSm: smallIcon, Cursor: cursor, ClassName: cls}
	reg, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if reg == 0 {
		panic(fmt.Sprintf("RegisterClassEx failed: %v", err))
	}
	style := uintptr(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_CLIPCHILDREN)
	bounds := RECT{0, 0, scale(clientW), scale(clientH)}
	pAdjustWindowRect.Call(uintptr(unsafe.Pointer(&bounds)), style, 0, 0)
	w := bounds.Right - bounds.Left
	h := bounds.Bottom - bounds.Top
	mainWindow, _, err = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("Sleep Timer"))), style, uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT), uintptr(w), uintptr(h), 0, 0, hInst, 0)
	if mainWindow == 0 {
		panic(fmt.Sprintf("CreateWindowEx failed: %v", err))
	}
	pSendMessage.Call(mainWindow, WM_SETICON, ICON_BIG, bigIcon)
	pSendMessage.Call(mainWindow, WM_SETICON, ICON_SMALL, smallIcon)
	pShowWindow.Call(mainWindow, SW_SHOW)
	pUpdateWindow.Call(mainWindow)
	var m MSG
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		consumed, _, _ := pIsDialogMessage.Call(mainWindow, uintptr(unsafe.Pointer(&m)))
		if consumed != 0 {
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	for _, h := range fonts {
		pDeleteObject.Call(h)
	}
	pDeleteObject.Call(editBrush)
}
