package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestScanner(t *testing.T) {
	file, err := os.OpenFile("testdata/maps", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanAddr := false
	for scanner.Scan() {
		if scanAddr {
			hex := strings.Split(scanner.Text(), "-")[0]
			dec, err := strconv.ParseInt("0x"+hex, 0, 64)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(dec)

			scanAddr = false
		}
		if scanner.Text() == `/dev/fb0` {
			scanAddr = true
		}
	}
}
