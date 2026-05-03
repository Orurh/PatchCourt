package report

import (
	"bytes"
	"testing"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteCheckHTML_RendersSelfContainedReport(t *testing.T) {
	var out bytes.Buffer

	project := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:         "discovery.controllers.depends_on.server",
				Kind:       model.FindingKindDiscoveryHint,
				Severity:   model.SeverityMedium,
				Title:      "Controller layer depends on server layer",
				Confidence: model.ConfidenceMedium,
				Evidence: []model.Evidence{
					{
						File:      "src/controllers/device_orchestrator.cc",
						Message:   "includes src/server/mapper.h, creating discovered layer dependency controllers -> server",
						FromLayer: "controllers",
						ToLayer:   "server",
						FromFile:  "src/controllers/device_orchestrator.cc",
						ToFile:    "src/server/mapper.h",
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
			{
				FromFile:  "src/controllers/camera_manager_controller.cc",
				ToFile:    "src/server/api_router.h",
				Resolved:  true,
				FromLayer: "controllers",
				ToLayer:   "server",
				Usage:     model.DependencyUsageUsed,
			},
		},
	}

	layerGraph := graphmodel.LayerGraph{
		Nodes: []string{"controllers", "server"},
		Edges: []graphmodel.LayerEdge{
			{
				From:  "controllers",
				To:    "server",
				Count: 2,
			},
		},
	}

	err := WriteCheckHTML(&out, CheckHTMLInput{
		Report: reportmodel.CheckReport{
			Root:           "/repo",
			OutDir:         "/repo/.patchcourt/out",
			Summary:        model.ScanSummary{},
			FindingCount:   len(project.Findings),
			GraphNodeCount: len(layerGraph.Nodes),
			GraphEdgeCount: len(layerGraph.Edges),
		},
		Project:    project,
		LayerGraph: layerGraph,
	})

	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "<!doctype html>")
	require.Contains(t, got, "PatchCourt Report")
	require.Contains(t, got, "patchcourt-data")
	require.Contains(t, got, `"report"`)
	require.Contains(t, got, "controllers")
	require.Contains(t, got, "server")
	require.Contains(t, got, "discovery.controllers.depends_on.server")

	require.Contains(t, got, "edgeSearch")
	require.Contains(t, got, "onlyFindings")
	require.Contains(t, got, "onlyPolicy")
	require.Contains(t, got, "edgeMatchesFilters")
	require.Contains(t, got, "from_layer")
	require.Contains(t, got, "to_layer")
	require.Contains(t, got, "Evidence:")
	require.Contains(t, got, "edgeDiagram")
	require.Contains(t, got, "buildEdgeDiagramData")
	require.Contains(t, got, "Detailed file-level picture")
	require.Contains(t, got, "overviewGraph")
	require.Contains(t, got, "renderOverviewGraph")
	require.Contains(t, got, "overview-edge")
	require.Contains(t, got, "Layer graph")
}
