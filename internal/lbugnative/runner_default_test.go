//go:build !ladybugdb

package lbugnative

import (
	"errors"
	"testing"
)

func TestOpenWriteRunnerUnavailableWithoutNativeBuild(t *testing.T) {
	runner, err := OpenWriteRunner(t.TempDir())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("OpenWriteRunner() error = %v, want ErrUnavailable", err)
	}
	if runner != nil {
		t.Fatalf("OpenWriteRunner() runner = %#v, want nil", runner)
	}
}

func TestOpenReadRunnerUnavailableWithoutNativeBuild(t *testing.T) {
	runner, err := OpenReadRunner(t.TempDir())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("OpenReadRunner() error = %v, want ErrUnavailable", err)
	}
	if runner != nil {
		t.Fatalf("OpenReadRunner() runner = %#v, want nil", runner)
	}
}
