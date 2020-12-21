# goMarkableStream

I use this toy project to stream my remarkable 2 (firmware 2.5) on my laptop using the local wifi.

## Quick start

You need ssh access to your remarkable

Download two files:

- the server "`Linux/Armv7`" for your remarkable
- the client for your laptop according to the couple `OS/arch`

### The server

Copy the server on the remarkable and start it.

```shell
scp goMarkableStreamServer.arm remarkable:
ssh remarkable "./goMarkableStreamServer.arm"
```

### The client

- Start the client: `RK_SERVER_ADDRESS=ip.of.remarkable:2000 ./goMarkableClient`

- Point your browser to [`http://localhost:8080/video`](http://localhost:8080/video)

### Configuration

It is possible to tweak the configuration via environment variables:

#### Server

| Env var             |  Default  |  Descri[ption
|---------------------|-----------|---------------
| RK_SERVER_BIND_ADDR | :2000     | the TCP listen address
| RK_FB_ADDRESS       | 4387048   | the location of the pointer to the framebuffer in the `xochitl` process. Default works for firmware 2.5

#### Client

| Env var             |  Default        |  Descri[ption
|---------------------|-----------------|---------------
| RK_CLIENT_BIND_ADDR | :8080           | the TCP listen address
| RK_SERVER_ADDRESS   | remarkabke:2000 | the address of the remarkable

## How it works?

### The server loop 

- The server gets the address of the framebuffer in the memory space of the `xochitl`
- Then it opens a TCP connection and waits for a client (a unique client)
- When it has a connexion, it enters a for loop:
  - Every 200ms, it copies all the bytes of a frame
  - then, it serializes the data in a protobuf message
  - eventually, it sends it in a compressed stream over the wire (zip with the fastest compression)

_Note_ this process is streamed with `io.*` for maximum efficiency.

### The client loop

- The client creates an `MJPEG` stream and serves it over HTTP on the provided address
- The client opens a TCP connection to the server and triggers a goroutine
- It gets the stream and decodes the protobuf message
- For each message:
  - it creates an `Image.Gray` object
  - it encodes it in jpeg
  - it adds it to the mjpeg stream

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

# Acknowledgement

All the people in the reStream projet and specially
[@ddvk](https://github.com/ddvk) and [@raisjn](https://github.com/raisjn)
