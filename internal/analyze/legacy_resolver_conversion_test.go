package analyze

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestRunLegacyExpressRouteMappingConversion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "server.ts", `import express from 'express';
const app = express();

app.get('/api/users', (req, res) => {
  res.json({ users: [] });
});

app.post('/api/users', (req, res) => {
  res.json({ id: 1, created: true });
});

app.put('/api/users/:id', (req, res) => {
  res.json({ updated: true });
});

app.delete('/api/users/:id', (req, res) => {
  res.json({ deleted: true });
});

const router = express.Router();
router.get('/api/health', (req, res) => {
  res.json({ status: 'ok' });
});
`)
	writeFile(t, dir, "app.js", `const app = require('express')();

app.get('/api/items', (req, res) => {
  res.json({ items: [] });
});

app.post('/api/items', (req, res) => {
  res.json({ created: true });
});
`)

	result, err := Run(context.Background(), dir, Options{
		Parser:   parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner: &recordingDBRunner{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Metrics.Routes.RoutesEmitted != 4 || result.Metrics.Routes.HandlesEmitted != 4 || result.Metrics.Routes.Duplicates != 3 {
		t.Fatalf("route metrics = %#v", result.Metrics.Routes)
	}

	for _, route := range []string{"/api/users", "/api/users/:id", "/api/health", "/api/items"} {
		requireLegacyGraphNode(t, result.Graph, graph.GenerateID(string(scopeir.NodeRoute), route), scopeir.NodeRoute)
	}
	requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "server.ts"), graph.GenerateID(string(scopeir.NodeRoute), "/api/users"), "framework-route")
	requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "server.ts"), graph.GenerateID(string(scopeir.NodeRoute), "/api/health"), "framework-route")
	requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "app.js"), graph.GenerateID(string(scopeir.NodeRoute), "/api/items"), "framework-route")
}

func TestRunLegacyPythonMCPToolsConversion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "server.py", `from mcp import tool

@mcp.tool()
def get_weather(city: str) -> str:
    """Get weather for a city."""
    return f"Weather in {city}: sunny"

@mcp.tool()
def search_docs(query: str) -> list:
    """Search documentation."""
    return []
`)

	result, err := Run(context.Background(), dir, Options{
		Parser:   parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner: &recordingDBRunner{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Metrics.Tools.ToolsEmitted != 2 || result.Metrics.Tools.HandlesEmitted != 2 {
		t.Fatalf("tool metrics = %#v", result.Metrics.Tools)
	}

	for _, tool := range []string{"get_weather", "search_docs"} {
		toolID := graph.GenerateID(string(scopeir.NodeTool), tool)
		requireLegacyGraphNode(t, result.Graph, toolID, scopeir.NodeTool)
		requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesTool, graph.GenerateID(string(scopeir.NodeFile), "server.py"), toolID, "tool-definition")
	}
}

func TestRunLegacyAPIDeepFlowConversion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app/api/grants/route.ts", `import { NextResponse } from 'next/server';
export async function GET() {
  try {
    return NextResponse.json({ data: [], pagination: { page: 1 } });
  } catch (err) {
    return NextResponse.json({ error: 'Failed', message: String(err) }, { status: 400 });
  }
}
`)
	writeFile(t, dir, "app/api/secure/route.ts", `import { NextResponse } from 'next/server';
export const GET = withAuth(withRateLimit(async (req: Request) => {
  return NextResponse.json({ items: [], count: 0 });
}));
`)
	writeFile(t, dir, "components/GrantsList.tsx", `export async function GrantsList() {
  const res = await fetch('/api/grants');
  const { data, pagination } = await res.json();
  return data.map((g: any) => pagination.page);
}
`)
	writeFile(t, dir, "hooks/useGrants.ts", `export async function useGrants() {
  const result = await fetch('/api/grants');
  const data = await result.json();
  return data.items;
}
`)
	writeFile(t, dir, "hooks/useMulti.ts", `export async function useMulti() {
  const [grantsRes, secureRes] = await Promise.all([fetch('/api/grants'), fetch('/api/secure')]);
  const grants = await grantsRes.json();
  const secure = await secureRes.json();
  return { grants: grants.data, secure: secure.items, meta: grants.meta };
}
`)

	result, err := Run(context.Background(), dir, Options{
		Parser:   parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner: &recordingDBRunner{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	requireLegacyRouteProperties(t, result.Graph, "/api/grants", "responseKeys", []string{"data", "pagination"})
	requireLegacyRouteProperties(t, result.Graph, "/api/grants", "errorKeys", []string{"error", "message"})
	requireLegacyRouteProperties(t, result.Graph, "/api/secure", "responseKeys", []string{"count", "items"})
	requireLegacyRouteProperties(t, result.Graph, "/api/secure", "middleware", []string{"withAuth", "withRateLimit"})
	requireLegacyFetchReasonContains(t, result.Graph, "components/GrantsList.tsx", "/api/grants", []string{"keys:data,pagination"})
	requireLegacyFetchReasonContains(t, result.Graph, "hooks/useGrants.ts", "/api/grants", []string{"keys:items"})
	requireLegacyFetchReasonContains(t, result.Graph, "hooks/useMulti.ts", "/api/grants", []string{"keys:data,items,meta", "fetches:2"})
	requireLegacyFetchReasonContains(t, result.Graph, "hooks/useMulti.ts", "/api/secure", []string{"keys:data,items,meta", "fetches:2"})
}

func TestRunLegacyPHPResponseShapeConversion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/items.php", `<?php
if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
    http_response_code(405);
    echo json_encode(['error' => 'Method not allowed']);
    exit;
}
$items = [];
echo json_encode(['data' => $items, 'total' => count($items)]);
`)
	writeFile(t, dir, "api/submit.php", `<?php
