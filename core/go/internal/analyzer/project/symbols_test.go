package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestBuild_CollectsCPPDeclaredSymbols(t *testing.T) {
	root := t.TempDir()

	writeProjectTestFile(t, root, "src/domain/camera_status.h", `#pragma once

class CameraManager;
struct CameraStatus {};
enum class CameraState {};
using CameraList = std::vector<int>;
`)

	project, err := Build(Options{
		Root: root,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	assertProjectSymbol(t, project.Symbols, "CameraManager", model.SymbolKindClass)
	assertProjectSymbol(t, project.Symbols, "CameraStatus", model.SymbolKindStruct)
	assertProjectSymbol(t, project.Symbols, "CameraState", model.SymbolKindEnum)
	assertProjectSymbol(t, project.Symbols, "CameraList", model.SymbolKindUsing)

	if len(project.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(project.Files))
	}

	assertProjectSymbol(t, project.Files[0].Symbols, "CameraStatus", model.SymbolKindStruct)
}

func writeProjectTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func assertProjectSymbol(t *testing.T, symbols []model.SymbolModel, name string, kind model.SymbolKind) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Name == name && symbol.Kind == kind {
			return
		}
	}

	t.Fatalf("expected symbol %s/%s in %#v", name, kind, symbols)
}
