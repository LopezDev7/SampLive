# SampLive

SampLive is a command-line tool that reloads your Pawn gamemode when you save
your code. It works for SA-MP and open.mp.

## How it works

Three steps, in a loop:

1. It watches your `.pwn` and `.inc` files.
2. When you save, it runs `pawncc` to compile.
3. If the build is clean, it reloads the gamemode the way your server
   supports it. If the build fails, it prints the errors and leaves the
   server alone.

It lives outside the server. It can't be an include or a filterscript,
because an include runs inside the server and has no access to the disk or
the compiler. The watching and compiling have to happen on your machine.

## What actually happens on a reload

The honest part:

- **SA-MP**: reloads over RCON with `changemode`. Players stay connected, but
  the gamemode restarts: variables, timers and player data reset, and
  `OnGameModeInit` runs again. Data already saved to your database is safe;
  data only in memory is gone.
- **open.mp**: restarts the server process. Everyone gets disconnected.
- **Remote hosts**: the `.amx` uploads over SFTP and the reload runs over
  RCON or an SSH command. Depends on the server.

Keeping the state (variables, timers, players) across a reload is the hard
part and it's not solved yet. A reload restarts the gamemode. That's the MVP.

## What it isn't

- It's not a way to run `.amx` code outside a server. The compile target is
  still the server.
- It's not a package manager. For that, look at
  [sampctl](https://github.com/Southclaws/sampctl).
- It's not a drop-in replacement for a proper CI pipeline. It's a comfort
  tool for development.
