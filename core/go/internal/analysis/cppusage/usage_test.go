package cppusage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestAnalyze_MarksIncludeAsUsedWhenTargetSymbolIsReferenced(t *testing.T) {
	root := t.TempDir()

	writeUsageTestFile(t, root, "src/domain/camera_status.h", `#pragma once

struct CameraStatus {};
`)

	writeUsageTestFile(t, root, "src/server/api_router.cc", `#include "src/domain/camera_status.h"

CameraStatus MakeStatus() {
	return CameraStatus{};
}
`)

	project := &model.ProjectModel{
		Root: root,
		Symbols: []model.SymbolModel{
			{
				File:     "src/domain/camera_status.h",
				Name:     "CameraStatus",
				Kind:     model.SymbolKindStruct,
				Exported: true,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/domain/camera_status.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnknown,
			},
		},
	}

	Analyze(project)

	if project.Dependencies[0].Usage != model.DependencyUsageUsed {
		t.Fatalf("expected used include, got %q", project.Dependencies[0].Usage)
	}
}

func TestAnalyze_MarksIncludeAsUnusedWhenTargetSymbolIsNotReferenced(t *testing.T) {
	root := t.TempDir()

	writeUsageTestFile(t, root, "src/domain/camera_status.h", `#pragma once

struct CameraStatus {};
`)

	writeUsageTestFile(t, root, "src/server/api_router.cc", `#include "src/domain/camera_status.h"

int Health() {
	return 200;
}
`)

	project := &model.ProjectModel{
		Root: root,
		Symbols: []model.SymbolModel{
			{
				File:     "src/domain/camera_status.h",
				Name:     "CameraStatus",
				Kind:     model.SymbolKindStruct,
				Exported: true,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/domain/camera_status.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnknown,
			},
		},
	}

	Analyze(project)

	if project.Dependencies[0].Usage != model.DependencyUsageUnused {
		t.Fatalf("expected unused include, got %q", project.Dependencies[0].Usage)
	}
}

func TestAnalyze_IgnoresSymbolMentionedOnlyInCommentOrString(t *testing.T) {
	root := t.TempDir()

	writeUsageTestFile(t, root, "src/domain/camera_status.h", `#pragma once

struct CameraStatus {};
`)

	writeUsageTestFile(t, root, "src/server/api_router.cc", `#include "src/domain/camera_status.h"

// CameraStatus appears only in a comment.
const char* name = "CameraStatus";
`)

	project := &model.ProjectModel{
		Root: root,
		Symbols: []model.SymbolModel{
			{
				File:     "src/domain/camera_status.h",
				Name:     "CameraStatus",
				Kind:     model.SymbolKindStruct,
				Exported: true,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/domain/camera_status.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnknown,
			},
		},
	}

	Analyze(project)

	if project.Dependencies[0].Usage != model.DependencyUsageUnused {
		t.Fatalf("expected unused include, got %q", project.Dependencies[0].Usage)
	}
}

func TestAnalyze_MarksIncludeAsMaybeWhenTargetHasNoSymbols(t *testing.T) {
	root := t.TempDir()

	writeUsageTestFile(t, root, "src/domain/constants.h", `#pragma once

#define CAMERA_COUNT 9
`)

	writeUsageTestFile(t, root, "src/server/api_router.cc", `#include "src/domain/constants.h"

int Health() {
	return 200;
}
`)

	project := &model.ProjectModel{
		Root: root,
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/domain/constants.h",
				Kind:     model.DependencyKindInclude,
				Resolved: true,
				Usage:    model.DependencyUsageUnknown,
			},
		},
	}

	Analyze(project)

	if project.Dependencies[0].Usage != model.DependencyUsageMaybe {
		t.Fatalf("expected maybe include, got %q", project.Dependencies[0].Usage)
	}
}

func TestAnalyze_IgnoresExternalAndUnresolvedIncludes(t *testing.T) {
	project := &model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				Target:   "vector",
				Kind:     model.DependencyKindInclude,
				External: true,
				Usage:    model.DependencyUsageUnknown,
			},
			{
				FromFile: "src/server/api_router.cc",
				Target:   "missing.h",
				Kind:     model.DependencyKindInclude,
				Resolved: false,
				Usage:    model.DependencyUsageUnknown,
			},
		},
	}

	Analyze(project)

	for _, dep := range project.Dependencies {
		if dep.Usage != model.DependencyUsageUnknown {
			t.Fatalf("expected ignored include to remain unknown, got %q", dep.Usage)
		}
	}
}

func writeUsageTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
