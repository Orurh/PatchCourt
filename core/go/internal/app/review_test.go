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
