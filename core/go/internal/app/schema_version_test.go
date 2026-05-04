package app

import (
	"context"
	"encoding/json"
	"testing"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/stretchr/testify/require"
)

func TestBuildCheckReport_SetsSchemaVersion(t *testing.T) {
	report := BuildCheckReport(&CheckResult{
		Root: "/repo",
		Project: &model.ProjectModel{
			Root: "/repo",
		},
		LayerGraph: graphmodel.LayerGraph{},
	})

	require.Equal(t, reportmodel.CheckReportSchemaVersion, report.SchemaVersion)

	data, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(data), `"schema_version":"patchcourt.check.v1"`)
}

func TestBuildEdgeReport_SetsSchemaVersion(t *testing.T) {
	project := &model.ProjectModel{
		Root: "/repo",
	}

	result := BuildEdgeReport(project, EdgeReportOptions{
		Root:      "/repo",
		Source:    "model.json",
		FromLayer: "app",
		ToLayer:   "model",
	})

	require.Equal(t, reportmodel.EdgeResultSchemaVersion, result.SchemaVersion)
}

func TestRunReview_SetsSchemaVersion(t *testing.T) {
	root := t.TempDir()

	beforePath := writeProjectModelJSON(t, root, "before.json", model.ProjectModel{})
	afterPath := writeProjectModelJSON(t, root, "after.json", model.ProjectModel{})

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforePath: beforePath,
		AfterPath:  afterPath,
	})
	require.NoError(t, err)

	require.Equal(t, reportmodel.ReviewResultSchemaVersion, result.SchemaVersion)
}
