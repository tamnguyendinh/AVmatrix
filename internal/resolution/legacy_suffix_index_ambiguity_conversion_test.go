package resolution

import (
	"reflect"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestLegacySuffixIndexAmbiguityConversionCoversPythonProximityAndFallbacks(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		current string
		target  string
		want    []string
	}{
		{
			name:    "bare import prefers same directory over suffix match",
			files:   []string{"app/models/user.py", "app/services/user.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  "user",
			want:    []string{"app/services/user.py"},
		},
		{
			name:    "bare import falls back to suffix match",
			files:   []string{"app/models/user.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  "user",
			want:    []string{"app/models/user.py"},
		},
		{
			name:    "repo root importer falls back to root file",
			files:   []string{"user.py", "auth.py"},
			current: "auth.py",
			target:  "user",
			want:    []string{"user.py"},
		},
		{
			name:    "dotted import uses suffix fallback",
			files:   []string{"app/models/utils/helpers.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  "utils.helpers",
			want:    []string{"app/models/utils/helpers.py"},
		},
		{
			name:    "bare import prefers same directory package",
			files:   []string{"app/models/user/__init__.py", "app/services/user/__init__.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  "user",
			want:    []string{"app/services/user/__init__.py"},
		},
		{
			name:    "bare package import falls back to suffix package",
			files:   []string{"app/models/__init__.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  "models",
			want:    []string{"app/models/__init__.py"},
		},
		{
			name:    "windows style current file still uses same directory",
			files:   []string{"app/services/user.py", "app/services/auth.py"},
			current: `app\services\auth.py`,
			target:  "user",
			want:    []string{"app/services/user.py"},
		},
		{
			name:    "pep328 relative import resolves same directory file",
			files:   []string{"app/services/user.py", "app/services/auth.py"},
			current: "app/services/auth.py",
			target:  ".user",
			want:    []string{"app/services/user.py"},
		},
		{
			name:    "pep328 over traversal returns unresolved",
			files:   []string{"app/auth.py", "user.py"},
			current: "app/auth.py",
			target:  "...user",
		},
		{
			name:    "namespace package bare import stays unresolved",
			files:   []string{"app/services/auth.py", "app/services/user/model.py", "app/services/user/queries.py"},
			current: "app/services/auth.py",
			target:  "user",
		},
		{
			name:    "namespace package submodule resolves through suffix fallback",
			files:   []string{"app/services/auth.py", "app/services/user/model.py", "app/services/user/queries.py"},
			current: "app/services/auth.py",
			target:  "user.model",
			want:    []string{"app/services/user/model.py"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace, err := buildWorkspace(scopeIRFiles(scanner.Python, test.files...))
			if err != nil {
				t.Fatalf("buildWorkspace() error = %v", err)
			}
			if got := workspace.resolveImportFiles(scanner.Python, test.current, test.target); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("resolveImportFiles(Python, %s, %s) = %#v, want %#v", test.current, test.target, got, test.want)
			}
		})
	}
}

func TestLegacySuffixIndexAmbiguityConversionKeepsRubyAndExplicitRelativeBehavior(t *testing.T) {
	tests := []struct {
		name     string
		language scanner.Language
		files    []string
		current  string
		target   string
		want     []string
	}{
		{
			name:     "ruby bare require has no python-style proximity",
			language: scanner.Ruby,
			files:    []string{"lib/core/helpers.rb", "lib/utils/helpers.rb", "lib/utils/formatter.rb"},
			current:  "lib/utils/formatter.rb",
			target:   "helpers",
			want:     []string{"lib/core/helpers.rb"},
		},
		{
			name:     "ruby require relative resolves beside importer",
			language: scanner.Ruby,
			files:    []string{"lib/utils/helpers.rb", "lib/utils/formatter.rb"},
			current:  "lib/utils/formatter.rb",
			target:   "./helpers",
			want:     []string{"lib/utils/helpers.rb"},
		},
		{
			name:     "java fully qualified import resolves by unique suffix",
			language: scanner.Java,
			files:    []string{"src/com/a/User.java", "src/com/b/User.java", "src/com/b/Service.java"},
			current:  "src/com/b/Service.java",
			target:   "com.b.User",
			want:     []string{"src/com/b/User.java"},
		},
		{
			name:     "typescript relative import resolves beside importer",
			language: scanner.TypeScript,
			files:    []string{"src/services/user.ts", "src/services/auth.ts", "src/models/user.ts"},
			current:  "src/services/auth.ts",
			target:   "./user",
			want:     []string{"src/services/user.ts"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace, err := buildWorkspace(scopeIRFiles(test.language, test.files...))
			if err != nil {
				t.Fatalf("buildWorkspace() error = %v", err)
			}
			if got := workspace.resolveImportFiles(test.language, test.current, test.target); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("resolveImportFiles(%s, %s, %s) = %#v, want %#v", test.language, test.current, test.target, got, test.want)
			}
		})
	}
}

func scopeIRFiles(language scanner.Language, filePaths ...string) []scopeir.ScopeIR {
	files := make([]scopeir.ScopeIR, 0, len(filePaths))
	for _, filePath := range filePaths {
		files = append(files, scopeir.ScopeIR{FilePath: filePath, Language: language})
	}
	return files
}
