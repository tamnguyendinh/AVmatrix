package sfc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	"github.com/tamnguyendinh/avmatrix-go/internal/providers/tsjs"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

type Request struct {
	FilePath string
	FileHash string
	Language scanner.Language
	Source   []byte
}

type Options struct {
	Name            string
	Language        scanner.Language
	ScriptExtractor func([]byte) ScriptBlock
}

type ScriptBlock struct {
	Language scanner.Language
	Source   string
}

func Extract(request Request, options Options) (scopeir.ScopeIR, error) {
	if request.Language != options.Language {
		return scopeir.ScopeIR{}, fmt.Errorf("%s extract: unsupported language %q", options.Name, request.Language)
	}
	script := options.ScriptExtractor(request.Source)
	if strings.TrimSpace(script.Source) == "" {
		return scopeir.ScopeIR{
			FilePath: request.FilePath,
			FileHash: request.FileHash,
			Language: options.Language,
		}.Normalized(), nil
	}

	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()
	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: request.FilePath,
		Language: script.Language,
		Source:   []byte(script.Source),
	})
	if err != nil {
		return scopeir.ScopeIR{}, err
	}
	defer parsed.Close()

	ir, err := tsjs.Extract(tsjs.Request{
		FilePath: request.FilePath,
		FileHash: request.FileHash,
		Language: script.Language,
		Source:   []byte(script.Source),
		Root:     parsed.Tree.RootNode(),
	})
	if err != nil {
		return scopeir.ScopeIR{}, err
	}
	ir.Language = options.Language
	return ir.Normalized(), nil
}

func ExtractHTMLScript(source []byte) ScriptBlock {
	raw := string(source)
	lower := strings.ToLower(raw)
	if setup := findHTMLScriptBlock(raw, lower, true); setup.Source != "" {
		return setup
	}
	return findHTMLScriptBlock(raw, lower, false)
}

func findHTMLScriptBlock(raw string, lower string, setupOnly bool) ScriptBlock {
	searchFrom := 0
	for {
		start := strings.Index(lower[searchFrom:], "<script")
		if start < 0 {
			return ScriptBlock{Language: scanner.TypeScript}
		}
		start += searchFrom
		tagEnd := strings.Index(lower[start:], ">")
		if tagEnd < 0 {
			return ScriptBlock{Language: scanner.TypeScript}
		}
		tagEnd += start
		closeStart := strings.Index(lower[tagEnd:], "</script>")
		if closeStart < 0 {
			return ScriptBlock{Language: scanner.TypeScript}
		}
		closeStart += tagEnd
		tag := lower[start : tagEnd+1]
		if setupOnly && !strings.Contains(tag, "setup") {
			searchFrom = closeStart + len("</script>")
			continue
		}
		if strings.Contains(tag, "src=") {
			searchFrom = closeStart + len("</script>")
			continue
		}
		content := raw[tagEnd+1 : closeStart]
		language := scanner.JavaScript
		if scriptTagIsTypeScript(tag) {
			language = scanner.TypeScript
		}
		padded := strings.Repeat("\n", strings.Count(raw[:tagEnd+1], "\n")) + content
		return ScriptBlock{Language: language, Source: padded}
	}
}

func ExtractTemplateComponents(source []byte) []string {
	raw := string(source)
	lower := strings.ToLower(raw)
	start := strings.Index(lower, "<template")
	if start < 0 {
		return nil
	}
	tagEnd := strings.Index(lower[start:], ">")
	if tagEnd < 0 {
		return nil
	}
	tagEnd += start
	closeStart := strings.Index(lower[tagEnd:], "</template>")
	if closeStart < 0 {
		return nil
	}
	closeStart += tagEnd
	template := raw[tagEnd+1 : closeStart]
	seen := map[string]struct{}{}
	out := []string{}
	for index := 0; index < len(template); index++ {
		if template[index] != '<' || index+1 >= len(template) || template[index+1] == '/' || template[index+1] == '!' {
			continue
		}
		nameStart := index + 1
		nameEnd := nameStart
		for nameEnd < len(template) && isTagNameByte(template[nameEnd]) {
			nameEnd++
		}
		name := template[nameStart:nameEnd]
		if !isComponentTagName(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func ExtractAstroFrontmatter(source []byte) ScriptBlock {
	raw := string(source)
	trimmed := strings.TrimLeft(raw, "\ufeff\r\n\t ")
	prefixLen := len(raw) - len(trimmed)
	if !strings.HasPrefix(trimmed, "---") {
		return ScriptBlock{Language: scanner.TypeScript}
	}
	start := prefixLen + len("---")
	if start < len(raw) && raw[start] == '\r' {
		start++
	}
	if start < len(raw) && raw[start] == '\n' {
		start++
	}
	closeRel := strings.Index(raw[start:], "\n---")
	if closeRel < 0 {
		return ScriptBlock{Language: scanner.TypeScript}
	}
	closeStart := start + closeRel
	content := raw[start:closeStart]
	padded := strings.Repeat("\n", strings.Count(raw[:start], "\n")) + content
	return ScriptBlock{Language: scanner.TypeScript, Source: padded}
}

func scriptTagIsTypeScript(tag string) bool {
	return strings.Contains(tag, `lang="ts"`) ||
		strings.Contains(tag, `lang='ts'`) ||
		strings.Contains(tag, "lang=ts") ||
		strings.Contains(tag, `type="text/typescript"`) ||
		strings.Contains(tag, `type='text/typescript'`)
}

func isTagNameByte(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' ||
		ch == '_' ||
		ch == '.'
}

func isComponentTagName(name string) bool {
	if name == "" {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}
