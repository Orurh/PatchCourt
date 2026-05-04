package projectdiff

import (
	"sort"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
)

type Result struct {
	ChangedFiles      []string                    `json:"changed_files,omitempty"`
	ContractChanges   []contracts.SymbolChange    `json:"contract_changes"`
	DependencyChanges []depdiff.DependencyChange  `json:"dependency_changes"`
	LayerEdgeChanges  []depdiff.LayerEdgeChange   `json:"layer_edge_changes"`
	FindingChanges    []findingdiff.FindingChange `json:"finding_changes"`
	Risk              risk.Score                  `json:"risk"`
}

func Compare(before *model.ProjectModel, after *model.ProjectModel) Result {
	if before == nil {
		before = &model.ProjectModel{}
	}

	if after == nil {
		after = &model.ProjectModel{}
	}

	contractChanges := contracts.DiffSymbols(before.Symbols, after.Symbols)
	dependencyChanges := depdiff.DiffDependencies(before.Dependencies, after.Dependencies)
	layerEdgeChanges := depdiff.DiffLayerEdges(before.Dependencies, after.Dependencies)
	findingChanges := findingdiff.DiffFindings(before.Findings, after.Findings)
	changedFiles := DiffChangedFiles(before, after)
	changedFiles = MergeChangedFiles(changedFiles, changedFilesFromDiffs(contractChanges, dependencyChanges, findingChanges))

	reviewRisk := risk.Calculate(risk.Input{
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
	})

	return Result{
		ChangedFiles:      changedFiles,
		ContractChanges:   contractChanges,
		DependencyChanges: dependencyChanges,
		LayerEdgeChanges:  layerEdgeChanges,
		FindingChanges:    findingChanges,
		Risk:              reviewRisk,
	}
}

func DiffChangedFiles(before *model.ProjectModel, after *model.ProjectModel) []string {
	beforeIndex := indexFiles(before.Files)
	afterIndex := indexFiles(after.Files)

	seen := make(map[string]struct{}, len(beforeIndex)+len(afterIndex))

	for path := range beforeIndex {
		seen[path] = struct{}{}
	}

	for path := range afterIndex {
		seen[path] = struct{}{}
	}

	changed := make([]string, 0)

	for path := range seen {
		beforeFile, hadBefore := beforeIndex[path]
		afterFile, hasAfter := afterIndex[path]

		switch {
		case !hadBefore && hasAfter:
			changed = append(changed, path)
		case hadBefore && !hasAfter:
			changed = append(changed, path)
		case hadBefore && hasAfter && fileModelChanged(beforeFile, afterFile):
			changed = append(changed, path)
		}
	}

	sort.Strings(changed)
	return changed
}

func indexFiles(files []model.FileModel) map[string]model.FileModel {
	index := make(map[string]model.FileModel, len(files))

	for _, file := range files {
		if file.Path == "" {
			continue
		}

		index[file.Path] = file
	}

	return index
}

func fileModelChanged(before model.FileModel, after model.FileModel) bool {
	if before.Language != after.Language ||
		before.Kind != after.Kind ||
		before.Role != after.Role ||
		before.Layer != after.Layer ||
		before.LayerSource != after.LayerSource ||
		before.IsTest != after.IsTest {
		return true
	}

	if !sameStrings(before.Imports, after.Imports) {
		return true
	}

	if !sameStrings(before.Includes, after.Includes) {
		return true
	}

	if !sameSymbols(before.Symbols, after.Symbols) {
		return true
	}

	return false
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)

	sort.Strings(leftCopy)
	sort.Strings(rightCopy)

	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}

	return true
}

func sameSymbols(left []model.SymbolModel, right []model.SymbolModel) bool {
	if len(left) != len(right) {
		return false
	}

	leftKeys := make([]string, 0, len(left))
	for _, symbol := range left {
		leftKeys = append(leftKeys, symbolKey(symbol))
	}

	rightKeys := make([]string, 0, len(right))
	for _, symbol := range right {
		rightKeys = append(rightKeys, symbolKey(symbol))
	}

	sort.Strings(leftKeys)
	sort.Strings(rightKeys)

	for i := range leftKeys {
		if leftKeys[i] != rightKeys[i] {
			return false
		}
	}

	return true
}

func symbolKey(symbol model.SymbolModel) string {
	return string(symbol.Kind) + "|" + symbol.Parent + "|" + symbol.Name + "|" + symbol.Signature
}

func changedFilesFromDiffs(
	contractChanges []contracts.SymbolChange,
	dependencyChanges []depdiff.DependencyChange,
	findingChanges []findingdiff.FindingChange,
) []string {
	seen := make(map[string]struct{})

	for _, change := range contractChanges {
		if change.Before != nil {
			addChangedFile(seen, change.Before.File)
		}
		if change.After != nil {
			addChangedFile(seen, change.After.File)
		}
	}

	for _, change := range dependencyChanges {
		if change.Before != nil {
			addChangedFile(seen, change.Before.FromFile)
		}
		if change.After != nil {
			addChangedFile(seen, change.After.FromFile)
		}
	}

	for _, change := range findingChanges {
		if change.Before != nil {
			addFindingEvidenceFiles(seen, change.Before.Evidence)
		}
		if change.After != nil {
			addFindingEvidenceFiles(seen, change.After.Evidence)
		}
	}

	return sortedChangedFiles(seen)
}

func addFindingEvidenceFiles(seen map[string]struct{}, evidence []model.Evidence) {
	for _, item := range evidence {
		addChangedFile(seen, item.File)
		addChangedFile(seen, item.FromFile)
	}
}

func MergeChangedFiles(primary []string, extra []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(extra))

	for _, file := range primary {
		addChangedFile(seen, file)
	}
	for _, file := range extra {
		addChangedFile(seen, file)
	}

	return sortedChangedFiles(seen)
}

func addChangedFile(seen map[string]struct{}, file string) {
	if file == "" {
		return
	}

	seen[file] = struct{}{}
}

func sortedChangedFiles(seen map[string]struct{}) []string {
	files := make([]string, 0, len(seen))

	for file := range seen {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}
