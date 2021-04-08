package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

func savePicture(c configuration, img *image.Gray) error {
	name := filepath.Join(c.ScreenShotDest, fmt.Sprintf("goMarkable Screenshot at %v.png", time.Now().Format("2006-01-02 15.04.05")))
	fmt.Println(name)
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	defer f.Close()
	mask := image.NewAlpha(img.Bounds())
	//draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)
	for x := 0; x < mask.Rect.Dx(); x++ {
		for y := 0; y < mask.Rect.Dy(); y++ {
			//get one of r, g, b on the mask image ...
			r, _, _, _ := img.At(x, y).RGBA()
			//... and set it as the alpha value on the mask.
			mask.SetAlpha(x, y, color.Alpha{uint8(255 - r)}) //Assuming that white is your transparency, subtract it from 255
		}
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), image.Transparent, image.Point{}, draw.Src)

	draw.DrawMask(m, img.Bounds(), img, image.Point{}, mask, image.Point{}, draw.Over)

	//if err := png.Encode(f, img); err != nil {
	if err := png.Encode(f, m); err != nil {
		return err
	}
	return nil
}

func screenshotEvent(screenshotC chan<- struct{}) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("press enter to take screenshot -> ")
		reader.ReadString('\n')
		screenshotC <- struct{}{}
	}
}
