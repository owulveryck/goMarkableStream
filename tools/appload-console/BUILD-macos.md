# Building GMS Console on macOS

These are macOS-specific steps for building the `gms-console` AppLoad app. For
what the app does and how to install it on the tablet, see [README.md](./README.md).
If you'd rather not install any toolchain, just download the prebuilt
`gms-console` artifact from the GitHub Actions run (see the README) instead.

You need two things: **Go** (to cross-compile the backend for the reMarkable) and
**Qt6's `rcc`** (to pack the QML into `resources.rcc`). Both come from Homebrew.

## 1. Install Homebrew (skip if you already have it)

```sh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

## 2. Install Go and Qt6

```sh
brew install go qt
```

`brew install qt` installs Qt 6. Its command-line tools (`rcc`, `moc`, `uic`)
are **not** symlinked onto your `PATH` — they live under the keg's `libexec`.

## 3. Point the build at `rcc`

`build.sh` uses `$RCC` if set, otherwise it looks for `rcc` on your `PATH`.
Since Homebrew doesn't put `rcc` on the `PATH`, set it explicitly:

```sh
export RCC="$(brew --prefix qt)/libexec/rcc"
# Some Qt formula versions place it one level deeper; if the path above doesn't
# exist, find it with:
#   export RCC="$(find "$(brew --prefix qt)" -name rcc -type f | head -n1)"
"$RCC" --version   # sanity check
```

## 4. Build

```sh
cd tools/appload-console
./build.sh
```

This produces `build/gms-console/` containing `manifest.json`, `resources.rcc`
and `backend/entry` (the backend is cross-compiled to `linux/arm/v7` for the
reMarkable 2 — no extra flags needed, `build.sh` sets `GOOS/GOARCH/GOARM`).

## 5. Install on the tablet

```sh
scp -r build/gms-console root@10.11.99.1:/home/root/xovi/exthome/appload/
```

Then restart XOVI on the device (`/home/root/xovi/stop; /home/root/xovi/start`)
or reboot. A "GMS Console" icon appears in AppLoad.

## Troubleshooting

- **`rcc: command not found`** — `$RCC` isn't set or points at a missing file.
  Re-run step 3; confirm `"$RCC" --version` prints a Qt 6 version.
- **`go: command not found`** — open a new terminal after `brew install go`, or
  add Homebrew to your shell: `eval "$(brew shellenv)"`.
- **Apple Silicon vs Intel** — no difference here: `brew --prefix qt` resolves to
  the right location on both, and the backend is cross-compiled for the tablet
  regardless of your Mac's architecture.
