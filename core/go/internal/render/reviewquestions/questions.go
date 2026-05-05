package reviewquestions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

type Question struct {
	Text string
}

func Build(result reportmodel.ReviewResult, limit int) []Question {
	if limit <= 0 {
		return nil
	}

	questions := make([]Question, 0, limit)

	for _, item := range result.Impact.Worse {
		if len(questions) >= limit {
			return questions
		}

		text := fmt.Sprintf("Check whether this regression is intentional: %s", item.Title)
		if item.ID != "" {
			text += fmt.Sprintf(" `%s`", item.ID)
		}
		if item.Detail != "" {
			text += " — " + item.Detail
		}

		questions = append(questions, Question{Text: text})
	}

	for _, change := range result.ContractChanges {
		if len(questions) >= limit {
			return questions
		}

		switch change.Kind {
		case contracts.ChangeKindRemoved, contracts.ChangeKindSignatureChanged, contracts.ChangeKindModifiersChanged:
			if hasRelatedChangedTest(result.ChangedFiles, change) {
				questions = append(questions, Question{
					Text: fmt.Sprintf("Public contract changed `%s`; test-like files changed in this patch. Verify they actually cover this contract migration.", change.SymbolKey),
				})
			} else {
				questions = append(questions, Question{
					Text: fmt.Sprintf("Public contract changed `%s`, but no test-like files changed. Verify callers and add or update tests.", change.SymbolKey),
				})
			}
		}
	}

	if len(questions) == 0 {
		questions = append(questions, Question{
			Text: "No specific high-signal questions generated from the current facts.",
		})
	}

	return questions
}

func hasRelatedChangedTest(changedFiles []string, change contracts.SymbolChange) bool {
	candidates := contractFiles(change)
	if len(candidates) == 0 {
		return anyTestLikeFileChanged(changedFiles)
	}

	for _, changedFile := range changedFiles {
		if !isTestLikeFile(changedFile) {
			continue
		}

		changedBase := normalizedBaseName(changedFile)
		for _, candidate := range candidates {
			candidateBase := normalizedBaseName(candidate)
			if changedBase == candidateBase ||
				strings.Contains(changedBase, candidateBase) ||
				strings.Contains(candidateBase, changedBase) {
				return true
			}
		}
	}

	return false
}

func contractFiles(change contracts.SymbolChange) []string {
	seen := make(map[string]struct{})
	files := make([]string, 0, 2)

	add := func(file string) {
		if file == "" {
			return
		}
		if _, ok := seen[file]; ok {
			return
		}
		seen[file] = struct{}{}
		files = append(files, file)
	}

	if change.Before != nil {
		add(change.Before.File)
	}
	if change.After != nil {
		add(change.After.File)
	}

	return files
}

func anyTestLikeFileChanged(changedFiles []string) bool {
	for _, file := range changedFiles {
		if isTestLikeFile(file) {
			return true
		}
	}

	return false
}

func isTestLikeFile(file string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(file, "\\", "/"))
	base := filepath.Base(normalized)

	if strings.Contains(normalized, "/test/") ||
		strings.Contains(normalized, "/tests/") ||
		strings.Contains(normalized, "/mocks/") ||
		strings.Contains(normalized, "/mock/") {
		return true
	}

	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.cc") ||
		strings.HasSuffix(base, "_test.cpp") ||
		strings.HasSuffix(base, "_test.cxx") ||
		strings.HasSuffix(base, "_test.h") ||
		strings.HasSuffix(base, "_test.hpp")
}

func normalizedBaseName(file string) string {
	base := strings.ToLower(filepath.Base(strings.ReplaceAll(file, "\\", "/")))
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.TrimSuffix(base, "_test")
	base = strings.TrimPrefix(base, "test_")
	base = strings.TrimPrefix(base, "mock_")
	base = strings.TrimSuffix(base, "_mock")
	return base
}
