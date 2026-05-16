package resolution

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestBindingAccumulatorLifecycle(t *testing.T) {
	accumulator := newBindingAccumulator()
	userType := scopeir.TypeRef{
		RawName: "User",
		Source:  scopeir.TypeSourceAnnotation,
	}

	if err := accumulator.appendFile("src/user.ts", []bindingEntry{{Name: "user", Type: userType}}); err != nil {
		t.Fatalf("appendFile() error = %v", err)
	}
	if accumulator.fileCount() != 1 || accumulator.total() != 1 {
		t.Fatalf("unexpected counts before finalize: files=%d total=%d", accumulator.fileCount(), accumulator.total())
	}
	if got, ok := accumulator.fileScopeGet("src/user.ts", "user"); !ok || got.RawName != "User" {
		t.Fatalf("fileScopeGet() = (%#v, %v), want User true", got, ok)
	}
	if err := accumulator.finalize(); err != nil {
		t.Fatalf("finalize() error = %v", err)
	}
	if !accumulator.finalized {
		t.Fatalf("finalize() did not mark accumulator finalized")
	}
	if err := accumulator.appendFile("src/late.ts", []bindingEntry{{Name: "late", Type: userType}}); err == nil || !strings.Contains(err.Error(), "after finalize") {
		t.Fatalf("append after finalize error = %v, want after finalize", err)
	}

	accumulator.dispose()
	if !accumulator.disposed {
		t.Fatalf("dispose() did not mark accumulator disposed")
	}
	if accumulator.fileCount() != 0 || accumulator.total() != 0 {
		t.Fatalf("dispose() did not clear counts: files=%d total=%d", accumulator.fileCount(), accumulator.total())
	}
	if _, ok := accumulator.fileScopeGet("src/user.ts", "user"); ok {
		t.Fatalf("fileScopeGet() returned data after dispose")
	}
	accumulator.dispose()
	if err := accumulator.appendFile("src/disposed.ts", []bindingEntry{{Name: "disposed", Type: userType}}); err == nil || !strings.Contains(err.Error(), "after dispose") {
		t.Fatalf("append after dispose error = %v, want after dispose", err)
	}
}

func TestResolveSkipCompatibilityCrossFileReportsDiagnosticWithoutChangingGraph(t *testing.T) {
	irs := parseFixtureWorkspace(t)
	defaultResult, err := Resolve(irs, Options{})
	if err != nil {
		t.Fatalf("default resolve failed: %v", err)
	}
	skipResult, err := Resolve(irs, Options{SkipCompatibilityCrossFile: true})
	if err != nil {
		t.Fatalf("skip resolve failed: %v", err)
	}

	if !reflect.DeepEqual(buildGraphSignature(defaultResult.Graph), buildGraphSignature(skipResult.Graph)) {
		t.Fatalf("skipCompatibilityCrossFile changed graph signature")
	}
	if !defaultResult.Metrics.CrossFileSkipped || defaultResult.Metrics.CrossFileFilesReprocessed != 0 {
		t.Fatalf("unexpected default cross-file metrics: %#v", defaultResult.Metrics)
	}
	if defaultResult.Metrics.CrossFileSkipReason != "covered-by-scopeir-single-pass-resolution" {
		t.Fatalf("default skip reason = %q", defaultResult.Metrics.CrossFileSkipReason)
	}
	if !skipResult.Metrics.CrossFileSkipped || skipResult.Metrics.CrossFileFilesReprocessed != 0 {
		t.Fatalf("unexpected skip cross-file metrics: %#v", skipResult.Metrics)
	}
	if skipResult.Metrics.CrossFileSkipReason != "disabled-by-pipeline-option" {
		t.Fatalf("skip reason = %q", skipResult.Metrics.CrossFileSkipReason)
	}
	if !skipResult.Metrics.BindingAccumulatorFinalized || !skipResult.Metrics.BindingAccumulatorDisposed {
		t.Fatalf("accumulator lifecycle metrics not closed: %#v", skipResult.Metrics)
	}
}
