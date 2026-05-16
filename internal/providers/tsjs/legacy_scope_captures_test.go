package tsjs

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestExtractTypeScriptLegacyDerivedTypeBindingsAndMemberAccesses(t *testing.T) {
	source := []byte(`
class Profile {
  save() {}
}

class User {
  profile: Profile;
  getProfile(): Profile {
    return this.profile;
  }
}

function makeUser(): User {
  return new User();
}

const makeOther = (): User => new User();

function run(user: User) {
  const current = user;
  const fromField = user.profile;
  const fromMethod = user.getProfile();
  const made = makeUser();
  const other = makeOther();
  current.save();
  user.profile = new Profile();
  user.profile++;
}
`)
	ir := parseAndExtract(t, "src/member-derived.ts", "hash-ts", scanner.TypeScript, source)

	requireTypeBindingWithSource(t, ir, "current", "user", scopeir.TypeSourceReceiver)
	requireTypeBindingWithSource(t, ir, "fromField", "user.profile", scopeir.TypeSourceFieldAccess)
	requireTypeBindingWithSource(t, ir, "fromMethod", "user.getProfile", scopeir.TypeSourceMethodReturn)
	requireTypeBindingWithSource(t, ir, "made", "User", scopeir.TypeSourceReturn)
	requireTypeBindingWithSource(t, ir, "other", "User", scopeir.TypeSourceReturn)
	requireReturnType(t, ir, requireDefinition(t, ir, "makeUser", scopeir.NodeFunction).ID, "User")
	if got := requireDefinition(t, ir, "makeOther", scopeir.NodeFunction).ReturnType; got != "User" {
		t.Fatalf("makeOther return type = %q, want User", got)
	}

	if got := countAccesses(ir, "save", scopeir.AccessRead); got != 0 {
		t.Fatalf("save read access count = %d, want 0 for member-call property", got)
	}
	if got := countCalls(ir, "save", scopeir.CallMember); got != 1 {
		t.Fatalf("save member call count = %d, want 1", got)
	}
	if got := countAccesses(ir, "profile", scopeir.AccessWrite); got != 2 {
		t.Fatalf("profile write access count = %d, want 2", got)
	}
}

func TestExtractTypeScriptLegacyInterfacePropertyAndTypeAliasFacts(t *testing.T) {
	source := []byte(`
class User {}
class Task {}
class Result {}

interface Runnable {
  current: User;
  run(input: Task): Result;
}

type MaybeUser = User | null;
`)
	ir := parseAndExtract(t, "src/interface-facts.ts", "hash-ts", scanner.TypeScript, source)

	runnable := requireDefinition(t, ir, "Runnable", scopeir.NodeInterface)
	current := requireQualifiedDefinition(t, ir, "Runnable.current", scopeir.NodeProperty)
	run := requireQualifiedDefinition(t, ir, "Runnable.run", scopeir.NodeMethod)
	maybeUser := requireDefinition(t, ir, "MaybeUser", scopeir.NodeTypeAlias)

	if current.OwnerID != runnable.ID || current.DeclaredType != "User" {
		t.Fatalf("interface property = %#v, want owner %s and User type", current, runnable.ID)
	}
	if run.OwnerID != runnable.ID || run.ReturnType != "Result" {
		t.Fatalf("interface method = %#v, want owner %s and Result return", run, runnable.ID)
	}
	if maybeUser.QualifiedName != "MaybeUser" {
		t.Fatalf("type alias qualified name = %q, want MaybeUser", maybeUser.QualifiedName)
	}

	requireTypeBindingWithSource(t, ir, "current", "User", scopeir.TypeSourceAnnotation)
	requireTypeBindingWithSource(t, ir, "input", "Task", scopeir.TypeSourceParameter)
	requireTypeAnnotation(t, ir, "Result")
	if got := countTypeAnnotations(ir, "User"); got != 2 {
		t.Fatalf("User type annotation count = %d, want 2", got)
	}
}

func requireTypeBindingWithSource(t *testing.T, ir scopeir.ScopeIR, name string, rawName string, source scopeir.TypeRefSource) {
	t.Helper()
	for _, scope := range ir.Scopes {
		for _, binding := range scope.TypeBindings {
			if binding.Name == name && binding.Type.RawName == rawName && binding.Type.Source == source {
				return
			}
		}
	}
	t.Fatalf("missing type binding %s -> %s/%s in %#v", name, rawName, source, ir.Scopes)
}

func requireQualifiedDefinition(t *testing.T, ir scopeir.ScopeIR, qualified string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.QualifiedName == qualified && def.Label == label {
			return def
		}
	}
	t.Fatalf("missing qualified definition %s/%s in %#v", qualified, label, ir.Definitions)
	return scopeir.DefinitionFact{}
}

func countCalls(ir scopeir.ScopeIR, name string, form scopeir.CallForm) int {
	count := 0
	for _, call := range ir.Calls {
		if call.Name == name && call.CallForm == form {
			count++
		}
	}
	return count
}

func countAccesses(ir scopeir.ScopeIR, name string, kind scopeir.AccessKind) int {
	count := 0
	for _, access := range ir.Accesses {
		if access.Name == name && access.Kind == kind {
			count++
		}
	}
	return count
}

func countTypeAnnotations(ir scopeir.ScopeIR, name string) int {
	count := 0
	for _, annotation := range ir.TypeAnnotations {
		if annotation.Name == name {
			count++
		}
	}
	return count
}
