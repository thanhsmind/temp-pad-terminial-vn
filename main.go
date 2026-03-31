package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
	advapi32  = syscall.NewLazyDLL("advapi32.dll")
	comctl32  = syscall.NewLazyDLL("comctl32.dll")
	comdlg32  = syscall.NewLazyDLL("comdlg32.dll")

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
	pDestroyWindow       = user32.NewProc("DestroyWindow")
	pUpdateWindow        = user32.NewProc("UpdateWindow")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pGlobalFree       = kernel32.NewProc("GlobalFree")
	pSleep            = kernel32.NewProc("Sleep")

	pCreateFontIndirectW = gdi32.NewProc("CreateFontIndirectW")

	pRegOpenKeyExW  = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey    = advapi32.NewProc("RegCloseKey")

	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	pGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")

	pPostMessageW    = user32.NewProc("PostMessageW")
	pEnableWindow    = user32.NewProc("EnableWindow")
	pIsWindowVisible = user32.NewProc("IsWindowVisible")
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
	WS_CLIPSIBLINGS     = 0x04000000

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
	VK_P       = 0x50
	VK_A       = 0x41
	VK_UP      = 0x26
	VK_DOWN    = 0x28

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

	IDC_EDIT_CTRL    = 1001
	IDC_OK_BTN       = 1002
	IDC_CANCEL_BTN   = 1003
	IDC_PROMPTS_BTN  = 1004
	IDC_PICKER_LIST  = 2001
	IDC_PICKER_OK    = 2002
	IDC_PICKER_CANCEL = 2003
	IDC_PICKER_SEARCH = 2004

	IDC_TAB_CTRL       = 3000
	IDC_CONV_FILE_BTN  = 3001
	IDC_CONV_FILE_LABEL = 3002
	IDC_CONV_WEBM_BTN  = 3003
	IDC_CONV_H265_BTN  = 3004
	IDC_CONV_MP3_BTN   = 3005
	IDC_CONV_PROGRESS  = 3006
	IDC_CONV_STATUS    = 3007
	IDC_CONV_CANCEL_BTN = 3008

	EM_SETSEL          = 0x00B1
	EN_CHANGE          = 0x0300
	LB_RESETCONTENT    = 0x0184

	LBS_NOTIFY           = 0x0001
	LBS_NOINTEGRALHEIGHT = 0x0100
	LBN_DBLCLK           = 2
	LB_ADDSTRING          = 0x0180
	LB_GETCURSEL          = 0x0188
	LB_SETCURSEL          = 0x0186
	LB_GETCOUNT           = 0x018B
	LB_ERR                = -1

	HOTKEY_ID = 9999

	// Tab control
	WM_NOTIFY      = 0x004E
	WM_APP         = 0x8000
	TCM_INSERTITEM = 0x133E // TCM_INSERTITEMW
	TCM_GETCURSEL  = 0x130B
	TCN_SELCHANGE  = -551 // TCN_FIRST - 1
	TCS_FIXEDWIDTH = 0x0400

	// Progress bar
	PBM_SETRANGE = 0x0401
	PBM_SETPOS   = 0x0402
	PBM_SETMARQUEE = 0x040A
	PBS_SMOOTH   = 0x01
	PBS_MARQUEE  = 0x08

	// Common controls
	ICC_TAB_CLASSES      = 0x00000008
	ICC_PROGRESS_CLASS   = 0x00000020

	// File dialog
	OFN_FILEMUSTEXIST  = 0x00001000
	OFN_PATHMUSTEXIST  = 0x00000800
	OFN_OVERWRITEPROMPT = 0x00000002
	OFN_NOCHANGEDIR    = 0x00000008

	// Static control
	SS_LEFT    = 0x00000000
	SS_PATHELLIPSIS = 0x00008000

	// Custom messages for converter
	WM_CONVERT_PROGRESS = WM_APP + 1
	WM_CONVERT_DONE     = WM_APP + 2

	HKEY_CURRENT_USER         = 0x80000001
	KEY_SET_VALUE             = 0x0002
	REG_SZ                    = 1
	ERROR_SUCCESS             = 0
	ERROR_FILE_NOT_FOUND      = 2
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

type INITCOMMONCONTROLSEX struct {
	Size uint32
	ICC  uint32
}

type NMHDR struct {
	HwndFrom uintptr
	IdFrom   uintptr
	Code     int32
	_pad     int32 // alignment padding on 64-bit
}

type TCITEMW struct {
	Mask       uint32
	State      uint32
	StateMask  uint32
	Text       *uint16
	TextMax    int32
	Image      int32
	LParam     uintptr
}

// OPENFILENAMEW — 64-bit aligned struct for file dialogs
type OPENFILENAMEW struct {
	StructSize      uint32
	_pad1           uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustFilter   uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	_pad2           uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	_pad3           uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefExt          *uint16
	CustData        uintptr
	Hook            uintptr
	TemplateName    *uint16
	PvReserved      uintptr
	DwReserved      uint32
	FlagsEx         uint32
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
	AutoStart bool `json:"auto_start"`
}

var cfg Config

