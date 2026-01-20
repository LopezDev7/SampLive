// Package adapter maps detected runtimes to reload strategies. Each runtime
// advertises its capabilities; the core pipeline uses the best method the
// runtime actually supports instead of assuming universal behavior.
package adapter

import (
	"path/filepath"
	"strings"

	"samplive/internal/config"
	"samplive/internal/detect"
	"samplive/internal/reload"
)

// Result describes a completed reload.
type Result struct {
	Method    string // e.g. "rcon:changemode", "process:restart"
	Output    string // server output, if any
	Preserved bool   // whether gamemode state was preserved. always false for now
}

// Capabilities advertises which reload methods a runtime supports.
type Capabilities struct {
	RconChangemode bool // server can switch gamemode over RCON
	InPlaceReload  bool // server can reload the code in place
	RestartProcess bool // tool can restart the local server process
}

// RuntimeAdapter handles one server runtime end to end.
type RuntimeAdapter interface {
	Kind() detect.Kind
	Capabilities() Capabilities
	Reload(info *detect.RuntimeInfo, cfg *config.Config, amx string) (*Result, error)
}

// gamemodeName returns the gamemode name (base name without extension) from
// a compiled .amx path.
func gamemodeName(amx string) string {
	base := filepath.Base(amx)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// rconFor builds an RCON client from the config and detected info, or
// returns ok=false when RCON is disabled or has no password.
func rconFor(cfg *config.Config, info *detect.RuntimeInfo) (*reload.RCON, bool) {
	if !cfg.Rcon.Enabled {
		return nil, false
	}
	pw := cfg.Rcon.Password
	if pw == "" {
		pw = info.RconPassword
	}
	if pw == "" {
		return nil, false
	}
	host := cfg.Rcon.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Rcon.Port
	if port == 0 {
		port = info.Port
	}
	return &reload.RCON{Host: host, Port: port, Password: pw, Timeout: cfg.RconTimeout()}, true
}

// processController builds the local restart controller, or ok=false when no
// start command is configured.
func processController(cfg *config.Config, info *detect.RuntimeInfo) (*reload.ProcessController, bool) {
	if cfg.Server.Command == "" {
		return nil, false
	}
	return &reload.ProcessController{
		Command: cfg.Server.Command,
		Args:    cfg.Server.Args,
		Port:    info.Port,
	}, true
}
