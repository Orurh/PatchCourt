package rules

import (
	"testing"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

type fakeRule struct {
	finding model.Finding
}

func (r fakeRule) Apply(project *model.ProjectModel, cfg *config.Config) []model.Finding {
	return []model.Finding{r.finding}
}

func TestApply_AppendsFindingsFromRuleSet(t *testing.T) {
	project := &model.ProjectModel{}

	Apply(project, &config.Config{}, []Rule{
		fakeRule{
			finding: model.Finding{
				ID:         "test.rule",
				Severity:   model.SeverityLow,
				Title:      "Test finding",
				Confidence: model.ConfidenceHigh,
			},
		},
	})

	if len(project.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(project.Findings))
	}

	if project.Findings[0].ID != "test.rule" {
		t.Fatalf("unexpected finding id: %q", project.Findings[0].ID)
	}
}

func TestDefaultRules_ContainsArchitectureRule(t *testing.T) {
	ruleSet := DefaultRules()

	if len(ruleSet) == 0 {
		t.Fatalf("expected default rules")
	}

	found := false
	for _, rule := range ruleSet {
		if _, ok := rule.(ArchitectureRule); ok {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected ArchitectureRule in default rules")
	}
}

func TestArchitectureRule_ImplementsRuleInterface(t *testing.T) {
	var _ Rule = ArchitectureRule{}
}
