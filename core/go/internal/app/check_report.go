package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
)

const (
	defaultCheckTopFindings    = 5
	defaultCheckCoupledEdges   = 5
	defaultCheckSuspiciousEdge = 5
	defaultCheckNextEdges      = 2
)

type CheckReport struct {
	Root       string            `json:"root"`
	ConfigPath string            `json:"config_path,omitempty"`
	OutDir     string            `json:"out_dir"`
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

func BuildCheckReport(result *CheckResult) CheckReport {
	if result == nil {
		return CheckReport{}
	}

	var findings []model.Finding
	if result.Project != nil {
		findings = result.Project.Findings
	}

	report := CheckReport{
		Root:           result.Root,
		ConfigPath:     result.ConfigPath,
		OutDir:         result.OutDir,
		Summary:        result.Summary,
		FindingCount:   len(findings),
		GraphNodeCount: len(result.LayerGraph.Nodes),
		GraphEdgeCount: len(result.LayerGraph.Edges),
		Artifacts:      result.Artifacts,
	}

	report.TopFindings = topFindingSummaries(findings, defaultCheckTopFindings)
	report.MostCoupledEdges = mostCoupledEdges(result.LayerGraph, defaultCheckCoupledEdges)
	report.SuspiciousEdges = suspiciousEdges(result.LayerGraph, findings, defaultCheckSuspiciousEdge)
	report.NextSteps = checkNextSteps(result, report)

	return report
}

func topFindingSummaries(findings []model.Finding, limit int) []FindingSummary {
	if limit <= 0 || len(findings) == 0 {
		return nil
	}

	if len(findings) < limit {
		limit = len(findings)
	}

	result := make([]FindingSummary, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, FindingSummary{
			ID:       findings[i].ID,
			Kind:     string(findings[i].Kind),
			Severity: string(findings[i].Severity),
			Title:    findings[i].Title,
		})
	}

	return result
}

func mostCoupledEdges(layerGraph graphmodel.LayerGraph, limit int) []EdgeSummary {
	edges := make([]EdgeSummary, 0, len(layerGraph.Edges))

	for _, edge := range layerGraph.Edges {
		edges = append(edges, EdgeSummary{
			From:  edge.From,
			To:    edge.To,
			Count: edge.Count,
		})
	}

	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Count != edges[j].Count {
			return edges[i].Count > edges[j].Count
		}

		left := edges[i].From + "->" + edges[i].To
		right := edges[j].From + "->" + edges[j].To
		return left < right
	})

	return limitEdgeSummaries(edges, limit)
}

func suspiciousEdges(layerGraph graphmodel.LayerGraph, findings []model.Finding, limit int) []EdgeSummary {
	findingByEdge := indexFindingsByEdge(findings)
	result := make([]EdgeSummary, 0)

	for _, edge := range layerGraph.Edges {
		key := edgeKey(edge.From, edge.To)
		finding, ok := findingByEdge[key]
		if !ok {
			continue
		}

		result = append(result, EdgeSummary{
			From:      edge.From,
			To:        edge.To,
			Count:     edge.Count,
			FindingID: finding.ID,
			Priority:  finding.Priority,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}

		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}

		left := result[i].From + "->" + result[i].To
		right := result[j].From + "->" + result[j].To
		return left < right
	})

	return limitEdgeSummaries(result, limit)
}

func limitEdgeSummaries(edges []EdgeSummary, limit int) []EdgeSummary {
	if limit <= 0 || len(edges) == 0 {
		return nil
	}

	if len(edges) <= limit {
		return edges
	}

	return edges[:limit]
}

