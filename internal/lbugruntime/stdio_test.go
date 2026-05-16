package lbugruntime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

var errSentinel = errors.New("sentinel")

func TestStdioSilencerSuppressesScopedOutputAndRestores(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer

	silencer := &StdioSilencer{}
	if err := silencer.Run(func() error {
		fmt.Fprint(os.Stdout, "hidden stdout")
		fmt.Fprint(os.Stderr, "hidden stderr")
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	fmt.Fprint(os.Stdout, "visible stdout")

	os.Stdout = originalStdout
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := string(raw)
	if strings.Contains(output, "hidden") {
		t.Fatalf("silenced output leaked: %q", output)
	}
	if output != "visible stdout" {
		t.Fatalf("captured output = %q, want visible stdout", output)
	}
}

func TestStdioSilencerRestoresAfterError(t *testing.T) {
	originalStdout := os.Stdout
	silencer := &StdioSilencer{}
	err := silencer.Run(func() error {
		return errSentinel
	})
	if err != errSentinel {
		t.Fatalf("Run() error = %v, want sentinel", err)
	}
	if os.Stdout != originalStdout {
		t.Fatalf("stdout was not restored")
	}
}

func TestStdioSilencerRejectsNilOperation(t *testing.T) {
	silencer := &StdioSilencer{}
	if err := silencer.Run(nil); err == nil || !strings.Contains(err.Error(), "operation is nil") {
		t.Fatalf("Run(nil) error = %v, want operation is nil", err)
	}
}

func TestStdioSilencerSerializesConcurrentUsageAndRestores(t *testing.T) {
	originalStdout := os.Stdout
	silencer := &StdioSilencer{}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := silencer.Run(func() error {
				fmt.Fprintf(os.Stdout, "hidden %d", index)
				return nil
			}); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if os.Stdout != originalStdout {
		t.Fatalf("stdout was not restored after concurrent runs")
	}
}
