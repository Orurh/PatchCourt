package usecase

import (
	"context"
	"testing"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/stretchr/testify/require"
)

func TestApp_RunExplainFindsFindingFromAnalysisResult(t *testing.T) {
	fakeAnalysis := &fakeAnalysisService{
		result: &engine.AnalyzeResult{
			Project: &model.ProjectModel{
				Root: "/repo",
				Findings: []model.Finding{
					{
						ID:         "architecture.api.cameras",
						Kind:       model.FindingKindPolicyViolation,
						Severity:   model.SeverityHigh,
						Title:      "Include-level architecture boundary violation",
						Confidence: model.ConfidenceHigh,
					},
				},
			},
			Config: &config.Config{},
		},
	}

	application := NewWithAnalysis(logx.Nop(), fakeAnalysis)

	result, err := application.RunExplain(context.Background(), ExplainRequest{
		FindingID:  "architecture.api.cameras",
		Root:       "/repo",
		ConfigPath: "/repo/.patchcourt.yaml",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "architecture.api.cameras", result.Finding.ID)
	require.Equal(t, "/repo", result.Source)
	require.Equal(t, "explain", fakeAnalysis.req.Operation)
	require.Equal(t, "/repo", fakeAnalysis.req.Root)
	require.Equal(t, "/repo/.patchcourt.yaml", fakeAnalysis.req.ConfigPath)
}

func TestApp_RunExplainFindsFindingFromProjectModelJSON(t *testing.T) {
	root := t.TempDir()

	project := model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:         "discovery.cpp.unused_includes",
				Kind:       model.FindingKindDiscoveryHint,
				Severity:   model.SeverityLow,
				Title:      "Possibly unused C++ include",
				Confidence: model.ConfidenceMedium,
			},
		},
	}

	modelPath := writeProjectModelJSON(t, root, "model.json", project)

	result, err := New(nil).RunExplain(context.Background(), ExplainRequest{
		FindingID: "discovery.cpp.unused_includes",
		ModelPath: modelPath,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "discovery.cpp.unused_includes", result.Finding.ID)
	require.Equal(t, modelPath, result.Source)
}

func TestApp_RunExplainReturnsErrorWhenFindingIsMissing(t *testing.T) {
	root := t.TempDir()

	modelPath := writeProjectModelJSON(t, root, "model.json", model.ProjectModel{})

	result, err := New(nil).RunExplain(context.Background(), ExplainRequest{
		FindingID: "missing.finding",
		ModelPath: modelPath,
	})

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), `finding "missing.finding" was not found`)
}

func TestApp_RunExplainRequiresFindingID(t *testing.T) {
	result, err := New(nil).RunExplain(context.Background(), ExplainRequest{})

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "finding id is required")
}
