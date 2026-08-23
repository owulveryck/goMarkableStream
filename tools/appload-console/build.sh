#!/bin/bash
# Build the GMS Console AppLoad app on Linux/WSL/macOS.
# Requires: go, and Qt6 rcc.
#   Debian/Ubuntu:  sudo apt install golang qt6-base-dev-tools
#   macOS:          brew install go qt   (rcc is auto-detected, see find_rcc below)
set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/build/gms-console"

# Locate the Qt6 resource compiler (rcc).
#   1. Honor an explicit $RCC if it is set and executable.
#   2. Otherwise use `rcc` from PATH.
#   3. Otherwise, on macOS, search the Homebrew installation (the Qt formulae
#      keep rcc under share/qt/libexec, which is not on PATH).
# If several candidates are found we use the first and remember the rest, so we
# can point the user at them only if the build actually fails.
RCC_CANDIDATES=""
find_rcc() {
    if [ -n "$RCC" ]; then
        if [ -x "$RCC" ]; then
            echo "Using rcc from \$RCC: $RCC" >&2
            printf '%s' "$RCC"
            return 0
        fi
        echo "warning: \$RCC is set to '$RCC' but it is not executable; ignoring it." >&2
    fi

    if command -v rcc >/dev/null 2>&1; then
        printf '%s' "$(command -v rcc)"
        return 0
    fi

    # macOS / Homebrew fallback.
    if command -v brew >/dev/null 2>&1; then
        local prefix
        prefix="$(brew --prefix 2>/dev/null)"
        if [ -n "$prefix" ]; then
            # Look under the Qt formulae kegs (qt, qtbase, qt@6, ...).
            RCC_CANDIDATES="$(find "$prefix/Cellar" "$prefix/opt" -type f -name rcc 2>/dev/null \
                | grep -E '/(qt[^/]*)/' | sort -u)"
        fi
    fi

    if [ -z "$RCC_CANDIDATES" ]; then
        return 1
    fi

    printf '%s' "$(printf '%s\n' "$RCC_CANDIDATES" | head -n1)"
    return 0
}

DEST_RCC="$(find_rcc || true)"
if [ -z "$DEST_RCC" ]; then
    echo "ERROR: could not find the Qt6 resource compiler 'rcc'." >&2
    echo "  - Debian/Ubuntu: sudo apt install qt6-base-dev-tools" >&2
    echo "  - macOS:         brew install qt" >&2
    echo "  - Or set RCC=/full/path/to/rcc and re-run." >&2
    exit 1
fi

rm -rf "$DEST"
mkdir -p "$DEST/backend"

echo "Building backend (linux/arm/v7 for reMarkable 2)..."
( cd "$ROOT/backend" && GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o "$DEST/backend/entry" . )

echo "Packing resources.rcc with: $DEST_RCC"
if ! "$DEST_RCC" --binary -o "$DEST/resources.rcc" "$ROOT/application.qrc"; then
    echo "ERROR: rcc failed while packing resources.rcc." >&2
    if [ -n "$RCC_CANDIDATES" ] && [ "$(printf '%s\n' "$RCC_CANDIDATES" | wc -l)" -gt 1 ]; then
        echo "Multiple rcc candidates were found; the first one was used. If it is the" >&2
        echo "wrong one, set RCC to the correct path and re-run, e.g.:" >&2
        printf '%s\n' "$RCC_CANDIDATES" | sed 's/^/    export RCC=/' >&2
    fi
    exit 1
fi

cp "$ROOT/manifest.json" "$DEST/"
[ -f "$ROOT/icon.png" ] && cp "$ROOT/icon.png" "$DEST/"

echo "Done -> $DEST"
echo "Deploy: scp -r \"$DEST\" root@10.11.99.1:/home/root/xovi/exthome/appload/"
