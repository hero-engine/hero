package focus

import (
	"fmt"

	"github.com/hero-engine/hero/internal/filelock"
)

type fileLock struct {
	lock *filelock.Lock
}

func acquireLock(path string) (*fileLock, error) {
	lock, err := filelock.Acquire(path, 0o600)
	if err != nil {
		return nil, fmt.Errorf("lock focus store: %w", err)
	}
	return &fileLock{lock: lock}, nil
}

func (l *fileLock) Close() error {
	if err := l.lock.Close(); err != nil {
		return fmt.Errorf("close focus store lock: %w", err)
	}
	return nil
}
