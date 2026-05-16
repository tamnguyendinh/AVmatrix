package parser

import (
	"fmt"
	"path"
	"strings"

	tree_sitter_dart "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	tree_sitter_swift "github.com/flamingoosesoftwareinc/tree-sitter-swift/bindings/go"
	tree_sitter_kotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_c_sharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

type Grammar struct {
	Key      string
	Name     string
	Language scanner.Language
	build    func() *sitter.Language
}

type Registry struct {
	grammars map[scanner.Language]Grammar
	tsx      Grammar
}

func DefaultRegistry() *Registry {
	return &Registry{
		grammars: map[scanner.Language]Grammar{
			scanner.JavaScript: {
				Key:      "javascript",
				Name:     "JavaScript",
				Language: scanner.JavaScript,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_javascript.Language()) },
			},
			scanner.TypeScript: {
				Key:      "typescript",
				Name:     "TypeScript",
				Language: scanner.TypeScript,
				build: func() *sitter.Language {
					return sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
				},
			},
			scanner.Go: {
				Key:      "go",
				Name:     "Go",
				Language: scanner.Go,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_go.Language()) },
			},
			scanner.Python: {
				Key:      "python",
				Name:     "Python",
				Language: scanner.Python,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_python.Language()) },
			},
			scanner.Java: {
				Key:      "java",
				Name:     "Java",
				Language: scanner.Java,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_java.Language()) },
			},
			scanner.Kotlin: {
				Key:      "kotlin",
				Name:     "Kotlin",
				Language: scanner.Kotlin,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_kotlin.Language()) },
			},
			scanner.C: {
				Key:      "c",
				Name:     "C",
				Language: scanner.C,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_c.Language()) },
			},
			scanner.CSharp: {
				Key:      "csharp",
				Name:     "C#",
				Language: scanner.CSharp,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_c_sharp.Language()) },
			},
			scanner.CPlusPlus: {
				Key:      "cpp",
				Name:     "C++",
				Language: scanner.CPlusPlus,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_cpp.Language()) },
			},
			scanner.Rust: {
				Key:      "rust",
				Name:     "Rust",
				Language: scanner.Rust,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_rust.Language()) },
			},
			scanner.PHP: {
				Key:      "php",
				Name:     "PHP",
				Language: scanner.PHP,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_php.LanguagePHP()) },
			},
			scanner.Dart: {
				Key:      "dart",
				Name:     "Dart",
				Language: scanner.Dart,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_dart.Language()) },
			},
			scanner.Swift: {
				Key:      "swift",
				Name:     "Swift",
				Language: scanner.Swift,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_swift.Language()) },
			},
			scanner.Ruby: {
				Key:      "ruby",
				Name:     "Ruby",
				Language: scanner.Ruby,
				build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_ruby.Language()) },
			},
		},
		tsx: Grammar{
			Key:      "tsx",
			Name:     "TSX",
			Language: scanner.TypeScript,
			build:    func() *sitter.Language { return sitter.NewLanguage(tree_sitter_typescript.LanguageTSX()) },
		},
	}
}

func (r *Registry) Resolve(language scanner.Language, filePath string) (Grammar, error) {
	if r == nil {
		r = DefaultRegistry()
	}
	if language == scanner.TypeScript && strings.EqualFold(path.Ext(filepathSlash(filePath)), ".tsx") {
		return r.tsx, nil
	}
	grammar, ok := r.grammars[language]
	if !ok {
		return Grammar{}, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, language)
	}
	return grammar, nil
}

func filepathSlash(filePath string) string {
	return strings.ReplaceAll(filePath, "\\", "/")
}
