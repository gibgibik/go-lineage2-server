package internal

import (
	"fmt"
	"github.com/gibgibik/go-lineage2-server/internal/config"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	WS_EX_LAYERED      = 0x00080000
	WS_EX_TRANSPARENT  = 0x00000020
	WS_EX_TOPMOST      = 0x00000008
	WS_POPUP           = 0x80000000
	SW_SHOWNOACTIVATE  = 4
	TH32CS_SNAPPROCESS = 0x00000002
	PS_SOLID           = 0
	BKMODE_TRANSPARENT = 1
	MAX_PATH           = 260
	WM_GETTEXT         = 0x000D
	WM_GETTEXTLENGTH   = 0x000E
	SMTO_ABORTIFHUNG   = 0x0002
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procSetLayeredAttrs  = user32.NewProc("SetLayeredWindowAttributes")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procFillRect         = user32.NewProc("FillRect")

	procGetDC                    = user32.NewProc("GetDC")
	procReleaseDC                = user32.NewProc("ReleaseDC")
	procRectangle                = gdi32.NewProc("Rectangle")
	procGetStockObject           = gdi32.NewProc("GetStockObject")
	procCreatePen                = gdi32.NewProc("CreatePen")
	procSelectObject             = gdi32.NewProc("SelectObject")
	procDeleteObject             = gdi32.NewProc("DeleteObject")
	procSetTextColor             = gdi32.NewProc("SetTextColor")
	procSetBkMode                = gdi32.NewProc("SetBkMode")
	procCreateFont               = gdi32.NewProc("CreateFontW")
	procTextOutW                 = gdi32.NewProc("TextOutW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetLastError             = syscall.NewLazyDLL("kernel32.dll").NewProc("GetLastError")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	attachThreadInput            = user32.NewProc("AttachThreadInput")
	getForegroundWindow          = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessId     = user32.NewProc("GetWindowThreadProcessId")
	getCurrentThreadId           = kernel32.NewProc("GetCurrentThreadId")
	setForegroundWindowInternal  = user32.NewProc("SetForegroundWindow")
	switchToThisWindow           = user32.NewProc("SwitchToThisWindow")
	procSetFocus                 = user32.NewProc("SetFocus")

	Hwnd          uintptr
	enumerateLock sync.Mutex
)

type POINT struct{ X, Y int32 }
type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type PROCESSENTRY32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

func InitWinApi(mainRun func(hwnd uintptr)) {
	className := syscall.StringToUTF16Ptr("TransparentOverlay")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	// Minimal WndProc
	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		return callDefWindowProc(hwnd, msg, wParam, lParam)
	})
	// Register class
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   wndProc,
		HInstance:     syscall.Handle(hInstance),
		LpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Create overlay window
	Hwnd, _, _ = procCreateWindowExW.Call(
		WS_EX_LAYERED|WS_EX_TRANSPARENT|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		0,
		WS_POPUP,
		0, 0, uintptr(config.Cnf.ClientConfig.Resolution[0]), uintptr(config.Cnf.ClientConfig.Resolution[1]),
		0, 0, hInstance, 0,
	)

	// Make transparent
	procSetLayeredAttrs.Call(Hwnd, 0, 0, 0x00000001)

	// Show
	procShowWindow.Call(Hwnd, SW_SHOWNOACTIVATE)
	procUpdateWindow.Call(Hwnd)
	var msg MSG
	mainRun(Hwnd)
	//fmt.Println("end")
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			//fmt.Println("win window exit")
			return
		}
		if int32(r) != 0 {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func Draw(hwnd uintptr, left uintptr, top uintptr, right uintptr, bottom uintptr, text string) {
	hdc, _, _ := procGetDC.Call(hwnd)
	if left != 0 && top != 0 && right != 0 && bottom != 0 {

		// Create red pen (BGR format: 0x00RRGGBB → 0x000000FF = red)
		pen, _, _ := procCreatePen.Call(PS_SOLID, 3, 0x008000)
		oldPen, _, _ := procSelectObject.Call(hdc, pen)

		// Select NULL_BRUSH to avoid filling the rectangle
		nullBrush, _, _ := procGetStockObject.Call(5)
		oldBrush, _, _ := procSelectObject.Call(hdc, nullBrush)

		// Draw transparent (non-filled) red rectangle
		procRectangle.Call(hdc, left, top, right, bottom)
		procSelectObject.Call(hdc, oldPen)
		procSelectObject.Call(hdc, oldBrush)
		procDeleteObject.Call(pen)
	}

	if text != "" {
		font := createFont(48)
		hdc, _, _ := procGetDC.Call(hwnd)
		procSelectObject.Call(hdc, font)
		defer procDeleteObject.Call(font) //
		procSetTextColor.Call(hdc, 0x00FF00)
		procSetBkMode.Call(hdc, BKMODE_TRANSPARENT)
		tx := syscall.StringToUTF16Ptr(text)
		procTextOutW.Call(hdc, 770, 100, uintptr(unsafe.Pointer(tx)), uintptr(len(text)))
	}
	// Optionally draw text (commented out for clarity)
	//
	//

	// Cleanup

	procReleaseDC.Call(hwnd, hdc)
}

func createFont(height int32) uintptr {
	hFont, _, _ := procCreateFont.Call(
		uintptr(height), 0, 0, 0, // height, width, escapement, orientation
		400, 0, 0, 0, // weight, italic, underline, strikeout
		1, 0, 0, 0, // charset, outPrecision, clipPrecision, quality
		uintptr(1), // pitchAndFamily
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Arial"))),
	)
	return hFont
}

func ClearOverlay(hwnd uintptr) {
	var rect struct {
		Left, Top, Right, Bottom int32
	}
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	hdc, _, _ := procGetDC.Call(hwnd)

	// Brush: match SetLayeredWindowAttributes color key (e.g., black)
	brush, _, _ := procCreateSolidBrush.Call(0x000000)

	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect)), brush)

	procDeleteObject.Call(brush)
	procReleaseDC.Call(hwnd, hdc)
}

func callDefWindowProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func GetPids() map[uint32]string {
	var result = make(map[uint32]string, 0)
	PidsMap = make(map[uint32]uintptr)
	cb := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		//fmt.Println("tick", hwnd)
		if !isWindowVisible(hwnd) {
			return 1 // skip
		}
		pid := getWindowProcessId(hwnd)
		//const maxCount = 250
		//buf := make([]uint16, maxCount)
		procGetClassNameW := user32.NewProc("GetClassNameW")
		clsBuf := make([]uint16, 256)
		procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&clsBuf[0])), uintptr(len(clsBuf)))
		//fmt.Println("ClassName:", syscall.UTF16ToString(clsBuf), " ", pid)
		//w := GetWindowTextW(hwnd)
		//fmt.Println("window text:", w)
		//fmt.Println(pid, title)
		//if ret == 0 {
		//	fmt.Println(err)
		//	//return 1
		//	//return "", fmt.Errorf("GetWindowTextW failed: %v", err)
		//} else {
		//if pid == 26228 {
		if syscall.UTF16ToString(clsBuf) == "mwUnrealWWindowsViewportWindow" {
			title := getWindowTextSafe(hwnd)
			fmt.Println(pid, hwnd)
			result[pid] = title
			PidsMap[pid] = hwnd
		}
		//}
		return 1 // continue
	})

	enumWindows(cb, 0)
	//fmt.Println("pids filled")
	return result
}

func GetWindowTextW(hwnd uintptr) string {
	buf := make([]uint16, 255)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func enumWindows(enumFunc uintptr, lParam uintptr) bool {
	ret, _, _ := procEnumWindows.Call(enumFunc, lParam)
	return ret != 0
}

func isWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

func getWindowProcessId(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

func getForegroundWindowHwnd() uintptr {
	fgHwnd, _, _ := getForegroundWindow.Call()
	return fgHwnd
}

func setForegroundWindow(hwnd uintptr) {

	//switchToThisWindow.Call(hwnd, 1) // ✅ use this!
	//setForegroundWindowInternal.Call(hwnd)
	//return
	fgHwnd, _, _ := getForegroundWindow.Call()
	var fgThreadID, _, _ = getWindowThreadProcessId.Call(fgHwnd, 0)
	curThreadID, _, _ := getCurrentThreadId.Call()

	// Attach input
	attachThreadInput.Call(curThreadID, fgThreadID, 1)
	procSetFocus.Call(uintptr(hwnd))
	procSetForegroundWindow.Call(uintptr(hwnd))
	setForegroundWindowInternal.Call(uintptr(hwnd))
	attachThreadInput.Call(curThreadID, fgThreadID, 0)
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	//return ret != 0
}

func getWindowTextSafe(hwnd uintptr) string {
	// Get length first
	length, _, _ := procSendMessageTimeoutW.Call(
		hwnd,
		uintptr(WM_GETTEXTLENGTH),
		0,
		0,
		SMTO_ABORTIFHUNG,
		500, // timeout in ms
		0,
	)
	if length == 0 {
		return ""
	}

	// Allocate buffer
	length = 256
	buf := make([]uint16, length+1)
	ret, _, _ := procSendMessageTimeoutW.Call(
		hwnd,
		uintptr(WM_GETTEXT),
		length+1,
		uintptr(unsafe.Pointer(&buf[0])),
		SMTO_ABORTIFHUNG,
		500,
		0,
	)
	//fmt.Printf("HWND: 0x%X, Text: %q, Length: %d, ret: %d\newOcrClient", hwnd, syscall.UTF16ToString(buf), length, ret)

	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
