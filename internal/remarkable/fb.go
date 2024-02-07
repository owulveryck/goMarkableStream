package remarkable

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

func GetFileAndPointer() (io.ReaderAt, int64, error) {
	pid := findXochitlPID()
	file, err := os.OpenFile("/proc/"+pid+"/mem", os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return file, 0, err
	}
	pointerAddr, err := getFramePointer(pid)
	if err != nil {
		return file, 0, err
	}
	return file, pointerAddr, nil
}

type Memory struct {
	Backend []byte
	memPath string
	fd      int
	addr    uintptr
}

const length = 1872 * 1404 * 8

func (m *Memory) Close() {
	syscall.Close(m.fd)
	_, _, errno := syscall.Syscall(syscall.SYS_MUNMAP, m.addr, uintptr(length), 0)
	var err error
	if errno != 0 {
		err = errno
	}
	if err != nil {
		panic(err)
	}
}

func (m *Memory) Init() error {
	pid := findXochitlPID()
	var err error
	m.memPath = fmt.Sprintf("/proc/%s/mem", pid)
	m.fd, err = syscall.Open(m.memPath, syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	pointerAddr, err := getFramePointer(pid)
	if err != nil {
		return err
	}
	// Perform the mmap syscall
	var errno syscall.Errno
	m.addr, _, errno = syscall.Syscall6(syscall.SYS_MMAP, 0, uintptr(length), syscall.PROT_READ, syscall.MAP_PRIVATE, uintptr(m.fd), uintptr(pointerAddr))
	if errno != 0 {
		err = errno
	}
	if err != nil {
		return err
	}

	// Now you can access the memory region through the addr pointer
	m.Backend = (*[length]byte)(unsafe.Pointer(m.addr))[:length:length] // Create a slice backed by the mmap'ed memory
	return nil
}
