//go:build linux

package ptytest

import (
	"fmt"
	"os"

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

	if err := unix.IoctlSetPointerInt(int(m.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		return nil, nil, fmt.Errorf("TIOCSPTLCK: %w", err)
	}
	n, err := unix.IoctlGetInt(int(m.Fd()), unix.TIOCGPTN)
	if err != nil {
		return nil, nil, fmt.Errorf("TIOCGPTN: %w", err)
	}

	s, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	ok = true
	return m, s, nil
}
