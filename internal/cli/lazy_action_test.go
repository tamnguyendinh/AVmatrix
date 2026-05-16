package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestLegacyLazyActionDefersLoaderUntilInvocation(t *testing.T) {
	calls := 0
	var seenArgs []string
	action := createLazyAction(func() (map[string]any, error) {
		calls++
		return map[string]any{
			"run": func(args ...string) error {
				seenArgs = append([]string(nil), args...)
				return nil
			},
		}, nil
	}, "run")

	if calls != 0 {
		t.Fatalf("loader called before invocation")
	}
	if err := action("arg-1"); err != nil {
		t.Fatalf("action() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1", calls)
	}
	if !reflect.DeepEqual(seenArgs, []string{"arg-1"}) {
		t.Fatalf("seen args = %#v", seenArgs)
	}
}

func TestLegacyLazyActionReportsMissingOrNonFunctionExports(t *testing.T) {
	action := createLazyAction(func() (map[string]any, error) {
		return map[string]any{"notAFunction": "string-value"}, nil
	}, "notAFunction")
	if err := action(); err == nil || !strings.Contains(err.Error(), "notAFunction") {
		t.Fatalf("non-function export error = %v", err)
	}

	cause := errors.New("load failed")
	action = createLazyAction(func() (map[string]any, error) {
		return nil, cause
	}, "run")
	if err := action(); !errors.Is(err, cause) {
		t.Fatalf("loader error = %v, want %v", err, cause)
	}
}
