package frameworks

import (
	"strings"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type astPatternConfig struct {
	framework            string
	entryPointMultiplier float64
	reason               string
	patterns             []string
}

var astPatternsByLanguage = map[scanner.Language][]astPatternConfig{
	scanner.JavaScript: {
		{framework: "react", entryPointMultiplier: 1.8, reason: "react-pattern", patterns: lowerPatterns("useState", "useEffect", "React.FC", "createRoot", "jsx-runtime")},
		{framework: "nextjs", entryPointMultiplier: 2.8, reason: "nextjs-pattern", patterns: lowerPatterns("getServerSideProps", "getStaticProps", "NextRequest", "NextResponse", "next/navigation")},
		{framework: "electron", entryPointMultiplier: 3.0, reason: "electron-pattern", patterns: lowerPatterns("BrowserWindow", "ipcMain", "ipcRenderer", "contextBridge", "app.whenReady")},
		{framework: "react-native", entryPointMultiplier: 2.5, reason: "react-native-pattern", patterns: lowerPatterns("react-native", "StyleSheet.create", "NavigationContainer", "useNavigation", "<View")},
		{framework: "nuxt", entryPointMultiplier: 2.8, reason: "nuxt-pattern", patterns: lowerPatterns("defineNuxtConfig", "useAsyncData", "useFetch", "NuxtPage", "definePageMeta")},
		{framework: "svelte", entryPointMultiplier: 2.0, reason: "svelte-pattern", patterns: lowerPatterns("svelte/store", "onMount", "createEventDispatcher", "$:")},
		{framework: "astro", entryPointMultiplier: 2.5, reason: "astro-pattern", patterns: lowerPatterns("Astro.props", "Astro.request", "getStaticPaths", "Astro.glob")},
		{framework: "nestjs", entryPointMultiplier: 3.2, reason: "nestjs-decorator", patterns: lowerPatterns("@Controller", "@Get", "@Post", "@Put", "@Delete", "@Patch")},
		{framework: "expo-router", entryPointMultiplier: 2.5, reason: "expo-router-navigation", patterns: lowerPatterns("router.push", "router.replace", "router.navigate", "useRouter", "useLocalSearchParams", "useSegments", "expo-router")},
		{framework: "express", entryPointMultiplier: 2.8, reason: "express-pattern", patterns: lowerPatterns("express()", "app.get", "app.post", "app.put", "app.delete", "router.get", "router.post", "router.put", "router.delete")},
	},
	scanner.TypeScript: {
		{framework: "react", entryPointMultiplier: 1.8, reason: "react-pattern", patterns: lowerPatterns("useState", "useEffect", "React.FC", "createRoot", "jsx-runtime")},
		{framework: "nextjs", entryPointMultiplier: 2.8, reason: "nextjs-pattern", patterns: lowerPatterns("getServerSideProps", "getStaticProps", "NextRequest", "NextResponse", "next/navigation")},
		{framework: "electron", entryPointMultiplier: 3.0, reason: "electron-pattern", patterns: lowerPatterns("BrowserWindow", "ipcMain", "ipcRenderer", "contextBridge", "app.whenReady")},
		{framework: "react-native", entryPointMultiplier: 2.5, reason: "react-native-pattern", patterns: lowerPatterns("react-native", "StyleSheet.create", "NavigationContainer", "useNavigation", "<View")},
		{framework: "nuxt", entryPointMultiplier: 2.8, reason: "nuxt-pattern", patterns: lowerPatterns("defineNuxtConfig", "useAsyncData", "useFetch", "NuxtPage", "definePageMeta")},
		{framework: "svelte", entryPointMultiplier: 2.0, reason: "svelte-pattern", patterns: lowerPatterns("svelte/store", "onMount", "createEventDispatcher", "$:")},
		{framework: "astro", entryPointMultiplier: 2.5, reason: "astro-pattern", patterns: lowerPatterns("Astro.props", "Astro.request", "getStaticPaths", "Astro.glob")},
		{framework: "nestjs", entryPointMultiplier: 3.2, reason: "nestjs-decorator", patterns: lowerPatterns("@Controller", "@Get", "@Post", "@Put", "@Delete", "@Patch")},
		{framework: "expo-router", entryPointMultiplier: 2.5, reason: "expo-router-navigation", patterns: lowerPatterns("router.push", "router.replace", "router.navigate", "useRouter", "useLocalSearchParams", "useSegments", "expo-router")},
		{framework: "express", entryPointMultiplier: 2.8, reason: "express-pattern", patterns: lowerPatterns("express()", "app.get", "app.post", "app.put", "app.delete", "router.get", "router.post", "router.put", "router.delete")},
	},
	scanner.Python: {
		{framework: "fastapi", entryPointMultiplier: 3.0, reason: "fastapi-decorator", patterns: lowerPatterns("@app.get", "@app.post", "@app.put", "@app.delete", "@router.get")},
		{framework: "flask", entryPointMultiplier: 2.8, reason: "flask-decorator", patterns: lowerPatterns("@app.route", "@blueprint.route")},
	},
	scanner.Java: {
		{framework: "spring", entryPointMultiplier: 3.2, reason: "spring-annotation", patterns: lowerPatterns("@RestController", "@Controller", "@GetMapping", "@PostMapping", "@RequestMapping")},
		{framework: "jaxrs", entryPointMultiplier: 3.0, reason: "jaxrs-annotation", patterns: lowerPatterns("@Path", "@GET", "@POST", "@PUT", "@DELETE")},
	},
	scanner.Kotlin: {
		{framework: "spring-kotlin", entryPointMultiplier: 3.2, reason: "spring-kotlin-annotation", patterns: lowerPatterns("@RestController", "@Controller", "@GetMapping", "@PostMapping", "@RequestMapping")},
		{framework: "jaxrs", entryPointMultiplier: 3.0, reason: "jaxrs-annotation", patterns: lowerPatterns("@Path", "@GET", "@POST", "@PUT", "@DELETE")},
		{framework: "ktor", entryPointMultiplier: 2.8, reason: "ktor-routing", patterns: lowerPatterns("routing", "embeddedServer", "Application.module")},
		{framework: "android-kotlin", entryPointMultiplier: 2.5, reason: "android-annotation", patterns: lowerPatterns("@AndroidEntryPoint", "AppCompatActivity", "Fragment(")},
		{framework: "jetpack-compose", entryPointMultiplier: 2.8, reason: "compose-pattern", patterns: lowerPatterns("@Composable", "setContent", "remember {", "Modifier.", "LazyColumn", "Scaffold(")},
	},
	scanner.CSharp: {
		{framework: "aspnet", entryPointMultiplier: 3.2, reason: "aspnet-attribute", patterns: lowerPatterns("[ApiController]", "[HttpGet]", "[HttpPost]", "[HttpPut]", "[HttpDelete]", "[Route]", "[Authorize]", "[AllowAnonymous]")},
		{framework: "signalr", entryPointMultiplier: 2.8, reason: "signalr-attribute", patterns: lowerPatterns("[HubMethodName]", ": Hub", ": Hub<")},
		{framework: "blazor", entryPointMultiplier: 2.5, reason: "blazor-attribute", patterns: lowerPatterns("@page", "[Parameter]", "@inject")},
		{framework: "efcore", entryPointMultiplier: 2.0, reason: "efcore-pattern", patterns: lowerPatterns("DbContext", "DbSet<", "OnModelCreating")},
	},
	scanner.PHP: {
		{framework: "laravel", entryPointMultiplier: 3.0, reason: "php-route-attribute", patterns: lowerPatterns("Route::get", "Route::post", "Route::put", "Route::delete", "Route::resource", "Route::apiResource", "#[Route(")},
	},
	scanner.Go: {
		{framework: "go-http", entryPointMultiplier: 2.5, reason: "go-http-handler", patterns: lowerPatterns("http.Handler", "http.HandlerFunc", "ServeHTTP", "http.ResponseWriter", "http.Request")},
		{framework: "gin", entryPointMultiplier: 3.0, reason: "gin-handler", patterns: lowerPatterns("gin.Context", "gin.Default", "gin.New")},
		{framework: "echo", entryPointMultiplier: 3.0, reason: "echo-handler", patterns: lowerPatterns("echo.Context", "echo.New")},
		{framework: "fiber", entryPointMultiplier: 3.0, reason: "fiber-handler", patterns: lowerPatterns("fiber.Ctx", "fiber.New", "fiber.App")},
		{framework: "go-grpc", entryPointMultiplier: 2.8, reason: "grpc-service", patterns: lowerPatterns("grpc.Server", "RegisterServer", "pb.Unimplemented")},
	},
	scanner.Rust: {
		{framework: "actix-web", entryPointMultiplier: 3.0, reason: "actix-attribute", patterns: lowerPatterns("#[get", "#[post", "#[put", "#[delete", "#[actix_web", "HttpRequest", "HttpResponse")},
		{framework: "axum", entryPointMultiplier: 3.0, reason: "axum-routing", patterns: lowerPatterns("Router::new", "axum::extract", "axum::routing")},
		{framework: "rocket", entryPointMultiplier: 3.0, reason: "rocket-attribute", patterns: lowerPatterns("#[get", "#[post", "#[launch", "rocket::")},
		{framework: "tokio", entryPointMultiplier: 2.5, reason: "tokio-runtime", patterns: lowerPatterns("#[tokio::main]", "#[tokio::test]")},
	},
	scanner.CPlusPlus: {
		{framework: "qt", entryPointMultiplier: 2.8, reason: "qt-macro", patterns: lowerPatterns("Q_OBJECT", "Q_INVOKABLE", "Q_PROPERTY", "Q_SIGNALS", "Q_SLOTS", "Q_SIGNAL", "Q_SLOT", "QWidget", "QApplication")},
	},
	scanner.Swift: {
		{framework: "uikit", entryPointMultiplier: 2.5, reason: "uikit-lifecycle", patterns: lowerPatterns("viewDidLoad", "viewWillAppear", "viewDidAppear", "UIViewController", "@IBOutlet", "@IBAction", "@objc")},
		{framework: "swiftui", entryPointMultiplier: 2.8, reason: "swiftui-pattern", patterns: lowerPatterns("@main", "WindowGroup", "ContentView", "@StateObject", "@ObservedObject", "@EnvironmentObject", "@Published")},
		{framework: "vapor", entryPointMultiplier: 3.0, reason: "vapor-routing", patterns: lowerPatterns("app.get", "app.post", "req.content.decode", "Vapor")},
	},
	scanner.Ruby: {
		{framework: "rails", entryPointMultiplier: 3.0, reason: "rails-pattern", patterns: lowerPatterns("ApplicationController", "ApplicationRecord", "ActiveRecord::Base", "before_action", "after_action", "has_many", "belongs_to", "has_one", "validates")},
		{framework: "sinatra", entryPointMultiplier: 2.8, reason: "sinatra-pattern", patterns: lowerPatterns("Sinatra::Base", "Sinatra::Application")},
	},
	scanner.Dart: {
		{framework: "flutter", entryPointMultiplier: 2.5, reason: "flutter-widget", patterns: lowerPatterns("StatelessWidget", "StatefulWidget", "BuildContext", "Widget build", "ChangeNotifier", "GetxController", "Cubit<", "Bloc<", "ConsumerWidget", "MaterialApp", "Scaffold", "Navigator.")},
		{framework: "riverpod", entryPointMultiplier: 2.8, reason: "riverpod-pattern", patterns: lowerPatterns("@riverpod", "ref.watch", "ref.read", "AsyncNotifier", "Notifier")},
	},
	scanner.Vue: {
		{framework: "vue", entryPointMultiplier: 2.0, reason: "vue-pattern", patterns: lowerPatterns("defineComponent", "createApp", "ref(", "computed(", "watch(")},
		{framework: "nuxt", entryPointMultiplier: 2.8, reason: "nuxt-pattern", patterns: lowerPatterns("defineNuxtConfig", "useAsyncData", "useFetch", "NuxtPage", "definePageMeta")},
	},
	scanner.Svelte: {
		{framework: "svelte", entryPointMultiplier: 2.0, reason: "svelte-pattern", patterns: lowerPatterns("svelte/store", "onMount", "createEventDispatcher", "$:")},
		{framework: "sveltekit", entryPointMultiplier: 2.8, reason: "sveltekit-pattern", patterns: lowerPatterns("@sveltejs/kit", "load(", "actions", "redirect(")},
	},
	scanner.Astro: {
		{framework: "astro", entryPointMultiplier: 2.5, reason: "astro-pattern", patterns: lowerPatterns("Astro.props", "Astro.request", "getStaticPaths", "Astro.glob")},
	},
}

func DetectFromAST(language scanner.Language, definitionText string) (Hint, bool) {
	if language == "" || definitionText == "" {
		return Hint{}, false
	}
	configs := astPatternsByLanguage[language]
	if len(configs) == 0 {
		return Hint{}, false
	}
	normalized := strings.ToLower(definitionText)
	for _, config := range configs {
		for _, pattern := range config.patterns {
			if strings.Contains(normalized, pattern) {
				return Hint{
					Framework:            config.framework,
					EntryPointMultiplier: config.entryPointMultiplier,
					Reason:               config.reason,
				}, true
			}
		}
	}
	return Hint{}, false
}

func AnnotateScopeIR(ir scopeir.ScopeIR, source []byte) scopeir.ScopeIR {
	if len(ir.Definitions) == 0 || len(source) == 0 {
		return ir
	}
	out := ir
	existing := make(map[string]struct{}, len(out.Frameworks))
	for _, fact := range out.Frameworks {
		existing[frameworkFactKey(fact)] = struct{}{}
	}
	windows := newDefinitionWindowIndex(source)
	for _, def := range out.Definitions {
		text := windows.window(def.Range)
		if len(def.Annotations) > 0 {
			text = strings.Join(def.Annotations, "\n") + "\n" + text
		}
		hint, ok := DetectFromAST(ir.Language, text)
		if !ok {
			continue
		}
		fact := scopeir.FrameworkFact{
			DefID:                def.ID,
			FilePath:             def.FilePath,
			FileHash:             def.FileHash,
			Framework:            hint.Framework,
			Reason:               hint.Reason,
			EntryPointMultiplier: hint.EntryPointMultiplier,
			Range:                def.Range,
		}
		key := frameworkFactKey(fact)
		if _, ok := existing[key]; ok {
			continue
		}
		out.Frameworks = append(out.Frameworks, fact)
		existing[key] = struct{}{}
	}
	return out
}

func lowerPatterns(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.ToLower(value))
	}
	return out
}

