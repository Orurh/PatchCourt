package usecase

import (
	"fmt"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"path/filepath"
	"sort"
	"strings"

	graphmodel "github.com/orurh/patchcourt/internal/analyzer/graph"
	"github.com/orurh/patchcourt/internal/model"
)

type CheckReport = reportmodel.CheckReport
type FindingSummary = reportmodel.FindingSummary
type EdgeSummary = reportmodel.EdgeSummary
type NextStep = reportmodel.NextStep

const (
	defaultCheckTopFindings    = 5
	defaultCheckCoupledEdges   = 5
	defaultCheckSuspiciousEdge = 5
	defaultCheckNextEdges      = 2
)

func BuildCheckReport(result *CheckResult) CheckReport {
	if result == nil {
		return CheckReport{}
	}

	var findings []model.Finding
	if result.Project != nil {
		findings = result.Project.Findings
	}

	report := CheckReport{
		SchemaVersion:  reportmodel.CheckReportSchemaVersion,
		Root:           result.Root,
		ConfigPath:     result.ConfigPath,
		OutDir:         result.OutDir,
		StatePath:      result.StatePath,
		Summary:        result.Summary,
		FindingCount:   len(findings),
		GraphNodeCount: len(result.LayerGraph.Nodes),
		GraphEdgeCount: len(result.LayerGraph.Edges),
		Artifacts:      result.Artifacts,
	}

	report.TopFindings = topFindingSummaries(findings, defaultCheckTopFindings)
	report.MostCoupledEdges = mostCoupledEdges(result.LayerGraph, defaultCheckCoupledEdges)
	report.SuspiciousEdges = suspiciousEdges(result.Project, result.LayerGraph, findings, defaultCheckSuspiciousEdge)
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

func suspiciousEdges(project *model.ProjectModel, layerGraph graphmodel.LayerGraph, findings []model.Finding, limit int) []EdgeSummary {
	findingByEdge := indexFindingsByEdge(findings)
	result := make([]EdgeSummary, 0)

	for _, edge := range layerGraph.Edges {
		key := edgeKey(edge.From, edge.To)
		finding, ok := findingByEdge[key]
		if !ok {
			continue
		}

		edgeReport := BuildEdgeReport(project, EdgeReportOptions{
			FromLayer: edge.From,
			ToLayer:   edge.To,
			Limit:     1,
		})

		result = append(result, EdgeSummary{
			From:      edge.From,
			To:        edge.To,
			Count:     edgeReport.Count,
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
		} else if result.Root != "" {
			steps = append(steps, NextStep{
				Label:   fmt.Sprintf("Explain finding %s", findingID),
				Command: fmt.Sprintf("patchcourt explain %s --root %s", findingID, result.Root),
			})
		}
	}

	if result.OutDir != "" {
		steps = append(steps, NextStep{
			Label:   "Open markdown scan report",
			Command: fmt.Sprintf("xdg-open %s", filepath.Join(result.OutDir, "scan.md")),
		})

		steps = append(steps, NextStep{
			Label: "Render layer graph SVG",
			Command: fmt.Sprintf(
				"dot -Tsvg %s -o %s",
				filepath.Join(result.OutDir, "layer-graph.dot"),
				filepath.Join(result.OutDir, "layer-graph.svg"),
			),
		})
	}

	return steps
}

type findingForEdge struct {
	ID       string
	Priority int
}

func indexFindingsByEdge(findings []model.Finding) map[string]findingForEdge {
	index := make(map[string]findingForEdge)

	for _, finding := range findings {
		priority := findingPriority(finding)

		for _, key := range findingEdgeKeys(finding) {
			existing, ok := index[key]
			if ok && existing.Priority >= priority {
				continue
			}

			index[key] = findingForEdge{
				ID:       finding.ID,
				Priority: priority,
			}
		}
	}

	return index
}

func findingEdgeKeys(finding model.Finding) []string {
	keys := make([]string, 0, 4)

	for _, evidence := range finding.Evidence {
		if evidence.FromLayer == "" || evidence.ToLayer == "" {
			continue
		}

		keys = append(keys, edgeKey(evidence.FromLayer, evidence.ToLayer))
	}

	if from, to, ok := architectureFindingEdge(finding.ID); ok {
		keys = append(keys, edgeKey(from, to))
	}

	if left, right, ok := bidirectionalFindingLayers(finding.ID); ok {
		keys = append(keys, edgeKey(left, right), edgeKey(right, left))
	}

	return uniqueStrings(keys)
}

func architectureFindingEdge(id string) (string, string, bool) {
	const prefix = "architecture."

	if !strings.HasPrefix(id, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(id, prefix)
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func bidirectionalFindingLayers(id string) (string, string, bool) {
	const prefix = "discovery.bidirectional."

	if !strings.HasPrefix(id, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(id, prefix)
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func edgeKey(from string, to string) string {
	return from + "->" + to
}

func findingPriority(finding model.Finding) int {
	priority := model.SeverityRank(finding.Severity)

	if finding.Kind == model.FindingKindPolicyViolation {
		priority += 20
	}

	switch {
	case strings.HasPrefix(finding.ID, "architecture."):
		priority += 40
	case strings.HasPrefix(finding.ID, "discovery.domain.depends_on."):
		priority += 32
	case finding.ID == "discovery.controllers.depends_on.server":
		priority += 22
	case finding.ID == "discovery.shared.depends_on.domain":
		priority += 51
	case strings.HasPrefix(finding.ID, "discovery.bidirectional."):
		priority += 2
	}

	return priority
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
