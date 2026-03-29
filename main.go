package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// Lock OS thread — required for Win32 message loop
func init() {
	runtime.LockOSThread()
}

// ============================================================
// Windows API
// ============================================================

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	pRegisterClassExW    = user32.NewProc("RegisterClassExW")
	pCreateWindowExW     = user32.NewProc("CreateWindowExW")
	pShowWindow          = user32.NewProc("ShowWindow")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pDefWindowProcW      = user32.NewProc("DefWindowProcW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pSendMessageW        = user32.NewProc("SendMessageW")
	pSetFocus            = user32.NewProc("SetFocus")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	pGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	pMoveWindow          = user32.NewProc("MoveWindow")
	pMessageBoxW         = user32.NewProc("MessageBoxW")
	pGetKeyState         = user32.NewProc("GetKeyState")
	pRegisterHotKey      = user32.NewProc("RegisterHotKey")
	pUnregisterHotKey    = user32.NewProc("UnregisterHotKey")
	pOpenClipboard       = user32.NewProc("OpenClipboard")
	pCloseClipboard      = user32.NewProc("CloseClipboard")
	pEmptyClipboard      = user32.NewProc("EmptyClipboard")
	pSetClipboardData    = user32.NewProc("SetClipboardData")
	pGetClipboardData    = user32.NewProc("GetClipboardData")
	pSendInput           = user32.NewProc("SendInput")
	pLoadCursorW         = user32.NewProc("LoadCursorW")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pGlobalFree       = kernel32.NewProc("GlobalFree")
	pSleep            = kernel32.NewProc("Sleep")

	pCreateFontIndirectW = gdi32.NewProc("CreateFontIndirectW")
)

// ============================================================
// Constants
// ============================================================

const (
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_TABSTOP          = 0x00010000
	WS_VSCROLL          = 0x00200000

	WS_EX_TOPMOST    = 0x00000008
	WS_EX_CLIENTEDGE = 0x00000200
	WS_EX_TOOLWINDOW = 0x00000080

	ES_MULTILINE   = 0x0004
	ES_AUTOVSCROLL = 0x0040
	ES_WANTRETURN  = 0x1000

	BS_DEFPUSHBUTTON = 0x00000001
	BS_PUSHBUTTON    = 0x00000000

	WM_CREATE        = 0x0001
	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_SETFONT       = 0x0030
	WM_SETTEXT       = 0x000C
	WM_GETTEXT       = 0x000D
	WM_GETTEXTLENGTH = 0x000E
	WM_KEYDOWN       = 0x0100
	WM_COMMAND       = 0x0111
	WM_HOTKEY        = 0x0312

	SW_SHOW = 5
	SW_HIDE = 0

	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	MOD_WIN     = 0x0008

	VK_RETURN  = 0x0D
	VK_ESCAPE  = 0x1B
	VK_CONTROL = 0x11
	VK_V       = 0x56
	VK_SPACE   = 0x20

	CF_UNICODETEXT  = 13
	GMEM_MOVEABLE   = 0x0002
	INPUT_KEYBOARD  = 1
	KEYEVENTF_KEYUP = 0x0002

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	MB_OK              = 0x00000000
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040

	IDC_ARROW = 32512

	IDC_EDIT_CTRL  = 1001
	IDC_OK_BTN     = 1002
	IDC_CANCEL_BTN = 1003

	HOTKEY_ID = 9999
)

// ============================================================
// Structs — careful with alignment for 64-bit Windows
// ============================================================

type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// KEYBDINPUT for SendInput — must match C layout exactly
// On 64-bit: wVk(2) + wScan(2) + dwFlags(4) + time(4) + dwExtraInfo(8) = 20 bytes
// But INPUT has type(4) + padding(4) + union(24 on x64) = 32 bytes
// We define INPUT_KEYBOARD_64 to match the exact C layout.

