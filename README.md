[![Go](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml/badge.svg)](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml)

# goMarkableStream

I use this toy project to stream my remarkable 2 (firmware 2.5) on my laptop using the local wifi.

[video that shows some features](https://www.youtube.com/watch?v=PzlQ2hEIdCc)

Note: click on the video to take a screenshot. The screenshot is a png file with transparent background.
## Quick start

You need ssh access to your remarkable

Download two files from the [release page](https://github.com/owulveryck/goMarkableStream/releases):

- the server "`Linux/Armv7`" for your remarkable
- the client for your laptop according to the couple `OS/arch`

or build it yourself if you have the go toolchain installed on your machine.

### The server

Copy the server on the remarkable and start it.

```shell
scp goMarkableStreamServer.arm remarkable:
ssh remarkable './goMarkableStreamServer.arm'
```

### The client

- Start the client: `RK_SERVER_ADDR=ip.of.remarkable:2000 ./goMarkableClient`

- Point your browser to [`http://localhost:8080/`](http://localhost:8080/)
- take a screenshot:

There is a `/screenshot` endpoint to grab a picture. 

```shell
ex: 
```shell
❯ curl -o /tmp/screenshot.png http://localhost:8080/screenshot
❯ file /tmp/screenshot.png
/tmp/screenshot.png: PNG image data, 1404 x 1872, 8-bit/color RGBA, non-interlaced
```

### Configuration

It is possible to tweak the configuration via environment variables:

#### Server

| Env var             |  Default  |  Descri[ption
|---------------------|-----------|---------------
| RK_SERVER_BIND_ADDR | :2000     | the TCP listen address

#### Client

| Env var                   |  Default        |  Descri[ption
|---------------------------|-----------------|---------------
| RK_CLIENT_BIND_ADDR       | :8080           | the TCP listen address
| RK_SERVER_ADDR            | remarkabke:2000 | the address of the remarkable
| RK_CLIENT_AUTOROTATE      | true            | activate autorotate (see below)
| RK_CLIENT_SCREENSHOT_DEST | .               | the destination directory to store the screenshots
| RK_CLIENT_PAPER_TEXTURE   | null            | a path to a texture


## How it works?

### Full explanation

I wrote a [blog post](https://blog.owulveryck.info/2021/03/30/streaming-the-remarkable-2.html) that explains all the wiring.
Otherwise a summary is written here.

### The server loop

- The server gets the address of the framebuffer in the memory space of the `xochitl`
- The server launches a "ticketing system" to avoid congestion. The ticketing system is a channel that gets an event every 200ms.
- Then it exposes a gRPC function (with TLS and mutual authentication).
- The gRPC function waits for a "ticket" on the channel, and then grabs the data from the framebuffer.
- It packs the data into an `image` message encoded in protobuf and sends it to the consumer

### The client loop

- The client creates an `MJPEG` stream and serves it over HTTP on the provided address
- The client dial server and sends its certificate, and add the compression header.
- Then it triggers a goroutine to get the `image` in a for loop.
- The image is then encoded into JPEG format and added to the MJPEG stream.

#### Auto-rotate

The client tries to locate the location of the top level switch on the picture (the round one) and rotate the picture accordingly.
This experimental behavior can be disabled by env variables in the client.

Note: the browser does not like the switch of the rotation; the reload of the page solves the problem

#### Texture

There is an experimental texture feature that reads a texture file and use is as a background in the output. The texture does
not apply to the screenshot.
The texture must have this format:

```shell
> identify textures/oldpaper.png
textures/oldpaper.png PNG 1872x1404 1872x1404+0+0 8-bit Gray 256c 886691B 0.010u 0:00.001
```

Example:

```shell
RK_CLIENT_PAPER_TEXTURE=./textures/oldpaper.png goMarkableClient
```

### Security

The communication is using TLS. The client and the server owns an embedded certificate chain (with the CA). There are performing mutual authentication.
A new certificate chain is generated per build. Therefore, if you want restrict the access to your server to your client only, you must rebuild the tool yourself.

### Manual build

_Note_: you need go > 1.16beta to build the tool because of the embedding mechanism for the certificate.

To build the tool manually, the easiest way is to use `goreleaser`:

```shell
goreleaser --snapshot --skip-publish --rm-dist
```

To build the services manually:

```shell
go generate ./... # This generates the certificates
cd server && GOOS=linux GOARCH=arm GOARM=7 go build -o goStreamServer.arm
cd client && go build -o goStreamClient
```

# Tipping

If you plan to buy a reMarkable 2, you can use my [referal program link](https://remarkable.com/referral/PY5B-PH8U). It will provide a discount for you and also for me.

## Acknowledgement

All the people in the reStream projet and specially
[@ddvk](https://github.com/ddvk) and [@raisjn](https://github.com/raisjn)
