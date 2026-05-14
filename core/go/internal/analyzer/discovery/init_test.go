package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateInitConfig_DiscoversLayersAndBaselineDependencies(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/server/api_router.cc", `#include "controllers/device_orchestrator.h"
`)
	writeFile(t, root, "src/controllers/device_orchestrator.h", `#pragma once
#include "domain/camera_status.h"
`)
	writeFile(t, root, "src/domain/camera_status.h", `#pragma once
`)

	result, err := GenerateInitConfig(InitOptions{
		Root: root,
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	if result == nil {
		t.Fatalf("expected init result")
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `cpp:`)
	assertContains(t, yaml, `compile_commands:`)
	assertContains(t, yaml, `auto_discover: true`)
	assertContains(t, yaml, `- "src"`)
	assertContains(t, yaml, `server:`)
	assertContains(t, yaml, `controllers:`)
	assertContains(t, yaml, `domain:`)
	assertContains(t, yaml, `- "src/server/**"`)
	assertContains(t, yaml, `- "src/controllers/**"`)
	assertContains(t, yaml, `- "src/domain/**"`)
	assertContains(t, yaml, `      - controllers`)
	assertContains(t, yaml, `      - domain`)
}

func TestGenerateInitConfig_StrictModeDoesNotInferAllowedDependencies(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/server/api_router.cc", `#include "controllers/device_orchestrator.h"
`)
	writeFile(t, root, "src/controllers/device_orchestrator.h", `#pragma once
#include "domain/camera_status.h"
`)
	writeFile(t, root, "src/domain/camera_status.h", `#pragma once
`)

	result, err := GenerateInitConfig(InitOptions{
		Root:   root,
		Strict: true,
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `# Strict mode: may_depend_on is intentionally empty for discovered layers.`)
	assertContains(t, yaml, `server:`)
	assertContains(t, yaml, `controllers:`)
	assertContains(t, yaml, `domain:`)

	if strings.Contains(yaml, `      - controllers`) {
		t.Fatalf("strict config must not infer server -> controllers dependency:\n%s", yaml)
	}

	if strings.Contains(yaml, `      - domain`) {
		t.Fatalf("strict config must not infer controllers -> domain dependency:\n%s", yaml)
	}
}

func TestGenerateInitConfig_IgnoresBuildAndGeneratedFiles(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/server/api_router.cc", `#include "domain/foo.h"
`)
	writeFile(t, root, "src/domain/foo.h", `#pragma once
`)
	writeFile(t, root, "build/temp.cc", `#include "missing.h"
`)
	writeFile(t, root, "generated/foo.pb.h", `#pragma once
`)

	result, err := GenerateInitConfig(InitOptions{
		Root: root,
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `server:`)
	assertContains(t, yaml, `domain:`)

	if strings.Contains(yaml, `build:`) {
		t.Fatalf("did not expect build layer in generated config:\n%s", yaml)
	}

	if strings.Contains(yaml, `generated:`) {
		t.Fatalf("did not expect generated layer in generated config:\n%s", yaml)
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

func assertContains(t *testing.T, value string, expected string) {
	t.Helper()

	if !strings.Contains(value, expected) {
		t.Fatalf("expected generated config to contain %q\n\nconfig:\n%s", expected, value)
	}
}

func TestGenerateInitConfig_GoCleanPreset(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "go.mod", `module github.com/orurh/patchcourt

go 1.26
`)
	writeFile(t, root, "cmd/patchcourt/main.go", `package main
`)
	writeFile(t, root, "internal/adapter/cli/root.go", `package cli
`)
	writeFile(t, root, "internal/usecase/app.go", `package usecase
`)
	writeFile(t, root, "internal/diff/project/compare.go", `package projectdiff
`)
	writeFile(t, root, "internal/source/source.go", `package source
`)
	writeFile(t, root, "internal/state/project_model.go", `package state
`)
	writeFile(t, root, "internal/analyzer/project/builder.go", `package project
`)
	writeFile(t, root, "internal/config/config.go", `package config
`)
	writeFile(t, root, "internal/model/project.go", `package model
`)
	writeFile(t, root, "internal/render/scan/text.go", `package scan
`)
	writeFile(t, root, "internal/reportmodel/reportmodel.go", `package scanmodel
`)
	writeFile(t, root, "internal/platform/git/worktree.go", `package git
`)

	result, err := GenerateInitConfig(InitOptions{
		Root:   root,
		Preset: "go-clean",
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `# Preset: go-clean`)
	assertContains(t, yaml, `cmd:`)
	assertContains(t, yaml, `cli:`)
	assertContains(t, yaml, `usecase:`)
	assertContains(t, yaml, `diff:`)
	assertContains(t, yaml, `source:`)
	assertContains(t, yaml, `state:`)
	assertContains(t, yaml, `analyzer:`)
	assertContains(t, yaml, `config:`)
	assertContains(t, yaml, `model:`)
	assertContains(t, yaml, `render:`)
	assertContains(t, yaml, `reportmodel:`)
	assertContains(t, yaml, `platform:`)

	assertContains(t, yaml, `      - "internal/usecase/**"`)
	assertContains(t, yaml, `      - analyzer`)
	assertContains(t, yaml, `      - diff`)
	assertContains(t, yaml, `      - source`)
	assertContains(t, yaml, `      - state`)
	assertContains(t, yaml, `      - render`)
	assertContains(t, yaml, `      - reportmodel`)
}

