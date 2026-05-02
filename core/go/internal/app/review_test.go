package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/contracts"
	"github.com/orurh/patchcourt/internal/model"
)

func TestApp_RunReviewDiffsProjectModelSnapshots(t *testing.T) {
	root := t.TempDir()

	before := model.ProjectModel{
		Symbols: []model.SymbolModel{
			{
				Name:       "RunPreflight",
				Kind:       model.SymbolKindMethod,
				Parent:     "ICameraAdapter",
				Signature:  "virtual bool RunPreflight() const = 0;",
				Modifiers:  []string{"virtual", "const", "pure_virtual"},
				Exported:   true,
				Visibility: "public",
				Confidence: model.ConfidenceLow,
			},
		},
	}

	after := model.ProjectModel{
		Symbols: []model.SymbolModel{
			{
				Name:       "RunPreflight",
				Kind:       model.SymbolKindMethod,
				Parent:     "ICameraAdapter",
				Signature:  "bool RunPreflight();",
				Exported:   true,
				Visibility: "public",
				Confidence: model.ConfidenceLow,
			},
		},
	}

	beforePath := writeProjectModelJSON(t, root, "before.json", before)
	afterPath := writeProjectModelJSON(t, root, "after.json", after)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforePath: beforePath,
		AfterPath:  afterPath,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	if len(result.ContractChanges) != 2 {
		t.Fatalf("expected signature and modifier changes, got %d: %#v", len(result.ContractChanges), result.ContractChanges)
	}

	if findContractChange(result.ContractChanges, contracts.ChangeKindSignatureChanged) == nil {
		t.Fatalf("expected signature change in %#v", result.ContractChanges)
	}

	if findContractChange(result.ContractChanges, contracts.ChangeKindModifiersChanged) == nil {
		t.Fatalf("expected modifiers change in %#v", result.ContractChanges)
	}
}

func writeProjectModelJSON(t *testing.T, root string, name string, project model.ProjectModel) string {
	t.Helper()

	path := filepath.Join(root, name)

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("marshal project model: %v", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write project model: %v", err)
	}

	return path
}

func findContractChange(changes []contracts.SymbolChange, kind contracts.ChangeKind) *contracts.SymbolChange {
	for i := range changes {
		if changes[i].Kind == kind {
			return &changes[i]
		}
	}

	return nil
}

func TestApp_RunReviewDiffsProjectRoots(t *testing.T) {
	root := t.TempDir()

	beforeRoot := filepath.Join(root, "before")
	afterRoot := filepath.Join(root, "after")
	configPath := filepath.Join(root, ".patchcourt.yaml")

	writeReviewTestFile(t, beforeRoot, "src/domain/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
	virtual bool RunPreflight() const = 0;
};
`)

	writeReviewTestFile(t, afterRoot, "src/domain/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
	bool RunPreflight();
};
`)

	writeReviewTestFile(t, root, ".patchcourt.yaml", `
ignore:
  paths:
    - build/**
layers:
  domain:
    paths:
      - src/domain/**
    may_depend_on: []
`)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforeRoot: beforeRoot,
		AfterRoot:  afterRoot,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	if len(result.ContractChanges) != 2 {
		t.Fatalf("expected signature and modifier changes, got %d: %#v", len(result.ContractChanges), result.ContractChanges)
	}

	if findContractChange(result.ContractChanges, contracts.ChangeKindSignatureChanged) == nil {
		t.Fatalf("expected signature change in %#v", result.ContractChanges)
	}

	if findContractChange(result.ContractChanges, contracts.ChangeKindModifiersChanged) == nil {
		t.Fatalf("expected modifiers change in %#v", result.ContractChanges)
	}
}

func writeReviewTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
