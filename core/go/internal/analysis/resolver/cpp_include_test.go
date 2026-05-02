package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestCPPIncludeResolver_ResolveExactProjectPath(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/domain/interfaces/i_camera_adapter.h"},
	}

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), nil)

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

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), nil)

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

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), nil)

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

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), nil)

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

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), nil)

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

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), testIncludePaths("src"))

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

func TestCPPIncludeResolver_MarksPhysicalFileOutsideIndexAsExternal(t *testing.T) {
	root := t.TempDir()

	writeResolverTestFile(t, root, "libs/logx/include/LogX.h")

	resolver := NewCPPIncludeResolver(
		root,
		NewFileIndex(nil),
		testIncludePaths("libs/logx/include"),
	)

	got := resolver.Resolve(
		"src/server/api_router.cc",
		"LogX.h",
	)

	if !got.External {
		t.Fatalf("expected include to be marked external")
	}

	if got.Resolved {
		t.Fatalf("external file outside project index must not be resolved to project file")
	}

	if got.Source != model.ResolutionSourceConfig {
		t.Fatalf("expected config source, got %q", got.Source)
	}

	if got.Confidence != model.ResolutionConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", got.Confidence)
	}
}

func TestCPPIncludeResolver_PreservesIncludePathSource(t *testing.T) {
	files := []model.FileModel{
		{Path: "src/application/constants.h"},
	}

	resolver := NewCPPIncludeResolver("", NewFileIndex(files), []IncludePath{
		{
			Path:       "src",
			Source:     model.ResolutionSourceCompileCommands,
			Confidence: model.ResolutionConfidenceHigh,
		},
	})

	got := resolver.Resolve("src/main.cc", "application/constants.h")
	if got.Source != model.ResolutionSourceCompileCommands {
		t.Fatalf("expected compile_commands source, got %q", got.Source)
	}
}

func testIncludePaths(paths ...string) []IncludePath {
	result := make([]IncludePath, 0, len(paths))
	for _, path := range paths {
		result = append(result, IncludePath{
			Path:       path,
			Source:     model.ResolutionSourceConfig,
			Confidence: model.ResolutionConfidenceHigh,
		})
	}

	return result
}

func writeResolverTestFile(t *testing.T, root string, relPath string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("create test dir: %v", err)
	}

	if err := os.WriteFile(absPath, []byte("#pragma once\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func TestCPPIncludeResolver_MarksSystemIncludePathFileAsExternal(t *testing.T) {
	root := t.TempDir()

	writeResolverTestFile(t, root, "sysroot/include/vendor.h")

	resolver := NewCPPIncludeResolver(
		root,
		NewFileIndex(nil),
		[]IncludePath{
			{
				Path:       "sysroot/include",
				Source:     model.ResolutionSourceConfig,
				Confidence: model.ResolutionConfidenceMedium,
				System:     true,
			},
		},
	)

	got := resolver.Resolve(
		"src/server/api_router.cc",
		"vendor.h",
	)

	if !got.External {
		t.Fatalf("expected system include path file to be marked external")
	}

	if got.Resolved {
		t.Fatalf("external system file outside project index must not be resolved")
	}

	if got.Source != model.ResolutionSourceConfig {
		t.Fatalf("expected config source, got %q", got.Source)
	}

	if got.Confidence != model.ResolutionConfidenceMedium {
		t.Fatalf("expected medium confidence, got %q", got.Confidence)
	}
}
