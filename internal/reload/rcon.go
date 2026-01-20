// Package reload implements the transport mechanisms SampLive uses to apply
// a compiled gamemode to a running server: SA-MP RCON over TCP and local
// server process control.
package reload

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	bannerTimeout  = 5 * time.Second
	quietThreshold = 200 * time.Millisecond
)

// RCON speaks the SA-MP RCON protocol over TCP. The fun part: there's no
// separate admin port, RCON shares the same port the game runs on.
//
// Protocol: the server sends a banner on connect, the client authenticates
// with the password, sends "x" to enter command mode, then sends one command
// per line. Output is read until the connection goes quiet. Not pretty, but
// it's the protocol we got.
type RCON struct {
	Host     string
	Port     int
	Password string
	Timeout  time.Duration
}

// Exec sends a single command and returns the server output.
func (r *RCON) Exec(command string) (string, error) {
	addr := net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
	conn, err := net.DialTimeout("tcp", addr, r.Timeout)
	if err != nil {
		return "", fmt.Errorf("rcon: dial %s: %w", addr, err)
	}
	defer conn.Close()

	// 1. banner sent by the server on connect.
	_, _ = readUntil(conn, bannerTimeout)

	// 2. authenticate.
	if _, err := conn.Write([]byte(r.Password + "\n")); err != nil {
		return "", fmt.Errorf("rcon: write password: %w", err)
	}
	resp, _ := readUntil(conn, bannerTimeout)
	if strings.Contains(resp, "Wrong password") || strings.Contains(resp, "denied") {
		return resp, fmt.Errorf("rcon: authentication failed")
	}

	// 3. enter command mode.
	_, _ = conn.Write([]byte("x\n"))
	_, _ = readUntil(conn, bannerTimeout)

	// 4. send the command and read the response.
	if _, err := conn.Write([]byte(command + "\n")); err != nil {
		return "", fmt.Errorf("rcon: write command: %w", err)
	}
	out := readQuiet(conn, quietThreshold)
	return strings.TrimSpace(out), nil
}

// readUntil reads bytes until a newline or timeout.
func readUntil(conn net.Conn, timeout time.Duration) (string, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var buf []byte
	tmp := make([]byte, 1)
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if tmp[0] == '\n' {
				return strings.TrimSpace(string(buf)), nil
			}
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return strings.TrimSpace(string(buf)), nil
			}
			return strings.TrimSpace(string(buf)), err
		}
	}
}

// readQuiet reads until no data arrives for the quiet window.
func readQuiet(conn net.Conn, quiet time.Duration) string {
	_ = conn.SetReadDeadline(time.Now().Add(quiet))
	var buf []byte
	tmp := make([]byte, 4096)
	last := time.Now()
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			last = time.Now()
			_ = conn.SetReadDeadline(time.Now().Add(quiet))
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() && time.Since(last) < quiet {
				_ = conn.SetReadDeadline(time.Now().Add(quiet))
				continue
			}
			return string(buf)
		}
	}
}

// IsUnknownCommand reports whether server output indicates the command is
// not recognized. Used by adapters whose runtime may not implement a command.
func IsUnknownCommand(output string) bool {
	lower := strings.ToLower(output)
	for _, hint := range []string{"unknown", "unrecognized", "not found", "no such", "invalid command", "unknown command"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
