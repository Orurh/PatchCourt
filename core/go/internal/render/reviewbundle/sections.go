package reviewbundle

import (
	"sort"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const (
	findingsSchemaVersion         = "patchcourt.findings.v1"
	contractsSchemaVersion        = "patchcourt.contracts.v1"
	dependenciesSchemaVersion     = "patchcourt.dependencies.v1"
	maxEdgeDependencyEvidenceRows = 20
)

type FindingsReport struct {
	SchemaVersion string                      `json:"schema_version"`
	Summary       FindingsSummary             `json:"summary"`
	Changes       []findingdiff.FindingChange `json:"changes"`
}

type FindingsSummary struct {
	ChangeCount  int `json:"change_count"`
	AddedCount   int `json:"added_count"`
	RemovedCount int `json:"removed_count"`
	ChangedCount int `json:"changed_count"`
}

type ContractsReport struct {
	SchemaVersion string                       `json:"schema_version"`
	Summary       ContractsSummary             `json:"summary"`
	Changes       []contracts.SymbolChange     `json:"changes"`
	Impacts       []reportmodel.ContractImpact `json:"impacts,omitempty"`
}

type ContractsSummary struct {
	ChangeCount  int `json:"change_count"`
	AddedCount   int `json:"added_count"`
	RemovedCount int `json:"removed_count"`
	ChangedCount int `json:"changed_count"`
	ImpactCount  int `json:"impact_count"`
}

type DependenciesReport struct {
	SchemaVersion     string                     `json:"schema_version"`
	Summary           DependenciesSummary        `json:"summary"`
	DependencyChanges []depdiff.DependencyChange `json:"dependency_changes"`
	LayerEdgeChanges  []depdiff.LayerEdgeChange  `json:"layer_edge_changes"`
	EdgeDependencies  []EdgeDependencyGroup      `json:"edge_dependencies,omitempty"`
}

type EdgeDependencyGroup struct {
	FromLayer      string               `json:"from_layer"`
	ToLayer        string               `json:"to_layer"`
	Count          int                  `json:"count"`
	Dependencies   []DependencyEvidence `json:"dependencies"`
	TruncatedCount int                  `json:"truncated_count,omitempty"`
}

type DependencyEvidence struct {
	FromFile             string                     `json:"from_file,omitempty"`
	ToFile               string                     `json:"to_file,omitempty"`
	FromLayer            string                     `json:"from_layer,omitempty"`
	ToLayer              string                     `json:"to_layer,omitempty"`
	Target               string                     `json:"target,omitempty"`
	Kind                 model.DependencyKind       `json:"kind,omitempty"`
	Usage                model.DependencyUsage      `json:"usage,omitempty"`
	Resolved             bool                       `json:"resolved"`
	External             bool                       `json:"external"`
	ResolutionSource     model.ResolutionSource     `json:"resolution_source,omitempty"`
	ResolutionConfidence model.ResolutionConfidence `json:"resolution_confidence,omitempty"`
}

type DependenciesSummary struct {
	DependencyChangeCount int `json:"dependency_change_count"`
	LayerEdgeChangeCount  int `json:"layer_edge_change_count"`
	AddedDependencies     int `json:"added_dependencies"`
	RemovedDependencies   int `json:"removed_dependencies"`
	AddedLayerEdges       int `json:"added_layer_edges"`
	RemovedLayerEdges     int `json:"removed_layer_edges"`
	ChangedLayerEdges     int `json:"changed_layer_edges"`
}

func BuildFindingsReport(result reportmodel.ReviewResult) FindingsReport {
	summary := FindingsSummary{
		ChangeCount: len(result.FindingChanges),
	}

	for _, change := range result.FindingChanges {
		switch change.Kind {
		case findingdiff.FindingChangeKindAdded:
			summary.AddedCount++
		case findingdiff.FindingChangeKindRemoved:
			summary.RemovedCount++
		case findingdiff.FindingChangeKindChanged:
			summary.ChangedCount++
		}
	}

	return FindingsReport{
		SchemaVersion: findingsSchemaVersion,
		Summary:       summary,
		Changes:       result.FindingChanges,
	}
}

func BuildContractsReport(result reportmodel.ReviewResult) ContractsReport {
	summary := ContractsSummary{
		ChangeCount: len(result.ContractChanges),
		ImpactCount: len(result.ContractImpacts),
	}

	for _, change := range result.ContractChanges {
		switch string(change.Kind) {
		case "added":
			summary.AddedCount++
		case "removed":
			summary.RemovedCount++
		default:
			summary.ChangedCount++
		}
	}

	return ContractsReport{
		SchemaVersion: contractsSchemaVersion,
		Summary:       summary,
		Changes:       result.ContractChanges,
		Impacts:       result.ContractImpacts,
	}
}

func BuildDependenciesReport(result reportmodel.ReviewResult) DependenciesReport {
	summary := DependenciesSummary{
		DependencyChangeCount: len(result.DependencyChanges),
		LayerEdgeChangeCount:  len(result.LayerEdgeChanges),
	}

	for _, change := range result.DependencyChanges {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			summary.AddedDependencies++
		case depdiff.DependencyChangeKindRemoved:
			summary.RemovedDependencies++
		}
	}

	for _, change := range result.LayerEdgeChanges {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			summary.AddedLayerEdges++
		case depdiff.DependencyChangeKindRemoved:
			summary.RemovedLayerEdges++
		case depdiff.DependencyChangeKindChanged:
			summary.ChangedLayerEdges++
		}
	}

	return DependenciesReport{
		SchemaVersion:     dependenciesSchemaVersion,
		Summary:           summary,
		DependencyChanges: result.DependencyChanges,
		LayerEdgeChanges:  result.LayerEdgeChanges,
		EdgeDependencies:  buildEdgeDependencies(result.AfterProject),
	}
}

