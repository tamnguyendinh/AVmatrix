package scanner

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyRegistryPrimaryFlagEnvNamesAndCoverage(t *testing.T) {
	tests := map[Language]string{
		Python:     "REGISTRY_PRIMARY_PYTHON",
		TypeScript: "REGISTRY_PRIMARY_TYPESCRIPT",
		JavaScript: "REGISTRY_PRIMARY_JAVASCRIPT",
		CPlusPlus:  "REGISTRY_PRIMARY_CPP",
		CSharp:     "REGISTRY_PRIMARY_CSHARP",
	}
	for language, want := range tests {
		if got := RegistryPrimaryEnvVarName(language); got != want {
			t.Fatalf("RegistryPrimaryEnvVarName(%s) = %q, want %q", language, got, want)
		}
	}

	seen := make(map[string]struct{})
	for _, language := range RegistryPrimaryCodeLanguages() {
		name := RegistryPrimaryEnvVarName(language)
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate registry primary env var %q", name)
		}
		seen[name] = struct{}{}
	}
	if len(seen) != 16 {
		t.Fatalf("code language env var count = %d, want 16", len(seen))
	}
}

func TestLegacyRegistryPrimaryFlagTruthinessAndIsolation(t *testing.T) {
	clearRegistryPrimaryEnv(t)

	for _, language := range RegistryPrimaryCodeLanguages() {
		if IsRegistryPrimary(language) {
			t.Fatalf("%s is registry-primary by default", language)
		}
	}

	for _, value := range []string{"true", "1", "yes", "  TRUE  ", "Yes"} {
		t.Setenv("REGISTRY_PRIMARY_PYTHON", value)
		if !IsRegistryPrimary(Python) {
			t.Fatalf("Python flag %q should be true", value)
		}
		t.Setenv("REGISTRY_PRIMARY_PYTHON", "")
	}

	for _, value := range []string{"false", "0", "", "off", "no", "disabled", "ture", "enable", "y"} {
		t.Setenv("REGISTRY_PRIMARY_PYTHON", value)
		if IsRegistryPrimary(Python) {
			t.Fatalf("Python flag %q should be false", value)
		}
	}

	t.Setenv("REGISTRY_PRIMARY_PYTHON", "true")
	if !IsRegistryPrimary(Python) || IsRegistryPrimary(Java) || IsRegistryPrimary(Go) {
		t.Fatalf("registry primary flags are not isolated by language")
	}
	t.Setenv("REGISTRY_PRIMARY_CPP", "true")
	t.Setenv("REGISTRY_PRIMARY_CPLUSPLUS", "true")
	if !IsRegistryPrimary(CPlusPlus) {
		t.Fatalf("C++ should read REGISTRY_PRIMARY_CPP")
	}
	t.Setenv("REGISTRY_PRIMARY_CPP", "")
	if IsRegistryPrimary(CPlusPlus) {
		t.Fatalf("C++ should not read TS key-style REGISTRY_PRIMARY_CPLUSPLUS")
	}
}

func TestLegacyRegistryPrimaryFlagPrimaryLanguages(t *testing.T) {
	clearRegistryPrimaryEnv(t)
	if got := PrimaryLanguages(); len(got) != 0 {
		t.Fatalf("PrimaryLanguages() = %#v, want empty", got)
	}

	t.Setenv("REGISTRY_PRIMARY_PYTHON", "true")
	t.Setenv("REGISTRY_PRIMARY_GO", "1")
	t.Setenv("REGISTRY_PRIMARY_JAVA", "false")
	enabled := PrimaryLanguages()
	if _, ok := enabled[Python]; !ok {
		t.Fatalf("PrimaryLanguages missing Python: %#v", enabled)
	}
	if _, ok := enabled[Go]; !ok {
		t.Fatalf("PrimaryLanguages missing Go: %#v", enabled)
	}
	if _, ok := enabled[Java]; ok {
		t.Fatalf("PrimaryLanguages included Java despite false flag: %#v", enabled)
	}
	if len(enabled) != 2 {
		t.Fatalf("PrimaryLanguages size = %d, want 2: %#v", len(enabled), enabled)
	}
}

func clearRegistryPrimaryEnv(t *testing.T) {
	t.Helper()
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(key, "REGISTRY_PRIMARY_") {
			t.Setenv(key, "")
		}
	}
}
