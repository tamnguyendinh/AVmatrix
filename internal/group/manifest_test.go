package group

import "testing"

func TestManifestContractsAndLinksCreateProviderConsumerPairs(t *testing.T) {
	contracts, links := manifestContractsAndLinks([]ManifestLink{
		{
			From:     "hr/payroll/backend",
			To:       "hr/hiring/backend",
			Type:     "topic",
			Contract: "employee.hired",
			Role:     "provider",
		},
	})
	if len(contracts) != 2 || len(links) != 1 {
		t.Fatalf("contracts=%#v links=%#v", contracts, links)
	}
	provider := manifestContractByRole(t, contracts, "provider")
	consumer := manifestContractByRole(t, contracts, "consumer")
	if provider.ContractID != "topic::employee.hired" || provider.Type != "topic" || provider.Confidence != 1 {
		t.Fatalf("provider = %#v", provider)
	}
	if consumer.ContractID != provider.ContractID || consumer.Confidence != 1 {
		t.Fatalf("consumer = %#v", consumer)
	}
	if links[0].MatchType != "manifest" || links[0].Confidence != 1 || links[0].From.Repo != "hr/hiring/backend" || links[0].To.Repo != "hr/payroll/backend" {
		t.Fatalf("manifest link = %#v", links[0])
	}
}

func TestManifestContractsAndLinksRoleConsumerUsesFromAsConsumer(t *testing.T) {
	contracts, links := manifestContractsAndLinks([]ManifestLink{
		{
			From:     "sales/admin/bff",
			To:       "sales/crm/backend",
			Type:     "http",
			Contract: "/api/v2/leads/*",
			Role:     "consumer",
		},
	})
	provider := manifestContractByRole(t, contracts, "provider")
	consumer := manifestContractByRole(t, contracts, "consumer")
	if provider.ContractID != "http::*::/api/v2/leads/*" || consumer.ContractID != provider.ContractID {
		t.Fatalf("contracts = %#v", contracts)
	}
	if links[0].From.Repo != "sales/admin/bff" || links[0].To.Repo != "sales/crm/backend" {
		t.Fatalf("manifest link = %#v", links[0])
	}
}

func TestManifestHTTPContractIDCanonicalization(t *testing.T) {
	tests := []struct {
		name     string
		contract string
		want     string
	}{
		{name: "bare path", contract: "/api/orders", want: "http::*::/api/orders"},
		{name: "trailing slash", contract: "/api/orders/", want: "http::*::/api/orders"},
		{name: "relative path", contract: "api/orders", want: "http::*::/api/orders"},
		{name: "duplicate slashes", contract: "//api//orders", want: "http::*::/api/orders"},
		{name: "explicit method", contract: "GET::/api/orders", want: "http::GET::/api/orders"},
		{name: "lowercase method", contract: "get::/api/orders", want: "http::GET::/api/orders"},
		{name: "parameterized path", contract: "POST::/users/:id", want: "http::POST::/users/:id"},
		{name: "empty method path", contract: "GET::", want: "http::GET::/"},
		{name: "empty method portion is bare path", contract: "::/api/orders", want: "http::*::/::/api/orders"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestContractID("http", tt.contract); got != tt.want {
				t.Fatalf("manifestContractID(http, %q) = %q, want %q", tt.contract, got, tt.want)
			}
		})
	}
}

func TestManifestContractsAndLinksCanonicalizesMethodCasing(t *testing.T) {
	lower, _ := manifestContractsAndLinks([]ManifestLink{{
		From:     "gateway",
		To:       "orders-svc",
		Type:     "http",
		Contract: "get::/api/orders",
		Role:     "consumer",
	}})
	upper, _ := manifestContractsAndLinks([]ManifestLink{{
		From:     "gateway",
		To:       "orders-svc",
		Type:     "http",
		Contract: "GET::/api/orders",
		Role:     "consumer",
	}})
	lowerID := manifestContractByRole(t, lower, "provider").ContractID
	upperID := manifestContractByRole(t, upper, "provider").ContractID
	if lowerID != "http::GET::/api/orders" || lowerID != upperID {
		t.Fatalf("lowerID=%q upperID=%q", lowerID, upperID)
	}
}

func TestManifestContractIDForNonHTTPTypes(t *testing.T) {
	tests := []struct {
		name         string
		contractType string
		contract     string
		want         string
	}{
		{name: "grpc", contractType: "grpc", contract: "auth.AuthService/Login", want: "grpc::auth.AuthService/Login"},
		{name: "topic", contractType: "topic", contract: "employee.hired", want: "topic::employee.hired"},
		{name: "lib", contractType: "lib", contract: "@platform/contracts", want: "lib::@platform/contracts"},
		{name: "custom", contractType: "custom", contract: "warehouse-sync", want: "custom::warehouse-sync"},
		{name: "unknown", contractType: "queue", contract: "jobs.created", want: "queue::jobs.created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestContractID(tt.contractType, tt.contract); got != tt.want {
				t.Fatalf("manifestContractID(%q, %q) = %q, want %q", tt.contractType, tt.contract, got, tt.want)
			}
		})
	}
}

func TestManifestContractsAndLinksUseSyntheticSymbols(t *testing.T) {
	contracts, links := manifestContractsAndLinks([]ManifestLink{{
		From:     "platform/orders",
		To:       "platform/auth",
		Type:     "grpc",
		Contract: "auth.AuthService/Login",
		Role:     "consumer",
	}})
	provider := manifestContractByRole(t, contracts, "provider")
	consumer := manifestContractByRole(t, contracts, "consumer")
	if provider.SymbolUID != "manifest::platform/auth::grpc::auth.AuthService/Login" {
		t.Fatalf("provider symbol uid = %q", provider.SymbolUID)
	}
	if consumer.SymbolUID != "manifest::platform/orders::grpc::auth.AuthService/Login" {
		t.Fatalf("consumer symbol uid = %q", consumer.SymbolUID)
	}
	if provider.SymbolRef.Name != "auth.AuthService/Login" || consumer.SymbolRef.Name != "auth.AuthService/Login" {
		t.Fatalf("symbol refs provider=%#v consumer=%#v", provider.SymbolRef, consumer.SymbolRef)
	}
	if len(links) != 1 || links[0].From.SymbolUID != consumer.SymbolUID || links[0].To.SymbolUID != provider.SymbolUID {
		t.Fatalf("manifest cross-link = %#v", links)
	}
}

func TestManifestContractsAndLinksEmptyInput(t *testing.T) {
	contracts, links := manifestContractsAndLinks(nil)
	if len(contracts) != 0 || len(links) != 0 {
		t.Fatalf("contracts=%#v links=%#v", contracts, links)
	}
}

func manifestContractByRole(t *testing.T, contracts []StoredContract, role string) StoredContract {
	t.Helper()
	for _, contract := range contracts {
		if contract.Role == role {
			return contract
		}
	}
	t.Fatalf("missing %s contract in %#v", role, contracts)
	return StoredContract{}
}
