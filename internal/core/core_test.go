package core

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"samplive/internal/adapter"
	"samplive/internal/compiler"
	"samplive/internal/config"
	"samplive/internal/detect"
)

// fakeCompiler writes a fake pawncc that always fails to a temp dir and
// returns its path. Works on Windows (.bat via cmd) and everywhere else
// (.sh).
func fakeCompiler(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		bin := filepath.Join(dir, "pawncc.bat")
		body := "@echo off\r\necho test.pwn(1) : error 025: bad code\r\nexit /b 1\r\n"
		if err := os.WriteFile(bin, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return bin
	}
	bin := filepath.Join(dir, "pawncc")
	body := "#!/bin/sh\necho 'test.pwn(1) : error 025: bad code'\nexit 1\n"
	if err := os.WriteFile(bin, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestRunOnceReturnsErrorOnFailedBuild(t *testing.T) {
	root := t.TempDir()
	gm := filepath.Join(root, "gamemodes")
	if err := os.MkdirAll(gm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gm, "demo.pwn"), []byte("main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := fakeCompiler(t)
	cfg := &config.Config{
		Project: config.Project{
			Root:     root,
			Gamemode: "demo",
			Debounce: "100ms",
			Compiler: config.Compiler{Path: bin},
		},
		Rcon: config.Rcon{Enabled: false},
	}
	info := &detect.RuntimeInfo{Kind: detect.KindSAMP, Root: root, GamemodesDir: "gamemodes", Port: 7777}
	c := &Core{
		cfg:     cfg,
		log:     log.New(os.Stderr, "", 0),
		comp:    compiler.New(bin, nil, nil),
		adapter: adapter.SAMPRuntime{},
		info:    info,
	}

	if err := c.RunOnce(); err == nil {
		t.Fatal("expected RunOnce to fail when the build fails")
	}
}
