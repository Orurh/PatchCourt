package rules

import (
	"testing"

	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
)

func TestApplyArchitectureRules_DetectsForbiddenLayerDependency(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/server/api_router.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleProduction,
			},
			{
				Path:     "src/cameras/sony/sony_camera_manager.h",
				Language: model.LanguageCPP,
				Kind:     model.FileKindHeader,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/cameras/sony/sony_camera_manager.h",
				Target:   "src/cameras/sony/sony_camera_manager.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
			},
		},
	}

	cfg := testArchitectureConfig()

	ApplyArchitectureRules(project, cfg)

	if len(project.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(project.Findings))
	}

	finding := project.Findings[0]

	if finding.ID != "architecture.api.cameras" {
		t.Fatalf("unexpected finding id: %q", finding.ID)
	}

	if finding.Severity != model.SeverityHigh {
		t.Fatalf("unexpected severity: %q", finding.Severity)
	}

	if finding.Confidence != model.ConfidenceHigh {
		t.Fatalf("unexpected confidence: %q", finding.Confidence)
	}

	if project.Dependencies[0].FromLayer != "api" {
		t.Fatalf("expected from layer api, got %q", project.Dependencies[0].FromLayer)
	}

	if project.Dependencies[0].ToLayer != "cameras" {
		t.Fatalf("expected to layer cameras, got %q", project.Dependencies[0].ToLayer)
	}
}

func TestApplyArchitectureRules_AllowsConfiguredDependency(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/server/api_router.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleProduction,
			},
			{
				Path:     "src/controllers/device_orchestrator.h",
				Language: model.LanguageCPP,
				Kind:     model.FileKindHeader,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/controllers/device_orchestrator.h",
				Target:   "src/controllers/device_orchestrator.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
			},
		},
	}

	cfg := testArchitectureConfig()

	ApplyArchitectureRules(project, cfg)

	if len(project.Findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(project.Findings))
	}

	if project.Dependencies[0].FromLayer != "api" {
		t.Fatalf("expected from layer api, got %q", project.Dependencies[0].FromLayer)
	}

	if project.Dependencies[0].ToLayer != "controllers" {
		t.Fatalf("expected to layer controllers, got %q", project.Dependencies[0].ToLayer)
	}
}

func TestApplyArchitectureRules_IgnoresTestFiles(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "tests/api_router_test.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindTest,
				Role:     model.FileRoleTest,
			},
			{
				Path:     "src/cameras/sony/sony_camera_manager.h",
				Language: model.LanguageCPP,
				Kind:     model.FileKindHeader,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "tests/api_router_test.cc",
				ToFile:   "src/cameras/sony/sony_camera_manager.h",
				Target:   "src/cameras/sony/sony_camera_manager.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
			},
		},
	}

	cfg := testArchitectureConfig()

	ApplyArchitectureRules(project, cfg)

	if len(project.Findings) != 0 {
		t.Fatalf("expected test dependency to be ignored, got %d findings", len(project.Findings))
	}
}

func TestApplyArchitectureRules_IgnoresExternalDependencies(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/server/api_router.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				Target:   "boost/asio.hpp",
				Kind:     model.DependencyKindInclude,
				External: true,
				Resolved: false,
			},
		},
	}

	cfg := testArchitectureConfig()

	ApplyArchitectureRules(project, cfg)

	if len(project.Findings) != 0 {
		t.Fatalf("expected external dependency to be ignored, got %d findings", len(project.Findings))
	}
}

func TestApplyArchitectureRules_IgnoresUnresolvedDependencies(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/server/api_router.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				Target:   "missing.h",
				Kind:     model.DependencyKindInclude,
				Resolved: false,
			},
		},
	}

	cfg := testArchitectureConfig()

	ApplyArchitectureRules(project, cfg)

	if len(project.Findings) != 0 {
		t.Fatalf("expected unresolved dependency to be ignored, got %d findings", len(project.Findings))
	}
}

func testArchitectureConfig() *config.Config {
	return &config.Config{
		Layers: map[string]config.LayerConfig{
			"api": {
				Paths: []string{
					"src/server/**",
				},
				MayDependOn: []string{
					"controllers",
					"domain",
				},
			},
			"controllers": {
				Paths: []string{
					"src/controllers/**",
				},
				MayDependOn: []string{
					"domain",
				},
			},
			"domain": {
				Paths:       []string{"src/domain/**"},
				MayDependOn: []string{},
			},
			"cameras": {
				Paths: []string{
					"src/cameras/**",
				},
				MayDependOn: []string{
					"domain",
				},
			},
		},
	}
}

func TestApplyArchitectureRules_PrefersMoreSpecificLayerPattern(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/application/constants.h",
				Language: model.LanguageCPP,
				Kind:     model.FileKindHeader,
				Role:     model.FileRoleProduction,
			},
		},
	}

	cfg := &config.Config{
		Layers: map[string]config.LayerConfig{
			"application": {
				Paths: []string{
					"src/application/**",
				},
			},
			"shared": {
				Paths: []string{
					"src/application/constants.h",
				},
			},
		},
	}

	ApplyArchitectureRules(project, cfg)

	if project.Files[0].Layer != "shared" {
		t.Fatalf("expected shared layer, got %q", project.Files[0].Layer)
	}
}

func TestApplyArchitectureRules_RespectsLayerExcludePaths(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/application/constants.h",
				Language: model.LanguageCPP,
				Kind:     model.FileKindHeader,
				Role:     model.FileRoleProduction,
			},
		},
	}

	cfg := &config.Config{
		Layers: map[string]config.LayerConfig{
			"application": {
				Paths: []string{
					"src/application/**",
				},
				ExcludePaths: []string{
					"src/application/constants.h",
				},
			},
			"shared": {
				Paths: []string{
					"src/application/constants.h",
				},
			},
		},
	}

	ApplyArchitectureRules(project, cfg)

	if project.Files[0].Layer != "shared" {
		t.Fatalf("expected shared layer, got %q", project.Files[0].Layer)
	}
}

func TestApplyArchitectureRules_AttachesStructuredEdgeEvidence(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{Path: "src/server/api_router.cc", Language: model.LanguageCPP, Kind: model.FileKindSource, Role: model.FileRoleProduction},
			{Path: "src/cameras/sony.h", Language: model.LanguageCPP, Kind: model.FileKindHeader, Role: model.FileRoleProduction},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/cameras/sony.h",
				Target:   "src/cameras/sony.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
			},
		},
	}

	ApplyArchitectureRules(project, testArchitectureConfig())

	if len(project.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(project.Findings))
	}

	evidence := project.Findings[0].Evidence[0]
	if evidence.FromLayer != "api" || evidence.ToLayer != "cameras" {
		t.Fatalf("expected structured edge api -> cameras, got %q -> %q", evidence.FromLayer, evidence.ToLayer)
	}

	if evidence.FromFile != "src/server/api_router.cc" || evidence.ToFile != "src/cameras/sony.h" {
		t.Fatalf("unexpected evidence files: %q -> %q", evidence.FromFile, evidence.ToFile)
	}
}
