package report

import (
	"bytes"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestWriteCheckReportText_RendersCoupledSuspiciousEdgesAndNextCommands(t *testing.T) {
	var out bytes.Buffer

	WriteCheckReportText(&out, reportmodel.CheckReport{
		Root:           "/repo",
		OutDir:         "/repo/.patchcourt/out",
		StatePath:      "/repo/.patchcourt/state/latest/project-model.json",
		Summary:        model.ScanSummary{ProductionFiles: 2, TotalEdges: 16, Resolved: 16},
		FindingCount:   1,
		GraphNodeCount: 3,
		GraphEdgeCount: 2,
		Artifacts: []reportmodel.CheckArtifact{
			{
				Name: "project model",
				Path: "/repo/.patchcourt/out/project-model.json",
			},
			{
				Name: "layer graph dot",
				Path: "/repo/.patchcourt/out/layer-graph.dot",
			},
		},
		TopFindings: []reportmodel.FindingSummary{
			{
				ID:       "discovery.controllers.depends_on.server",
				Kind:     string(model.FindingKindDiscoveryHint),
				Severity: string(model.SeverityMedium),
				Title:    "Controller layer depends on server layer",
			},
		},
		MostCoupledEdges: []reportmodel.EdgeSummary{
			{
				From:  "controllers",
				To:    "domain",
				Count: 15,
			},
		},
		SuspiciousEdges: []reportmodel.EdgeSummary{
			{
				From:      "controllers",
				To:        "server",
				Count:     1,
				FindingID: "discovery.controllers.depends_on.server",
				Priority:  72,
			},
		},
		NextSteps: []reportmodel.NextStep{
			{
				Label:   "Inspect edge controllers -> server",
				Command: "patchcourt edge --model /repo/.patchcourt/out/project-model.json controllers server",
			},
			{
				Label:   "Explain finding discovery.controllers.depends_on.server",
				Command: "patchcourt explain discovery.controllers.depends_on.server --model /repo/.patchcourt/out/project-model.json",
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
