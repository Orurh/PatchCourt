package graph

import (
	"testing"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBuildLayerGraph_CountsEdgesAndKeepsEvidence(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{Path: "src/server/api_router.cc", Layer: "server"},
			{Path: "src/server/admin_router.cc", Layer: "server"},
			{Path: "src/domain/status.h", Layer: "domain"},
		},
		Dependencies: []model.DependencyEdge{
			resolvedGraphDep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
			resolvedGraphDep("src/server/admin_router.cc", "src/domain/status.h", "server", "domain"),
		},
	}

	layerGraph := BuildLayerGraph(project, nil)

	require.Len(t, layerGraph.Edges, 1)

	edge := layerGraph.Edges[0]
	require.Equal(t, "server", edge.From)
	require.Equal(t, "domain", edge.To)
	require.Equal(t, 2, edge.Count)
	require.Len(t, edge.Evidence, 2)
}

func TestBuildLayerGraph_MarksPolicyViolation(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{Path: "src/server/api_router.cc", Layer: "api"},
			{Path: "src/cameras/sony.h", Layer: "cameras"},
		},
		Dependencies: []model.DependencyEdge{
			resolvedGraphDep("src/server/api_router.cc", "src/cameras/sony.h", "api", "cameras"),
		},
	}

	cfg := &config.Config{
		Layers: map[string]config.LayerConfig{
			"api": {
				MayDependOn: []string{"domain", "controllers"},
			},
		},
	}

	layerGraph := BuildLayerGraph(project, cfg)

	require.Len(t, layerGraph.Edges, 1)
	require.True(t, layerGraph.Edges[0].Violation)
}

func TestBuildLayerGraph_IgnoresExternalUnresolvedAndSameLayerDeps(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			{
				FromLayer: "server",
				ToLayer:   "domain",
				Resolved:  false,
			},
			{
				FromLayer: "server",
				ToLayer:   "domain",
				Resolved:  true,
				External:  true,
			},
			resolvedGraphDep("src/server/a.cc", "src/server/b.h", "server", "server"),
		},
	}

	layerGraph := BuildLayerGraph(project, nil)

	require.Empty(t, layerGraph.Edges)
}

func resolvedGraphDep(fromFile string, toFile string, fromLayer string, toLayer string) model.DependencyEdge {
	return model.DependencyEdge{
		FromFile:  fromFile,
		ToFile:    toFile,
		Target:    toFile,
		Kind:      model.DependencyKindInclude,
		Resolved:  true,
		FromLayer: fromLayer,
		ToLayer:   toLayer,
		Usage:     model.DependencyUsageUsed,
	}
}
