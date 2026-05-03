package risk

import (
	"fmt"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/depdiff"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
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
		if change.Kind != findingdiff.FindingChangeKindAdded || change.After == nil {
			continue
		}

		points := severityPoints(change.After.Severity)
		if change.After.Kind == model.FindingKindPolicyViolation {
			points += 2
		}

		addReason(
			&score,
			points,
			fmt.Sprintf("added %s %s: %s", change.After.Severity, model.HumanFindingKind(change.After.Kind), change.ID),
		)
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
			addReason(&score, 1, fmt.Sprintf("public contract symbol added: %s", change.SymbolKey))
		}
	}

	for _, change := range input.DependencyChanges {
		if change.Kind != depdiff.DependencyChangeKindAdded {
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
