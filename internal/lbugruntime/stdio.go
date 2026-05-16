package lbugruntime

import (
	"fmt"
	"os"
	"sync"
)

type StdioSilencer struct {
	mu sync.Mutex
}

func (s *StdioSilencer) Run(operation func() error) error {
	if operation == nil {
		return fmt.Errorf("operation is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	os.Stdout = devNull
	os.Stderr = devNull
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	return operation()
}
