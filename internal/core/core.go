// Package core wires the pipeline together:
//
//	save .pwn/.inc -> watch -> compile -> report errors -> locate .amx
//	-> detect runtime -> reload (local adapter or remote deploy + RCON)
package core

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"samplive/internal/adapter"
	"samplive/internal/compiler"
	"samplive/internal/config"
	"samplive/internal/detect"
	"samplive/internal/reload"
	"samplive/internal/remote"
	"samplive/internal/watcher"
)

// Core is the running SampLive instance.
type Core struct {
	cfg     *config.Config
	log     *log.Logger
	comp    *compiler.Compiler
	adapter adapter.RuntimeAdapter
	info    *detect.RuntimeInfo
}

// New builds a Core: checks the compiler, detects the runtime and resolves
// the matching adapter. Remote-only projects (no local server files) require
// runtime.force.
func New(cfg *config.Config, logger *log.Logger) (*Core, error) {
	c := &Core{cfg: cfg, log: logger}
	includes := make([]string, 0, len(cfg.Project.Compiler.Includes))
	for _, inc := range cfg.Project.Compiler.Includes {
		includes = append(includes, cfg.AbsPath(inc))
	}
	c.comp = compiler.New(cfg.Project.Compiler.Path, includes, cfg.Project.Compiler.Flags)
	if err := c.comp.Check(); err != nil {
		return nil, err
	}

	info, err := detect.Detect(cfg.Project.Root, cfg.Runtime.Force)
	if err != nil {
		if !cfg.Remote.Enabled {
			return nil, err
		}
		kind := detect.Kind(strings.ToLower(strings.TrimSpace(cfg.Runtime.Force)))
		if kind != detect.KindSAMP && kind != detect.KindOMP {
			return nil, fmt.Errorf("%v; for remote-only projects set runtime.force to \"samp\" or \"omp\"", err)
		}
		port := cfg.Rcon.Port
		if port == 0 {
			port = 7777
		}
		info = &detect.RuntimeInfo{Kind: kind, Root: cfg.Project.Root, GamemodesDir: "gamemodes", Port: port, RconEnabled: true}
		logger.Printf("no local server found; running remote-only with runtime.force=%s", kind)
	}
	ad, err := adapter.For(info.Kind)
	if err != nil {
		return nil, err
	}
	c.adapter = ad
	c.info = info
	return c, nil
}

// Info returns the detected runtime information.
func (c *Core) Info() *detect.RuntimeInfo { return c.info }

// RunOnce compiles the gamemode and reloads it a single time. It returns an
// error when the build fails so scripts can tell the difference.
func (c *Core) RunOnce() error {
	pwn, amx, err := c.gamemodePaths()
	if err != nil {
		return err
	}
	ok, err := c.cycle(pwn, amx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("compilation failed with errors")
	}
	return nil
}

// CompileOnce compiles the gamemode and reports the result without reloading.
func (c *Core) CompileOnce() error {
	pwn, amx, err := c.gamemodePaths()
	if err != nil {
		return err
	}
	_, ok, err := c.compileAndReport(pwn, amx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("compilation failed with errors")
	}
	return nil
}

// Run watches for changes and reloads on each save.
func (c *Core) Run() error {
	pwn, amx, err := c.gamemodePaths()
	if err != nil {
		return err
	}
	w, err := watcher.New(c.watchDirs(), []string{".pwn", ".inc"}, c.cfg.DebounceDuration())
	if err != nil {
		return err
	}
	defer w.Close()

	c.log.Printf("runtime: %s (port %d, gamemode %q)", c.info.Kind, c.info.Port, c.info.Gamemode)
	c.log.Printf("adapter capabilities: %+v", c.adapter.Capabilities())
	c.log.Printf("watching: %v (.pwn/.inc)", c.watchDirs())
	if c.cfg.Remote.Enabled {
		c.log.Printf("remote deploy: enabled -> %s@%s (%s)", c.cfg.Remote.User, c.cfg.Remote.Host, c.cfg.Remote.AMXPath)
	}

	w.Start()
	for ev := range w.Events() {
		c.log.Printf("file changed: %s", ev.Path)
		if _, err := c.cycle(pwn, amx); err != nil {
			c.log.Printf("ERROR: %v", err)
		}
	}
	return nil
}

// cycle dispatches to the local or remote pipeline. The bool reports whether
// the build succeeded (a failed build never reloads the server).
func (c *Core) cycle(pwn, amx string) (bool, error) {
	if c.cfg.Remote.Enabled {
		return c.cycleRemote(pwn, amx)
	}
	return c.cycleLocal(pwn, amx)
}

