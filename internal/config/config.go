// Package config loads and validates SampLive configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level SampLive configuration.
type Config struct {
	Project Project `yaml:"project"`
	Runtime Runtime `yaml:"runtime"`
	Rcon    Rcon    `yaml:"rcon"`
	Server  Server  `yaml:"server"`
	Remote  Remote  `yaml:"remote"`
}

// Project describes the Pawn project to watch, compile and reload.
type Project struct {
	Root     string   `yaml:"root"`     // directory that contains the server
	Gamemode string   `yaml:"gamemode"` // gamemode name without extension
	Debounce string   `yaml:"debounce"` // debounce window, e.g. "300ms"
	Compiler Compiler `yaml:"compiler"`
}

// Compiler points to the pawncc binary and its include directories.
type Compiler struct {
	Path     string   `yaml:"path"`     // pawncc binary (per-runtime compiler)
	Includes []string `yaml:"includes"` // include directories, -i flags
	Flags    []string `yaml:"flags"`    // extra compiler flags
}

// Runtime controls runtime detection.
type Runtime struct {
	Force string `yaml:"force"` // "samp", "omp" or "" for auto-detection
}

// Rcon configures the SA-MP RCON connection used for hot reload.
type Rcon struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`     // default 127.0.0.1
	Port     int    `yaml:"port"`     // 0 = use the detected server port
	Password string `yaml:"password"` // overrides the detected rcon_password
	Timeout  string `yaml:"timeout"`  // e.g. "10s"
}

// Server configures the local server process (fallback reload via restart).
type Server struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// Remote deploys the compiled gamemode to a remote server over SFTP/SSH.
// The reload itself runs over the network via RCON (or a configured SSH
// restart command), so it works regardless of the remote OS.
type Remote struct {
	Enabled             bool   `yaml:"enabled"`
	Host                string `yaml:"host"`
	SSHPort             int    `yaml:"ssh_port"` // default 22
	User                string `yaml:"user"`
	Password            string `yaml:"password"`
	Keyfile             string `yaml:"keyfile"`                // path to an SSH private key
	AMXPath             string `yaml:"amx_path"`               // remote path for the .amx (default: gamemodes/<gamemode>.amx)
	RconHost            string `yaml:"rcon_host"`              // defaults to host
	RconPort            int    `yaml:"rcon_port"`              // 0 = use the detected server port
	RconPassword        string `yaml:"rcon_password"`          // defaults to rcon.password
	RestartCmd          string `yaml:"restart_cmd"`            // optional SSH command to restart the server
	InsecureSkipHostKey bool   `yaml:"insecure_skip_host_key"` // skip the host key check against ~/.ssh/known_hosts. not recommended
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// Validate checks required fields and fills in defaults.
func (c *Config) Validate() error {
	if c.Project.Root == "" {
		return fmt.Errorf("project.root is required")
	}
	info, err := os.Stat(c.Project.Root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project.root %q is not a directory", c.Project.Root)
	}
	if c.Project.Gamemode == "" {
		return fmt.Errorf("project.gamemode is required")
	}
	if c.Project.Debounce == "" {
		c.Project.Debounce = "300ms"
	}
	if _, err := time.ParseDuration(c.Project.Debounce); err != nil {
		return fmt.Errorf("project.debounce: %v", err)
	}
	switch c.Runtime.Force {
	case "", "samp", "omp":
	default:
		return fmt.Errorf("runtime.force must be \"samp\", \"omp\" or empty, got %q", c.Runtime.Force)
	}
	if c.Remote.Enabled {
		if c.Remote.Host == "" {
			return fmt.Errorf("remote.enabled requires remote.host")
		}
		if c.Remote.User == "" {
			return fmt.Errorf("remote.enabled requires remote.user")
		}
		if c.Remote.Password == "" && c.Remote.Keyfile == "" {
			return fmt.Errorf("remote.enabled requires remote.password or remote.keyfile")
		}
	}
	return nil
}

// DebounceDuration returns the parsed debounce window.
func (c *Config) DebounceDuration() time.Duration {
	d, err := time.ParseDuration(c.Project.Debounce)
	if err != nil || d <= 0 {
		return 300 * time.Millisecond
	}
	return d
}

// RconTimeout returns the parsed RCON timeout.
func (c *Config) RconTimeout() time.Duration {
	d, err := time.ParseDuration(c.Rcon.Timeout)
	if err != nil || d <= 0 {
		return 10 * time.Second
	}
	return d
}

// AbsPath resolves a path relative to the project root when it is not absolute.
func (c *Config) AbsPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.Project.Root, p)
}
