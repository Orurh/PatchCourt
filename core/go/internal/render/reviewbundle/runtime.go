package reviewbundle

import (
	"sort"

	findingdiff "github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const runtimeSchemaVersion = "patchcourt.runtime.v1"

type RuntimeReport struct {
	SchemaVersion string              `json:"schema_version"`
	Summary       RuntimeSummary      `json:"summary"`
	Changes       []RuntimeRiskChange `json:"changes"`
}

type RuntimeSummary struct {
	ChangeCount  int `json:"change_count"`
	AddedCount   int `json:"added_count"`
	RemovedCount int `json:"removed_count"`
	ChangedCount int `json:"changed_count"`
	HighCount    int `json:"high_count"`
	MediumCount  int `json:"medium_count"`
	LowCount     int `json:"low_count"`
}

type RuntimeRiskChange struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`

	BeforeSeverity   string `json:"before_severity,omitempty"`
	AfterSeverity    string `json:"after_severity,omitempty"`
	BeforeConfidence string `json:"before_confidence,omitempty"`
	AfterConfidence  string `json:"after_confidence,omitempty"`

	Title      string `json:"title,omitempty"`
	Risk       string `json:"risk,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`

	BeforeEvidenceCount int `json:"before_evidence_count,omitempty"`
	AfterEvidenceCount  int `json:"after_evidence_count,omitempty"`

	AddedEvidence   []model.Evidence `json:"added_evidence,omitempty"`
	RemovedEvidence []model.Evidence `json:"removed_evidence,omitempty"`
	Evidence        []model.Evidence `json:"evidence,omitempty"`
}

func BuildRuntimeReport(result reportmodel.ReviewResult) RuntimeReport {
	changes := make([]RuntimeRiskChange, 0)
	summary := RuntimeSummary{}

	for _, change := range result.FindingChanges {
		finding := runtimeFindingForChange(change)
		if finding == nil {
			continue
		}

		row := RuntimeRiskChange{
			Kind: string(change.Kind),
			ID:   finding.ID,

			Title:      finding.Title,
			Risk:       finding.Risk,
			Suggestion: finding.Suggestion,

			BeforeEvidenceCount: change.BeforeEvidenceCount,
			AfterEvidenceCount:  change.AfterEvidenceCount,

			AddedEvidence:   change.AddedEvidence,
			RemovedEvidence: change.RemovedEvidence,
			Evidence:        runtimeChangeEvidence(change, *finding),
		}

		if change.Before != nil {
			row.BeforeSeverity = string(change.Before.Severity)
			row.BeforeConfidence = string(change.Before.Confidence)
		}

		if change.After != nil {
			row.AfterSeverity = string(change.After.Severity)
			row.AfterConfidence = string(change.After.Confidence)
		}

		changes = append(changes, row)
		updateRuntimeSummary(&summary, change, *finding)
	}

	sort.Slice(changes, func(i, j int) bool {
		if severityRank(changes[i].AfterSeverity, changes[i].BeforeSeverity) != severityRank(changes[j].AfterSeverity, changes[j].BeforeSeverity) {
			return severityRank(changes[i].AfterSeverity, changes[i].BeforeSeverity) > severityRank(changes[j].AfterSeverity, changes[j].BeforeSeverity)
		}

		if changes[i].Kind != changes[j].Kind {
			return changes[i].Kind < changes[j].Kind
		}

		return changes[i].ID < changes[j].ID
	})

	return RuntimeReport{
		SchemaVersion: runtimeSchemaVersion,
		Summary:       summary,
		Changes:       changes,
	}
}

func runtimeFindingForChange(change findingdiff.FindingChange) *model.Finding {
	if change.After != nil && change.After.Kind == model.FindingKindRuntimeRisk {
		return change.After
	}

	if change.Before != nil && change.Before.Kind == model.FindingKindRuntimeRisk {
		return change.Before
	}

	return nil
}

func runtimeChangeEvidence(change findingdiff.FindingChange, finding model.Finding) []model.Evidence {
	switch change.Kind {
	case findingdiff.FindingChangeKindAdded:
		if len(change.AddedEvidence) > 0 {
			return change.AddedEvidence
		}
	case findingdiff.FindingChangeKindRemoved:
		if len(change.RemovedEvidence) > 0 {
			return change.RemovedEvidence
		}
	case findingdiff.FindingChangeKindChanged:
		if len(change.AddedEvidence) > 0 {
			return change.AddedEvidence
		}
	}

	return finding.Evidence
}

func updateRuntimeSummary(summary *RuntimeSummary, change findingdiff.FindingChange, finding model.Finding) {
	summary.ChangeCount++

	switch change.Kind {
	case findingdiff.FindingChangeKindAdded:
		summary.AddedCount++
	case findingdiff.FindingChangeKindRemoved:
		summary.RemovedCount++
	case findingdiff.FindingChangeKindChanged:
		summary.ChangedCount++
	}

	severity := finding.Severity
	if change.After != nil {
		severity = change.After.Severity
	}

	switch severity {
	case model.SeverityHigh, model.SeverityCritical:
		summary.HighCount++
	case model.SeverityMedium:
		summary.MediumCount++
	case model.SeverityLow:
		summary.LowCount++
	}
}

func severityRank(primary string, fallback string) int {
	severity := primary
	if severity == "" {
		severity = fallback
	}

	switch severity {
	case string(model.SeverityCritical):
		return 4
	case string(model.SeverityHigh):
		return 3
	case string(model.SeverityMedium):
		return 2
	case string(model.SeverityLow):
		return 1
	default:
		return 0
	}
}
