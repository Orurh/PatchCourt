package cpp

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

var (
	classRE     = regexp.MustCompile(`^\s*(?:template\s*<[^>]+>\s*)?(?:class|struct)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	enumRE      = regexp.MustCompile(`^\s*enum\s+(?:class\s+)?([A-Za-z_][A-Za-z0-9_]*)\b`)
	usingRE     = regexp.MustCompile(`^\s*using\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	typedefRE   = regexp.MustCompile(`^\s*typedef\s+.+\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	commentLine = regexp.MustCompile(`//.*$`)
)

type DeclaredSymbol struct {
	Name       string
	Kind       model.SymbolKind
	Signature  string
	Confidence model.Confidence
}

func ExtractDeclaredSymbols(path string) ([]DeclaredSymbol, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var symbols []DeclaredSymbol

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := normalizeDeclarationLine(scanner.Text())
		if line == "" {
			continue
		}

		if symbol, ok := extractDeclaredSymbolFromLine(line); ok {
			symbols = append(symbols, symbol)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}

func extractDeclaredSymbolFromLine(line string) (DeclaredSymbol, bool) {
	if match := classRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       classOrStructKind(line),
			Signature:  line,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := enumRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindEnum,
			Signature:  line,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := usingRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindUsing,
			Signature:  line,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := typedefRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindTypedef,
			Signature:  line,
			Confidence: model.ConfidenceLow,
		}, true
	}

	return DeclaredSymbol{}, false
}

func classOrStructKind(line string) model.SymbolKind {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "template")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "struct ") || strings.Contains(line, "> struct ") {
		return model.SymbolKindStruct
	}

	return model.SymbolKindClass
}

func normalizeDeclarationLine(line string) string {
	line = commentLine.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "#") {
		return ""
	}

	return line
}
