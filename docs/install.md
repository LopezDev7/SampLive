# Installing SampLive

## Releases

Grab the binary for your system from the
[releases](https://github.com/LopezDev7/SampLive/releases) page:

- `samplive-windows-amd64.exe`
- `samplive-windows-arm64.exe`
- `samplive-linux-amd64`
- `samplive-linux-arm64`

Put it somewhere on your `PATH` and you're done.

## From source

You need Go 1.26 or newer.

    go build -o samplive ./cmd/samplive

## Check it works

    samplive version
