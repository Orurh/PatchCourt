package rules

import (
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type Rule interface {
	Apply(project *model.ProjectModel, cfg *config.Config) []model.Finding
}

func DefaultRules() []Rule {
	return []Rule{
		ArchitectureRule{},
	}
}

func Apply(project *model.ProjectModel, cfg *config.Config, ruleSet []Rule) {
	for _, rule := range ruleSet {
		findings := rule.Apply(project, cfg)
		project.Findings = append(project.Findings, findings...)
	}
}

func ApplyDefault(project *model.ProjectModel, cfg *config.Config) {
	Apply(project, cfg, DefaultRules())
}
