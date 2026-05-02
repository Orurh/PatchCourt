package cpp

import (
	"regexp"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

var (
	classRE   = regexp.MustCompile(`^\s*(?:template\s*<[^>]+>\s*)?(class|struct)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	enumRE    = regexp.MustCompile(`^\s*enum\s+(?:class\s+)?([A-Za-z_][A-Za-z0-9_]*)\b`)
	usingRE   = regexp.MustCompile(`^\s*using\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	typedefRE = regexp.MustCompile(`^\s*typedef\s+.+\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	friendRE  = regexp.MustCompile(`^\s*friend\s+(.+);`)
	methodRE  = regexp.MustCompile(`^\s*(?:virtual\s+)?[A-Za-z_][A-Za-z0-9_:<>,~*&\s]+\s+([A-Za-z_~][A-Za-z0-9_]*)\s*\([^;{}]*\)\s*(?:const\s*)?(?:noexcept\s*)?(?:override\s*)?(?:final\s*)?(?:=\s*0\s*)?;`)
)

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

func classOrStructKind(keyword string) model.SymbolKind {
	if keyword == "struct" {
		return model.SymbolKindStruct
	}

	return model.SymbolKindClass
}
