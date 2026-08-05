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

---

## Servidores Linux de host (VPS / hosting)

SampLive funciona perfecto en hosts Linux, siempre que tengas acceso **SSH**.

El flujo en un VPS:

1. Compilás en tu máquina (Windows o lo que sea).
2. SampLive sube el `.amx` por **SFTP**.
3. Recarga por **RCON** (`changemode`) y, si el server no acepta el comando,
   por un comando SSH de restart (`restart_cmd`).

Config mínima en `samplive.yaml`:

```yaml
remote:
  enabled: true
  host: 203.0.113.10
  user: root
  password: tu-clave-ssh        # o keyfile: /ruta/a/llave
  rcon_password: cambiemela      # el rcon_password del server remoto
```

### Hosting sin SSH (paneles)

Si el host solo te da un panel (Pterodactyl y similares), todavía no hay
integración con el panel: no se puede reiniciar desde SampLive. Lo que sí
funciona es usar las credenciales SFTP del panel en el bloque `remote` para
subir el `.amx`. La integración del panel está planeada.

### open.mp en Linux

La recarga en el lugar (jugadores conectados) hoy existe solo en **Windows**,
porque ahí SampLive le escribe el comando en la consola. En Linux open.mp cae
al restart del proceso y todos se desconectan — es un TODO pendiente.
