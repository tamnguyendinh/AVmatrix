package scopeir

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestMarshalDeterministicMatchesGolden(t *testing.T) {
	raw, err := sampleScopeIR().MarshalDeterministic()
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}
	golden, err := os.ReadFile("testdata/scopeir.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(raw) != string(golden) {
		t.Fatalf("golden mismatch\nwant:\n%s\ngot:\n%s", golden, raw)
	}
}

func TestUnmarshalNormalizesScopeIR(t *testing.T) {
	raw, err := sampleScopeIR().MarshalDeterministic()
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}
	decoded, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	roundTrip, err := decoded.MarshalDeterministic()
	if err != nil {
		t.Fatalf("round trip marshal failed: %v", err)
	}
	if string(roundTrip) != string(raw) {
		t.Fatalf("round trip changed JSON\nfirst:\n%s\nsecond:\n%s", raw, roundTrip)
	}
	if decoded.Version != Version {
		t.Fatalf("Version = %q, want %q", decoded.Version, Version)
	}
}

func TestNormalizedDoesNotMutateSource(t *testing.T) {
	ir := largeUnorderedScopeIR(3)
	firstScopeID := ir.Scopes[0].ID
	firstBinding := ir.Scopes[0].Bindings[0].Name

	normalized := ir.Normalized()

	if ir.Scopes[0].ID != firstScopeID {
		t.Fatalf("Normalized mutated scope order: got %q, want %q", ir.Scopes[0].ID, firstScopeID)
	}
	if ir.Scopes[0].Bindings[0].Name != firstBinding {
		t.Fatalf("Normalized mutated nested bindings: got %q, want %q", ir.Scopes[0].Bindings[0].Name, firstBinding)
	}
	if reflect.DeepEqual(ir, normalized) {
		t.Fatalf("test fixture should differ after normalization")
	}
}

func TestNormalizeInPlaceMatchesNormalized(t *testing.T) {
	ir := largeUnorderedScopeIR(12)

	normalized := ir.Normalized()
	inPlace := ir.NormalizeInPlace()

	if !reflect.DeepEqual(inPlace, normalized) {
		t.Fatalf("NormalizeInPlace() differed from Normalized()")
	}
}

func TestNormalizeOwnedMatchesNormalized(t *testing.T) {
	ir := largeUnorderedScopeIR(12)

	normalized := ir.Normalized()
	owned := ir.NormalizeOwned()

	if !reflect.DeepEqual(owned, normalized) {
		t.Fatalf("NormalizeOwned() differed from Normalized()")
	}
}

func BenchmarkScopeIRSerialization(b *testing.B) {
	ir := sampleScopeIR()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		raw, err := ir.MarshalDeterministic()
		if err != nil {
			b.Fatalf("marshal: %v", err)
		}
		if _, err := Unmarshal(raw); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
}

func BenchmarkScopeIRNormalizedLargeSort(b *testing.B) {
	ir := largeUnorderedScopeIR(2000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		normalized := ir.Normalized()
		if len(normalized.Calls) != len(ir.Calls) {
			b.Fatalf("Calls = %d, want %d", len(normalized.Calls), len(ir.Calls))
		}
	}
}

