package routes

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/graph"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
	"github.com/tamnguyendinh/avmatrix-go/internal/structure"
)

func TestApplyEmitsFilesystemFrameworkAndFetchRouteEdges(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "app/users/page.tsx", "export default function Page() { return null }\n")
	writeRouteTestFile(t, dir, "app/api/orders/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeRouteTestFile(t, dir, "src/server.ts", "app.get('/health', handler)\n")
	writeRouteTestFile(t, dir, "src/client.ts", "export function load() { return fetch('/api/orders?active=1') }\n")

	files := []scanner.File{
		{Path: "app/users/page.tsx", Language: scanner.TypeScript},
		{Path: "app/api/orders/route.ts", Language: scanner.TypeScript},
		{Path: "src/server.ts", Language: scanner.TypeScript},
		{Path: "src/client.ts", Language: scanner.TypeScript},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 3 || result.Metrics.HandlesEmitted != 3 || result.Metrics.FetchesEmitted != 1 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}

	usersRoute := graph.GenerateID(string(scopeir.NodeRoute), "/users")
	ordersRoute := graph.GenerateID(string(scopeir.NodeRoute), "/api/orders")
	healthRoute := graph.GenerateID(string(scopeir.NodeRoute), "/health")
	requireRouteNode(t, g, usersRoute)
	requireRouteNode(t, g, ordersRoute)
	requireRouteNode(t, g, healthRoute)
	requireRouteRelationship(t, g, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "app/users/page.tsx"), usersRoute, "nextjs-filesystem-route")
	requireRouteRelationship(t, g, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "src/server.ts"), healthRoute, "framework-route")
	requireRouteRelationship(t, g, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "src/client.ts"), ordersRoute, "fetch-route")
}

func TestApplySkipsDuplicateRoutesAndExternalFetches(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "app/users/page.tsx", "fetch('https://example.com/users')\n")
	writeRouteTestFile(t, dir, "src/server.ts", "router.get('/users', handler)\n")

	files := []scanner.File{
		{Path: "app/users/page.tsx", Language: scanner.TypeScript},
		{Path: "src/server.ts", Language: scanner.TypeScript},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 1 || result.Metrics.Duplicates != 1 || result.Metrics.FetchesEmitted != 0 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
}

func TestNextRouteURLCoversAppPagesDynamicAndWindowsPaths(t *testing.T) {
	tests := []struct {
		path       string
		wantURL    string
		wantSource string
	}{
		{path: "app/api/grants/route.ts", wantURL: "/api/grants", wantSource: "nextjs-filesystem-route"},
		{path: "app/api/auth/login/route.tsx", wantURL: "/api/auth/login", wantSource: "nextjs-filesystem-route"},
		{path: "app/(admin)/api/users/route.ts", wantURL: "/api/users", wantSource: "nextjs-filesystem-route"},
		{path: "app/api/users/[id]/route.ts", wantURL: "/api/users/:id", wantSource: "nextjs-filesystem-route"},
		{path: "app/api/docs/[...slug]/route.ts", wantURL: "/api/docs/*slug", wantSource: "nextjs-filesystem-route"},
		{path: "app/api/docs/[[...slug]]/route.ts", wantURL: "/api/docs/*slug", wantSource: "nextjs-filesystem-route"},
		{path: "pages/api/auth/login.ts", wantURL: "/api/auth/login", wantSource: "nextjs-pages-route"},
		{path: "pages/api/index.ts", wantURL: "/api", wantSource: "nextjs-pages-route"},
		{path: "app\\api\\grants\\route.ts", wantURL: "/api/grants", wantSource: "nextjs-filesystem-route"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotURL, gotSource := nextRouteURL(tt.path)
			if gotURL != tt.wantURL || gotSource != tt.wantSource {
				t.Fatalf("nextRouteURL(%q) = (%q, %q), want (%q, %q)", tt.path, gotURL, gotSource, tt.wantURL, tt.wantSource)
			}
		})
	}
}

func TestPHPRouteURLCoversLegacyAPIRoutePatterns(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "api/upload.php", want: "/api/upload"},
		{path: "api/next_sign.php", want: "/api/next_sign"},
		{path: "api/v2/users.php", want: "/api/v2/users"},
		{path: "api/contest.php", want: "/api/contest"},
		{path: "api/attestation.php", want: "/api/attestation"},
		{path: "api/latest.php", want: "/api/latest"},
		{path: "api/base64_encode.php", want: "/api/base64_encode"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, source := phpRouteURL(tt.path)
			if got != tt.want || source != "php-api-file-route" {
				t.Fatalf("phpRouteURL(%q) = (%q, %q), want (%q, php-api-file-route)", tt.path, got, source, tt.want)
			}
		})
	}
}

