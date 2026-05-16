package repo

import (
	"runtime"
	"testing"
)

func TestSamePathUsesWindowsCaseRules(t *testing.T) {
	left := `C:\AVmatrix\Repo`
	right := `c:\avmatrix\repo`

	got := SamePath(left, right)
	if runtime.GOOS == "windows" && !got {
		t.Fatal("expected Windows comparison to be case-insensitive")
	}
	if runtime.GOOS != "windows" && got {
		t.Fatal("expected non-Windows comparison to be case-sensitive")
	}
}

func TestRuntimeIDIsStable(t *testing.T) {
	path := "/tmp/avmatrix/repo"
	if RuntimeID(path) != RuntimeID(path) {
		t.Fatal("RuntimeID must be stable for the same path")
	}
	if RuntimeID(path) == RuntimeID("/tmp/avmatrix/other") {
		t.Fatal("RuntimeID should differ for different paths")
	}
}

func TestDisplayLabelDisambiguatesDuplicateNames(t *testing.T) {
	entries := []RegistryEntry{
		{Name: "app", Path: "/repos/one/app"},
		{Name: "app", Path: "/repos/two/app"},
		{Name: "api", Path: "/repos/api"},
	}

	if got := DisplayLabel(entries[0], entries); got != "app (/repos/one/app)" {
		t.Fatalf("DisplayLabel duplicate = %q", got)
	}
	if got := DisplayLabel(entries[2], entries); got != "api" {
		t.Fatalf("DisplayLabel unique = %q", got)
	}
}
