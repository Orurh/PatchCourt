package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
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

func TestApp_RunReviewDiffsDependencyEdgesFromSnapshots(t *testing.T) {
	root := t.TempDir()

	before := model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			reviewDep("src/server/api_router.cc", "src/domain/status.h", "server", "domain"),
		},
	}

	after := model.ProjectModel{
		Dependencies: []model.DependencyEdge{
			reviewDep("src/server/api_router.cc", "src/cameras/sony.h", "server", "cameras"),
		},
	}

	beforePath := writeProjectModelJSON(t, root, "before-deps.json", before)
	afterPath := writeProjectModelJSON(t, root, "after-deps.json", after)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforePath: beforePath,
		AfterPath:  afterPath,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	if len(result.DependencyChanges) != 2 {
		t.Fatalf("expected added and removed dependency changes, got %d: %#v", len(result.DependencyChanges), result.DependencyChanges)
	}

	if findDependencyChange(result.DependencyChanges, depdiff.DependencyChangeKindAdded) == nil {
		t.Fatalf("expected added dependency change in %#v", result.DependencyChanges)
	}

	if findDependencyChange(result.DependencyChanges, depdiff.DependencyChangeKindRemoved) == nil {
		t.Fatalf("expected removed dependency change in %#v", result.DependencyChanges)
	}

	if len(result.LayerEdgeChanges) != 2 {
		t.Fatalf("expected added and removed layer edge changes, got %d: %#v", len(result.LayerEdgeChanges), result.LayerEdgeChanges)
	}
}

func reviewDep(fromFile string, toFile string, fromLayer string, toLayer string) model.DependencyEdge {
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

func findDependencyChange(changes []depdiff.DependencyChange, kind depdiff.DependencyChangeKind) *depdiff.DependencyChange {
	for i := range changes {
		if changes[i].Kind == kind {
			return &changes[i]
		}
	}

	return nil
}

func TestApp_RunReviewDiffsFindingsAndCalculatesRisk(t *testing.T) {
	root := t.TempDir()

	before := model.ProjectModel{}

	after := model.ProjectModel{
		Findings: []model.Finding{
			{
				ID:         "architecture.api.cameras",
				Kind:       model.FindingKindPolicyViolation,
				Severity:   model.SeverityHigh,
				Title:      "Include-level architecture boundary violation",
				Confidence: model.ConfidenceHigh,
			},
		},
	}

	beforePath := writeProjectModelJSON(t, root, "before-findings.json", before)
	afterPath := writeProjectModelJSON(t, root, "after-findings.json", after)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforePath: beforePath,
		AfterPath:  afterPath,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	if len(result.FindingChanges) != 1 {
		t.Fatalf("expected 1 finding change, got %d: %#v", len(result.FindingChanges), result.FindingChanges)
	}

	if result.Summary.FindingChanges != 1 {
		t.Fatalf("expected finding changes summary 1, got %d", result.Summary.FindingChanges)
	}

	if result.Summary.AddedFindings != 1 {
		t.Fatalf("expected added findings summary 1, got %d", result.Summary.AddedFindings)
	}

	if result.Summary.AddedHighFindings != 1 {
		t.Fatalf("expected added high findings summary 1, got %d", result.Summary.AddedHighFindings)
	}

	if result.Summary.AddedPolicyFindings != 1 {
		t.Fatalf("expected added policy findings summary 1, got %d", result.Summary.AddedPolicyFindings)
	}

	if result.Risk.Points == 0 {
		t.Fatalf("expected non-zero risk score")
	}
}
