package adapter

import (
	"fmt"

	"samplive/internal/config"
	"samplive/internal/detect"
	"samplive/internal/reload"
)

// OMPRuntime handles open.mp servers.
//
// Honest limitations:
//   - open.mp does not guarantee an in-place gamemode swap equivalent to
//     SA-MP's "changemode". We do not invent an API for it.
//   - If RCON is configured we opportunistically try "changemode"; if the
//     command is not recognized the output is detected and we fall through.
//   - The verified reload path is restarting the local server process.
type OMPRuntime struct{}

// Kind reports the runtime kind.
func (OMPRuntime) Kind() detect.Kind { return detect.KindOMP }

// Capabilities reports what this adapter can do.
func (OMPRuntime) Capabilities() Capabilities {
	return Capabilities{RestartProcess: true}
}

// Reload applies the compiled gamemode. Preferred method: local process
// restart. Opportunistic RCON changemode is attempted first when configured.
func (OMPRuntime) Reload(info *detect.RuntimeInfo, cfg *config.Config, amx string) (*Result, error) {
	name := gamemodeName(amx)

	if rc, ok := rconFor(cfg, info); ok {
		out, err := rc.Exec("changemode " + name)
		if err == nil && !reload.IsUnknownCommand(out) {
			return &Result{Method: "rcon:changemode (unverified on omp)", Output: out, Preserved: false}, nil
		}
	}

	if pc, ok := processController(cfg, info); ok {
		out, err := pc.Restart()
		if err != nil {
			return nil, fmt.Errorf("omp: process restart failed: %w", err)
		}
		return &Result{Method: "process:restart", Output: out, Preserved: false}, nil
	}

	return nil, fmt.Errorf("omp: no verified reload method: set server.command to enable process restart")
}
