package discovery

import "github.com/orurh/patchcourt/internal/model"

func ignoredFromFiles(project *model.ProjectModel) map[string]bool {
	ignored := make(map[string]bool)

	if project == nil {
		return ignored
	}

	for _, file := range project.Files {
		switch file.Role {
		case model.FileRoleTest, model.FileRoleGenerated, model.FileRoleExternal:
			ignored[file.Path] = true
		}
	}

	return ignored
}
