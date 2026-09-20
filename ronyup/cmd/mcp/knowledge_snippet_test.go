package mcp

import (
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
)

var (
	goFenceRE       = regexp.MustCompile("(?s)```go\\n(.*?)```")
	goPlaceholderRE = regexp.MustCompile(`\{\{\s*\.?[A-Za-z_][A-Za-z0-9_]*\s*\}\}`)
)

func TestKnowledgeGoSnippetsParse(t *testing.T) {
	t.Parallel()

	kb := mustLoadKB(t)
	docs := collectKnowledgeDocs(kb)

	var n int

	for _, doc := range docs {
		for i, raw := range extractGoFences(doc.body) {
			n++
			if skipGoSnippet(raw) {
				continue
			}

			src := normalizeGoSnippet(raw)
			if err := parseGoSnippet(src); err != nil {
				t.Errorf("%s fence %d: %v\n---\n%s\n---", doc.src, i+1, err, src)
			}
		}
	}

	if n == 0 {
		t.Fatal("expected at least one ```go fence in MCP knowledge")
	}
}

type knowledgeDoc struct {
	src  string
	body string
}

func collectKnowledgeDocs(kb *knowledge.Base) []knowledgeDoc {
	docs := []knowledgeDoc{
		{src: "server/instructions", body: kb.ServerInstructions},
	}

	for _, p := range kb.Prompts {
		docs = append(docs, knowledgeDoc{src: "prompts/" + p.Name, body: p.Template})
	}

	for _, p := range kb.Packages {
		docs = append(docs, knowledgeDoc{
			src:  "packages/" + p.ShortName,
			body: p.Description + "\n" + p.UsageHint,
		})
	}

	for _, a := range kb.ArchitectureHints {
		docs = append(docs, knowledgeDoc{src: "architecture/" + a.Slug, body: a.Text})
	}

	for _, c := range kb.Characteristics {
		docs = append(docs, knowledgeDoc{
			src:  "characteristics/" + c.Name,
			body: c.ServiceHint + "\n" + c.FileHint,
		})
	}

	for name, tool := range kb.Tools {
		docs = append(docs, knowledgeDoc{src: "tools/" + name, body: tool.ExtendedGuidance})
	}

	return docs
}

func extractGoFences(body string) []string {
	matches := goFenceRE.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))

	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}

	return out
}

func normalizeGoSnippet(src string) string {
	src = goPlaceholderRE.ReplaceAllString(src, "Example")
	src = strings.ReplaceAll(src, "\t", "    ")
	src = strings.TrimSpace(src)
	src = strings.TrimRight(src, ", \t\n")

	return src
}

func skipGoSnippet(src string) bool {
	return strings.TrimSpace(src) == "" || strings.Contains(src, "...")
}

func parseGoSnippet(src string) error {
	imports, rest := hoistImports(src)
	candidates := []string{
		withPackage(src),
		withPackage(imports + "func _() {\n" + rest + "\n}"),
		withPackage(wrapTrailingStatements(src)),
		withPackage(imports + wrapTrailingStatements(rest)),
	}

	fset := token.NewFileSet()

	var last error

	for _, cand := range candidates {
		_, err := parser.ParseFile(fset, "snippet.go", cand, parser.SkipObjectResolution)
		if err == nil {
			return nil
		}

		last = err
	}

	return last
}

func withPackage(src string) string {
	if strings.HasPrefix(strings.TrimSpace(src), "package ") {
		return src
	}

	return "package snippet\n\n" + src
}

func hoistImports(src string) (imports, rest string) {
	lines := strings.Split(src, "\n")
	importLines := make([]string, 0)
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" && len(importLines) > 0 {
			importLines = append(importLines, lines[i])
			i++

			continue
		}

		if !strings.HasPrefix(line, "import ") && line != "import (" {
			break
		}

		importLines = append(importLines, lines[i])
		if strings.HasPrefix(line, "import (") {
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != ")" {
				importLines = append(importLines, lines[i])
				i++
			}

			if i < len(lines) {
				importLines = append(importLines, lines[i])
			}
		}

		i++
	}

	if len(importLines) == 0 {
		return "", src
	}

	return strings.Join(importLines, "\n") + "\n\n", strings.TrimSpace(strings.Join(lines[i:], "\n"))
}

func wrapTrailingStatements(src string) string {
	end := lastDeclarationEnd(src)
	if end < 0 || end >= len(src) {
		return src
	}

	tail := strings.TrimSpace(src[end:])
	if tail == "" {
		return src
	}

	for _, prefix := range []string{"func ", "type ", "var ", "const ", "import "} {
		if strings.HasPrefix(tail, prefix) {
			return src
		}
	}

	return src[:end] + "\n\nfunc init() {\n" + tail + "\n}\n"
}

func lastDeclarationEnd(src string) int {
	lastStart := -1
	depth := 0

	for i := 0; i < len(src); i++ {
		if depth == 0 && isDeclStart(src, i) {
			lastStart = i
		}

		switch src[i] {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}

	if lastStart < 0 {
		return -1
	}

	depth = 0
	started := false

	for i := lastStart; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
			started = true
		case '}':
			if depth > 0 {
				depth--
			}

			if started && depth == 0 {
				return i + 1
			}
		}
	}

	// type aliases / no-brace decls end at the next blank line.
	if nl := strings.Index(src[lastStart:], "\n\n"); nl >= 0 {
		return lastStart + nl
	}

	return len(src)
}

func isDeclStart(src string, i int) bool {
	if i > 0 && !isBoundary(src[i-1]) {
		return false
	}

	for _, prefix := range []string{"func ", "type ", "var ", "const "} {
		if strings.HasPrefix(src[i:], prefix) {
			return true
		}
	}

	return false
}

func isBoundary(b byte) bool {
	return b == '\n' || b == '\t' || b == ' '
}
