#!/bin/bash
# Build the GMS Console AppLoad app on Linux/WSL/macOS.
# Requires: go, and Qt6 rcc (Debian/Ubuntu: sudo apt install golang qt6-base-dev-tools)
set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/build/gms-console"

rm -rf "$DEST"
mkdir -p "$DEST/backend"

echo "Building backend (linux/arm/v7 for reMarkable 2)..."
( cd "$ROOT/backend" && GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o "$DEST/backend/entry" . )

echo "Packing resources.rcc..."
rcc --binary -o "$DEST/resources.rcc" "$ROOT/application.qrc"

cp "$ROOT/manifest.json" "$DEST/"
[ -f "$ROOT/icon.png" ] && cp "$ROOT/icon.png" "$DEST/"

echo "Done -> $DEST"
echo "Deploy: scp -r \"$DEST\" root@10.11.99.1:/home/root/xovi/exthome/appload/"
