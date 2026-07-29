# GMS Console — run goMarkableStream in a windowed AppLoad app with live output

This is a small AppLoad application that launches your existing `goMarkableStream`
binary, shows its **stdout + stderr live in a resizable window**, and keeps it
**running in the background** when you minimize/close the window.

## Why this instead of a plain `external.manifest.json`?

AppLoad launches external apps with `QProcess::ForwardedChannels`, so a plain
external app's stdout/stderr go to AppLoad's own log — **never into a window**.
The external window body only renders a QTFB framebuffer, which goMarkableStream
(a headless HTTP server) does not draw to. So to see output on-device you need a
tiny frontend that displays it. That is what this app is.

## Files
- `manifest.json`      — AppLoad native-app manifest (`loadsBackend`, `supportsScaling`)
- `application.qrc`    — lists the QML to pack into `resources.rcc`
- `ui/main.qml`        — the console window (scrolling log + Stop/Clear buttons)
- `backend/main.go`    — spawns goMarkableStream, forwards its output over the AppLoad socket
- `build.ps1`          — Windows build
- `build.sh`           — Linux/WSL build

## Before building
Edit `backend/main.go` and set `defaultBinary` to the absolute path of the
goMarkableStream binary you already installed on the tablet (or set `GMS_BINARY`
at runtime). Optionally set `defaultArgs` for CLI flags.

## Build
Backend cross-compiles with plain Go (no CGO). The QML must be packed into
`resources.rcc` with Qt6 `rcc` (output is not architecture-specific, so any
machine with Qt6 works — Windows Qt, or WSL `apt install qt6-base-dev-tools`).

Windows:
    winget install GoLang.Go        # if needed
    powershell -ExecutionPolicy Bypass -File .\build.ps1
    # if rcc isn't on Windows, produce the rcc via WSL:
    #   wsl bash -c "rcc --binary -o build/gms-console/resources.rcc application.qrc"

Linux/WSL:
    sudo apt install golang qt6-base-dev-tools
    ./build.sh

## Install
    scp -r build/gms-console root@10.11.99.1:/home/root/xovi/exthome/appload/
Then restart XOVI on the device (`/home/root/xovi/stop; /home/root/xovi/start`)
or reboot. A "GMS Console" icon appears in AppLoad.

## Use
- **Windowed (not fullscreen):** long-press the GMS Console icon in AppLoad.
- **See output:** the log streams live in the window.
- **Hide / background:** tap the `_` (minimize) button in the window title bar.
  The stream keeps running. Tap `_` again to restore. Because this is a native
  AppLoad app, the backend also survives the frontend closing and reconnects
  (replaying recent output) when you reopen it from AppLoad.
- **Stop the stream:** the "Stop stream" button (or press-and-hold the title-bar `X`).

## Optional: an icon
Drop a 1404-friendly `icon.png` in this folder before building for a custom icon.
