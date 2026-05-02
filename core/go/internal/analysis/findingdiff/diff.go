package findingdiff

import (
	"sort"

	"github.com/orurh/patchcourt/internal/model"
)

type FindingChangeKind string

const (
	FindingChangeKindAdded   FindingChangeKind = "added"
	FindingChangeKindRemoved FindingChangeKind = "removed"
)

type FindingChange struct {
	Kind   FindingChangeKind `json:"kind"`
	ID     string            `json:"id"`
	Before *model.Finding    `json:"before,omitempty"`
	After  *model.Finding    `json:"after,omitempty"`
}

func DiffFindings(before []model.Finding, after []model.Finding) []FindingChange {
	beforeIndex := indexFindings(before)
	afterIndex := indexFindings(after)

	keys := mergedSortedKeys(beforeIndex, afterIndex)
	changes := make([]FindingChange, 0)

	for _, key := range keys {
		beforeFinding, hadBefore := beforeIndex[key]
		afterFinding, hasAfter := afterIndex[key]

		switch {
		case !hadBefore && hasAfter:
			afterCopy := afterFinding
			changes = append(changes, FindingChange{
				Kind:  FindingChangeKindAdded,
				ID:    key,
				After: &afterCopy,
			})

		case hadBefore && !hasAfter:
			beforeCopy := beforeFinding
			changes = append(changes, FindingChange{
				Kind:   FindingChangeKindRemoved,
				ID:     key,
				Before: &beforeCopy,
			})
		}
	}

	return changes
}

func indexFindings(findings []model.Finding) map[string]model.Finding {
	index := make(map[string]model.Finding)

	for _, finding := range findings {
		key := FindingKey(finding)
		if key == "" {
			continue
		}

		index[key] = finding
	}

	return index
}

func FindingKey(finding model.Finding) string {
	if finding.ID != "" {
		return finding.ID
	}

	if finding.Title != "" {
		return string(finding.Kind) + "|" + string(finding.Severity) + "|" + finding.Title
	}

	return ""
}

func mergedSortedKeys(left map[string]model.Finding, right map[string]model.Finding) []string {
	seen := make(map[string]struct{}, len(left)+len(right))

	for key := range left {
		seen[key] = struct{}{}
	}

	for key := range right {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}
