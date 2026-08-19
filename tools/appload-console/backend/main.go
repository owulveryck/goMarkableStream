//go:build linux

// AppLoad backend for the GMS Console app.
//
// It does NOT run goMarkableStream itself. Instead it manages the systemd
// service that was created when you installed goMarkableStream on the device
// (unit "goMarkableStream.service"). The frontend has four buttons which map to
// the commands documented in the goMarkableStream README:
//
//   Start          -> systemctl start   goMarkableStream.service
//   Stop           -> systemctl stop    goMarkableStream.service
//   Restart        -> systemctl restart goMarkableStream.service
//   Status & Logs  -> systemctl status  goMarkableStream.service --no-pager
//                     journalctl -u     goMarkableStream.service -n 200 --no-pager
//
// The combined stdout+stderr of each command is forwarded to the QML window so
// you see exactly what you would see running it from a terminal.
//
// Opening or closing this app has NO effect on the service: nothing is started
// or stopped on launch/teardown. Only the buttons act on the service, and a
// running service keeps running when you close the window.
//
// AppLoad starts this binary with argv[1] = path of the unix socket to connect
// to. Wire format (little-endian, matches src/protocol.h and the rust client):
//   header = { uint32 type ; uint32 length } followed by `length` payload bytes,
//   each sent as its own SOCK_SEQPACKET datagram.
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

// ---- Configuration --------------------------------------------------------
// systemd unit that goMarkableStream's `-install` step creates. Override at
// runtime with the GMS_SERVICE env var if you used a different unit name.
const defaultService = "goMarkableStream.service"

// Number of recent journal lines shown by "Status & Logs".
const journalLines = "200"

// ---------------------------------------------------------------------------

const (
	// backend -> frontend
	msgAppendLine uint32 = 1
	msgFullBuffer uint32 = 2

	// frontend -> backend
	msgRequestBuf uint32 = 100
	msgStart      uint32 = 101
	msgStop       uint32 = 102
	msgRestart    uint32 = 103
	msgStatus     uint32 = 104
	msgClear      uint32 = 105

	msgSysTerminate uint32 = 0xFFFFFFFF
)

var (
	sockFd int
	sendMu sync.Mutex

	bufMu sync.Mutex
	buf   strings.Builder

	// Guards against overlapping commands (one systemctl action at a time).
	busy int32

	service = getenv("GMS_SERVICE", defaultService)
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

	appendLine(fmt.Sprintf("GMS Console — managing %q\n", service))
	appendLine("Use the buttons above to control the service.\n")
	// Show the current state on launch. This is read-only (no side effects).
	dispatch(showStatus)

	// Receive loop: react to frontend + system messages.
	for {
		mtype, _, err := recvMessage()
		if err != nil {
			// Frontend/coordinator went away. Keep the backend alive so a
			// reopened window can reconnect; exit only on a hard socket close.
			if err == io.EOF {
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
		case msgStart:
			dispatch(func() { run("Starting "+service, "systemctl", "start", service); showStatus() })
		case msgStop:
			dispatch(func() { run("Stopping "+service, "systemctl", "stop", service); showStatus() })
		case msgRestart:
			dispatch(func() { run("Restarting "+service, "systemctl", "restart", service); showStatus() })
		case msgStatus:
			dispatch(showStatus)
		case msgClear:
			bufMu.Lock()
			buf.Reset()
			bufMu.Unlock()
			_ = sendMessage(msgFullBuffer, "")
		case msgSysTerminate:
			// AppLoad is tearing us down. Do NOT touch the service.
			return
		}
	}
}

// dispatch runs a command function in the background, rejecting overlapping
// commands so two buttons can't race on the same unit.
func dispatch(fn func()) {
	if !atomic.CompareAndSwapInt32(&busy, 0, 1) {
		appendLine("[busy: a command is already running — please wait]\n")
		return
	}
	go func() {
		defer atomic.StoreInt32(&busy, 0)
		fn()
	}()
}

func showStatus() {
	run("Status of "+service, "systemctl", "status", service, "--no-pager")
	run("Recent logs", "journalctl", "-u", service, "-n", journalLines, "--no-pager")
}

// run executes a command and streams its combined stdout+stderr to the window.
// A non-zero exit is reported but not treated as fatal (e.g. `systemctl status`
// returns 3 when the unit is inactive — we still want to show its output).
func run(title, name string, args ...string) {
	appendLine(fmt.Sprintf("\n===== %s =====\n", title))
	path := resolveTool(name)
	appendLine(fmt.Sprintf("$ %s %s\n", name, strings.Join(args, " ")))
	if path == "" {
		appendLine(fmt.Sprintf("[error: %q not found on PATH]\n", name))
		return
	}

	cmd := exec.Command(path, args...)
	pr, pw, err := os.Pipe()
	if err != nil {
		appendLine(fmt.Sprintf("[error: pipe: %v]\n", err))
		return
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		appendLine(fmt.Sprintf("[error: start: %v]\n", err))
		pw.Close()
		pr.Close()
		return
	}
	pw.Close() // parent keeps only the read end

	sc := bufio.NewScanner(pr)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		appendLine(sc.Text() + "\n")
	}
	pr.Close()

	if err := cmd.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			appendLine(fmt.Sprintf("[exit status %d]\n", ee.ExitCode()))
		} else {
			appendLine(fmt.Sprintf("[error: %v]\n", err))
		}
	} else {
		appendLine("[ok]\n")
	}
}

// resolveTool finds an executable, falling back to common absolute locations in
// case AppLoad launches us with a minimal PATH.
func resolveTool(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, dir := range []string{"/bin", "/usr/bin", "/sbin", "/usr/sbin"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// appendLine appends to the retained buffer (for replay on reconnect) and
// pushes the text to the frontend.
func appendLine(line string) {
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
