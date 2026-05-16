package lbugruntime

import (
	"errors"
	"fmt"
	"os"
)

func WALSidecarPaths(dbPath string) []string {
	return []string{dbPath + ".wal", dbPath + ".lock"}
}

func RemoveWALSidecars(dbPath string) error {
	for _, path := range WALSidecarPaths(dbPath) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove WAL sidecar %s: %w", path, err)
		}
	}
	return nil
}

func RunWithWALRecovery(dbPath string, operation func() error) error {
	if operation == nil {
		return fmt.Errorf("operation is nil")
	}
	err := operation()
	if err == nil || !IsWALCorruptionError(err) {
		return err
	}
	if cleanupErr := RemoveWALSidecars(dbPath); cleanupErr != nil {
		return cleanupErr
	}
	return operation()
}