func sampleScopeIR() ScopeIR {
	moduleScope := "scope:src/app.ts#1:0-6:0:Module"
	functionScope := "scope:src/app.ts#3:0-5:1:Function"
	userDef := "def:src/app.ts#2:0:Class:User"
	runDef := "def:src/app.ts#3:0:Function:run"
	targetRaw := "./user"
	arity := 1
	parameterCount := 1
	requiredParameterCount := 1
	async := true

	return ScopeIR{
		FilePath:    "src/app.ts",
		FileHash:    "hash-app",
		Language:    scanner.TypeScript,
		ModuleScope: moduleScope,
		Scopes: []ScopeFact{
			{
				ID:       functionScope,
				Parent:   &moduleScope,
				Kind:     ScopeFunction,
				Range:    Range{StartLine: 3, StartCol: 0, EndLine: 5, EndCol: 1},
				FilePath: "src/app.ts",
				Bindings: []BindingFact{
					{Name: "run", DefID: runDef, Origin: BindingLocal},
				},
				TypeBindings: []TypeBindingFact{
					{
						Name: "user",
						Type: TypeRef{
							RawName:         "User",
							DeclaredAtScope: functionScope,
							Source:          TypeSourceParameter,
						},
					},
				},
			},
			{
				ID:       moduleScope,
				Kind:     ScopeModule,
				Range:    Range{StartLine: 1, StartCol: 0, EndLine: 6, EndCol: 0},
				FilePath: "src/app.ts",
				Bindings: []BindingFact{
					{Name: "run", DefID: runDef, Origin: BindingLocal},
					{Name: "User", DefID: userDef, Origin: BindingLocal},
				},
				OwnedDefIDs: []string{runDef, userDef},
			},
		},
		Definitions: []DefinitionFact{
			{
				ID:                     runDef,
				FilePath:               "src/app.ts",
				Name:                   "run",
				Label:                  NodeFunction,
				Range:                  Range{StartLine: 3, StartCol: 0, EndLine: 5, EndCol: 1},
				QualifiedName:          "run",
				ParameterCount:         &parameterCount,
				RequiredParameterCount: &requiredParameterCount,
				ParameterTypes:         []string{"User"},
				ReturnType:             "Promise<void>",
				Async:                  &async,
			},
			{
				ID:            userDef,
				FilePath:      "src/app.ts",
				Name:          "User",
				Label:         NodeClass,
				Range:         Range{StartLine: 2, StartCol: 0, EndLine: 2, EndCol: 18},
				QualifiedName: "User",
			},
		},
		Imports: []ImportFact{
			{
				FilePath:     "src/app.ts",
				Kind:         ImportNamed,
				LocalName:    "User",
				ImportedName: "User",
				TargetRaw:    &targetRaw,
			},
		},
		Calls: []CallSiteFact{
			{
				FilePath:         "src/app.ts",
				Name:             "save",
				Range:            Range{StartLine: 4, StartCol: 7, EndLine: 4, EndCol: 11},
				InScope:          functionScope,
				CallForm:         CallMember,
				ExplicitReceiver: "user",
				Arity:            &arity,
			},
		},
		Accesses: []AccessFact{
			{
				FilePath:         "src/app.ts",
				Name:             "id",
				Kind:             AccessRead,
				Range:            Range{StartLine: 4, StartCol: 12, EndLine: 4, EndCol: 14},
				InScope:          functionScope,
				ExplicitReceiver: "user",
			},
		},
		Heritage: []HeritageFact{
			{
				FilePath: "src/app.ts",
				Name:     "BaseUser",
				Kind:     HeritageExtends,
				Range:    Range{StartLine: 2, StartCol: 19, EndLine: 2, EndCol: 27},
				InScope:  moduleScope,
			},
		},
		TypeAnnotations: []TypeAnnotationFact{
			{
				FilePath: "src/app.ts",
				Name:     "user",
				Range:    Range{StartLine: 3, StartCol: 19, EndLine: 3, EndCol: 23},
				InScope:  functionScope,
				Type: TypeRef{
					RawName:         "User",
					DeclaredAtScope: functionScope,
					Source:          TypeSourceParameter,
				},
			},
		},
		ReturnTypes: []ReturnTypeFact{
			{
				DefID:    runDef,
				FilePath: "src/app.ts",
				Range:    Range{StartLine: 3, StartCol: 26, EndLine: 3, EndCol: 39},
				Type: TypeRef{
					RawName:         "Promise<void>",
					DeclaredAtScope: functionScope,
					Source:          TypeSourceReturn,
				},
			},
		},
		Frameworks: []FrameworkFact{
			{
				DefID:                runDef,
				FilePath:             "src/app.ts",
				Framework:            "express",
				Reason:               "decorator",
				EntryPointMultiplier: 2.5,
				Range:                Range{StartLine: 3, StartCol: 0, EndLine: 5, EndCol: 1},
			},
		},
		Domains: []DomainFact{
			{
				DefID:    userDef,
				FilePath: "src/app.ts",
				Domain:   "identity",
				Role:     "model",
				Reason:   "path",
				Range:    Range{StartLine: 2, StartCol: 0, EndLine: 2, EndCol: 18},
			},
		},
	}
}

