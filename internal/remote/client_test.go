package remote

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"samplive/internal/config"
)

func newTestKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signer.PublicKey()
}

func TestHostKeyCallbackSkip(t *testing.T) {
	cb, err := hostKeyCallback(&config.Remote{InsecureSkipHostKey: true}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb == nil {
		t.Fatal("nil callback")
	}
}

func TestHostKeyCallbackMissingFile(t *testing.T) {
	cfg := &config.Remote{Host: "myvps.example.com"}
	missing := filepath.Join(t.TempDir(), "known_hosts")
	_, err := hostKeyCallback(cfg, missing)
	if err == nil {
		t.Fatal("expected an error for a missing known_hosts file")
	}
	if !strings.Contains(err.Error(), "ssh-keyscan") {
		t.Fatalf("error should tell the user to use ssh-keyscan, got: %v", err)
	}
}

func TestHostKeyCallbackAcceptReject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	host := "myvps.example.com:22"
	good := newTestKey(t)

	line := knownhosts.Line([]string{host}, good)
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	cb, err := hostKeyCallback(&config.Remote{Host: "myvps.example.com"}, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	addr := &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 22}
	if err := cb(host, addr, good); err != nil {
		t.Fatalf("expected the known key to pass, got: %v", err)
	}

	other := newTestKey(t)
	if err := cb(host, addr, other); err == nil {
		t.Fatal("expected a wrong key to be rejected")
	}

	unknown := newTestKey(t)
	if err := cb("otherhost:22", addr, unknown); err == nil {
		t.Fatal("expected an unknown host to be rejected")
	} else if !strings.Contains(err.Error(), "ssh-keyscan") {
		t.Fatalf("error should mention ssh-keyscan, got: %v", err)
	}
}
