package report

import (
	"fmt"
	"io"

	"github.com/orurh/patchcourt/internal/model"
)

func WriteScanText(w io.Writer, project *model.ProjectModel) {
	summary := model.BuildScanSummary(project)

	fmt.Fprintln(w, "PatchCourt scan")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Root: %s\n", project.Root)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Files:")
	fmt.Fprintf(w, "  C++ headers: %d\n", summary.CPPHeaders)
	fmt.Fprintf(w, "  C++ sources: %d\n", summary.CPPSources)
	fmt.Fprintf(w, "  C++ tests:   %d\n", summary.CPPTests)
	fmt.Fprintf(w, "  Go files:    %d\n", summary.GoFiles)
	fmt.Fprintf(w, "  symbols:     %d\n", summary.Symbols)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "File roles:")
	fmt.Fprintf(w, "  production: %d\n", summary.ProductionFiles)
	fmt.Fprintf(w, "  test:       %d\n", summary.TestFiles)
	fmt.Fprintf(w, "  generated:  %d\n", summary.GeneratedFiles)
	fmt.Fprintf(w, "  external:   %d\n", summary.ExternalFiles)
	fmt.Fprintf(w, "  config:     %d\n", summary.ConfigFiles)
	fmt.Fprintf(w, "  unknown:    %d\n", summary.UnknownFiles)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Dependencies:")
	fmt.Fprintf(w, "  total edges: %d\n", summary.TotalEdges)
	fmt.Fprintf(w, "  resolved:    %d\n", summary.Resolved)
	fmt.Fprintf(w, "  unresolved:  %d\n", summary.Unresolved)
	fmt.Fprintf(w, "  external:    %d\n", summary.External)

	writeResolutionDiagnosticsText(w, project)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Findings:")
	fmt.Fprintf(w, "  total: %d\n", len(project.Findings))

	for _, finding := range project.Findings {
		fmt.Fprintf(w, "  [%s] %s\n", finding.Severity, finding.Title)
		for _, evidence := range finding.Evidence {
			fmt.Fprintf(w, "    - %s: %s\n", evidence.File, evidence.Message)
		}
	}
}

func writeResolutionDiagnosticsText(w io.Writer, project *model.ProjectModel) {
	unresolved := unresolvedDependencies(project)
	ambiguous := ambiguousDependencies(project)

	if len(unresolved) == 0 && len(ambiguous) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Resolution diagnostics:")

	if len(unresolved) > 0 {
		fmt.Fprintf(w, "  unresolved includes: %d\n", len(unresolved))
		for _, dep := range unresolved {
			fmt.Fprintf(w, "    - %s: %s", dep.FromFile, dep.Target)
			if dep.ResolutionSource != "" {
				fmt.Fprintf(w, " [%s/%s]", dep.ResolutionSource, dep.ResolutionConfidence)
			}
			fmt.Fprintln(w)
		}
	}

	if len(ambiguous) > 0 {
		fmt.Fprintf(w, "  ambiguous includes: %d\n", len(ambiguous))
		for _, dep := range ambiguous {
			fmt.Fprintf(w, "    - %s: %s", dep.FromFile, dep.Target)
			if dep.ResolutionSource != "" {
				fmt.Fprintf(w, " [%s/%s]", dep.ResolutionSource, dep.ResolutionConfidence)
			}
			fmt.Fprintln(w)

			for _, candidate := range dep.Candidates {
				fmt.Fprintf(w, "        candidate: %s\n", candidate)
			}
		}
	}
}

func unresolvedDependencies(project *model.ProjectModel) []model.DependencyEdge {
	result := make([]model.DependencyEdge, 0)

	for _, dep := range project.Dependencies {
		if dep.External || dep.Resolved || dep.Ambiguous {
			continue
		}

		result = append(result, dep)
	}

	return result
}

func ambiguousDependencies(project *model.ProjectModel) []model.DependencyEdge {
	result := make([]model.DependencyEdge, 0)

	for _, dep := range project.Dependencies {
		if dep.External || !dep.Ambiguous {
			continue
		}

		result = append(result, dep)
	}

	return result
}
