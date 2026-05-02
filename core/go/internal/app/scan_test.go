package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

func TestApp_RunScan_EndToEndDetectsArchitectureViolation(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/domain/interfaces/i_camera_adapter.h", `#pragma once

namespace demo {

class ICameraAdapter {
public:
    virtual ~ICameraAdapter() = default;
    virtual bool RunPreflight() const = 0;
};

}
`)

	writeFile(t, root, "src/controllers/device_orchestrator.h", `#pragma once

#include "src/domain/interfaces/i_camera_adapter.h"

namespace demo {

class DeviceOrchestrator {
public:
    bool RunPreflight() const;
};

}
`)

	writeFile(t, root, "src/cameras/sony/sony_camera_manager.h", `#pragma once

#include "src/domain/interfaces/i_camera_adapter.h"

namespace demo {

class SonyCameraManager final : public ICameraAdapter {
public:
    bool RunPreflight() const override;
};

}
`)

	writeFile(t, root, "src/server/api_router.cc", `#include "src/controllers/device_orchestrator.h"
#include "src/cameras/sony/sony_camera_manager.h"

int main() {
    return 0;
}
`)

	writeConfig(t, root, testConfig())

	application := New(logx.Nop())

	result, err := application.RunScan(context.Background(), ScanRequest{
		Root:       root,
		ConfigPath: filepath.Join(root, ".patchcourt.yaml"),
	})
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	if result == nil || result.Project == nil {
		t.Fatalf("expected scan result with project")
	}

	if len(result.Project.Files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(result.Project.Files))
	}

	if len(result.Project.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(result.Project.Dependencies))
	}

	if len(result.Project.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Project.Findings))
	}

	finding := result.Project.Findings[0]
	if finding.ID != "architecture.api.cameras" {
		t.Fatalf("unexpected finding id: %q", finding.ID)
	}

	if finding.Severity != model.SeverityHigh {
		t.Fatalf("unexpected severity: %q", finding.Severity)
	}

	dep, found := findDependency(
		result.Project.Dependencies,
		"src/server/api_router.cc",
		"src/cameras/sony/sony_camera_manager.h",
	)
	if !found {
		t.Fatalf("expected api -> cameras dependency")
	}

	if dep.FromLayer != "api" {
		t.Fatalf("expected from layer api, got %q", dep.FromLayer)
	}

	if dep.ToLayer != "cameras" {
		t.Fatalf("expected to layer cameras, got %q", dep.ToLayer)
	}

	if dep.ResolutionSource != model.ResolutionSourceHeuristic {
		t.Fatalf("unexpected resolution source: %q", dep.ResolutionSource)
	}

	if dep.ResolutionConfidence != model.ResolutionConfidenceMedium {
		t.Fatalf("unexpected resolution confidence: %q", dep.ResolutionConfidence)
	}

	if dep.Usage != model.DependencyUsageUnknown {
		t.Fatalf("expected unknown dependency usage for C++ include, got %q", dep.Usage)
	}
}

func TestApp_RunScan_EndToEndIgnoresGeneratedAndBuildFiles(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/server/api_router.cc", `#include "src/controllers/device_orchestrator.h"
`)
	writeFile(t, root, "src/controllers/device_orchestrator.h", `#pragma once
`)
	writeFile(t, root, "generated/foo.pb.h", `#pragma once
`)
	writeFile(t, root, "build/temp.cc", `#include "missing.h"
`)

	writeConfig(t, root, testConfig())

	application := New(logx.Nop())

	result, err := application.RunScan(context.Background(), ScanRequest{
		Root:       root,
		ConfigPath: filepath.Join(root, ".patchcourt.yaml"),
	})
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	for _, file := range result.Project.Files {
		switch file.Path {
		case "generated/foo.pb.h":
			t.Fatalf("generated file must be ignored by config")
		case "build/temp.cc":
			t.Fatalf("build file must be ignored by config")
		}
	}
}

func TestApp_RunScan_UsesCompileCommandsIncludePaths(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/application/constants.h", `#pragma once
`)

	writeFile(t, root, "src/server/api_router.cc", `#include "application/constants.h"
`)

	writeFile(t, root, "build/compile_commands.json", `[
  {
    "directory": "`+filepath.ToSlash(filepath.Join(root, "build"))+`",
    "arguments": [
      "clang++",
      "-I",
      "../src",
      "-c",
      "../src/server/api_router.cc"
    ],
    "file": "`+filepath.ToSlash(filepath.Join(root, "src/server/api_router.cc"))+`"
  }
]`)

	writeFile(t, root, ".patchcourt.yaml", `
ignore:
  paths:
    - generated/**

cpp:
  compile_commands:
    auto_discover: true

layers:
  api:
    paths:
      - src/server/**
    may_depend_on:
      - application

  application:
    paths:
      - src/application/**
    may_depend_on: []
`)

	application := New(logx.Nop())

	result, err := application.RunScan(context.Background(), ScanRequest{
		Root:       root,
		ConfigPath: filepath.Join(root, ".patchcourt.yaml"),
	})
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	dep, found := findDependency(
		result.Project.Dependencies,
		"src/server/api_router.cc",
		"src/application/constants.h",
	)
	if !found {
		t.Fatalf("expected dependency resolved through compile_commands include path")
	}

	if dep.ResolutionSource != model.ResolutionSourceCompileCommands {
		t.Fatalf("expected compile_commands source, got %q", dep.ResolutionSource)
	}

	if dep.ResolutionConfidence != model.ResolutionConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", dep.ResolutionConfidence)
	}
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("create dir for %s: %v", relPath, err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", relPath, err)
	}
}

func writeConfig(t *testing.T, root string, content string) {
	t.Helper()
	writeFile(t, root, ".patchcourt.yaml", content)
}

func findDependency(deps []model.DependencyEdge, fromFile string, toFile string) (model.DependencyEdge, bool) {
	for _, dep := range deps {
		if dep.FromFile == fromFile && dep.ToFile == toFile {
			return dep, true
		}
	}

	return model.DependencyEdge{}, false
}

func testConfig() string {
	return `
ignore:
  paths:
    - build/**
    - generated/**
    - "**/*.pb.h"
    - "**/*.pb.cc"

layers:
  api:
    paths:
      - src/server/**
    may_depend_on:
      - controllers
      - domain

  controllers:
    paths:
      - src/controllers/**
    may_depend_on:
      - domain

  domain:
    paths:
      - src/domain/**
    may_depend_on: []

  cameras:
    paths:
      - src/cameras/**
    may_depend_on:
      - domain
`
}
