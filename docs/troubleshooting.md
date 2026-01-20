# Troubleshooting

## pawncc not found

SampLive looks for `pawncc` near your server folder, in `pawno/` and
`compiler/`, and on `PATH`. Run `samplive setup` and give it the path
directly, or put pawncc where SampLive can find it.

## Could not detect runtime

SampLive needs a `samp-server`/`samp03svr` binary or an `omp-server` binary,
plus `server.cfg` (SA-MP) or `config.json` (open.mp). If it's a remote-only
setup, set `runtime.force`.

## RCON: authentication failed

The RCON password in `samplive.yaml` doesn't match the server's
`rcon_password`. Check both.

## RCON: connection refused

The server isn't reachable on that port, or the port is wrong. Check
`rcon.port` and that the server is running.

## Host key not in known_hosts

See [remote.md](remote.md#host-keys). Run:

    ssh-keyscan your.server >> ~/.ssh/known_hosts

## No reload method available

For SA-MP you need RCON enabled with a password. For open.mp you need
`server.command` set, because open.mp reloads by restarting the process.

## Players get disconnected

That's the open.mp path: it restarts the server process. On SA-MP players
stay connected, but the gamemode restarts.

## My variables reset on reload

Yes. A reload runs `OnGameModeInit` again. Save what matters in
`OnGameModeExit` and load it back. Preserving state across reloads is the
hard problem that isn't solved yet.
