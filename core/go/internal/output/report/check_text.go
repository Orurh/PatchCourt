package report

import (
	"fmt"
	"io"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/model"
)

const maxCheckTopFindings = 5

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

func writeCheckNextSteps(w io.Writer, result CheckTextResult) {
	fmt.Fprintln(w, "Next:")

	projectModelPath := artifactPathByName(result.Artifacts, "project model")
	layerGraphDOTPath := artifactPathByName(result.Artifacts, "layer graph dot")

	if result.Project != nil && len(result.Project.Findings) > 0 && projectModelPath != "" {
		first := result.Project.Findings[0]
		if first.ID != "" {
			fmt.Fprintf(w, "  patchcourt explain %s --model %s\n", first.ID, projectModelPath)
		}
	}

	if layerGraphDOTPath != "" {
		fmt.Fprintf(w, "  dot -Tsvg %s -o %s/layer-graph.svg\n", layerGraphDOTPath, result.OutDir)
	}
}

func artifactPathByName(artifacts []CheckTextArtifact, name string) string {
	for _, artifact := range artifacts {
		if artifact.Name == name {
			return artifact.Path
		}
	}

	return ""
}
