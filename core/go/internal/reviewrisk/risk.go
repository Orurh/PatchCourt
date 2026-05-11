package risk

import (
	"fmt"
	"strings"

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

type ContractImpact struct {
	SymbolKey        string
	ChangeKind       string
	TestsChanged     bool
	DeliveryImpacted bool
	ImpactedFiles    []ContractImpactedFile
}

type ContractImpactedFile struct {
	File string
}

type Input struct {
	ContractChanges   []contracts.SymbolChange
	ContractImpacts   []ContractImpact
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

	scoreContractChanges(&score, input.ContractChanges, input.ContractImpacts)

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

func scoreContractChanges(score *Score, changes []contracts.SymbolChange, impacts []ContractImpact) {
	impactIndex := contractImpactIndex(impacts)
	hardRemovedParentMethods := hardRemovedMethodCountsByRemovedParent(changes, impactIndex)

	for _, change := range changes {
		impact := impactIndex[contractImpactKey(string(change.Kind), change.SymbolKey)]

		switch change.Kind {
		case contracts.ChangeKindSignatureChanged:
			if isHardRiskContractImpact(impact) {
				addReason(score, 2, fmt.Sprintf("public contract signature changed with impacted callers: %s", change.SymbolKey))
			}

		case contracts.ChangeKindModifiersChanged:
			if isHardRiskContractImpact(impact) {
				addReason(score, 1, fmt.Sprintf("public contract modifiers changed with impacted callers: %s", change.SymbolKey))
			}

		case contracts.ChangeKindRemoved:
			parentName, methodName := contractSymbolParts(change.SymbolKey)
			if methodName != "" && hardRemovedParentMethods[parentName] > 0 {
				continue
			}

			methodCount := 0
			if methodName == "" {
				methodCount = hardRemovedParentMethods[parentName]
			}

			if methodCount == 0 && !isHardRiskContractImpact(impact) {
				continue
			}

			addReason(score, 3, removedContractReasonMessage(change.SymbolKey, methodCount))

		case contracts.ChangeKindAdded:
			// Adding a public symbol is not inherently risky. Risk is driven by
			// removals, signature/modifier changes, new findings, and dependency
			// direction changes.
		}
	}
}

func removedContractReasonMessage(symbolKey string, removedMethodCount int) string {
	if removedMethodCount <= 0 {
		return fmt.Sprintf("public contract symbol removed with impacted callers: %s", symbolKey)
	}

	return fmt.Sprintf(
		"contract boundary changed: %s removed with %d delivery/API-impacted methods",
		symbolKey,
		removedMethodCount,
	)
}

func hardRemovedMethodCountsByRemovedParent(
	changes []contracts.SymbolChange,
	impacts map[string]ContractImpact,
) map[string]int {
	removedParents := make(map[string]struct{})
	methodCounts := make(map[string]int)

	for _, change := range changes {
		if change.Kind != contracts.ChangeKindRemoved {
			continue
		}

		parentName, methodName := contractSymbolParts(change.SymbolKey)
		if parentName == "" {
			continue
		}

		if methodName == "" {
			removedParents[parentName] = struct{}{}
			continue
		}

		impact := impacts[contractImpactKey(string(change.Kind), change.SymbolKey)]
		if isHardRiskContractImpact(impact) {
			methodCounts[parentName]++
		}
	}

	result := make(map[string]int)
	for parentName := range removedParents {
		if count := methodCounts[parentName]; count > 0 {
			result[parentName] = count
		}
	}

	return result
}

func contractImpactIndex(impacts []ContractImpact) map[string]ContractImpact {
	index := make(map[string]ContractImpact, len(impacts))

	for _, impact := range impacts {
		index[contractImpactKey(impact.ChangeKind, impact.SymbolKey)] = impact
	}

	return index
}

func contractImpactKey(kind string, symbolKey string) string {
	return kind + "|" + symbolKey
}

func isHardRiskContractImpact(impact ContractImpact) bool {
	if impact.SymbolKey == "" {
		return false
	}

	if impact.DeliveryImpacted {
		return true
	}

	return len(impact.ImpactedFiles) > 0 && !impact.TestsChanged
}

func contractSymbolParts(symbolKey string) (parentName string, methodName string) {
	parts := strings.Split(symbolKey, "::")
	if len(parts) < 2 {
		return "", ""
	}

	if len(parts) >= 3 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}

	return parts[len(parts)-1], ""
}

func scoreAddedFinding(score *Score, change findingdiff.FindingChange) {
	if change.After == nil {
		return
	}
	if change.After.Kind == model.FindingKindDiscoveryHint {
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
	if change.Before.Kind == model.FindingKindDiscoveryHint || change.After.Kind == model.FindingKindDiscoveryHint {
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

	evidenceDelta := len(change.AddedEvidence) - len(change.RemovedEvidence)
	if evidenceDelta > 0 {
		scoreChangedFindingEvidence(score, change, evidenceDelta)
	}

	if confidenceRank(change.After.Confidence) > confidenceRank(change.Before.Confidence) &&
		evidenceDelta <= 0 &&
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

func scoreChangedFindingEvidence(score *Score, change findingdiff.FindingChange, evidenceDelta int) {
	if change.After == nil || evidenceDelta <= 0 {
		return
	}

	switch change.After.Kind {
	case model.FindingKindDiscoveryHint:
		points := evidenceDelta
		if points > 2 {
			points = 2
		}

		addReason(
			score,
			points,
			fmt.Sprintf("discovery signal gained evidence: %s (+%d net evidence)", change.ID, evidenceDelta),
		)

	default:
		points := evidenceDelta
		if points > 3 {
			points = 3
		}

		addReason(
			score,
			points,
			fmt.Sprintf("finding evidence increased: %s (+%d net evidence)", change.ID, evidenceDelta),
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
