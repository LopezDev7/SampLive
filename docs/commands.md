# Commands

    samplive                watch mode. The one you'll actually use.
    samplive once           compile and reload one time.
    samplive compile        compile only, to check for errors.
    samplive setup          wizard that writes samplive.yaml for you.
    samplive init           write a default samplive.yaml.
    samplive version        print the version.
    samplive help           this screen.

## Flags

All commands accept `-config <path>` to point at a config file other than
`samplive.yaml`.

The old flags still work for backwards compatibility: `-once`, `-watch`,
`-init`, `-version`, `-config`.

## Exit codes

- `0` — it worked.
- `1` — config problem, startup problem, compile problem, or reload problem.

`samplive once` and `samplive compile` exit with `1` when the build fails, so
you can wire them into editors or CI.
