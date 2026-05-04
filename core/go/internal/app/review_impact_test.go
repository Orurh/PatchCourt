package app

import (
	"testing"

	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBuildReviewImpactReport_DoesNotPutNoisyAddedDependenciesInWorse(t *testing.T) {
	result := &ReviewResult{
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|string",
				After: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					Target:    "string",
					FromLayer: "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/domain/a.h|src/domain/b.h",
				After: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					ToFile:    "src/domain/b.h",
					Target:    "src/domain/b.h",
					FromLayer: "domain",
					ToLayer:   "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "include|src/cameras/a.h|src/domain/b.h",
				After: &model.DependencyEdge{
					FromFile:  "src/cameras/a.h",
					ToFile:    "src/domain/b.h",
					Target:    "src/domain/b.h",
					FromLayer: "cameras",
					ToLayer:   "domain",
					Resolved:  true,
				},
			},
		},
	}

	afterProject := &model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:   "architecture.cameras.domain",
				Kind: model.FindingKindPolicyViolation,
				Evidence: []model.Evidence{
					{
						FromLayer: "cameras",
						ToLayer:   "domain",
					},
				},
			},
		},
	}

	impact := BuildReviewImpactReport(result, nil, afterProject)

	require.Len(t, impact.Worse, 1)
	require.Equal(t, "dependency_added", impact.Worse[0].Kind)
	require.Equal(t, "include|src/cameras/a.h|src/domain/b.h", impact.Worse[0].ID)
}

func TestBuildReviewImpactReport_DoesNotPutNoisyRemovedDependenciesInBetter(t *testing.T) {
	result := &ReviewResult{
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindRemoved,
				Key:  "include|src/cameras/a.cc|src/cameras/b.h",
				Before: &model.DependencyEdge{
					FromFile:  "src/cameras/a.cc",
					ToFile:    "src/cameras/b.h",
					Target:    "src/cameras/b.h",
					FromLayer: "cameras",
					ToLayer:   "cameras",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindRemoved,
				Key:  "include|src/domain/a.h|string",
				Before: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					Target:    "string",
					FromLayer: "domain",
					Resolved:  true,
				},
			},
			{
				Kind: depdiff.DependencyChangeKindRemoved,
				Key:  "include|src/domain/a.h|src/cameras/b.h",
				Before: &model.DependencyEdge{
					FromFile:  "src/domain/a.h",
					ToFile:    "src/cameras/b.h",
					Target:    "src/cameras/b.h",
					FromLayer: "domain",
					ToLayer:   "cameras",
					Resolved:  true,
				},
			},
		},
	}

	impact := BuildReviewImpactReport(result, nil, nil)

	require.Len(t, impact.Better, 1)
	require.Equal(t, "dependency_removed", impact.Better[0].Kind)
	require.Equal(t, "include|src/domain/a.h|src/cameras/b.h", impact.Better[0].ID)
}

func TestBuildReviewImpactReport_DoesNotPutAllowedAddedCrossLayerDependencyInWorse(t *testing.T) {
	result := &ReviewResult{
		DependencyChanges: []depdiff.DependencyChange{
			{
				Kind: depdiff.DependencyChangeKindAdded,
				Key:  "import|internal/app/check.go|internal/reportmodel/reportmodel.go",
				After: &model.DependencyEdge{
					FromFile:  "internal/app/check.go",
					ToFile:    "internal/reportmodel/reportmodel.go",
					Target:    "github.com/orurh/patchcourt/internal/reportmodel",
					Kind:      model.DependencyKindImport,
					FromLayer: "app",
					ToLayer:   "reportmodel",
					Resolved:  true,
				},
			},
		},
	}

	impact := BuildReviewImpactReport(result, nil, &model.ProjectModel{})

	require.Empty(t, impact.Worse)
}