func buildEdgeDependencies(project *model.ProjectModel) []EdgeDependencyGroup {
	if project == nil {
		return nil
	}

	type groupState struct {
		from         string
		to           string
		count        int
		dependencies []DependencyEvidence
	}

	groups := make(map[string]*groupState)

	for _, dependency := range project.Dependencies {
		if dependency.External || !dependency.Resolved {
			continue
		}
		if dependency.FromLayer == "" || dependency.ToLayer == "" {
			continue
		}
		if dependency.FromLayer == dependency.ToLayer {
			continue
		}

		key := dependency.FromLayer + "\x00" + dependency.ToLayer
		group := groups[key]
		if group == nil {
			group = &groupState{
				from: dependency.FromLayer,
				to:   dependency.ToLayer,
			}
			groups[key] = group
		}

		group.count++
		if len(group.dependencies) >= maxEdgeDependencyEvidenceRows {
			continue
		}

		group.dependencies = append(group.dependencies, dependencyEvidenceFromEdge(dependency))
	}

	rows := make([]EdgeDependencyGroup, 0, len(groups))
	for _, group := range groups {
		truncated := group.count - len(group.dependencies)
		if truncated < 0 {
			truncated = 0
		}

		rows = append(rows, EdgeDependencyGroup{
			FromLayer:      group.from,
			ToLayer:        group.to,
			Count:          group.count,
			Dependencies:   group.dependencies,
			TruncatedCount: truncated,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FromLayer != rows[j].FromLayer {
			return rows[i].FromLayer < rows[j].FromLayer
		}
		return rows[i].ToLayer < rows[j].ToLayer
	})

	return rows
}

func dependencyEvidenceFromEdge(dependency model.DependencyEdge) DependencyEvidence {
	return DependencyEvidence{
		FromFile:             dependency.FromFile,
		ToFile:               dependency.ToFile,
		FromLayer:            dependency.FromLayer,
		ToLayer:              dependency.ToLayer,
		Target:               dependency.Target,
		Kind:                 dependency.Kind,
		Usage:                dependency.Usage,
		Resolved:             dependency.Resolved,
		External:             dependency.External,
		ResolutionSource:     dependency.ResolutionSource,
		ResolutionConfidence: dependency.ResolutionConfidence,
	}
}
