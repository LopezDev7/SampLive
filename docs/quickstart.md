# Quick start

SampLive works from the folder where your server lives.

## 1. Setup

    samplive setup

It tries to guess everything: your server, the runtime (SA-MP or open.mp),
your gamemode, your `pawncc` and the RCON password. Most of the time you just
press Enter.

If it can't find something, it asks. The result is a `samplive.yaml` in the
current folder.

## 2. Run

    samplive

That's it. Now edit a `.pwn`, save it, and watch it reload.

## The loop

1. Save a `.pwn` or `.inc`.
2. SampLive compiles.
3. Errors show up on the terminal. Failed build? The server stays alone.
4. Clean build? The server reloads.

## What you'll see on open.mp (Windows)

On Windows with open.mp the reload is **in place**: SampLive types
`changemode` into the server's console, so your players stay connected and the
process keeps running. The terminal prints:

    reloaded via console:changemode (players stay connected)

No setup needed, it just works. The server runs in the background with a
hidden console that SampLive drives for you; you never have to look at it.

On Linux/macOS (or if console injection isn't available) it falls back to
restarting the server process, which disconnects everyone — that's normal.
