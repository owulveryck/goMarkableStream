package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

func getPointer(pid string) (int64, error) {
	file, err := os.OpenFile("/proc/"+pid+"/maps", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanAddr := false
	var addr int64
	for scanner.Scan() {
		if scanAddr {
			hex := strings.Split(scanner.Text(), "-")[0]
			addr, err = strconv.ParseInt("0x"+hex, 0, 64)
			break
		}
		if scanner.Text() == `/dev/fb0` {
			scanAddr = true
		}
	}
	return addr, err
}
