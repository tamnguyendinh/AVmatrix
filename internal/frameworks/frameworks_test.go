package frameworks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestDetectFromPathMatchesFrameworkConventions(t *testing.T) {
	tests := []struct {
		path       string
		framework  string
		reason     string
		multiplier float64
	}{
		{"app/api/users/route.ts", "nextjs-api", "nextjs-api-route", 3.0},
		{"pages/users.tsx", "nextjs-pages", "nextjs-page", 3.0},
		{"app/dashboard/page.tsx", "nextjs-app", "nextjs-app-page", 3.0},
		{"pages/api/users.ts", "nextjs-api", "nextjs-api-route", 3.0},
		{"app/layout.tsx", "nextjs-app", "nextjs-layout", 2.0},
		{"app/_layout.tsx", "expo-router", "expo-layout", 2.0},
		{"app/users+api.ts", "expo-router", "expo-api-route", 3.0},
		{"app/(tabs)/settings.tsx", "expo-router", "expo-screen", 2.5},
		{"routes/auth.ts", "express", "routes-folder", 2.5},
		{"controllers/UserController.ts", "mvc", "controllers-folder", 2.5},
		{"handlers/auth.ts", "handlers", "handlers-folder", 2.5},
		{"views/Button.tsx", "react", "react-component", 1.5},
		{"src/components/Button.tsx", "react", "react-component", 1.5},
		{"electron/main.ts", "electron", "electron-entry", 3.0},
		{"nuxt.config.ts", "nuxt", "nuxt-config", 3.0},
		{"pages/users.vue", "nuxt", "nuxt-page", 3.0},
		{"src/routes/+page.svelte", "sveltekit", "sveltekit-route", 3.0},
		{"src/pages/index.astro", "astro", "astro-page", 3.0},
		{"src/screens/Home.tsx", "react-native", "react-native-screen", 2.5},
		{"service/views.py", "django", "django-views", 3.0},
		{"myapp/urls.py", "django", "django-urls", 2.0},
		{"routers/users.py", "fastapi", "api-routers", 2.5},
		{"controller/UserController.java", "spring", "spring-controller", 3.0},
		{"src/main/java/com/acme/UserController.java", "spring", "spring-controller-file", 3.0},
		{"service/UserService.java", "java-service", "java-service", 1.8},
		{"controllers/UsersController.cs", "aspnet", "aspnet-controller", 3.0},
		{"pages/Index.razor", "blazor", "blazor-page", 2.5},
		{"handlers/user.go", "go-http", "go-handlers", 2.5},
		{"src/routes/UserRoutes.kt", "ktor", "ktor-routes", 2.5},
		{"app/src/main/java/com/acme/ui/theme/Theme.kt", "jetpack-compose", "compose-theme", 2.0},
		{"handlers/auth.rs", "rust-web", "rust-handlers", 2.5},
		{"src/main.rs", "rust", "rust-main", 3.0},
		{"app/Http/Controllers/UserController.php", "laravel", "laravel-controller", 3.0},
		{"routes/web.php", "laravel", "laravel-routes", 3.0},
		{"jobs/SendEmail.php", "laravel", "laravel-job", 2.5},
		{"http/middleware/Auth.php", "laravel", "laravel-middleware", 2.5},
		{"models/User.php", "laravel", "laravel-model", 1.5},
		{"Sources/AppDelegate.swift", "ios", "ios-app-entry", 3.0},
		{"ViewControllers/LoginVC.swift", "uikit", "uikit-viewcontroller", 2.5},
		{"Coordinators/AppCoordinator.swift", "ios-coordinator", "ios-coordinator", 2.5},
		{"views/ContentView.swift", "swiftui", "swiftui-view", 1.8},
		{"lib/screens/HomePage.dart", "flutter", "flutter-screen", 2.5},
		{"cmd/server/main.go", "go", "go-main", 3.0},
		{"src/bin/server.rs", "rust", "rust-bin", 2.5},
		{"src/main.c", "c-cpp", "c-main", 3.0},
		{"src/main.cpp", "c-cpp", "c-main", 3.0},
		{"bin/avmatrix.rb", "ruby", "ruby-executable", 2.5},
		{"routes\\auth.ts", "express", "routes-folder", 2.5},
	}

	for _, tt := range tests {
		got, ok := DetectFromPath(tt.path)
		if !ok {
			t.Fatalf("%s: expected framework hint", tt.path)
		}
		if got.Framework != tt.framework || got.Reason != tt.reason || got.EntryPointMultiplier != tt.multiplier {
			t.Fatalf("%s: hint = %#v, want framework=%s reason=%s multiplier=%v", tt.path, got, tt.framework, tt.reason, tt.multiplier)
		}
	}
}