func loadConfig() {
	// Defaults
	cfg.Hotkey.Modifiers = []string{"Ctrl", "Shift"}
	cfg.Hotkey.Key = "Space"
	cfg.Window.Width = 750
	cfg.Window.Height = 600
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

	logf("Config: hotkey=%s+%s, window=%dx%d, auto_start=%v",
		strings.Join(cfg.Hotkey.Modifiers, "+"), cfg.Hotkey.Key,
		cfg.Window.Width, cfg.Window.Height, cfg.AutoStart)
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
// Auto-start (Windows Registry)
// ============================================================

const (
	autoStartKeyPath  = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartValueName = "VNInputHelper"
)

func setAutoStart() {
	exePath, err := os.Executable()
	if err != nil {
		logf("Auto-start: failed to get exe path: %v", err)
		return
	}

	var hKey uintptr
	r, _, e := pRegOpenKeyExW.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(utf16Ptr(autoStartKeyPath))),
		0,
		KEY_SET_VALUE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r != ERROR_SUCCESS {
		logf("Auto-start: RegOpenKeyEx failed: %v", e)
		return
	}
	defer pRegCloseKey.Call(hKey)

	valueUTF16 := syscall.StringToUTF16(exePath)
	dataSize := len(valueUTF16) * 2

	r, _, e = pRegSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(utf16Ptr(autoStartValueName))),
		0,
		REG_SZ,
		uintptr(unsafe.Pointer(&valueUTF16[0])),
		uintptr(dataSize),
	)
	if r != ERROR_SUCCESS {
		logf("Auto-start: RegSetValueEx failed: %v", e)
		return
	}

	logf("Auto-start: registered '%s'", exePath)
}

func removeAutoStart() {
	var hKey uintptr
	r, _, e := pRegOpenKeyExW.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(utf16Ptr(autoStartKeyPath))),
		0,
		KEY_SET_VALUE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r != ERROR_SUCCESS {
		// Key doesn't exist or can't open — nothing to remove
		logf("Auto-start: RegOpenKeyEx for removal: %v (may be OK)", e)
		return
	}
	defer pRegCloseKey.Call(hKey)

	r, _, e = pRegDeleteValueW.Call(
		hKey,
		uintptr(unsafe.Pointer(utf16Ptr(autoStartValueName))),
	)
	if r != ERROR_SUCCESS && r != ERROR_FILE_NOT_FOUND {
		logf("Auto-start: RegDeleteValue failed: %v", e)
		return
	}

	logf("Auto-start: registry entry removed")
}

func manageAutoStart() {
	if cfg.AutoStart {
		logf("Auto-start: enabled, updating registry")
		setAutoStart()
	} else {
		logf("Auto-start: disabled, cleaning registry")
		removeAutoStart()
	}
}

// ============================================================
// Prompt templates
// ============================================================

type Prompt struct {
	Name        string
	Description string
	Content     string
}

func loadPrompts() []Prompt {
	exePath, err := os.Executable()
	if err != nil {
		logf("Prompts: failed to get exe path: %v", err)
		return nil
	}
	promptsDir := filepath.Join(filepath.Dir(exePath), "prompts")

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		logf("Prompts: cannot read directory '%s': %v", promptsDir, err)
		return nil
	}

	var prompts []Prompt
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		p, err := parsePromptFile(filepath.Join(promptsDir, entry.Name()))
		if err != nil {
			logf("Prompts: skipping '%s': %v", entry.Name(), err)
			continue
		}
		prompts = append(prompts, p)
	}

	logf("Prompts: loaded %d prompt(s)", len(prompts))
	return prompts
}

func parsePromptFile(path string) (Prompt, error) {
	f, err := os.Open(path)
	if err != nil {
		return Prompt{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var p Prompt
	inFrontmatter := false
	frontmatterDone := false
	var contentLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFrontmatter && !frontmatterDone && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && trimmed == "---" {
			inFrontmatter = false
			frontmatterDone = true
			continue
		}
		if inFrontmatter {
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				switch strings.ToLower(key) {
				case "name":
					p.Name = val
				case "description":
					p.Description = val
				}
			}
			continue
		}
		if frontmatterDone {
			contentLines = append(contentLines, line)
		}
	}

	if p.Name == "" {
		// Use filename as fallback
		p.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	p.Content = strings.TrimSpace(strings.Join(contentLines, "\r\n"))

	if !frontmatterDone {
		return Prompt{}, fmt.Errorf("no frontmatter found")
	}

	return p, nil
}

// ============================================================
// Prompt picker window
// ============================================================

var (
	pickerHwnd       syscall.Handle
	pickerListHwnd   syscall.Handle
	pickerSearchHwnd syscall.Handle
	loadedPrompts    []Prompt
	filteredIndices  []int
	pickerClassName  *uint16
	pickerRegistered bool
)

func showPromptPicker() {
	prompts := loadPrompts()
	if len(prompts) == 0 {
		pMessageBoxW.Call(uintptr(mainHwnd),
			uintptr(unsafe.Pointer(utf16Ptr("Không tìm thấy prompt nào.\nTạo thư mục 'prompts/' cạnh file exe và thêm các file .md vào đó."))),
			uintptr(unsafe.Pointer(utf16Ptr("Prompts"))),
			MB_OK|MB_ICONINFORMATION)
		return
	}
	loadedPrompts = prompts

	if !pickerRegistered {
		pickerClassName = utf16Ptr("VNPromptPicker")
		cursor, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
		hInst, _, _ := pGetModuleHandleW.Call(0)
		wcex := WNDCLASSEXW{
			Size:       uint32(unsafe.Sizeof(WNDCLASSEXW{})),
			WndProc:    syscall.NewCallback(pickerWndProc),
			Cursor:     syscall.Handle(cursor),
			Background: 16, // COLOR_BTNFACE + 1
			ClassName:  pickerClassName,
			Instance:   syscall.Handle(hInst),
		}
		ret, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcex)))
		if ret == 0 {
			logf("Prompts: failed to register picker window class")
			return
		}
		pickerRegistered = true
	}

	// Get screen size for centering
	screenW, _, _ := pGetSystemMetrics.Call(0)
	screenH, _, _ := pGetSystemMetrics.Call(1)
	pickerW := 450
	pickerH := 400
	x := (int(screenW) - pickerW) / 2
	y := (int(screenH) - pickerH) / 2

	ret, _, _ := pCreateWindowExW.Call(
		WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(pickerClassName)),
		uintptr(unsafe.Pointer(utf16Ptr("Chọn Prompt"))),
		WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU,
		uintptr(x), uintptr(y),
		uintptr(pickerW), uintptr(pickerH),
		0, 0, 0, 0)
	pickerHwnd = syscall.Handle(ret)

	pShowWindow.Call(uintptr(pickerHwnd), SW_SHOW)
	pUpdateWindow.Call(uintptr(pickerHwnd))
}

func pickerWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		// Search box
		ret, _, _ := pCreateWindowExW.Call(
			WS_EX_CLIENTEDGE,
			uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
			0,
			WS_CHILD|WS_VISIBLE|WS_TABSTOP,
			10, 10,
			410, 28,
			uintptr(hwnd), IDC_PICKER_SEARCH, 0, 0)
		pickerSearchHwnd = syscall.Handle(ret)

		// Listbox
		ret, _, _ = pCreateWindowExW.Call(
			WS_EX_CLIENTEDGE,
			uintptr(unsafe.Pointer(utf16Ptr("LISTBOX"))),
			0,
			WS_CHILD|WS_VISIBLE|WS_VSCROLL|WS_TABSTOP|LBS_NOTIFY|LBS_NOINTEGRALHEIGHT,
			10, 45,
			410, 265,
			uintptr(hwnd), IDC_PICKER_LIST, 0, 0)
		pickerListHwnd = syscall.Handle(ret)

		// Populate listbox with all prompts
		filteredIndices = nil
		for i, p := range loadedPrompts {
			displayText := p.Name
			if p.Description != "" {
				displayText = p.Name + " — " + p.Description
			}
			pSendMessageW.Call(uintptr(pickerListHwnd), LB_ADDSTRING, 0,
				uintptr(unsafe.Pointer(utf16Ptr(displayText))))
			filteredIndices = append(filteredIndices, i)
		}

		// Insert button
		pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Chèn"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
			230, 320,
			100, 35,
			uintptr(hwnd), IDC_PICKER_OK, 0, 0)

		// Cancel button
		pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Hủy"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
			340, 320,
			80, 35,
			uintptr(hwnd), IDC_PICKER_CANCEL, 0, 0)

		// Apply font
		if hFont != 0 {
			pSendMessageW.Call(uintptr(pickerSearchHwnd), WM_SETFONT, hFont, 1)
			pSendMessageW.Call(uintptr(pickerListHwnd), WM_SETFONT, hFont, 1)
		}

		// Focus search box
		pSetFocus.Call(uintptr(pickerSearchHwnd))

		return 0

	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		notif := int((wParam >> 16) & 0xFFFF)

		switch {
		case id == IDC_PICKER_OK, (id == IDC_PICKER_LIST && notif == LBN_DBLCLK):
			insertSelectedPrompt()
		case id == IDC_PICKER_CANCEL:
			pDestroyWindow.Call(uintptr(hwnd))
		case id == IDC_PICKER_SEARCH && notif == EN_CHANGE:
			filterPrompts()
		}
		return 0

	case WM_CLOSE:
		pDestroyWindow.Call(uintptr(hwnd))
		return 0

	case WM_DESTROY:
		pickerHwnd = 0
		pickerListHwnd = 0
		pickerSearchHwnd = 0
		// Return focus to edit control
		pSetFocus.Call(uintptr(editHwnd))
		return 0
	}

	r, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func filterPrompts() {
	if pickerSearchHwnd == 0 || pickerListHwnd == 0 {
		return
	}

	// Get search text
	length, _, _ := pSendMessageW.Call(uintptr(pickerSearchHwnd), WM_GETTEXTLENGTH, 0, 0)
	query := ""
	if length > 0 {
		buf := make([]uint16, length+1)
		pSendMessageW.Call(uintptr(pickerSearchHwnd), WM_GETTEXT, length+1, uintptr(unsafe.Pointer(&buf[0])))
		query = strings.ToLower(syscall.UTF16ToString(buf))
	}

	// Clear listbox
	pSendMessageW.Call(uintptr(pickerListHwnd), LB_RESETCONTENT, 0, 0)
	filteredIndices = nil

	for i, p := range loadedPrompts {
		if query != "" {
			searchable := strings.ToLower(p.Name + " " + p.Description + " " + p.Content)
			if !strings.Contains(searchable, query) {
				continue
			}
		}
		displayText := p.Name
		if p.Description != "" {
			displayText = p.Name + " — " + p.Description
		}
		pSendMessageW.Call(uintptr(pickerListHwnd), LB_ADDSTRING, 0,
			uintptr(unsafe.Pointer(utf16Ptr(displayText))))
		filteredIndices = append(filteredIndices, i)
	}

	// Auto-select first item so Enter works immediately after typing
	if len(filteredIndices) > 0 {
		pSendMessageW.Call(uintptr(pickerListHwnd), LB_SETCURSEL, 0, 0)
	}
}

