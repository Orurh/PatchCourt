package discovery

import (
	analysisproject "github.com/orurh/patchcourt/internal/analyzer/project"
	"github.com/orurh/patchcourt/internal/model"
)

func ignoredFromFiles(project *model.ProjectModel) map[string]bool {
	if project == nil {
		return map[string]bool{}
	}

	return analysisproject.IgnoredAnalysisFileSet(project.Files)
}
