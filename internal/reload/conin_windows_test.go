//go:build windows

package reload

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestConsoleInjection is a live test for the console:changemode reload. It
// starts a real open.mp server exactly like the OMP adapter does (detachConsole
// hands it a console stdin) and types commands into its console. It is skipped
// unless OMP_SERVER points to an omp-server executable; OMP_DIR optionally
// overrides the working directory, which must contain a config.json that loads
// the "demo" gamemode.
func TestConsoleInjection(t *testing.T) {
	srv := os.Getenv("OMP_SERVER")
	if srv == "" {
		t.Skip("OMP_SERVER not set; skipping live console-injection test")
	}
	dir := os.Getenv("OMP_DIR")
	if dir == "" {
		dir = filepath.Dir(srv)
	}

	cmd := exec.Command(srv)
	cmd.Dir = dir
	detachConsole(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer func() {
		exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/F", "/T").Run()
	}()

	logPath := filepath.Join(dir, "log.txt")
	os.Remove(logPath)
	waitFor(t, 30*time.Second, func() bool {
		b, err := os.ReadFile(logPath)
		return err == nil && strings.Contains(string(b), "Network started")
	})

	pid := uint32(cmd.Process.Pid)
	if err := InjectConsoleCommand(pid, "echo console-test-ok"); err != nil {
		t.Fatalf("inject echo: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		b, err := os.ReadFile(logPath)
		return err == nil && strings.Contains(string(b), "console-test-ok")
	})

	// changemode to the same gamemode must reload it in place.
	if err := InjectConsoleCommand(pid, "changemode demo"); err != nil {
		t.Fatalf("inject changemode: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		b, err := os.ReadFile(logPath)
		return err == nil && strings.Count(string(b), "SampLive demo started") >= 2
	})
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
