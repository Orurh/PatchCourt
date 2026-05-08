package reviewbundle

import (
	"path"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
)

const treeSchemaVersion = "patchcourt.project_tree.v1"

type ProjectTreeReport struct {
	SchemaVersion string          `json:"schema_version"`
	Root          ProjectTreeNode `json:"root"`
}

type ProjectTreeNode struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
	Kind string `json:"kind"`

	Language string `json:"language,omitempty"`
	Layer    string `json:"layer,omitempty"`
	Role     string `json:"role,omitempty"`

	ChangeKind string `json:"change_kind,omitempty"`

	ChangedFilesCount   int `json:"changed_files_count,omitempty"`
	FindingCount        int `json:"finding_count,omitempty"`
	RuntimeFindingCount int `json:"runtime_finding_count,omitempty"`
	RiskPoints          int `json:"risk_points,omitempty"`

	Children []ProjectTreeNode `json:"children,omitempty"`
}

type mutableTreeNode struct {
	Name string
	Path string
	Kind string

	Language string
	Layer    string
	Role     string

	ChangeKind string

	ChangedFilesCount   int
	FindingCount        int
	RuntimeFindingCount int
	RiskPoints          int

	children map[string]*mutableTreeNode
}

func BuildProjectTree(result reportmodel.ReviewResult) ProjectTreeReport {
	root := &mutableTreeNode{
		Name:     ".",
		Path:     "",
		Kind:     "dir",
		children: make(map[string]*mutableTreeNode),
	}

	beforeFiles := fileIndex(result.BeforeProject)
	afterFiles := fileIndex(result.AfterProject)
	allFiles := mergedFilePaths(beforeFiles, afterFiles)
	changedFiles := changedFileSet(result.ChangedFiles)

	findingStats := findingStatsByFile(result)

	for _, filePath := range allFiles {
		beforeFile, hadBefore := beforeFiles[filePath]
		afterFile, hasAfter := afterFiles[filePath]

		file := afterFile
		if !hasAfter {
			file = beforeFile
		}

		node := ensureTreeFile(root, filePath)
		node.Language = string(file.Language)
		node.Layer = file.Layer
		node.Role = string(file.Role)
		node.ChangeKind = fileChangeKind(filePath, hadBefore, hasAfter, changedFiles)

		if stats, ok := findingStats[filePath]; ok {
			node.FindingCount = stats.FindingCount
			node.RuntimeFindingCount = stats.RuntimeFindingCount
			node.RiskPoints = stats.RiskPoints
		}
	}

	aggregateTree(root)

	return ProjectTreeReport{
		SchemaVersion: treeSchemaVersion,
		Root:          freezeTreeNode(root),
	}
}

type treeFileFindingStats struct {
	FindingCount        int
	RuntimeFindingCount int
	RiskPoints          int
}

func fileIndex(project *model.ProjectModel) map[string]model.FileModel {
	index := make(map[string]model.FileModel)
	if project == nil {
		return index
	}

	for _, file := range project.Files {
		if file.Path == "" {
			continue
		}

		index[file.Path] = file
	}

	return index
}

func mergedFilePaths(before map[string]model.FileModel, after map[string]model.FileModel) []string {
	seen := make(map[string]struct{}, len(before)+len(after))

	for file := range before {
		seen[file] = struct{}{}
	}
	for file := range after {
		seen[file] = struct{}{}
	}

	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}

func ensureTreeFile(root *mutableTreeNode, filePath string) *mutableTreeNode {
	cleanPath := strings.Trim(filePath, "/")
	if cleanPath == "" {
		return root
	}

	parts := strings.Split(cleanPath, "/")
	current := root

	for i, part := range parts {
		currentPath := strings.Join(parts[:i+1], "/")
		kind := "dir"
		if i == len(parts)-1 {
			kind = "file"
		}

		child, ok := current.children[part]
		if !ok {
			child = &mutableTreeNode{
				Name:     part,
				Path:     currentPath,
				Kind:     kind,
				children: make(map[string]*mutableTreeNode),
			}
			current.children[part] = child
		}

		current = child
	}

	return current
}

