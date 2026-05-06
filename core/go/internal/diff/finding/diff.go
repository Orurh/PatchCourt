package findingdiff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type FindingChangeKind string

const (
	FindingChangeKindAdded   FindingChangeKind = "added"
	FindingChangeKindRemoved FindingChangeKind = "removed"
	FindingChangeKindChanged FindingChangeKind = "changed"
)

type FindingChange struct {
	Kind   FindingChangeKind `json:"kind"`
	ID     string            `json:"id"`
	Before *model.Finding    `json:"before,omitempty"`
	After  *model.Finding    `json:"after,omitempty"`

	BeforeEvidenceCount int              `json:"before_evidence_count,omitempty"`
	AfterEvidenceCount  int              `json:"after_evidence_count,omitempty"`
	AddedEvidence       []model.Evidence `json:"added_evidence,omitempty"`
	RemovedEvidence     []model.Evidence `json:"removed_evidence,omitempty"`
	SeverityChanged     bool             `json:"severity_changed,omitempty"`
	ConfidenceChanged   bool             `json:"confidence_changed,omitempty"`
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
				Kind:               FindingChangeKindAdded,
				ID:                 key,
				After:              &afterCopy,
				AfterEvidenceCount: len(afterFinding.Evidence),
				AddedEvidence:      sortedEvidenceCopy(afterFinding.Evidence),
			})

		case hadBefore && !hasAfter:
			beforeCopy := beforeFinding
			changes = append(changes, FindingChange{
				Kind:                FindingChangeKindRemoved,
				ID:                  key,
				Before:              &beforeCopy,
				BeforeEvidenceCount: len(beforeFinding.Evidence),
				RemovedEvidence:     sortedEvidenceCopy(beforeFinding.Evidence),
			})

		case hadBefore && hasAfter:
			change, ok := changedFinding(key, beforeFinding, afterFinding)
			if ok {
				changes = append(changes, change)
			}
		}
	}

	return changes
}

func changedFinding(key string, before model.Finding, after model.Finding) (FindingChange, bool) {
	addedEvidence, removedEvidence := diffEvidence(before.Evidence, after.Evidence)

	changed := before.Kind != after.Kind ||
		before.Severity != after.Severity ||
		before.Title != after.Title ||
		before.Risk != after.Risk ||
		before.Suggestion != after.Suggestion ||
		before.Confidence != after.Confidence ||
		len(addedEvidence) > 0 ||
		len(removedEvidence) > 0

	if !changed {
		return FindingChange{}, false
	}

	beforeCopy := before
	afterCopy := after

	return FindingChange{
		Kind:                FindingChangeKindChanged,
		ID:                  key,
		Before:              &beforeCopy,
		After:               &afterCopy,
		BeforeEvidenceCount: len(before.Evidence),
		AfterEvidenceCount:  len(after.Evidence),
		AddedEvidence:       addedEvidence,
		RemovedEvidence:     removedEvidence,
		SeverityChanged:     before.Severity != after.Severity,
		ConfidenceChanged:   before.Confidence != after.Confidence,
	}, true
}

func diffEvidence(before []model.Evidence, after []model.Evidence) (added []model.Evidence, removed []model.Evidence) {
	beforeIndex := indexEvidence(before)
	afterIndex := indexEvidence(after)

	for key, evidence := range afterIndex {
		if _, ok := beforeIndex[key]; !ok {
			added = append(added, evidence)
		}
	}

	for key, evidence := range beforeIndex {
		if _, ok := afterIndex[key]; !ok {
			removed = append(removed, evidence)
		}
	}

	sortEvidence(added)
	sortEvidence(removed)

	return added, removed
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

func indexEvidence(items []model.Evidence) map[string]model.Evidence {
	index := make(map[string]model.Evidence, len(items))

	for _, item := range items {
		key := EvidenceKey(item)
		if key == "" {
			continue
		}

		index[key] = item
	}

	return index
}

func EvidenceKey(evidence model.Evidence) string {
	parts := []string{
		evidence.File,
		fmt.Sprintf("%d", evidence.LineStart),
		fmt.Sprintf("%d", evidence.LineEnd),
		evidence.Snippet,
		evidence.Message,
		evidence.FromLayer,
		evidence.ToLayer,
		evidence.FromFile,
		evidence.ToFile,
	}

	key := strings.Join(parts, "|")
	if strings.Trim(key, "|") == "" {
		return ""
	}

	return key
}

func sortedEvidenceCopy(items []model.Evidence) []model.Evidence {
	result := append([]model.Evidence(nil), items...)
	sortEvidence(result)
	return result
}

func sortEvidence(items []model.Evidence) {
	sort.SliceStable(items, func(i int, j int) bool {
		return EvidenceKey(items[i]) < EvidenceKey(items[j])
	})
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
