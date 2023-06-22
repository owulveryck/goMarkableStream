//go:build js && wasm

package main

import (
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"log"
	"syscall/js"

	"github.com/owulveryck/goMarkableStream/internal/client"

	grpcGzip "google.golang.org/grpc/encoding/gzip"
)

func init() {
	err := grpcGzip.SetLevel(zlib.BestSpeed)
	if err != nil {
		panic(err)
	}
}

func main() {
	// Get the JavaScript canvas element
	doc := js.Global().Get("document")

	// Create a new canvas element
	canvas := doc.Call("createElement", "canvas")
	canvas.Set("width", 1400)  // Set the canvas width
	canvas.Set("height", 1872) // Set the canvas height

	// Append the canvas to the document body
	doc.Get("body").Call("appendChild", canvas)
	//canvas := doc.Call("getElementById", "canvas")

	// Get the 2D rendering context of the canvas
	ctx := canvas.Call("getContext", "2d")
	cd := &canvasDisplayer{ctx}
	c := &client.Configuration{
		ServerAddr:     "127.0.0.1:2000",
		BindAddr:       "",
		AutoRotate:     false,
		ScreenShotDest: "",
		PaperTexture:   "",
		Highlight:      false,
		Colorize:       false,
	}
	g := client.NewGrabber(c, cd)
	log.Println("new grabber created")
	err := g.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

type canvasDisplayer struct {
	ctx js.Value
}

func (c *canvasDisplayer) Display(img *image.Gray) error {
	// Iterate over each pixel of the image
	// Get the image bounds
	clearCanvas(c.ctx)
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get the color of the pixel at (x, y)
			pixel := img.At(x, y)
			r, g, b, a := pixel.RGBA()
			setPixel(c.ctx, x, y, fmt.Sprintf("rgba(%v,%v,%v,%v)", r, g, b, a))
		}
	}
	return nil
}

func setPixel(ctx js.Value, x, y int, color string) {
	// Set the fill style to the specified color
	ctx.Set("fillStyle", color)

	// Draw a filled rectangle representing the pixel at (x, y)
	ctx.Call("fillRect", x, y, 1, 1)
}

func clearCanvas(ctx js.Value) {
	// Get the canvas dimensions
	width := ctx.Get("canvas").Get("width").Int()
	height := ctx.Get("canvas").Get("height").Int()

	// Clear the canvas by drawing a transparent rectangle
	ctx.Call("clearRect", 0, 0, width, height)
}