func TestPHPRouteURLSkipsNonHandlers(t *testing.T) {
	for _, path := range []string{
		"index.php",
		"includes/database.php",
		"vendor/lib/api/config.php",
		"api/_helpers.php",
		"api/helper_utils.php",
		"api/test_upload.php",
		"api/fixture_data.php",
		"api/readme.md",
	} {
		if got, source := phpRouteURL(path); got != "" || source != "" {
			t.Fatalf("phpRouteURL(%q) = (%q, %q), want empty", path, got, source)
		}
	}
}

func TestNextRouteURLSkipsNonRouteFiles(t *testing.T) {
	for _, path := range []string{
		"src/components/Button.tsx",
		"src/lib/utils.ts",
		"app/api/users/handler.txt",
	} {
		if gotURL, gotSource := nextRouteURL(path); gotURL != "" || gotSource != "" {
			t.Fatalf("nextRouteURL(%q) = (%q, %q), want empty", path, gotURL, gotSource)
		}
	}
}

func TestNormalizeRouteURLAdditionalPatterns(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "/api/grants", want: "/api/grants"},
		{raw: "/api/users/123", want: "/api/users/123"},
		{raw: "/api/grants?page=1&limit=10", want: "/api/grants"},
		{raw: "/api/grants#summary", want: "/api/grants"},
		{raw: "/v1/users", want: "/v1/users"},
		{raw: "/graphql", want: "/graphql"},
		{raw: "api/relative", want: "/api/relative"},
		{raw: "`/api/grants`", want: "/api/grants"},
		{raw: "/api/organizations/${slug}/grants", want: "/api/organizations/[param]/grants"},
		{raw: "/api/trailing/", want: "/api/trailing"},
		{raw: "/api/+endpoint", want: ""},
		{raw: "getApiUrl()", want: ""},
		{raw: "https://example.com/api/users", want: ""},
		{raw: "http://example.com/api/users", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := normalizeRouteURL(tt.raw); got != tt.want {
				t.Fatalf("normalizeRouteURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRouteMatchesDynamicSegmentsAndCatchAll(t *testing.T) {
	tests := []struct {
		candidate string
		route     string
		want      bool
	}{
		{candidate: "/api/grants", route: "/api/grants", want: true},
		{candidate: "/api/grants", route: "/api/users"},
		{candidate: "/api/grants/123", route: "/api/grants"},
		{candidate: "/api/orgs/[param]", route: "/api/orgs/:slug", want: true},
		{candidate: "/api/orgs/acme", route: "/api/orgs/:slug", want: true},
		{candidate: "/api/orgs/[param]/grants", route: "/api/orgs/:slug/grants", want: true},
		{candidate: "/api/docs/a/b/c", route: "/api/docs/*slug", want: true},
		{candidate: "/v1/docs/a", route: "/api/docs/*slug"},
	}
	for _, tt := range tests {
		t.Run(tt.candidate+"->"+tt.route, func(t *testing.T) {
			if got := routeMatches(tt.candidate, tt.route); got != tt.want {
				t.Fatalf("routeMatches(%q, %q) = %v, want %v", tt.candidate, tt.route, got, tt.want)
			}
		})
	}
}

func TestFetchCallsExtractsFetchRouterAndHrefNavigations(t *testing.T) {
	content := `
		export function Page() {
			fetch('/api/orders?active=1')
			router.push('/dashboard')
			router.replace('/settings#profile')
			return <Link href="/profile">Profile</Link>
		}
	`
	calls := fetchCalls("app/page.tsx", content)
	got := make([]string, 0, len(calls))
	for _, call := range calls {
		got = append(got, call.URL)
		if call.FilePath != "app/page.tsx" {
			t.Fatalf("call file path = %q", call.FilePath)
		}
	}
	want := []string{"/api/orders", "/dashboard", "/settings", "/profile"}
	if len(got) != len(want) {
		t.Fatalf("fetchCalls URLs = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fetchCalls URLs = %#v, want %#v", got, want)
		}
	}
}

func TestFetchCallsExtractsKeysAndMultiFetchCount(t *testing.T) {
	content := `
		export async function useMulti() {
			const [grantsRes, secureRes] = await Promise.all([
				fetch('/api/grants'),
				fetch('/api/secure'),
			]);
			const grants = await grantsRes.json();
			const secure = await secureRes.json();
			return { grants: grants.data, secure: secure.items, meta: grants.meta };
		}
	`
	calls := fetchCalls("hooks/useMulti.ts", content)
	if len(calls) != 2 {
		t.Fatalf("fetchCalls count = %d, want 2", len(calls))
	}
	for _, call := range calls {
		if call.FetchCount != 2 {
			t.Fatalf("fetch count = %d, want 2", call.FetchCount)
		}
		for _, want := range []string{"data", "items", "meta"} {
			if !containsString(call.AccessedKeys, want) {
				t.Fatalf("accessed keys = %#v, missing %s", call.AccessedKeys, want)
			}
		}
	}
}

func TestExtractResponseShapesCoversQuotedKeysAndStatusSeparation(t *testing.T) {
	shapes := extractResponseShapes(`
		if (bad) return NextResponse.json({ error: 'fail', details: 'x' }, { status: 400 });
		return NextResponse.json({ 'courses': coursesData, "articles": articlesData, pagination: { page: 1 } });
	`)
	if !reflect.DeepEqual(shapes.ResponseKeys, []string{"articles", "courses", "pagination"}) {
		t.Fatalf("response keys = %#v", shapes.ResponseKeys)
	}
	if !reflect.DeepEqual(shapes.ErrorKeys, []string{"details", "error"}) {
		t.Fatalf("error keys = %#v", shapes.ErrorKeys)
	}
	for _, key := range shapes.ResponseKeys {
		if strings.ContainsAny(key, `"'`) {
			t.Fatalf("response key still quoted: %q", key)
		}
	}
}

func TestExtractPHPResponseShapesCoversStatusAndExitBoundaries(t *testing.T) {
	shapes := extractPHPResponseShapes(`
		if (!$user) {
			http_response_code(401);
			echo json_encode(['error' => 'Unauthorized', 'code' => 'AUTH_REQUIRED']);
			exit;
		}
		echo json_encode(['data' => $items, 'total' => count($items)]);
		header('Status: 500 Server Error');
		echo json_encode(array('message' => 'failed'));
	`)
	if !reflect.DeepEqual(shapes.ResponseKeys, []string{"data", "total"}) {
		t.Fatalf("response keys = %#v", shapes.ResponseKeys)
	}
	if !reflect.DeepEqual(shapes.ErrorKeys, []string{"code", "error", "message"}) {
		t.Fatalf("error keys = %#v", shapes.ErrorKeys)
	}
}

func TestExtractConsumerAccessedKeysFiltersDOMButKeepsAPIFields(t *testing.T) {
	content := `
		const { url } = await response.json();
		const data = await res.json();
		console.log(data.type);
		console.log(data.href);
		console.log(data.target);
		console.log(data.style);
		const link = document.createElement('a');
		link.href = url;
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
	`
	keys := extractConsumerAccessedKeys(content)
	for _, want := range []string{"url", "type", "href", "target", "style"} {
		if !containsString(keys, want) {
			t.Fatalf("consumer keys = %#v, missing %s", keys, want)
		}
	}
	for _, blocked := range []string{"appendChild", "removeChild", "createElement", "click"} {
		if containsString(keys, blocked) {
			t.Fatalf("consumer keys = %#v, unexpectedly contains %s", keys, blocked)
		}
	}
}

func TestExtractMiddlewareChainAndProjectMatcher(t *testing.T) {
	chain := extractMiddlewareChain(`export const POST = withRateLimit(withCSRF(withAuth(async (req) => { return NextResponse.json({ ok: true }); })));`)
	if !reflect.DeepEqual(chain, []string{"withAuth", "withCSRF", "withRateLimit"}) {
		t.Fatalf("middleware chain = %#v", chain)
	}
	config, ok := extractProjectMiddleware("middleware.ts", `
		export function middleware(request) { return NextResponse.next(); }
		export const config = { matcher: ['/api/:path*'] };
	`)
	if !ok || !reflect.DeepEqual(config.Names, []string{"middleware"}) || !reflect.DeepEqual(config.Matchers, []string{"/api/:path*"}) {
		t.Fatalf("project middleware config = %#v, ok=%v", config, ok)
	}
	if !middlewareMatcherMatchesRoute("/api/:path*", "/api/users") || middlewareMatcherMatchesRoute("/dashboard/:path*", "/api/users") {
		t.Fatalf("middleware matcher result was incorrect")
	}
}

func TestExpoFilesystemRouteConventions(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"app/index.tsx", "/"},
		{"app/settings.tsx", "/settings"},
		{"app/(tabs)/index.tsx", "/"},
		{"app/(tabs)/settings.tsx", "/settings"},
		{"app/(auth)/login.tsx", "/login"},
		{"app/(tabs)/(home)/feed.tsx", "/feed"},
		{"app/users+api.ts", "/users"},
		{"app/user/[id]+api.ts", "/user/[id]"},
		{"app/user/[id].tsx", "/user/[id]"},
		{"app/posts/[...slug].tsx", "/posts/[...slug]"},
		{"app\\(tabs)\\settings.tsx", "/settings"},
	}
	for _, tt := range tests {
		routes := fileSystemRoutes(tt.path)
		if len(routes) != 1 || routes[0].URL != tt.want || routes[0].Source != "expo-filesystem-route" {
			t.Fatalf("fileSystemRoutes(%q) = %#v, want one Expo route %q", tt.path, routes, tt.want)
		}
	}
}

