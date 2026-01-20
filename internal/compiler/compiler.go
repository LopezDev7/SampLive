// Package compiler wraps the Pawn compiler (pawncc) and parses its output.
package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Result describes a single compile run.
type Result struct {
	Success bool
	Errors  []Error
	Output  string
}

// Compiler runs pawncc for one project. Each runtime ships its own compiler
// and include set, so the binary path and includes come from the config.
type Compiler struct {
	bin      string
	includes []string
	flags    []string
}

// New returns a Compiler bound to the given pawncc binary.
func New(bin string, includes, flags []string) *Compiler {
	return &Compiler{bin: bin, includes: includes, flags: flags}
}

// Check verifies that the configured compiler binary exists.
func (c *Compiler) Check() error {
	if c.bin == "" {
		return fmt.Errorf("compiler binary not configured (project.compiler.path)")
	}
	info, err := os.Stat(c.bin)
	if err != nil || info.IsDir() {
		return fmt.Errorf("compiler binary not found: %s", c.bin)
	}
	return nil
}

func (c *Compiler) args(pwn, amx string) []string {
	args := append([]string{}, c.flags...)
	args = append(args, pwn)
	if amx != "" {
		args = append(args, "-o"+amx)
	}
	for _, inc := range c.includes {
		args = append(args, "-i"+inc)
	}
	return args
}

// Compile builds pwn into amx and parses the compiler output. pawncc can
// write to both stdout and stderr depending on its mood, so we capture both
// into one buffer and parse whatever comes out.
func (c *Compiler) Compile(pwn, amx string) (*Result, error) {
	cmd := exec.Command(c.bin, c.args(pwn, amx)...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()

	output := buf.String()
	entries, errCount, _ := ParseOutput(output)
	return &Result{
		Success: runErr == nil && errCount == 0,
		Errors:  entries,
		Output:  output,
	}, nil
}
