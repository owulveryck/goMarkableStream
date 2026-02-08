# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

goMarkableStream is a Go application that streams reMarkable tablet screens to web browsers in real-time. It runs directly on the device, reads the framebuffer from memory, and serves a WebGL-based client. Key design principles: no third-party dependencies (except optional Tailscale), delta compression for bandwidth efficiency, embedded frontend assets.

## Build Commands

```bash
# Build for reMarkable 2 (ARM, GOARM=7)
make build-remarkable-2

# Build for reMarkable Paper Pro (ARM64)
make build-remarkable-paper-pro

# Manual build with Tailscale
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -v -trimpath -tags tailscale -ldflags="-s -w" .

# Build without Tailscale (smaller binary)
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" .
```

## Testing

```bash
go test ./...
```

## Architecture

### Server-Client Model
- **Server (Go)**: Runs on reMarkable, reads framebuffer via direct memory access, serves HTTP endpoints
- **Client (JavaScript/WebGL)**: Browser-based rendering with Web Workers for stream/event processing

### Key Packages

| Package | Purpose |
|---------|---------|
| `internal/remarkable/` | Device abstraction: framebuffer access, device detection, input events. Architecture-specific files: `*_arm64.go` for Paper Pro, default for RM2 |
| `internal/stream/` | HTTP stream handler with delta encoding and rate control |
| `internal/delta/` | Delta compression: compares frames, sends only changed pixel runs |
| `internal/pubsub/` | Thread-safe event bus for broadcasting pen/touch events |
| `internal/eventhttphandler/` | SSE endpoints for pen events and gesture handling |
| `client/` | Embedded frontend assets (WebGL canvas, Web Workers) |

### Build Tags
- `tailscale`: Enables Tailscale integration (remote access, Funnel)
- Platform detection via `arm`/`arm64`/`!linux` tags

### Device Differences
- **RM2**: 1872x1404, 16-bit grayscale, ARM (GOARM=7)
- **Paper Pro**: 2160x1620, 32-bit BGRA, ARM64

### Configuration
All settings via environment variables with `RK_` prefix (uses `envconfig`). Key variables:
- `RK_SERVER_BIND_ADDR` (default `:2001`)
- `RK_SERVER_USERNAME`/`RK_SERVER_PASSWORD` (default `admin`/`password`)
- `RK_HTTPS` (default `true`)
- `RK_DELTA_THRESHOLD` (default `0.30`, range 0.0-1.0)
- `RK_TAILSCALE_*` for Tailscale options

### HTTP Endpoints
- `/` - Main UI
- `/stream` - Binary frame stream (delta-encoded)
- `/events` - Pen events via SSE
- `/gestures` - Touch gesture handling
- `/version` - Version info

## Development Notes

- Framebuffer reading is platform-specific: test on actual hardware when possible
- Delta compression is critical for bandwidth - benchmark changes in `internal/delta/`
- Client files in `client/` are embedded into the binary
- The `tailscale` build tag is conditional - ensure code handles both presence and absence
