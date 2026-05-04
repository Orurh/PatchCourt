package report

import (
	"bytes"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteReviewMarkdown_RendersSummaryRiskAndFinding(t *testing.T) {
	var out bytes.Buffer

	WriteReviewMarkdown(&out, ReviewMarkdownResult{
		Summary: reportmodel.ReviewSummary{
			DependencyChanges:   1,
			LayerEdgeChanges:    1,
			FindingChanges:      1,
			AddedFindings:       1,
			AddedHighFindings:   1,
			AddedPolicyFindings: 1,
		},
		Risk: risk.Score{
			Points: 11,
			Level:  risk.LevelHigh,
			Reasons: []risk.Reason{
				{
					Points:  7,
					Message: "dependency edge added: include|src/server/api_router.cc|src/cameras/sony.h",
				},
			},
		},
		FindingChanges: []findingdiff.FindingChange{
			{
				Kind: findingdiff.FindingChangeKindAdded,
				ID:   "architecture.api.cameras",
				After: &model.Finding{
					ID:         "architecture.api.cameras",
					Kind:       model.FindingKindPolicyViolation,
					Severity:   model.SeverityHigh,
					Confidence: model.ConfidenceHigh,
					Title:      "Include-level architecture boundary violation",
					Risk:       "Layer api includes cameras.",
					Suggestion: "Move dependency behind an interface.",
					Evidence: []model.Evidence{
						{
							File:    "src/server/api_router.cc",
							Message: "includes src/cameras/sony.h",
						},
					},
				},
			},
		},
		LayerEdgeChanges: []depdiff.LayerEdgeChange{
			{
				Kind:       depdiff.DependencyChangeKindAdded,
				FromLayer:  "api",
				ToLayer:    "cameras",
				AfterCount: 1,
			},
		},
		AfterRoot:  "/repo/after",
		ConfigPath: "/repo/.patchcourt.yaml",
	})

	got := out.String()

	require.Contains(t, got, "# PatchCourt Review")
	require.Contains(t, got, "- **Risk:** `high`, **11** points")
	require.Contains(t, got, "## Risk reasons")
	require.Contains(t, got, "**+7** dependency edge added: include|src/server/api_router.cc|src/cameras/sony.h")
	require.NotContains(t, got, "include\\|")
	require.Contains(t, got, "### `architecture.api.cameras` `added`")
	require.Contains(t, got, "- `src/server/api_router.cc`: includes src/cameras/sony.h")
	require.Contains(t, got, "patchcourt explain architecture.api.cameras --root /repo/after --config /repo/.patchcourt.yaml")
	require.Contains(t, got, "| `added` | `api -> cameras` | 0 | 1 |")
}
