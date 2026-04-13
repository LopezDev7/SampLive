//go:build !windows

package reload

import (
	"os/exec"
	"syscall"
)

// detach starts the server in its own process group so it keeps running even
// if SampLive exits.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// detachConsole is the same as detach outside Windows: open.mp reads console
// commands from a redirected stdin pipe there, so no console juggling is
// needed.
func detachConsole(cmd *exec.Cmd) {
	detach(cmd)
}
