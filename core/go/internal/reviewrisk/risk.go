package risk

import (
	"fmt"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/diff/finding"
	"github.com/orurh/patchcourt/internal/model"
)

type Level string

const (
	LevelLow      Level = "low"
	LevelMedium   Level = "medium"
	LevelHigh     Level = "high"
	LevelCritical Level = "critical"
)

type Reason struct {
	Message string `json:"message"`
	Points  int    `json:"points"`
}

type Score struct {
	Points  int      `json:"points"`
	Level   Level    `json:"level"`
	Reasons []Reason `json:"reasons,omitempty"`
}

type Input struct {
	ContractChanges   []contracts.SymbolChange
	DependencyChanges []depdiff.DependencyChange
	LayerEdgeChanges  []depdiff.LayerEdgeChange
	FindingChanges    []findingdiff.FindingChange
}

func Calculate(input Input) Score {
	var score Score

	for _, change := range input.FindingChanges {
		switch change.Kind {
		case findingdiff.FindingChangeKindAdded:
			scoreAddedFinding(&score, change)

		case findingdiff.FindingChangeKindChanged:
			scoreChangedFinding(&score, change)
		}
	}

	for _, change := range input.ContractChanges {
		switch change.Kind {
		case contracts.ChangeKindSignatureChanged:
			addReason(&score, 2, fmt.Sprintf("public contract signature changed: %s", change.SymbolKey))
		case contracts.ChangeKindModifiersChanged:
			addReason(&score, 1, fmt.Sprintf("public contract modifiers changed: %s", change.SymbolKey))
		case contracts.ChangeKindRemoved:
			addReason(&score, 3, fmt.Sprintf("public contract symbol removed: %s", change.SymbolKey))
		case contracts.ChangeKindAdded:
			// Adding a public symbol is not inherently risky. Risk is driven by
			// removals, signature/modifier changes, new findings, and dependency
			// direction changes.
		}
	}

	for _, change := range input.DependencyChanges {
		if change.Kind != depdiff.DependencyChangeKindAdded || change.After == nil {
			continue
		}

		if !isRiskyAddedDependency(*change.After) {
			continue
		}

		addReason(&score, 1, fmt.Sprintf("dependency edge added: %s", change.Key))
	}

	for _, change := range input.LayerEdgeChanges {
		switch change.Kind {
		case depdiff.DependencyChangeKindAdded:
			addReason(&score, 3, fmt.Sprintf("layer edge added: %s -> %s", change.FromLayer, change.ToLayer))
		case depdiff.DependencyChangeKindChanged:
			if change.AfterCount > change.BeforeCount {
				addReason(
					&score,
					1,
					fmt.Sprintf(
						"layer edge count increased: %s -> %s (%d -> %d)",
						change.FromLayer,
						change.ToLayer,
						change.BeforeCount,
						change.AfterCount,
					),
				)
			}
		}
	}

	score.Level = levelForPoints(score.Points)
	return score
}

func scoreAddedFinding(score *Score, change findingdiff.FindingChange) {
	if change.After == nil {
		return
	}

	points := severityPoints(change.After.Severity)
	if change.After.Kind == model.FindingKindPolicyViolation {
		points += 2
	}

	addReason(
		score,
		points,
		fmt.Sprintf("added %s %s: %s", change.After.Severity, model.HumanFindingKind(change.After.Kind), change.ID),
	)
}

func scoreChangedFinding(score *Score, change findingdiff.FindingChange) {
	if change.Before == nil || change.After == nil {
		return
	}

	beforeSeverityPoints := severityPoints(change.Before.Severity)
	afterSeverityPoints := severityPoints(change.After.Severity)

	if afterSeverityPoints > beforeSeverityPoints {
		addReason(
			score,
			afterSeverityPoints-beforeSeverityPoints,
			fmt.Sprintf(
				"finding severity increased: %s (%s -> %s)",
				change.ID,
				change.Before.Severity,
				change.After.Severity,
			),
		)
	}

	if len(change.AddedEvidence) > 0 {
		points := len(change.AddedEvidence)
		if points > 3 {
			points = 3
		}

		addReason(
			score,
			points,
			fmt.Sprintf("finding evidence increased: %s (+%d evidence)", change.ID, len(change.AddedEvidence)),
		)
	}

	if confidenceRank(change.After.Confidence) > confidenceRank(change.Before.Confidence) &&
		len(change.AddedEvidence) == 0 &&
		afterSeverityPoints <= beforeSeverityPoints {
		addReason(
			score,
			1,
			fmt.Sprintf(
				"finding confidence increased: %s (%s -> %s)",
				change.ID,
				change.Before.Confidence,
				change.After.Confidence,
			),
		)
	}
}

func severityPoints(severity model.Severity) int {
	switch severity {
	case model.SeverityCritical:
		return 10
	case model.SeverityHigh:
		return 5
	case model.SeverityMedium:
		return 3
	case model.SeverityLow:
		return 1
	default:
		return 1
	}
}

func confidenceRank(confidence model.Confidence) int {
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

func addReason(score *Score, points int, message string) {
	if points <= 0 {
		return
	}

	score.Points += points
	score.Reasons = append(score.Reasons, Reason{
		Message: message,
		Points:  points,
	})
}

func levelForPoints(points int) Level {
	switch {
	case points >= 15:
		return LevelCritical
	case points >= 8:
		return LevelHigh
	case points >= 3:
		return LevelMedium
	default:
		return LevelLow
	}
}

func isRiskyAddedDependency(dep model.DependencyEdge) bool {
	if dep.External {
		return false
	}

	if dep.FromLayer == "" || dep.ToLayer == "" {
		return false
	}

	if dep.FromLayer == dep.ToLayer {
		return false
	}

	return true
}
