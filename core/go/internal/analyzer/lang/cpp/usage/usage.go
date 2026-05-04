package usage

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

var stringLiteralRE = regexp.MustCompile(`"([^"\\]|\\.)*"|'([^'\\]|\\.)*'`)

// Analyze enriches resolved C++ include edges with lightweight symbol-usage
// information.
//
// This is syntactic and intentionally conservative. It does not try to be a C++
// compiler or AST analyzer. It answers only:
//
//   - does the includer text mention symbols declared in the included file?
//   - if not, the include may be unused
//
// For macro-heavy headers, template-only headers, transitive includes, and
// generated code this can be imprecise, so findings built on top of this should
// remain hints, not hard errors.
func Analyze(project *model.ProjectModel) {
	if project == nil {
		return
	}

	fileSymbols := symbolsByFile(project)
	contentCache := make(map[string]string)

	for i := range project.Dependencies {
		dep := &project.Dependencies[i]

		if dep.Kind != model.DependencyKindInclude {
			continue
		}

		if dep.External || !dep.Resolved || dep.ToFile == "" {
			continue
		}

		symbols := fileSymbols[dep.ToFile]
		if len(symbols) == 0 {
			dep.Usage = model.DependencyUsageMaybe
			continue
		}

		content, ok := contentCache[dep.FromFile]
		if !ok {
			data, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(dep.FromFile)))
			if err != nil {
				dep.Usage = model.DependencyUsageUnknown
				continue
			}

			content = normalizeContentForUsage(string(data))
			contentCache[dep.FromFile] = content
		}

		if usesAnySymbol(content, symbols) {
			dep.Usage = model.DependencyUsageUsed
		} else {
			dep.Usage = model.DependencyUsageUnused
		}
	}
}

func symbolsByFile(project *model.ProjectModel) map[string][]string {
	result := make(map[string][]string)

	for _, symbol := range project.Symbols {
		if symbol.File == "" || symbol.Name == "" {
			continue
		}

		switch symbol.Kind {
		case model.SymbolKindClass,
			model.SymbolKindStruct,
			model.SymbolKindEnum,
			model.SymbolKindUsing,
			model.SymbolKindTypedef:
			result[symbol.File] = append(result[symbol.File], symbol.Name)
		}
	}

	for file := range result {
		result[file] = uniqueSorted(result[file])
	}

	return result
}

func normalizeContentForUsage(content string) string {
	content = stripComments(content)
	content = stripIncludeLines(content)
	content = stringLiteralRE.ReplaceAllString(content, " ")
	return content
}

func stripIncludeLines(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#include") {
			lines[i] = ""
		}
	}

	return strings.Join(lines, "\n")
}

func stripComments(content string) string {
	var b strings.Builder

	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				b.WriteByte('\n')
			}
			continue
		}

		if inBlockComment {
			if ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if ch == '/' && i+1 < len(content) && content[i+1] == '/' {
			inLineComment = true
			i++
			continue
		}

		if ch == '/' && i+1 < len(content) && content[i+1] == '*' {
			inBlockComment = true
			i++
			continue
		}

		b.WriteByte(ch)
	}

	return b.String()
}

func usesAnySymbol(content string, symbols []string) bool {
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}

		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
		if re.FindStringIndex(content) != nil {
			return true
		}
	}

	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Strings(result)
	return result
}
