package cpp

import (
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type classContext struct {
	name       string
	kind       model.SymbolKind
	visibility string
	braceDepth int
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