func TestExpoFilesystemRoutesSkipLayoutsSpecialFilesAndNonAppPaths(t *testing.T) {
	tests := []string{
		"app/_layout.tsx",
		"app/(tabs)/_layout.tsx",
		"app/+not-found.tsx",
		"app/types.d.ts",
		"src/utils/helper.ts",
	}
	for _, filePath := range tests {
		if routes := fileSystemRoutes(filePath); len(routes) != 0 {
			t.Fatalf("fileSystemRoutes(%q) = %#v, want none", filePath, routes)
		}
	}
}

func TestApplyEmitsExpoRouteAndNavigationEdges(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "app/index.tsx", "export default function Home() { return null }\n")
	writeRouteTestFile(t, dir, "app/(auth)/login.tsx", "import { useRouter } from 'expo-router';\nexport default function Login() { const router = useRouter(); router.push('/'); return null }\n")
	writeRouteTestFile(t, dir, "app/settings.tsx", "import { Link } from 'expo-router';\nexport default function Settings() { return <Link href=\"/profile\">Go</Link> }\n")
	writeRouteTestFile(t, dir, "app/profile.tsx", "export default function Profile() { return null }\n")
	writeRouteTestFile(t, dir, "app/user/[id].tsx", "import { useLocalSearchParams } from 'expo-router';\nexport default function UserDetail() { return null }\n")
	writeRouteTestFile(t, dir, "app/_layout.tsx", "import { Slot } from 'expo-router';\nexport default function Layout() { return <Slot /> }\n")

	files := []scanner.File{
		{Path: "app/index.tsx", Language: scanner.TypeScript},
		{Path: "app/(auth)/login.tsx", Language: scanner.TypeScript},
		{Path: "app/settings.tsx", Language: scanner.TypeScript},
		{Path: "app/profile.tsx", Language: scanner.TypeScript},
		{Path: "app/user/[id].tsx", Language: scanner.TypeScript},
		{Path: "app/_layout.tsx", Language: scanner.TypeScript},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 5 || result.Metrics.FetchesEmitted != 2 {
		t.Fatalf("metrics = %#v, want five Expo routes and two navigation edges", result.Metrics)
	}

	for _, route := range []string{"/", "/login", "/settings", "/profile", "/user/[id]"} {
		requireRouteNode(t, g, graph.GenerateID(string(scopeir.NodeRoute), route))
	}
	if _, ok := g.GetNode(graph.GenerateID(string(scopeir.NodeRoute), "/_layout")); ok {
		t.Fatal("layout route was emitted")
	}
	requireRouteRelationship(t, g, graph.RelHandlesRoute, graph.GenerateID(string(scopeir.NodeFile), "app/(auth)/login.tsx"), graph.GenerateID(string(scopeir.NodeRoute), "/login"), "expo-filesystem-route")
	requireRouteRelationship(t, g, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "app/(auth)/login.tsx"), graph.GenerateID(string(scopeir.NodeRoute), "/"), "fetch-route")
	requireRouteRelationship(t, g, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "app/settings.tsx"), graph.GenerateID(string(scopeir.NodeRoute), "/profile"), "fetch-route")
}

