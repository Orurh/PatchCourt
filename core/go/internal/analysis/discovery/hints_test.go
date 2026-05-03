package discovery

import (
	"strings"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestAnalyzeHints_DetectsBidirectionalLayerDependency(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/domain/foo.h", "src/session/foo.h", "domain", "session"),
			resolvedLayerDep("src/session/bar.cc", "src/domain/bar.h", "session", "domain"),
		},
	}

	findings := AnalyzeHints(project)

	finding := findFinding(findings, "discovery.bidirectional.domain.session")
	if finding == nil {
		t.Fatalf("expected canonical bidirectional finding, got %#v", findings)
	}

	if findFinding(findings, "discovery.bidirectional.session.domain") != nil {
		t.Fatalf("did not expect non-canonical bidirectional finding, got %#v", findings)
	}
}

func TestAnalyzeHints_DetectsDomainDependencyOnOuterLayer(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/domain/session_status.h", "src/session/session_errors.h", "domain", "session"),
		},
	}

	findings := AnalyzeHints(project)

	finding := findFinding(findings, "discovery.domain.depends_on.session")
	if finding == nil {
		t.Fatalf("expected domain dependency finding, got %#v", findings)
	}

	if finding.Kind != model.FindingKindDiscoveryHint {
		t.Fatalf("expected discovery hint kind, got %q", finding.Kind)
	}

	if finding.Severity != model.SeverityMedium {
		t.Fatalf("expected medium severity, got %q", finding.Severity)
	}
}

func TestAnalyzeHints_IgnoresDomainDependencyOnShared(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/domain/foo.h", "src/shared/types.h", "domain", "shared"),
		},
	}

	findings := AnalyzeHints(project)

	if findFinding(findings, "discovery.domain.depends_on.shared") != nil {
		t.Fatalf("did not expect domain -> shared finding, got %#v", findings)
	}
}

func TestAnalyzeHints_DetectsControllersDependingOnServer(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/controllers/device_orchestrator.cc", "src/server/mappers/foo.h", "controllers", "server"),
		},
	}

	findings := AnalyzeHints(project)

	if findFinding(findings, "discovery.controllers.depends_on.server") == nil {
		t.Fatalf("expected controllers -> server finding, got %#v", findings)
	}
}

func TestAnalyzeHints_DetectsSharedDependingOnDomain(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/utils/json_serializer.cc", "src/domain/models/status.h", "shared", "domain"),
		},
	}

	findings := AnalyzeHints(project)

	if findFinding(findings, "discovery.shared.depends_on.domain") == nil {
		t.Fatalf("expected shared -> domain finding, got %#v", findings)
	}
}

func TestAnalyzeHints_IgnoresExternalUnresolvedAndSameLayerDependencies(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			{
				FromLayer: "domain",
				ToLayer:   "session",
				Resolved:  false,
			},
			{
				FromLayer: "domain",
				ToLayer:   "session",
				Resolved:  true,
				External:  true,
			},
			resolvedLayerDep("src/domain/a.h", "src/domain/b.h", "domain", "domain"),
		},
	}

	findings := AnalyzeHints(project)

	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func resolvedLayerDep(fromFile string, toFile string, fromLayer string, toLayer string) model.DependencyEdge {
	return model.DependencyEdge{
		FromFile:  fromFile,
		ToFile:    toFile,
		Target:    toFile,
		Kind:      model.DependencyKindInclude,
		Resolved:  true,
		FromLayer: fromLayer,
		ToLayer:   toLayer,
	}
}

func findFinding(findings []model.Finding, id string) *model.Finding {
	for i := range findings {
		if findings[i].ID == id {
			return &findings[i]
		}
	}

	return nil
}

func TestAnalyzeHints_IgnoresDependenciesFromTestGeneratedAndExternalFiles(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{Path: "tests/controller_test.cc", Role: model.FileRoleTest},
			{Path: "generated/foo.pb.cc", Role: model.FileRoleGenerated},
			{Path: "third_party/lib/lib.cc", Role: model.FileRoleExternal},
			{Path: "src/domain/status.h", Role: model.FileRoleProduction},
			{Path: "src/session/session_errors.h", Role: model.FileRoleProduction},
		},
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("tests/controller_test.cc", "src/session/session_errors.h", "domain", "session"),
			resolvedLayerDep("generated/foo.pb.cc", "src/session/session_errors.h", "domain", "session"),
			resolvedLayerDep("third_party/lib/lib.cc", "src/session/session_errors.h", "domain", "session"),
		},
	}

	findings := AnalyzeHints(project)

	if len(findings) != 0 {
		t.Fatalf("expected ignored from-files to produce no findings, got %#v", findings)
	}
}

