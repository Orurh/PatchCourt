package report

import (
	"bytes"
	"testing"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestWriteCheckHTML_RendersSelfContainedReport(t *testing.T) {
	var out bytes.Buffer

	err := WriteCheckHTML(&out, CheckTextResult{
		Root:   "/repo",
		OutDir: "/repo/.patchcourt/out",
		Project: &model.ProjectModel{
			Findings: []model.Finding{
				{
					ID:       "discovery.controllers.depends_on.server",
					Kind:     model.FindingKindDiscoveryHint,
					Severity: model.SeverityMedium,
					Title:    "Controller layer depends on server layer",
					Evidence: []model.Evidence{
						{
							File:    "src/controllers/device_orchestrator.cc",
							Message: "includes src/server/mapper.h, creating discovered layer dependency controllers -> server",
						},
					},
				},
			},
			Dependencies: []model.DependencyEdge{
				{
					FromFile:  "src/controllers/device_orchestrator.cc",
					ToFile:    "src/server/mapper.h",
					Resolved:  true,
					FromLayer: "controllers",
					ToLayer:   "server",
					Usage:     model.DependencyUsageUsed,
				},
			},
		},
		LayerGraph: graphmodel.LayerGraph{
			Nodes: []string{"controllers", "server"},
			Edges: []graphmodel.LayerEdge{
				{
					From:  "controllers",
					To:    "server",
					Count: 1,
				},
			},
		},
	})

	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "<!doctype html>")
	require.Contains(t, got, "PatchCourt Report")
	require.Contains(t, got, "patchcourt-data")
	require.Contains(t, got, "controllers")
	require.Contains(t, got, "server")
	require.Contains(t, got, "discovery.controllers.depends_on.server")
}
