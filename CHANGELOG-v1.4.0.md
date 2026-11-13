# goMarkableStream v1.4.0

## Performance & Bandwidth Improvements

### Zstd Delta Compression
- **New**: Zstd compression for delta frames (`FrameTypeDeltaZstd = 0x04`)
- Achieves ~25x compression ratio on stroke data (long runs of identical background pixels)
- Smart compression threshold: only compresses deltas >1KB (avoids overhead on tiny changes)
- Separate encoder pools for full frames vs deltas with optimized window sizes
- Delta encoder uses 256KB window (4x smaller memory footprint vs full-frame encoder)
- Adaptive frame selection: compares compressed delta size vs full frame cost before sending
- Concurrency pinned to 1 to avoid memory overhead and core contention on dual-core devices

### GPU-Accelerated Color Conversion
- **Removed CPU bottleneck**: BGRA→RGBA conversion now happens in GPU fragment shader
- Eliminates 10MB per-frame CPU swizzling (reMarkable 2 firmware 3.24+, Paper Pro)
- Zero-cost shader swizzle via `.bgra` component accessor in WebGL
- Maintains compatibility with grayscale-only devices

### Reader Pacing & Throttling
- **Intelligent framebuffer reading**: paces background reader at 2x consumer rate
- Reduces wasted memory bandwidth from continuous reading
- Adaptive pacing based on current frame rate (adjusts with delta size)
- Automatic backoff on read errors (100ms cooldown) prevents busy loops
- Triple-buffered async reader maintains low latency while staying efficient

## Robustness Improvements

### Error Handling
- **Fixed**: `findXochitlPID` no longer calls `log.Fatal` - now returns proper errors
- Gracefully handles vanished processes during /proc scanning (race condition fix)
- Skips unreadable process directories instead of aborting entire scan
- Added comprehensive unit tests with synthetic procfs fixtures

### Concurrency Fixes
- **Fixed**: Stream handler now uses per-connection encoder mutex
- Prevents concurrent access to delta encoder scratch buffers
- Second connection properly rejected with `429 Too Many Requests` instead of data corruption
- CORS preflight handled before lock acquisition (never blocked by active stream)
- `ReleaseMemory()` safely skips cleanup when encoder is in use

## Testing

### New Test Coverage
- `internal/delta/roundtrip_test.go`: Full compression/decompression roundtrip validation (277 lines)
- `internal/stream/asyncreader_test.go`: Reader pacing and error handling tests (171 lines)
- `internal/remarkable/findpid_test.go`: Procfs scanning edge cases (86 lines)
- `internal/stream/handler_test.go`: Concurrent connection and CORS tests (92 new lines)

## Protocol Changes

### Backward Compatible
- Added new frame type `0x04` (FrameTypeDeltaZstd) alongside existing types
- Clients automatically decompress zstd deltas when frame type indicates compression
- Falls back to uncompressed delta when payload is incompressible
- No changes required for existing clients (new frame type handled transparently)

## Dependencies

- Updated `github.com/go-json-experiment/json` to latest version
- No new external dependencies added (zstd library already present)

## Performance Comparison

| Metric | v1.3.1 | v1.4.0 | Improvement |
|--------|--------|--------|-------------|
| Delta bandwidth (typical stroke) | Raw runs | Zstd compressed | ~25x reduction |
| BGRA conversion overhead | 10MB CPU swizzle | GPU shader | Zero CPU cost |
| Wasted framebuffer reads | Continuous | 2x consumer rate | ~50% reduction |
| Idle frame detection | XOR-fold checksum | (unchanged) | - |
| Concurrent streams | Race condition | Mutex protected | Safe |

## Files Changed

- 15 files modified
- 1,211 insertions, 78 deletions
- New test coverage: 626 lines across 4 test files

## Migration Notes

### For Users
- **No action required** - changes are transparent to end users
- Binary size remains similar (zstd already included)
- Bandwidth usage reduced automatically for stroke-heavy workloads

### For Developers
- New frame type `0x04` handled in `worker_stream_processing.js`
- `findXochitlPID()` signature unchanged but now returns errors properly
- Stream handler concurrency model formalized with explicit mutex

---

**Installation:**
```bash
# Set your device: RM2 for reMarkable 2, RMPRO for Paper Pro
DEVICE=RM2

# Download latest version
VERSION=v1.4.0
wget -O goMarkableStream https://github.com/owulveryck/goMarkableStream/releases/download/$VERSION/gomarkablestream-$DEVICE
chmod +x goMarkableStream
./goMarkableStream
```

**Access:** https://10.11.99.1:2001 (default credentials: `admin` / `password`)
