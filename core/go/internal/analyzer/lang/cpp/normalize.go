package cpp

import (
	"regexp"
	"strings"
)

var commentLineRE = regexp.MustCompile(`//.*$`)

func normalizeDeclarationLine(line string) string {
	line = commentLineRE.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "#") {
		return ""
	}

	return line
}
