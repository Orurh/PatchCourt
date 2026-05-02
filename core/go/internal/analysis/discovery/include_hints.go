package discovery

import (
	"fmt"

	"github.com/orurh/patchcourt/internal/model"
)

const maxUnusedIncludeEvidence = 20

func unusedIncludeHints(project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	evidence := make([]model.Evidence, 0, maxUnusedIncludeEvidence)
	total := 0

	for _, dep := range project.Dependencies {
		if dep.Kind != model.DependencyKindInclude {
			continue
		}

		if dep.External || !dep.Resolved {
			continue
		}

		if dep.Usage != model.DependencyUsageUnused {
			continue
		}

		total++

		if len(evidence) >= maxUnusedIncludeEvidence {
			continue
		}

		target := dep.ToFile
		if target == "" {
			target = dep.Target
		}

		message := fmt.Sprintf(
			"includes %s, but no declared top-level symbols from that header were referenced by the includer",
			target,
		)

		if dep.ResolutionSource != "" || dep.ResolutionConfidence != "" {
			message = fmt.Sprintf(
				"%s [%s/%s]",
				message,
				dep.ResolutionSource,
				dep.ResolutionConfidence,
			)
		}

		evidence = append(evidence, model.Evidence{
			File:    dep.FromFile,
			Message: message,
		})
	}

	if total == 0 {
		return nil
	}

	title := "Possibly unused C++ includes"
	if total == 1 {
		title = "Possibly unused C++ include"
	}

	return []model.Finding{
		{
			ID:         "discovery.cpp.unused_includes",
			Kind:       model.FindingKindDiscoveryHint,
			Severity:   model.SeverityLow,
			Title:      title,
			Confidence: model.ConfidenceMedium,
			Risk:       "These include dependencies were resolved, but lightweight syntactic analysis did not find references to top-level symbols declared in the included headers. This may indicate avoidable compile-time coupling.",
			Suggestion: "Review whether these includes can be removed or replaced with forward declarations. Verify manually, especially for macro-heavy, template-heavy, or transitive include cases.",
			Evidence:   evidence,
		},
	}
}
