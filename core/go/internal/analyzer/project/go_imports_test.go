package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBuild_CollectsGoImportDependencies(t *testing.T) {
	root := t.TempDir()

	writeGoImportProjectTestFile(t, root, "go.mod", `module github.com/orurh/patchcourt

go 1.26
`)

	writeGoImportProjectTestFile(t, root, "cmd/patchcourt/main.go", `package main

import (
	"context"

	"github.com/orurh/patchcourt/internal/usecase"
)

func main() {
	_ = context.Background()
}
`)

	writeGoImportProjectTestFile(t, root, "internal/usecase/app.go", `package usecase

type App struct{}
`)

	project, err := Build(Options{
		Root: root,
	})
	require.NoError(t, err)

	local := findDependency(project.Dependencies, "cmd/patchcourt/main.go", "github.com/orurh/patchcourt/internal/usecase")
	require.NotNil(t, local)
	require.Equal(t, model.DependencyKindImport, local.Kind)
	require.True(t, local.Resolved)
	require.False(t, local.External)
	require.Equal(t, "internal/usecase/app.go", local.ToFile)

	stdlib := findDependency(project.Dependencies, "cmd/patchcourt/main.go", "context")
	require.NotNil(t, stdlib)
	require.Equal(t, model.DependencyKindImport, stdlib.Kind)
	require.False(t, stdlib.Resolved)
	require.True(t, stdlib.External)
}

func findDependency(deps []model.DependencyEdge, fromFile string, target string) *model.DependencyEdge {
	for i := range deps {
		if deps[i].FromFile == fromFile && deps[i].Target == target {
			return &deps[i]
		}
	}

	return nil
}

func writeGoImportProjectTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
}

func TestBuild_CollectsGoDependenciesFromNestedModule(t *testing.T) {
	root := t.TempDir()

	writeGoImportProjectTestFile(t, root, "services/billing/go.mod", `module example.com/billing

go 1.22
`)

	writeGoImportProjectTestFile(t, root, "services/billing/cmd/app/main.go", `package main

import "example.com/billing/internal/usecase"

func main() {
	usecase.Run()
}
`)

	writeGoImportProjectTestFile(t, root, "services/billing/internal/usecase/run.go", `package usecase

func Run() {}
`)

	project, err := Build(Options{Root: root})
	require.NoError(t, err)

	dep := findDependency(project.Dependencies, "services/billing/cmd/app/main.go", "example.com/billing/internal/usecase")
	require.NotNil(t, dep)
	require.True(t, dep.Resolved)
	require.Equal(t, "services/billing/internal/usecase/run.go", dep.ToFile)
}
