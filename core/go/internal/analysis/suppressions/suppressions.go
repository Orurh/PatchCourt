package suppressions

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type Scope string

const (
	ScopeFinding Scope = "finding"
	ScopeFile    Scope = "file"
)

type Suppression struct {
	ID     string `json:"id"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Scope  Scope  `json:"scope"`
	Reason string `json:"reason,omitempty"`
}

var directiveRE = regexp.MustCompile(`patchcourt:ignore(-file)?(?:\s+([A-Za-z0-9_.:*:-]+))?(?:\s+(.*))?$`)

func Collect(root string, files []model.FileModel) ([]Suppression, error) {
	if root == "" {
		root = "."
	}

	suppressions := make([]Suppression, 0)

	for _, file := range files {
		if file.Role == model.FileRoleGenerated || file.Role == model.FileRoleExternal {
			continue
		}

		if file.Language != model.LanguageCPP && file.Language != model.LanguageGo {
			continue
		}

		fileSuppressions, err := collectFileSuppressions(root, file.Path)
		if err != nil {
			return nil, err
		}

		suppressions = append(suppressions, fileSuppressions...)
	}

	return suppressions, nil
}

func Apply(project *model.ProjectModel, suppressions []Suppression) int {
	if project == nil || len(project.Findings) == 0 || len(suppressions) == 0 {
		return 0
	}

	index := buildIndex(suppressions)

	filtered := make([]model.Finding, 0, len(project.Findings))
	suppressedFindings := 0

	for _, finding := range project.Findings {
		nextFinding, keep := applyToFinding(finding, index)
		if !keep {
			suppressedFindings++
			continue
		}

		filtered = append(filtered, nextFinding)
	}

	project.Findings = filtered
	return suppressedFindings
}

func collectFileSuppressions(root string, relPath string) ([]Suppression, error) {
	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("read suppressions from %s: %w", relPath, err)
	}
	defer file.Close()

	var result []Suppression

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		suppression, ok := parseDirective(scanner.Text())
		if !ok {
			continue
		}

		suppression.File = relPath
		suppression.Line = lineNumber
		result = append(result, suppression)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan suppressions from %s: %w", relPath, err)
	}

	return result, nil
}

func parseDirective(line string) (Suppression, bool) {
	match := directiveRE.FindStringSubmatch(line)
	if len(match) != 4 {
		return Suppression{}, false
	}

	scope := ScopeFinding
	if match[1] == "-file" {
		scope = ScopeFile
	}

	id := strings.TrimSpace(match[2])
	if id == "" {
		if scope == ScopeFile {
			id = "*"
		} else {
			return Suppression{}, false
		}
	}

	return Suppression{
		ID:     id,
		Scope:  scope,
		Reason: strings.TrimSpace(match[3]),
	}, true
}

type suppressionIndex map[string][]Suppression

func buildIndex(suppressions []Suppression) suppressionIndex {
	index := make(suppressionIndex)

	for _, suppression := range suppressions {
		if suppression.File == "" || suppression.ID == "" {
			continue
		}

		index[suppression.File] = append(index[suppression.File], suppression)
	}

	return index
}

func applyToFinding(finding model.Finding, index suppressionIndex) (model.Finding, bool) {
	if finding.ID == "" || len(finding.Evidence) == 0 {
		return finding, true
	}

	remainingEvidence := make([]model.Evidence, 0, len(finding.Evidence))

	for _, evidence := range finding.Evidence {
		if evidence.File == "" {
			remainingEvidence = append(remainingEvidence, evidence)
			continue
		}

		if isSuppressed(finding.ID, evidence.File, index) {
			continue
		}

		remainingEvidence = append(remainingEvidence, evidence)
	}

	if len(remainingEvidence) == 0 {
		return model.Finding{}, false
	}

	finding.Evidence = remainingEvidence
	return finding, true
}

func isSuppressed(findingID string, file string, index suppressionIndex) bool {
	for _, suppression := range index[file] {
		if suppression.ID == "*" || suppression.ID == findingID {
			return true
		}
	}

	return false
}
