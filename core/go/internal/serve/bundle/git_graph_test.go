package bundle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGitGraphLayout_LinearHistory(t *testing.T) {
	commits := []GitCommit{
		{Hash: "c", ShortHash: "c", Parents: []string{"b"}, Message: "c"},
		{Hash: "b", ShortHash: "b", Parents: []string{"a"}, Message: "b"},
		{Hash: "a", ShortHash: "a", Message: "a"},
	}

	graph := buildGitGraph("/repo", "HEAD", false, 10, commits)

	require.Equal(t, gitGraphSchemaVersion, graph.SchemaVersion)
	require.Len(t, graph.Commits, 3)
	require.Len(t, graph.Layout.Segments, 1)
	require.Empty(t, graph.Layout.Edges)

	require.Equal(t, GitGraphSegment{
		Lane:    0,
		FromRow: 0,
		ToRow:   2,
	}, graph.Layout.Segments[0])
}

func TestBuildGitGraphLayout_BranchEdge(t *testing.T) {
	commits := []GitCommit{
		{
			Hash:      "feature",
			ShortHash: "feature",
			Parents:   []string{"main"},
			Refs:      []string{"feature/test"},
			Message:   "feature",
		},
		{
			Hash:      "main",
			ShortHash: "main",
			Refs:      []string{"main"},
			Message:   "main",
		},
	}

	graph := buildGitGraph("/repo", "feature/test", false, 10, commits)

	require.Empty(t, graph.Layout.Segments)
	require.Len(t, graph.Layout.Edges, 1)

	edge := graph.Layout.Edges[0]
	require.Equal(t, 0, edge.FromRow)
	require.Equal(t, 1, edge.ToRow)
	require.Equal(t, "branch", edge.Kind)
	require.NotEqual(t, edge.FromLane, edge.ToLane)
}

func TestBuildGitGraphLayout_MergeEdge(t *testing.T) {
	commits := []GitCommit{
		{
			Hash:      "merge",
			ShortHash: "merge",
			Parents:   []string{"main", "feature"},
			Refs:      []string{"main"},
			Message:   "merge",
		},
		{
			Hash:      "main",
			ShortHash: "main",
			Refs:      []string{"main"},
			Message:   "main",
		},
		{
			Hash:      "feature",
			ShortHash: "feature",
			Refs:      []string{"feature/test"},
			Message:   "feature",
		},
	}

	graph := buildGitGraph("/repo", "main", false, 10, commits)

	require.NotEmpty(t, graph.Layout.Edges)

	var mergeEdge *GitGraphEdge
	for index := range graph.Layout.Edges {
		edge := &graph.Layout.Edges[index]
		if edge.Kind == "merge" {
			mergeEdge = edge
			break
		}
	}

	require.NotNil(t, mergeEdge)
	require.Equal(t, 0, mergeEdge.FromRow)
	require.Equal(t, 2, mergeEdge.ToRow)
	require.NotEqual(t, mergeEdge.FromLane, mergeEdge.ToLane)
}

func TestBuildGitGraphLayout_SkipsParentOutsideVisibleRange(t *testing.T) {
	commits := []GitCommit{
		{
			Hash:      "head",
			ShortHash: "head",
			Parents:   []string{"missing-parent"},
			Message:   "head",
		},
	}

	graph := buildGitGraph("/repo", "HEAD", false, 1, commits)

	require.Len(t, graph.Commits, 1)
	require.Empty(t, graph.Layout.Segments)
	require.Empty(t, graph.Layout.Edges)
}

func TestMergeGitGraphSegments_MergesAdjacentSameLaneSegments(t *testing.T) {
	segments := mergeGitGraphSegments([]GitGraphSegment{
		{Lane: 0, FromRow: 0, ToRow: 1},
		{Lane: 0, FromRow: 1, ToRow: 2},
		{Lane: 1, FromRow: 4, ToRow: 5},
	})

	require.Equal(t, []GitGraphSegment{
		{Lane: 0, FromRow: 0, ToRow: 2},
		{Lane: 1, FromRow: 4, ToRow: 5},
	}, segments)
}

func TestMergeGitGraphSegments_NormalizesReversedRows(t *testing.T) {
	segments := mergeGitGraphSegments([]GitGraphSegment{
		{Lane: 0, FromRow: 3, ToRow: 1},
	})

	require.Equal(t, []GitGraphSegment{
		{Lane: 0, FromRow: 1, ToRow: 3},
	}, segments)
}
