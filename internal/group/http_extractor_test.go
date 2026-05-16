package group

import (
	"sort"
	"testing"
)

func TestExtractHTTPProvidersFromSourceFrameworks(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "src/controller/UserController.java", `
@RestController
@RequestMapping("/api/v2")
public class UserController {
    @GetMapping("/users")
    public List<User> list() { return service.findAll(); }
    @PostMapping("/users")
    public User create(@RequestBody User user) { return service.save(user); }
    @GetMapping("/users/{id}")
    public User getById(@PathVariable Long id) { return service.findById(id); }
}`)
	writeGroupFile(t, tmpDir, "src/routes/users.ts", `
import { Router } from 'express';
const router = Router();
router.get('/api/users', async (req, res) => { res.json([]); });
router.post('/api/users', async (req, res) => { res.json({}); });
router.delete('/api/users/:id', async (req, res) => { res.sendStatus(204); });`)
	writeGroupFile(t, tmpDir, "cmd/server.go", `
package main
func createOrder(c *gin.Context) {}
func listOrders(c echo.Context) error { return nil }
func healthHandler(w http.ResponseWriter, r *http.Request) {}
func main() {
  r := gin.Default()
  r.POST("/api/orders/:id", createOrder)
  e := echo.New()
  e.GET("/api/orders", listOrders)
  http.HandleFunc("/api/health", healthHandler)
}`)
	writeGroupFile(t, tmpDir, "src/orders.controller.ts", `
import { Controller, Patch } from '@nestjs/common';
@Controller('orders')
export class OrdersController {
  @Patch(':id')
  updateOrder() { return {}; }
}`)
	writeGroupFile(t, tmpDir, "routes/api.php", `<?php
Route::get('/users', [UserController::class, 'index']);
Route::post('/users', [UserController::class, 'store']);
Route::delete('/users/{id}', [UserController::class, 'destroy']);`)
	writeGroupFile(t, tmpDir, "app/main.py", `from fastapi import FastAPI
app = FastAPI()
@app.get("/users")
async def list_users():
    return []
@app.post("/users")
async def create_user(user):
    return user`)

	contracts, err := ExtractHTTPContractsFromSource(tmpDir)
	if err != nil {
		t.Fatalf("ExtractHTTPContractsFromSource() error = %v", err)
	}
	providers := filterContractsByRole(contracts, "provider")
	for _, want := range []string{
		"http::GET::/api/v2/users",
		"http::POST::/api/v2/users",
		"http::GET::/api/v2/users/{param}",
		"http::GET::/api/users",
		"http::POST::/api/users",
		"http::DELETE::/api/users/{param}",
		"http::POST::/api/orders/{param}",
		"http::GET::/api/orders",
		"http::GET::/api/health",
		"http::PATCH::/orders/{param}",
		"http::GET::/users",
		"http::POST::/users",
		"http::DELETE::/users/{param}",
	} {
		if findContract(providers, want) == nil {
			t.Fatalf("providers missing %s\nall=%v", want, contractIDs(providers))
		}
	}
	if got := findContract(providers, "http::POST::/api/orders/{param}"); got == nil || got.SymbolName != "createOrder" {
		t.Fatalf("gin route symbol = %#v", got)
	}
	if got := findContract(providers, "http::GET::/api/health"); got == nil || got.SymbolName != "healthHandler" {
		t.Fatalf("stdlib route symbol = %#v", got)
	}
}