func changedFileSet(files []string) map[string]struct{} {
	set := make(map[string]struct{}, len(files))

	for _, file := range files {
		file = strings.Trim(file, "/")
		if file == "" {
			continue
		}

		set[file] = struct{}{}
	}

	return set
}

func fileChangeKind(filePath string, hadBefore bool, hasAfter bool, changedFiles map[string]struct{}) string {
	switch {
	case !hadBefore && hasAfter:
		return "added"
	case hadBefore && !hasAfter:
		return "removed"
	case hadBefore && hasAfter:
		if _, ok := changedFiles[strings.Trim(filePath, "/")]; ok {
			return "modified"
		}

		return ""
	default:
		return ""
	}
}

func findingStatsByFile(result reportmodel.ReviewResult) map[string]treeFileFindingStats {
	stats := make(map[string]treeFileFindingStats)

	for _, change := range result.FindingChanges {
		finding := change.After
		if finding == nil {
			finding = change.Before
		}
		if finding == nil {
			continue
		}

		files := evidenceFilesForTree(*finding)
		if len(files) == 0 {
			continue
		}

		points := severityRiskPoints(finding.Severity)
		isRuntime := finding.Kind == model.FindingKindRuntimeRisk

		for _, file := range files {
			current := stats[file]
			current.FindingCount++
			current.RiskPoints += points
			if isRuntime {
				current.RuntimeFindingCount++
			}
			stats[file] = current
		}
	}

	return stats
}

func evidenceFilesForTree(finding model.Finding) []string {
	seen := make(map[string]struct{})
	files := make([]string, 0)

	for _, evidence := range finding.Evidence {
		file := evidence.File
		if file == "" {
			file = evidence.FromFile
		}
		if file == "" {
			continue
		}

		file = strings.Trim(file, "/")
		if file == "" {
			continue
		}

		if _, ok := seen[file]; ok {
			continue
		}

		seen[file] = struct{}{}
		files = append(files, file)
	}

	sort.Strings(files)
	return files
}

func aggregateTree(node *mutableTreeNode) {
	if node == nil {
		return
	}

	if node.Kind == "file" {
		if node.ChangeKind != "" && node.ChangeKind != "unchanged" {
			node.ChangedFilesCount = 1
		}
		return
	}

	children := make([]*mutableTreeNode, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		left := children[i]
		right := children[j]

		if left.Kind != right.Kind {
			return left.Kind == "dir"
		}

		return left.Name < right.Name
	})

	for _, child := range children {
		aggregateTree(child)

		node.ChangedFilesCount += child.ChangedFilesCount
		node.FindingCount += child.FindingCount
		node.RuntimeFindingCount += child.RuntimeFindingCount
		node.RiskPoints += child.RiskPoints
	}
}

func freezeTreeNode(node *mutableTreeNode) ProjectTreeNode {
	row := ProjectTreeNode{
		Name: node.Name,
		Path: node.Path,
		Kind: node.Kind,

		Language: node.Language,
		Layer:    node.Layer,
		Role:     node.Role,

		ChangeKind: node.ChangeKind,

		ChangedFilesCount:   node.ChangedFilesCount,
		FindingCount:        node.FindingCount,
		RuntimeFindingCount: node.RuntimeFindingCount,
		RiskPoints:          node.RiskPoints,
	}

	if len(node.children) == 0 {
		return row
	}

	keys := make([]string, 0, len(node.children))
	for key := range node.children {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		left := node.children[keys[i]]
		right := node.children[keys[j]]

		if left.Kind != right.Kind {
			return left.Kind == "dir"
		}

		return left.Name < right.Name
	})

	row.Children = make([]ProjectTreeNode, 0, len(keys))
	for _, key := range keys {
		row.Children = append(row.Children, freezeTreeNode(node.children[key]))
	}

	return row
}

func cleanTreePath(value string) string {
	return path.Clean(strings.Trim(value, "/"))
}
