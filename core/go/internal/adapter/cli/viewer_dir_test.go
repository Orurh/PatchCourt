package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveViewerDirExplicit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveViewerDir(dir)
	if err != nil {
		t.Fatalf("resolveViewerDir returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty viewer dir")
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
}

func TestResolveViewerDirExplicitMissingIndex(t *testing.T) {
	dir := t.TempDir()

	_, err := resolveViewerDir(dir)
	if err == nil {
		t.Fatal("expected error for missing index.html")
	}
}

func TestResolveOptionalViewerDirEmpty(t *testing.T) {
	got, err := resolveOptionalViewerDir("")
	if err != nil {
		t.Fatalf("resolveOptionalViewerDir returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty viewer dir, got %q", got)
	}
}

func TestRequireViewerDistRejectsIndexDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "index.html"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := requireViewerDist(dir)
	if err == nil {
		t.Fatal("expected error for index.html directory")
	}
}
