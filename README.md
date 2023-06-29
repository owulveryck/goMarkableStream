[![Go](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml/badge.svg)](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml)

# goMarkableStream
## Overview

The goMarkableStream is a lightweight and user-friendly application designed specifically for the reMarkable tablet. 
Its primary goal is to enable users to stream their reMarkable tablet screen to a web browser without the need for any hacks or modifications that could void the warranty.

## Features

- No hacks or warranty voiding: The tool operates within the boundaries of the reMarkable tablet's intended functionality and does not require any unauthorized modifications.
- No subscription required: Unlike other screen streaming solutions, this tool does not impose any subscription fees or recurring charges. It is completely free to use.
- No client-side installation: Users can access the screen streaming feature directly through their web browser without the need to install any additional software or plugins.
- Color support

## Quick Start

```bash
localhost> ssh root#remarkable
reMarkable: ~/ export GORKVERSION=$(curl -s https://api.github.com/repos/owulveryck/goMarkableStream/releases/latest | grep tag_name | awk -F\" '{print $4}')
reMarkable: ~/ curl -L -s https://github.com/owulveryck/goMarkableStream/releases/download/$GORKVERSION/goMarkableStream_${GORKVERSION//v}_linux_arm.tar.gz | tar xzvf - -O goMarkableStream_${GORKVERSION//v}_linux_arm/goMarkableStream > goMarkableStream
reMarkable: ~/ chmod+x goMarkableStream
reMarkable: ~/ ./goMarkableStream
```

then go to [https://remarkable:2001](https://remarkable:2001) and login with `admin`/`password` (can be changed through environment variables)

_note_: replace _remarkable_ by the IP address if needed.
_note 2_: you can use this to update to a new version (ensure that you killed the previous version before with `kill $(pidof goMarkableStream)`)

## Technical Details

### Data Retrieval from reMarkable Memory

The reMarkable Screen Streaming Tool leverages a combination of techniques to capture the screen data from the reMarkable tablet's memory. 
It utilizes low-level access provided by the device's operating system to retrieve the necessary data. 
This approach ensures that the tool does not require any unauthorized modifications to the reMarkable tablet.

### Data Transmission via WebSocket

Once the screen data is obtained from the reMarkable tablet's memory, the tool serves it to clients via a WebSocket connection. 
A WebSocket is a communication protocol that provides full-duplex communication channels over a single TCP connection, making it ideal for real-time data streaming. 
The WebSocket connection ensures that the screen data is transmitted efficiently and promptly to connected clients.

### Client-Side Rendering with HTML Canvas

On the client side, the reMarkable Screen Streaming Tool fetches the transmitted screen data and renders it using an HTML canvas element. 
The HTML canvas provides a powerful and flexible platform for displaying graphics and images on web pages. 
By leveraging the capabilities of the HTML canvas, the tool ensures a performant and lossless streaming experience for users.

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
This application is configured via the environment. The following environment
variables can be used:

KEY                    TYPE             DEFAULT     REQUIRED    DESCRIPTION
RK_SERVER_BIND_ADDR    String           :2001       true        
RK_SERVER_DEV          True or False    false                   
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