func definitionWindow(source []byte, rng scopeir.Range) string {
	return newDefinitionWindowIndex(source).window(rng)
}

type definitionWindowIndex struct {
	source     []byte
	lineStarts []int
}

func newDefinitionWindowIndex(source []byte) definitionWindowIndex {
	lineStarts := make([]int, 0, 1+len(source)/80)
	lineStarts = append(lineStarts, 0)
	for index, value := range source {
		if value == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}
	return definitionWindowIndex{
		source:     source,
		lineStarts: lineStarts,
	}
}

func (index definitionWindowIndex) window(rng scopeir.Range) string {
	lineCount := len(index.lineStarts)
	if lineCount == 0 {
		return ""
	}
	start := rng.StartLine - 5
	if start < 1 {
		start = 1
	}
	end := rng.EndLine
	if end < rng.StartLine {
		end = rng.StartLine
	}
	if end > lineCount {
		end = lineCount
	}
	if start > lineCount || end < start {
		return ""
	}
	from := index.lineStarts[start-1]
	to := index.lineEnd(end - 1)
	if to < from {
		return ""
	}
	if to-from > 600 {
		to = from + 600
	}
	return string(index.source[from:to])
}

func (index definitionWindowIndex) lineEnd(lineIndex int) int {
	if lineIndex < 0 || lineIndex >= len(index.lineStarts) {
		return len(index.source)
	}
	if lineIndex+1 < len(index.lineStarts) {
		return index.lineStarts[lineIndex+1] - 1
	}
	return len(index.source)
}

func frameworkFactKey(fact scopeir.FrameworkFact) string {
	return fact.DefID + "\x00" + fact.Framework + "\x00" + fact.Reason
}