type KEYBDINPUT64 struct {
	Type        uint32
	_pad        uint32 // alignment padding on 64-bit
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// ============================================================
// Logging
// ============================================================

var logFile *os.File

func initLog() {
	dir, _ := os.Getwd()
	path := filepath.Join(dir, "vn-input-helper.log")

	// Also try exe dir
	if exePath, err := os.Executable(); err == nil {
		path = filepath.Join(filepath.Dir(exePath), "vn-input-helper.log")
	}

	var err error
	logFile, err = os.Create(path)
	if err != nil {
		// Try current dir as fallback
		logFile, err = os.Create("vn-input-helper.log")
		if err != nil {
			logFile = nil
		}
	}

	logf("=== VN Input Helper started ===")
	logf("Log file: %s", path)
	logf("GOARCH: %s, sizeof(uintptr): %d", runtime.GOARCH, unsafe.Sizeof(uintptr(0)))
}

func logf(format string, args ...interface{}) {
	if logFile != nil {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintln(logFile, msg)
		logFile.Sync()
	}
}

func msgBox(title, text string, flags uint32) {
	pMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(flags))
}

func fatalBox(text string) {
	logf("FATAL: %s", text)
	msgBox("VN Input Helper - Lỗi", text, MB_OK|MB_ICONERROR)
	os.Exit(1)
}

// ============================================================
// Helpers
// ============================================================

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// ============================================================
// Config
// ============================================================

type Config struct {
	Hotkey struct {
		Modifiers []string `json:"modifiers"`
		Key       string   `json:"key"`
	} `json:"hotkey"`
	Window struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Title  string `json:"title"`
	} `json:"window"`
}

var cfg Config

func loadConfig() {
	// Defaults
	cfg.Hotkey.Modifiers = []string{"Ctrl", "Shift"}
	cfg.Hotkey.Key = "Space"
	cfg.Window.Width = 600
	cfg.Window.Height = 350
	cfg.Window.Title = "Nhập nội dung"

	paths := []string{"config.json"}
	if exePath, err := os.Executable(); err == nil {
		paths = append([]string{filepath.Join(filepath.Dir(exePath), "config.json")}, paths...)
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			logf("Config parse error: %v", err)
		} else {
			logf("Config loaded: %s", p)
		}
		break
	}

	logf("Config: hotkey=%s+%s, window=%dx%d",
		strings.Join(cfg.Hotkey.Modifiers, "+"), cfg.Hotkey.Key,
		cfg.Window.Width, cfg.Window.Height)
}

func getModifiers() uint32 {
	var r uint32
	for _, m := range cfg.Hotkey.Modifiers {
		switch strings.ToLower(m) {
		case "ctrl", "control":
			r |= MOD_CONTROL
		case "shift":
			r |= MOD_SHIFT
		case "alt":
			r |= MOD_ALT
		case "win":
			r |= MOD_WIN
		}
	}
	return r
}

func getVK() uint32 {
	switch strings.ToLower(cfg.Hotkey.Key) {
	case "space":
		return VK_SPACE
	case "return", "enter":
		return VK_RETURN
	default:
		if len(cfg.Hotkey.Key) == 1 {
			return uint32(strings.ToUpper(cfg.Hotkey.Key)[0])
		}
		if strings.HasPrefix(strings.ToLower(cfg.Hotkey.Key), "f") {
			n := 0
			fmt.Sscanf(cfg.Hotkey.Key[1:], "%d", &n)
			if n >= 1 && n <= 24 {
				return uint32(0x70 + n - 1)
			}
		}
		return VK_SPACE
	}
}

// ============================================================
// App state
// ============================================================

var (
	mainHwnd     syscall.Handle
	editHwnd     syscall.Handle
	okBtn        syscall.Handle
	cancelBtn    syscall.Handle
	prevWindow   uintptr
	popupVisible bool
	hFont        uintptr
	savedClip    string
)

