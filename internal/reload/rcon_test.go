package reload

import (
	"net"
	"strings"
	"testing"
	"time"
)

// fakeSAMP simulates the SA-MP RCON protocol: banner, password check, "x"
// prompt and one command response.
func fakeSAMP(t *testing.T, password string, responses map[string]string) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		_, _ = conn.Write([]byte("----- SAMP RCON -----\n"))

		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		got := strings.TrimSpace(string(buf[:n]))
		if got != password {
			_, _ = conn.Write([]byte("Wrong password.\n"))
			return
		}
		_, _ = conn.Write([]byte("Logged in successfully.\n"))

		n, _ = conn.Read(buf)
		if strings.TrimSpace(string(buf[:n])) != "x" {
			_, _ = conn.Write([]byte("Wrong password.\n"))
			return
		}
		_, _ = conn.Write([]byte("\n"))

		n, _ = conn.Read(buf)
		cmd := strings.TrimSpace(string(buf[:n]))
		if resp, ok := responses[cmd]; ok {
			_, _ = conn.Write([]byte(resp + "\n\n"))
		} else {
			_, _ = conn.Write([]byte("Unknown command.\n\n"))
		}
		// keep the connection open briefly so the client sees the quiet window
		time.Sleep(500 * time.Millisecond)
	}()
	return "127.0.0.1", port
}

func TestRCONSuccess(t *testing.T) {
	host, port := fakeSAMP(t, "secret", map[string]string{"changemode mymode": "Gamemode set to: mymode"})
	rc := &RCON{Host: host, Port: port, Password: "secret", Timeout: 5 * time.Second}
	out, err := rc.Exec("changemode mymode")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if !strings.Contains(out, "Gamemode set to: mymode") {
		t.Errorf("out = %q", out)
	}
}

func TestRCONWrongPassword(t *testing.T) {
	host, port := fakeSAMP(t, "secret", map[string]string{})
	rc := &RCON{Host: host, Port: port, Password: "wrong", Timeout: 5 * time.Second}
	if _, err := rc.Exec("gmx"); err == nil {
		t.Error("expected authentication error")
	}
}

func TestIsUnknownCommand(t *testing.T) {
	if !IsUnknownCommand("Unknown command: changemode") {
		t.Error("expected unknown command detection")
	}
	if IsUnknownCommand("Gamemode set to: mymode") {
		t.Error("unexpected unknown detection")
	}
}
