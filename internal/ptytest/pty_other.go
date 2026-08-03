//go:build !darwin && !linux

package ptytest

import (
	"errors"
	"os"
)

func open() (*os.File, *os.File, error) {
	return nil, nil, errors.New("pseudo-terminals are not supported on this platform")
}
