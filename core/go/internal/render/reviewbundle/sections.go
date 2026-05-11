package reviewbundle

import (
	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const (
	findingsSchemaVersion     = "patchcourt.findings.v1"
	contractsSchemaVersion    = "patchcourt.contracts.v1"
	dependenciesSchemaVersion = "patchcourt.dependencies.v1"
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
	}
}
