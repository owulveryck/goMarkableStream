package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

var (
	outputFile string
	qrPDFName  string
)

func init() {
	flag.StringVar(&qrPDFName, "output", "goMarkableStreamQRCode.pdf", "The output PDF file")
}

func main() {
	flag.Parse()
	matchedFiles := searchContentFiles(".local/share/remarkable/xochitl/")
	if len(matchedFiles) != 1 {
		log.Fatal("did not find the " + qrPDFName)
	}
	outputFile = matchedFiles[0]
	outputFile = outputFile[:len(outputFile)-len(filepath.Ext(outputFile))] + ".pdf"
	log.Println("using ", outputFile)

	// Fetch initial IP addresses
	ips, err := getIPAddresses()
	if err != nil {
		log.Println("Error fetching IP addresses:", err)
		return
	}

	// Generate initial PDF
	err = generatePDF(ips, outputFile)
	if err != nil {
		log.Println("Error generating PDF:", err)
		return
	}

	for {
		// Wait for a while before checking again
		time.Sleep(1 * time.Minute)

		// Fetch current IP addresses
		currentIPs, err := getIPAddresses()
		if err != nil {
			log.Println("Error fetching IP addresses:", err)
			continue
		}

		// Check if the IP addresses have changed
		if !isEqual(ips, currentIPs) {

			// IP addresses have changed, update the PDF
			err = generatePDF(currentIPs, outputFile)
			if err != nil {
				log.Println("Error generating PDF:", err)
				continue
			}

			// Update the known IP addresses
			ips = currentIPs
		}
	}
}

func generatePDF(ips []net.IP, outputFile string) error {
	// Initialize PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("goMarkableStream IP addresses", false)
	pdf.SetAuthor("The goMarkableStream authors", false)
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)

	// Define QR code and label positions
	x := 10.0
	y := 10.0
	width := 60.0
	height := 60.0
	labelHeight := 10.0

	for _, ip := range ips {
		// Format IP address as URL
		url := fmt.Sprintf("https://%s:2001/", ip)

		// Generate QR Code
		png, err := qrcode.Encode(url, qrcode.Medium, 256)
		if err != nil {
			return fmt.Errorf("error generating QR code for URL %s: %w", url, err)
		}

		// Save QR Code to file
		fileName := fmt.Sprintf("qrcode_%s.png", ip)
		err = os.WriteFile(fileName, png, 0644)
		if err != nil {
			return fmt.Errorf("error saving QR code file: %w", err)
		}

		// Add QR Code to PDF
		pdf.ImageOptions(fileName, x, y, width, height, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")

		// Add IP address label under QR Code
		pdf.SetXY(x, y+height)
		pdf.MultiCell(width, labelHeight, url, "0", "C", false)

		// Update x position for next QR code and label
		x += width + 10

		// Check if we need to create a new line
		if x+width > 210 { // 210mm is the width of an A4 page
			x = 10
			y += height + labelHeight + 10
		}
	}
	pdf.SetXY(x, y+height)
	pdf.MultiCell(width, labelHeight, time.Now().String(), "0", "C", false)

	// Save PDF to file
	err := pdf.OutputFileAndClose(outputFile)
	if err != nil {
		return fmt.Errorf("error saving PDF file: %w", err)
	}

	return nil
}

func getIPAddresses() ([]net.IP, error) {
	ips := []net.IP{}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, intf := range interfaces {
		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.IsGlobalUnicast() {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}

func isEqual(ips1, ips2 []net.IP) bool {
	if len(ips1) != len(ips2) {
		return false
	}

	for i := range ips1 {
		if !ips1[i].Equal(ips2[i]) {
			return false
		}
	}

	return true
}