func TestGenerateInitConfig_NestedCPPPreset(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "src/core/client/http_session.h", `#pragma once
`)
	writeFile(t, root, "src/core/gopro_camera/gopro_camera.h", `#pragma once
#include "core/client/http_session.h"
`)
	writeFile(t, root, "src/core/gopro_camera/gopro_camera.cc", `#include "core/gopro_camera/gopro_camera.h"
`)
	writeFile(t, root, "src/core/gopro_manager/gopro_cameras_manager.cc", `#include "core/gopro_camera/gopro_camera.h"
`)
	writeFile(t, root, "src/utility/log.h", `#pragma once
`)

	result, err := GenerateInitConfig(InitOptions{
		Root:   root,
		Preset: "nested-cpp",
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `# Preset: nested-cpp`)
	assertContains(t, yaml, `src/core`)
	assertContains(t, yaml, `client:`)
	assertContains(t, yaml, `      - "src/core/client/**"`)
	assertContains(t, yaml, `gopro_camera:`)
	assertContains(t, yaml, `      - "src/core/gopro_camera/**"`)
	assertContains(t, yaml, `gopro_manager:`)
	assertContains(t, yaml, `      - "src/core/gopro_manager/**"`)
	assertContains(t, yaml, `utility:`)
	assertContains(t, yaml, `      - "src/utility/**"`)
	assertContains(t, yaml, `      - client`)
	assertContains(t, yaml, `      - gopro_camera`)
}

func TestGenerateInitConfig_SuggestsCurrentProjectStructure(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "go.mod", `module example.com/project

go 1.26
`)
	writeFile(t, root, "cmd/app/main.go", `package main

import _ "example.com/project/internal/adapter/cli"
`)
	writeFile(t, root, "internal/adapter/cli/root.go", `package cli

import _ "example.com/project/internal/usecase"
`)
	writeFile(t, root, "internal/usecase/app.go", `package usecase

import _ "example.com/project/internal/model"
`)
	writeFile(t, root, "internal/analyzer/project/builder.go", `package project

import _ "example.com/project/internal/model"
`)
	writeFile(t, root, "internal/model/project.go", `package model
`)
	writeFile(t, root, "internal/render/check/text.go", `package check

import _ "example.com/project/internal/reportmodel"
`)
	writeFile(t, root, "internal/reportmodel/reportmodel.go", `package reportmodel

import _ "example.com/project/internal/model"
`)

	result, err := GenerateInitConfig(InitOptions{
		Root:    root,
		Suggest: true,
	})
	if err != nil {
		t.Fatalf("GenerateInitConfig failed: %v", err)
	}

	yaml := result.ConfigYAML

	assertContains(t, yaml, `# Suggested config: layers are inferred from the current project structure.`)
	assertContains(t, yaml, `# Use this to replace outdated or too narrow .patchcourt.yaml files.`)

	assertContains(t, yaml, `internal_adapter:`)
	assertContains(t, yaml, `      - "internal/adapter/**"`)
	assertContains(t, yaml, `internal_usecase:`)
	assertContains(t, yaml, `      - "internal/usecase/**"`)
	assertContains(t, yaml, `internal_analyzer:`)
	assertContains(t, yaml, `      - "internal/analyzer/**"`)
	assertContains(t, yaml, `internal_model:`)
	assertContains(t, yaml, `      - "internal/model/**"`)
	assertContains(t, yaml, `internal_render:`)
	assertContains(t, yaml, `      - "internal/render/**"`)
	assertContains(t, yaml, `internal_reportmodel:`)
	assertContains(t, yaml, `      - "internal/reportmodel/**"`)

	assertContains(t, yaml, `      - internal_usecase`)
	assertContains(t, yaml, `      - internal_model`)
	assertContains(t, yaml, `      - internal_reportmodel`)
}

func TestGenerateInitConfig_RejectsSuggestWithPreset(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "internal/model/project.go", `package model
`)

	_, err := GenerateInitConfig(InitOptions{
		Root:    root,
		Suggest: true,
		Preset:  "go-clean",
	})
	if err == nil {
		t.Fatalf("expected suggest with preset to fail")
	}

	if !strings.Contains(err.Error(), "--suggest cannot be combined with preset") {
		t.Fatalf("unexpected error: %v", err)
	}
}
