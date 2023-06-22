package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/sethvargo/go-envconfig"
	"nhooyr.io/websocket"
)

type configuration struct {
	BindAddr string `env:"RK_SERVER_BIND_ADDR,default=:2001"`
}

const (
	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404
)

var (
	pointerAddr int64
	file        io.ReaderAt
)

//go:embed index.html
var index []byte

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var c configuration
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Fatal(err)
	}
	pid := findPid()
	if len(os.Args) == 2 {
		pid = os.Args[1]
	}

	var err error
	file, err = os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		log.Fatal("cannot open file: ", err)
	}
	pointerAddr, err = getPointer(pid)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		io.Copy(w, bytes.NewReader(index))
	})
	http.HandleFunc("/ws", handleWebSocket)
	err = http.ListenAndServe(c.BindAddr, nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}

}

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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	//conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "Internal Server Error")

	// Simulated pixel data

	imageData := make([]byte, ScreenWidth*ScreenHeight)
	// the informations are int4, therefore store it in a uint8array to reduce data transfer
	uint8Array := make([]uint8, len(imageData)/2)

	for {
		_, err := file.ReadAt(imageData, pointerAddr)
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(imageData); i += 2 {
			packedValue := (uint8(imageData[i]) << 4) | uint8(imageData[i+1])
			uint8Array[i/2] = packedValue
		}

		err = conn.Write(r.Context(), websocket.MessageBinary, uint8Array)
		if err != nil {
			log.Println("Error sending pixel data:", err)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
