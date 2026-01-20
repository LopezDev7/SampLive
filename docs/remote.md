# Remote servers

SampLive compiles on your machine, uploads the `.amx` over SFTP and reloads
the server over the network, so the remote server's OS doesn't matter.

## Config

The `remote` block in `samplive.yaml`. The minimum you need:

```yaml
remote:
  enabled: true
  host: 203.0.113.10
  user: root
  password: your-ssh-password   # or keyfile: /path/to/key
  rcon_password: changeme       # the server's rcon_password
```

## Reload order

1. Upload the `.amx` over SFTP.
2. Try RCON `changemode`. If the server doesn't know the command, fall back
   to:
3. `restart_cmd` over SSH.

If both fail, SampLive tells you what's missing.

## Host keys

Before connecting, SampLive checks the server's host key against
`~/.ssh/known_hosts`. If it's not there:

    ssh-keyscan your.server.ip >> ~/.ssh/known_hosts

Or set `remote.insecure_skip_host_key: true` to skip the check. You usually
don't want that.

## Pterodactyl

Pterodactyl gives every server SFTP credentials. Use those in the `remote`
block and the same flow works. The panel API part (restarting from the panel)
is planned.

## Remote-only projects

If you don't have a local server folder, set `runtime.force` to `samp` or
`omp` so SampLive knows which runtime to assume.