func largeUnorderedScopeIR(count int) ScopeIR {
	ir := ScopeIR{
		FilePath:    "src/large.ts",
		FileHash:    "hash-large",
		Language:    scanner.TypeScript,
		ModuleScope: "scope:src/large.ts#1:0-9999:0:Module",
	}
	for i := count - 1; i >= 0; i-- {
		scopeID := fmt.Sprintf("scope:src/large.ts#%d:0-%d:0:Function", i+2, i+3)
		defID := fmt.Sprintf("def:src/large.ts#%d:0:Function:item%d", i+2, i)
		ir.Scopes = append(ir.Scopes, ScopeFact{
			ID:       scopeID,
			Kind:     ScopeFunction,
			Range:    Range{StartLine: i + 2, EndLine: i + 3},
			FilePath: "src/large.ts",
			Bindings: []BindingFact{
				{Name: fmt.Sprintf("local%d", i), DefID: defID, Origin: BindingLocal},
				{Name: fmt.Sprintf("arg%d", i), DefID: defID, Origin: BindingLocal},
			},
			OwnedDefIDs: []string{defID},
			TypeBindings: []TypeBindingFact{{
				Name: fmt.Sprintf("arg%d", i),
				Type: TypeRef{
					RawName:         fmt.Sprintf("Type%d", i),
					DeclaredAtScope: scopeID,
					Source:          TypeSourceParameter,
				},
			}},
		})
		ir.Definitions = append(ir.Definitions, DefinitionFact{
			ID:             defID,
			FilePath:       "src/large.ts",
			Name:           fmt.Sprintf("item%d", i),
			Label:          NodeFunction,
			Range:          Range{StartLine: i + 2, EndLine: i + 3},
			QualifiedName:  fmt.Sprintf("item%d", i),
			ParameterTypes: []string{fmt.Sprintf("Type%d", i)},
			Annotations:    []string{fmt.Sprintf("@route%d", i)},
		})
		ir.Calls = append(ir.Calls, CallSiteFact{
			FilePath: "src/large.ts",
			Name:     fmt.Sprintf("call%d", i),
			Range:    Range{StartLine: i + 2, StartCol: 2, EndLine: i + 2, EndCol: 20},
			InScope:  scopeID,
			ArgTypes: []string{fmt.Sprintf("Type%d", i)},
		})
		ir.Accesses = append(ir.Accesses, AccessFact{
			FilePath: "src/large.ts",
			Name:     fmt.Sprintf("field%d", i),
			Kind:     AccessRead,
			Range:    Range{StartLine: i + 2, StartCol: 21, EndLine: i + 2, EndCol: 30},
			InScope:  scopeID,
		})
		ir.Heritage = append(ir.Heritage, HeritageFact{
			FilePath: "src/large.ts",
			Name:     fmt.Sprintf("Base%d", i),
			Kind:     HeritageExtends,
			Range:    Range{StartLine: i + 2, StartCol: 31, EndLine: i + 2, EndCol: 40},
			InScope:  scopeID,
		})
		ir.TypeAnnotations = append(ir.TypeAnnotations, TypeAnnotationFact{
			FilePath: "src/large.ts",
			Name:     fmt.Sprintf("arg%d", i),
			Range:    Range{StartLine: i + 2, StartCol: 41, EndLine: i + 2, EndCol: 50},
			InScope:  scopeID,
			Type: TypeRef{
				RawName:         fmt.Sprintf("Type%d", i),
				DeclaredAtScope: scopeID,
				Source:          TypeSourceParameter,
			},
		})
		ir.ReturnTypes = append(ir.ReturnTypes, ReturnTypeFact{
			DefID:    defID,
			FilePath: "src/large.ts",
			Range:    Range{StartLine: i + 2, StartCol: 51, EndLine: i + 2, EndCol: 60},
			Type: TypeRef{
				RawName:         fmt.Sprintf("Result%d", i),
				DeclaredAtScope: scopeID,
				Source:          TypeSourceReturn,
			},
		})
		ir.Frameworks = append(ir.Frameworks, FrameworkFact{
			DefID:                defID,
			FilePath:             "src/large.ts",
			Framework:            "synthetic",
			Reason:               fmt.Sprintf("reason%d", i),
			EntryPointMultiplier: 1,
			Range:                Range{StartLine: i + 2, EndLine: i + 3},
		})
		ir.Domains = append(ir.Domains, DomainFact{
			DefID:    defID,
			FilePath: "src/large.ts",
			Domain:   "synthetic",
			Role:     fmt.Sprintf("role%d", i),
			Range:    Range{StartLine: i + 2, EndLine: i + 3},
		})
	}
	return ir
}
