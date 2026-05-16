package scanner

import (
	"os"
	"sort"
	"strings"
)

var registryPrimaryCodeLanguages = []Language{
	JavaScript,
	TypeScript,
	Python,
	Java,
	C,
	CPlusPlus,
	CSharp,
	Go,
	Ruby,
	Rust,
	PHP,
	Kotlin,
	Swift,
	Dart,
	Vue,
	Cobol,
}

func RegistryPrimaryCodeLanguages() []Language {
	out := append([]Language(nil), registryPrimaryCodeLanguages...)
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func RegistryPrimaryEnvVarName(language Language) string {
	return "REGISTRY_PRIMARY_" + strings.ToUpper(string(language))
}

func IsRegistryPrimary(language Language) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(RegistryPrimaryEnvVarName(language))))
	switch value {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func PrimaryLanguages() map[Language]struct{} {
	enabled := make(map[Language]struct{})
	for _, language := range registryPrimaryCodeLanguages {
		if IsRegistryPrimary(language) {
			enabled[language] = struct{}{}
		}
	}
	return enabled
}
