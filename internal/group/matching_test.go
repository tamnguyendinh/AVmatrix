package group

import "testing"

func TestNormalizeContractID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"http method uppercase", "http::get::/api/users", "http::GET::/api/users"},
		{"http trailing slash stripped", "http::GET::/api/users/", "http::GET::/api/users"},
		{"grpc package lowercase", "grpc::Hr.UserService/GetUser", "grpc::hr.userservice/GetUser"},
		{"grpc malformed leading slash preserved", "grpc::/MyPkg/DoThing", "grpc::/MyPkg/DoThing"},
		{"grpc leading slash no package", "grpc::/Method", "grpc::/Method"},
		{"grpc no slash lowercase", "grpc::ServiceName", "grpc::servicename"},
		{"topic trimmed lowercase", "topic::  Employee.Hired  ", "topic::employee.hired"},
		{"lib lowercase", "lib::@Hr/Common::UserDTO", "lib::@hr/common::userdto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeContractID(tt.in); got != tt.want {
				t.Fatalf("normalizeContractID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRunExactMatchHTTPContracts(t *testing.T) {
	provider := makeGroupTestContract("http::GET::/api/users", "provider", "backend")
	consumer := makeGroupTestContract("http::GET::/api/users", "consumer", "frontend")
	matched, unmatched := runExactMatch([]StoredContract{provider, consumer})
	if len(matched) != 1 {
		t.Fatalf("matched = %#v", matched)
	}
	if matched[0].ContractID != "http::GET::/api/users" || matched[0].MatchType != "exact" || matched[0].From.Repo != "frontend" || matched[0].To.Repo != "backend" {
		t.Fatalf("match = %#v", matched[0])
	}
	if len(unmatched) != 0 {
		t.Fatalf("unmatched = %#v", unmatched)
	}
}

func TestRunExactMatchMultipleConsumersAndUnmatched(t *testing.T) {
	contracts := []StoredContract{
		makeGroupTestContract("http::GET::/api/users", "provider", "backend"),
		makeGroupTestContract("http::GET::/api/users", "consumer", "frontend"),
		makeGroupTestContract("http::GET::/api/users", "consumer", "bff"),
		makeGroupTestContract("http::GET::/api/orphan", "consumer", "mobile"),
	}
	matched, unmatched := runExactMatch(contracts)
	if len(matched) != 2 {
		t.Fatalf("matched = %#v", matched)
	}
	if len(unmatched) != 1 || unmatched[0].ContractID != "http::GET::/api/orphan" {
		t.Fatalf("unmatched = %#v", unmatched)
	}
}

func TestRunExactMatchNormalizesAndSupportsHTTPWildcard(t *testing.T) {
	contracts := []StoredContract{
		makeGroupTestContract("http::POST::/api/users", "provider", "backend"),
		makeGroupTestContract("http::*::/api/users", "consumer", "frontend"),
		makeGroupTestContract("http::GET::/api/orders/", "provider", "orders"),
		makeGroupTestContract("http::get::/api/orders", "consumer", "web"),
	}
	matched, unmatched := runExactMatch(contracts)
	if len(matched) != 2 {
		t.Fatalf("matched = %#v unmatched=%#v", matched, unmatched)
	}
	if matched[0].ContractID != "http::*::/api/users" && matched[1].ContractID != "http::*::/api/users" {
		t.Fatalf("http wildcard was not matched: %#v", matched)
	}
	if len(unmatched) != 0 {
		t.Fatalf("unmatched = %#v", unmatched)
	}
}

func TestRunExactMatchSameRepoRequiresDifferentServices(t *testing.T) {
	provider := makeGroupTestContract("http::GET::/api/users", "provider", "monorepo")
	provider.Service = "services/auth"

	consumer := makeGroupTestContract("http::GET::/api/users", "consumer", "monorepo")
	consumer.Service = "services/gateway"
	matched, _ := runExactMatch([]StoredContract{provider, consumer})
	if len(matched) != 1 || matched[0].From.Service != "services/gateway" || matched[0].To.Service != "services/auth" {
		t.Fatalf("same repo different service match = %#v", matched)
	}

	consumer.Service = "services/auth"
	matched, _ = runExactMatch([]StoredContract{provider, consumer})
	if len(matched) != 0 {
		t.Fatalf("same service should not match: %#v", matched)
	}

	consumer.Service = ""
	matched, _ = runExactMatch([]StoredContract{provider, consumer})
	if len(matched) != 0 {
		t.Fatalf("missing service should not match same repo: %#v", matched)
	}
}

func TestRunExactMatchSkipsGRPCWildcards(t *testing.T) {
	contracts := []StoredContract{
		makeGroupTestGRPCContract("grpc::com.example.UserService/*", "consumer", "frontend", nil),
		makeGroupTestGRPCContract("grpc::com.example.UserService/*", "provider", "backend", nil),
	}
	matched, unmatched := runExactMatch(contracts)
	if len(matched) != 0 || len(unmatched) != 2 {
		t.Fatalf("matched=%#v unmatched=%#v", matched, unmatched)
	}
}

func TestProviderIndexCreatesNormalizedKeys(t *testing.T) {
	index := providerIndex([]StoredContract{
		makeGroupTestGRPCContract("grpc::Com.Example.UserService/GetUser", "provider", "backend", nil),
		makeGroupTestGRPCContract("grpc::Com.Example.UserService/GetUser", "consumer", "frontend", nil),
	})
	providers := index["grpc::com.example.userservice/GetUser"]
	if len(index) != 1 || len(providers) != 1 || providers[0].Role != "provider" {
		t.Fatalf("provider index = %#v", index)
	}
}

func TestRunWildcardMatchGRPC(t *testing.T) {
	provider := makeGroupTestGRPCContract("grpc::com.example.UserService/GetUser", "provider", "backend", nil)
	providerIndex := providerIndex([]StoredContract{provider})

	consumer := makeGroupTestGRPCContract("grpc::com.example.UserService/*", "consumer", "frontend", nil)
	matched, remaining := runWildcardMatch([]StoredContract{consumer}, providerIndex)
	if len(matched) != 1 || len(remaining) != 0 || matched[0].MatchType != "wildcard" || matched[0].ContractID != consumer.ContractID {
		t.Fatalf("fq wildcard matched=%#v remaining=%#v", matched, remaining)
	}

	bareConsumer := makeGroupTestGRPCContract("grpc::UserService/*", "consumer", "mobile", nil)
	matched, remaining = runWildcardMatch([]StoredContract{bareConsumer}, providerIndex)
	if len(matched) != 1 || len(remaining) != 0 || matched[0].From.Repo != "mobile" {
		t.Fatalf("bare wildcard matched=%#v remaining=%#v", matched, remaining)
	}
}

func TestRunWildcardMatchGRPCSkipsNonMatchesAndWildcardProviders(t *testing.T) {
	otherProvider := makeGroupTestGRPCContract("grpc::com.example.OtherService/GetUser", "provider", "backend", nil)
	wildcardProvider := makeGroupTestGRPCContract("grpc::com.example.UserService/*", "provider", "backend", nil)
	consumer := makeGroupTestGRPCContract("grpc::UserService/*", "consumer", "frontend", nil)
	matched, remaining := runWildcardMatch([]StoredContract{consumer}, providerIndex([]StoredContract{otherProvider, wildcardProvider}))
	if len(matched) != 0 || len(remaining) != 1 {
		t.Fatalf("matched=%#v remaining=%#v", matched, remaining)
	}
}

func TestRunWildcardMatchUsesMinimumConfidence(t *testing.T) {
	consumer := makeGroupTestGRPCContract("grpc::com.example.UserService/*", "consumer", "frontend", map[string]any{"confidence": 0.7})
	provider := makeGroupTestGRPCContract("grpc::com.example.UserService/GetUser", "provider", "backend", map[string]any{"confidence": 0.5})
	matched, _ := runWildcardMatch([]StoredContract{consumer}, providerIndex([]StoredContract{provider}))
	if len(matched) != 1 || matched[0].Confidence != 0.5 {
		t.Fatalf("matched = %#v", matched)
	}
}

func TestDedupeContractsAndCrossLinks(t *testing.T) {
	detectedProvider := makeGroupTestGRPCContract("grpc::auth.AuthService/Login", "provider", "platform/auth", nil)
	detectedProvider.SymbolUID = "uid-auth-login"
	detectedProvider.SymbolRef = SymbolRef{FilePath: "src/auth.proto", Name: "Login"}
	detectedProvider.SymbolName = "Login"
	detectedProvider.Meta = map[string]any{"source": "analyze"}

	manifestDuplicate := detectedProvider
	manifestDuplicate.SymbolUID = ""
	manifestDuplicate.SymbolName = "auth.AuthService/Login"
	manifestDuplicate.Meta = map[string]any{"source": "manifest"}

	consumer := makeGroupTestGRPCContract("grpc::auth.AuthService/Login", "consumer", "platform/orders", nil)
	consumer.SymbolUID = "uid-orders-client"
	consumer.SymbolRef = SymbolRef{FilePath: "src/client.ts", Name: "AuthServiceClient"}

	contracts := dedupeContracts([]StoredContract{detectedProvider, manifestDuplicate, consumer, consumer})
	if len(contracts) != 2 {
		t.Fatalf("dedupeContracts() = %#v, want provider and consumer", contracts)
	}
	if contracts[0].Repo != "platform/auth" || contracts[0].SymbolUID != "uid-auth-login" || contracts[0].SymbolName != "Login" {
		t.Fatalf("dedupeContracts kept wrong provider = %#v", contracts[0])
	}

	link := CrossLink{
		From:       CrossLinkEndpoint{Repo: consumer.Repo, SymbolUID: consumer.SymbolUID, SymbolRef: consumer.SymbolRef},
		To:         CrossLinkEndpoint{Repo: detectedProvider.Repo, SymbolUID: detectedProvider.SymbolUID, SymbolRef: detectedProvider.SymbolRef},
		Type:       "grpc",
		ContractID: detectedProvider.ContractID,
		MatchType:  "manifest",
		Confidence: 1,
	}
	links := dedupeCrossLinks([]CrossLink{link, link})
	if len(links) != 1 || links[0].From.Repo != "platform/orders" || links[0].To.Repo != "platform/auth" {
		t.Fatalf("dedupeCrossLinks() = %#v", links)
	}
}

func makeGroupTestContract(id string, role string, repo string) StoredContract {
	return StoredContract{
		ContractID: id,
		Type:       "http",
		Role:       role,
		SymbolUID:  "uid-" + repo + "-" + id,
		SymbolRef:  SymbolRef{FilePath: "src/" + repo + ".ts", Name: "fn-" + id},
		SymbolName: "fn-" + id,
		Confidence: 0.8,
		Meta:       map[string]any{},
		Repo:       repo,
	}
}

func makeGroupTestGRPCContract(id string, role string, repo string, overrides map[string]any) StoredContract {
	contract := StoredContract{
		ContractID: id,
		Type:       "grpc",
		Role:       role,
		SymbolUID:  "uid-" + repo + "-" + id,
		SymbolRef:  SymbolRef{FilePath: "src/" + repo + ".ts", Name: "fn-" + id},
		SymbolName: "fn-" + id,
		Confidence: 0.9,
		Meta:       map[string]any{},
		Repo:       repo,
	}
	if value, ok := overrides["confidence"]; ok {
		contract.Confidence = value.(float64)
	}
	return contract
}
