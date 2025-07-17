package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
)

type HWND uintptr

func enumWindows(enumFunc uintptr, lParam uintptr) bool {
	ret, _, _ := procEnumWindows.Call(enumFunc, lParam)
	return ret != 0
}

func isWindowVisible(hwnd HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

func getWindowProcessId(hwnd HWND) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

func setForegroundWindow(hwnd HWND) bool {
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return ret != 0
}

func findMainWindowByPID(targetPID uint32) HWND {
	var found HWND = 0

	cb := syscall.NewCallback(func(hwnd HWND, lParam uintptr) uintptr {
		if !isWindowVisible(hwnd) {
			return 1 // skip
		}
		pid := getWindowProcessId(hwnd)
		if pid == targetPID {
			found = hwnd
			return 0 // stop enumeration
		}
		return 1 // continue
	})

	enumWindows(cb, 0)
	return found
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

const (
	MAX_PATH           = 260
	TH32CS_SNAPPROCESS = 0x00000002
)

func main() {

	//fmt.Println(internal.GetLu4Pids())
	//return
	//fmt.Println(internal.GetPidData())
	//return
	var pid uint32 = 15208 // 🔁 заміни на потрібний PID

	hwnd := findMainWindowByPID(pid)
	if hwnd == 0 {
		fmt.Println("Window not found")
		return
	}

	if !setForegroundWindow(hwnd) {
		fmt.Println("Failed to set foreground window")
		return
	}
	//
	//fmt.Printf("Set window %v to foreground\n", hwnd)
}
