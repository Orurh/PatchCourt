package app

import (
	"context"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/engine"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeCheckAnalysis struct {
	req engine.AnalyzeRequest
	res *engine.AnalyzeResult
	err error
}

func (f *fakeCheckAnalysis) Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error) {
	f.req = req
	return f.res, f.err
}

func TestApp_RunCheckBuildsResult(t *testing.T) {
	analysis := &fakeCheckAnalysis{
		res: &engine.AnalyzeResult{
			Project: &model.ProjectModel{
				Root: "/repo",
				Files: []model.FileModel{
					{Path: "src/server/api_router.cc", Layer: "server", Role: model.FileRoleProduction},
					{Path: "src/domain/status.h", Layer: "domain", Role: model.FileRoleProduction},
				},
				Dependencies: []model.DependencyEdge{
					{
						FromFile:  "src/server/api_router.cc",
						ToFile:    "src/domain/status.h",
						Target:    "src/domain/status.h",
						Kind:      model.DependencyKindInclude,
						Resolved:  true,
						FromLayer: "server",
						ToLayer:   "domain",
					},
				},
			},
			Config: &config.Config{},
		},
	}

	application := NewWithAnalysis(nil, analysis)

	result, err := application.RunCheck(context.Background(), CheckRequest{
		Root:       "/repo",
		ConfigPath: "/repo/.patchcourt.yaml",
		OutDir:     "/tmp/patchcourt-out",
	})
	require.NoError(t, err)

	require.Equal(t, "/repo", analysis.req.Root)
	require.Equal(t, "/repo/.patchcourt.yaml", analysis.req.ConfigPath)

	require.Equal(t, "/repo", result.Root)
	require.Equal(t, "/repo/.patchcourt.yaml", result.ConfigPath)
	require.Equal(t, "/tmp/patchcourt-out", result.OutDir)
	require.Equal(t, 2, result.Summary.ProductionFiles)
	require.Len(t, result.LayerGraph.Edges, 1)
	require.Empty(t, result.Artifacts)
}
