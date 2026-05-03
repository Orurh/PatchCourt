package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
)

const (
	maxCheckTopFindings    = 5
	maxCheckCoupledEdges   = 5
	maxCheckSuspiciousEdge = 5
	maxCheckNextEdges      = 2
)

type CheckTextArtifact struct {
	Name string
	Path string
}

type CheckTextResult struct {
	Root       string
	ConfigPath string
	OutDir     string
	Project    *model.ProjectModel
	Summary    model.ScanSummary
	LayerGraph graphmodel.LayerGraph
	Artifacts  []CheckTextArtifact
}

type checkEdgeSummary struct {
	From       string
	To         string
	Count      int
	Suspicious bool
	FindingID  string
}

func WriteCheckText(w io.Writer, result CheckTextResult) {
	fmt.Fprintln(w, "PatchCourt check")
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Root: %s\n", result.Root)
	if result.ConfigPath != "" {
		fmt.Fprintf(w, "Config: %s\n", result.ConfigPath)
	} else {
		fmt.Fprintln(w, "Config: defaults")
	}
	fmt.Fprintf(w, "Out: %s\n", result.OutDir)
	fmt.Fprintln(w)

	writeCheckSummary(w, result)
	fmt.Fprintln(w)

	writeCheckArtifacts(w, result.Artifacts)
	fmt.Fprintln(w)

	writeCheckTopFindings(w, result.Project)
	fmt.Fprintln(w)

	writeCheckMostCoupledEdges(w, result)
	fmt.Fprintln(w)

	writeCheckSuspiciousEdges(w, result)
	fmt.Fprintln(w)

	writeCheckNextSteps(w, result)
}

func writeCheckSummary(w io.Writer, result CheckTextResult) {
	summary := result.Summary

	findings := 0
	if result.Project != nil {
		findings = len(result.Project.Findings)
	}

	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  production files: %d\n", summary.ProductionFiles)
	fmt.Fprintf(w, "  test files:       %d\n", summary.TestFiles)
	fmt.Fprintf(w, "  dependencies:     %d\n", summary.TotalEdges)
	fmt.Fprintf(w, "  resolved:         %d\n", summary.Resolved)
	fmt.Fprintf(w, "  unresolved:       %d\n", summary.Unresolved)
	fmt.Fprintf(w, "  findings:         %d\n", findings)
	fmt.Fprintf(w, "  graph nodes:      %d\n", len(result.LayerGraph.Nodes))
	fmt.Fprintf(w, "  graph edges:      %d\n", len(result.LayerGraph.Edges))
}

func writeCheckArtifacts(w io.Writer, artifacts []CheckTextArtifact) {
	fmt.Fprintln(w, "Artifacts:")

	if len(artifacts) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	for _, artifact := range artifacts {
		fmt.Fprintf(w, "  - %s: %s\n", artifact.Name, artifact.Path)
	}
}

func writeCheckTopFindings(w io.Writer, project *model.ProjectModel) {
	fmt.Fprintln(w, "Top findings:")

	if project == nil || len(project.Findings) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	limit := len(project.Findings)
	if limit > maxCheckTopFindings {
		limit = maxCheckTopFindings
	}

	for i := 0; i < limit; i++ {
		finding := project.Findings[i]
		fmt.Fprintf(w, "  - [%s/%s] %s", finding.Severity, finding.Kind, finding.ID)
		if finding.Title != "" {
			fmt.Fprintf(w, " — %s", finding.Title)
		}
		fmt.Fprintln(w)
	}

	if len(project.Findings) > limit {
		fmt.Fprintf(w, "  ... %d more\n", len(project.Findings)-limit)
	}
}

func writeCheckMostCoupledEdges(w io.Writer, result CheckTextResult) {
	fmt.Fprintln(w, "Most coupled edges:")

	edges := sortedCheckEdges(result.LayerGraph.Edges, nil)
	if len(edges) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	limit := len(edges)
	if limit > maxCheckCoupledEdges {
		limit = maxCheckCoupledEdges
	}

	for i := 0; i < limit; i++ {
		edge := edges[i]
		fmt.Fprintf(w, "  %d  %s -> %s\n", edge.Count, edge.From, edge.To)
	}
}

func writeCheckSuspiciousEdges(w io.Writer, result CheckTextResult) {
	fmt.Fprintln(w, "Suspicious edges:")

	suspicious := suspiciousEdgeIndex(result.Project)
	edges := sortedCheckEdges(result.LayerGraph.Edges, suspicious)
	edges = onlySuspiciousEdges(edges)

	if len(edges) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}

	limit := len(edges)
	if limit > maxCheckSuspiciousEdge {
		limit = maxCheckSuspiciousEdge
	}

	for i := 0; i < limit; i++ {
		edge := edges[i]
		if edge.FindingID != "" {
			fmt.Fprintf(w, "  %d  %s -> %s  [%s]\n", edge.Count, edge.From, edge.To, edge.FindingID)
		} else {
			fmt.Fprintf(w, "  %d  %s -> %s\n", edge.Count, edge.From, edge.To)
		}
	}
}

