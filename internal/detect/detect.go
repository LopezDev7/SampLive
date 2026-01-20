// Package detect identifies which runtime (SA-MP, open.mp) a server
// directory belongs to, using file-based signals.
package detect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Kind identifies a supported server runtime.
type Kind string

const (
	KindSAMP    Kind = "samp"
	KindOMP     Kind = "omp"
	KindUnknown Kind = "unknown"
)

// RuntimeInfo describes a detected server.
type RuntimeInfo struct {
	Kind         Kind   `json:"kind"`
	Root         string `json:"root"`
	Binary       string `json:"binary,omitempty"`
	Port         int    `json:"port"`
	MaxPlayers   int    `json:"max_players"`
	RconEnabled  bool   `json:"rcon_enabled"`
	RconPassword string `json:"rcon_password,omitempty"`
	Gamemode     string `json:"gamemode"`
	GamemodesDir string `json:"gamemodes_dir"`
}

// Detect identifies the runtime in root. If force is set ("samp"/"omp")
// detection is constrained to that kind.
func Detect(root string, force string) (*RuntimeInfo, error) {
	switch Kind(strings.ToLower(strings.TrimSpace(force))) {
	case KindSAMP:
		info, ok := detectKind(root, KindSAMP)
		if !ok {
			return nil, fmt.Errorf("no SA-MP server detected in %q", root)
		}
		return info, nil
	case KindOMP:
		info, ok := detectKind(root, KindOMP)
		if !ok {
			return nil, fmt.Errorf("no open.mp server detected in %q", root)
		}
		return info, nil
	case "":
		if info, ok := detectKind(root, KindSAMP); ok {
			return info, nil
		}
		if info, ok := detectKind(root, KindOMP); ok {
			return info, nil
		}
		return nil, fmt.Errorf("could not detect runtime in %q: no samp-server/omp-server binary and no server.cfg/config.json found", root)
	default:
		return nil, fmt.Errorf("unknown runtime kind %q (expected \"samp\" or \"omp\")", force)
	}
}

func detectKind(root string, kind Kind) (*RuntimeInfo, bool) {
	info := &RuntimeInfo{Kind: kind, Root: root, GamemodesDir: "gamemodes", Port: 7777}
	switch kind {
	case KindSAMP:
		if b := findExisting(root, "samp-server.exe", "samp03svr"); b != "" {
			info.Binary = b
		}
		cfg := filepath.Join(root, "server.cfg")
		if _, err := os.Stat(cfg); err != nil {
			return info, info.Binary != ""
		}
		applyServerCfg(info, parseServerCfg(cfg))
		return info, true
	case KindOMP:
		if b := findExisting(root, "omp-server", "omp-server.exe"); b != "" {
			info.Binary = b
		}
		cfg := filepath.Join(root, "config.json")
		if _, err := os.Stat(cfg); err != nil {
			return info, info.Binary != ""
		}
		applyOMPConfig(info, cfg)
		return info, true
	}
	return nil, false
}

func findExisting(root string, names ...string) string {
	for _, n := range names {
		p := filepath.Join(root, n)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// parseServerCfg parses a SA-MP server.cfg (also accepted by open.mp in
// legacy mode). Lines are "key value" or "key=value"; comments start with #.
func parseServerCfg(path string) map[string]string {
	kv := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return kv
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		toks := strings.Fields(line)
		if len(toks) == 0 {
			continue
		}
		key := strings.ToLower(toks[0])
		if len(toks) > 1 {
			kv[key] = strings.Trim(toks[1], "\"")
		}
	}
	return kv
}

func applyServerCfg(info *RuntimeInfo, kv map[string]string) {
	for i := 0; i < 16; i++ {
		if v, ok := kv[fmt.Sprintf("gamemode%d", i)]; ok {
			info.Gamemode = v
			break
		}
	}
	if v, ok := kv["rcon_password"]; ok {
		info.RconEnabled = true
		info.RconPassword = v
	}
	if v, ok := kv["port"]; ok {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			info.Port = p
		}
	}
	if v, ok := kv["maxplayers"]; ok {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			info.MaxPlayers = p
		}
	}
}

// applyOMPConfig reads an open.mp config.json leniently. The schema has
// changed across versions, so we search for the relevant keys instead of
// assuming a fixed structure.
func applyOMPConfig(info *RuntimeInfo, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	if v := firstScalar(root, "mode", "gamemode"); v != "" {
		info.Gamemode = v
	}
	for _, section := range []string{"server", "game"} {
		if m, ok := root[section].(map[string]any); ok {
			if info.Gamemode == "" {
				info.Gamemode = firstScalar(m, "mode", "gamemode")
			}
			if p, ok := scalarInt(m, "port"); ok && p > 0 {
				info.Port = p
			}
		}
	}
	if m, ok := root["rcon"].(map[string]any); ok {
		if v := firstScalar(m, "password"); v != "" {
			info.RconEnabled = true
			info.RconPassword = v
		}
	}
}

func firstScalar(m map[string]any, keys ...string) string {
	for _, key := range keys {
		for k, v := range m {
			if !strings.EqualFold(k, key) {
				continue
			}
			switch t := v.(type) {
			case string:
				return t
			case float64:
				return strconv.Itoa(int(t))
			}
		}
	}
	return ""
}

func scalarInt(m map[string]any, key string) (int, bool) {
	for k, v := range m {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch t := v.(type) {
		case float64:
			return int(t), true
		case string:
			if p, err := strconv.Atoi(t); err == nil {
				return p, true
			}
		}
	}
	return 0, false
}
