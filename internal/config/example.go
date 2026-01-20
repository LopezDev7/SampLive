package config

import "os"

// exampleYAML is the default configuration written by `samplive init`.
const exampleYAML = `# SampLive - hot reload for Pawn (SA-MP / open.mp) servers.
# Easiest way to get this file: run ` + "`samplive setup`" + `. It detects
# most of this by itself. This is the manual version, in case you like
# typing things by hand.
project:
  root: ./server          # where your server lives
  gamemode: mymode        # your gamemode name, no extension
  debounce: 300ms         # ignores quick bursts of writes so it doesn't compile mid-save
  watch: []               # reserved: extra files/globs to watch
  compiler:
    path: ""              # pawncc. leave empty and setup finds it for you
    includes: []          # include folders. setup finds these too
    flags: []

runtime:
  force: ""               # "samp" | "omp" | "" (auto-detect). Required when
                          # there's no local server (remote only).

rcon:
  enabled: true
  host: 127.0.0.1
  port: 0                 # 0 = use the server port it detected
  password: changeme
  timeout: 10s

server:
  command: ""             # how to start your server. used for the restart fallback
  args: []

remote:
  enabled: false          # deploy the .amx to a remote box over SFTP/SSH
  host: ""                # remote host, like your vps ip
  ssh_port: 22
  user: ""
  password: ""            # or keyfile:
  keyfile: ""
  amx_path: ""            # remote path for the .amx. default gamemodes/<gamemode>.amx
  rcon_host: ""           # default: the remote host
  rcon_port: 0            # default: the detected server port
  rcon_password: ""       # default: rcon.password
  restart_cmd: ""         # optional ssh command to restart the remote server
  insecure_skip_host_key: false  # check the ssh host key against ~/.ssh/known_hosts.
                          # set true to skip the check (not recommended)
`

// WriteExample writes a default config to path.
func WriteExample(path string) error {
	return os.WriteFile(path, []byte(exampleYAML), 0o644)
}
