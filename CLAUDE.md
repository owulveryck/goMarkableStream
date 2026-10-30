# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

goMarkableStream is a Go application that streams reMarkable tablet screens to web browsers in real-time. It runs directly on the device, reads the framebuffer from memory, and serves a WebGL-based client. Key design principles: no third-party dependencies (except optional Tailscale), delta compression for bandwidth efficiency, embedded frontend assets.

## Build Commands

```bash
# Build for reMarkable 1 (ARM, GOARM=7) - standard
make build-remarkable-1

# Build for reMarkable 1 with tracing support
make build-remarkable-1-trace

# Build for reMarkable 2 (ARM, GOARM=7) - standard
make build-remarkable-2

# Build for reMarkable 2 with tracing support
make build-remarkable-2-trace

# Build for reMarkable Paper Pro (ARM64) - standard
make build-remarkable-paper-pro

# Build for reMarkable Paper Pro with tracing support
make build-remarkable-paper-pro-trace

# Manual build with Tailscale and Tracing
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -v -trimpath -tags "tailscale,trace" -ldflags="-s -w" .

# Build without Tailscale or Tracing (smallest binary)
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" .
```

**Important:** The `trace` build tag is optional. By default, tracing is NOT compiled in to minimize binary size. Use `-tags trace` only when you need tracing/profiling capabilities.

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
- **RM1**: 1408x1872 (1404 visible width + 4px stride), 16-bit RGB565 via `mmap` `/dev/fb0`, single-core Cortex-A9 ARMv7 + NEON (`GOARM=7`), Compare-Before-Convert 1-byte Color8 Delta + Zstd by default
- **RM2**: 1404x1872 (32-bit BGRA on firmware 3.24+; 1872x1404 16-bit grayscale on legacy firmware), dual-core Cortex-A7 ARMv7 + NEON (`GOARM=7`)
- **Paper Pro**: 1632x2154 (visible 1620x2160), 32-bit BGRA, ARM64

### Configuration
All settings via environment variables with `RK_` prefix (uses `envconfig`). Key variables:
- `RK_SERVER_BIND_ADDR` (default `:2001`)
- `RK_SERVER_USERNAME`/`RK_SERVER_PASSWORD` (default `admin`/`password`)
- `RK_HTTPS` (default `true`)
- `RK_DELTA_THRESHOLD` (default `0.30`, range 0.0-1.0)
- `RK_RM1_RGB565` (default `false`, lossless 16-bit RGB565 Delta + Zstd on RM1)
- `RK_RLE_COMPRESSION` (default `false`, legacy full-frame RLE fallback on RM1)
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
