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
	writeFile(t, root, "internal/cli/root.go", `package cli
`)
	writeFile(t, root, "internal/app/app.go", `package app
`)
	writeFile(t, root, "internal/changes/compare.go", `package changes
`)
	writeFile(t, root, "internal/analysis/project/builder.go", `package project
`)
	writeFile(t, root, "internal/config/config.go", `package config
`)
	writeFile(t, root, "internal/model/project.go", `package model
`)
	writeFile(t, root, "internal/output/report/text.go", `package report
`)
	writeFile(t, root, "internal/reportmodel/reportmodel.go", `package reportmodel
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
	assertContains(t, yaml, `app:`)
	assertContains(t, yaml, `changes:`)
	assertContains(t, yaml, `analysis:`)
	assertContains(t, yaml, `config:`)
	assertContains(t, yaml, `model:`)
	assertContains(t, yaml, `output:`)
	assertContains(t, yaml, `reportmodel:`)
	assertContains(t, yaml, `platform:`)

	assertContains(t, yaml, `      - "internal/app/**"`)
	assertContains(t, yaml, `      - analysis`)
	assertContains(t, yaml, `      - changes`)
	assertContains(t, yaml, `      - output`)
	assertContains(t, yaml, `      - reportmodel`)
}
