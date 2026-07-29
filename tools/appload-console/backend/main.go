//go:build linux

// AppLoad backend that runs goMarkableStream and forwards its stdout+stderr
// to the QML console frontend over the AppLoad unix SOCK_SEQPACKET socket.
//
// AppLoad starts this binary with argv[1] = path of the unix socket to connect to.
// Wire format (little-endian, matches src/protocol.h and the rust backend client):
//   header = { uint32 type ; uint32 length } followed by `length` payload bytes,
//   each sent as its own SEQPACKET datagram.
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// ---- EDIT THESE for your device -------------------------------------------
// Absolute path to the goMarkableStream binary you already installed.
// (Overridable at runtime with the GMS_BINARY env var.)
const defaultBinary = "/home/root/xovi/exthome/appload/GoMarkableStream/gomarkablestream-RM2"

// Extra args passed to the binary, space-separated. Overridable with GMS_ARGS.
const defaultArgs = ""

// ---------------------------------------------------------------------------

const (
	msgAppendLine  uint32 = 1   // backend -> frontend
	msgFullBuffer  uint32 = 2   // backend -> frontend
	msgRequestBuf  uint32 = 100 // frontend -> backend
	msgStop        uint32 = 101 // frontend -> backend
	msgSysTerminate uint32 = 0xFFFFFFFF
)

var (
	sockFd  int
	sendMu  sync.Mutex

	bufMu sync.Mutex
	buf   strings.Builder

	child   *exec.Cmd
	childMu sync.Mutex
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "missing socket path (argv[1])")
		os.Exit(1)
	}

	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "socket:", err)
		os.Exit(1)
	}
	sockFd = fd
	if err := syscall.Connect(sockFd, &syscall.SockaddrUnix{Name: os.Args[1]}); err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}

	startChild()

	// Receive loop: react to frontend + system messages.
	for {
		mtype, _, err := recvMessage()
		if err != nil {
			// coordinator/frontend went away -> keep running in the background.
			// AppLoad keeps the backend alive; a new frontend reconnects on relaunch.
			if err == io.EOF {
				// Socket closed permanently only when AppLoad tears us down.
				return
			}
			return
		}
		switch mtype {
		case msgRequestBuf:
			bufMu.Lock()
			snapshot := buf.String()
			bufMu.Unlock()
			_ = sendMessage(msgFullBuffer, snapshot)
		case msgStop, msgSysTerminate:
			killChild()
			return
		}
	}
}

func startChild() {
	binary := getenv("GMS_BINARY", defaultBinary)
	argStr := getenv("GMS_ARGS", defaultArgs)
	var args []string
	if strings.TrimSpace(argStr) != "" {
		args = strings.Fields(argStr)
	}

	cmd := exec.Command(binary, args...)
	pr, pw, err := os.Pipe()
	if err != nil {
		appendAndSend(fmt.Sprintf("failed to create pipe: %v\n", err))
		return
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	appendAndSend(fmt.Sprintf("$ %s %s\n", binary, argStr))
	if err := cmd.Start(); err != nil {
		appendAndSend(fmt.Sprintf("failed to start: %v\n", err))
		pw.Close()
		pr.Close()
		return
	}
	pw.Close() // parent keeps only the read end

	childMu.Lock()
	child = cmd
	childMu.Unlock()

	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			appendAndSend(sc.Text() + "\n")
		}
		err := cmd.Wait()
		if err != nil {
			appendAndSend(fmt.Sprintf("[process exited: %v]\n", err))
		} else {
			appendAndSend("[process exited normally]\n")
		}
	}()
}

func killChild() {
	childMu.Lock()
	defer childMu.Unlock()
	if child != nil && child.Process != nil {
		_ = child.Process.Signal(syscall.SIGTERM)
	}
}

func appendAndSend(line string) {
	bufMu.Lock()
	buf.WriteString(line)
	// Cap the retained buffer so re-attach payloads stay small.
	if buf.Len() > 200000 {
		s := buf.String()
		buf.Reset()
		buf.WriteString(s[len(s)-150000:])
	}
	bufMu.Unlock()
	_ = sendMessage(msgAppendLine, line)
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// ---- AppLoad wire protocol ------------------------------------------------

func sendMessage(mtype uint32, contents string) error {
	sendMu.Lock()
	defer sendMu.Unlock()

	var header [8]byte
	binary.LittleEndian.PutUint32(header[0:4], mtype)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(contents)))
	if _, err := syscall.Write(sockFd, header[:]); err != nil {
		return err
	}
	if len(contents) > 0 {
		if _, err := syscall.Write(sockFd, []byte(contents)); err != nil {
			return err
		}
	}
	return nil
}

func recvMessage() (uint32, string, error) {
	header := make([]byte, 8)
	n, err := syscall.Read(sockFd, header)
	if err != nil {
		return 0, "", err
	}
	if n == 0 {
		return 0, "", io.EOF
	}
	mtype := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])
	if length == 0 {
		return mtype, "", nil
	}
	payload := make([]byte, length)
	if _, err := syscall.Read(sockFd, payload); err != nil {
		return 0, "", err
	}
	return mtype, string(payload), nil
}