func insertSelectedPrompt() {
	if pickerListHwnd == 0 {
		return
	}
	sel, _, _ := pSendMessageW.Call(uintptr(pickerListHwnd), LB_GETCURSEL, 0, 0)
	if int(sel) == LB_ERR || int(sel) < 0 || int(sel) >= len(filteredIndices) {
		return
	}

	prompt := loadedPrompts[filteredIndices[int(sel)]]
	existing := getEditText()

	var newText string
	if existing == "" {
		newText = prompt.Content
	} else {
		newText = existing + "\r\n" + prompt.Content
	}

	pSendMessageW.Call(uintptr(editHwnd), WM_SETTEXT, 0,
		uintptr(unsafe.Pointer(utf16Ptr(newText))))

	logf("Prompts: inserted '%s'", prompt.Name)

	// Close picker
	if pickerHwnd != 0 {
		pDestroyWindow.Call(uintptr(pickerHwnd))
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
	promptsBtn   syscall.Handle
	prevWindow   uintptr
	popupVisible bool
	hFont        uintptr
	savedClip    string
	lastDraft    string // auto-saved draft for recovery

	// Tab control
	tabHwnd       syscall.Handle
	tab1Controls  []syscall.Handle
	tab2Controls  []syscall.Handle
	currentTab    int

	// Converter controls
	convFileLabelHwnd syscall.Handle
	convProgressHwnd  syscall.Handle
	convStatusHwnd    syscall.Handle
	convWebmBtn       syscall.Handle
	convH265Btn       syscall.Handle
	convMp3Btn        syscall.Handle
	convCancelBtn     syscall.Handle

	// Converter state
	selectedMP4   string
	converting    bool
	ffmpegCmd     *exec.Cmd
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
// File dialogs
// ============================================================

// makeFilterUTF16 builds a NUL-separated UTF16 filter string for file dialogs.
// Each pair is (description, pattern). The result is double-NUL terminated.
// syscall.StringToUTF16 cannot be used because it panics on embedded NUL bytes.
func makeFilterUTF16(pairs ...string) []uint16 {
	var result []uint16
	for _, s := range pairs {
		u, _ := syscall.UTF16FromString(s)
		// UTF16FromString appends a NUL terminator, which is exactly what we need
		// as the separator between filter parts
		result = append(result, u...)
	}
	// Double-NUL terminator (one already from the last string, add another)
	result = append(result, 0)
	return result
}

func openMP4File() {
	fileBuf := make([]uint16, 260) // MAX_PATH
	filter := makeFilterUTF16("MP4 Files (*.mp4)", "*.mp4", "All Files (*.*)", "*.*")

	ofn := OPENFILENAMEW{
		StructSize: uint32(unsafe.Sizeof(OPENFILENAMEW{})),
		Owner:      uintptr(mainHwnd),
		Filter:     &filter[0],
		File:       &fileBuf[0],
		MaxFile:    uint32(len(fileBuf)),
		Flags:      OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR,
	}

	ret, _, _ := pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		logf("Open file dialog cancelled")
		return
	}

	selectedMP4 = syscall.UTF16ToString(fileBuf)
	logf("Selected MP4: %s", selectedMP4)

	// Display basename in label
	basename := filepath.Base(selectedMP4)
	pSendMessageW.Call(uintptr(convFileLabelHwnd), WM_SETTEXT, 0,
		uintptr(unsafe.Pointer(utf16Ptr(basename))))

	// Clear previous status
	pSendMessageW.Call(uintptr(convStatusHwnd), WM_SETTEXT, 0,
		uintptr(unsafe.Pointer(utf16Ptr(""))))
	pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETPOS, 0, 0)
}

func showSaveDialog(defaultName string, filterDesc string, filterExt string, defExt string) string {
	fileBuf := make([]uint16, 260)
	// Pre-fill with default name
	defNameUTF16, _ := syscall.UTF16FromString(defaultName)
	copy(fileBuf, defNameUTF16)

	filterStr := makeFilterUTF16(filterDesc, "*."+filterExt, "All Files (*.*)", "*.*")
	defExtUTF16 := utf16Ptr(defExt)

	ofn := OPENFILENAMEW{
		StructSize: uint32(unsafe.Sizeof(OPENFILENAMEW{})),
		Owner:      uintptr(mainHwnd),
		Filter:     &filterStr[0],
		File:       &fileBuf[0],
		MaxFile:    uint32(len(fileBuf)),
		Flags:      OFN_OVERWRITEPROMPT | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR,
		DefExt:     defExtUTF16,
	}

	ret, _, _ := pGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		logf("Save dialog cancelled")
		return ""
	}

	path := syscall.UTF16ToString(fileBuf)
	logf("Save path: %s", path)
	return path
}

