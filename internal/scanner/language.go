package scanner

import (
	"path"
	"strings"
)

type Language string

const (
	JavaScript  Language = "javascript"
	TypeScript  Language = "typescript"
	Python      Language = "python"
	Java        Language = "java"
	C           Language = "c"
	CPlusPlus   Language = "cpp"
	CSharp      Language = "csharp"
	Go          Language = "go"
	Ruby        Language = "ruby"
	Rust        Language = "rust"
	PHP         Language = "php"
	Kotlin      Language = "kotlin"
	Swift       Language = "swift"
	Dart        Language = "dart"
	Vue         Language = "vue"
	Svelte      Language = "svelte"
	Astro       Language = "astro"
	Cobol       Language = "cobol"
	Markdown    Language = "markdown"
	PDF         Language = "pdf"
	Word        Language = "word"
	Spreadsheet Language = "spreadsheet"
)

var extensionLanguage = map[string]Language{
	".js":       JavaScript,
	".jsx":      JavaScript,
	".mjs":      JavaScript,
	".cjs":      JavaScript,
	".ts":       TypeScript,
	".tsx":      TypeScript,
	".mts":      TypeScript,
	".cts":      TypeScript,
	".py":       Python,
	".java":     Java,
	".c":        C,
	".cpp":      CPlusPlus,
	".cc":       CPlusPlus,
	".cxx":      CPlusPlus,
	".h":        CPlusPlus,
	".hpp":      CPlusPlus,
	".hxx":      CPlusPlus,
	".hh":       CPlusPlus,
	".cs":       CSharp,
	".go":       Go,
	".rb":       Ruby,
	".rake":     Ruby,
	".gemspec":  Ruby,
	".rs":       Rust,
	".php":      PHP,
	".phtml":    PHP,
	".php3":     PHP,
	".php4":     PHP,
	".php5":     PHP,
	".php8":     PHP,
	".kt":       Kotlin,
	".kts":      Kotlin,
	".swift":    Swift,
	".dart":     Dart,
	".vue":      Vue,
	".svelte":   Svelte,
	".astro":    Astro,
	".cbl":      Cobol,
	".cob":      Cobol,
	".cpy":      Cobol,
	".cobol":    Cobol,
	".copybook": Cobol,
	".jcl":      Cobol,
	".job":      Cobol,
	".proc":     Cobol,
	".md":       Markdown,
	".mdx":      Markdown,
	".pdf":      PDF,
	".doc":      Word,
	".docx":     Word,
	".odt":      Word,
	".rtf":      Word,
	".xls":      Spreadsheet,
	".xlsx":     Spreadsheet,
	".xlsm":     Spreadsheet,
	".xlsb":     Spreadsheet,
	".xlt":      Spreadsheet,
	".xltx":     Spreadsheet,
	".xltm":     Spreadsheet,
	".xlam":     Spreadsheet,
	".ods":      Spreadsheet,
	".csv":      Spreadsheet,
	".tsv":      Spreadsheet,
}

var rubyExtensionless = map[string]struct{}{
	"Rakefile":    {},
	"Gemfile":     {},
	"Guardfile":   {},
	"Vagrantfile": {},
	"Brewfile":    {},
}

func DetectLanguage(filePath string) (Language, bool) {
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(filePath, "\\", "/")))
	if lang, ok := extensionLanguage[ext]; ok {
		return lang, true
	}
	if _, ok := rubyExtensionless[path.Base(filePath)]; ok {
		return Ruby, true
	}
	return "", false
}
