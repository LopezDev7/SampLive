package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectSAMP(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "server.cfg"), "echo Executing Server Config...\ngamemode0 grandlarc 1\nrcon_password secret\nport 7777\nmaxplayers 100\n")
	writeFile(t, filepath.Join(dir, "samp-server.exe"), "x")

	info, err := Detect(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != KindSAMP {
		t.Errorf("kind = %s, want samp", info.Kind)
	}
	if info.Gamemode != "grandlarc" {
		t.Errorf("gamemode = %q, want grandlarc", info.Gamemode)
	}
	if !info.RconEnabled || info.RconPassword != "secret" {
		t.Errorf("rcon = %+v", info)
	}
	if info.Port != 7777 || info.MaxPlayers != 100 {
		t.Errorf("port/maxplayers = %d/%d", info.Port, info.MaxPlayers)
	}
}

func TestDetectOMP(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"),
		`{"server":{"port":8123,"mode":"mymode"},"rcon":{"enabled":true,"password":"pw"}}`)
	writeFile(t, filepath.Join(dir, "omp-server"), "x")

	info, err := Detect(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != KindOMP {
		t.Errorf("kind = %s, want omp", info.Kind)
	}
	if info.Gamemode != "mymode" || info.Port != 8123 {
		t.Errorf("info = %+v", info)
	}
	if !info.RconEnabled || info.RconPassword != "pw" {
		t.Errorf("rcon = %+v", info)
	}
}

func TestDetectUnknown(t *testing.T) {
	dir := t.TempDir()
	if _, err := Detect(dir, ""); err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestForce(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), `{"rcon":{"password":"pw"}}`)
	info, err := Detect(dir, "omp")
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != KindOMP {
		t.Errorf("kind = %s, want omp", info.Kind)
	}
	if _, err := Detect(dir, "samp"); err == nil {
		t.Error("expected error forcing samp on an omp directory")
	}
	if _, err := Detect(dir, "bogus"); err == nil {
		t.Error("expected error for bogus forced kind")
	}
}
