package suppressions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCollect_FindsIgnoreDirective(t *testing.T) {
	root := t.TempDir()

	writeSuppressionTestFile(t, root, "src/server/api_router.cc", `// patchcourt:ignore architecture.api.cameras reason: legacy boundary
#include "src/cameras/sony.h"
`)

	got, err := Collect(root, []model.FileModel{
		{
			Path:     "src/server/api_router.cc",
			Language: model.LanguageCPP,
			Role:     model.FileRoleProduction,
		},
	})

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "architecture.api.cameras", got[0].ID)
	require.Equal(t, "src/server/api_router.cc", got[0].File)
	require.Equal(t, 1, got[0].Line)
	require.Equal(t, ScopeFinding, got[0].Scope)
	require.Equal(t, "reason: legacy boundary", got[0].Reason)
}

func TestCollect_FindsIgnoreFileDirectiveWithoutIDAsWildcard(t *testing.T) {
	root := t.TempDir()

	writeSuppressionTestFile(t, root, "src/server/api_router.cc", `// patchcourt:ignore-file
`)

	got, err := Collect(root, []model.FileModel{
		{
			Path:     "src/server/api_router.cc",
			Language: model.LanguageCPP,
			Role:     model.FileRoleProduction,
		},
	})

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "*", got[0].ID)
	require.Equal(t, ScopeFile, got[0].Scope)
}

func TestApply_RemovesFindingWhenAllEvidenceIsSuppressed(t *testing.T) {
	project := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:       "architecture.api.cameras",
				Severity: model.SeverityHigh,
				Evidence: []model.Evidence{
					{
						File:    "src/server/api_router.cc",
						Message: "includes src/cameras/sony.h",
					},
				},
			},
		},
	}

	count := Apply(project, []Suppression{
		{
			ID:   "architecture.api.cameras",
			File: "src/server/api_router.cc",
		},
	})

	require.Equal(t, 1, count)
	require.Empty(t, project.Findings)
}

func TestApply_RemovesOnlySuppressedEvidenceFromGroupedFinding(t *testing.T) {
	project := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:       "discovery.cpp.unused_includes",
				Severity: model.SeverityLow,
				Evidence: []model.Evidence{
					{
						File:    "src/server/api_router.cc",
						Message: "unused include A",
					},
					{
						File:    "src/controllers/controller.cc",
						Message: "unused include B",
					},
				},
			},
		},
	}

	count := Apply(project, []Suppression{
		{
			ID:   "discovery.cpp.unused_includes",
			File: "src/server/api_router.cc",
		},
	})

	require.Equal(t, 0, count)
	require.Len(t, project.Findings, 1)
	require.Len(t, project.Findings[0].Evidence, 1)
	require.Equal(t, "src/controllers/controller.cc", project.Findings[0].Evidence[0].File)
}

func TestApply_WildcardSuppressesAnyFindingOnFile(t *testing.T) {
	project := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID: "architecture.api.cameras",
				Evidence: []model.Evidence{
					{File: "src/server/api_router.cc"},
				},
			},
			{
				ID: "discovery.cpp.unused_includes",
				Evidence: []model.Evidence{
					{File: "src/server/api_router.cc"},
				},
			},
		},
	}

	count := Apply(project, []Suppression{
		{
			ID:   "*",
			File: "src/server/api_router.cc",
		},
	})

	require.Equal(t, 2, count)
	require.Empty(t, project.Findings)
}

func TestApply_DoesNotSuppressFindingWithoutMatchingEvidenceFile(t *testing.T) {
	project := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID: "architecture.api.cameras",
				Evidence: []model.Evidence{
					{File: "src/server/api_router.cc"},
				},
			},
		},
	}

	count := Apply(project, []Suppression{
		{
			ID:   "architecture.api.cameras",
			File: "src/domain/status.h",
		},
	})

	require.Equal(t, 0, count)
	require.Len(t, project.Findings, 1)
}

func writeSuppressionTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
}
