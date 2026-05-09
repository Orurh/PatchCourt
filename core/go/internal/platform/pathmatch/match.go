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

// IsTestLikeFile reports whether a normalized repository path looks like a test,
// mock, or test-support file.
//
// It is path-only and does not depend on analyzer/project so it can be reused
// from usecase and render packages without creating analyzer dependencies.
func IsTestLikeFile(filePath string) bool {
	normalized := strings.ToLower(Normalize(filePath))
	base := path.Base(normalized)

	if normalized == "" {
		return false
	}

	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case "test", "tests", "unit_test", "unit_tests", "integration_test", "integration_tests", "e2e", "e2e_tests", "mock", "mocks":
			return true
		}
	}

	if strings.Contains(base, "_test.") ||
		strings.Contains(base, "test_") ||
		strings.Contains(base, "_spec.") ||
		strings.Contains(base, "spec_") {
		return true
	}

	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.cc") ||
		strings.HasSuffix(base, "_test.cpp") ||
		strings.HasSuffix(base, "_test.cxx") ||
		strings.HasSuffix(base, "_test.h") ||
		strings.HasSuffix(base, "_test.hpp")
}