// ============================================================
// Converter helpers
// ============================================================

func enableConvertButtons(enable bool) {
	val := uintptr(1) // TRUE = enable
	if !enable {
		val = 0
	}
	pEnableWindow.Call(uintptr(convWebmBtn), val)
	pEnableWindow.Call(uintptr(convH265Btn), val)
	pEnableWindow.Call(uintptr(convMp3Btn), val)
}

// ============================================================
// FFmpeg conversion
// ============================================================

func locateFFmpeg() string {
	exePath, err := os.Executable()
	if err != nil {
		logf("Cannot get exe path: %v", err)
		return ""
	}
	ffmpegPath := filepath.Join(filepath.Dir(exePath), "ffmpeg.exe")
	if _, err := os.Stat(ffmpegPath); err != nil {
		logf("ffmpeg.exe not found at %s", ffmpegPath)
		return ""
	}
	return ffmpegPath
}

func startConversion(format string) {
	if converting {
		logf("Conversion already in progress, ignoring")
		return
	}
	if selectedMP4 == "" {
		msgBox("Video Convert", "Vui lòng chọn file MP4 trước!", MB_OK|MB_ICONINFORMATION)
		return
	}

	ffmpegPath := locateFFmpeg()
	if ffmpegPath == "" {
		msgBox("Video Convert", "Không tìm thấy ffmpeg.exe.\nVui lòng đặt ffmpeg.exe cạnh vn-input-helper.exe", MB_OK|MB_ICONERROR)
		return
	}

	// Determine output format details
	var filterDesc, filterExt, defExt string
	baseName := strings.TrimSuffix(filepath.Base(selectedMP4), filepath.Ext(selectedMP4))
	var defaultName string

	switch format {
	case "webm":
		filterDesc = "WebM Files (*.webm)"
		filterExt = "webm"
		defExt = "webm"
		defaultName = baseName + ".webm"
	case "h265":
		filterDesc = "MP4 Files (*.mp4)"
		filterExt = "mp4"
		defExt = "mp4"
		defaultName = baseName + "_h265.mp4"
	case "mp3":
		filterDesc = "MP3 Files (*.mp3)"
		filterExt = "mp3"
		defExt = "mp3"
		defaultName = baseName + ".mp3"
	default:
		logf("Unknown format: %s", format)
		return
	}

	// Show Save As dialog
	outputPath := showSaveDialog(defaultName, filterDesc, filterExt, defExt)
	if outputPath == "" {
		return // User cancelled
	}

	// Start conversion
	converting = true
	enableConvertButtons(false)
	pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETPOS, 0, 0)
	pSendMessageW.Call(uintptr(convStatusHwnd), WM_SETTEXT, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Đang chuẩn bị convert..."))))
	pShowWindow.Call(uintptr(convCancelBtn), SW_SHOW)

	logf("Starting conversion: %s -> %s (format: %s)", selectedMP4, outputPath, format)
	go runFFmpeg(ffmpegPath, selectedMP4, outputPath, format)
}

func cancelConversion() {
	if !converting || ffmpegCmd == nil {
		return
	}
	logf("Cancelling conversion...")
	if ffmpegCmd.Process != nil {
		ffmpegCmd.Process.Kill()
	}
}