func writeCheckNextSteps(w io.Writer, result CheckTextResult) {
	fmt.Fprintln(w, "Next:")

	projectModelPath := artifactPathByName(result.Artifacts, "project model")
	layerGraphDOTPath := artifactPathByName(result.Artifacts, "layer graph dot")

	wrote := false

	if projectModelPath != "" {
		for _, edge := range suggestedCheckEdges(result) {
			fmt.Fprintf(w, "  patchcourt edge --model %s %s %s\n", projectModelPath, edge.From, edge.To)
			wrote = true
		}
	}

	if result.Project != nil && len(result.Project.Findings) > 0 && projectModelPath != "" {
		first := result.Project.Findings[0]
		if first.ID != "" {
			fmt.Fprintf(w, "  patchcourt explain %s --model %s\n", first.ID, projectModelPath)
			wrote = true
		}
	}

	if layerGraphDOTPath != "" {
		fmt.Fprintf(w, "  dot -Tsvg %s -o %s/layer-graph.svg\n", layerGraphDOTPath, result.OutDir)
		wrote = true
	}

	if !wrote {
		fmt.Fprintln(w, "  none")
	}
}

func suggestedCheckEdges(result CheckTextResult) []checkEdgeSummary {
	suspicious := suspiciousEdgeIndex(result.Project)

	suspiciousEdges := onlySuspiciousEdges(sortedCheckEdges(result.LayerGraph.Edges, suspicious))
	if len(suspiciousEdges) > 0 {
		return limitCheckEdges(suspiciousEdges, maxCheckNextEdges)
	}

	return limitCheckEdges(sortedCheckEdges(result.LayerGraph.Edges, nil), maxCheckNextEdges)
}

func sortedCheckEdges(edges []graphmodel.LayerEdge, suspicious map[string]string) []checkEdgeSummary {
	result := make([]checkEdgeSummary, 0, len(edges))

	for _, edge := range edges {
		if edge.From == "" || edge.To == "" {
			continue
		}

		key := checkEdgeKey(edge.From, edge.To)
		findingID := suspicious[key]

		result = append(result, checkEdgeSummary{
			From:       edge.From,
			To:         edge.To,
			Count:      edge.Count,
			Suspicious: findingID != "",
			FindingID:  findingID,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Suspicious != result[j].Suspicious {
			return result[i].Suspicious
		}

		if result[i].Count == result[j].Count {
			if result[i].From == result[j].From {
				return result[i].To < result[j].To
			}

			return result[i].From < result[j].From
		}

		return result[i].Count > result[j].Count
	})

	return result
}

func onlySuspiciousEdges(edges []checkEdgeSummary) []checkEdgeSummary {
	result := make([]checkEdgeSummary, 0, len(edges))

	for _, edge := range edges {
		if edge.Suspicious {
			result = append(result, edge)
		}
	}

	return result
}

func limitCheckEdges(edges []checkEdgeSummary, limit int) []checkEdgeSummary {
	if limit <= 0 || len(edges) <= limit {
		return edges
	}

	return edges[:limit]
}

func suspiciousEdgeIndex(project *model.ProjectModel) map[string]string {
	result := make(map[string]string)

	if project == nil {
		return result
	}

	for _, finding := range project.Findings {
		if finding.ID == "" {
			continue
		}

		for _, evidence := range finding.Evidence {
			from, to, ok := edgeFromEvidenceMessage(evidence.Message)
			if !ok {
				continue
			}

			key := checkEdgeKey(from, to)
			if _, exists := result[key]; !exists {
				result[key] = finding.ID
			}
		}
	}

	return result
}

func edgeFromEvidenceMessage(message string) (string, string, bool) {
	const marker = " dependency "

	index := strings.LastIndex(message, marker)
	if index == -1 {
		return "", "", false
	}

	edge := strings.TrimSpace(message[index+len(marker):])
	edge = strings.TrimSuffix(edge, ".")
	edge = strings.TrimSuffix(edge, ",")

	if bracket := strings.Index(edge, " ["); bracket != -1 {
		edge = strings.TrimSpace(edge[:bracket])
	}

	parts := strings.Split(edge, " -> ")
	if len(parts) != 2 {
		return "", "", false
	}

	from := strings.TrimSpace(parts[0])
	to := strings.TrimSpace(parts[1])

	if from == "" || to == "" {
		return "", "", false
	}

	return from, to, true
}

func checkEdgeKey(from string, to string) string {
	return from + "->" + to
}

func artifactPathByName(artifacts []CheckTextArtifact, name string) string {
	for _, artifact := range artifacts {
		if artifact.Name == name {
			return artifact.Path
		}
	}

	return ""
}
