package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Find locates a pawncc binary and its include directory near the project
// root. It checks the server directory, its pawno/compiler subfolders, the
// parent directory and PATH.
//
// The parent check is there because SA-MP packages are weird: the server
// lives in one folder and pawno sits next to it, not inside it.
func Find(root string) (bin string, includes []string, err error) {
	names := []string{"pawncc", "pawncc.exe"}
	var candidates []string
	for _, name := range names {
		candidates = append(candidates,
			filepath.Join(root, name),
			filepath.Join(root, "pawno", name),
			filepath.Join(root, "compiler", name),
			filepath.Join(root, "..", "pawno", name),
		)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			bin = c
			break
		}
	}
	if bin == "" {
		if p, err := exec.LookPath("pawncc"); err == nil {
			bin = p
		}
	}
	if bin == "" {
		return "", nil, fmt.Errorf("pawncc not found: place it under %q (or its pawno/compiler subfolder), or add it to PATH, then re-run setup", root)
	}

	includeCandidates := []string{
		filepath.Join(filepath.Dir(bin), "includes"),
		filepath.Join(filepath.Dir(bin), "..", "includes"),
		filepath.Join(root, "includes"),
		filepath.Join(root, "pawno", "includes"),
	}
	seen := map[string]bool{}
	for _, c := range includeCandidates {
		c = filepath.Clean(c)
		if seen[c] {
			continue
		}
		seen[c] = true
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			includes = append(includes, c)
		}
	}
	return bin, includes, nil
}
