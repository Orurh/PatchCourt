package resolver

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestCPPIncludeResolver_ResolveExactProjectPath(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/domain/interfaces/i_camera_adapter.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), nil)

	got := resolver.Resolve(
		"src/controllers/device_orchestrator.h",
		"src/domain/interfaces/i_camera_adapter.h",
	)

	if !got.Resolved {
		t.Fatalf("expected include to be resolved")
	}

	if got.ToFile != "src/domain/interfaces/i_camera_adapter.h" {
		t.Fatalf("unexpected resolved file: %q", got.ToFile)
	}

	if got.Source != model.ResolutionSourceHeuristic {
		t.Fatalf("unexpected source: %q", got.Source)
	}

	if got.Confidence != model.ResolutionConfidenceMedium {
		t.Fatalf("unexpected confidence: %q", got.Confidence)
	}

	if got.Ambiguous {
		t.Fatalf("expected non-ambiguous resolution")
	}
}

func TestCPPIncludeResolver_ResolveRelativeToCurrentFile(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/controllers/detail/helper.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), nil)

	got := resolver.Resolve(
		"src/controllers/device_orchestrator.h",
		"detail/helper.h",
	)

	if !got.Resolved {
		t.Fatalf("expected include to be resolved")
	}

	if got.ToFile != "src/controllers/detail/helper.h" {
		t.Fatalf("unexpected resolved file: %q", got.ToFile)
	}

	if got.Confidence != model.ResolutionConfidenceMedium {
		t.Fatalf("unexpected confidence: %q", got.Confidence)
	}
}

func TestCPPIncludeResolver_ResolveUniqueBasenameFallback(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/domain/interfaces/i_camera_adapter.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), nil)

	got := resolver.Resolve(
		"src/controllers/device_orchestrator.h",
		"i_camera_adapter.h",
	)

	if !got.Resolved {
		t.Fatalf("expected include to be resolved")
	}

	if got.ToFile != "src/domain/interfaces/i_camera_adapter.h" {
		t.Fatalf("unexpected resolved file: %q", got.ToFile)
	}

	if got.Confidence != model.ResolutionConfidenceLow {
		t.Fatalf("basename fallback must be low confidence, got %q", got.Confidence)
	}

	if len(got.Candidates) != 1 {
		t.Fatalf("expected exactly one candidate, got %d", len(got.Candidates))
	}
}

func TestCPPIncludeResolver_AmbiguousBasenameFallback(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/domain/config.h"},
		{Path: "src/infrastructure/config.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), nil)

	got := resolver.Resolve(
		"src/controllers/device_orchestrator.h",
		"config.h",
	)

	if got.Resolved {
		t.Fatalf("expected ambiguous include to stay unresolved")
	}

	if !got.Ambiguous {
		t.Fatalf("expected ambiguous resolution")
	}

	if got.Confidence != model.ResolutionConfidenceLow {
		t.Fatalf("ambiguous basename fallback must be low confidence, got %q", got.Confidence)
	}

	if len(got.Candidates) != 2 {
		t.Fatalf("expected two candidates, got %d", len(got.Candidates))
	}
}

func TestCPPIncludeResolver_UnresolvedInclude(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/domain/interfaces/i_camera_adapter.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), nil)

	got := resolver.Resolve(
		"src/controllers/device_orchestrator.h",
		"missing.h",
	)

	if got.Resolved {
		t.Fatalf("expected include to stay unresolved")
	}

	if got.Source != model.ResolutionSourceNone {
		t.Fatalf("unexpected source: %q", got.Source)
	}

	if got.Confidence != model.ResolutionConfidenceLow {
		t.Fatalf("unexpected confidence: %q", got.Confidence)
	}

	if got.Ambiguous {
		t.Fatalf("missing include must not be ambiguous")
	}
}

func TestCPPIncludeResolver_ResolveUsingConfiguredIncludePath(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/application/constants.h"},
		{Path: "src/session/constants.h"},
	}

	resolver := NewCPPIncludeResolver(NewFileIndex(files), []string{"src"})

	got := resolver.Resolve(
		"src/cameras/sony_camera_manager_impl/sony_camera_manager.cc",
		"application/constants.h",
	)

	if !got.Resolved {
		t.Fatalf("expected include to be resolved")
	}

	if got.ToFile != "src/application/constants.h" {
		t.Fatalf("unexpected resolved file: %q", got.ToFile)
	}

	if got.Source != model.ResolutionSourceConfig {
		t.Fatalf("expected config source, got %q", got.Source)
	}

	if got.Confidence != model.ResolutionConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", got.Confidence)
	}

	if got.Ambiguous {
		t.Fatalf("config include path should avoid ambiguity")
	}
}