func createFont() uintptr {
	// Use CreateFontW with explicit parameters
	pCreateFontW := gdi32.NewProc("CreateFontW")
	h, _, _ := pCreateFontW.Call(
		uintptr(0xFFFFFFEC), // height = -20 (as uint32 → wraps correctly)
		0, 0, 0,
		400, // weight = normal
		0, 0, 0,
		1, // charset = DEFAULT
		0, 0,
		5, // quality = CLEARTYPE
		0,
		uintptr(unsafe.Pointer(utf16Ptr("Segoe UI"))),
	)
	return h
}

// ============================================================
// Window Proc
// ============================================================

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		logf("WM_CREATE received")

		w := cfg.Window.Width
		h := cfg.Window.Height

		// Edit control
		ret, _, _ := pCreateWindowExW.Call(
			WS_EX_CLIENTEDGE,
			uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
			0,
			WS_CHILD|WS_VISIBLE|WS_VSCROLL|WS_TABSTOP|ES_MULTILINE|ES_AUTOVSCROLL|ES_WANTRETURN,
			10, 10,
			uintptr(w-40), uintptr(h-100),
			uintptr(hwnd), IDC_EDIT_CTRL, 0, 0)
		editHwnd = syscall.Handle(ret)
		logf("editHwnd = %d", editHwnd)

		// OK button
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("OK (Ctrl+Enter)"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
			uintptr(w-300), uintptr(h-80),
			160, 35,
			uintptr(hwnd), IDC_OK_BTN, 0, 0)
		okBtn = syscall.Handle(ret)

		// Cancel button
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Hủy (Esc)"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
			uintptr(w-130), uintptr(h-80),
			120, 35,
			uintptr(hwnd), IDC_CANCEL_BTN, 0, 0)
		cancelBtn = syscall.Handle(ret)

		// Font
		hFont = createFont()
		pSendMessageW.Call(uintptr(editHwnd), WM_SETFONT, hFont, 1)
		pSendMessageW.Call(uintptr(okBtn), WM_SETFONT, hFont, 1)
		pSendMessageW.Call(uintptr(cancelBtn), WM_SETFONT, hFont, 1)

		pShowWindow.Call(uintptr(hwnd), SW_HIDE)
		logf("Window initialized and hidden")
		return 0

	case WM_HOTKEY:
		logf("WM_HOTKEY wParam=%d", wParam)
		if wParam == HOTKEY_ID {
			togglePopup()
		}
		return 0

	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		switch id {
		case IDC_OK_BTN:
			doOK()
		case IDC_CANCEL_BTN:
			hidePopup()
		}
		return 0

	case WM_CLOSE:
		hidePopup()
		return 0

	case WM_DESTROY:
		pUnregisterHotKey.Call(uintptr(hwnd), HOTKEY_ID)
		pPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// ============================================================
// Popup logic
// ============================================================

func togglePopup() {
	if popupVisible {
		hidePopup()
	} else {
		showPopup()
	}
}

func showPopup() {
	prevWindow, _, _ = pGetForegroundWindow.Call()
	logf("Saved prev window: %d", prevWindow)

	// Clear text
	pSendMessageW.Call(uintptr(editHwnd), WM_SETTEXT, 0, uintptr(unsafe.Pointer(utf16Ptr(""))))

	// Center
	sw, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
	x := (int(sw) - cfg.Window.Width) / 2
	y := (int(sh) - cfg.Window.Height) / 2

	pMoveWindow.Call(uintptr(mainHwnd), uintptr(x), uintptr(y),
		uintptr(cfg.Window.Width), uintptr(cfg.Window.Height), 1)
	pShowWindow.Call(uintptr(mainHwnd), SW_SHOW)
	pSetForegroundWindow.Call(uintptr(mainHwnd))
	pSetFocus.Call(uintptr(editHwnd))
	popupVisible = true
	logf("Popup shown")
}

func hidePopup() {
	pShowWindow.Call(uintptr(mainHwnd), SW_HIDE)
	popupVisible = false
	if prevWindow != 0 {
		pSetForegroundWindow.Call(prevWindow)
	}
	logf("Popup hidden")
}