// cycleLocal compiles and reloads via the local runtime adapter.
func (c *Core) cycleLocal(pwn, amx string) (bool, error) {
	amx, ok, err := c.compileAndReport(pwn, amx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	rr, err := c.adapter.Reload(c.info, c.cfg, amx)
	if err != nil {
		return true, fmt.Errorf("reload: %w", err)
	}
	if rr.Preserved {
		c.log.Printf("reloaded via %s (players stay connected)", rr.Method)
	} else {
		c.log.Printf("reloaded via %s (state was reset, that's normal)", rr.Method)
	}
	if rr.Output != "" {
		c.log.Printf("server: %s", rr.Output)
	}
	return true, nil
}

// cycleRemote compiles locally, uploads the .amx over SFTP and reloads the
// remote server over RCON (or an SSH restart command when RCON can't).
func (c *Core) cycleRemote(pwn, amx string) (bool, error) {
	amx, ok, err := c.compileAndReport(pwn, amx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	cli, err := remote.Connect(&c.cfg.Remote)
	if err != nil {
		return true, err
	}
	defer cli.Close()

	remotePath := c.remoteAMXPath()
	if err := cli.Upload(amx, remotePath); err != nil {
		return true, err
	}
	c.log.Printf("uploaded %s -> %s", amx, remotePath)

	name := gamemodeFromAMX(remotePath)
	if rc, ok := c.remoteRCON(); ok {
		out, err := rc.Exec("changemode " + name)
		if err == nil && !reload.IsUnknownCommand(out) {
			c.log.Printf("reloaded over rcon (changemode). state was reset, normal")
			if out != "" {
				c.log.Printf("server: %s", out)
			}
			return true, nil
		}
		c.log.Printf("rcon changemode didn't work on the remote server (%v), trying the restart command", err)
	}

	if c.cfg.Remote.RestartCmd != "" {
		out, err := cli.Exec(c.cfg.Remote.RestartCmd)
		if err != nil {
			return true, fmt.Errorf("remote restart: %w", err)
		}
		c.log.Printf("reloaded via ssh: %s", c.cfg.Remote.RestartCmd)
		if out != "" {
			c.log.Printf("server: %s", out)
		}
		return true, nil
	}

	return true, fmt.Errorf("remote: no reload method left: configure rcon (rcon.password or remote.rcon_password) or remote.restart_cmd")
}

// compileAndReport compiles, prints diagnostics and reports the .amx path
// plus whether the build succeeded. Failed builds never reload.
func (c *Core) compileAndReport(pwn, amx string) (string, bool, error) {
	res, err := c.comp.Compile(pwn, amx)
	if err != nil {
		return "", false, fmt.Errorf("compile: %w", err)
	}
	for _, e := range res.Errors {
		if e.File != "" {
			c.log.Printf("%s(%d): %s %d: %s", e.File, e.Line, e.Level, e.Code, e.Message)
		} else {
			c.log.Printf("%s %d: %s", e.Level, e.Code, e.Message)
		}
	}
	if !res.Success {
		c.log.Printf("build failed: %d error(s). not reloading.", countErrors(res.Errors))
		return "", false, nil
	}
	if _, err := os.Stat(amx); err != nil {
		return "", false, fmt.Errorf("generated amx not found: %s", amx)
	}
	c.log.Printf("compiled ok -> %s", amx)
	return amx, true, nil
}

// remoteAMXPath resolves the remote destination for the compiled .amx.
func (c *Core) remoteAMXPath() string {
	if c.cfg.Remote.AMXPath != "" {
		return c.cfg.Remote.AMXPath
	}
	dir := c.info.GamemodesDir
	if dir == "" {
		dir = "gamemodes"
	}
	return path.Join(dir, c.cfg.Project.Gamemode+".amx")
}

// remoteRCON builds an RCON client targeting the remote server.
func (c *Core) remoteRCON() (*reload.RCON, bool) {
	host := c.cfg.Remote.RconHost
	if host == "" {
		host = c.cfg.Remote.Host
	}
	if host == "" {
		return nil, false
	}
	pw := c.cfg.Remote.RconPassword
	if pw == "" {
		pw = c.cfg.Rcon.Password
	}
	if pw == "" {
		pw = c.info.RconPassword
	}
	if pw == "" {
		return nil, false
	}
	port := c.cfg.Remote.RconPort
	if port == 0 {
		port = c.cfg.Rcon.Port
	}
	if port == 0 {
		port = c.info.Port
	}
	return &reload.RCON{Host: host, Port: port, Password: pw, Timeout: c.cfg.RconTimeout()}, true
}

func gamemodeFromAMX(p string) string {
	base := path.Base(p)
	return strings.TrimSuffix(base, path.Ext(base))
}

// gamemodePaths resolves the source .pwn and output .amx for the configured
// gamemode inside the runtime's gamemodes directory.
func (c *Core) gamemodePaths() (string, string, error) {
	dir := c.info.GamemodesDir
	if dir == "" {
		dir = "gamemodes"
	}
	dir = filepath.Join(c.cfg.Project.Root, dir)
	pwn := filepath.Join(dir, c.cfg.Project.Gamemode+".pwn")
	if _, err := os.Stat(pwn); err != nil {
		return "", "", fmt.Errorf("source gamemode not found: %s", pwn)
	}
	return pwn, filepath.Join(dir, c.cfg.Project.Gamemode+".amx"), nil
}

// watchDirs returns the directories to watch: the gamemodes dir plus any
// existing compiler include directories.
func (c *Core) watchDirs() []string {
	dirs := []string{}
	if d := filepath.Join(c.cfg.Project.Root, c.info.GamemodesDir); dirExists(d) {
		dirs = append(dirs, d)
	}
	for _, inc := range c.cfg.Project.Compiler.Includes {
		abs := c.cfg.AbsPath(inc)
		if dirExists(abs) {
			dirs = append(dirs, abs)
		}
	}
	if len(dirs) == 0 {
		dirs = append(dirs, c.cfg.Project.Root)
	}
	return dirs
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func countErrors(errs []compiler.Error) int {
	n := 0
	for _, e := range errs {
		if e.Level == "error" || e.Level == "fatal error" {
			n++
		}
	}
	return n
}
