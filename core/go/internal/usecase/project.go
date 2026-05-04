package usecase

import "github.com/orurh/patchcourt/internal/usecase/projectbuild"

type ProjectBuildRequest = projectbuild.Request
type ProjectBuildResult = projectbuild.Result
type ProjectBuilder = projectbuild.Builder

func NewProjectBuilder(analysis AnalysisService) ProjectBuilder {
	return projectbuild.New(analysis)
}
