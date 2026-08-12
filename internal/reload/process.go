package reload

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// ProcessController restarts a local server process: find the PID on the
// server port, kill it, start the configured command. The universal fallback
// reload method. Only works when SampLive runs on the same machine as the
// server (on Linux, seeing another process's PID needs the server user).
type ProcessController struct {
	Command string
	Args    []string
	Dir     string
	Port    int
	// Console starts the server attached to its own hidden console instead of
	// detached. open.mp on Windows reads commands from the console input
	// buffer, so a server started detached can only be reloaded by restarting
	// it. Starting it with a console enables the console:changemode reload.
	Console bool
}

// Restart kills the process on Port (if any) and starts Command+Args.
func (p *ProcessController) Restart() (string, error) {
	pid, err := findPIDOnPort(p.Port)
	if err != nil {
		return "", fmt.Errorf("process: find pid on port %d: %w", p.Port, err)
	}
	if pid != 0 {
		if err := killPID(pid); err != nil {
			return "", fmt.Errorf("process: kill pid %d: %w", pid, err)
		}
	}
	cmd := exec.Command(p.Command, p.Args...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	if p.Console {
		detachConsole(cmd)
	} else {
		detach(cmd)
	}
	err = cmd.Start()
	// The child holds an inherited duplicate of its stdin handle; the copy we
	// opened for it can be released now to avoid leaking a handle per restart.
	if f, ok := cmd.Stdin.(*os.File); ok {
		f.Close()
	}
	if err != nil {
		return "", fmt.Errorf("process: start %s: %w", p.Command, err)
	}
	return fmt.Sprintf("restarted %s (pid %d)", p.Command, cmd.Process.Pid), nil
}

// FindPIDOnPort returns the PID of the local process listening on port, or 0
// when nothing is listening.
func FindPIDOnPort(port int) (int, error) {
	return findPIDOnPort(port)
}

// InjectConsoleCommandTo spawns `samplive console <pid> <command>` so the
// long-running SampLive process keeps its own terminal: the helper attaches
// to the server's console, types the command and exits.
func InjectConsoleCommandTo(pid uint32, command string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("console: locate samplive: %w", err)
	}
	cmd := exec.Command(exe, "console", strconv.FormatUint(uint64(pid), 10), command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("console injection into pid %d: %w (%s)", pid, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// findPIDOnPort returns the PID of the local process listening on port, or 0
// when nothing is listening. Errors when the port is in use but the PID is
// not visible to us.
func findPIDOnPort(port int) (int, error) {
	if port <= 0 {
		return 0, nil
	}
	if runtime.GOOS == "windows" {
		return findPIDWindows(port)
	}
	// Linux: prefer ss (modern distros), fall back to netstat. SA-MP binds
	// TCP (RCON), but open.mp only binds a UDP socket, so check UDP after
	// TCP; the parsing is identical apart from the LISTEN state check.
	for _, udp := range []bool{false, true} {
		if pid, err := findPIDSS(port, udp); err != nil {
			return 0, err
		} else if pid != 0 {
			return pid, nil
		}
	}
	for _, udp := range []bool{false, true} {
		if pid, err := findPIDNetstatLinux(port, udp); err != nil {
			return 0, err
		} else if pid != 0 {
			return pid, nil
		}
	}
	return 0, nil
}

// findPIDWindows parses `netstat -ano`. Matches TCP rows in LISTENING state
// and UDP rows, because open.mp only binds a UDP socket (no TCP RCON).
func findPIDWindows(port int) (int, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return 0, err
	}
	return parseNetstat(out, port)
}

func parseNetstat(out []byte, port int) (int, error) {
	target := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "TCP":
			// TCP rows: proto, local, foreign, state, pid
			if len(fields) >= 5 && fields[3] == "LISTENING" {
				if pid, err := strconv.Atoi(fields[len(fields)-1]); err == nil && pid > 0 {
					return pid, nil
				}
			}
		case "UDP":
			// UDP rows have no state column; the PID is the last field.
			if len(fields) >= 4 {
				if pid, err := strconv.Atoi(fields[len(fields)-1]); err == nil && pid > 0 {
					return pid, nil
				}
			}
		}
	}
	return 0, nil
}

// findPIDSS parses `ss` output (Linux). PIDs appear as "pid=1234". udp
// selects UDP sockets (open.mp binds UDP only): UDP rows have no LISTEN
// state to filter on.
func findPIDSS(port int, udp bool) (int, error) {
	flags := "-tlnp"
	if udp {
		flags = "-ulnp"
	}
	out, err := exec.Command("ss", flags).Output()
	if err != nil {
		return 0, nil // ss unavailable; try netstat
	}
	return parseSS(out, port, udp)
}

func parseSS(out []byte, port int, udp bool) (int, error) {
	target := fmt.Sprintf(":%d", port)
	pidRe := regexp.MustCompile(`pid=(\d+)`)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) {
			continue
		}
		if !udp && !strings.Contains(line, "LISTEN") {
			continue
		}
		if m := pidRe.FindStringSubmatch(line); m != nil {
			if pid, _ := strconv.Atoi(m[1]); pid > 0 {
				return pid, nil
			}
		}
		return 0, fmt.Errorf("port %d is in use but its PID is not visible (run SampLive as the server user or root)", port)
	}
	return 0, nil
}

// findPIDNetstatLinux parses `netstat` output (legacy Linux). The last field
// is "PID/Name"; UDP rows have no state column.
func findPIDNetstatLinux(port int, udp bool) (int, error) {
	flags := "-tlnp"
	if udp {
		flags = "-ulnp"
	}
	out, err := exec.Command("netstat", flags).Output()
	if err != nil {
		return 0, err
	}
	return parseNetstatLinux(out, port, udp)
}

func parseNetstatLinux(out []byte, port int, udp bool) (int, error) {
	target := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) {
			continue
		}
		if !udp && !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		last := fields[len(fields)-1] // e.g. "1234/omp-server" or "-"
		if last == "-" {
			return 0, fmt.Errorf("port %d is in use but its PID is not visible (run SampLive as the server user or root)", port)
		}
		if idx := strings.Index(last, "/"); idx > 0 {
			last = last[:idx]
		}
		if pid, err := strconv.Atoi(last); err == nil && pid > 0 {
			return pid, nil
		}
	}
	return 0, nil
}

func killPID(pid int) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F", "/T").Run()
	}
	return exec.Command("kill", strconv.Itoa(pid)).Run()
}
