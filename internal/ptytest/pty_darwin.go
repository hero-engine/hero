//go:build darwin

package ptytest

import (
	"bytes"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func open() (*os.File, *os.File, error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	ok := false
	defer func() {
		if !ok {
			m.Close()
		}
	}()

	if err := unix.IoctlSetInt(int(m.Fd()), unix.TIOCPTYGRANT, 0); err != nil {
		return nil, nil, fmt.Errorf("TIOCPTYGRANT: %w", err)
	}
	if err := unix.IoctlSetInt(int(m.Fd()), unix.TIOCPTYUNLK, 0); err != nil {
		return nil, nil, fmt.Errorf("TIOCPTYUNLK: %w", err)
	}

	var buf [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, m.Fd(),
		uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&buf[0]))); errno != 0 {
		return nil, nil, fmt.Errorf("TIOCPTYGNAME: %w", errno)
	}
	end := bytes.IndexByte(buf[:], 0)
	if end < 0 {
		end = len(buf)
	}

	s, err := os.OpenFile(string(buf[:end]), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	ok = true
	return m, s, nil
}
