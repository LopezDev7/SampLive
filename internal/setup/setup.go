// Package setup implements the interactive `samplive setup` wizard that
// discovers a server, its compiler and reload settings, then writes a
// working samplive.yaml.
package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"samplive/internal/compiler"
	"samplive/internal/config"
	"samplive/internal/detect"
)

type wizard struct {
	in  *bufio.Scanner
	out io.Writer
}

// Run walks the user through the configuration steps and writes outPath.
func Run(outPath string) error {
	w := &wizard{in: bufio.NewScanner(os.Stdin), out: os.Stdout}
	fmt.Fprintln(w.out, "SampLive setup - hot reload for Pawn servers")
	fmt.Fprintln(w.out, "I'll try to guess as much as I can. Press Enter to accept what I suggest.")
	fmt.Fprintln(w.out)

	cfg := &config.Config{
		Project: config.Project{Debounce: "300ms"},
		Rcon:    config.Rcon{Enabled: true, Host: "127.0.0.1"},
	}

	// 1. Server directory + runtime detection.
	root, err := w.ask("Where's your server? (folder)", ".")
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	cfg.Project.Root = abs

	info, detErr := detect.Detect(abs, "")
	if detErr != nil {
		fmt.Fprintf(w.out, "No local server files found in %q.\n", abs)
		force, err := w.ask("Runtime (\"samp\" or \"omp\")", "samp")
		if err != nil {
			return err
		}
		cfg.Runtime.Force = force
		info = &detect.RuntimeInfo{Kind: detect.Kind(force), Root: abs, GamemodesDir: "gamemodes", Port: 7777}
	} else {
		fmt.Fprintf(w.out, "Found it: %s (port %d)\n", info.Kind, info.Port)
	}

	// 2. Gamemode.
	gms := listGamemodes(abs)
	def := info.Gamemode
	if def == "" && len(gms) > 0 {
		def = gms[0]
	}
	if def == "" {
		def = "gamemode"
	}
	gm, err := w.ask("Which gamemode are we hot reloading?", def)
	if err != nil {
		return err
	}
	cfg.Project.Gamemode = gm

	// 3. Compiler + includes.
	bin, includes, findErr := compiler.Find(abs)
	if findErr != nil {
		bin, err = w.ask("Path to pawncc", "")
		if err != nil {
			return err
		}
		if _, err := os.Stat(bin); err != nil {
			return fmt.Errorf("pawncc not found at %s", bin)
		}
		incStr, err := w.ask("Include directories (comma separated)", "")
		if err != nil {
			return err
		}
		includes = splitList(incStr)
	} else {
		fmt.Fprintf(w.out, "Found your pawncc: %s\n", bin)
		if len(includes) > 0 {
			fmt.Fprintf(w.out, "Found includes: %s\n", strings.Join(includes, ", "))
		}
	}
	cfg.Project.Compiler.Path = bin
	cfg.Project.Compiler.Includes = includes

	// 4. Reload method.
	rcPass := info.RconPassword
	if !info.RconEnabled {
		rcPass = ""
	}
	method, err := w.ask("Reload method (\"rcon\" / \"restart\")", "rcon")
	if err != nil {
		return err
	}
	switch strings.ToLower(method) {
	case "rcon", "":
		cfg.Rcon.Enabled = true
		if info.Port != 0 {
			cfg.Rcon.Port = info.Port
		}
		pass, err := w.ask("RCON password (the server's rcon_password)", rcPass)
		if err != nil {
			return err
		}
		cfg.Rcon.Password = pass
		if cfg.Rcon.Password == "" {
			cfg.Rcon.Enabled = false
			fmt.Fprintln(w.out, "No RCON password? ok, then we'll need a server.command to restart things.")
		}
	case "restart":
		cfg.Rcon.Enabled = false
		cmd, err := w.ask("Command to start the server", "")
		if err != nil {
			return err
		}
		cfg.Server.Command = cmd
		args, err := w.ask("Arguments (space separated)", "")
		if err != nil {
			return err
		}
		cfg.Server.Args = splitList(args)
	default:
		return fmt.Errorf("unknown reload method %q", method)
	}

	// 5. Remote deploy (optional).
	remoteYN, err := w.ask("Deploy to a remote host? (y/N)", "N")
	if err != nil {
		return err
	}
	if strings.EqualFold(remoteYN, "y") || strings.EqualFold(remoteYN, "yes") {
		cfg.Remote.Enabled = true
		host, err := w.ask("Remote host", "")
		if err != nil {
			return err
		}
		cfg.Remote.Host = host
		user, err := w.ask("SSH user", "root")
		if err != nil {
			return err
		}
		cfg.Remote.User = user
		auth, err := w.ask("Password or keyfile path", "")
		if err != nil {
			return err
		}
		if strings.Contains(auth, "\\") || strings.Contains(auth, "/") || strings.HasSuffix(auth, ".pem") {
			cfg.Remote.Keyfile = auth
		} else if auth != "" {
			cfg.Remote.Password = auth
		}
		defPath := path.Join(info.GamemodesDir, gm+".amx")
		amxPath, err := w.ask("Remote .amx path", defPath)
		if err != nil {
			return err
		}
		cfg.Remote.AMXPath = amxPath
		restart, err := w.ask("SSH command to restart the server (optional)", "")
		if err != nil {
			return err
		}
		cfg.Remote.RestartCmd = restart
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(w.out, "\nConfig written to %s\n", outPath)
	fmt.Fprintln(w.out, "Next: run `samplive` in this directory and get coding.")
	return nil
}

// ask prints a prompt and reads one line, returning def when empty.
func (w *wizard) ask(prompt, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(w.out, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(w.out, "%s: ", prompt)
	}
	if !w.in.Scan() {
		if err := w.in.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	answer := strings.TrimSpace(w.in.Text())
	if answer == "" {
		return def, nil
	}
	return answer, nil
}

func listGamemodes(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "gamemodes"))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".pwn") {
			names = append(names, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	sort.Strings(names)
	return names
}

func splitList(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' })
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
