package main

import (
	"crypto/rand"
	"log"

	"github.com/owulveryck/goMarkableStream/internal/certificate"
)

func main() {
	cw := certificate.NewCertConfigCarrier(rand.Reader)
	err := cw.Make()
	if err != nil {
		log.Fatal(err)
	}
}
