package main

import (
	"errors"
	"image"
	"image/png"
	"io"
	"log"
	"os"
)

func processTexture(c *configuration) error {
	if c.PaperTexture == "" {
		return nil
	}
	f, err := os.Open(c.PaperTexture)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := processTextureFromReader(f)
	if err != nil {
		return err
	}
	c.paperTextureLandscape = img
	c.paperTexturePortrait = cloneImage(img)
	rotate(c.paperTexturePortrait)
	return nil
}

func cloneImage(src *image.Gray) *image.Gray {
	output := &image.Gray{
		Pix:    make([]uint8, len(src.Pix)),
		Rect:   src.Rect,
		Stride: src.Stride,
	}
	copy(output.Pix, src.Pix)
	return output
}

func processTextureFromReader(r io.Reader) (*image.Gray, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	var ok bool
	var imageG *image.Gray
	if imageG, ok = img.(*image.Gray); !ok {
		return nil, errors.New("texture is not gray")
	}
	if imageG.Bounds().Dx() != 1872 || imageG.Bounds().Dy() != 1404 {
		log.Println(imageG.Bounds())
		return nil, errors.New("bad dimensions")
	}
	return imageG, nil
}
