package discovery

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestAnalyzeHints_DetectsUnusedIncludes(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			{
				FromFile:             "src/server/api_router.cc",
				ToFile:               "src/domain/unused_type.h",
				Target:               "src/domain/unused_type.h",
				Kind:                 model.DependencyKindInclude,
				Resolved:             true,
				Usage:                model.DependencyUsageUnused,
				ResolutionSource:     model.ResolutionSourceHeuristic,
				ResolutionConfidence: model.ResolutionConfidenceMedium,
			},
		},
	}

	findings := AnalyzeHints(project)

	finding := findFinding(findings, "discovery.cpp.unused_includes")
	if finding == nil {
		t.Fatalf("expected unused include finding, got %#v", findings)
	}

	if finding.Kind != model.FindingKindDiscoveryHint {
		t.Fatalf("expected discovery hint kind, got %q", finding.Kind)
	}

	if finding.Severity != model.SeverityLow {
		t.Fatalf("expected low severity, got %q", finding.Severity)
	}

	if len(finding.Evidence) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(finding.Evidence))
	}
}

func TestAnalyzeHints_IgnoresUsedMaybeUnknownExternalAndUnresolvedIncludes(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			{
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUsed,
			},
			{
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageMaybe,
			},
			{
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnknown,
			},
			{
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				External: true,
				Usage:    model.DependencyUsageUnused,
			},
			{
				Kind:     model.DependencyKindInclude,
				Resolved: false,
				Usage:    model.DependencyUsageUnused,
			},
		},
	}

	findings := AnalyzeHints(project)

	if findFinding(findings, "discovery.cpp.unused_includes") != nil {
		t.Fatalf("did not expect unused include finding, got %#v", findings)
	}
}

func TestAnalyzeHints_IgnoresUnusedIncludesFromTestGeneratedAndExternalFiles(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{Path: "tests/controller_test.cc", Role: model.FileRoleTest},
			{Path: "generated/foo.pb.cc", Role: model.FileRoleGenerated},
			{Path: "third_party/lib/lib.cc", Role: model.FileRoleExternal},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "tests/controller_test.cc",
				ToFile:   "src/domain/unused.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnused,
			},
			{
				FromFile: "generated/foo.pb.cc",
				ToFile:   "src/domain/unused.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnused,
			},
			{
				FromFile: "third_party/lib/lib.cc",
				ToFile:   "src/domain/unused.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnused,
			},
		},
	}

	findings := AnalyzeHints(project)

	if findFinding(findings, "discovery.cpp.unused_includes") != nil {
		t.Fatalf("did not expect unused include finding from ignored files, got %#v", findings)
	}
}
