package runtime

import "github.com/orurh/patchcourt/internal/model"

func emitRuntimeSite(site RuntimeSite, builders map[string]*findingBuilder) {
	if site.Score <= 0 || site.FindingID == "" || site.Severity == "" {
		return
	}

	builder := builders[site.FindingID]
	if builder == nil {
		return
	}

	if len(builder.finding.Evidence) == 0 {
		builder.finding.Severity = site.Severity
		builder.finding.Confidence = site.Confidence
	} else {
		builder.finding.Severity = maxSeverity(builder.finding.Severity, site.Severity)
		builder.finding.Confidence = maxConfidence(builder.finding.Confidence, site.Confidence)
	}

	addEvidence(builder, model.Evidence{
		File:      site.File,
		LineStart: site.Line,
		Snippet:   site.Snippet,
		Message:   site.Message,
	})
}
func maxSeverity(left model.Severity, right model.Severity) model.Severity {
	if model.SeverityRank(right) > model.SeverityRank(left) {
		return right
	}

	return left
}
func maxConfidence(left model.Confidence, right model.Confidence) model.Confidence {
	if runtimeConfidenceRank(right) > runtimeConfidenceRank(left) {
		return right
	}

	return left
}
func runtimeConfidenceRank(confidence model.Confidence) int {
	switch confidence {
	case model.ConfidenceHigh:
		return 3
	case model.ConfidenceMedium:
		return 2
	case model.ConfidenceLow:
		return 1
	default:
		return 0
	}
}
