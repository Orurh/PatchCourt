package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/engine"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/state"
	"github.com/stretchr/testify/require"
)

type fakeSourceAnalyzer struct {
	req engine.AnalyzeRequest
	res *engine.AnalyzeResult
	err error
}

func (f *fakeSourceAnalyzer) Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error) {
	f.req = req
	return f.res, f.err
}

func TestSnapshotSource_LoadsProjectModel(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "project-model.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"root":"/repo"}`), 0o644))

	project, err := SnapshotSource{Path: path}.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "/repo", project.Root)
}

func TestStateSource_LoadsLatestState(t *testing.T) {
	root := t.TempDir()

	_, err := state.SaveState(state.SaveStateOptions{
		Root: root,
		Project: &model.ProjectModel{
			Root: root,
			Files: []model.FileModel{
				{Path: "src/main.cc"},
			},
		},
	})
	require.NoError(t, err)

	project, err := StateSource{Root: root}.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, root, project.Root)
	require.Len(t, project.Files, 1)
}

func TestRootSource_AnalyzesRoot(t *testing.T) {
	analysis := &fakeSourceAnalyzer{
		res: &engine.AnalyzeResult{
			Project: &model.ProjectModel{
				Root: "/repo",
			},
		},
	}

	project, err := RootSource{
		Root:       "/repo",
		ConfigPath: "/repo/.patchcourt.yaml",
		Operation:  "review-after",
		Analyzer:   analysis,
	}.Load(context.Background())

	require.NoError(t, err)
	require.Equal(t, "/repo", project.Root)
	require.Equal(t, "review-after", analysis.req.Operation)
	require.Equal(t, "/repo", analysis.req.Root)
	require.Equal(t, "/repo/.patchcourt.yaml", analysis.req.ConfigPath)
}

func TestLoadPair_LoadsBeforeAndAfter(t *testing.T) {
	root := t.TempDir()

	beforePath := filepath.Join(root, "before.json")
	afterPath := filepath.Join(root, "after.json")

	require.NoError(t, os.WriteFile(beforePath, []byte(`{"root":"/before"}`), 0o644))
	require.NoError(t, os.WriteFile(afterPath, []byte(`{"root":"/after"}`), 0o644))

	before, after, err := LoadPair(context.Background(), SourcePair{
		Before: SnapshotSource{Path: beforePath},
		After:  SnapshotSource{Path: afterPath},
	})

	require.NoError(t, err)
	require.Equal(t, "/before", before.Root)
	require.Equal(t, "/after", after.Root)
}

func TestLoadPair_RequiresSources(t *testing.T) {
	_, _, err := LoadPair(context.Background(), SourcePair{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "before project model source is required")
}
