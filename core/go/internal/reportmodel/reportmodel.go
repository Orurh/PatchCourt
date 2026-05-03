package reportmodel

import (
	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type CheckArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type CheckResult struct {
	Root       string                `json:"root"`
	ConfigPath string                `json:"config_path,omitempty"`
	OutDir     string                `json:"out_dir"`
	StatePath  string                `json:"state_path,omitempty"`
	Project    *model.ProjectModel   `json:"project,omitempty"`
	Config     *config.Config        `json:"config,omitempty"`
	LayerGraph graphmodel.LayerGraph `json:"layer_graph"`
	Summary    model.ScanSummary     `json:"summary"`
	Artifacts  []CheckArtifact       `json:"artifacts"`
}

type CheckReport struct {
	Root       string            `json:"root"`
	ConfigPath string            `json:"config_path,omitempty"`
	OutDir     string            `json:"out_dir"`
	StatePath  string            `json:"state_path,omitempty"`
	Summary    model.ScanSummary `json:"summary"`

	FindingCount   int `json:"finding_count"`
	GraphNodeCount int `json:"graph_node_count"`
	GraphEdgeCount int `json:"graph_edge_count"`

	Artifacts []CheckArtifact `json:"artifacts,omitempty"`

	TopFindings      []FindingSummary `json:"top_findings,omitempty"`
	MostCoupledEdges []EdgeSummary    `json:"most_coupled_edges,omitempty"`
	SuspiciousEdges  []EdgeSummary    `json:"suspicious_edges,omitempty"`
	NextSteps        []NextStep       `json:"next_steps,omitempty"`
}

type FindingSummary struct {
	ID       string `json:"id,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title,omitempty"`
}

type EdgeSummary struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Count     int    `json:"count"`
	FindingID string `json:"finding_id,omitempty"`
	Priority  int    `json:"priority,omitempty"`
}

type NextStep struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

type EdgeResult struct {
	Root          string                 `json:"root,omitempty"`
	Source        string                 `json:"source"`
	FromLayer     string                 `json:"from_layer"`
	ToLayer       string                 `json:"to_layer"`
	Count         int                    `json:"count"`
	Usage         EdgeUsageSummary       `json:"usage"`
	Findings      []model.Finding        `json:"findings,omitempty"`
	TopFromFiles  []EdgeFileCount        `json:"top_from_files,omitempty"`
	TopToFiles    []EdgeFileCount        `json:"top_to_files,omitempty"`
	Dependencies  []model.DependencyEdge `json:"dependencies,omitempty"`
	TruncatedDeps int                    `json:"truncated_deps,omitempty"`
}

type EdgeUsageSummary struct {
	Used    int `json:"used"`
	Maybe   int `json:"maybe"`
	Unused  int `json:"unused"`
	Unknown int `json:"unknown"`
}

type EdgeFileCount struct {
	File  string `json:"file"`
	Count int    `json:"count"`
}

type ExplainResult struct {
	Finding model.Finding `json:"finding"`
	Source  string        `json:"source"`
}

type ReviewSummary struct {
	ContractChanges     int `json:"contract_changes"`
	DependencyChanges   int `json:"dependency_changes"`
	LayerEdgeChanges    int `json:"layer_edge_changes"`
	FindingChanges      int `json:"finding_changes"`
	AddedFindings       int `json:"added_findings"`
	RemovedFindings     int `json:"removed_findings"`
	AddedHighFindings   int `json:"added_high_findings"`
	AddedPolicyFindings int `json:"added_policy_findings"`
}

type ReviewResult struct {
	Summary           ReviewSummary               `json:"summary"`
	Risk              risk.Score                  `json:"risk"`
	Impact            ReviewImpactReport          `json:"impact"`
	ContractChanges   []contracts.SymbolChange    `json:"contract_changes"`
	DependencyChanges []depdiff.DependencyChange  `json:"dependency_changes"`
	LayerEdgeChanges  []depdiff.LayerEdgeChange   `json:"layer_edge_changes"`
	FindingChanges    []findingdiff.FindingChange `json:"finding_changes"`
}

type ReviewImpactReport struct {
	Worse         []ReviewImpactItem `json:"worse"`
	Better        []ReviewImpactItem `json:"better"`
	UnchangedDebt []ReviewImpactItem `json:"unchanged_debt"`
}

type ReviewImpactItem struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	ID       string `json:"id,omitempty"`
}

func (result *CheckResult) ArtifactPathByName(name string) string {
	if result == nil {
		return ""
	}

	for _, artifact := range result.Artifacts {
		if artifact.Name == name {
			return artifact.Path
		}
	}

	return ""
}
