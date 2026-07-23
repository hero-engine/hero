package mail

import (
	"fmt"
	"os"
	"syscall"
)

type fileLock struct{ file *os.File }

func acquireLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open mail lock: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock mail store: %w", err)
	}
	return &fileLock{file: f}, nil
}

func (l *fileLock) Close() error {
	u := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	c := l.file.Close()
	if u != nil {
		return fmt.Errorf("unlock mail store: %w", u)
	}
	return c
}
