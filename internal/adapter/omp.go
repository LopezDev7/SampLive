package adapter

import (
	"fmt"
	"runtime"

	"samplive/internal/config"
	"samplive/internal/detect"
	"samplive/internal/reload"
)

// OMPRuntime handles open.mp servers. Its console accepts "changemode <name>",
// which reloads the gamemode in place while players stay connected. On
// Windows the server only reads its console through the input buffer (stdin
// pipes are ignored), so we type the command into it; the fallback, a process
// restart, disconnects everyone.
type OMPRuntime struct{}

func (OMPRuntime) Kind() detect.Kind { return detect.KindOMP }

func (OMPRuntime) Capabilities() Capabilities {
	return Capabilities{
		InPlaceReload:  runtime.GOOS == "windows",
		RconChangemode: true,
		RestartProcess: true,
	}
}

// Reload applies the compiled gamemode. Preferred: console:changemode (keeps
// players connected). RCON changemode when configured. Last resort: restart.
func (OMPRuntime) Reload(info *detect.RuntimeInfo, cfg *config.Config, amx string) (*Result, error) {
	name := gamemodeName(amx)

	if runtime.GOOS == "windows" {
		pid, _ := reload.FindPIDOnPort(info.Port)
		if pid > 0 && reload.InjectConsoleCommandTo(uint32(pid), "changemode "+name) == nil {
			return &Result{Method: "console:changemode", Preserved: true}, nil
		}
	}

	if rc, ok := rconFor(cfg, info); ok {
		out, err := rc.Exec("changemode " + name)
		if err == nil && !reload.IsUnknownCommand(out) {
			return &Result{Method: "rcon:changemode (unverified on omp)", Output: out, Preserved: true}, nil
		}
	}

	if pc, ok := processController(cfg, info); ok {
		pc.Console = true // give the restarted server a console so the next reload can stay in place
		out, err := pc.Restart()
		if err != nil {
			return nil, fmt.Errorf("omp: process restart failed: %w", err)
		}
		return &Result{Method: "process:restart", Output: out, Preserved: false}, nil
	}

	return nil, fmt.Errorf("omp: no reload method available: run the server from a console so SampLive can type \"changemode %s\" into it, or set server.command for process restarts", name)
}
