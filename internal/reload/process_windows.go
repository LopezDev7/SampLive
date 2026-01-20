//go:build windows

package reload

import (
	"os/exec"
	"syscall"
)

// detach starts the server in its own process group so it keeps running even
// if SampLive exits.
func detach(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP (0x00000200) | DETACHED_PROCESS (0x00000008)
	const flags = 0x00000200 | 0x00000008
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
}