func TestExtractHTTPConsumersFromSourceFrameworks(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "src/api/users.ts", `
export async function fetchUsers() {
  const res = await fetch('/api/users');
  return res.json();
}
export async function createUser(data: any) {
  const res = await fetch('/api/users', { method: 'POST', body: JSON.stringify(data) });
  return res.json();
}`)
	writeGroupFile(t, tmpDir, "src/api.ts",
		"import axios from 'axios';\n"+
			"export const getUsers = () => axios.get('/api/users');\n"+
			"export const deleteUser = (id: string) => axios.delete(\x60/api/users/${id}\x60);\n"+
			"function createOrder(data: unknown) {\n"+
			"  return axios({ method: 'POST', url: '/api/orders', data });\n"+
			"}\n"+
			"function updateUser(id: string) {\n"+
			"  return axios({ url: \x60/api/users/${id}\x60, method: 'PUT' });\n"+
			"}\n"+
			"function listDefaults() {\n"+
			"  return axios({ url: '/api/defaults' });\n"+
			"}")
	writeGroupFile(t, tmpDir, "public/js/orders.js",
		"$.get('/api/users', function (data) {});\n"+
			"$.post('/api/users', {});\n"+
			"$.ajax({ url: '/api/orders', method: 'PUT', data: {} });\n"+
			"$.ajax({ url: '/api/items',  type:   'DELETE' });\n"+
			"$.ajax({ url: '/api/default' });\n"+
			"function reloadOrder(id) {\n"+
			"  return $.ajax({ url: \x60/api/orders/${id}\x60, method: 'GET' });\n"+
			"}\n"+
			"$.fn.extend({ url: '/nope', method: 'POST' });\n"+
			"function myHelper(opts) { return opts; }\n"+
			"myHelper({ url: '/nope', method: 'POST' });")
	writeGroupFile(t, tmpDir, "src/client.py", `
import requests
def create_order():
    return requests.post("https://svc.local/api/orders/42", json={"id": 42})`)
	writeGroupFile(t, tmpDir, "src/ApiClient.java", `
class ApiClient {
  void run(RestTemplate restTemplate, WebClient webClient) {
    restTemplate.getForObject("/api/users/{id}", String.class, 42);
    webClient.method(HttpMethod.PATCH, "/api/users/42");
    new Request.Builder().url("/api/orders/42").build();
  }
}`)
	writeGroupFile(t, tmpDir, "cmd/client.go", `
package main
func main() {
  http.Get("/api/health")
  client := resty.New()
  client.R().Delete("/api/orders/42")
}`)

	contracts, err := ExtractHTTPContractsFromSource(tmpDir)
	if err != nil {
		t.Fatalf("ExtractHTTPContractsFromSource() error = %v", err)
	}
	consumers := filterContractsByRole(contracts, "consumer")
	for _, want := range []string{
		"http::GET::/api/users",
		"http::POST::/api/users",
		"http::DELETE::/api/users/{param}",
		"http::POST::/api/orders",
		"http::PUT::/api/users/{param}",
		"http::GET::/api/defaults",
		"http::PUT::/api/orders",
		"http::DELETE::/api/items",
		"http::GET::/api/default",
		"http::GET::/api/orders/{param}",
		"http::POST::/api/orders/{param}",
		"http::PATCH::/api/users/{param}",
		"http::GET::/api/health",
	} {
		if findContract(consumers, want) == nil {
			t.Fatalf("consumers missing %s\nall=%v", want, contractIDs(consumers))
		}
	}
	for _, contract := range consumers {
		if path, ok := contract.Meta["path"].(string); ok && path == "/nope" {
			t.Fatalf("negative control emitted /nope consumer: %#v", contract)
		}
	}
}