func TestApplyEmitsLegacyDeepFlowRouteShapeMetadata(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "app/api/grants/route.ts", `import { NextResponse } from 'next/server';
export async function GET() {
  try {
    return NextResponse.json({ data: [], pagination: { page: 1 } });
  } catch (err) {
    return NextResponse.json({ error: 'Failed', message: String(err) }, { status: 400 });
  }
}
`)
	writeRouteTestFile(t, dir, "app/api/secure/route.ts", `import { NextResponse } from 'next/server';
export const GET = withAuth(withRateLimit(async (req: Request) => {
  return NextResponse.json({ items: [], count: 0 });
}));
`)
	writeRouteTestFile(t, dir, "components/GrantsList.tsx", `export async function GrantsList() {
  const res = await fetch('/api/grants');
  const { data, pagination } = await res.json();
  return data.map((g: any) => pagination.page);
}
`)
	writeRouteTestFile(t, dir, "hooks/useMulti.ts", `export async function useMulti() {
  const [grantsRes, secureRes] = await Promise.all([fetch('/api/grants'), fetch('/api/secure')]);
  const grants = await grantsRes.json();
  const secure = await secureRes.json();
  return { grants: grants.data, secure: secure.items, meta: grants.meta };
}
`)
	files := []scanner.File{
		{Path: "app/api/grants/route.ts", Language: scanner.TypeScript},
		{Path: "app/api/secure/route.ts", Language: scanner.TypeScript},
		{Path: "components/GrantsList.tsx", Language: scanner.TypeScript},
		{Path: "hooks/useMulti.ts", Language: scanner.TypeScript},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 2 || result.Metrics.FetchesEmitted != 3 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	grants := requireRouteNodeProperties(t, g, "/api/grants")
	requireStringPropertyValues(t, grants, "responseKeys", []string{"data", "pagination"})
	requireStringPropertyValues(t, grants, "errorKeys", []string{"error", "message"})
	secure := requireRouteNodeProperties(t, g, "/api/secure")
	requireStringPropertyValues(t, secure, "responseKeys", []string{"count", "items"})
	requireStringPropertyValues(t, secure, "middleware", []string{"withAuth", "withRateLimit"})
	requireRouteRelationshipReasonContains(t, g, "components/GrantsList.tsx", "/api/grants", []string{"keys:data,pagination"})
	requireRouteRelationshipReasonContains(t, g, "hooks/useMulti.ts", "/api/grants", []string{"keys:data,items,meta", "fetches:2"})
	requireRouteRelationshipReasonContains(t, g, "hooks/useMulti.ts", "/api/secure", []string{"keys:data,items,meta", "fetches:2"})
}

