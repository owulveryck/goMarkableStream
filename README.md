[![Go](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml/badge.svg)](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml)
![Static Badge](https://img.shields.io/badge/reMarkable_2-Firmware_3.24+-green)

# goMarkableStream

[Screen recording 2026-02-04 11.24.44.webm](https://github.com/user-attachments/assets/facdf1c5-41b2-4b82-bb84-2dd169858a80)



## Overview

goMarkableStream is a lightweight and user-friendly application designed specifically for the reMarkable tablet.

Its primary goal is to enable users to stream their reMarkable tablet screen to a web browser without the need for any hacks or modifications that could void the warranty.

## Device Support

**Actively supported and tested:**
- reMarkable 2 with firmware 3.24+

**Experimental (not actively tested):**
- reMarkable Paper Pro - initial support, some features may not work as expected

## Version Support

The latest version of goMarkableStream is actively developed and tested on reMarkable 2 with firmware 3.24+.

For older firmware versions:
- Firmware < 3.4: use goMarkableStream version < 0.8.6
- Firmware >= 3.4 and < 3.6: use version >= 0.8.6 and < 0.11.0
- Firmware >= 3.6 and < 3.24: use version >= 0.11.0 (may work, but not actively tested)

## Features

- **No Warranty Voiding**: Operates within the reMarkable tablet's intended functionality without unauthorized modifications.
- **No Subscription Required**: Completely free to use, with no subscription fees or recurring charges.
- **No Client-Side Installation**: Access the screen streaming feature directly through a web browser, with no need for additional software or plugins.
- **Color Support**: Enhanced streaming experience with color support.
- **High Performance**: Utilizes WebGL for smooth and efficient performance.
- **Laser Pointer**: Features a laser pointer that activates on hovering.
- **Gesture Support**: Full integration with Reveal.js, allowing slide switching directly from the reMarkable.
- **Overlay Feature**: Allows overlaying over existing websites that support iframe embedding.
- **Live Parameter Tweaking**: Side menu for live adjustments, including screen rotation.
- **Dark Mode**: Toggle between light and dark themes for comfortable viewing in any environment.
- **Version API**: Check the current version via the `/version` endpoint.

## Available Binaries

Each release provides four binary variants:

| Binary | Device | Tailscale Support |
|--------|--------|-------------------|
| `gomarkablestream-RMPRO` | reMarkable Paper Pro (arm64) | Yes |
| `gomarkablestream-RM2` | reMarkable 2 (arm) | Yes |
| `gomarkablestream-RMPRO-lite` | reMarkable Paper Pro (arm64) | No |
| `gomarkablestream-RM2-lite` | reMarkable 2 (arm) | No |

**Which binary should I use?**
- Use `RMPRO` variants for reMarkable Paper Pro
- Use `RM2` variants for reMarkable 2
- Use `-lite` variants if you don't need Tailscale remote access (smaller binary size)

## Quick Start

```bash
localhost> ssh root@remarkable
```

**For reMarkable 2:**

```bash
export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
wget -O goMarkableStream https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/gomarkablestream-RM2
chmod +x goMarkableStream
./goMarkableStream
```

**For reMarkable Paper Pro:**

```bash
export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
wget -O goMarkableStream https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/gomarkablestream-RMPRO
chmod +x goMarkableStream
./goMarkableStream
```

**For lite versions (without Tailscale):** Replace `gomarkablestream-RM2` or `gomarkablestream-RMPRO` with `gomarkablestream-RM2-lite` or `gomarkablestream-RMPRO-lite` respectively.

Then go to [https://remarkable.local.:2001](https://remarkable.local.:2001) and log in with `admin`/`password` (can be changed through environment variables or disable authentication with `-unsafe`)

_Note_: _remarkable.local._ may work from Apple devices (mDNS resolution).
Please note the `.` at the end.
If it does not work, you may need to replace `remarkable.local.` with the IP address of the tablet.

_Note 2_: You can use this to update to a new version (ensure that you killed the previous version before with `kill $(pidof goMarkableStream)`)

### Errors due to missing packages

If you get errors such as `wget: note: TLS certificate validation not implemented` when running through the process above, you can first download goMarkableStream to your local computer and then copy it over to your reMarkable as follows:

1. Run this on your local computer to download goMarkableStream into the current directory (use `gomarkablestream-RM2` for reMarkable 2 or `gomarkablestream-RMPRO` for reMarkable Paper Pro):

```bash
localhost> export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
wget -O goMarkableStream https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/gomarkablestream-RM2
chmod +x goMarkableStream
```

2. Copy it over to your reMarkable (`remarkable` is the IP of your reMarkable):
```bash
localhost> scp ./goMarkableStream root@remarkable:/home/root/goMarkableStream
```

3. Log in to your reMarkable:
```bash
localhost> ssh root@remarkable
```

4. Start goMarkableStream (to make it permanent, see section about turning goMarkableStream into a systemd service):
```bash
./goMarkableStream
```



## Setup goMarkableStream as a systemd service

This section explains how to set up goMarkableStream as a system service that will stay running through
device restart and sleep. Note, however, this setup script will need to be executed 
for any reMarkable system update/installation. 

First, we'll write the script, saving it to the home directory. Then, we'll execute the script which performs all
setup necessary to register goMarkableStream as a system service. It can be executed after every system update.
Note, this script assumes the goMarkableStream executable exists in the home directory.

```bash
localhost> ssh root@remarkable
```

Create a bash script under the home directory:
```bash
touch setupGoMarkableStream.sh
chmod +x setupGoMarkableStream.sh
```

Then open the file in nano:
```bash
nano setupGoMarkableStream.sh
```

Finally, paste (ctrl-shift-v) the following into the nano editor. Then save and quit (ctrl-X, Y, [enter]).
```bash
pushd /etc/systemd/system
touch goMarkableStream.service

cat <<EOF>goMarkableStream.service
[Unit]
Description=Go Remarkable Stream Server

[Service]
ExecStart=/home/root/goMarkableStream
Restart=always

[Install]
WantedBy=multi-user.target
EOF

systemctl enable goMarkableStream.service
systemctl start goMarkableStream.service
systemctl status goMarkableStream.service
popd
```

Executing setupGoMarkableStream.sh will register the goMarkableStream executable as a systemd service!

## Configurations

### Device Configuration
Configure the application via environment variables:
- `RK_SERVER_BIND_ADDR`: (String, default: `:2001`) Server bind address.
- `RK_SERVER_USERNAME`: (String, default: `admin`) Username for server access.
- `RK_SERVER_PASSWORD`: (String, default: `password`) Password for server access.
- `RK_HTTPS`: (True/False, default: `true`) Enable or disable HTTPS.
- `RK_DEV_MODE`: (True/False, default: `false`) Enable or disable developer mode.
- `RK_DELTA_THRESHOLD`: (Float, default: `0.30`) Change ratio threshold (0.0-1.0) above which a full frame is sent instead of delta.

### Tailscale Configuration

Tailscale allows secure remote access to your reMarkable tablet from anywhere on your tailnet, without exposing the device to the public internet. When enabled, goMarkableStream creates both a local listener (on `RK_SERVER_BIND_ADDR`) and a Tailscale listener.

**Requirements:**
- Build with the `tailscale` tag: `go build -tags tailscale`
- A Tailscale account

**Environment variables:**
- `RK_TAILSCALE_ENABLED`: (True/False, default: `false`) Enable Tailscale listener.
- `RK_TAILSCALE_PORT`: (String, default: `:8443`) Tailscale listener port.
- `RK_TAILSCALE_HOSTNAME`: (String, default: `gomarkablestream`) Device name in your tailnet.
- `RK_TAILSCALE_STATE_DIR`: (String, default: `/home/root/.tailscale/gomarkablestream`) State directory for Tailscale.
- `RK_TAILSCALE_AUTHKEY`: (String, default: empty) Auth key for headless setup. If unset, Tailscale will display a login URL in the console for interactive authentication.
- `RK_TAILSCALE_EPHEMERAL`: (True/False, default: `false`) Register as ephemeral node (removed when disconnected). **Recommended for most users.** When enabled, a random suffix is appended to the hostname (e.g., `gomarkablestream-a1b2c3`) to avoid naming conflicts if multiple instances are started.
- `RK_TAILSCALE_FUNNEL`: (True/False, default: `false`) Enable public internet access via Tailscale Funnel.
- `RK_TAILSCALE_USE_TLS`: (True/False, default: `false`) Use Tailscale's automatic TLS certificates.
- `RK_TAILSCALE_VERBOSE`: (True/False, default: `false`) Verbose Tailscale logging.

**Example usage:**

```bash
# Enable Tailscale with interactive login (displays login URL in console)
RK_TAILSCALE_ENABLED=true ./goMarkableStream

# Enable Tailscale with auth key (headless setup)
RK_TAILSCALE_ENABLED=true RK_TAILSCALE_AUTHKEY=tskey-auth-xxx ./goMarkableStream

# Recommended: ephemeral mode with auth key (node removed on disconnect, random hostname suffix)
RK_TAILSCALE_ENABLED=true RK_TAILSCALE_EPHEMERAL=true RK_TAILSCALE_AUTHKEY=tskey-auth-xxx ./goMarkableStream

# Access via Tailscale: https://gomarkablestream.your-tailnet.ts.net:8443
# Access locally: https://remarkable.local.:2001
```

**Systemd service with Tailscale:**

Add the environment variables to your systemd service file:
```bash
[Service]
Environment="RK_TAILSCALE_ENABLED=true"
Environment="RK_TAILSCALE_AUTHKEY=tskey-auth-xxx"
ExecStart=/home/root/goMarkableStream
```

### Endpoint Configuration
Add query parameters to the URL (`?parameter=value&otherparameter=value`):
- `color`: (true/false) Enable or disable color.
- `portrait`: (true/false) Enable or disable portrait mode.
- `rate`: (integer, 100-...) Set the frame rate.
- `flip`: (true/false) Enable or disable flipping 180 degrees.

### API Endpoints
- `/`: Main web interface
- `/stream`: The image data stream
- `/events`: WebSocket endpoint for pen input events
- `/gestures`: Endpoint for touch events
- `/version`: Returns the current version of goMarkableStream

## Presentation Mode
`goMarkableStream` introduces an innovative experimental feature that allows users to set a presentation or video in the background, enabling live annotations using a reMarkable tablet.
This feature is ideal for enhancing presentations or educational content by allowing dynamic, real-time interaction.

### How It Works

- To use this feature, append `?present=https://url-of-the-embedded-file` to your streaming URL.
- This action will embed your chosen presentation or video in the stream's background.
- You can then annotate or draw on the reMarkable tablet, with your input appearing over the embedded content in the stream.

### Usage Example

- **Live Presentation Enhancement**: For instance, using Google Slides, you can leave spaces in your slides or use a blank slide to write additional content live.
This feature is perfect for educators, presenters, and anyone looking to make their presentations more interactive and engaging.

![](docs/gorgoniaExample.png)

### Compatibility

- The feature works with any content that can be embedded in an iframe.
This includes a variety of presentation and video platforms.
- Ensure that the content you wish to embed allows iframe integration.

`goMarkableStream` is fully integrated with Reveal.js, making it a perfect tool for presentations.
Switch slides or navigate through your presentation directly from your reMarkable tablet.
This seamless integration enhances the experience of both presenting and viewing, making it ideal for educational and professional environments.

How to: add the `?present=https://your-reveal-js-presentation`

_Note_: Due to browser restrictions, the URL must be HTTPS.

### Limitations and Performance

- **Screen Size**: Currently, the drawing screen size on the tablet is smaller than the presentations, which may affect how content is displayed.
- **Control**: There is no way to control the underlying presentation directly from the tablet.
Users must use the side menu for navigation and control.
- This feature operates seamlessly, with no additional load on the reMarkable tablet, as all rendering is done in the client's browser.

### UI Features

- **Dark Mode**: Toggle between light and dark themes using the sun/moon icon in the sidebar
- **Modern Interface**: Improved UI with better typography and layout
- **Tooltips**: Helpful tooltips on hover for all buttons
- **Feedback Messages**: Visual feedback for user actions

## Technical Details

This tool suits my needs and is an ongoing development. You can find various information about the journey on my blog:
- [Streaming the reMarkable 2](https://blog.owulveryck.info/2021/03/30/streaming-the-remarkable-2.html)
- [Evolving the Game: A clientless streaming tool for reMarkable 2](https://blog.owulveryck.info/2023/07/25/evolving-the-game-a-clientless-streaming-tool-for-remarkable-2.html)

### reMarkable HTTP Server

This is a standalone application that runs directly on a reMarkable tablet.
It does not have any dependencies on third-party libraries, making it a completely self-sufficient solution.
This application exposes an HTTP server with several endpoints.

### Implementation

The image data is read directly from the main process's memory as a BGRA byte array.

**Delta Compression**: The streaming uses delta encoding to minimize bandwidth:
- Only changed pixels are sent between frames (typically 1-5% for e-ink usage)
- Full frames are gzip-compressed (~5-10x reduction) and sent when >30% of pixels change or on first connection
- Delta frames encode runs of changed pixels with their positions

The CPU footprint is relatively low, using about 10% of the CPU for a frame every 200 ms.
You can increase the frame rate, but it will consume slightly more CPU.

On the client side, the streamed byte data is decoded (with automatic gzip decompression for full frames using the browser's native DecompressionStream API) and displayed on a canvas through WebGL.

Additionally, the application features a side menu which allows users to rotate the displayed image.
All image transformations utilize native browser implementations, providing optimized performance.

## Compilation

```bash
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" .
```

To install and run, you can then execute:

```bash
scp goMarkableStream root@remarkable:
ssh root@remarkable ./goMarkableStream
```

## Contributing

I welcome contributions from the community to improve and enhance the reMarkable Screen Streaming Tool.
If you have any ideas, bug reports, or feature requests, please submit them through the GitHub repository's issue tracker.

## License

The reMarkable Screen Streaming Tool is released under the [MIT License](https://opensource.org/licenses/MIT) .
Feel free to modify, distribute, and use the tool in accordance with the terms of the license.

## Tipping

If you plan to buy a reMarkable 2, you can use my [referral program link](https://remarkable.com/referral/PY5B-PH8U).
It will provide a discount for you and also for me.
