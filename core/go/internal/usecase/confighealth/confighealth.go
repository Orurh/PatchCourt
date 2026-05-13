package confighealth

import (
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const (
	lowCoverageMinDependencies = 20
	lowCoverageThreshold       = 10.0
)

func Build(project *model.ProjectModel, configPath string, graphNodeCount int, graphEdgeCount int) reportmodel.ConfigHealth {
	health := reportmodel.ConfigHealth{
		ConfigPath:     configPath,
		ConfigExplicit: configPath != "",
		GraphNodeCount: graphNodeCount,
		GraphEdgeCount: graphEdgeCount,
	}

	if project == nil {
		return health
	}

	for _, dependency := range project.Dependencies {
		if dependency.External || !dependency.Resolved {
			continue
		}
		if dependency.FromFile == "" || dependency.ToFile == "" {
			continue
		}

		health.InternalResolvedDependencies++

		if dependency.FromLayer != "" && dependency.ToLayer != "" {
			health.LayerAnnotatedDependencies++
		}
	}

	if health.InternalResolvedDependencies > 0 {
		health.LayerCoveragePercent = float64(health.LayerAnnotatedDependencies) * 100 / float64(health.InternalResolvedDependencies)
	}

	if health.ConfigExplicit &&
		health.InternalResolvedDependencies >= lowCoverageMinDependencies &&
		health.LayerCoveragePercent < lowCoverageThreshold {
		health.Warnings = append(health.Warnings, reportmodel.ConfigHealthWarning{
			Code:    "config.low_layer_coverage",
			Message: "Configured layers match very few internal resolved dependencies.",
			Hint:    "The config may be outdated or too narrow. Try running without --config or regenerate .patchcourt.yaml.",
		})
	}

	return health
}
