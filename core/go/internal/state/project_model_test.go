package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadState(t *testing.T) {
	root := t.TempDir()

	project := &model.ProjectModel{
		Root: root,
		Files: []model.FileModel{
			{Path: "src/main.cc"},
			{Path: "src/domain/status.h"},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/main.cc",
				ToFile:   "src/domain/status.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
			},
		},
		Findings: []model.Finding{
			{
				ID:       "discovery.test",
				Severity: model.SeverityLow,
			},
		},
	}

	metadata, err := SaveState(SaveStateOptions{
		Root:       root,
		ConfigPath: filepath.Join(root, ".patchcourt.yaml"),
		Project:    project,
	})
	require.NoError(t, err)

	require.Equal(t, 1, metadata.SchemaVersion)
	require.Equal(t, 2, metadata.Files)
	require.Equal(t, 1, metadata.Dependencies)
	require.Equal(t, 1, metadata.Findings)

	loaded, err := LoadState(LoadStateOptions{
		Root: root,
	})
	require.NoError(t, err)

	require.Equal(t, filepath.Join(root, ".patchcourt/state/latest/project-model.json"), loaded.Path)
	require.Equal(t, project.Root, loaded.Project.Root)
	require.Len(t, loaded.Project.Files, 2)
	require.Len(t, loaded.Project.Dependencies, 1)
	require.Len(t, loaded.Project.Findings, 1)
	require.Equal(t, metadata.Files, loaded.Metadata.Files)
}

func TestLoadState_ReturnsHelpfulErrorWhenMissing(t *testing.T) {
	root := t.TempDir()

	_, err := LoadState(LoadStateOptions{
		Root: root,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "read state project model")
}

func TestStateDirDefaultsToLatest(t *testing.T) {
	root := t.TempDir()

	require.Equal(t, filepath.Join(root, ".patchcourt/state/latest"), StateDir(root, ""))
	require.Equal(t, filepath.Join(root, ".patchcourt/state/custom"), StateDir(root, "custom"))
}

func TestReadProjectModel(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "project-model.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"root":"/repo"}`), 0o644))

	project, err := ReadProjectModel(path)
	require.NoError(t, err)
	require.Equal(t, "/repo", project.Root)
}
