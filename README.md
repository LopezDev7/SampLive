# SampLive

Hot reload for Pawn servers (SA-MP and open.mp).

You save your `.pwn`. It compiles. If it failed, you see the errors and the
server stays alone. If it worked, the server runs your new code — on open.mp
for Windows it doesn't even disconnect anyone.

## Quick start

    samplive setup
    samplive

## En español

SampLive recarga tu gamemode solo. Guardás el archivo `.pwn` y él compila y lo
aplica, sin que tengas que tocar el server.

Pasos, sin ciencia:

1. Bajá el programa desde
   [Releases](https://github.com/LopezDev7/SampLive/releases) (o compilalo
   con Go).
2. Poné el archivo `samplive` en cualquier carpeta.
3. Abrí una terminal en la carpeta de tu servidor (donde está tu gamemode).
4. Corré `samplive setup`. Te va a hacer preguntas: casi siempre es solo
   apretar Enter.
5. Corré `samplive` y listo.

Ahora cada vez que guardás el `.pwn`, SampLive lo compila y lo carga. Si la
compilación falla, te muestra los errores y el server no se toca.

Más ayuda: [Empezá acá](docs/quickstart.md).

## Documentation

- Start here: [docs/index.md](docs/index.md)
- Install: [docs/install.md](docs/install.md)
- Quick start: [docs/quickstart.md](docs/quickstart.md)
- Commands: [docs/commands.md](docs/commands.md)
- Configuration: [docs/configuration.md](docs/configuration.md)
- Remote servers: [docs/remote.md](docs/remote.md)
- Troubleshooting: [docs/troubleshooting.md](docs/troubleshooting.md)

## License

MIT. Take it and do whatever you want with it.
