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
