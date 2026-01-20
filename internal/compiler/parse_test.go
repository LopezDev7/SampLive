package compiler

import "testing"

func TestParseOutput(t *testing.T) {
	out := `Pawn compiler 3.2.3664	 	 	Copyright (c) 1997-2006, ITB CompuPhase
gamemodes/mode.pwn(12) : error 025: function definition does not match prototype
gamemodes/mode.pwn(14) : warning 203: symbol is never used: "x"
gamemodes/mode.pwn(19) : fatal error 100: cannot read from file: "a_samp"
1 Error.`

	entries, errCount, warnCount := ParseOutput(out)
	if errCount != 2 {
		t.Errorf("errCount = %d, want 2", errCount)
	}
	if warnCount != 1 {
		t.Errorf("warnCount = %d, want 1", warnCount)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if entries[0].Line != 12 || entries[0].Code != 25 || entries[0].Level != "error" {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[1].Code != 203 || entries[1].Level != "warning" {
		t.Errorf("entry[1] = %+v", entries[1])
	}
	if entries[2].Level != "fatal error" || entries[2].Code != 100 {
		t.Errorf("entry[2] = %+v", entries[2])
	}
}

func TestParseOutputFatalWithoutFile(t *testing.T) {
	out := "fatal error 100: cannot read from file: \"a_samp\"\nCompilation aborted."
	entries, errCount, _ := ParseOutput(out)
	if errCount != 1 {
		t.Errorf("errCount = %d, want 1", errCount)
	}
	if len(entries) != 1 || entries[0].Code != 100 {
		t.Errorf("entries = %+v", entries)
	}
}

func TestParseOutputClean(t *testing.T) {
	out := "Pawn compiler 3.2.3664\n\nHeader size:      4092 bytes\nCode size:       12345 bytes\n"
	entries, errCount, warnCount := ParseOutput(out)
	if len(entries) != 0 || errCount != 0 || warnCount != 0 {
		t.Errorf("expected clean output, got entries=%v errs=%d warns=%d", entries, errCount, warnCount)
	}
}
