# Building GMS Console on macOS

These are macOS-specific steps for building the `gms-console` AppLoad app. For
what the app does and how to install it on the tablet, see [README.md](./README.md).
If you'd rather not install any toolchain, just download the prebuilt
`gms-console` artifact from the GitHub Actions run (see the README) instead.

You need two things: **Go** (to cross-compile the backend for the reMarkable) and
**Qt6's `rcc`** (to pack the QML + bundled font into `resources.rcc`). Both come
from Homebrew.

## 1. Install Homebrew (skip if you already have it)

```sh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

## 2. Install Go and Qt6

```sh
brew install go qt
```

`brew install qt` installs Qt 6. Its command-line tools (`rcc`, `moc`, `uic`)
are **not** symlinked onto your `PATH` — they live inside the keg under
`share/qt/libexec` (e.g. `/opt/homebrew/Cellar/qtbase/6.11.1/share/qt/libexec/rcc`).

## 3. Build

```sh
cd tools/appload-console
./build.sh
```

`build.sh` finds `rcc` for you: it honors `$RCC` if set, then tries `rcc` on
your `PATH`, and finally searches your Homebrew installation (the Qt/qtbase
kegs). You normally don't have to do anything.

- **If `rcc` isn't found**, the script tells you and stops — run
  `brew install qt` (or `sudo apt install qt6-base-dev-tools` on Linux).
- **If several `rcc` copies are found**, the script uses the first and only asks
  you to choose if the build actually fails. In that case set `RCC` to the right
  path (the script prints the candidates) and re-run, e.g.:

  ```sh
  export RCC="$(brew --prefix)/Cellar/qtbase/6.11.1/share/qt/libexec/rcc"
  ./build.sh
  ```

The build produces `build/gms-console/` containing `manifest.json`,
`resources.rcc` and `backend/entry` (the backend is cross-compiled to
`linux/arm/v7` for the reMarkable 2 — no extra flags needed).

## 4. Install on the tablet

```sh
scp -r build/gms-console root@10.11.99.1:/home/root/xovi/exthome/appload/
```

Then restart XOVI on the device (`/home/root/xovi/stop; /home/root/xovi/start`)
or reboot. A "GMS Console" icon appears in AppLoad.

## Troubleshooting

- **`go: command not found`** — open a new terminal after `brew install go`, or
  add Homebrew to your shell: `eval "$(brew shellenv)"`.
- **Apple Silicon vs Intel** — no difference here: `build.sh` searches whichever
  Homebrew prefix you have, and the backend is cross-compiled for the tablet
  regardless of your Mac's architecture.
