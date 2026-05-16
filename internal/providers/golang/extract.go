package golang

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Request struct {
	FilePath string
	FileHash string
	Language scanner.Language
	Source   []byte
	Root     *sitter.Node
}

func Extract(request Request) (scopeir.ScopeIR, error) {
	if request.Root == nil {
		return scopeir.ScopeIR{}, fmt.Errorf("golang extract: missing root node")
	}
	if request.Language != scanner.Go {
		return scopeir.ScopeIR{}, fmt.Errorf("golang extract: unsupported language %q", request.Language)
	}

	c := newCollector(request)
	c.collectScopesAndContext(request.Root)
	c.buildScopes()
	walkKind(request.Root, func(node *sitter.Node, kind string) {
		c.emitDefinitionKind(node, kind)
		c.emitImportKind(node, kind)
		c.emitTypeBindingKind(node, kind)
		c.emitReferenceKind(node, kind)
	})

	return c.result(), nil
}

type collector struct {
	filePath string
	fileHash string
	source   []byte

	scopeCandidates []scopeCandidate
	scopes          []scopeir.ScopeFact
	scopeIndex      map[string]int

	typeDefIDsByName          map[string]string
	returnTypesByCallableName map[string]string

	definitions     []scopeir.DefinitionFact
	imports         []scopeir.ImportFact
	calls           []scopeir.CallSiteFact
	accesses        []scopeir.AccessFact
	heritage        []scopeir.HeritageFact
	typeAnnotations []scopeir.TypeAnnotationFact
	returnTypes     []scopeir.ReturnTypeFact
}

func newCollector(request Request) *collector {
	return &collector{
		filePath:                  request.FilePath,
		fileHash:                  request.FileHash,
		source:                    request.Source,
		scopeIndex:                make(map[string]int),
		typeDefIDsByName:          make(map[string]string),
		returnTypesByCallableName: make(map[string]string),
	}
}

func (c *collector) result() scopeir.ScopeIR {
	moduleScope := ""
	for _, scope := range c.scopes {
		if scope.Kind == scopeir.ScopeModule {
			moduleScope = scope.ID
			break
		}
	}
	return scopeir.ScopeIR{
		FilePath:        c.filePath,
		FileHash:        c.fileHash,
		Language:        scanner.Go,
		ModuleScope:     moduleScope,
		Scopes:          c.scopes,
		Definitions:     c.definitions,
		Imports:         c.imports,
		Calls:           c.calls,
		Accesses:        c.accesses,
		Heritage:        c.heritage,
		TypeAnnotations: c.typeAnnotations,
		ReturnTypes:     c.returnTypes,
	}.NormalizeOwned()
}

func walk(node *sitter.Node, visit func(*sitter.Node)) {
	if node == nil {
		return
	}
	visit(node)
	for index := uint(0); index < node.NamedChildCount(); index++ {
		walk(node.NamedChild(index), visit)
	}
}

func walkKind(node *sitter.Node, visit func(*sitter.Node, string)) {
	if node == nil {
		return
	}
	visit(node, node.Kind())
	for index := uint(0); index < node.NamedChildCount(); index++ {
		walkKind(node.NamedChild(index), visit)
	}
}
