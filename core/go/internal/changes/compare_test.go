package changes

import (
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCompare_DetectsDependencyAndLayerEdgeChanges(t *testing.T) {
	before := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
		},
	}

	after := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			dep("src/server/api_router.cc", "src/cameras/sony.h", "server", "cameras"),
		},
	}

	result := Compare(before, after)

	require.Len(t, result.DependencyChanges, 2)
	require.Len(t, result.LayerEdgeChanges, 2)
	require.NotEqual(t, 0, result.Risk.Points)

	require.NotNil(t, findDependencyChange(result.DependencyChanges, depdiff.DependencyChangeKindAdded))
	require.NotNil(t, findDependencyChange(result.DependencyChanges, depdiff.DependencyChangeKindRemoved))
}

func TestCompare_AllowsNilModels(t *testing.T) {
	result := Compare(nil, nil)

	require.Empty(t, result.ContractChanges)
	require.Empty(t, result.DependencyChanges)
	require.Empty(t, result.LayerEdgeChanges)
	require.Empty(t, result.FindingChanges)
	require.Equal(t, 0, result.Risk.Points)
}

func dep(fromFile string, toFile string, fromLayer string, toLayer string) model.DependencyEdge {
	return model.DependencyEdge{
		FromFile:  fromFile,
		ToFile:    toFile,
		Target:    toFile,
		Kind:      model.DependencyKindInclude,
		Resolved:  true,
		FromLayer: fromLayer,
		ToLayer:   toLayer,
	}
}

func findDependencyChange(changes []depdiff.DependencyChange, kind depdiff.DependencyChangeKind) *depdiff.DependencyChange {
	for i := range changes {
		if changes[i].Kind == kind {
			return &changes[i]
		}
	}

	return nil
}
