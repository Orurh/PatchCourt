package project

import (
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
)

func DetectLanguage(path string) model.Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return model.LanguageGo
	case ".h", ".hh", ".hpp", ".hxx", ".c", ".cc", ".cpp", ".cxx":
		return model.LanguageCPP
	default:
		return model.LanguageUnknown
	}
}

func DetectFileKind(path string, lang model.Language) model.FileKind {
	if IsTestFile(path) {
		return model.FileKindTest
	}

	ext := strings.ToLower(filepath.Ext(path))

	if lang == model.LanguageCPP {
		switch ext {
		case ".h", ".hh", ".hpp", ".hxx":
			return model.FileKindHeader
		case ".c", ".cc", ".cpp", ".cxx":
			return model.FileKindSource
		}
	}

	return model.FileKindUnknown
}

func DetectFileRole(path string, lang model.Language) model.FileRole {
	if IsTestFile(path) {
		return model.FileRoleTest
	}

	if IsGeneratedFile(path) {
		return model.FileRoleGenerated
	}

	if IsExternalFile(path) {
		return model.FileRoleExternal
	}

	if IsConfigFile(path) {
		return model.FileRoleConfig
	}

	switch lang {
	case model.LanguageCPP, model.LanguageGo:
		return model.FileRoleProduction
	default:
		return model.FileRoleUnknown
	}
}

func IsTestFile(path string) bool {
	normalized := strings.ToLower(pathmatch.Normalize(path))
	base := filepath.Base(normalized)

	if strings.Contains(base, "_test.") ||
		strings.Contains(base, "test_") ||
		strings.Contains(base, "_spec.") ||
		strings.Contains(base, "spec_") {
		return true
	}

	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case "test", "tests", "unit_test", "unit_tests", "integration_test", "integration_tests", "e2e", "e2e_tests":
			return true
		}
	}

	return false
}

func IsGeneratedFile(path string) bool {
	normalized := strings.ToLower(pathmatch.Normalize(path))
	base := filepath.Base(normalized)

	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case "generated", "gen":
			return true
		}
	}

	if strings.Contains(base, ".generated.") ||
		strings.Contains(base, "_generated.") ||
		strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, ".pb.cc") ||
		strings.HasSuffix(base, ".pb.cpp") ||
		strings.HasSuffix(base, ".pb.h") ||
		strings.HasSuffix(base, ".grpc.pb.go") ||
		strings.HasSuffix(base, ".grpc.pb.cc") ||
		strings.HasSuffix(base, ".grpc.pb.h") {
		return true
	}

	return false
}

func IsExternalFile(path string) bool {
	normalized := strings.ToLower(pathmatch.Normalize(path))

	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case "third_party", "3rdparty", "external", "vendor", "deps", "contrib":
			return true
		}
	}

	return false
}

func IsConfigFile(path string) bool {
	normalized := strings.ToLower(pathmatch.Normalize(path))
	base := filepath.Base(normalized)

	switch base {
	case "cmakelists.txt", "go.mod", "go.sum", "compile_commands.json", ".patchcourt.yaml", ".patchcourt.yml":
		return true
	}

	switch filepath.Ext(base) {
	case ".yaml", ".yml", ".json", ".toml", ".ini":
		return true
	default:
		return false
	}
}
