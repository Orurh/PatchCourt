package pathmatch

import (
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func Match(pattern string, filePath string) bool {
	pattern = Normalize(pattern)
	filePath = Normalize(filePath)

	if pattern == "" {
		return false
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if filePath == prefix {
			return true
		}
	}

	matched, err := doublestar.Match(pattern, filePath)
	return err == nil && matched
}

func MatchAny(patterns []string, filePath string) bool {
	for _, pattern := range patterns {
		if Match(pattern, filePath) {
			return true
		}
	}

	return false
}

// Normalize приводит путь к единому внутреннему формату PatchCourt.
//
// Внутренний формат:
//   - использует "/" как разделитель;
//   - убирает лишние "." и повторные разделители;
//   - сохраняет относительность пути;
//   - работает одинаково на Linux, macOS и Windows.
//
// Эта функция намеренно заменяет "\" вручную, потому что filepath.ToSlash()
// зависит от текущей ОС и на Linux не считает "\" разделителем пути.
func Normalize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "/")

	if value == "" {
		return ""
	}

	cleaned := path.Clean(value)
	if cleaned == "." {
		return ""
	}

	return cleaned
}
