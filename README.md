reMarkable: ~/ cat /usr/share/remarkable/update.conf
[General]
#REMARKABLE_RELEASE_APPID={98DA7DF2-4E3E-4744-9DE6-EC931886ABAB}
#SERVER=https://get-updates.cloud.remarkable.engineering/service/update2
#GROUP=Prod
#PLATFORM=reMarkable2
REMARKABLE_RELEASE_VERSION=2.5.0.27

strace xochitl

...
563 openat(AT_FDCWD, "/dev/fb0", O_RDWR)    = 5
564 ioctl(5, FBIOGET_FSCREENINFO, 0x7ee9d5f4) = 0
565 ioctl(5, FBIOGET_VSCREENINFO, 0x42f0ec) = 0
566 ioctl(5, FBIOPUT_VSCREENINFO, 0x42f0ec) = 0

Global framebuffer is located at 0x42f0ec-4 =0x42f0e8 (4387048 in decimal)


1404*1872 =2628288 

#!/bin/sh
pid=`pidof xochitl`
addr=`dd if=/proc/$pid/mem bs=1 count=4 skip=4387048  2>/dev/null | hexdump | awk '{print $3$2}'`
skipbytes=`printf "%d" $((16#$addr))`
dd if=/proc/$pid/mem bs=1 count=2628288 skip=$skipbytes > out.data

convert -depth 8 -size 1872x1404+0 gray:out.data out.png

2020/12/20 09:34:43 [8 160 187 114]
                     08 a0  bb  72
reMarkable: ~/ dd if=/proc/515/mem  bs=1 count=4 skip=4387048 | hexdump
4+0 records in
4+0 records out
0000000 a008 72bb

reMarkable: ~/ dd if=/proc/515/mem  bs=1 count=4 skip=4387048 | hexdump | awk '{print $3$2}'
4+0 records in
4+0 records out
72bba008