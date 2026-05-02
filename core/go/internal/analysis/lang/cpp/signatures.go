package cpp

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

var (
	classRE     = regexp.MustCompile(`^\s*(?:template\s*<[^>]+>\s*)?(class|struct)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	enumRE      = regexp.MustCompile(`^\s*enum\s+(?:class\s+)?([A-Za-z_][A-Za-z0-9_]*)\b`)
	usingRE     = regexp.MustCompile(`^\s*using\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	typedefRE   = regexp.MustCompile(`^\s*typedef\s+.+\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	friendRE    = regexp.MustCompile(`^\s*friend\s+(.+);`)
	methodRE    = regexp.MustCompile(`^\s*(?:virtual\s+)?[A-Za-z_][A-Za-z0-9_:<>,~*&\s]+\s+([A-Za-z_~][A-Za-z0-9_]*)\s*\([^;{}]*\)\s*(?:const\s*)?(?:noexcept\s*)?(?:override\s*)?(?:final\s*)?(?:=\s*0\s*)?;`)
	commentLine = regexp.MustCompile(`//.*$`)
)

type DeclaredSymbol struct {
	Name       string
	Kind       model.SymbolKind
	Parent     string
	Signature  string
	Modifiers  []string
	Exported   bool
	Visibility string
	Confidence model.Confidence
}

type classContext struct {
	name       string
	kind       model.SymbolKind
	visibility string
	braceDepth int
}

func ExtractDeclaredSymbols(path string) ([]DeclaredSymbol, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var symbols []DeclaredSymbol
	var currentClass *classContext

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := normalizeDeclarationLine(scanner.Text())
		if line == "" {
			continue
		}

		if currentClass != nil {
			if visibility, ok := parseVisibilityLabel(line); ok {
				currentClass.visibility = visibility
				currentClass.braceDepth += braceDelta(line)
				if currentClass.braceDepth <= 0 {
					currentClass = nil
				}
				continue
			}

			if symbol, ok := extractFriendFromLine(line, currentClass.name, currentClass.visibility); ok {
				symbols = append(symbols, symbol)
			}

			if currentClass.visibility == "public" {
				if symbol, ok := extractMethodFromLine(line, currentClass.name); ok {
					symbols = append(symbols, symbol)
				}
			}
		}

		if symbol, ok := extractDeclaredSymbolFromLine(line); ok {
			symbols = append(symbols, symbol)

			if symbol.Kind == model.SymbolKindClass || symbol.Kind == model.SymbolKindStruct {
				if strings.Contains(line, "{") {
					currentClass = &classContext{
						name:       symbol.Name,
						kind:       symbol.Kind,
						visibility: defaultVisibility(symbol.Kind),
						braceDepth: braceDelta(line),
					}

					if currentClass.braceDepth <= 0 {
						currentClass = nil
					}
				}
			}
		}

		if currentClass != nil {
			currentClass.braceDepth += braceDeltaWithoutOpeningClassLine(line)
			if currentClass.braceDepth <= 0 {
				currentClass = nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}

func extractDeclaredSymbolFromLine(line string) (DeclaredSymbol, bool) {
	if match := classRE.FindStringSubmatch(line); len(match) == 3 {
		return DeclaredSymbol{
			Name:       match[2],
			Kind:       classOrStructKind(match[1]),
			Signature:  line,
			Exported:   true,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := enumRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindEnum,
			Signature:  line,
			Exported:   true,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := usingRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindUsing,
			Signature:  line,
			Exported:   true,
			Confidence: model.ConfidenceMedium,
		}, true
	}

	if match := typedefRE.FindStringSubmatch(line); len(match) == 2 {
		return DeclaredSymbol{
			Name:       match[1],
			Kind:       model.SymbolKindTypedef,
			Signature:  line,
			Exported:   true,
			Confidence: model.ConfidenceLow,
		}, true
	}

	return DeclaredSymbol{}, false
}

func extractMethodFromLine(line string, parent string) (DeclaredSymbol, bool) {
	match := methodRE.FindStringSubmatch(line)
	if len(match) != 2 {
		return DeclaredSymbol{}, false
	}

	name := match[1]
	if strings.HasPrefix(name, "~") {
		return DeclaredSymbol{}, false
	}

	return DeclaredSymbol{
		Name:       name,
		Kind:       model.SymbolKindMethod,
		Parent:     parent,
		Signature:  line,
		Modifiers:  methodModifiers(line),
		Exported:   true,
		Visibility: "public",
		Confidence: model.ConfidenceLow,
	}, true
}

func extractFriendFromLine(line string, parent string, visibility string) (DeclaredSymbol, bool) {
	match := friendRE.FindStringSubmatch(line)
	if len(match) != 2 {
		return DeclaredSymbol{}, false
	}

	signature := strings.TrimSpace(line)
	name := friendName(match[1])

	return DeclaredSymbol{
		Name:       name,
		Kind:       model.SymbolKindFriend,
		Parent:     parent,
		Signature:  signature,
		Exported:   true,
		Visibility: visibility,
		Confidence: model.ConfidenceLow,
	}, true
}

func friendName(decl string) string {
	decl = strings.TrimSpace(decl)

	if strings.HasPrefix(decl, "class ") {
		return strings.TrimSpace(strings.TrimPrefix(decl, "class "))
	}

	if strings.HasPrefix(decl, "struct ") {
		return strings.TrimSpace(strings.TrimPrefix(decl, "struct "))
	}

	openParen := strings.Index(decl, "(")
	if openParen == -1 {
		fields := strings.Fields(decl)
		if len(fields) == 0 {
			return decl
		}

		return strings.Trim(fields[len(fields)-1], "*&")
	}

	beforeParen := strings.TrimSpace(decl[:openParen])
	fields := strings.Fields(beforeParen)
	if len(fields) == 0 {
		return decl
	}

	return strings.Trim(fields[len(fields)-1], "*&")
}

func methodModifiers(line string) []string {
	modifiers := make([]string, 0, 6)

	if strings.Contains(line, "virtual ") {
		modifiers = append(modifiers, "virtual")
	}

	if strings.Contains(line, ") const") {
		modifiers = append(modifiers, "const")
	}

	if strings.Contains(line, "noexcept") {
		modifiers = append(modifiers, "noexcept")
	}

	if strings.Contains(line, "override") {
		modifiers = append(modifiers, "override")
	}

	if strings.Contains(line, "final") {
		modifiers = append(modifiers, "final")
	}

	if strings.Contains(line, "= 0") || strings.Contains(line, "=0") {
		modifiers = append(modifiers, "pure_virtual")
	}

	return modifiers
}

func classOrStructKind(keyword string) model.SymbolKind {
	if keyword == "struct" {
		return model.SymbolKindStruct
	}

	return model.SymbolKindClass
}

func defaultVisibility(kind model.SymbolKind) string {
	if kind == model.SymbolKindStruct {
		return "public"
	}

	return "private"
}

func parseVisibilityLabel(line string) (string, bool) {
	switch strings.TrimSpace(line) {
	case "public:":
		return "public", true
	case "protected:":
		return "protected", true
	case "private:":
		return "private", true
	default:
		return "", false
	}
}

func braceDelta(line string) int {
	return strings.Count(line, "{") - strings.Count(line, "}")
}

func braceDeltaWithoutOpeningClassLine(line string) int {
	if classRE.MatchString(line) {
		return 0
	}

	return braceDelta(line)
}

func normalizeDeclarationLine(line string) string {
	line = commentLine.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "#") {
		return ""
	}

	return line
}
