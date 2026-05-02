package cpp

import "strings"

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