func TestApplyEmitsPHPResponseShapeRoutes(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "api/items.php", `<?php
if (!is_logged_in()) {
    http_response_code(401);
    echo json_encode(['error' => 'Unauthorized'], JSON_UNESCAPED_UNICODE);
    exit;
}
echo json_encode(['data' => $items, 'total' => count($items)], JSON_UNESCAPED_UNICODE);
`)
	writeRouteTestFile(t, dir, "api/submit.php", `<?php
if (empty($data['name'])) {
    http_response_code(400);
    echo json_encode(['error' => 'Validation failed', 'field' => 'name']);
    exit(1);
}
echo json_encode(['ok' => true, 'id' => $id, 'created_at' => date('c')]);
`)
	files := []scanner.File{
		{Path: "api/items.php", Language: scanner.PHP},
		{Path: "api/submit.php", Language: scanner.PHP},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 2 {
		t.Fatalf("routes emitted = %d, want 2", result.Metrics.RoutesEmitted)
	}
	items := requireRouteNodeProperties(t, g, "/api/items")
	requireStringPropertyValues(t, items, "responseKeys", []string{"data", "total"})
	requireStringPropertyValues(t, items, "errorKeys", []string{"error"})
	submit := requireRouteNodeProperties(t, g, "/api/submit")
	requireStringPropertyValues(t, submit, "responseKeys", []string{"created_at", "id", "ok"})
	requireStringPropertyValues(t, submit, "errorKeys", []string{"error", "field"})
}