if (empty($data['name'])) {
    http_response_code(400);
    echo json_encode(['error' => 'Validation failed', 'field' => 'name']);
    exit(1);
}
echo json_encode(['ok' => true, 'id' => $id, 'created_at' => date('c')]);
`)

	result, err := Run(context.Background(), dir, Options{
		Parser:   parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner: &recordingDBRunner{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	requireLegacyRouteProperties(t, result.Graph, "/api/items", "responseKeys", []string{"data", "total"})
	requireLegacyRouteProperties(t, result.Graph, "/api/items", "errorKeys", []string{"error"})
	requireLegacyRouteProperties(t, result.Graph, "/api/submit", "responseKeys", []string{"created_at", "id", "ok"})
	requireLegacyRouteProperties(t, result.Graph, "/api/submit", "errorKeys", []string{"error", "field"})
	requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "api/items.php"), graph.GenerateID(string(scopeir.NodeRoute), "/api/items"), "php-api-file-route")
	requireLegacyGraphRelationship(t, result.Graph, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "api/submit.php"), graph.GenerateID(string(scopeir.NodeRoute), "/api/submit"), "php-api-file-route")
}

func TestRunLegacyRouteMappingConversion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "middleware.ts", `export function middleware(request) { return NextResponse.next(); }
export const config = { matcher: ['/api/:path*'] };
`)
	writeFile(t, dir, "app/api/grants/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeFile(t, dir, "app/api/organizations/[slug]/grants/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeFile(t, dir, "components/GrantsList.tsx", "export function GrantsList({ slug }: { slug: string }) { return fetch(`/api/organizations/${slug}/grants`).then(r => r.json()) }\n")
	writeFile(t, dir, "hooks/useGrants.ts", "export function useGrants() { return fetch('/api/grants').then(r => r.json()) }\n")
	writeFile(t, dir, "api/upload.php", "<?php echo json_encode(['ok' => true]);\n")
	writeFile(t, dir, "api/status.php", "<?php echo json_encode(['status' => 'ok']);\n")

	result, err := Run(context.Background(), dir, Options{
		Parser:   parser.PoolOptions{ParseTimeout: time.Second},
		DBRunner: &recordingDBRunner{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, route := range []string{"/api/grants", "/api/organizations/:slug/grants", "/api/upload", "/api/status"} {
		requireLegacyGraphNode(t, result.Graph, graph.GenerateID(string(scopeir.NodeRoute), route), scopeir.NodeRoute)
		requireLegacyRouteProperties(t, result.Graph, route, "middleware", []string{"middleware"})
	}
	requireLegacyGraphRelationship(t, result.Graph, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "components/GrantsList.tsx"), graph.GenerateID(string(scopeir.NodeRoute), "/api/organizations/:slug/grants"), "fetch-route")
	requireLegacyGraphRelationship(t, result.Graph, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "hooks/useGrants.ts"), graph.GenerateID(string(scopeir.NodeRoute), "/api/grants"), "fetch-route")
}

func requireLegacyGraphNode(t *testing.T, g *graph.Graph, id string, label scopeir.NodeLabel) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing graph node %s", id)
	}
	if node.Label != label {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, label)
	}
}

func requireLegacyGraphRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, reason string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID && rel.Reason == reason {
			return
		}
	}
	t.Fatalf("missing %s %s -> %s reason %s", relType, sourceID, targetID, reason)
}

func requireLegacyRouteProperties(t *testing.T, g *graph.Graph, route string, key string, want []string) {
	t.Helper()
	node, ok := g.GetNode(graph.GenerateID(string(scopeir.NodeRoute), route))
	if !ok {
		t.Fatalf("missing route %s", route)
	}
	got, ok := node.Properties[key].([]string)
	if !ok {
		t.Fatalf("route %s property %s = %#v, want []string", route, key, node.Properties[key])
	}
	if len(got) != len(want) {
		t.Fatalf("route %s property %s = %#v, want %#v", route, key, got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("route %s property %s = %#v, want %#v", route, key, got, want)
		}
	}
}

func requireLegacyFetchReasonContains(t *testing.T, g *graph.Graph, sourcePath string, route string, wantParts []string) {
	t.Helper()
	sourceID := graph.GenerateID(string(scopeir.NodeFile), sourcePath)
	targetID := graph.GenerateID(string(scopeir.NodeRoute), route)
	for _, rel := range g.Relationships {
		if rel.Type != graph.RelFetches || rel.SourceID != sourceID || rel.TargetID != targetID {
			continue
		}
		for _, want := range wantParts {
			if !strings.Contains(rel.Reason, want) {
				t.Fatalf("FETCHES reason = %q, missing %q", rel.Reason, want)
			}
		}
		return
	}
	t.Fatalf("missing FETCHES %s -> %s", sourcePath, route)
}
