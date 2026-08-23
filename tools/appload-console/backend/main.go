//go:build linux

// AppLoad backend for the GMS Console app.
//
// It does NOT run goMarkableStream itself. Instead it manages the systemd
// service that was created when you installed goMarkableStream on the device
// (unit "goMarkableStream.service"). The frontend has buttons which map to the
// commands documented in the goMarkableStream README:
//
//   Start          -> systemctl start   goMarkableStream.service
//   Stop           -> systemctl stop    goMarkableStream.service
//   Restart        -> systemctl restart goMarkableStream.service
//   Status & Logs  -> systemctl status  goMarkableStream.service --no-pager
//                     journalctl -u     goMarkableStream.service -n 200 --no-pager
//
// It also reports a one-word service state ("active"/"inactive"/"failed"/...)
// via `systemctl is-active` so the frontend can show a status summary.
//
// Opening or closing this app has NO effect on the service: nothing is started
// or stopped on launch/teardown. Only the buttons act on the service, and a
// running service keeps running when you close the window.
//
// AppLoad starts this binary with argv[1] = path of the unix socket to connect
// to. Wire format (little-endian, matches src/protocol.h and management.cpp):
//   header = { int32 type ; int32 length }, sent as one SOCK_SEQPACKET datagram,
//   followed by the payload as a SECOND datagram. NOTE: the AppLoad coordinator
//   ALWAYS sends that second datagram even when length == 0, so we must always
//   consume exactly one payload datagram per message. Only a zero-length read of
//   the *header* means the peer actually closed.
package main

import (
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
	msgAppendLine    uint32 = 1
	msgFullBuffer    uint32 = 2
	msgStatusSummary uint32 = 3

	// frontend -> backend
	msgRequestBuf uint32 = 100
	msgStart      uint32 = 101
	msgStop       uint32 = 102
	msgRestart    uint32 = 103
	msgStatus     uint32 = 104
	msgClear      uint32 = 105

	msgSysTerminate uint32 = 0xFFFFFFFF // -1 as int32 (MESSAGE_SYSTEM_TERMINATE)
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

	emit(fmt.Sprintf("GMS Console — managing %q\n", service))
	emit("Use the buttons above to control the service.\n")
	// Show the current state on launch. This is read-only (no side effects).
	dispatch(refresh)

	// Receive loop: react to frontend + system messages.
	for {
		mtype, _, err := recvMessage()
		if err != nil {
			// Peer actually closed. Do NOT touch the service; just exit.
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
			// Refresh the status badge for the (re)attached frontend.
			go updateSummary()
		case msgStart:
			dispatch(func() { run("Starting "+service, "systemctl", "start", service); refresh() })
		case msgStop:
			dispatch(func() { run("Stopping "+service, "systemctl", "stop", service); refresh() })
		case msgRestart:
			dispatch(func() { run("Restarting "+service, "systemctl", "restart", service); refresh() })
		case msgStatus:
			dispatch(refresh)
		case msgClear:
			bufMu.Lock()
			buf.Reset()
			bufMu.Unlock()
			_ = sendMessage(msgFullBuffer, "")
			go updateSummary()
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
		emit("[busy: a command is already running — please wait]\n")
		return
	}
	go func() {
		defer atomic.StoreInt32(&busy, 0)
		fn()
	}()
}

// refresh updates the status badge and prints status + recent logs.
func refresh() {
	updateSummary()
	run("Status of "+service, "systemctl", "status", service, "--no-pager")
	run("Recent logs", "journalctl", "-u", service, "-n", journalLines, "--no-pager")
}

// updateSummary reports a one-word service state to the frontend badge.
func updateSummary() {
	state := "unknown"
	if path := resolveTool("systemctl"); path != "" {
		// `is-active` prints the state word to stdout even on a non-zero exit.
		out, _ := exec.Command(path, "is-active", service).Output()
		if s := strings.TrimSpace(string(out)); s != "" {
			state = s
		}
	}
	_ = sendMessage(msgStatusSummary, state)
}

// run executes a command and sends its combined stdout+stderr to the window.
// A non-zero exit is reported but not treated as fatal (e.g. `systemctl status`
// returns 3 when the unit is inactive — we still want to show its output).
func run(title, name string, args ...string) {
	emit(fmt.Sprintf("\n===== %s =====\n$ %s %s\n", title, name, strings.Join(args, " ")))
	path := resolveTool(name)
	if path == "" {
		emit(fmt.Sprintf("[error: %q not found on PATH]\n", name))
		return
	}

	out, err := exec.Command(path, args...).CombinedOutput()
	if len(out) > 0 {
		emit(string(out))
		if out[len(out)-1] != '\n' {
			emit("\n")
		}
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			emit(fmt.Sprintf("[exit status %d]\n", ee.ExitCode()))
		} else {
			emit(fmt.Sprintf("[error: %v]\n", err))
		}
	} else {
		emit("[ok]\n")
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

// emit appends to the retained buffer (for replay on reconnect) and pushes the
// text to the frontend, split into datagrams small enough for the socket.
func emit(text string) {
	bufMu.Lock()
	buf.WriteString(text)
	if buf.Len() > 200000 { // cap retained buffer so re-attach payloads stay small
		s := buf.String()
		buf.Reset()
		buf.WriteString(s[len(s)-150000:])
	}
	bufMu.Unlock()

	const maxChunk = 8000
	for len(text) > 0 {
		n := len(text)
		if n > maxChunk {
			n = maxChunk
			// Prefer to break at a newline so we never split a UTF-8 rune.
			if i := strings.LastIndexByte(text[:n], '\n'); i > 0 {
				n = i + 1
			}
		}
		_ = sendMessage(msgAppendLine, text[:n])
		text = text[n:]
	}
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
		return 0, "", io.EOF // peer closed the connection
	}
	if n < 8 {
		return 0, "", fmt.Errorf("short header read: %d bytes", n)
	}
	mtype := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	// The coordinator always sends a payload datagram after the header, even
	// when length == 0. We must consume exactly one datagram to stay in sync;
	// otherwise a stray zero-length datagram is later misread as EOF and the
	// backend exits, closing the app. SEQPACKET needs a >=1-byte buffer to
	// actually dequeue a zero-length datagram.
	bufLen := int(length)
	if bufLen == 0 {
		bufLen = 1
	}
	payload := make([]byte, bufLen)
	pn, perr := syscall.Read(sockFd, payload)
	if perr != nil {
		return 0, "", perr
	}
	return mtype, string(payload[:pn]), nil
}