func TestApplyEmitsRouteMappingDynamicPHPAndProjectMiddleware(t *testing.T) {
	dir := t.TempDir()
	writeRouteTestFile(t, dir, "middleware.ts", `export function middleware(request) { return NextResponse.next(); }
export const config = { matcher: ['/api/:path*'] };
`)
	writeRouteTestFile(t, dir, "app/api/grants/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeRouteTestFile(t, dir, "app/api/organizations/[slug]/grants/route.ts", "export async function GET() { return Response.json({ ok: true }) }\n")
	writeRouteTestFile(t, dir, "components/GrantsList.tsx", "export function GrantsList({ slug }: { slug: string }) { return fetch(`/api/organizations/${slug}/grants`).then(r => r.json()) }\n")
	writeRouteTestFile(t, dir, "hooks/useGrants.ts", "export function useGrants() { return fetch('/api/grants').then(r => r.json()) }\n")
	writeRouteTestFile(t, dir, "api/upload.php", "<?php echo json_encode(['ok' => true]);\n")
	writeRouteTestFile(t, dir, "api/status.php", "<?php echo json_encode(['status' => 'ok']);\n")
	files := []scanner.File{
		{Path: "middleware.ts", Language: scanner.TypeScript},
		{Path: "app/api/grants/route.ts", Language: scanner.TypeScript},
		{Path: "app/api/organizations/[slug]/grants/route.ts", Language: scanner.TypeScript},
		{Path: "components/GrantsList.tsx", Language: scanner.TypeScript},
		{Path: "hooks/useGrants.ts", Language: scanner.TypeScript},
		{Path: "api/upload.php", Language: scanner.PHP},
		{Path: "api/status.php", Language: scanner.PHP},
	}
	g := graph.New()
	structure.Apply(g, files)

	result, err := Apply(g, dir, files)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Metrics.RoutesEmitted != 4 || result.Metrics.FetchesEmitted != 2 {
		t.Fatalf("metrics = %#v", result.Metrics)
	}
	for _, route := range []string{"/api/grants", "/api/organizations/:slug/grants", "/api/upload", "/api/status"} {
		props := requireRouteNodeProperties(t, g, route)
		requireStringPropertyValues(t, props, "middleware", []string{"middleware"})
	}
	requireRouteRelationship(t, g, graph.RelFetches, graph.GenerateID(string(scopeir.NodeFile), "components/GrantsList.tsx"), graph.GenerateID(string(scopeir.NodeRoute), "/api/organizations/:slug/grants"), "fetch-route")
}

func writeRouteTestFile(t *testing.T, root string, rel string, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requireRouteNode(t *testing.T, g *graph.Graph, id string) {
	t.Helper()
	node, ok := g.GetNode(id)
	if !ok {
		t.Fatalf("missing node %s", id)
	}
	if node.Label != scopeir.NodeRoute {
		t.Fatalf("node %s label = %s, want %s", id, node.Label, scopeir.NodeRoute)
	}
}

func requireRouteRelationship(t *testing.T, g *graph.Graph, relType graph.RelationshipType, sourceID string, targetID string, reason string) {
	t.Helper()
	for _, rel := range g.Relationships {
		if rel.Type == relType && rel.SourceID == sourceID && rel.TargetID == targetID && rel.Reason == reason {
			return
		}
	}
	t.Fatalf("missing %s %s -> %s reason %s", relType, sourceID, targetID, reason)
}

func requireRouteNodeProperties(t *testing.T, g *graph.Graph, route string) graph.NodeProperties {
	t.Helper()
	nodeID := graph.GenerateID(string(scopeir.NodeRoute), route)
	node, ok := g.GetNode(nodeID)
	if !ok {
		t.Fatalf("missing route node %s", route)
	}
	return node.Properties
}

func requireStringPropertyValues(t *testing.T, props graph.NodeProperties, key string, want []string) {
	t.Helper()
	got, ok := props[key].([]string)
	if !ok {
		t.Fatalf("property %s = %#v, want []string", key, props[key])
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("property %s = %#v, want %#v", key, got, want)
	}
}

func requireRouteRelationshipReasonContains(t *testing.T, g *graph.Graph, sourcePath string, route string, wantParts []string) {
	t.Helper()
	sourceID := graph.GenerateID(string(scopeir.NodeFile), sourcePath)
	targetID := graph.GenerateID(string(scopeir.NodeRoute), route)
	for _, rel := range g.Relationships {
		if rel.Type != graph.RelFetches || rel.SourceID != sourceID || rel.TargetID != targetID {
			continue
		}
		for _, want := range wantParts {
			if !strings.Contains(rel.Reason, want) {
				t.Fatalf("relationship reason = %q, missing %q", rel.Reason, want)
			}
		}
		return
	}
	t.Fatalf("missing FETCHES %s -> %s", sourcePath, route)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
