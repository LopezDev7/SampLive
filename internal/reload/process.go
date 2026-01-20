package reload

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// ProcessController restarts a local server process. It finds the process
// listening on the server port, kills it, then starts the configured command.
// This is the universal fallback reload method.
//
// Honest limitation: process control only works when SampLive runs on the
// same machine as the server. On Linux, seeing the PID of another process
// requires running as the server user (or root).
type ProcessController struct {
	Command string
	Args    []string
	Port    int
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
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("process: start %s: %w", p.Command, err)
	}
	return fmt.Sprintf("restarted %s (pid %d)", p.Command, cmd.Process.Pid), nil
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
	// Linux: prefer ss (modern distros), fall back to netstat -tlnp.
	if pid, err := findPIDSS(port); err != nil {
		return 0, err
	} else if pid != 0 {
		return pid, nil
	}
	if pid, err := findPIDNetstatLinux(port); err != nil {
		return 0, err
	} else {
		return pid, nil
	}
}

// findPIDWindows parses `netstat -ano` rows in LISTENING state.
func findPIDWindows(port int) (int, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return 0, err
	}
	target := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[4] == "LISTENING" {
			if pid, err := strconv.Atoi(fields[len(fields)-1]); err == nil && pid > 0 {
				return pid, nil
			}
		}
	}
	return 0, nil
}

// findPIDSS parses `ss -tlnp` (Linux). PIDs appear as "pid=1234".
func findPIDSS(port int) (int, error) {
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return 0, nil // ss unavailable; try netstat
	}
	target := fmt.Sprintf(":%d", port)
	pidRe := regexp.MustCompile(`pid=(\d+)`)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) || !strings.Contains(line, "LISTEN") {
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

// findPIDNetstatLinux parses `netstat -tlnp` (legacy Linux). The last field
// is "PID/Name".
func findPIDNetstatLinux(port int) (int, error) {
	out, err := exec.Command("netstat", "-tlnp").Output()
	if err != nil {
		return 0, err
	}
	target := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) || !strings.Contains(line, "LISTEN") {
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
