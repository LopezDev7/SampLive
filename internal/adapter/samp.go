package adapter

import (
	"fmt"

	"samplive/internal/config"
	"samplive/internal/detect"
)

// SAMPRuntime handles SA-MP servers.
//
// SA-MP has RCON over TCP on the game port, which is nice. "changemode
// <name>" switches to a different gamemode file without a full server
// restart; "gmx" restarts the current one. Both re-run OnGameModeInit, so
// gamemode state gets reset. That's the deal with SA-MP reloading.
type SAMPRuntime struct{}

// Kind reports the runtime kind.
func (SAMPRuntime) Kind() detect.Kind { return detect.KindSAMP }

// Capabilities reports what this adapter can do.
func (SAMPRuntime) Capabilities() Capabilities {
	return Capabilities{RconChangemode: true, RestartProcess: true}
}

// Reload applies the compiled gamemode using the best available method:
// RCON changemode, falling back to a local process restart.
func (SAMPRuntime) Reload(info *detect.RuntimeInfo, cfg *config.Config, amx string) (*Result, error) {
	name := gamemodeName(amx)

	if rc, ok := rconFor(cfg, info); ok {
		out, err := rc.Exec("changemode " + name)
		if err != nil {
			return nil, fmt.Errorf("samp: rcon changemode failed: %w", err)
		}
		return &Result{Method: "rcon:changemode", Output: out, Preserved: false}, nil
	}

	if pc, ok := processController(cfg, info); ok {
		out, err := pc.Restart()
		if err != nil {
			return nil, fmt.Errorf("samp: process restart failed: %w", err)
		}
		return &Result{Method: "process:restart", Output: out, Preserved: false}, nil
	}

	return nil, fmt.Errorf("samp: no reload method available: enable rcon (rcon.enabled + rcon.password) or set server.command")
}
