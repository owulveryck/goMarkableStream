# GMS Console — an on-device AppLoad app to control the goMarkableStream service

`gms-console` is a small [AppLoad](https://github.com/asivery/rm-appload) (XOVI)
application for the reMarkable that lets you **start, stop, restart and inspect**
the `goMarkableStream` systemd service directly from the tablet — no SSH, no
laptop. It shows a live service-state badge plus the exact terminal output of
each command (including `journalctl` logs) in a large, scrollable window.

It assumes goMarkableStream is already installed as a systemd service on the
device (i.e. you ran the binary's `-install` step, or created the unit manually
as described in the main README). This app does **not** install or bundle
goMarkableStream; it only talks to the existing `goMarkableStream.service` unit.

## What the buttons do

| Button          | Command it runs |
|-----------------|-----------------|
| **Start**       | `systemctl start goMarkableStream.service` |
| **Stop**        | `systemctl stop goMarkableStream.service` |
| **Restart**     | `systemctl restart goMarkableStream.service` |
| **Status & Logs** | `systemctl status goMarkableStream.service --no-pager` + `journalctl -u goMarkableStream.service -n 200 --no-pager` |
| **Clear**       | clears the on-screen output |

A **status badge** under the buttons summarises the service state at a glance
(*running* / *not running* / *errored out* / *starting…*), backed by
`systemctl is-active`. Start/Stop/Restart automatically refresh the badge and
re-run *Status & Logs* so you immediately see the result. The combined
stdout+stderr of every command is streamed into the window, exactly as you'd see
it from a shell.

**Opening or closing this app never touches the service.** Nothing is started or
stopped on launch or teardown — only the buttons act. A running stream keeps
running after you close the window, and the backend replays recent output when
you reopen it.

## Why an app instead of a plain `external.manifest.json`?

AppLoad launches external apps with `QProcess::ForwardedChannels`, so a plain
external app's stdout/stderr go to AppLoad's own log — **never into a window**.
The external window body only renders a QTFB framebuffer. So to see command
output on-device you need a tiny frontend that displays it. That is this app: a
small QML console plus a Go backend that runs the systemctl/journalctl commands
and forwards their output over the AppLoad socket.

## Files
- `manifest.json`   — AppLoad native-app manifest (`loadsBackend`, `supportsScaling`)
- `application.qrc` — lists the QML + bundled font packed into `resources.rcc`
- `ui/main.qml`     — the console window (buttons, status badge, scrolling output)
- `backend/main.go` — runs systemctl/journalctl, forwards output over the AppLoad socket
- `fonts/`          — bundled [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) (SIL OFL 1.1) used for the log view
- `build.ps1`       — Windows build
- `build.sh`        — Linux/WSL/macOS build (auto-detects Qt6 `rcc`)
- `BUILD-macos.md`  — step-by-step macOS (Homebrew) build guide

## Configuration
The unit name defaults to `goMarkableStream.service`. If you installed under a
different name, set the `GMS_SERVICE` environment variable for the app (e.g. via
AppLoad's environment) — no rebuild needed.

## Get a build

### Option A — download from CI (recommended, no toolchain needed)
Every push that touches this folder builds the app in GitHub Actions
(`.github/workflows/appload-console.yml`). Open the run and download the
`gms-console` artifact — a ready-to-install `gms-console/` folder zipped up. You
can also trigger it manually from the Actions tab ("Run workflow"). Tagged
releases attach the same `gms-console-RM2.zip` next to the main binaries.

### Option B — build locally
The backend cross-compiles with plain Go (no CGO). The QML and bundled font are
packed into `resources.rcc` with Qt6 `rcc` (output is not architecture-specific,
so any machine with Qt6 works).

Windows:

    winget install GoLang.Go        # if needed
    powershell -ExecutionPolicy Bypass -File .\build.ps1
    # if rcc isn't on Windows, produce the rcc via WSL:
    #   wsl bash -c "rcc --binary -o build/gms-console/resources.rcc application.qrc"

Linux/WSL:

    sudo apt install golang qt6-base-dev-tools
    ./build.sh

macOS (Homebrew): `brew install go qt`, then `./build.sh` — it auto-detects
`rcc` inside Homebrew. See **[BUILD-macos.md](./BUILD-macos.md)** for details.

All of these produce `build/gms-console/` containing `manifest.json`,
`resources.rcc` and `backend/entry`.

## Install
    scp -r build/gms-console root@10.11.99.1:/home/root/xovi/exthome/appload/
Then restart XOVI on the device (`/home/root/xovi/stop; /home/root/xovi/start`)
or reboot. A "GMS Console" icon appears in AppLoad.

## Use
- **Windowed (not fullscreen):** long-press the GMS Console icon in AppLoad.
- Tap **Start / Stop / Restart** to control the service, or **Status & Logs** to
  see its state and recent journal output. The badge under the buttons shows the
  current state.
- **Hide / background:** tap the `_` (minimize) button in the window title bar;
  the service is unaffected. Tap `_` again to restore.

## Optional: an icon
Drop a 1404-friendly `icon.png` in this folder before building for a custom icon.
