package main

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

func decodePacked(data []byte) []byte {
	decoded := make([]uint8, 0, len(data)*255)
	total := 0
	for i := 0; i < len(data); i += 2 {
		count := data[i]
		item := data[i+1]
		total = total + int(count) + 1
		for i := 0; i < int(count)+1; i++ {
			decoded = append(decoded, uint8(item))
		}
	}
	log.Println("Total count is:", total)
	log.Println("Decoded size is:", len(decoded))
	return decoded
}

func main() {
	// Basic Authentication credentials
	username := "admin"
	password := "password"

	// Target host and port
	//host := "192.168.1.44"
	host := "192.168.1.47"
	port := "2001" // HTTPS default port

	// Construct Basic Auth header
	auth := fmt.Sprintf("%s:%s", username, password)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	// URL to the /stream endpoint
	url := fmt.Sprintf("https://%s:%s/stream", host, port)

	// Disable SSL certificate verification (not recommended for production)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	// Make a GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set Basic Auth header
	req.Header.Add("Authorization", authHeader)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body into a byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	log.Printf("Received %v bytes", len(body))

	// Create an MD5 hash object
	hash := md5.New()

	// Write the byte array to the hash object
	hash.Write(body)

	// Get the MD5 hash as a byte slice
	hashBytes := hash.Sum(nil)

	// Convert the MD5 hash byte slice to a hexadecimal string
	md5String := hex.EncodeToString(hashBytes)

	log.Println(md5String)

	data := decodePacked(body)

	boundaries := image.Rectangle{
		Min: image.Point{
			X: 0,
			Y: 0,
		},
		Max: image.Point{
			X: remarkable.ScreenWidth,
			Y: remarkable.ScreenHeight,
		},
	}
	img := image.NewGray(boundaries)
	for i := 0; i < len(img.Pix); i++ {
		img.Pix[i] = data[i] * 17
	}
	log.Println(len(data))
	if err := png.Encode(os.Stdout, img); err != nil {
		log.Fatal("Error encoding PNG:", err)
	}

}
