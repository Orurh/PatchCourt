package changes

import (
	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/model"
)

type Result struct {
	ContractChanges   []contracts.SymbolChange    `json:"contract_changes"`
	DependencyChanges []depdiff.DependencyChange  `json:"dependency_changes"`
	LayerEdgeChanges  []depdiff.LayerEdgeChange   `json:"layer_edge_changes"`
	FindingChanges    []findingdiff.FindingChange `json:"finding_changes"`
	Risk              risk.Score                  `json:"risk"`
}

func Compare(before *model.ProjectModel, after *model.ProjectModel) Result {
	if before == nil {
		before = &model.ProjectModel{}
	}

	if after == nil {
		after = &model.ProjectModel{}
	}

	contractChanges := contracts.DiffSymbols(before.Symbols, after.Symbols)
	dependencyChanges := depdiff.DiffDependencies(before.Dependencies, after.Dependencies)
	layerEdgeChanges := depdiff.DiffLayerEdges(before.Dependencies, after.Dependencies)
	findingChanges := findingdiff.DiffFindings(before.Findings, after.Findings)

	reviewRisk := risk.Calculate(risk.Input{
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
	})

	return Result{
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
		Risk:              reviewRisk,
	}
}
