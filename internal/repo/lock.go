package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrLockHeld = errors.New("repository index lock is already held")

type StorageLock struct {
	path string
	file *os.File
}

func AcquireStorageLock(lockPath string) (*StorageLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrLockHeld
		}
		return nil, err
	}
	lock := &StorageLock{path: lockPath, file: file}
	if _, err := fmt.Fprintf(file, "pid=%d\nacquiredAt=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		_ = lock.Release()
		return nil, err
	}
	return lock, nil
}

func (lock *StorageLock) Release() error {
	if lock == nil {
		return nil
	}
	var closeErr error
	if lock.file != nil {
		closeErr = lock.file.Close()
		lock.file = nil
	}
	removeErr := os.Remove(lock.path)
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}
