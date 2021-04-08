[![Go](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml/badge.svg)](https://github.com/owulveryck/goMarkableStream/actions/workflows/go.yml)

# goMarkableStream

I use this toy project to stream my remarkable 2 (firmware 2.5) on my laptop using the local wifi.

[video/demo here](https://www.youtube.com/watch?v=c4-hJ6xRzg4)

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
ssh remarkable './goMarkableStreamServer.arm $(pidof xochitl)'
```

### The client

- Start the client: `RK_SERVER_ADDR=ip.of.remarkable:2000 ./goMarkableClient`

- Point your browser to [`http://localhost:8080/video`](http://localhost:8080/video)

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

## How it works?

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

### Inside the remarkable

Most of the information on how to hack the remarkable 2 comes from the reStream project [See #28 for more info](https://github.com/rien/reStream/issues/28). All I did was to plumb the information to suit my own need.

Here is the recap:

- To get the remarkable version:

```shell
reMarkable: ~/ cat /usr/share/remarkable/update.conf
[General]
#REMARKABLE_RELEASE_APPID={98DA7DF2-4E3E-4744-9DE6-EC931886ABAB}
#SERVER=https://get-updates.cloud.remarkable.engineering/service/update2
#GROUP=Prod
#PLATFORM=reMarkable2
REMARKABLE_RELEASE_VERSION=2.5.0.27
```

- To find the location of the framebuffer pointer:

```shell
strace xochitl

...
563 openat(AT_FDCWD, "/dev/fb0", O_RDWR)    = 5
564 ioctl(5, FBIOGET_FSCREENINFO, 0x7ee9d5f4) = 0
565 ioctl(5, FBIOGET_VSCREENINFO, 0x42f0ec) = 0
566 ioctl(5, FBIOPUT_VSCREENINFO, 0x42f0ec) = 0
```

Global framebuffer is located at 0x42f0ec-4 =0x42f0e8 (4387048 in decimal)

- To extract a picture:

```shell
#!/bin/sh
pid=`pidof xochitl`
addr=`dd if=/proc/$pid/mem bs=1 count=4 skip=4387048  2>/dev/null | hexdump | awk '{print $3$2}'`
skipbytes=`printf "%d" $((16#$addr))`
dd if=/proc/$pid/mem bs=1 count=2628288 skip=$skipbytes > out.data
```

_Note:_ 1404*1872 =2628288 is the size of the binary data to get

## Acknowledgement

All the people in the reStream projet and specially
[@ddvk](https://github.com/ddvk) and [@raisjn](https://github.com/raisjn)
