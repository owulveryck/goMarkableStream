package remarkable

import (
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/sys/unix"
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
	fd      *os.File
	addr    uintptr
}

// const length = 1872 * 1404
const length = 5259264

func (m *Memory) Close() {
	m.fd.Close()
	err := unix.Munmap(m.Backend)
	if err != nil {
		panic(err)
	}
}

func (m *Memory) Init() error {
	pid := findXochitlPID()
	log.Println(pid)
	var err error
	m.memPath = fmt.Sprintf("/proc/%s/mem", pid)
	log.Println(m.memPath)
	// Open the file
	m.fd, err = os.Open(m.memPath)
	if err != nil {
		return err
	}
	offset, err := getFramePointer(pid)
	if err != nil {
		return err
	}
	pageSize := unix.Getpagesize()
	fmt.Printf("The system's memory page size is: %d bytes, offset is %v\n", pageSize, offset)
	// Check if the offset is aligned to the page size
	if offset%int64(pageSize) == 0 {
		fmt.Println("The offset is correctly aligned with the page size.")
	} else {
		fmt.Println("The offset is not aligned with the page size.")
	}

	// Perform the mmap syscall
	// Perform mmap
	m.Backend, err = unix.Mmap(int(m.fd.Fd()), offset, length, unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println("cannot mmap", err)
		return err
	}
	return nil
}
