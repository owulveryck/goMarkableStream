package main

import (
	"crypto/md5"
	"image"
)

const (
	landscape byte = iota
	portrait
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

func isPortraitLeft(content []byte) bool {
	sig := []byte{83, 234, 230, 173, 67, 108, 25, 219, 155, 106, 67, 4, 203, 188, 104, 255}
	return compareSig(sig, md5.Sum(content[2517769:2517807]))
}

func isLandscapeLeft(content []byte) bool {
	sig := []byte{27, 40, 215, 193, 32, 81, 169, 131, 14, 179, 31, 13, 229, 70, 130, 21}
	return compareSig(sig, md5.Sum(content[115992:116029]))
}

func isPortraitRight(content []byte) bool {
	sig := []byte{5, 185, 165, 108, 82, 71, 18, 100, 38, 92, 191, 135, 173, 171, 224, 97}
	return compareSig(sig, md5.Sum(content[115993:116030]))
}
func isLandscapeRight(content []byte) bool {
	sig := []byte{218, 169, 170, 11, 85, 65, 69, 163, 162, 252, 246, 118, 194, 76, 176, 41}
	return compareSig(sig, md5.Sum(content[114241:114279]))
}

type rotation struct {
	orientation byte
	isActive    bool
}

func (r *rotation) rotate(img *image.Gray) {
	if !r.isActive {
		return
	}
	switch {
	case (isPortraitLeft(img.Pix) || isPortraitRight(img.Pix)) && r.orientation != portrait:
		r.orientation = portrait
	case (isLandscapeLeft(img.Pix) || isLandscapeRight(img.Pix)) && r.orientation != landscape:
		r.orientation = landscape
	}
	if r.orientation == portrait {
		rotate(img)
	}
}

func rotate(img *image.Gray) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	l := len(img.Pix)
	out := make([]uint8, l)
	for i := 0; i < l; i++ {
		j := w*(i%h+1) - i/h - 1
		out[i] = img.Pix[j]
	}
	(*img).Pix = out
	(*img).Rect = image.Rectangle{Max: image.Point{X: h, Y: w}}
	(*img).Stride = h

}