func getEditText() string {
	length, _, _ := pSendMessageW.Call(uintptr(editHwnd), WM_GETTEXTLENGTH, 0, 0)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	pSendMessageW.Call(uintptr(editHwnd), WM_GETTEXT, length+1, uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

// ============================================================
// Clipboard
// ============================================================

func setClipboard(text string) {
	r, _, _ := pOpenClipboard.Call(0)
	if r == 0 {
		logf("OpenClipboard failed")
		return
	}
	defer pCloseClipboard.Call()

	pEmptyClipboard.Call()
	u := syscall.StringToUTF16(text)
	size := len(u) * 2

	hMem, _, _ := pGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		logf("GlobalAlloc failed")
		return
	}
	pMem, _, _ := pGlobalLock.Call(hMem)
	if pMem == 0 {
		pGlobalFree.Call(hMem)
		return
	}

	// memcpy
	for i, v := range u {
		*(*uint16)(unsafe.Pointer(pMem + uintptr(i*2))) = v
	}

	pGlobalUnlock.Call(hMem)
	pSetClipboardData.Call(CF_UNICODETEXT, hMem)
}

func getClipboard() string {
	r, _, _ := pOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer pCloseClipboard.Call()

	h, _, _ := pGetClipboardData.Call(CF_UNICODETEXT)
	if h == 0 {
		return ""
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer pGlobalUnlock.Call(h)

	var u []uint16
	for off := uintptr(0); ; off += 2 {
		c := *(*uint16)(unsafe.Pointer(p + off))
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return syscall.UTF16ToString(u)
}

// ============================================================
// SendInput — Ctrl+V
// ============================================================

func simulateCtrlV() {
	pSleep.Call(150) // wait for focus switch

	// Use raw bytes approach to avoid struct alignment issues
	// Each INPUT on 64-bit is 40 bytes: type(4) + pad(4) + ki(24) + pad(8)
	// We'll call SendInput one at a time to be safe

	sendKey := func(vk uint16, up bool) {
		var flags uint32
		if up {
			flags = KEYEVENTF_KEYUP
		}
		var input [40]byte
		// Type = INPUT_KEYBOARD = 1
		*(*uint32)(unsafe.Pointer(&input[0])) = INPUT_KEYBOARD
		// KEYBDINPUT starts at offset 8 on 64-bit
		*(*uint16)(unsafe.Pointer(&input[8])) = vk     // wVk
		*(*uint16)(unsafe.Pointer(&input[10])) = 0     // wScan
		*(*uint32)(unsafe.Pointer(&input[12])) = flags // dwFlags
		*(*uint32)(unsafe.Pointer(&input[16])) = 0     // time
		*(*uintptr)(unsafe.Pointer(&input[24])) = 0    // dwExtraInfo

		ret, _, err := pSendInput.Call(1, uintptr(unsafe.Pointer(&input[0])), 40)
		if ret == 0 {
			logf("SendInput failed: %v", err)
		}
	}

	sendKey(VK_CONTROL, false) // Ctrl down
	sendKey(VK_V, false)       // V down
	sendKey(VK_V, true)        // V up
	sendKey(VK_CONTROL, true)  // Ctrl up

	logf("Ctrl+V simulated")
}

// ============================================================
// OK handler
// ============================================================

func doOK() {
	text := getEditText()
	if text == "" {
		hidePopup()
		return
	}
	logf("Text: %s", text)

	savedClip = getClipboard()
	setClipboard(text)

	// Hide popup first
	pShowWindow.Call(uintptr(mainHwnd), SW_HIDE)
	popupVisible = false
	logf("Popup hidden")

	// Restore focus to previous window with retry
	if prevWindow != 0 {
		for i := 0; i < 10; i++ {
			pSetForegroundWindow.Call(prevWindow)
			pSleep.Call(50)

			// Check if focus actually switched
			cur, _, _ := pGetForegroundWindow.Call()
			logf("Focus check %d: current=%d, target=%d", i, cur, prevWindow)
			if cur == prevWindow {
				break
			}
		}
		// Extra settle time after focus confirmed
		pSleep.Call(100)
	}

	simulateCtrlV()

	go func() {
		pSleep.Call(800)
		if savedClip != "" {
			setClipboard(savedClip)
			logf("Clipboard restored")
		}
	}()
}

// ============================================================
// Ctrl+Enter check
// ============================================================

func isCtrlEnter(m *MSG) bool {
	if m.Message == WM_KEYDOWN && m.WParam == VK_RETURN {
		state, _, _ := pGetKeyState.Call(VK_CONTROL)
		return (state & 0x8000) != 0
	}
	return false
}

// ============================================================
// Main
// ============================================================

func main() {
	// Catch panics
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %v", r)
			logf(msg)
			msgBox("VN Input Helper - Crash", msg, MB_OK|MB_ICONERROR)
		}
	}()

	initLog()
	loadConfig()

	hotkeyStr := strings.Join(cfg.Hotkey.Modifiers, "+") + "+" + cfg.Hotkey.Key

	hInst, _, _ := pGetModuleHandleW.Call(0)
	logf("hInstance = %d", hInst)

	// Load cursor
	cursor, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	logf("cursor = %d", cursor)

	className := utf16Ptr("VNInputHelper")

	wc := WNDCLASSEXW{
		Size:       uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:      CS_HREDRAW | CS_VREDRAW,
		WndProc:    syscall.NewCallback(wndProc),
		Instance:   syscall.Handle(hInst),
		Cursor:     syscall.Handle(cursor),
		Background: 16, // COLOR_BTNFACE + 1
		ClassName:  className,
	}

	logf("WNDCLASSEXW size = %d", unsafe.Sizeof(wc))

	atom, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		fatalBox(fmt.Sprintf("RegisterClassEx failed: %v", err))
	}
	logf("Atom = %d", atom)

	hwnd, _, err := pCreateWindowExW.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(cfg.Window.Title))),
		WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU,
		100, 100,
		uintptr(cfg.Window.Width), uintptr(cfg.Window.Height),
		0, 0, hInst, 0)

	if hwnd == 0 {
		fatalBox(fmt.Sprintf("CreateWindowEx failed: %v", err))
	}
	mainHwnd = syscall.Handle(hwnd)
	logf("mainHwnd = %d", mainHwnd)

	// Register hotkey
	mod := getModifiers()
	vk := getVK()
	logf("RegisterHotKey: mod=0x%X vk=0x%X", mod, vk)

	r, _, err := pRegisterHotKey.Call(uintptr(mainHwnd), HOTKEY_ID, uintptr(mod), uintptr(vk))
	if r == 0 {
		fatalBox(fmt.Sprintf("Phím tắt %s không đăng ký được!\n%v\nCó thể bị app khác chiếm.", hotkeyStr, err))
	}
	logf("Hotkey registered OK")

	// Show startup notification
	msgBox("VN Input Helper",
		fmt.Sprintf("Đang chạy!\n\nPhím tắt: %s\nCtrl+Enter = Paste\nEsc = Hủy\n\nNhấn OK để ẩn thông báo.", hotkeyStr),
		MB_OK|MB_ICONINFORMATION)

	logf("Entering message loop")

	// Message loop
	var m MSG
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}

		// Hotkey from message queue
		if m.Message == WM_HOTKEY && m.WParam == HOTKEY_ID {
			logf("WM_HOTKEY in loop")
			togglePopup()
			continue
		}

		// Ctrl+Enter in edit
		if isCtrlEnter(&m) && m.Hwnd == editHwnd {
			doOK()
			continue
		}

		// Esc
		if m.Message == WM_KEYDOWN && m.WParam == VK_ESCAPE && popupVisible {
			hidePopup()
			continue
		}

		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	logf("Exiting")
}
