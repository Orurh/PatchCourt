package reviewquestions

import (
	"fmt"

	contracts "github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/platform/pathmatch"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const defaultLimit = 8

func Build(result reportmodel.ReviewResult) []string {
	questions := make([]string, 0)

	for _, item := range result.Impact.Worse {
		questions = append(questions, fmt.Sprintf(
			"How should we address the proven architecture problem `%s` (%s)?",
			item.Kind,
			impactQuestionDetail(item),
		))
	}

	for _, item := range result.Impact.NeedsReview {
		questions = append(questions, fmt.Sprintf(
			"Is `%s` intentional architecture change or accidental drift (%s)?",
			item.Kind,
			impactQuestionDetail(item),
		))
	}

	questions = append(questions, contractReviewQuestions(result.ContractChanges, result.ChangedFiles)...)

	if len(result.Impact.Worse) == 0 &&
		len(result.Impact.NeedsReview) == 0 &&
		len(result.ContractChanges) == 0 &&
		len(result.Impact.UnchangedDebt) > 0 {
		questions = append(questions, "No patch-specific architecture issue was proven. Should any existing debt be tracked separately from this patch?")
	}

	if len(questions) > defaultLimit {
		return questions[:defaultLimit]
	}

	return questions
}

func contractReviewQuestions(changes []contracts.SymbolChange, changedFiles []string) []string {
	if len(changes) == 0 {
		return nil
	}

	hasTests := hasTestLikeChangedFile(changedFiles)

	questions := make([]string, 0, len(changes))
	for _, change := range changes {
		if !isReviewRelevantContractChange(change.Kind) {
			continue
		}

		if hasTests {
			questions = append(questions, fmt.Sprintf(
				"Public contract changed `%s`, and test-like files changed in this patch. Verify callers and add or update tests if the compatibility or migration path is not covered.",
				change.SymbolKey,
			))
			continue
		}

		questions = append(questions, fmt.Sprintf(
			"Public contract changed `%s`, but no test-like files changed. Verify callers and add or update tests for compatibility or migration coverage.",
			change.SymbolKey,
		))
	}

	return questions
}

func isReviewRelevantContractChange(kind contracts.ChangeKind) bool {
	switch kind {
	case contracts.ChangeKindRemoved,
		contracts.ChangeKindSignatureChanged,
		contracts.ChangeKindModifiersChanged:
		return true
	default:
		return false
	}
}

func hasTestLikeChangedFile(files []string) bool {
	for _, file := range files {
		if pathmatch.IsTestLikeFile(file) {
			return true
		}
	}

	return false
}

func impactQuestionDetail(item reportmodel.ReviewImpactItem) string {
	if item.Detail != "" {
		return item.Detail
	}

	if item.ID != "" {
		return item.ID
	}

	if item.Title != "" {
		return item.Title
	}

	return "no detail"
}
