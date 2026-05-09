package runtime

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/orurh/patchcourt/internal/model"
)

func Analyze(root string, project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	builders := runtimeFindingBuilders()

	for _, file := range project.Files {
		if !isReviewableCPPFile(file) {
			continue
		}

		analyzeFile(root, file, builders)
	}

	findings := make([]model.Finding, 0, len(builders))
	for _, builder := range builders {
		if len(builder.finding.Evidence) == 0 {
			continue
		}

		sortEvidence(builder.finding.Evidence)
		findings = append(findings, builder.finding)
	}

	sort.Slice(findings, func(i int, j int) bool {
		return findings[i].ID < findings[j].ID
	})

	return findings
}

func isReviewableCPPFile(file model.FileModel) bool {
	if file.Language != model.LanguageCPP {
		return false
	}

	if file.Role == model.FileRoleGenerated || file.Role == model.FileRoleExternal {
		return false
	}

	return file.Kind == model.FileKindSource || file.Kind == model.FileKindHeader
}

func analyzeFile(root string, file model.FileModel, builders map[string]*findingBuilder) {
	path := filepath.Join(root, filepath.FromSlash(file.Path))

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := splitLines(string(data))
	rawPointers := make(map[string]rawPointerCandidate)

	for i, original := range lines {
		lineNumber := i + 1
		line := stripLineComment(original)

		expireRawPointerCandidates(rawPointers, lineNumber)
		removeShadowedRawPointers(rawPointers, line)

		for _, name := range rawPointerNamesFromLine(line) {
			rawPointers[name] = rawPointerCandidate{
				name: name,
				line: lineNumber,
			}
		}

		captures := lambdaCaptures(line)
		if len(captures) > 0 {
			context := classifyRuntimeContext(buildLineWindow(lines, i))
			analyzeCaptures(file, lineNumber, original, lines, i, captures, context, rawPointers, builders)
		}

		analyzeShutdownSleep(file, lineNumber, original, lines, i, builders)
	}
}
