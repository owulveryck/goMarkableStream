# Find signature

Simple tool to generate a signature in order to recognize the orientation.

## Left Handed

### Portrait

```shell
> (
cat << \EOF
dd if=/proc/$(pidof xochitl)/mem count=2628288 bs=1 skip=$((16#$(grep '/dev/fb0' /proc/$(pidof xochitl)/maps | sed 's/.*\-\([0-9a-f]*\) .*/\1/')))
EOF
) | ssh root@remarkable | go run . - Portrait
Pseudo-terminal will not be allocated because stdin is not a terminal.
2628288+0 records in
2628288+0 records out
```

```go
package main

import (
        "crypto/md5"
        "fmt"
)

func compareSig(src []byte, sig [16]byte) bool {
        if len(src) != 16 {
                return false
        }
        for i := 0; i < 16; i++ {
                if src[i] != sig[i] {
                        return false
                }
        }
        return true
}

func isPortrait(content []byte) bool {
        sig := []byte{83, 234, 230, 173, 67, 108, 25, 219, 155, 106, 67, 4, 203, 188, 104, 255}
        return compareSig(sig, md5.Sum(content[2517769:2517807]))
}
```

### Landscape

```shell
> (
cat << \EOF
dd if=/proc/$(pidof xochitl)/mem count=2628288 bs=1 skip=$((16#$(grep '/dev/fb0' /proc/$(pidof xochitl)/maps | sed 's/.*\-\([0-9a-f]*\) .*/\1/')))
EOF
) | ssh root@remarkable | go run . - Landscape
Pseudo-terminal will not be allocated because stdin is not a terminal.
2628288+0 records in
2628288+0 records out
```

```go
package main

import (
        "crypto/md5"
        "fmt"
)

func compareSig(src []byte, sig [16]byte) bool {
        if len(src) != 16 {
                return false
        }
        for i := 0; i < 16; i++ {
                if src[i] != sig[i] {
                        return false
                }
        }
        return true
}

func isLandscape(content []byte) bool {
        sig := []byte{27, 40, 215, 193, 32, 81, 169, 131, 14, 179, 31, 13, 229, 70, 130, 21}
        return compareSig(sig, md5.Sum(content[115992:116029]))
}
```

## Right handed

### Portrait

```shell
> (
cat << \EOF
dd if=/proc/$(pidof xochitl)/mem count=2628288 bs=1 skip=$((16#$(grep '/dev/fb0' /proc/$(pidof xochitl)/maps | sed 's/.*\-\([0-9a-f]*\) .*/\1/')))
EOF
) | ssh root@192.168.88.151 | go run . - PortraitRight
Pseudo-terminal will not be allocated because stdin is not a terminal.
2628288+0 records in
2628288+0 records out
```

```go
package main

import (
        "crypto/md5"
        "fmt"
)

func compareSig(src []byte, sig [16]byte) bool {
        if len(src) != 16 {
                return false
        }
        for i := 0; i < 16; i++ {
                if src[i] != sig[i] {
                        return false
                }
        }
        return true
}

func isPortraitright(content []byte) bool {
        sig := []byte{5, 185, 165, 108, 82, 71, 18, 100, 38, 92, 191, 135, 173, 171, 224, 97}
        return compareSig(sig, md5.Sum(content[115993:116030]))
}
```

### Landscape

```shell
 (
cat << \EOF
dd if=/proc/$(pidof xochitl)/mem count=2628288 bs=1 skip=$((16#$(grep '/dev/fb0' /proc/$(pidof xochitl)/maps | sed 's/.*\-\([0-9a-f]*\) .*/\1/')))
EOF
) | ssh root@192.168.88.151 | go run . - landscapeRight
Pseudo-terminal will not be allocated because stdin is not a terminal.
2628288+0 records in
2628288+0 records out
```

```go
package main

import (
        "crypto/md5"
        "fmt"
)

func compareSig(src []byte, sig [16]byte) bool {
        if len(src) != 16 {
                return false
        }
        for i := 0; i < 16; i++ {
                if src[i] != sig[i] {
                        return false
                }
        }
        return true
}

func isLandscaperight(content []byte) bool {
        sig := []byte{218, 169, 170, 11, 85, 65, 69, 163, 162, 252, 246, 118, 194, 76, 176, 41}
        return compareSig(sig, md5.Sum(content[114241:114279]))
}
```
