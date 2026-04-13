//go:build windows

package reload

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	user32      = syscall.NewLazyDLL("user32.dll")
	procCreateF = kernel32.NewProc("CreateFileW")
	procAllocC  = kernel32.NewProc("AllocConsole")
	procGetCW   = kernel32.NewProc("GetConsoleWindow")
	procShowWin = user32.NewProc("ShowWindow")
)

const (
	conAccess       = 0xC0000000 // GENERIC_READ | GENERIC_WRITE
	conShare        = 0x3        // FILE_SHARE_READ | FILE_SHARE_WRITE
	conOpenExisting = 3
	conSwHide       = 0
)

// detach starts the server in its own process group so it keeps running even
// if SampLive exits.
func detach(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008}
}

// detachConsole starts the server in its own process group with a console
// stdin. open.mp reads changemode from its console input, so the server needs
// a console buffer there or console:changemode can't reach it. We reuse our
// own console when there is one; otherwise a hidden console is allocated.
func detachConsole(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP (0x00000200): new group, shared console.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
	if in := consoleInput(); in != nil {
		cmd.Stdin = in
	}
}

// consoleInput returns the console input buffer for the child's stdin, or nil
// when no console can be had. A hidden console is allocated when this process
// has none.
func consoleInput() *os.File {
	if h := openConsoleInput(); h != 0 {
		return os.NewFile(uintptr(h), "CONIN$")
	}
	procAllocC.Call()
	if hwnd, _, _ := procGetCW.Call(); hwnd != 0 {
		procShowWin.Call(hwnd, conSwHide)
	}
	if h := openConsoleInput(); h != 0 {
		return os.NewFile(uintptr(h), "CONIN$")
	}
	return nil
}

// openConsoleInput opens the current process's console input buffer, or 0
// when this process is not attached to a console.
func openConsoleInput() syscall.Handle {
	conin, _ := syscall.UTF16PtrFromString("CONIN$")
	h, _, _ := procCreateF.Call(
		uintptr(unsafe.Pointer(conin)), conAccess, conShare, 0, conOpenExisting, 0, 0)
	if h == 0 || h == ^uintptr(0) {
		return 0
	}
	return syscall.Handle(h)
}
