package reviewquestions

import (
	"testing"

	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestBuild_AsksAboutProvenImpact(t *testing.T) {
	questions := Build(reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			Worse: []reportmodel.ReviewImpactItem{
				{
					Kind:   "layer_edge_added",
					Title:  "Added forbidden layer dependency",
					Detail: "domain -> application",
				},
			},
		},
	})

	require.Len(t, questions, 1)
	require.Contains(t, questions[0], "proven architecture problem")
	require.Contains(t, questions[0], "domain -> application")
}

func TestBuild_AsksAboutNeedsReviewImpact(t *testing.T) {
	questions := Build(reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			NeedsReview: []reportmodel.ReviewImpactItem{
				{
					Kind:   "contract_delivery_impact",
					Title:  "Public contract changed with delivery/API impact",
					Detail: "method::ICameraManagerController::StartSession",
				},
			},
		},
	})

	require.Len(t, questions, 1)
	require.Contains(t, questions[0], "intentional architecture change or accidental drift")
	require.Contains(t, questions[0], "method::ICameraManagerController::StartSession")
}

func TestBuild_AsksAboutExistingDebtWhenPatchHasNoImpact(t *testing.T) {
	questions := Build(reportmodel.ReviewResult{
		Impact: reportmodel.ReviewImpactReport{
			UnchangedDebt: []reportmodel.ReviewImpactItem{
				{
					Kind:  "unchanged_finding",
					Title: "Existing runtime risk finding",
					ID:    "cpp.lifetime.this_capture_async",
				},
			},
		},
	})

	require.Len(t, questions, 1)
	require.Contains(t, questions[0], "No patch-specific architecture issue was proven")
}
