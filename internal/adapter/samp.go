package adapter

import (
	"fmt"

	"samplive/internal/config"
	"samplive/internal/detect"
)

// SAMPRuntime handles SA-MP servers. SA-MP exposes RCON over TCP on the game
// port; "changemode <name>" switches gamemode without a full restart, though
// OnGameModeInit re-runs and gamemode state resets.
type SAMPRuntime struct{}

func (SAMPRuntime) Kind() detect.Kind { return detect.KindSAMP }

func (SAMPRuntime) Capabilities() Capabilities {
	return Capabilities{RconChangemode: true, RestartProcess: true}
}

// Reload applies the compiled gamemode: RCON changemode, then a process
// restart as fallback.
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
