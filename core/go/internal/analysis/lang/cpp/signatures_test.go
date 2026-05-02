package cpp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestExtractDeclaredSymbols_CppTypes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera.h")

	content := `#pragma once

class CameraManager;
struct CameraStatus {
	int connected;
};

enum CameraMode {
	CameraModePhoto,
};

enum class CameraState {
	Idle,
	Running,
};

using CameraList = std::vector<int>;
typedef unsigned long CameraId;

// Should be ignored:
#define class MacroClass
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertSymbol(t, symbols, "CameraManager", model.SymbolKindClass)
	assertSymbol(t, symbols, "CameraStatus", model.SymbolKindStruct)
	assertSymbol(t, symbols, "CameraMode", model.SymbolKindEnum)
	assertSymbol(t, symbols, "CameraState", model.SymbolKindEnum)
	assertSymbol(t, symbols, "CameraList", model.SymbolKindUsing)
	assertSymbol(t, symbols, "CameraId", model.SymbolKindTypedef)
}

func TestExtractDeclaredSymbols_TemplateClassAndStruct(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "container.h")

	content := `
template <typename T>
class CameraBox {};

template <typename T>
struct CameraView {};
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertSymbol(t, symbols, "CameraBox", model.SymbolKindClass)
	assertSymbol(t, symbols, "CameraView", model.SymbolKindStruct)
}

func assertSymbol(t *testing.T, symbols []DeclaredSymbol, name string, kind model.SymbolKind) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Name == name && symbol.Kind == kind {
			return
		}
	}

	t.Fatalf("expected symbol %s/%s in %#v", name, kind, symbols)
}
