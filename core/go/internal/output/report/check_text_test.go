package report

import (
	"bytes"
	"testing"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestWriteCheckText_RendersCoupledSuspiciousEdgesAndNextCommands(t *testing.T) {
	var out bytes.Buffer

	WriteCheckText(&out, CheckTextResult{
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
							Message: "includes src/server/mappers/lidar_status_mapper.h, creating discovered layer dependency controllers -> server",
						},
					},
				},
			},
		},
		LayerGraph: graphmodel.LayerGraph{
			Nodes: []string{"controllers", "domain", "server"},
			Edges: []graphmodel.LayerEdge{
				{
					From:  "controllers",
					To:    "domain",
					Count: 15,
				},
				{
					From:  "controllers",
					To:    "server",
					Count: 1,
				},
			},
		},
		Artifacts: []CheckTextArtifact{
			{
				Name: "project model",
				Path: "/repo/.patchcourt/out/project-model.json",
			},
			{
				Name: "layer graph dot",
				Path: "/repo/.patchcourt/out/layer-graph.dot",
			},
		},
	})

	got := out.String()

	require.Contains(t, got, "Most coupled edges:")
	require.Contains(t, got, "15  controllers -> domain")
	require.Contains(t, got, "Suspicious edges:")
	require.Contains(t, got, "1  controllers -> server  [discovery.controllers.depends_on.server]")
	require.Contains(t, got, "patchcourt edge --model /repo/.patchcourt/out/project-model.json controllers server")
	require.Contains(t, got, "patchcourt explain discovery.controllers.depends_on.server --model /repo/.patchcourt/out/project-model.json")
}

func TestEdgeFromEvidenceMessage_ParsesKnownEvidenceFormats(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		from string
		to   string
	}{
		{
			name: "discovered layer dependency",
			msg:  "includes src/server/foo.h, creating discovered layer dependency controllers -> server",
			from: "controllers",
			to:   "server",
		},
		{
			name: "include dependency",
			msg:  "includes src/cameras/sony.h, creating include dependency api -> cameras",
			from: "api",
			to:   "cameras",
		},
		{
			name: "layer dependency",
			msg:  "includes src/domain/status.h, creating layer dependency server -> domain [usage=used]",
			from: "server",
			to:   "domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, ok := edgeFromEvidenceMessage(tt.msg)

			require.True(t, ok)
			require.Equal(t, tt.from, from)
			require.Equal(t, tt.to, to)
		})
	}
}
