//go:build windows

package reload

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Console-input injection for open.mp on Windows.
//
// open.mp reads console commands via ReadConsoleW on the console input
// buffer, so a redirected stdin pipe is ignored. "Typing" means attaching to
// the target's console and writing INPUT_RECORDs, like real keystrokes. The
// input handle must come from CreateFile("CONIN$") after AttachConsole:
// GetStdHandle can still point at the caller's old console, so writes
// "succeed" without reaching the server.
//
// AttachConsole detaches the caller from its own console, so run this from a
// short-lived helper process (InjectConsoleCommandTo), never from SampLive.

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procFreeConsole        = modKernel32.NewProc("FreeConsole")
	procAttachConsole      = modKernel32.NewProc("AttachConsole")
	procCreateFileW        = modKernel32.NewProc("CreateFileW")
	procWriteConsoleInputW = modKernel32.NewProc("WriteConsoleInputW")
	procCloseHandle        = modKernel32.NewProc("CloseHandle")
)

const (
	// "CONIN$" needs read+write so the server's ReadConsoleW callers can read.
	accessGenericRead  = 0x80000000
	accessGenericWrite = 0x40000000
	shareRead          = 0x1
	shareWrite         = 0x2
	openExisting       = 3
)

// KEY_EVENT_RECORD: BOOL(4) WORD(2) WORD(2) WORD(2) WCHAR(2) DWORD(4) = 16 bytes.
type keyEventRecord struct {
	bKeyDown          int32
	wRepeatCount      uint16
	wVirtualKeyCode   uint16
	wVirtualScanCode  uint16
	uChar             uint16
	dwControlKeyState uint32
}

// INPUT_RECORD: WORD EventType (2) + 2 bytes padding + KEY_EVENT_RECORD (16)
// = 20 bytes. Matches the native layout so the array can go to
// WriteConsoleInputW as a single buffer.
type inputRecord struct {
	eventType uint16
	_         uint16
	keyEvent  keyEventRecord
}

// InjectConsoleCommand types command followed by Enter into the console input
// buffer of the process with the given PID.
func InjectConsoleCommand(pid uint32, command string) error {
	// AttachConsole fails while we already have a console, so detach first.
	// FreeConsole on a process without a console is a no-op.
	procFreeConsole.Call()

	r, _, err := procAttachConsole.Call(uintptr(pid))
	if r == 0 {
		return winErr(fmt.Sprintf("attach to console of pid %d", pid), err)
	}
	defer procFreeConsole.Call()

	conin, err := syscall.UTF16PtrFromString("CONIN$")
	if err != nil {
		return fmt.Errorf("console input path: %w", err)
	}
	h, _, _ := procCreateFileW.Call(
		uintptr(unsafe.Pointer(conin)),
		accessGenericRead|accessGenericWrite,
		shareRead|shareWrite,
		0,
		openExisting,
		0,
		0,
	)
	if h == 0 || h == ^uintptr(0) {
		return winErr(fmt.Sprintf("open console input of pid %d", pid), err)
	}
	defer procCloseHandle.Call(h)

	records := make([]inputRecord, 0, len(command)+1)
	for i := 0; i < len(command); i++ {
		records = append(records, inputRecord{
			eventType: 1, // KEY_EVENT
			keyEvent: keyEventRecord{
				bKeyDown:     1,
				wRepeatCount: 1,
				uChar:        uint16(command[i]),
			},
		})
	}
	records = append(records, inputRecord{
		eventType: 1,
		keyEvent: keyEventRecord{
			bKeyDown:        1,
			wRepeatCount:    1,
			wVirtualKeyCode: 0x0D, // VK_RETURN
			uChar:           '\r',
		},
	})

	var written uint32
	r, _, err = procWriteConsoleInputW.Call(
		h,
		uintptr(unsafe.Pointer(&records[0])),
		uintptr(len(records)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r == 0 {
		return winErr("write console input", err)
	}
	return nil
}

// winErr turns a failed Win32 call into an error, including the system error
// when one was reported.
func winErr(action string, err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno != 0 {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: failed", action)
}
