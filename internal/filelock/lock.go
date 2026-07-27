package filelock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Lock is an exclusive lock held through an open lock file.
type Lock struct {
	file *os.File
}

type lockOperation func(file *os.File, nonBlocking bool) (busy bool, err error)

// Acquire opens path and blocks until it holds an exclusive lock.
func Acquire(path string, perm fs.FileMode) (*Lock, error) {
	lock, busy, err := acquire(path, perm, false)
	if err != nil {
		return nil, err
	}
	if busy {
		return nil, errors.New("blocking file lock reported busy")
	}
	return lock, nil
}

// TryAcquire opens path and attempts to acquire an exclusive lock without
// blocking. Busy is true only when another owner holds the lock.
func TryAcquire(path string, perm fs.FileMode) (*Lock, bool, error) {
	return acquire(path, perm, true)
}

func acquire(path string, perm fs.FileMode, nonBlocking bool) (*Lock, bool, error) {
	return acquireWith(path, perm, nonBlocking, lockFile)
}

func acquireWith(path string, perm fs.FileMode, nonBlocking bool, operation lockOperation) (*Lock, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return nil, false, fmt.Errorf("open lock file: %w", err)
	}
	if err := file.Chmod(perm); err != nil {
		return nil, false, closeAfterFailure(file, fmt.Errorf("set lock file permissions: %w", err))
	}
	busy, err := operation(file, nonBlocking)
	if err != nil {
		return nil, false, closeAfterFailure(file, fmt.Errorf("lock file: %w", err))
	}
	if busy {
		if err := file.Close(); err != nil {
			return nil, false, fmt.Errorf("close busy lock file: %w", err)
		}
		return nil, true, nil
	}
	return &Lock{file: file}, false, nil
}

// Close releases the lock before closing its file.
func (l *Lock) Close() error {
	return l.closeWith(unlockFile)
}

func (l *Lock) closeWith(unlock func(*os.File) error) error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	unlockErr := unlock(file)
	closeErr := file.Close()
	if unlockErr != nil && closeErr != nil {
		return errors.Join(
			fmt.Errorf("unlock file: %w", unlockErr),
			fmt.Errorf("close lock file: %w", closeErr),
		)
	}
	if unlockErr != nil {
		return fmt.Errorf("unlock file: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close lock file: %w", closeErr)
	}
	return nil
}

func closeAfterFailure(file *os.File, cause error) error {
	if closeErr := file.Close(); closeErr != nil {
		return errors.Join(cause, fmt.Errorf("close lock file after failure: %w", closeErr))
	}
	return cause
}
