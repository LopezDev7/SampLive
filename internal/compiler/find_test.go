package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFind(t *testing.T) {
	dir := t.TempDir()
	// SA-MP-style layout: server + pawno/pawncc.exe + pawno/includes
	if err := os.MkdirAll(filepath.Join(dir, "pawno", "includes"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "pawno", "pawncc.exe")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, includes, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Errorf("bin = %s, want %s", got, bin)
	}
	if len(includes) != 1 || includes[0] != filepath.Join(dir, "pawno", "includes") {
		t.Errorf("includes = %v", includes)
	}
}

func TestFindMissing(t *testing.T) {
	if _, _, err := Find(t.TempDir()); err == nil {
		t.Error("expected error when pawncc is missing")
	}
}
