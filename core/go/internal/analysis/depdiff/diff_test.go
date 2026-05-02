package depdiff

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDiffDependencies_DetectsAddedDependency(t *testing.T) {
	after := []model.DependencyEdge{
		dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
	}

	changes := DiffDependencies(nil, after)

	require.Len(t, changes, 1)
	require.Equal(t, DependencyChangeKindAdded, changes[0].Kind)
	require.NotNil(t, changes[0].After)
	require.Nil(t, changes[0].Before)
	require.Equal(t, "src/domain/status.h", changes[0].After.ToFile)
}

func TestDiffDependencies_DetectsRemovedDependency(t *testing.T) {
	before := []model.DependencyEdge{
		dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
	}

	changes := DiffDependencies(before, nil)

	require.Len(t, changes, 1)
	require.Equal(t, DependencyChangeKindRemoved, changes[0].Kind)
	require.NotNil(t, changes[0].Before)
	require.Nil(t, changes[0].After)
	require.Equal(t, "src/domain/status.h", changes[0].Before.ToFile)
}

func TestDiffDependencies_IgnoresUnchangedDependency(t *testing.T) {
	before := []model.DependencyEdge{
		dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
	}
	after := []model.DependencyEdge{
		dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
	}

	changes := DiffDependencies(before, after)

	require.Empty(t, changes)
}

func TestDiffDependencies_UsesTargetForUnresolvedDependencies(t *testing.T) {
	before := []model.DependencyEdge{
		{
			FromFile: "src/server/api_router.cc",
			Target:   "missing.h",
			Kind:     model.DependencyKindInclude,
			Resolved: false,
		},
	}
	after := []model.DependencyEdge{
		{
			FromFile: "src/server/api_router.cc",
			Target:   "other_missing.h",
			Kind:     model.DependencyKindInclude,
			Resolved: false,
		},
	}

	changes := DiffDependencies(before, after)

	require.Len(t, changes, 2)
	require.Equal(t, DependencyChangeKindRemoved, changes[0].Kind)
	require.Equal(t, DependencyChangeKindAdded, changes[1].Kind)
}

func TestDiffLayerEdges_DetectsAddedAndRemovedLayerEdges(t *testing.T) {
	before := []model.DependencyEdge{
		dep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
	}
	after := []model.DependencyEdge{
		dep("src/controllers/device_orchestrator.cc", "src/cameras/sony.h", "controllers", "cameras"),
		dep("src/controllers/other.cc", "src/cameras/sony.h", "controllers", "cameras"),
	}

	changes := DiffLayerEdges(before, after)

	require.Len(t, changes, 2)

	require.Equal(t, DependencyChangeKindAdded, changes[0].Kind)
	require.Equal(t, "controllers", changes[0].FromLayer)
	require.Equal(t, "cameras", changes[0].ToLayer)
	require.Equal(t, 2, changes[0].AfterCount)

	require.Equal(t, DependencyChangeKindRemoved, changes[1].Kind)
	require.Equal(t, "server", changes[1].FromLayer)
	require.Equal(t, "domain", changes[1].ToLayer)
	require.Equal(t, 1, changes[1].BeforeCount)
}

func TestDiffLayerEdges_IgnoresExternalUnresolvedSameLayerAndUnlayeredDeps(t *testing.T) {
	before := []model.DependencyEdge{
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
		dep("src/server/a.cc", "src/server/b.h", "server", "server"),
		{
			FromFile: "src/server/a.cc",
			ToFile:   "src/domain/b.h",
			Kind:     model.DependencyKindInclude,
			Resolved: true,
		},
	}

	changes := DiffLayerEdges(before, nil)

	require.Empty(t, changes)
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

func TestDiffLayerEdges_IgnoresEdgesFromTestGeneratedAndExternalFiles(t *testing.T) {
	before := []model.DependencyEdge{}
	after := []model.DependencyEdge{
		dep("tests/api_router_test.cc", "src/domain/status.h", "server", "domain"),
		dep("generated/foo.pb.cc", "src/domain/status.h", "server", "domain"),
		dep("third_party/lib.cc", "src/domain/status.h", "server", "domain"),
	}

	changes := DiffLayerEdges(before, after)

	require.Empty(t, changes)
}
