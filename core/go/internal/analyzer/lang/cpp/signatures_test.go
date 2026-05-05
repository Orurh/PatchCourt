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

func TestExtractDeclaredSymbols_PublicMethods(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera_adapter.h")

	content := `#pragma once

class ICameraAdapter {
public:
	virtual bool RunPreflight() const = 0;
	bool StartSession(int count) noexcept;
	void StopSession();

private:
	bool InternalState() const;
};

struct CameraController {
	bool Connect();
protected:
	bool Reconnect();
};
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertMethod(t, symbols, "ICameraAdapter", "RunPreflight")
	assertMethod(t, symbols, "ICameraAdapter", "StartSession")
	assertMethod(t, symbols, "ICameraAdapter", "StopSession")
	assertMethod(t, symbols, "CameraController", "Connect")
	assertNoMethod(t, symbols, "ICameraAdapter", "InternalState")
	assertNoMethod(t, symbols, "CameraController", "Reconnect")
}

func assertMethod(t *testing.T, symbols []DeclaredSymbol, parent string, name string) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Kind == model.SymbolKindMethod && symbol.Parent == parent && symbol.Name == name {
			return
		}
	}

	t.Fatalf("expected method %s::%s in %#v", parent, name, symbols)
}

func assertNoMethod(t *testing.T, symbols []DeclaredSymbol, parent string, name string) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Kind == model.SymbolKindMethod && symbol.Parent == parent && symbol.Name == name {
			t.Fatalf("did not expect method %s::%s in %#v", parent, name, symbols)
		}
	}
}

func TestExtractDeclaredSymbols_MethodModifiers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera_adapter.h")

	content := `#pragma once

class ICameraAdapter {
public:
	virtual bool RunPreflight() const = 0;
	bool StartSession(int count) noexcept;
	void StopSession() override final;
};
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertMethodHasModifiers(t, symbols, "ICameraAdapter", "RunPreflight", []string{
		"virtual",
		"const",
		"pure_virtual",
	})

	assertMethodHasModifiers(t, symbols, "ICameraAdapter", "StartSession", []string{
		"noexcept",
	})

	assertMethodHasModifiers(t, symbols, "ICameraAdapter", "StopSession", []string{
		"override",
		"final",
	})
}

func assertMethodHasModifiers(t *testing.T, symbols []DeclaredSymbol, parent string, name string, want []string) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Kind == model.SymbolKindMethod && symbol.Parent == parent && symbol.Name == name {
			for _, modifier := range want {
				if !containsString(symbol.Modifiers, modifier) {
					t.Fatalf("expected method %s::%s to have modifier %q, got %#v", parent, name, modifier, symbol.Modifiers)
				}
			}

			return
		}
	}

	t.Fatalf("expected method %s::%s in %#v", parent, name, symbols)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func TestExtractDeclaredSymbols_FriendDeclarations(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera.h")

	content := `#pragma once

class Camera {
private:
	friend class CameraTest;
	friend bool operator==(const Camera& lhs, const Camera& rhs);

public:
	bool Start();
};
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertFriend(t, symbols, "Camera", "CameraTest")
	assertFriend(t, symbols, "Camera", "operator==")
}

func assertFriend(t *testing.T, symbols []DeclaredSymbol, parent string, name string) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Kind == model.SymbolKindFriend && symbol.Parent == parent && symbol.Name == name {
			if !symbol.Exported {
				t.Fatalf("expected friend %s::%s to be exported", parent, name)
			}

			return
		}
	}

	t.Fatalf("expected friend %s::%s in %#v", parent, name, symbols)
}

func TestExtractDeclaredSymbols_LineNumbers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera_adapter.h")

	content := `#pragma once

class ICameraAdapter {
public:
	virtual bool RunPreflight() const = 0;
	bool StartSession(int count) noexcept;
};
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write header: %v", err)
	}

	symbols, err := ExtractDeclaredSymbols(path)
	if err != nil {
		t.Fatalf("ExtractDeclaredSymbols failed: %v", err)
	}

	assertSymbolLine(t, symbols, model.SymbolKindClass, "", "ICameraAdapter", 3)
	assertSymbolLine(t, symbols, model.SymbolKindMethod, "ICameraAdapter", "RunPreflight", 5)
	assertSymbolLine(t, symbols, model.SymbolKindMethod, "ICameraAdapter", "StartSession", 6)
}

func assertSymbolLine(t *testing.T, symbols []DeclaredSymbol, kind model.SymbolKind, parent string, name string, wantLine int) {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Kind == kind && symbol.Parent == parent && symbol.Name == name {
			if symbol.Line != wantLine {
				t.Fatalf("expected %s/%s line %d, got %d in %#v", parent, name, wantLine, symbol.Line, symbol)
			}
			return
		}
	}

	t.Fatalf("expected symbol %s/%s in %#v", parent, name, symbols)
}