func TestDetectFromPathReturnsFalseForUnknownPath(t *testing.T) {
	if got, ok := DetectFromPath("src/lib/util.txt"); ok {
		t.Fatalf("unexpected hint: %#v", got)
	}
	if got, ok := DetectFromPath("pages/_app.tsx"); ok {
		t.Fatalf("unexpected Next.js internal page hint: %#v", got)
	}
	if got, ok := DetectFromPath("cmd/server/internal/util.go"); ok {
		t.Fatalf("unexpected Go helper hint: %#v", got)
	}
}

func TestDetectFromASTMatchesFrameworkConventions(t *testing.T) {
	tests := []struct {
		language   scanner.Language
		text       string
		framework  string
		reason     string
		multiplier float64
	}{
		{scanner.TypeScript, "@Controller('/users')\nexport class UsersController {}", "nestjs", "nestjs-decorator", 3.2},
		{scanner.JavaScript, "@Get('/')\nfunction handler() {}", "nestjs", "nestjs-decorator", 3.2},
		{scanner.TypeScript, "import { BrowserWindow } from 'electron';\napp.whenReady().then(() => new BrowserWindow())", "electron", "electron-pattern", 3.0},
		{scanner.TypeScript, "import { View, StyleSheet } from 'react-native';\nStyleSheet.create({ root: {} });", "react-native", "react-native-pattern", 2.5},
		{scanner.TypeScript, "const router = useRouter(); router.push('/settings')", "expo-router", "expo-router-navigation", 2.5},
		{scanner.TypeScript, "const app = express(); app.get('/users', handler)", "express", "express-pattern", 2.8},
		{scanner.TypeScript, "export default defineNuxtConfig({});", "nuxt", "nuxt-pattern", 2.8},
		{scanner.Python, "@app.get('/users')\ndef list_users(): pass", "fastapi", "fastapi-decorator", 3.0},
		{scanner.Python, "@app.route('/users')\ndef list_users(): pass", "flask", "flask-decorator", 2.8},
		{scanner.Java, "@RestController\nclass UsersController {}", "spring", "spring-annotation", 3.2},
		{scanner.Java, "@Path('/users')\nclass UsersResource {}", "jaxrs", "jaxrs-annotation", 3.0},
		{scanner.Kotlin, "@Composable\nfun Home() { Scaffold { Text(\"Hi\") } }", "jetpack-compose", "compose-pattern", 2.8},
		{scanner.CSharp, "[HttpGet]\npublic IActionResult Get() {}", "aspnet", "aspnet-attribute", 3.2},
		{scanner.CSharp, "class ChatHub : Hub {}", "signalr", "signalr-attribute", 2.8},
		{scanner.CSharp, "@page \"/counter\"\n@inject HttpClient Http", "blazor", "blazor-attribute", 2.5},
		{scanner.CSharp, "class AppDbContext : DbContext { DbSet<User> Users { get; set; } }", "efcore", "efcore-pattern", 2.0},
		{scanner.PHP, "Route::get('/users', [UserController::class, 'index']);", "laravel", "php-route-attribute", 3.0},
		{scanner.Go, "func handler(w http.ResponseWriter, r *http.Request) {}", "go-http", "go-http-handler", 2.5},
		{scanner.Go, "func main() { r := gin.Default(); r.GET(\"/\", handler) }", "gin", "gin-handler", 3.0},
		{scanner.Go, "func main() { e := echo.New(); _ = e }", "echo", "echo-handler", 3.0},
		{scanner.Go, "func main() { app := fiber.New(); _ = app }", "fiber", "fiber-handler", 3.0},
		{scanner.Go, "func register(s *grpc.Server) { pb.RegisterServer(s, svc) }", "go-grpc", "grpc-service", 2.8},
		{scanner.Rust, "#[get(\"/\")]\nasync fn index() -> HttpResponse { todo!() }", "actix-web", "actix-attribute", 3.0},
		{scanner.Rust, "let app = Router::new().route(\"/\", get(handler));", "axum", "axum-routing", 3.0},
		{scanner.Rust, "#[launch]\nfn rocket() -> _ { rocket::build() }", "rocket", "rocket-attribute", 3.0},
		{scanner.Rust, "#[tokio::main]\nasync fn main() {}", "tokio", "tokio-runtime", 2.5},
		{scanner.CPlusPlus, "class Widget : public QWidget { Q_OBJECT };", "qt", "qt-macro", 2.8},
		{scanner.Swift, "struct AppView: View { var body: some View { WindowGroup { ContentView() } } }", "swiftui", "swiftui-pattern", 2.8},
		{scanner.Swift, "func routes(_ app: Application) throws { app.get(\"users\") { req in } }", "vapor", "vapor-routing", 3.0},
		{scanner.Ruby, "class UsersController < ApplicationController\nend", "rails", "rails-pattern", 3.0},
		{scanner.Ruby, "class App < Sinatra::Base\nend", "sinatra", "sinatra-pattern", 2.8},
		{scanner.Dart, "class Home extends StatelessWidget { Widget build(BuildContext context) => Scaffold(); }", "flutter", "flutter-widget", 2.5},
		{scanner.Dart, "@riverpod\nFuture<User> user(UserRef ref) async => ref.watch(userProvider);", "riverpod", "riverpod-pattern", 2.8},
		{scanner.Vue, "export default defineComponent({ setup() { return {} } })", "vue", "vue-pattern", 2.0},
		{scanner.Svelte, "import { onMount } from 'svelte';\nonMount(() => {});", "svelte", "svelte-pattern", 2.0},
		{scanner.Svelte, "import { redirect } from '@sveltejs/kit';\nexport const load = () => ({})", "sveltekit", "sveltekit-pattern", 2.8},
		{scanner.Astro, "const title = Astro.props.title;\nexport async function getStaticPaths() { return []; }", "astro", "astro-pattern", 2.5},
	}

	for _, tt := range tests {
		got, ok := DetectFromAST(tt.language, tt.text)
		if !ok {
			t.Fatalf("%s: expected framework hint", tt.language)
		}
		if got.Framework != tt.framework || got.Reason != tt.reason || got.EntryPointMultiplier != tt.multiplier {
			t.Fatalf("%s: hint = %#v, want framework=%s reason=%s multiplier=%v", tt.language, got, tt.framework, tt.reason, tt.multiplier)
		}
	}
}

