//go:build !windows

package reload

import "fmt"

// InjectConsoleCommand is a no-op outside Windows. open.mp reads console
// commands from stdin on Linux, so a redirected stdin pipe is the way to send
// them there; console-input injection only exists on Windows.
func InjectConsoleCommand(pid uint32, command string) error {
	return fmt.Errorf("console injection is not supported on this platform")
}
