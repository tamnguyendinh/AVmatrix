package tsjs

import (
	"context"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestExtractTypeScriptLegacyP7TypeEnvironmentNormalization(t *testing.T) {
	source := []byte(`
class User {}
class Repo {}
class Profile {}

function save(user: User, repo: Repo) {
  const current: User | null = user;
  let optional: Profile | undefined;
  const ambiguous: User | Repo = user;
  const untyped = getUser();
  const made = new User();
  const alias = made;
  const fromProfile = alias.profile;
  const fromMethod = alias.getProfile();
}

const arrow = (profile: Profile) => profile;
`)
	ir := parseAndExtract(t, "src/type-env.ts", "hash-ts", scanner.TypeScript, source)

	requireTypeBindingWithSource(t, ir, "user", "User", scopeir.TypeSourceParameter)
	requireTypeBindingWithSource(t, ir, "repo", "Repo", scopeir.TypeSourceParameter)
	requireTypeBindingWithSource(t, ir, "profile", "Profile", scopeir.TypeSourceParameter)
	requireTypeBindingWithSource(t, ir, "current", "User", scopeir.TypeSourceAnnotation)
	requireTypeBindingWithSource(t, ir, "optional", "Profile", scopeir.TypeSourceAnnotation)
	requireTypeBindingWithSource(t, ir, "made", "User", scopeir.TypeSourceConstructor)
	requireTypeBindingWithSource(t, ir, "alias", "made", scopeir.TypeSourceReceiver)
	requireTypeBindingWithSource(t, ir, "fromProfile", "alias.profile", scopeir.TypeSourceFieldAccess)
	requireTypeBindingWithSource(t, ir, "fromMethod", "alias.getProfile", scopeir.TypeSourceMethodReturn)
	requireNoTypeBinding(t, ir, "ambiguous")
	requireNoTypeBinding(t, ir, "untyped")
}

func TestExtractTypeScriptLegacyP7ScopeBridgeFailuresAreParseLocal(t *testing.T) {
	if _, err := Extract(Request{
		FilePath: "src/missing-root.ts",
		FileHash: "hash-ts",
		Language: scanner.TypeScript,
		Source:   []byte("const x = 1;"),
	}); err == nil {
		t.Fatalf("Extract() with missing root succeeded, want local extraction error")
	}
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: "src/file.ts",
		Language: scanner.TypeScript,
		Source:   []byte("const x = 1;"),
	})
	if err != nil {
		t.Fatalf("parse fixture failed: %v", err)
	}
	defer parsed.Close()
	if _, err := Extract(Request{
		FilePath: "src/file.py",
		FileHash: "hash-py",
		Language: scanner.Python,
		Source:   []byte("x = 1"),
		Root:     parsed.Tree.RootNode(),
	}); err == nil {
		t.Fatalf("Extract() with unsupported language succeeded, want local extraction error")
	}
}

func requireNoTypeBinding(t *testing.T, ir scopeir.ScopeIR, name string) {
	t.Helper()
	for _, scope := range ir.Scopes {
		for _, binding := range scope.TypeBindings {
			if binding.Name == name {
				t.Fatalf("unexpected type binding %s -> %#v", name, binding)
			}
		}
	}
}
