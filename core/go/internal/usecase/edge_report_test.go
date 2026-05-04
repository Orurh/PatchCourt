package usecase

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBuildEdgeReport_BuildsDependencyDrilldown(t *testing.T) {
	project := &model.ProjectModel{
		Root: "/repo",
		Files: []model.FileModel{
			{Path: "src/controllers/a.cc", Role: model.FileRoleProduction},
			{Path: "src/controllers/b.cc", Role: model.FileRoleProduction},
			{Path: "tests/controllers/a_test.cc", Role: model.FileRoleTest},
		},
		Dependencies: []model.DependencyEdge{
			edgeReportDep("src/controllers/a.cc", "src/domain/status.h", "controllers", "domain", model.DependencyUsageUsed),
			edgeReportDep("src/controllers/a.cc", "src/domain/settings.h", "controllers", "domain", model.DependencyUsageUnused),
			edgeReportDep("src/controllers/b.cc", "src/domain/status.h", "controllers", "domain", model.DependencyUsageMaybe),
			edgeReportDep("tests/controllers/a_test.cc", "src/domain/test_only.h", "controllers", "domain", model.DependencyUsageUsed),
			edgeReportDep("src/controllers/a.cc", "src/server/router.h", "controllers", "server", model.DependencyUsageUsed),
		},
		Findings: []model.Finding{
			{
				ID:       "discovery.controllers.depends_on.domain",
				Severity: model.SeverityMedium,
				Kind:     model.FindingKindDiscoveryHint,
				Title:    "Controller depends on domain",
				Evidence: []model.Evidence{
					{Message: "creating discovered layer dependency controllers -> domain"},
				},
			},
		},
	}

	report := BuildEdgeReport(project, EdgeReportOptions{
		Source:    "/tmp/project-model.json",
		FromLayer: "controllers",
		ToLayer:   "domain",
		Limit:     2,
	})

	require.Equal(t, "/repo", report.Root)
	require.Equal(t, "/tmp/project-model.json", report.Source)
	require.Equal(t, "controllers", report.FromLayer)
	require.Equal(t, "domain", report.ToLayer)
	require.Equal(t, 3, report.Count)
	require.Equal(t, 1, report.Usage.Used)
	require.Equal(t, 1, report.Usage.Maybe)
	require.Equal(t, 1, report.Usage.Unused)
	require.Len(t, report.Findings, 1)
	require.Equal(t, "discovery.controllers.depends_on.domain", report.Findings[0].ID)
	require.Len(t, report.Dependencies, 2)
	require.Equal(t, 1, report.TruncatedDeps)

	require.Equal(t, []EdgeFileCount{
		{File: "src/controllers/a.cc", Count: 2},
		{File: "src/controllers/b.cc", Count: 1},
	}, report.TopFromFiles)

	require.Equal(t, []EdgeFileCount{
		{File: "src/domain/status.h", Count: 2},
		{File: "src/domain/settings.h", Count: 1},
	}, report.TopToFiles)
}

func edgeReportDep(from string, to string, fromLayer string, toLayer string, usage model.DependencyUsage) model.DependencyEdge {
	return model.DependencyEdge{
		FromFile:  from,
		ToFile:    to,
		Target:    to,
		Kind:      model.DependencyKindInclude,
		Resolved:  true,
		FromLayer: fromLayer,
		ToLayer:   toLayer,
		Usage:     usage,
	}
}