func runFFmpeg(ffmpegPath, input, output, format string) {
	// Build args based on format
	var args []string
	switch format {
	case "webm":
		args = []string{
			"-i", input,
			"-c:v", "libvpx-vp9",
			"-crf", "28",
			"-b:v", "0",
			"-cpu-used", "1",
			"-row-mt", "1",
			"-an",
			"-progress", "pipe:1",
			"-y",
			output,
		}
	case "h265":
		args = []string{
			"-i", input,
			"-c:v", "libx265",
			"-crf", "23",
			"-preset", "slow",
			"-pix_fmt", "yuv420p",
			"-tag:v", "hvc1",
			"-an",
			"-movflags", "+faststart",
			"-progress", "pipe:1",
			"-y",
			output,
		}
	case "mp3":
		args = []string{
			"-i", input,
			"-vn",
			"-c:a", "libmp3lame",
			"-b:a", "320k",
			"-progress", "pipe:1",
			"-y",
			output,
		}
	}

	cmd := exec.Command(ffmpegPath, args...)
	ffmpegCmd = cmd

	// Pipe stdout for progress (-progress pipe:1)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logf("Failed to create stdout pipe: %v", err)
		pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_DONE, 0, 0)
		return
	}

	// Pipe stderr for duration info
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logf("Failed to create stderr pipe: %v", err)
		pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_DONE, 0, 0)
		return
	}

	if err := cmd.Start(); err != nil {
		logf("Failed to start ffmpeg: %v", err)
		pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_DONE, 0, 0)
		return
	}

	// Parse total duration from stderr in a separate goroutine
	var totalDurationMs int64
	durationRe := regexp.MustCompile(`Duration:\s*(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if m := durationRe.FindStringSubmatch(line); m != nil && totalDurationMs == 0 {
				h, _ := strconv.ParseInt(m[1], 10, 64)
				min, _ := strconv.ParseInt(m[2], 10, 64)
				s, _ := strconv.ParseInt(m[3], 10, 64)
				cs, _ := strconv.ParseInt(m[4], 10, 64)
				totalDurationMs = (h*3600+min*60+s)*1000 + cs*10
				logf("Total duration: %dms", totalDurationMs)
			}
		}
	}()

	// Parse progress from stdout (-progress pipe:1 format)
	outTimeRe := regexp.MustCompile(`out_time_us=(\d+)`)
	progressRe := regexp.MustCompile(`progress=(\w+)`)
	scanner := bufio.NewScanner(stdout)
	lastPercent := 0

	for scanner.Scan() {
		line := scanner.Text()

		if m := outTimeRe.FindStringSubmatch(line); m != nil {
			outTimeUs, _ := strconv.ParseInt(m[1], 10, 64)
			outTimeMs := outTimeUs / 1000
			if totalDurationMs > 0 {
				percent := int(outTimeMs * 100 / totalDurationMs)
				if percent > 100 {
					percent = 100
				}
				if percent != lastPercent {
					lastPercent = percent
					pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_PROGRESS, uintptr(percent), 0)
				}
			}
		}

		if m := progressRe.FindStringSubmatch(line); m != nil && m[1] == "end" {
			break
		}
	}

	err = cmd.Wait()
	ffmpegCmd = nil

	if err != nil {
		logf("FFmpeg error: %v", err)
		// Clean up partial output file
		os.Remove(output)
		pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_DONE, 0, 0)
	} else {
		pPostMessageW.Call(uintptr(mainHwnd), WM_CONVERT_DONE, 1, 0)
	}
}

// ============================================================
// Tab switching
// ============================================================

func switchTab(index int) {
	currentTab = index
	show := func(controls []syscall.Handle) {
		for _, c := range controls {
			pShowWindow.Call(uintptr(c), SW_SHOW)
		}
	}
	hide := func(controls []syscall.Handle) {
		for _, c := range controls {
			pShowWindow.Call(uintptr(c), SW_HIDE)
		}
	}
	if index == 0 {
		show(tab1Controls)
		hide(tab2Controls)
		pSetFocus.Call(uintptr(editHwnd))
	} else {
		hide(tab1Controls)
		show(tab2Controls)
	}
	logf("Switched to tab %d", index)
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

		// Font (create early so we can apply to all controls)
		hFont = createFont()

		// Client area is smaller than window size due to title bar and borders
		// WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU: titlebar ~31px, borders ~8px
		clientW := w - 16  // approximate client width
		clientH := h - 62  // approximate client height (subtract title bar + borders)

		// Tab control fills the client area
		ret, _, _ := pCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("SysTabControl32"))),
			0,
			WS_CHILD|WS_VISIBLE|WS_CLIPSIBLINGS,
			0, 0,
			uintptr(clientW), uintptr(clientH),
			uintptr(hwnd), IDC_TAB_CTRL, 0, 0)
		tabHwnd = syscall.Handle(ret)
		logf("tabHwnd = %d", tabHwnd)
		pSendMessageW.Call(uintptr(tabHwnd), WM_SETFONT, hFont, 1)

		// Insert tab items
		tcif := uint32(0x0001) // TCIF_TEXT
		tab1Text := utf16Ptr("VN Input")
		tab1Item := TCITEMW{Mask: tcif, Text: tab1Text}
		pSendMessageW.Call(uintptr(tabHwnd), TCM_INSERTITEM, 0, uintptr(unsafe.Pointer(&tab1Item)))

		tab2Text := utf16Ptr("Video Convert")
		tab2Item := TCITEMW{Mask: tcif, Text: tab2Text}
		pSendMessageW.Call(uintptr(tabHwnd), TCM_INSERTITEM, 1, uintptr(unsafe.Pointer(&tab2Item)))

		// Content area inside tab (below tab header, with margins)
		tabTop := uintptr(32)
		contentW := uintptr(clientW - 20)
		contentH := uintptr(clientH - 40)

		// ---- Tab 1: VN Input controls ----
		btnRowY := contentH - 45 + tabTop

		ret, _, _ = pCreateWindowExW.Call(
			WS_EX_CLIENTEDGE,
			uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
			0,
			WS_CHILD|WS_VISIBLE|WS_VSCROLL|WS_TABSTOP|ES_MULTILINE|ES_AUTOVSCROLL|ES_WANTRETURN,
			10, tabTop,
			contentW, btnRowY-tabTop-10,
			uintptr(hwnd), IDC_EDIT_CTRL, 0, 0)
		editHwnd = syscall.Handle(ret)
		logf("editHwnd = %d", editHwnd)

		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("OK (Ctrl+Enter)"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
			contentW-270, btnRowY,
			160, 35,
			uintptr(hwnd), IDC_OK_BTN, 0, 0)
		okBtn = syscall.Handle(ret)

		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Hủy (Esc)"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
			contentW-100, btnRowY,
			110, 35,
			uintptr(hwnd), IDC_CANCEL_BTN, 0, 0)
		cancelBtn = syscall.Handle(ret)

		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Prompts"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
			10, btnRowY,
			100, 35,
			uintptr(hwnd), IDC_PROMPTS_BTN, 0, 0)
		promptsBtn = syscall.Handle(ret)

		tab1Controls = []syscall.Handle{editHwnd, okBtn, cancelBtn, promptsBtn}

		// Apply font to Tab 1 controls
		for _, c := range tab1Controls {
			pSendMessageW.Call(uintptr(c), WM_SETFONT, hFont, 1)
		}

		// ---- Tab 2: Video Convert controls ----
		convY := tabTop + 10

		// "Chọn file MP4" button
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Chọn file MP4..."))),
			WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,
			10, convY,
			160, 32,
			uintptr(hwnd), IDC_CONV_FILE_BTN, 0, 0)
		convFileBtn := syscall.Handle(ret)

		// File name label
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("STATIC"))),
			uintptr(unsafe.Pointer(utf16Ptr("Chưa chọn file"))),
			WS_CHILD|SS_LEFT|SS_PATHELLIPSIS,
			180, convY+7,
			contentW-180, 22,
			uintptr(hwnd), IDC_CONV_FILE_LABEL, 0, 0)
		convFileLabelHwnd = syscall.Handle(ret)

		// Convert buttons row
		convBtnY := convY + 50
		convBtnW := uintptr((int(contentW) - 20) / 3) // divide evenly with gaps
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("→ WebM VP9"))),
			WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,
			10, convBtnY,
			convBtnW, 38,
			uintptr(hwnd), IDC_CONV_WEBM_BTN, 0, 0)
		convWebmBtn = syscall.Handle(ret)

		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("→ MP4 H.265"))),
			WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,
			10+convBtnW+5, convBtnY,
			convBtnW, 38,
			uintptr(hwnd), IDC_CONV_H265_BTN, 0, 0)
		convH265Btn = syscall.Handle(ret)

		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("→ MP3 320kbps"))),
			WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,
			10+convBtnW*2+10, convBtnY,
			convBtnW, 38,
			uintptr(hwnd), IDC_CONV_MP3_BTN, 0, 0)
		convMp3Btn = syscall.Handle(ret)

		// Progress bar
		progressY := convBtnY + 55
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("msctls_progress32"))),
			0,
			WS_CHILD|PBS_SMOOTH,
			10, progressY,
			contentW, 22,
			uintptr(hwnd), IDC_CONV_PROGRESS, 0, 0)
		convProgressHwnd = syscall.Handle(ret)
		// Set range 0-100
		pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETRANGE, 0, uintptr(100<<16))

		// Status label
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("STATIC"))),
			uintptr(unsafe.Pointer(utf16Ptr(""))),
			WS_CHILD|SS_LEFT,
			10, progressY+28,
			contentW, 22,
			uintptr(hwnd), IDC_CONV_STATUS, 0, 0)
		convStatusHwnd = syscall.Handle(ret)

		// Cancel conversion button
		ret, _, _ = pCreateWindowExW.Call(0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Hủy convert"))),
			WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,
			10, progressY+55,
			130, 32,
			uintptr(hwnd), IDC_CONV_CANCEL_BTN, 0, 0)
		convCancelBtn = syscall.Handle(ret)

		tab2Controls = []syscall.Handle{convFileBtn, convFileLabelHwnd, convWebmBtn, convH265Btn, convMp3Btn, convProgressHwnd, convStatusHwnd, convCancelBtn}

		// Apply font to Tab 2 controls
		for _, c := range tab2Controls {
			pSendMessageW.Call(uintptr(c), WM_SETFONT, hFont, 1)
		}

		// Initially show Tab 1, hide Tab 2
		currentTab = 0
		switchTab(0)

		pShowWindow.Call(uintptr(hwnd), SW_HIDE)
		logf("Window initialized and hidden")
		return 0

	case WM_HOTKEY:
		logf("WM_HOTKEY wParam=%d", wParam)
		if wParam == HOTKEY_ID {
			togglePopup()
		}
		return 0

	case WM_NOTIFY:
		nmhdr := (*NMHDR)(unsafe.Pointer(lParam))
		if nmhdr.Code == TCN_SELCHANGE {
			sel, _, _ := pSendMessageW.Call(uintptr(tabHwnd), TCM_GETCURSEL, 0, 0)
			switchTab(int(sel))
		}
		return 0

	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		switch id {
		case IDC_OK_BTN:
			doOK()
		case IDC_CANCEL_BTN:
			hidePopup()
		case IDC_PROMPTS_BTN:
			showPromptPicker()
		case IDC_CONV_FILE_BTN:
			openMP4File()
		case IDC_CONV_WEBM_BTN:
			startConversion("webm")
		case IDC_CONV_H265_BTN:
			startConversion("h265")
		case IDC_CONV_MP3_BTN:
			startConversion("mp3")
		case IDC_CONV_CANCEL_BTN:
			cancelConversion()
		}
		return 0

	case WM_CONVERT_PROGRESS:
		percent := int(wParam)
		pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETPOS, uintptr(percent), 0)
		pSendMessageW.Call(uintptr(convStatusHwnd), WM_SETTEXT, 0,
			uintptr(unsafe.Pointer(utf16Ptr(fmt.Sprintf("Đang convert... %d%%", percent)))))
		return 0

	case WM_CONVERT_DONE:
		success := int(wParam)
		converting = false
		enableConvertButtons(true)
		pShowWindow.Call(uintptr(convCancelBtn), SW_HIDE)
		if success == 1 {
			pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETPOS, 100, 0)
			pSendMessageW.Call(uintptr(convStatusHwnd), WM_SETTEXT, 0,
				uintptr(unsafe.Pointer(utf16Ptr("Hoàn thành!"))))
			logf("Conversion completed successfully")
		} else {
			pSendMessageW.Call(uintptr(convProgressHwnd), PBM_SETPOS, 0, 0)
			errMsg := "Lỗi khi convert!"
			if lParam != 0 {
				// lParam contains pointer to error string
				errMsg = fmt.Sprintf("Lỗi: convert thất bại")
			}
			pSendMessageW.Call(uintptr(convStatusHwnd), WM_SETTEXT, 0,
				uintptr(unsafe.Pointer(utf16Ptr(errMsg))))
			logf("Conversion failed")
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

	// Restore last draft if available, otherwise clear
	if lastDraft != "" {
		pSendMessageW.Call(uintptr(editHwnd), WM_SETTEXT, 0, uintptr(unsafe.Pointer(utf16Ptr(lastDraft))))
		// Select all text so user can see recovered content and easily replace it
		pSendMessageW.Call(uintptr(editHwnd), EM_SETSEL, 0, uintptr(0xFFFFFFFF))
		logf("Draft restored (%d chars)", len(lastDraft))
	} else {
		pSendMessageW.Call(uintptr(editHwnd), WM_SETTEXT, 0, uintptr(unsafe.Pointer(utf16Ptr(""))))
	}

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
	// Save current text as draft for recovery
	lastDraft = getEditText()
	if lastDraft != "" {
		logf("Draft saved (%d chars)", len(lastDraft))
	}

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

	// Clear draft since user confirmed the text
	lastDraft = ""

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
	manageAutoStart()

	// Initialize common controls (tab control, progress bar)
	icc := INITCOMMONCONTROLSEX{
		Size: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})),
		ICC:  ICC_TAB_CLASSES | ICC_PROGRESS_CLASS,
	}
	ret0, _, err0 := pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	if ret0 == 0 {
		logf("InitCommonControlsEx failed: %v", err0)
	} else {
		logf("InitCommonControlsEx OK")
	}

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

		// Ctrl+P to open prompt picker
		if m.Message == WM_KEYDOWN && m.WParam == VK_P && popupVisible && pickerHwnd == 0 && currentTab == 0 {
			state, _, _ := pGetKeyState.Call(VK_CONTROL)
			if (state & 0x8000) != 0 {
				showPromptPicker()
				continue
			}
		}

		// Picker keyboard navigation (when picker is open)
		if m.Message == WM_KEYDOWN && pickerHwnd != 0 {
			// Down arrow in search box → move focus to listbox
			if m.Hwnd == pickerSearchHwnd && m.WParam == VK_DOWN {
				count, _, _ := pSendMessageW.Call(uintptr(pickerListHwnd), LB_GETCOUNT, 0, 0)
				if int(count) > 0 {
					cur, _, _ := pSendMessageW.Call(uintptr(pickerListHwnd), LB_GETCURSEL, 0, 0)
					if int(cur) == LB_ERR {
						pSendMessageW.Call(uintptr(pickerListHwnd), LB_SETCURSEL, 0, 0)
					}
					pSetFocus.Call(uintptr(pickerListHwnd))
				}
				continue
			}
			// Up arrow in listbox at first item → return focus to search box
			if m.Hwnd == pickerListHwnd && m.WParam == VK_UP {
				cur, _, _ := pSendMessageW.Call(uintptr(pickerListHwnd), LB_GETCURSEL, 0, 0)
				if int(cur) <= 0 {
					pSetFocus.Call(uintptr(pickerSearchHwnd))
					continue
				}
				// Otherwise fall through to native listbox Up handling
			}
			// Enter in listbox → insert selected prompt
			if m.Hwnd == pickerListHwnd && m.WParam == VK_RETURN {
				insertSelectedPrompt()
				continue
			}
			// Enter in search box → insert auto-selected first prompt
			if m.Hwnd == pickerSearchHwnd && m.WParam == VK_RETURN {
				insertSelectedPrompt()
				continue
			}
			// Esc when picker is open → close picker only (not main popup)
			if m.WParam == VK_ESCAPE {
				pDestroyWindow.Call(uintptr(pickerHwnd))
				continue
			}
		}

		// Ctrl+A in edit control → select all text
		if m.Message == WM_KEYDOWN && m.WParam == VK_A && m.Hwnd == editHwnd {
			state, _, _ := pGetKeyState.Call(VK_CONTROL)
			if (state & 0x8000) != 0 {
				pSendMessageW.Call(uintptr(editHwnd), EM_SETSEL, 0, uintptr(0xFFFFFFFF))
				continue
			}
		}

		// Esc (when picker is NOT open)
		if m.Message == WM_KEYDOWN && m.WParam == VK_ESCAPE && popupVisible {
			hidePopup()
			continue
		}

		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	logf("Exiting")
}
