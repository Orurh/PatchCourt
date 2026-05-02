package contracts

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestDiffSymbols_DetectsAddedSymbol(t *testing.T) {
	changes := DiffSymbols(nil, []model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "bool RunPreflight() const;", nil),
	})

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %#v", len(changes), changes)
	}

	if changes[0].Kind != ChangeKindAdded {
		t.Fatalf("expected added change, got %q", changes[0].Kind)
	}

	if changes[0].SymbolKey != "method::ICameraAdapter::RunPreflight" {
		t.Fatalf("unexpected symbol key: %q", changes[0].SymbolKey)
	}
}

func TestDiffSymbols_DetectsRemovedSymbol(t *testing.T) {
	changes := DiffSymbols([]model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "bool RunPreflight() const;", nil),
	}, nil)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %#v", len(changes), changes)
	}

	if changes[0].Kind != ChangeKindRemoved {
		t.Fatalf("expected removed change, got %q", changes[0].Kind)
	}
}

func TestDiffSymbols_DetectsSignatureChange(t *testing.T) {
	before := []model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "bool RunPreflight() const;", []string{"const"}),
	}

	after := []model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "bool RunPreflight(int cameraIndex) const;", []string{"const"}),
	}

	changes := DiffSymbols(before, after)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %#v", len(changes), changes)
	}

	if changes[0].Kind != ChangeKindSignatureChanged {
		t.Fatalf("expected signature change, got %q", changes[0].Kind)
	}
}

func TestDiffSymbols_DetectsModifierChange(t *testing.T) {
	before := []model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "virtual bool RunPreflight() const = 0;", []string{
			"virtual",
			"const",
			"pure_virtual",
		}),
	}

	after := []model.SymbolModel{
		methodSymbol("ICameraAdapter", "RunPreflight", "bool RunPreflight();", nil),
	}

	changes := DiffSymbols(before, after)

	if len(changes) != 2 {
		t.Fatalf("expected signature and modifier changes, got %d: %#v", len(changes), changes)
	}

	modifierChange := findChange(changes, ChangeKindModifiersChanged)
	if modifierChange == nil {
		t.Fatalf("expected modifier change in %#v", changes)
	}

	assertStringSliceContains(t, modifierChange.RemovedMods, "virtual")
	assertStringSliceContains(t, modifierChange.RemovedMods, "const")
	assertStringSliceContains(t, modifierChange.RemovedMods, "pure_virtual")
}

func TestDiffSymbols_IgnoresNonExportedSymbols(t *testing.T) {
	changes := DiffSymbols([]model.SymbolModel{
		{
			Name:     "InternalState",
			Kind:     model.SymbolKindMethod,
			Parent:   "ICameraAdapter",
			Exported: false,
		},
	}, nil)

	if len(changes) != 0 {
		t.Fatalf("expected non-exported symbol to be ignored, got %#v", changes)
	}
}

func TestDiffSymbols_DetectsFriendAdded(t *testing.T) {
	changes := DiffSymbols(nil, []model.SymbolModel{
		{
			Name:       "CameraTest",
			Kind:       model.SymbolKindFriend,
			Parent:     "Camera",
			Signature:  "friend class CameraTest;",
			Exported:   true,
			Visibility: "private",
			Confidence: model.ConfidenceLow,
		},
	})

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %#v", len(changes), changes)
	}

	if changes[0].Kind != ChangeKindAdded {
		t.Fatalf("expected friend added, got %q", changes[0].Kind)
	}

	if changes[0].SymbolKey != "friend::Camera::CameraTest" {
		t.Fatalf("unexpected symbol key: %q", changes[0].SymbolKey)
	}
}

func methodSymbol(parent string, name string, signature string, modifiers []string) model.SymbolModel {
	return model.SymbolModel{
		Name:       name,
		Kind:       model.SymbolKindMethod,
		Parent:     parent,
		Signature:  signature,
		Modifiers:  modifiers,
		Exported:   true,
		Visibility: "public",
		Confidence: model.ConfidenceLow,
	}
}

func findChange(changes []SymbolChange, kind ChangeKind) *SymbolChange {
	for i := range changes {
		if changes[i].Kind == kind {
			return &changes[i]
		}
	}

	return nil
}

func assertStringSliceContains(t *testing.T, values []string, expected string) {
	t.Helper()

	for _, value := range values {
		if value == expected {
			return
		}
	}

	t.Fatalf("expected %q in %#v", expected, values)
}
