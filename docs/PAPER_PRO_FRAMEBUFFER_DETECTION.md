# Paper Pro Framebuffer Detection

## Problem

### Symptoms
- Application hangs at startup after "JWT: Loaded secret key" message
- 100% CPU usage on one core
- Never reaches "listening on" message

### Root Cause
The framebuffer detection algorithm in `internal/remarkable/pointer_arm64.go` relied on hardcoded screen dimensions to calculate `ScreenSizeBytes`, then scanned memory headers looking for a buffer with `length >= ScreenSizeBytes`.

A Paper Pro firmware update changed the GPU memory layout, causing the algorithm to fail:
- **Before firmware update:** Memory headers contained values >= 14,061,312 bytes → loop exited ✓
- **After firmware update:** No headers >= 14,061,312 bytes → infinite loop ✗

### Investigation

**Device memory layout** (from `/proc/[pid]/maps`):
```
ffff7cdfd000-ffff7cfaa000 rw-s 00000000 00:06 287  /dev/dri/card0  // size: 1,757,184
ffff7cfaa000-ffff7d157000 rw-s 00000000 00:06 287  /dev/dri/card0  // size: 1,757,184
... (15+ identical 1,757,184-byte tiles)
```

**Key finding:** GPU allocates framebuffer memory in fixed-size tiles of **1,757,184 bytes**, which is stable across firmware versions.

**Screen dimension history:**
- 1624×2154 pixels (69c9947, f7afec6) - worked with initial firmware
- 1632×2154 pixels (fc8395f) - worked before firmware update, broke after
- Official specs: 2160×1620 pixels - never tested with real hardware

## Solution

**Use GPU tile size instead of calculated screen size:**
- Added constant: `GPUTileSize = 1,757,184`
- Changed loop condition from `length < ScreenSizeBytes` to `length < GPUTileSize`
- Added safety limits (max iterations, header validation)

**Benefits:**
- ✓ Works across firmware updates (uses observable, stable value)
- ✓ Decouples screen dimensions from framebuffer detection
- ✓ Future-proof (firmware can change memory layout without breaking startup)
- ✓ Explicit error messages if memory layout is completely unexpected

## Technical Details

**GPU Tile Size Calculation:**
```
1,757,184 bytes ÷ 4 bytes/pixel = 439,296 pixels per tile
```

**Screen size in tiles:**
```
Official screen: 2,160 × 1,620 pixels = 3,499,200 pixels
3,499,200 ÷ 439,296 = ~7.96 tiles (rounds to 8 tiles)
```

This suggests the GPU allocates approximately 8 tiles for the framebuffer, but the exact mapping between screen pixels and GPU tiles is firmware-specific and shouldn't be hardcoded.

## Future Work

1. **Determine correct screen dimensions** - Currently 1632×2154, official specs say 2160×1620
2. **Investigate rendering accuracy** - Do dimensions affect pixel mapping or pen coordinates?
3. **Add firmware version detection** - Log firmware version at startup for debugging
4. **Integration tests** - Mock `/proc/[pid]/mem` with various memory layouts