func TestDetectFromASTReturnsFalseForEmptyOrUnmatchedInput(t *testing.T) {
	tests := []struct {
		language scanner.Language
		text     string
	}{
		{"", ""},
		{scanner.TypeScript, ""},
		{"", "some code"},
		{scanner.C, "int main() { return 0; }"},
	}
	for _, tt := range tests {
		if got, ok := DetectFromAST(tt.language, tt.text); ok {
			t.Fatalf("DetectFromAST(%q, %q) = %#v, true; want false", tt.language, tt.text, got)
		}
	}
}

func TestASTPatternTableCoversExpectedFrameworkFamilies(t *testing.T) {
	expected := []string{
		"nestjs", "expo-router", "express", "fastapi", "flask", "spring", "jaxrs",
		"aspnet", "signalr", "blazor", "efcore", "go-http", "gin", "echo", "fiber",
		"go-grpc", "laravel", "actix-web", "axum", "rocket", "tokio", "qt", "uikit",
		"swiftui", "vapor", "rails", "sinatra", "flutter", "riverpod", "vue", "nuxt",
		"svelte", "sveltekit", "astro",
	}
	seen := map[string]bool{}
	for _, configs := range astPatternsByLanguage {
		for _, config := range configs {
			if len(config.patterns) == 0 {
				t.Fatalf("framework %s has no AST patterns", config.framework)
			}
			seen[config.framework] = true
		}
	}
	for _, framework := range expected {
		if !seen[framework] {
			t.Fatalf("missing AST pattern family %q; seen=%#v", framework, seen)
		}
	}
}

func TestAnnotateScopeIRAddsFrameworkFactsFromDefinitionWindow(t *testing.T) {
	source := []byte("@Controller('/users')\nexport class UsersController {\n  list() {}\n}\n")
	ir := scopeir.ScopeIR{
		FilePath: "src/users.controller.ts",
		FileHash: "hash-users",
		Language: scanner.TypeScript,
		Definitions: []scopeir.DefinitionFact{{
			ID:       "def:src/users.controller.ts#2:0:Class:UsersController",
			FilePath: "src/users.controller.ts",
			FileHash: "hash-users",
			Name:     "UsersController",
			Label:    scopeir.NodeClass,
			Range:    scopeir.Range{StartLine: 2, EndLine: 4},
		}},
	}

	got := AnnotateScopeIR(ir, source)
	if len(got.Frameworks) != 1 {
		t.Fatalf("framework facts = %#v, want one fact", got.Frameworks)
	}
	fact := got.Frameworks[0]
	if fact.DefID != ir.Definitions[0].ID || fact.Framework != "nestjs" || fact.Reason != "nestjs-decorator" || fact.EntryPointMultiplier != 3.2 {
		t.Fatalf("framework fact = %#v", fact)
	}
}

