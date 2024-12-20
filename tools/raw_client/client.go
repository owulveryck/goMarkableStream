package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
)

func downloadRawImage(ip, port, username, password string) ([]byte, error) {
	url := fmt.Sprintf("https://%s:%s/raw", ip, port)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Create a request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Add Basic Auth
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return rawData, nil
}

func convertToImage(rawData []byte, width int, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, rawData)
	return img
}

func saveImage(img *image.RGBA, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("error encoding image to PNG: %w", err)
	}

	return nil
}

func main() {
	ip := flag.String("ip", "127.0.0.1", "Server IP address")
	port := flag.String("port", "2001", "Server port")
	width := flag.Int("width", 1624, "Image width")
	height := flag.Int("height", 2154, "Image height")
	output := flag.String("output", "screenshot.png", "Output image file")
	username := flag.String("username", "admin", "Basic auth username")
	password := flag.String("password", "password", "Basic auth password")
	flag.Parse()

	rawData, err := downloadRawImage(*ip, *port, *username, *password)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	img := convertToImage(rawData, *width, *height)

	if err := saveImage(img, *output); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Image saved to %s\n", *output)
}
