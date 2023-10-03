[![Go](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml/badge.svg)](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml)

# goMarkableStream

![poster](docs/goMarkableStream.png)

## Overview

The goMarkableStream is a lightweight and user-friendly application designed specifically for the reMarkable tablet. 
Its primary goal is to enable users to stream their reMarkable tablet screen to a web browser without the need for any hacks or modifications that could void the warranty.

## Version support

- reMarkable with firmware < 3.4 may use goMarkableStream version < 0.8.6
- reMarkable with firmware >= 3.4 and < 3.6 may use version >= 0.8.6 and < 0.11.0
- reMarkable with firmware >= 3.6 may use version >= 0.11.0

## Features

- No hacks or warranty voiding: The tool operates within the boundaries of the reMarkable tablet's intended functionality and does not require any unauthorized modifications.
- No subscription required: Unlike other screen streaming solutions, this tool does not impose any subscription fees or recurring charges. It is completely free to use.
- No client-side installation: Users can access the screen streaming feature directly through their web browser without the need to install any additional software or plugins.
- Color support

## Quick Start

```bash
localhost> ssh root@remarkable
```

For version >= 3.6 

```bash
export GORKVERSION=$(wget -q -O - https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
wget -q -O - https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/goMarkableStream_${GORKVERSION//v}_linux_arm.tar.gz | tar xzvf - -O goMarkableStream_${GORKVERSION//v}_linux_arm/goMarkableStream > goMarkableStream
chmod +x goMarkableStream
./goMarkableStream
```

for version < 3.6

```bash
export GORKVERSION=$(curl -s https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
curl -L -s https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/goMarkableStream_${GORKVERSION//v}_linux_arm.tar.gz | tar xzvf - -O goMarkableStream_${GORKVERSION//v}_linux_arm/goMarkableStream > goMarkableStream
~/chmod +x goMarkableStream
./goMarkableStream
```

then go to [https://remarkable.local.:2001](https://remarkable.local.:2001) and login with `admin`/`password` (can be changed through environment variables or disable authentication with `-unsafe`)

_note_: _remarkable.local._ may work from apple devices (mDNS resolution). Please note the `.` at the end. If it does not work, you may need to replace `remarkable.local.` by the IP address of the tablet.

_note 2_: you can use this to update to a new version (ensure that you killed the previous version before with `kill $(pidof goMarkableStream)`)

## ngrok builtin

If your reMarkable is on a different network than the displaying device, you can use the `ngrok` builtin feature for automatic tunneling.
To utilize this tunneling, you need to sign up for an ngrok account and [obtain a token from the dashboard](https://dashboard.ngrok.com/get-started/your-authtoken).
Once you have the token, launch reMarkable using the following command:

`NGROK_AUTHTOKEN=YOURTOKEN RK_SERVER_BIND_ADDR=ngrok ./goMarkableStream`

The app will start, displaying a message similar to:

`2023/09/29 16:49:20 listening on 72e5-22-159-32-48.ngrok-free.app` 

Then, connect to `https://72e5-22-159-32-48.ngrok-free.app` to view the result.

## Exxperimental feature: video recording

There is a new exxperimental feature to record the stream in [webm](https://en.wikipedia.org/wiki/WebM) format. This is available on the side menu. 

![poster](docs/goMarkableStreamRecording.webm)

## Technical Details

### Remarkable HTTP Server

This is a standalone application that runs directly on a Remarkable tablet. It does not have any dependencies on third-party libraries, making it a completely self-sufficient solution. This application exposes an HTTP server with two endpoints:
### Endpoints 
- `/`: This endpoint serves an embedded HTML and JavaScript file containing the necessary logic to display an image from the Remarkable tablet on a client's web browser. 
- `/stream`: This endpoint streams the image data from the Remarkable tablet to the client continuously.
### Implementation

The image data is read directly from the main process's memory as a byte array. A simple Run-Length Encoding (RLE) compression algorithm is applied on the tablet to reduce the amount of data transferred between the tablet and the browser. 
The CPU footprint is relatively low, using about 10% of the CPU for a frame every 200 ms. You can increase the frame rate, but it will consume slightly more CPU.

On the client side, the streamed byte data is decoded and displayed on an invisible canvas that matches the size of the Remarkable tablet's display. This canvas is then copied to another responsive canvas for viewing.

Additionally, the application features a side menu which allows users to rotate the displayed image. All image transformations utilize native browser implementations, providing optimized performance.

### Performance

Despite the continuous streaming and processing of images, this application maintains a minimal impact on the CPU, using only about 10% of its capacity. However, increasing the frame rate will proportionally increase CPU usage.
On idle (if not browser is opened), the programs goes to idel and therefore does not drain the battery.

### Client-side Operations

All operations, including image rendering and transformations, are performed using the client's native browser capabilities. This ensures the best performance and compatibility across a range of devices.

### Transformations

The application includes features for rotating the displayed image. This is achieved using the browser's native capabilities, ensuring an optimal performance during transformations.

## Getting Started

To use the reMarkable Screen Streaming Tool, follow these steps:

### Installation

1. Ensure that you have a reMarkable tablet and a computer or device with an ssh client.
2. Get a compiled version from the release or compile it yourself
3. copy the utility on the tablet

### Run

1. launch the utility by conencting via ssh and launch `./goMarkableStream &`
2. go to https://IP-OF-REMARKABLE:2001 (you need to accept the self-signed certificate)

The application is configured via environment variables:

```text
KEY                    TYPE             DEFAULT     REQUIRED    DESCRIPTION
RK_SERVER_BIND_ADDR    String           :2001       true        
RK_SERVER_USERNAME     String           admin                   
RK_SERVER_PASSWORD     String           password                
RK_HTTPS               True or False    true                    
```

### Compilation

`GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build .`

## Contributing

I welcome contributions from the community to improve and enhance the reMarkable Screen Streaming Tool. If you have any ideas, bug reports, or feature requests, please submit them through the GitHub repository's issue tracker.

## License

The reMarkable Screen Streaming Tool is released under the [MIT License](https://opensource.org/licenses/MIT) . Feel free to modify, distribute, and use the tool in accordance with the terms of the license.

## Tipping

If you plan to buy a reMarkable 2, you can use my [referal program link](https://remarkable.com/referral/PY5B-PH8U). It will provide a discount for you and also for me.

