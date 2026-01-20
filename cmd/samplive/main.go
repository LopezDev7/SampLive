// SampLive - hot reload for Pawn (SA-MP / open.mp) servers.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"samplive/internal/config"
	"samplive/internal/core"
	"samplive/internal/setup"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "watch", "once", "compile", "setup", "init", "version", "help":
			runSubcommand(args[0], args[1:])
			return
		}
	}
	runLegacy(args)
}

func runSubcommand(cmd string, args []string) {
	switch cmd {
	case "watch":
		runWatch(args)
	case "once":
		runOnce(args)
	case "compile":
		runCompile(args)
	case "setup":
		runSetup(args)
	case "init":
		runInit(args)
	case "version":
		printVersion()
	case "help":
		printHelp()
	}
}

// runWatch watches for changes and reloads on each save.
func runWatch(args []string) {
	cfgPath := flagSet(args, "config", "samplive.yaml", "path to config file")
	_, c := loadCore(cfgPath)
	if err := c.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}

// runOnce compiles and reloads a single time, then exits.
func runOnce(args []string) {
	cfgPath := flagSet(args, "config", "samplive.yaml", "path to config file")
	_, c := loadCore(cfgPath)
	if err := c.RunOnce(); err != nil {
		log.Fatalf("run: %v", err)
	}
}

// runCompile compiles once and reports the result without reloading.
func runCompile(args []string) {
	cfgPath := flagSet(args, "config", "samplive.yaml", "path to config file")
	_, c := loadCore(cfgPath)
	if err := c.CompileOnce(); err != nil {
		log.Fatalf("compile: %v", err)
	}
}

// runSetup starts the interactive configuration wizard.
func runSetup(args []string) {
	cfgPath := flagSet(args, "config", "samplive.yaml", "path where the config will be written")
	if err := setup.Run(cfgPath); err != nil {
		log.Fatalf("setup: %v", err)
	}
}

// runInit writes a default config file.
func runInit(args []string) {
	cfgPath := flagSet(args, "config", "samplive.yaml", "path where the config will be written")
	if err := config.WriteExample(cfgPath); err != nil {
		log.Fatalf("init: %v", err)
	}
	fmt.Printf("default config written to %s\n", cfgPath)
}

// runLegacy supports the pre-subcommand flags: -config, -once, -watch, -init, -version.
func runLegacy(args []string) {
	fs := flag.NewFlagSet("samplive", flag.ExitOnError)
	cfgPath := fs.String("config", "samplive.yaml", "path to config file")
	once := fs.Bool("once", false, "compile and reload once, then exit")
	watch := fs.Bool("watch", true, "watch for changes (disable with -watch=false)")
	initOnly := fs.Bool("init", false, "write a default config and exit")
	showVersion := fs.Bool("version", false, "print version and exit")
	_ = fs.Parse(args)

	if *showVersion {
		printVersion()
		return
	}
	if *initOnly {
		if err := config.WriteExample(*cfgPath); err != nil {
			log.Fatalf("init: %v", err)
		}
		fmt.Printf("default config written to %s\n", *cfgPath)
		return
	}
	_, c := loadCore(*cfgPath)
	if *once {
		if err := c.RunOnce(); err != nil {
			log.Fatalf("run: %v", err)
		}
		return
	}
	if *watch {
		if err := c.Run(); err != nil {
			log.Fatalf("run: %v", err)
		}
	}
}

// flagSet parses a single -config flag for a subcommand.
func flagSet(args []string, name, def, usage string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String(name, def, usage)
	_ = fs.Parse(args)
	return *path
}

func loadCore(cfgPath string) (*config.Config, *core.Core) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	c, err := core.New(cfg, log.Default())
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	log.Printf("runtime: %s (port %d, gamemode %q)", c.Info().Kind, c.Info().Port, c.Info().Gamemode)
	return cfg, c
}

func printVersion() {
	fmt.Printf("samplive %s\n", version)
	fmt.Println("hot reload for Pawn (SA-MP / open.mp) servers")
}

func printHelp() {
	fmt.Println(`SampLive - hot reload for Pawn (SA-MP / open.mp) servers

Usage:
  samplive [command]

Commands:
  watch     watch for changes and reload on every save (default)
  once      compile and reload once, then exit
  compile   compile once and show errors, without reloading
  setup     wizard that writes a samplive.yaml for you. it tries to guess
            everything, you just press Enter
  init      write a default samplive.yaml
  version   print the version
  help      show this help

Flags (all commands):
  -config <path>   config file (default "samplive.yaml")

The flow: you save a .pwn or .inc, it compiles, errors show up on the
terminal (and on a failed build the server stays alone), and if it compiled
it reloads the way the runtime supports it: RCON changemode for SA-MP,
process restart for open.mp, and SFTP + RCON/SSH for remote servers.`)
}