func checkNextSteps(result *CheckResult, report CheckReport) []NextStep {
	modelPath := result.ArtifactPathByName("project model")
	if modelPath == "" && result.OutDir != "" {
		modelPath = filepath.Join(result.OutDir, "project-model.json")
	}

	steps := make([]NextStep, 0)

	for i, edge := range report.SuspiciousEdges {
		if i >= defaultCheckNextEdges {
			break
		}

		steps = append(steps, NextStep{
			Label:   fmt.Sprintf("Inspect edge %s -> %s", edge.From, edge.To),
			Command: fmt.Sprintf("patchcourt edge --model %s %s %s", modelPath, edge.From, edge.To),
		})
	}

	if findingID := firstSuggestedFindingID(report); findingID != "" {
		if modelPath != "" {
			steps = append(steps, NextStep{
				Label:   fmt.Sprintf("Explain finding %s", findingID),
				Command: fmt.Sprintf("patchcourt explain %s --model %s", findingID, modelPath),
			})
		} else {
			steps = append(steps, NextStep{
				Label:   fmt.Sprintf("Explain finding %s", findingID),
				Command: fmt.Sprintf("patchcourt explain %s --root %s", findingID, result.Root),
			})
		}
	}

	htmlPath := result.ArtifactPathByName("html report")
	if htmlPath != "" {
		steps = append(steps, NextStep{
			Label:   "Open HTML report",
			Command: fmt.Sprintf("xdg-open %s", htmlPath),
		})
	}

	dotPath := result.ArtifactPathByName("layer graph dot")
	if dotPath != "" {
		svgPath := filepath.Join(filepath.Dir(dotPath), "layer-graph.svg")
		steps = append(steps, NextStep{
			Label:   "Render layer graph SVG",
			Command: fmt.Sprintf("dot -Tsvg %s -o %s", dotPath, svgPath),
		})
	}

	return steps
}

type findingForEdge struct {
	ID       string
	Priority int
}

func indexFindingsByEdge(findings []model.Finding) map[string]findingForEdge {
	result := make(map[string]findingForEdge)

	for _, finding := range findings {
		priority := findingPriority(finding)

		for _, key := range findingEdgeKeys(finding) {
			current, exists := result[key]
			if exists && current.Priority >= priority {
				continue
			}

			result[key] = findingForEdge{
				ID:       finding.ID,
				Priority: priority,
			}
		}
	}

	return result
}

var evidenceEdgeRE = regexp.MustCompile(`dependency\s+([A-Za-z0-9_.:-]+)\s+->\s+([A-Za-z0-9_.:-]+)`)

func findingEdgeKeys(finding model.Finding) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	add := func(from string, to string) {
		key := edgeKey(from, to)
		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}
		result = append(result, key)
	}

	for _, evidence := range finding.Evidence {
		match := evidenceEdgeRE.FindStringSubmatch(evidence.Message)
		if len(match) != 3 {
			continue
		}

		add(match[1], match[2])
	}

	if strings.HasPrefix(finding.ID, "architecture.") {
		parts := strings.Split(finding.ID, ".")
		if len(parts) == 3 {
			add(parts[1], parts[2])
		}
	}

	return result
}

func edgeKey(from string, to string) string {
	return from + "->" + to
}

func findingPriority(finding model.Finding) int {
	if finding.Kind == model.FindingKindPolicyViolation {
		return 100 + severityRank(finding.Severity)
	}

	switch {
	case strings.HasPrefix(finding.ID, "discovery.domain.depends_on."):
		return 80 + severityRank(finding.Severity)
	case finding.ID == "discovery.controllers.depends_on.server":
		return 70 + severityRank(finding.Severity)
	case finding.ID == "discovery.shared.depends_on.domain":
		return 60 + severityRank(finding.Severity)
	case strings.HasPrefix(finding.ID, "discovery.bidirectional."):
		return 50 + severityRank(finding.Severity)
	default:
		return severityRank(finding.Severity)
	}
}

func severityRank(severity model.Severity) int {
	switch severity {
	case model.SeverityCritical:
		return 4
	case model.SeverityHigh:
		return 3
	case model.SeverityMedium:
		return 2
	case model.SeverityLow:
		return 1
	default:
		return 0
	}
}

func firstSuggestedFindingID(report CheckReport) string {
	for _, edge := range report.SuspiciousEdges {
		if edge.FindingID != "" {
			return edge.FindingID
		}
	}

	for _, finding := range report.TopFindings {
		if finding.ID != "" {
			return finding.ID
		}
	}

	return ""
}
