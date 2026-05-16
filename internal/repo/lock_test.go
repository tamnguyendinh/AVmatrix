package repo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireStorageLockExcludesConcurrentWritersAndReleases(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".avmatrix", "analyze.lock")

	lock, err := AcquireStorageLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireStorageLock() error = %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}

	_, err = AcquireStorageLock(lockPath)
	if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second AcquireStorageLock() error = %v, want ErrLockHeld", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after release stat error = %v, want not exist", err)
	}
}
