package main

import (
	"crypto/rand"
	"io/fs"
	"io/ioutil"
	"log"

	"github.com/owulveryck/goMarkableStream/internal/certificate"
)

func main() {
	cw := certificate.NewCertConfigCarrier(rand.Reader)
	err := cw.Make()
	if err != nil {
		log.Fatal(err)
	}
	b, err := cw.GobEncode()
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("certs.bin", b, fs.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}
