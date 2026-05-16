package lbugruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestExtensionStateLoadsFTSAndVectorIdempotently(t *testing.T) {
	runner := &recordingExecRunner{failOnce: map[string]error{
		"LOAD EXTENSION fts": errors.New("extension fts not installed"),
	}}
	var state ExtensionState

	if err := state.EnsureFTS(runner); err != nil {
		t.Fatalf("EnsureFTS() error = %v", err)
	}
	if err := state.EnsureVector(runner); err != nil {
		t.Fatalf("EnsureVector() error = %v", err)
	}
	if err := state.EnsureFTS(runner); err != nil {
		t.Fatalf("EnsureFTS() second call error = %v", err)
	}
	if err := state.EnsureVector(runner); err != nil {
		t.Fatalf("EnsureVector() second call error = %v", err)
	}

	want := []string{
		"LOAD EXTENSION fts",
		"INSTALL fts",
		"LOAD EXTENSION fts",
		"INSTALL VECTOR",
		"LOAD EXTENSION VECTOR",
	}
	if !reflect.DeepEqual(runner.queries, want) {
		t.Fatalf("queries = %#v, want %#v", runner.queries, want)
	}
}

func TestBusyRetryPolicyRetriesOnlyBusyErrors(t *testing.T) {
	attempts := 0
	var slept []time.Duration
	policy := BusyRetryPolicy{
		Attempts:  3,
		BaseDelay: time.Second,
		Sleep: func(ctx context.Context, delay time.Duration) error {
			slept = append(slept, delay)
			return nil
		},
	}
	err := policy.Run(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("database is busy")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if !reflect.DeepEqual(slept, []time.Duration{time.Second, 2 * time.Second}) {
		t.Fatalf("slept = %#v", slept)
	}

	attempts = 0
	err = policy.Run(context.Background(), func() error {
		attempts++
		return errors.New("syntax error")
	})
	if err == nil || attempts != 1 {
		t.Fatalf("non-busy error attempts = %d err = %v, want one failed attempt", attempts, err)
	}
}

func TestIsBusyErrorMatchesLegacyAdapterContract(t *testing.T) {
	busyErrors := []error{
		errors.New("Database is BUSY"),
		errors.New("busy"),
		errors.New("Could not set lock on file"),
		errors.New("database is locked"),
		errors.New("LOCK"),
		errors.New("file already in use by another process"),
		errors.New("already in use"),
		errors.New("Could not set lock on the database file"),
	}
	for _, err := range busyErrors {
		if !IsBusyError(err) {
			t.Fatalf("IsBusyError(%q) = false, want true", err.Error())
		}
	}

	nonBusyErrors := []error{
		errors.New("Table not found"),
		errors.New("Connection refused"),
		errors.New("Syntax error in Cypher query"),
		nil,
	}
	for _, err := range nonBusyErrors {
		if IsBusyError(err) {
			t.Fatalf("IsBusyError(%v) = true, want false", err)
		}
	}
}

func TestIsWALCorruptionErrorMatchesLegacyRecoverySignals(t *testing.T) {
	corruptionErrors := []error{
		errors.New("Runtime exception: Corrupted wal file. Read out invalid WAL record type."),
		errors.New("invalid wal record type in write-ahead log"),
	}
	for _, err := range corruptionErrors {
		if !IsWALCorruptionError(err) {
			t.Fatalf("IsWALCorruptionError(%q) = false, want true", err.Error())
		}
	}

	nonCorruptionErrors := []error{
		errors.New("database is busy"),
		errors.New("table not found"),
		nil,
	}
	for _, err := range nonCorruptionErrors {
		if IsWALCorruptionError(err) {
			t.Fatalf("IsWALCorruptionError(%v) = true, want false", err)
		}
	}
}

func TestBusyRetryPolicyStopsAfterMaxAttempts(t *testing.T) {
	attempts := 0
	policy := BusyRetryPolicy{
		Attempts:  3,
		BaseDelay: time.Second,
		Sleep: func(ctx context.Context, delay time.Duration) error {
			return nil
		},
	}
	err := policy.Run(context.Background(), func() error {
		attempts++
		return errors.New("Could not set lock")
	})
	if err == nil || err.Error() != "Could not set lock" {
		t.Fatalf("Run() error = %v, want final lock error", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestExtensionStateReloadsVectorAfterResetAndBusyRetryCleanup(t *testing.T) {
	var state ExtensionState
	runner := &recordingExecRunner{}

	if err := state.EnsureVector(runner); err != nil {
		t.Fatalf("EnsureVector() initial error = %v", err)
	}
	attempts := 0
	policy := BusyRetryPolicy{
		Attempts: 2,
		Sleep: func(ctx context.Context, delay time.Duration) error {
			return nil
		},
	}
	err := policy.Run(context.Background(), func() error {
		attempts++
		if attempts == 1 {
			state.VectorLoaded = false
			return errors.New("database is BUSY")
		}
		return state.EnsureVector(runner)
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}

	want := []string{
		"INSTALL VECTOR",
		"LOAD EXTENSION VECTOR",
		"INSTALL VECTOR",
		"LOAD EXTENSION VECTOR",
	}
	if !reflect.DeepEqual(runner.queries, want) {
		t.Fatalf("queries = %#v, want %#v", runner.queries, want)
	}
}

func TestRunWithWALRecoveryRemovesSidecarsAndRetries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lbug")
	for _, sidecar := range WALSidecarPaths(dbPath) {
		if err := os.WriteFile(sidecar, []byte("stale"), 0o644); err != nil {
			t.Fatalf("write sidecar %s: %v", sidecar, err)
		}
	}

	attempts := 0
	err := RunWithWALRecovery(dbPath, func() error {
		attempts++
		if attempts == 1 {
			return errors.New("Runtime exception: Corrupted wal file. Read out invalid WAL record type.")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunWithWALRecovery() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	for _, sidecar := range WALSidecarPaths(dbPath) {
		if _, err := os.Stat(sidecar); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("sidecar %s stat error = %v, want removed", sidecar, err)
		}
	}
}

func TestRunWithWALRecoveryDoesNotRetryNonWALErrors(t *testing.T) {
	attempts := 0
	err := RunWithWALRecovery(filepath.Join(t.TempDir(), "lbug"), func() error {
		attempts++
		return errors.New("syntax error")
	})
	if err == nil || err.Error() != "syntax error" {
		t.Fatalf("RunWithWALRecovery() error = %v, want syntax error", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

type recordingExecRunner struct {
	failOnce map[string]error
	queries  []string
}

func (r *recordingExecRunner) Exec(query string) error {
	r.queries = append(r.queries, query)
	if err := r.failOnce[query]; err != nil {
		delete(r.failOnce, query)
		return err
	}
	return nil
}