func TestDefinitionWindowLineRangeContract(t *testing.T) {
	source := []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\n")
	tests := []struct {
		name string
		rng  scopeir.Range
		want string
	}{
		{
			name: "includes five lines before start",
			rng:  scopeir.Range{StartLine: 6, EndLine: 7},
			want: "one\ntwo\nthree\nfour\nfive\nsix\nseven",
		},
		{
			name: "end before start falls back to start line",
			rng:  scopeir.Range{StartLine: 2, EndLine: 1},
			want: "one\ntwo",
		},
		{
			name: "start beyond source returns empty",
			rng:  scopeir.Range{StartLine: 20, EndLine: 20},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := definitionWindow(source, tt.rng); got != tt.want {
				t.Fatalf("definitionWindow() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefinitionWindowCapsLargeWindow(t *testing.T) {
	source := []byte(strings.Repeat("x", 700))
	got := definitionWindow(source, scopeir.Range{StartLine: 1, EndLine: 1})
	if len(got) != 600 {
		t.Fatalf("definitionWindow length = %d, want 600", len(got))
	}
}

func BenchmarkDetectFromPath(b *testing.B) {
	paths := []string{
		"app/api/users/route.ts",
		"src/components/Button.tsx",
		"service/views.py",
		"src/main/java/com/acme/UserController.java",
		"src/routes/UserRoutes.kt",
		"app/Http/Controllers/UserController.php",
		"Sources/AppDelegate.swift",
		"lib/screens/HomePage.dart",
		"cmd/server/main.go",
		"src/bin/server.rs",
		"bin/avmatrix.rb",
		"src/lib/util.txt",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, filePath := range paths {
			_, _ = DetectFromPath(filePath)
		}
	}
}

func BenchmarkDetectFromAST(b *testing.B) {
	text := "@Controller('/users')\nexport class UsersController {\n  @Get()\n  list() {}\n}\n"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = DetectFromAST(scanner.TypeScript, text)
	}
}

func BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions(b *testing.B) {
	source, ir := largeFrameworkScopeIR(2000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := AnnotateScopeIR(ir, source)
		if len(got.Frameworks) != len(ir.Definitions) {
			b.Fatalf("framework facts = %d, want %d", len(got.Frameworks), len(ir.Definitions))
		}
	}
}

func largeFrameworkScopeIR(definitionCount int) ([]byte, scopeir.ScopeIR) {
	var source strings.Builder
	source.Grow(definitionCount * 90)
	definitions := make([]scopeir.DefinitionFact, 0, definitionCount)

	for i := 0; i < definitionCount; i++ {
		lineBase := i*4 + 1
		name := fmt.Sprintf("UsersController%d", i)
		source.WriteString("@Controller('/users')\n")
		source.WriteString("export class ")
		source.WriteString(name)
		source.WriteString(" {\n")
		source.WriteString("  list() {}\n")
		source.WriteString("}\n")
		definitions = append(definitions, scopeir.DefinitionFact{
			ID:       fmt.Sprintf("def:%d", i),
			FilePath: "src/users.controller.ts",
			FileHash: "hash-users",
			Name:     name,
			Label:    scopeir.NodeClass,
			Range: scopeir.Range{
				StartLine: lineBase + 1,
				EndLine:   lineBase + 4,
			},
		})
	}

	return []byte(source.String()), scopeir.ScopeIR{
		FilePath:        "src/users.controller.ts",
		FileHash:        "hash-users",
		Language:        scanner.TypeScript,
		Definitions:     definitions,
		ModuleScope:     "module:src/users.controller.ts",
		Version:         scopeir.Version,
		Scopes:          nil,
		Imports:         nil,
		Calls:           nil,
		Accesses:        nil,
		Heritage:        nil,
		Frameworks:      nil,
		Domains:         nil,
		ReturnTypes:     nil,
		TypeAnnotations: nil,
	}
}
