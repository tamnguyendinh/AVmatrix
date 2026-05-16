package group

import "testing"

func TestContractNodeIDUsesFullStableSHA256(t *testing.T) {
	id := contractNodeID("backend", "http::GET::/api", "provider", "src/routes.ts")
	if len(id) != 64 {
		t.Fatalf("contractNodeID length = %d, want 64", len(id))
	}
	if id != contractNodeID("backend", "http::GET::/api", "provider", "src/routes.ts") {
		t.Fatal("contractNodeID is not stable")
	}
	if id == contractNodeID("backend", "http::GET::/api", "provider", "src/other.ts") {
		t.Fatal("contractNodeID did not include filePath")
	}
}

func TestContractLookupIndexResolutionTiers(t *testing.T) {
	index := createContractLookupIndex()
	if got := findContractNode(index, "backend", "provider", "uid-1", "src/a.ts", "foo"); got != "" {
		t.Fatalf("empty index lookup = %q", got)
	}

	indexContract(index, makeBridgeTestContract(StoredContract{SymbolUID: "uid-42"}), "node-A")
	if got := findContractNode(index, "backend", "provider", "uid-42", "anywhere.ts", "anyName"); got != "node-A" {
		t.Fatalf("tier 1 uid lookup = %q", got)
	}
	if got := findContractNode(index, "frontend", "provider", "uid-42", "src/routes.ts", "getUsers"); got != "" {
		t.Fatalf("uid lookup should be repo-scoped, got %q", got)
	}
	if got := findContractNode(index, "backend", "consumer", "uid-42", "src/routes.ts", "getUsers"); got != "" {
		t.Fatalf("uid lookup should be role-scoped, got %q", got)
	}

	refIndex := createContractLookupIndex()
	indexContract(refIndex, makeBridgeTestContract(StoredContract{
		SymbolUID:  "",
		SymbolRef:  SymbolRef{FilePath: "src/ctrl.ts", Name: "handler"},
		SymbolName: "handler",
	}), "node-B")
	if got := findContractNode(refIndex, "backend", "provider", "", "src/ctrl.ts", "handler"); got != "node-B" {
		t.Fatalf("tier 2 ref lookup = %q", got)
	}

	fallbackIndex := createContractLookupIndex()
	indexContract(fallbackIndex, makeBridgeTestContract(StoredContract{
		SymbolUID:  "",
		SymbolRef:  SymbolRef{FilePath: "src/solo.ts", Name: "actualName"},
		SymbolName: "actualName",
	}), "node-C")
	if got := findContractNode(fallbackIndex, "backend", "provider", "", "src/solo.ts", "wrongName"); got != "node-C" {
		t.Fatalf("tier 3 single-file lookup = %q", got)
	}
	indexContract(fallbackIndex, makeBridgeTestContract(StoredContract{
		SymbolUID:  "",
		SymbolRef:  SymbolRef{FilePath: "src/solo.ts", Name: "otherName"},
		SymbolName: "otherName",
		ContractID: "http::POST::/api/x",
	}), "node-D")
	if got := findContractNode(fallbackIndex, "backend", "provider", "", "src/solo.ts", "wrongName"); got != "" {
		t.Fatalf("tier 3 should refuse multiple contracts in same file, got %q", got)
	}
}

func TestContractLookupIndexPrefersUIDOverRef(t *testing.T) {
	index := createContractLookupIndex()
	indexContract(index, makeBridgeTestContract(StoredContract{
		SymbolUID: "uid-1",
		SymbolRef: SymbolRef{FilePath: "src/a.ts", Name: "first"},
	}), "tier1-id")
	indexContract(index, makeBridgeTestContract(StoredContract{
		SymbolUID:  "",
		SymbolRef:  SymbolRef{FilePath: "src/a.ts", Name: "first"},
		ContractID: "http::POST::/api/x",
	}), "tier2-id")
	if got := findContractNode(index, "backend", "provider", "uid-1", "src/a.ts", "first"); got != "tier1-id" {
		t.Fatalf("uid should win over ref, got %q", got)
	}
}

func makeBridgeTestContract(overrides StoredContract) StoredContract {
	contract := StoredContract{
		ContractID: "http::GET::/api/users",
		Type:       "http",
		Role:       "provider",
		SymbolUID:  "uid-1",
		SymbolRef:  SymbolRef{FilePath: "src/routes.ts", Name: "getUsers"},
		SymbolName: "getUsers",
		Confidence: 0.85,
		Meta:       map[string]any{},
		Repo:       "backend",
	}
	if overrides.ContractID != "" {
		contract.ContractID = overrides.ContractID
	}
	if overrides.Type != "" {
		contract.Type = overrides.Type
	}
	if overrides.Role != "" {
		contract.Role = overrides.Role
	}
	contract.SymbolUID = overrides.SymbolUID
	if overrides.SymbolRef.FilePath != "" || overrides.SymbolRef.Name != "" {
		contract.SymbolRef = overrides.SymbolRef
	}
	if overrides.SymbolName != "" {
		contract.SymbolName = overrides.SymbolName
	}
	if overrides.Confidence != 0 {
		contract.Confidence = overrides.Confidence
	}
	if overrides.Meta != nil {
		contract.Meta = overrides.Meta
	}
	if overrides.Repo != "" {
		contract.Repo = overrides.Repo
	}
	return contract
}
