package usecase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/diff/contract"
	"github.com/orurh/patchcourt/internal/diff/dep"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/reportmodel"
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

func TestApp_RunReviewDetectsCppContractDiffWithLineNumbers(t *testing.T) {
	root := t.TempDir()

	beforeRoot := filepath.Join(root, "before")
	afterRoot := filepath.Join(root, "after")

	writeFile(t, beforeRoot, "src/domain/interfaces/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
    virtual bool RunPreflight() const = 0;
    virtual bool StartSession(int count) const = 0;
};
`)

	writeFile(t, afterRoot, "src/domain/interfaces/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
    virtual bool RunPreflight(int camera_index) const = 0;
    bool StartSession(int count);
    virtual bool StopSession() const = 0;
};
`)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforeRoot: beforeRoot,
		AfterRoot:  afterRoot,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	if len(result.ContractChanges) != 4 {
		t.Fatalf("expected 4 contract changes, got %d: %#v", len(result.ContractChanges), result.ContractChanges)
	}

	runPreflightSignature := findContractChangeByKeyAndKind(
		result.ContractChanges,
		"method::ICameraAdapter::RunPreflight",
		contracts.ChangeKindSignatureChanged,
	)
	if runPreflightSignature == nil {
		t.Fatalf("expected RunPreflight signature change in %#v", result.ContractChanges)
	}

	if runPreflightSignature.Before == nil || runPreflightSignature.Before.Line != 5 {
		t.Fatalf("expected RunPreflight before line 5, got %#v", runPreflightSignature.Before)
	}

	if runPreflightSignature.After == nil || runPreflightSignature.After.Line != 5 {
		t.Fatalf("expected RunPreflight after line 5, got %#v", runPreflightSignature.After)
	}

	startSessionModifiers := findContractChangeByKeyAndKind(
		result.ContractChanges,
		"method::ICameraAdapter::StartSession",
		contracts.ChangeKindModifiersChanged,
	)
	if startSessionModifiers == nil {
		t.Fatalf("expected StartSession modifier change in %#v", result.ContractChanges)
	}

	requireStringSliceContains(t, startSessionModifiers.RemovedMods, "virtual")
	requireStringSliceContains(t, startSessionModifiers.RemovedMods, "const")
	requireStringSliceContains(t, startSessionModifiers.RemovedMods, "pure_virtual")

	stopSessionAdded := findContractChangeByKeyAndKind(
		result.ContractChanges,
		"method::ICameraAdapter::StopSession",
		contracts.ChangeKindAdded,
	)
	if stopSessionAdded == nil {
		t.Fatalf("expected StopSession added change in %#v", result.ContractChanges)
	}

	if stopSessionAdded.After == nil || stopSessionAdded.After.Line != 7 {
		t.Fatalf("expected StopSession after line 7, got %#v", stopSessionAdded.After)
	}
}

func findContractChangeByKeyAndKind(
	changes []contracts.SymbolChange,
	key string,
	kind contracts.ChangeKind,
) *contracts.SymbolChange {
	for i := range changes {
		if changes[i].SymbolKey == key && changes[i].Kind == kind {
			return &changes[i]
		}
	}

	return nil
}

func requireStringSliceContains(t *testing.T, values []string, want string) {
	t.Helper()

	for _, value := range values {
		if value == want {
			return
		}
	}

	t.Fatalf("expected %q in %#v", want, values)
}