func TestHTTPGraphAssistedMultiVerbDisambiguation(t *testing.T) {
	detections := []httpDetection{
		{Role: "provider", Method: "GET", Path: "/api/orders", Name: "listOrders"},
		{Role: "provider", Method: "POST", Path: "/api/orders", Name: "createOrder"},
		{Role: "provider", Method: "PUT", Path: "/api/orders", Name: "replaceOrder"},
	}
	symbols := []graphSymbolRow{
		{UID: "uid-listOrders", Name: "listOrders", FilePath: "routes.ts", Labels: []string{"Function"}},
		{UID: "uid-createOrder", Name: "createOrder", FilePath: "routes.ts", Labels: []string{"Function"}},
		{UID: "uid-replaceOrder", Name: "replaceOrder", FilePath: "routes.ts", Labels: []string{"Function"}},
	}

	post := httpProviderContractFromGraphRow(httpGraphRouteRow{
		FileID:      "f1",
		FilePath:    "routes.ts",
		RoutePath:   "/api/orders",
		RouteSource: "decorator-Post",
	}, detections, symbols)
	if post.SymbolName != "createOrder" || post.Meta["method"] != "POST" || post.ContractID != "http::POST::/api/orders" {
		t.Fatalf("post graph contract = %#v", post)
	}

	get := httpProviderContractFromGraphRow(httpGraphRouteRow{
		FileID:      "f1",
		FilePath:    "routes.ts",
		RoutePath:   "/api/orders",
		RouteSource: "decorator-Get",
	}, detections, symbols)
	if get.SymbolName != "listOrders" || get.Meta["method"] != "GET" {
		t.Fatalf("get graph contract = %#v", get)
	}

	put := httpProviderContractFromGraphRow(httpGraphRouteRow{
		FileID:      "f1",
		FilePath:    "routes.ts",
		RoutePath:   "/api/orders",
		RouteSource: "decorator-Put",
	}, detections, symbols)
	if put.SymbolName != "replaceOrder" || put.Meta["method"] != "PUT" {
		t.Fatalf("put graph contract = %#v", put)
	}

	ambiguous := httpProviderContractFromGraphRow(httpGraphRouteRow{
		FileID:      "f1",
		FilePath:    "routes.ts",
		RoutePath:   "/api/orders",
		RouteSource: "unknown-reason",
	}, detections[:2], symbols[:2])
	if ambiguous.SymbolName == "listOrders" || ambiguous.SymbolName == "createOrder" || ambiguous.SymbolUID != "" || ambiguous.Meta["method"] != "GET" {
		t.Fatalf("ambiguous provider should not inherit a handler: %#v", ambiguous)
	}

	consumerAmbiguous := httpConsumerContractFromGraphRow(httpGraphFetchRow{
		FileID:      "f1",
		FilePath:    "client.ts",
		RoutePath:   "/api/orders",
		FetchReason: "fetch",
	}, []httpDetection{
		{Role: "consumer", Method: "GET", Path: "/api/orders"},
		{Role: "consumer", Method: "POST", Path: "/api/orders"},
	}, nil)
	if consumerAmbiguous.Meta["method"] != "GET" || consumerAmbiguous.ContractID != "http::GET::/api/orders" {
		t.Fatalf("ambiguous consumer should default to GET: %#v", consumerAmbiguous)
	}
}

func TestExtractHTTPContractsEmptyRepoAndPathNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "README.md", "# Hello")
	contracts, err := ExtractHTTPContractsFromSource(tmpDir)
	if err != nil {
		t.Fatalf("ExtractHTTPContractsFromSource(empty) error = %v", err)
	}
	if len(contracts) != 0 {
		t.Fatalf("empty repo contracts = %#v", contracts)
	}

	writeGroupFile(t, tmpDir, "src/router.ts", `
router.get('/api/users/', handler);
router.get('/api/users/:id', handler1);
router.get('/api/posts/{postId}', handler2);
`)
	contracts, err = ExtractHTTPContractsFromSource(tmpDir)
	if err != nil {
		t.Fatalf("ExtractHTTPContractsFromSource(paths) error = %v", err)
	}
	if findContract(contracts, "http::GET::/api/users") == nil {
		t.Fatalf("trailing slash was not stripped: %v", contractIDs(contracts))
	}
	if findContract(contracts, "http::GET::/api/users/{param}") == nil || findContract(contracts, "http::GET::/api/posts/{param}") == nil {
		t.Fatalf("path params were not normalized: %v", contractIDs(contracts))
	}
}

func filterContractsByRole(contracts []StoredContract, role string) []StoredContract {
	out := make([]StoredContract, 0)
	for _, contract := range contracts {
		if contract.Role == role {
			out = append(out, contract)
		}
	}
	return out
}

func findContract(contracts []StoredContract, contractID string) *StoredContract {
	for i := range contracts {
		if contracts[i].ContractID == contractID {
			return &contracts[i]
		}
	}
	return nil
}

func contractIDs(contracts []StoredContract) []string {
	ids := make([]string, 0, len(contracts))
	for _, contract := range contracts {
		ids = append(ids, contract.ContractID)
	}
	sort.Strings(ids)
	return ids
}
