package compiler

import (
	"regexp"
	"strconv"
	"strings"
)

// Error is a single diagnostic emitted by pawncc.
type Error struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Level   string `json:"level"` // "error" | "warning" | "fatal error"
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// pawncc prints things like "gamemodes\mymode.pwn(14) : error 025: undefined
// symbol" (or a bare "fatal error 100: cannot read from file"). The two
// regexes below cover both shapes. Yes, the spaces around the colon are
// inconsistent in pawncc output, that's why the parsing is so careful.
var (
	reFileLine = regexp.MustCompile(`^(.+)\((\d+)\)\s*:\s*(error|warning|fatal error)\s+(\d+):\s?(.*)$`)
	reFatal    = regexp.MustCompile(`^fatal error\s+(\d+):\s?(.*)$`)
)

// ParseOutput extracts diagnostics from raw pawncc output.
// It returns the parsed entries, the number of errors and the number of warnings.
func ParseOutput(output string) ([]Error, int, int) {
	var entries []Error
	errCount, warnCount := 0, 0
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		if m := reFileLine.FindStringSubmatch(line); m != nil {
			ln, _ := strconv.Atoi(m[2])
			code, _ := strconv.Atoi(m[4])
			e := Error{File: m[1], Line: ln, Level: m[3], Code: code, Message: strings.TrimSpace(m[5])}
			entries = append(entries, e)
			if e.Level == "error" || e.Level == "fatal error" {
				errCount++
			} else {
				warnCount++
			}
			continue
		}
		if m := reFatal.FindStringSubmatch(line); m != nil {
			code, _ := strconv.Atoi(m[1])
			entries = append(entries, Error{Level: "fatal error", Code: code, Message: strings.TrimSpace(m[2])})
			errCount++
		}
	}
	return entries, errCount, warnCount
}
