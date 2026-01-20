# Configuration

The config lives in `samplive.yaml`. The easiest way to get one is
`samplive setup`. `samplive init` writes a default one with comments.

## project

- `root` — the folder containing your server.
- `gamemode` — your gamemode name, without extension.
- `debounce` — how long to ignore rapid saves before compiling. Default
  `300ms`.
- `watch` — reserved for future use.

### project.compiler

- `path` — the `pawncc` binary. Leave empty and setup finds it.
- `includes` — include folders, passed as `-i`.
- `flags` — extra compiler flags.

## runtime

- `force` — `"samp"`, `"omp"` or empty to auto-detect. Required when you have
  no local server (remote only).

## rcon

- `enabled` — use RCON to reload.
- `host` — default `127.0.0.1`.
- `port` — `0` means "use the server port I detected".
- `password` — the server's `rcon_password`. Empty means "take it from the
  config file".
- `timeout` — connection timeout, e.g. `10s`.

## server

- `command` — how to start the server, used for the restart fallback.
- `args` — extra arguments.

## remote

- `enabled` — deploy the `.amx` to a remote box.
- `host` — the remote host.
- `ssh_port` — default `22`.
- `user` — SSH user.
- `password` — SSH password, or:
- `keyfile` — path to an SSH private key.
- `amx_path` — remote path for the `.amx`. Defaults to
  `gamemodes/<gamemode>.amx`.
- `rcon_host` — defaults to `host`.
- `rcon_port` — `0` means "use the detected server port".
- `rcon_password` — defaults to `rcon.password`.
- `restart_cmd` — SSH command to restart the server, used when RCON can't.
- `insecure_skip_host_key` — set `true` to skip the host key check. Not
  recommended.

See [remote.md](remote.md) for the remote flow.
