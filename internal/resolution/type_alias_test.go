package resolution

import (
	"context"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/tsjs"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestResolveImportedTypeAlias(t *testing.T) {
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	irs := []scopeir.ScopeIR{
		parseTypeAliasFixture(t, pool, "src/types.ts", `export type UserID = string;`),
		parseTypeAliasFixture(t, pool, "src/consumer.ts", `import { UserID } from './types';

export function load(id: UserID): UserID {
  return id;
}
`),
	}
	result, err := Resolve(irs, Options{})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	consumerFile := graph.GenerateID("File", "src/consumer.ts")
	typeAlias := requireNode(t, result.Graph, "TypeAlias", "src/types.ts", "UserID")
	load := requireNode(t, result.Graph, "Function", "src/consumer.ts", "load")

	requireRelationship(t, result.Graph, graph.RelUses, consumerFile, typeAlias.ID)
	requireRelationship(t, result.Graph, graph.RelUses, load.ID, typeAlias.ID)
}

func parseTypeAliasFixture(t *testing.T, pool *parser.Pool, filePath string, sourceText string) scopeir.ScopeIR {
	t.Helper()
	source := []byte(sourceText)
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: filePath,
		Language: scanner.TypeScript,
		Source:   source,
	})
	if err != nil {
		t.Fatalf("parse %s failed: %v", filePath, err)
	}
	defer parsed.Close()
	ir, err := tsjs.Extract(tsjs.Request{
		FilePath: filePath,
		FileHash: "hash-" + filePath,
		Language: scanner.TypeScript,
		Source:   source,
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		t.Fatalf("extract %s failed: %v", filePath, err)
	}
	return ir
}