func TestApp_RunReviewBuildsContractBlastRadius(t *testing.T) {
	root := t.TempDir()

	beforeRoot := filepath.Join(root, "before")
	afterRoot := filepath.Join(root, "after")
	configPath := filepath.Join(root, ".patchcourt.yaml")

	writeReviewTestFile(t, root, ".patchcourt.yaml", `
ignore:
  paths:
    - build/**
layers:
  api:
    paths:
      - src/api/**
    may_depend_on:
      - application
      - domain
  application:
    paths:
      - src/application/**
    may_depend_on:
      - domain
  domain:
    paths:
      - src/domain/**
    may_depend_on: []
  cameras:
    paths:
      - src/infrastructure/cameras/**
    may_depend_on:
      - domain
`)

	writeReviewTestFile(t, beforeRoot, "src/domain/interfaces/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
    virtual bool RunPreflight() const = 0;
};
`)

	writeReviewTestFile(t, beforeRoot, "src/application/camera_service.cc", `#include "domain/interfaces/i_camera_adapter.h"

bool Preflight(ICameraAdapter& camera) {
    return camera.RunPreflight();
}
`)

	writeReviewTestFile(t, afterRoot, "src/domain/interfaces/i_camera_adapter.h", `#pragma once

class ICameraAdapter {
public:
    virtual bool RunPreflight(int camera_index) const = 0;
};
`)

	writeReviewTestFile(t, afterRoot, "src/application/camera_service.cc", `#include "domain/interfaces/i_camera_adapter.h"

bool Preflight(ICameraAdapter& camera) {
    return camera.RunPreflight(0);
}
`)

	writeReviewTestFile(t, afterRoot, "src/api/camera_routes.cc", `#include "application/camera_service.h"
#include "domain/interfaces/i_camera_adapter.h"

bool HandlePreflight(ICameraAdapter& camera) {
    return camera.RunPreflight(0);
}
`)

	writeReviewTestFile(t, afterRoot, "src/infrastructure/cameras/sony/sony_camera_manager.h", `#pragma once

#include "domain/interfaces/i_camera_adapter.h"

class SonyCameraManager final : public ICameraAdapter {
public:
    bool RunPreflight(int camera_index) const override;
};
`)

	result, err := New(nil).RunReview(context.Background(), ReviewRequest{
		BeforeRoot: beforeRoot,
		AfterRoot:  afterRoot,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("RunReview failed: %v", err)
	}

	impact := findContractImpact(
		result.ContractImpacts,
		contracts.ChangeKindSignatureChanged,
		"method::ICameraAdapter::RunPreflight",
	)
	if impact == nil {
		t.Fatalf("expected RunPreflight contract impact in %#v", result.ContractImpacts)
	}

	if impact.Impact != "breaking" {
		t.Fatalf("expected breaking impact, got %#v", impact)
	}

	if !impact.DeliveryImpacted {
		t.Fatalf("expected delivery impact, got %#v", impact)
	}

	if impact.TestsChanged {
		t.Fatalf("expected no test-like files changed, got %#v", impact)
	}

	requireContractImpactedFile(t, impact.ImpactedFiles, "src/application/camera_service.cc", "likely_method_reference")
	requireContractImpactedFile(t, impact.ImpactedFiles, "src/api/camera_routes.cc", "likely_method_reference")
	requireContractImpactedFile(t, impact.ImpactedFiles, "src/infrastructure/cameras/sony/sony_camera_manager.h", "likely_implementation")

	if findImpactItem(result.Impact.NeedsReview, "contract_delivery_impact") == nil {
		t.Fatalf("expected signature change with impacted callers in NeedsReview: %#v", result.Impact.NeedsReview)
	}
}

func findContractImpact(
	impacts []reportmodel.ContractImpact,
	kind contracts.ChangeKind,
	symbolKey string,
) *reportmodel.ContractImpact {
	for i := range impacts {
		if impacts[i].ChangeKind == string(kind) && impacts[i].SymbolKey == symbolKey {
			return &impacts[i]
		}
	}

	return nil
}

func requireContractImpactedFile(
	t *testing.T,
	files []reportmodel.ContractImpactedFile,
	file string,
	reason string,
) {
	t.Helper()

	for _, item := range files {
		if item.File == file && item.Reason == reason {
			return
		}
	}

	t.Fatalf("expected impacted file %s with reason %s in %#v", file, reason, files)
}

func findImpactItem(items []reportmodel.ReviewImpactItem, kind string) *reportmodel.ReviewImpactItem {
	for i := range items {
		if items[i].Kind == kind {
			return &items[i]
		}
	}

	return nil
}