func TestAnalyzeHints_BidirectionalFindingIDIsCanonicalRegardlessOfEdgeOrder(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/session/bar.cc", "src/domain/bar.h", "session", "domain"),
			resolvedLayerDep("src/domain/foo.h", "src/session/foo.h", "domain", "session"),
		},
	}

	findings := AnalyzeHints(project)

	requireFindingID(t, findings, "discovery.bidirectional.domain.session")
	if findFinding(findings, "discovery.bidirectional.session.domain") != nil {
		t.Fatalf("did not expect non-canonical bidirectional finding, got %#v", findings)
	}
}

func requireFindingID(t *testing.T, findings []model.Finding, id string) {
	t.Helper()

	if findFinding(findings, id) == nil {
		t.Fatalf("expected finding %q, got %#v", id, findings)
	}
}

func TestAnalyzeHints_AttachesStructuredEdgeEvidence(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep("src/domain/session_status.h", "src/session/session_errors.h", "domain", "session"),
		},
	}

	findings := AnalyzeHints(project)
	finding := findFinding(findings, "discovery.domain.depends_on.session")
	if finding == nil {
		t.Fatalf("expected domain dependency finding, got %#v", findings)
	}

	if len(finding.Evidence) != 1 {
		t.Fatalf("expected evidence, got %#v", finding.Evidence)
	}

	evidence := finding.Evidence[0]
	if evidence.FromLayer != "domain" || evidence.ToLayer != "session" {
		t.Fatalf("expected structured edge domain -> session, got %q -> %q", evidence.FromLayer, evidence.ToLayer)
	}
}

func TestAnalyzeHints_DowngradesBidirectionalHintWhenOneSideIsCompositionRoot(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep(
				"src/application/bootstrapper.cc",
				"src/cameras/camera_adapter_factory.h",
				"application",
				"cameras",
			),
			resolvedLayerDep(
				"src/application/bootstrapper.cc",
				"src/cameras/sony_camera_manager_impl/sony_camera_manager.h",
				"application",
				"cameras",
			),
			resolvedLayerDep(
				"src/cameras/sony_camera_manager_impl/sony_camera_manager.cc",
				"src/application/constants.h",
				"cameras",
				"application",
			),
		},
	}

	findings := AnalyzeHints(project)

	finding := findFinding(findings, "discovery.bidirectional.application.cameras")
	if finding == nil {
		t.Fatalf("expected bidirectional finding, got %#v", findings)
	}

	if finding.Severity != model.SeverityLow {
		t.Fatalf("expected low severity for composition-root bidirectional hint, got %q", finding.Severity)
	}

	if finding.Title != "Bidirectional layer dependency with composition-root side" {
		t.Fatalf("unexpected title: %q", finding.Title)
	}

	if !strings.Contains(finding.Risk, "composition-root") {
		t.Fatalf("expected risk to explain composition-root side, got %q", finding.Risk)
	}

	if len(finding.Evidence) == 0 {
		t.Fatalf("expected evidence")
	}

	if finding.Evidence[0].FromLayer != "cameras" || finding.Evidence[0].ToLayer != "application" {
		t.Fatalf("expected suspicious reverse side first, got %q -> %q", finding.Evidence[0].FromLayer, finding.Evidence[0].ToLayer)
	}
}

func TestAnalyzeHints_DetectsApplicationConstantsAsSharedCandidate(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			resolvedLayerDep(
				"src/cameras/sony_camera_manager_impl/sony_camera_manager.cc",
				"src/application/constants.h",
				"cameras",
				"application",
			),
			resolvedLayerDep(
				"src/cameras/gopro_camera_manager_impl/gopro_camera_manager.cc",
				"src/application/constants.h",
				"cameras",
				"application",
			),
		},
	}

	findings := AnalyzeHints(project)

	finding := findFinding(findings, "discovery.shared_candidate.application.constants")
	if finding == nil {
		t.Fatalf("expected shared candidate finding, got %#v", findings)
	}

	if finding.Severity != model.SeverityLow {
		t.Fatalf("expected low severity, got %q", finding.Severity)
	}

	if finding.Title != "Application file looks like shared dependency candidate" {
		t.Fatalf("unexpected title: %q", finding.Title)
	}

	if len(finding.Evidence) != 2 {
		t.Fatalf("expected 2 evidence items, got %d", len(finding.Evidence))
	}

	if finding.Evidence[0].FromLayer != "cameras" || finding.Evidence[0].ToLayer != "application" {
		t.Fatalf("expected cameras -> application evidence, got %q -> %q", finding.Evidence[0].FromLayer, finding.Evidence[0].ToLayer)
	}
}
