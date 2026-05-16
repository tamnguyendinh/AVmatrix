package frameworks

import (
	"path"
	"strings"
	"unicode"
)

type Hint struct {
	Framework            string
	EntryPointMultiplier float64
	Reason               string
}

func DetectFromPath(filePath string) (Hint, bool) {
	originalPath := strings.ReplaceAll(filePath, "\\", "/")
	p := strings.ToLower(originalPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	originalWithSlash := originalPath
	if !strings.HasPrefix(originalWithSlash, "/") {
		originalWithSlash = "/" + originalWithSlash
	}

	if strings.Contains(p, "/pages/") && !strings.Contains(p, "/_") && !strings.Contains(p, "/api/") && hasAnySuffix(p, ".tsx", ".ts", ".jsx", ".js") {
		return hint("nextjs-pages", 3.0, "nextjs-page")
	}
	if strings.Contains(p, "/app/") && hasAnySuffix(p, "page.tsx", "page.ts", "page.jsx", "page.js") {
		return hint("nextjs-app", 3.0, "nextjs-app-page")
	}
	if strings.Contains(p, "/pages/api/") || (strings.Contains(p, "/app/") && strings.Contains(p, "/api/") && strings.HasSuffix(p, "route.ts")) {
		return hint("nextjs-api", 3.0, "nextjs-api-route")
	}
	if strings.Contains(p, "/app/") && !strings.Contains(p, "_layout") && hasAnySuffix(p, "layout.tsx", "layout.ts") {
		return hint("nextjs-app", 2.0, "nextjs-layout")
	}
	if strings.Contains(p, "/app/") && hasAnySuffix(p, ".tsx", ".ts", ".jsx", ".js") {
		fileName := path.Base(p)
		switch {
		case strings.HasPrefix(fileName, "_layout"):
			return hint("expo-router", 2.0, "expo-layout")
		case strings.HasPrefix(fileName, "+") && !strings.HasPrefix(fileName, "+api"):
			return hint("expo-router", 1.5, "expo-special-route")
		case strings.HasSuffix(fileName, "+api.ts") || strings.HasSuffix(fileName, "+api.tsx"):
			return hint("expo-router", 3.0, "expo-api-route")
		default:
			return hint("expo-router", 2.5, "expo-screen")
		}
	}
	if strings.HasSuffix(p, "/electron/main.ts") || strings.HasSuffix(p, "/electron/main.js") ||
		strings.HasSuffix(p, "/electron/preload.ts") || strings.HasSuffix(p, "/electron/preload.js") ||
		strings.Contains(p, "/src-electron/") || strings.Contains(p, "/electron/") && hasAnySuffix(p, ".ts", ".js") {
		return hint("electron", 3.0, "electron-entry")
	}
	if strings.HasSuffix(p, "/nuxt.config.ts") || strings.HasSuffix(p, "/nuxt.config.js") || strings.HasSuffix(p, "/nuxt.config.mjs") {
		return hint("nuxt", 3.0, "nuxt-config")
	}
	if strings.Contains(p, "/pages/") && strings.HasSuffix(p, ".vue") {
		return hint("nuxt", 3.0, "nuxt-page")
	}
	if strings.Contains(p, "/server/api/") && hasAnySuffix(p, ".ts", ".js") {
		return hint("nuxt", 3.0, "nuxt-server-api")
	}
	if (strings.Contains(p, "/composables/") || strings.Contains(p, "/plugins/")) && hasAnySuffix(p, ".ts", ".js") {
		return hint("nuxt", 2.0, "nuxt-composable")
	}
	if strings.Contains(p, "/prisma/") && strings.HasSuffix(p, "schema.prisma") {
		return hint("prisma", 1.5, "prisma-schema")
	}
	if (strings.Contains(p, "/lib/supabase") || strings.Contains(p, "/utils/supabase") || strings.Contains(p, "/supabase/")) && hasAnySuffix(p, ".ts", ".js") {
		return hint("supabase", 1.5, "supabase-client")
	}
	if strings.Contains(p, "/routes/") && hasAnySuffix(p, ".ts", ".js") {
		return hint("express", 2.5, "routes-folder")
	}
	if strings.Contains(p, "/controllers/") && hasAnySuffix(p, ".ts", ".js") {
		return hint("mvc", 2.5, "controllers-folder")
	}
	if strings.Contains(p, "/handlers/") && hasAnySuffix(p, ".ts", ".js") {
		return hint("handlers", 2.5, "handlers-folder")
	}
	if (strings.Contains(p, "/components/") || strings.Contains(p, "/views/")) && hasAnySuffix(p, ".tsx", ".jsx") && startsWithUpper(path.Base(originalWithSlash)) {
		return hint("react", 1.5, "react-component")
	}
	if (strings.Contains(p, "/screens/") || strings.Contains(p, "/navigation/")) && hasAnySuffix(p, ".tsx", ".jsx", ".ts", ".js") {
		return hint("react-native", 2.5, "react-native-screen")
	}
	if strings.HasSuffix(p, "/app.tsx") || strings.HasSuffix(p, "/app.jsx") {
		return hint("react-native", 2.5, "react-native-app-entry")
	}
	if strings.Contains(p, "/src/routes/") && hasAnySuffix(p, ".svelte", ".ts", ".js") {
		return hint("sveltekit", 3.0, "sveltekit-route")
	}
	if strings.Contains(p, "/routes/") && hasAnySuffix(p, "+page.svelte", "+layout.svelte", "+server.ts", "+server.js") {
		return hint("sveltekit", 3.0, "sveltekit-route")
	}
	if strings.HasSuffix(p, ".svelte") {
		return hint("svelte", 2.0, "svelte-component")
	}
	if strings.Contains(p, "/src/pages/") && strings.HasSuffix(p, ".astro") {
		return hint("astro", 3.0, "astro-page")
	}
	if strings.HasSuffix(p, "/astro.config.ts") || strings.HasSuffix(p, "/astro.config.js") || strings.HasSuffix(p, "/astro.config.mjs") {
		return hint("astro", 3.0, "astro-config")
	}
	if strings.HasSuffix(p, ".astro") {
		return hint("astro", 2.0, "astro-component")
	}

	if strings.HasSuffix(p, "views.py") {
		return hint("django", 3.0, "django-views")
	}
	if strings.HasSuffix(p, "urls.py") {
		return hint("django", 2.0, "django-urls")
	}
	if (strings.Contains(p, "/routers/") || strings.Contains(p, "/endpoints/") || strings.Contains(p, "/routes/")) && strings.HasSuffix(p, ".py") {
		return hint("fastapi", 2.5, "api-routers")
	}
	if strings.Contains(p, "/api/") && strings.HasSuffix(p, ".py") && !strings.HasSuffix(p, "__init__.py") {
		return hint("python-api", 2.0, "api-folder")
	}

	if (strings.Contains(p, "/controller/") || strings.Contains(p, "/controllers/")) && strings.HasSuffix(p, ".java") {
		return hint("spring", 3.0, "spring-controller")
	}
	if strings.HasSuffix(p, "controller.java") {
		return hint("spring", 3.0, "spring-controller-file")
	}
	if (strings.Contains(p, "/service/") || strings.Contains(p, "/services/")) && strings.HasSuffix(p, ".java") {
		return hint("java-service", 1.8, "java-service")
	}

	if (strings.Contains(p, "/controller/") || strings.Contains(p, "/controllers/")) && strings.HasSuffix(p, ".kt") {
		return hint("spring-kotlin", 3.0, "spring-kotlin-controller")
	}
	if strings.HasSuffix(p, "controller.kt") {
		return hint("spring-kotlin", 3.0, "spring-kotlin-controller-file")
	}
	if strings.Contains(p, "/routes/") && strings.HasSuffix(p, ".kt") {
		return hint("ktor", 2.5, "ktor-routes")
	}
	if strings.Contains(p, "/plugins/") && strings.HasSuffix(p, ".kt") {
		return hint("ktor", 2.0, "ktor-plugin")
	}
	if strings.HasSuffix(p, "routing.kt") || strings.HasSuffix(p, "routes.kt") {
		return hint("ktor", 2.5, "ktor-routing-file")
	}
	if strings.Contains(p, "/ui/theme/") && strings.HasSuffix(p, ".kt") {
		return hint("jetpack-compose", 2.0, "compose-theme")
	}
	if (strings.Contains(p, "/activity/") || strings.Contains(p, "/ui/")) && strings.HasSuffix(p, ".kt") {
		return hint("android-kotlin", 2.5, "android-ui")
	}
	if strings.HasSuffix(p, "activity.kt") {
		return hint("jetpack-compose", 2.8, "compose-activity")
	}
	if strings.HasSuffix(p, "fragment.kt") {
		return hint("android-kotlin", 2.5, "android-component")
	}
	if strings.HasSuffix(p, "/main.kt") {
		return hint("kotlin", 3.0, "kotlin-main")
	}
	if strings.HasSuffix(p, "/application.kt") {
		return hint("kotlin", 2.5, "kotlin-application")
	}

	if strings.Contains(p, "/controllers/") && strings.HasSuffix(p, ".cs") {
		return hint("aspnet", 3.0, "aspnet-controller")
	}
	if strings.HasSuffix(p, "controller.cs") {
		return hint("aspnet", 3.0, "aspnet-controller-file")
	}
	if (strings.Contains(p, "/services/") || strings.Contains(p, "/service/")) && strings.HasSuffix(p, ".cs") {
		return hint("aspnet", 1.8, "aspnet-service")
	}
	if strings.Contains(p, "/middleware/") && strings.HasSuffix(p, ".cs") {
		return hint("aspnet", 2.5, "aspnet-middleware")
	}
	if strings.Contains(p, "/hubs/") && strings.HasSuffix(p, ".cs") {
		return hint("signalr", 2.5, "signalr-hub")
	}
	if strings.HasSuffix(p, "hub.cs") {
		return hint("signalr", 2.5, "signalr-hub-file")
	}
	if strings.HasSuffix(p, "/program.cs") || strings.HasSuffix(p, "/startup.cs") {
		return hint("aspnet", 3.0, "aspnet-entry")
	}
	if (strings.Contains(p, "/backgroundservices/") || strings.Contains(p, "/hostedservices/")) && strings.HasSuffix(p, ".cs") {
		return hint("aspnet", 2.0, "aspnet-background-service")
	}
	if strings.Contains(p, "/pages/") && strings.HasSuffix(p, ".razor") {
		return hint("blazor", 2.5, "blazor-page")
	}

	if (strings.Contains(p, "/handlers/") || strings.Contains(p, "/handler/")) && strings.HasSuffix(p, ".go") {
		return hint("go-http", 2.5, "go-handlers")
	}
	if strings.Contains(p, "/routes/") && strings.HasSuffix(p, ".go") {
		return hint("go-http", 2.5, "go-routes")
	}
	if strings.Contains(p, "/controllers/") && strings.HasSuffix(p, ".go") {
		return hint("go-mvc", 2.5, "go-controller")
	}
	if strings.HasSuffix(p, "/main.go") {
		return hint("go", 3.0, "go-main")
	}

	if (strings.Contains(p, "/handlers/") || strings.Contains(p, "/routes/")) && strings.HasSuffix(p, ".rs") {
		return hint("rust-web", 2.5, "rust-handlers")
	}
	if strings.HasSuffix(p, "/main.rs") {
		return hint("rust", 3.0, "rust-main")
	}
	if strings.Contains(p, "/bin/") && strings.HasSuffix(p, ".rs") {
		return hint("rust", 2.5, "rust-bin")
	}

	if strings.HasSuffix(p, "/main.c") || strings.HasSuffix(p, "/main.cpp") || strings.HasSuffix(p, "/main.cc") {
		return hint("c-cpp", 3.0, "c-main")
	}
	if strings.Contains(p, "/src/") && (strings.HasSuffix(p, "/app.c") || strings.HasSuffix(p, "/app.cpp")) {
		return hint("c-cpp", 2.5, "c-app")
	}

	if strings.Contains(p, "/routes/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 3.0, "laravel-routes")
	}
	if (strings.Contains(p, "/http/controllers/") || strings.Contains(p, "/controllers/")) && strings.HasSuffix(p, ".php") {
		return hint("laravel", 3.0, "laravel-controller")
	}
	if strings.HasSuffix(p, "controller.php") {
		return hint("laravel", 3.0, "laravel-controller-file")
	}
	if (strings.Contains(p, "/console/commands/") || strings.Contains(p, "/commands/")) && strings.HasSuffix(p, ".php") {
		return hint("laravel", 2.5, "laravel-command")
	}
	if strings.Contains(p, "/jobs/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 2.5, "laravel-job")
	}
	if strings.Contains(p, "/listeners/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 2.5, "laravel-listener")
	}
	if strings.Contains(p, "/http/middleware/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 2.5, "laravel-middleware")
	}
	if strings.Contains(p, "/providers/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 1.8, "laravel-provider")
	}
	if strings.Contains(p, "/policies/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 2.0, "laravel-policy")
	}
	if strings.Contains(p, "/models/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 1.5, "laravel-model")
	}
	if strings.Contains(p, "/services/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 1.8, "laravel-service")
	}
	if strings.Contains(p, "/repositories/") && strings.HasSuffix(p, ".php") {
		return hint("laravel", 1.5, "laravel-repository")
	}

	if (strings.Contains(p, "/bin/") || strings.Contains(p, "/exe/")) && strings.HasSuffix(p, ".rb") {
		return hint("ruby", 2.5, "ruby-executable")
	}
	if strings.HasSuffix(p, "/rakefile") || strings.HasSuffix(p, ".rake") {
		return hint("ruby", 1.5, "ruby-rake")
	}

	if strings.HasSuffix(p, "/appdelegate.swift") || strings.HasSuffix(p, "/scenedelegate.swift") || strings.HasSuffix(p, "/app.swift") {
		return hint("ios", 3.0, "ios-app-entry")
	}
	if strings.HasSuffix(p, "app.swift") && strings.Contains(p, "/sources/") {
		return hint("swiftui", 3.0, "swiftui-app")
	}
	if (strings.Contains(p, "/viewcontrollers/") || strings.Contains(p, "/controllers/") || strings.Contains(p, "/screens/")) && strings.HasSuffix(p, ".swift") {
		return hint("uikit", 2.5, "uikit-viewcontroller")
	}
	if strings.HasSuffix(p, "viewcontroller.swift") || strings.HasSuffix(p, "vc.swift") {
		return hint("uikit", 2.5, "uikit-viewcontroller-file")
	}
	if strings.Contains(p, "/coordinators/") && strings.HasSuffix(p, ".swift") {
		return hint("ios-coordinator", 2.5, "ios-coordinator")
	}
	if strings.HasSuffix(p, "coordinator.swift") {
		return hint("ios-coordinator", 2.5, "ios-coordinator-file")
	}
	if (strings.Contains(p, "/views/") || strings.Contains(p, "/scenes/")) && strings.HasSuffix(p, ".swift") {
		return hint("swiftui", 1.8, "swiftui-view")
	}
	if strings.Contains(p, "/services/") && strings.HasSuffix(p, ".swift") {
		return hint("ios-service", 1.8, "ios-service")
	}
	if strings.Contains(p, "/router/") && strings.HasSuffix(p, ".swift") {
		return hint("ios-router", 2.0, "ios-router")
	}

	if strings.HasSuffix(p, "/main.dart") || strings.HasSuffix(p, "/app.dart") {
		return hint("flutter", 3.0, "flutter-main")
	}
	if (strings.Contains(p, "/screens/") || strings.Contains(p, "/pages/") || strings.Contains(p, "/views/")) && strings.HasSuffix(p, ".dart") {
		return hint("flutter", 2.5, "flutter-screen")
	}
	if strings.Contains(p, "/routes/") && strings.HasSuffix(p, ".dart") {
		return hint("flutter", 2.5, "flutter-routes")
	}
	if (strings.Contains(p, "/bloc/") || strings.Contains(p, "/controllers/") || strings.Contains(p, "/cubit/") || strings.Contains(p, "/presentation/")) && strings.HasSuffix(p, ".dart") {
		return hint("flutter", 2.0, "flutter-state-management")
	}
	if (strings.Contains(p, "/services/") || strings.Contains(p, "/domain/")) && strings.HasSuffix(p, ".dart") {
		return hint("flutter", 1.8, "flutter-service")
	}
	if strings.Contains(p, "/widgets/") && strings.HasSuffix(p, ".dart") {
		return hint("flutter", 1.5, "flutter-widget")
	}

	if strings.Contains(p, "/api/") && (strings.HasSuffix(p, "/index.ts") || strings.HasSuffix(p, "/index.js") || strings.HasSuffix(p, "/__init__.py")) {
		return hint("api", 1.8, "api-index")
	}
	return Hint{}, false
}

func hint(framework string, multiplier float64, reason string) (Hint, bool) {
	return Hint{Framework: framework, EntryPointMultiplier: multiplier, Reason: reason}, true
}

func hasAnySuffix(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func startsWithUpper(fileName string) bool {
	for _, r := range fileName {
		return unicode.IsUpper(r)
	}
	return false
}
