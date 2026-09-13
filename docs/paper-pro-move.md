# reMarkable Paper Pro Move

The Move build is isolated behind the `rmmove` build tag. Existing reMarkable 2
and full-size Paper Pro builds keep their existing constants and readers.

Build the lightweight ARM64 binary without Tailscale:

```sh
make build-remarkable-paper-pro-move
```

The output is `goMarkableStream-RMMOVE`. Copy it to the tablet and launch it
manually. It is not installed as a service:

```sh
scp goMarkableStream-RMMOVE remarkable:/home/root/
ssh remarkable /home/root/goMarkableStream-RMMOVE
```

By default the server uses HTTPS and JWT authentication on port 2001. For an
isolated unauthenticated test:

```sh
RK_HTTPS=false RK_JWT_ENABLED=false /home/root/goMarkableStream-RMMOVE -unsafe
```

## Chiappa framebuffer layout

This layout was measured on a Paper Pro Move (`reMarkable Chiappa`) running
firmware 3.28.0.169:

- The framebuffer begins at the address found by the existing ARM64 DRM header
  lookup in the readable anonymous mapping immediately after the final
  `/dev/dri/card0` mapping.
- Storage is portrait BGRA, 960 x 1696 pixels, four bytes per pixel.
- The visible image is 954 x 1696 pixels.
- Each storage row is 3,840 bytes. Its first 3,816 bytes are visible pixels;
  the final 24 bytes are six right-edge alignment pixels and are discarded.
- No rotation, transpose, or channel conversion is required after row packing.
- The tightly packed response remains 6,471,936 bytes, preserving the existing
  `/stream`, `/raw`, and `/screenshot` response formats.

The Move pen reports X in 0-6760 and Y in 0-11960. It uses the Paper Pro's
portrait-native event transform.
