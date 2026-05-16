package svelte

import (
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/resolution"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

const svelteParityFixture = `<script lang="ts">
import { writable } from 'svelte/store';

interface Named { id: string; }

class Repository {
  write(value: string): void {}
}

export class Service implements Named {
  id: string;
  repo: Repository;
  constructor(repo: Repository) { this.repo = repo; }
  save(user: Named): void {
    const formatted = user.id;
    this.repo.write(formatted);
  }
}

const service = new Service(new Repository());
service.save({ id: 'svelte' });
</script>

<button on:click={() => service.save({ id: 'click' })}>Save</button>
`

func TestExtractSvelteScopeIR(t *testing.T) {
	ir := extract(t, "src/routes/+page.svelte", "hash-svelte", []byte(svelteParityFixture))

	if ir.Language != scanner.Svelte {
		t.Fatalf("language = %q, want %q", ir.Language, scanner.Svelte)
	}
	service := requireDefinition(t, ir, "Service", scopeir.NodeClass)
	requireDefinition(t, ir, "Named", scopeir.NodeInterface)
	requireDefinition(t, ir, "Repository", scopeir.NodeClass)
	requireQualifiedDefinition(t, ir, "Repository.write", scopeir.NodeMethod)
	save := requireQualifiedDefinition(t, ir, "Service.save", scopeir.NodeMethod)
	requireDefinition(t, ir, "formatted", scopeir.NodeVariable)
	requireDefinition(t, ir, "service", scopeir.NodeVariable)
	if save.OwnerID != service.ID {
		t.Fatalf("save owner = %q, want %q", save.OwnerID, service.ID)
	}

	requireImport(t, ir, scopeir.ImportNamed, "writable", "writable", "svelte/store")
	requireCall(t, ir, "Service", scopeir.CallConstructor)
	requireCall(t, ir, "Repository", scopeir.CallConstructor)
	requireCall(t, ir, "save", scopeir.CallMember)
	requireCall(t, ir, "write", scopeir.CallMember)
	requireAccess(t, ir, "repo", scopeir.AccessWrite)
	requireAccess(t, ir, "id", scopeir.AccessRead)
	requireHeritage(t, ir, "Named", scopeir.HeritageImplements)
	requireTypeBinding(t, ir, "this", "Service")
	requireTypeBinding(t, ir, "repo", "Repository")
	requireTypeBinding(t, ir, "user", "Named")
	requireReturnType(t, ir, save.ID, "void")
}

func TestExtractSvelteRejectsNonSvelteLanguage(t *testing.T) {
	_, err := Extract(Request{
		FilePath: "src/service.ts",
		FileHash: "hash-ts",
		Language: scanner.TypeScript,
		Source:   []byte("class Service {}"),
	})
	if err == nil {
		t.Fatal("expected non-Svelte language to fail")
	}
}

func TestExtractSvelteWithoutInlineScriptReturnsEmptyIR(t *testing.T) {
	ir := extract(t, "src/TemplateOnly.svelte", "hash-svelte-template", []byte(`<button>Save</button>`))
	if ir.Language != scanner.Svelte || ir.FileHash != "hash-svelte-template" || ir.FilePath != "src/TemplateOnly.svelte" {
		t.Fatalf("identity mismatch: %#v", ir)
	}
	if len(ir.Definitions) != 0 || len(ir.Calls) != 0 || len(ir.Accesses) != 0 {
		t.Fatalf("template-only Svelte file should not emit script facts: %#v", ir)
	}
}

func TestResolveSvelteGraphParityCounts(t *testing.T) {
	ir := extract(t, "src/routes/+page.svelte", "hash-svelte", []byte(svelteParityFixture))
	result, err := resolution.Resolve([]scopeir.ScopeIR{ir}, resolution.Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	counts := stringRelationshipCounts(result.Graph.RelationshipCountsByType())
	for _, key := range []string{"CALLS", "DEFINES", "HAS_METHOD", "IMPLEMENTS", "INHERITS", "USES"} {
		if counts[key] == 0 {
			t.Fatalf("relationship %s missing in counts %#v", key, counts)
		}
	}
	if result.Metrics.ResolvedCalls == 0 || result.Metrics.ResolvedInheritance == 0 {
		t.Fatalf("expected resolved calls/heritage, got metrics %#v", result.Metrics)
	}
}

func BenchmarkExtractSvelteScopeIR(b *testing.B) {
	source := []byte(svelteParityFixture)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir, err := Extract(Request{
			FilePath: "src/routes/+page.svelte",
			FileHash: "hash-svelte",
			Language: scanner.Svelte,
			Source:   source,
		})
		if err != nil {
			b.Fatalf("extract failed: %v", err)
		}
		if len(ir.Definitions) == 0 || len(ir.Calls) == 0 {
			b.Fatalf("incomplete extraction: %#v", ir)
		}
	}
}

func extract(t *testing.T, filePath string, fileHash string, source []byte) scopeir.ScopeIR {
	t.Helper()
	ir, err := Extract(Request{
		FilePath: filePath,
		FileHash: fileHash,
		Language: scanner.Svelte,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return ir
}

func stringRelationshipCounts(counts map[graph.RelationshipType]int) map[string]int {
	out := make(map[string]int, len(counts))
	for key, value := range counts {
		out[string(key)] = value
	}
	return out
}

func requireDefinition(t *testing.T, ir scopeir.ScopeIR, name string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.Name == name && def.Label == label {
			return def
		}
	}
	t.Fatalf("missing definition %s/%s in %#v", name, label, ir.Definitions)
	return scopeir.DefinitionFact{}
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

func requireImport(t *testing.T, ir scopeir.ScopeIR, kind scopeir.ImportKind, local string, imported string, target string) {
	t.Helper()
	for _, item := range ir.Imports {
		if item.Kind == kind && item.LocalName == local && item.ImportedName == imported && item.TargetRaw != nil && *item.TargetRaw == target {
			return
		}
	}
	t.Fatalf("missing import %s/%s/%s/%s in %#v", kind, local, imported, target, ir.Imports)
}

func requireCall(t *testing.T, ir scopeir.ScopeIR, name string, form scopeir.CallForm) {
	t.Helper()
	for _, call := range ir.Calls {
		if call.Name == name && call.CallForm == form {
			return
		}
	}
	t.Fatalf("missing call %s/%s in %#v", name, form, ir.Calls)
}

func requireAccess(t *testing.T, ir scopeir.ScopeIR, name string, kind scopeir.AccessKind) {
	t.Helper()
	for _, access := range ir.Accesses {
		if access.Name == name && access.Kind == kind {
			return
		}
	}
	t.Fatalf("missing access %s/%s in %#v", name, kind, ir.Accesses)
}

func requireHeritage(t *testing.T, ir scopeir.ScopeIR, name string, kind scopeir.HeritageKind) {
	t.Helper()
	for _, item := range ir.Heritage {
		if item.Name == name && item.Kind == kind {
			return
		}
	}
	t.Fatalf("missing heritage %s/%s in %#v", name, kind, ir.Heritage)
}

func requireTypeBinding(t *testing.T, ir scopeir.ScopeIR, name string, rawType string) {
	t.Helper()
	for _, scope := range ir.Scopes {
		for _, binding := range scope.TypeBindings {
			if binding.Name == name && binding.Type.RawName == rawType {
				return
			}
		}
	}
	t.Fatalf("missing type binding %s:%s in %#v", name, rawType, ir.Scopes)
}

func requireReturnType(t *testing.T, ir scopeir.ScopeIR, defID string, rawType string) {
	t.Helper()
	for _, item := range ir.ReturnTypes {
		if item.DefID == defID && item.Type.RawName == rawType {
			return
		}
	}
	t.Fatalf("missing return type %s:%s in %#v", defID, rawType, ir.ReturnTypes)
}
