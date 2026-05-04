package project

import (
	"testing"

	"github.com/orurh/patchcourt/internal/model"
)

func TestDiscoverLayer_EntrypointMainCC(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "src/main.cc",
			want: "entrypoint",
		},
		{
			path: "main.cc",
			want: "entrypoint",
		},
		{
			path: "cmd/patchcourt/main.go",
			want: "entrypoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DiscoverLayer(tt.path)
			if got != tt.want {
				t.Fatalf("DiscoverLayer(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDiscoverLayer_SrcDirectoryLayer(t *testing.T) {
	got := DiscoverLayer("src/controllers/device_orchestrator.cc")
	if got != "controllers" {
		t.Fatalf("expected controllers layer, got %q", got)
	}
}

func TestDiscoverLayer_SharedAliases(t *testing.T) {
	tests := []string{
		"src/utils/json_serializer.h",
		"src/configs/session_file_manager.h",
		"src/common/types.h",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			got := DiscoverLayer(path)
			if got != "shared" {
				t.Fatalf("expected shared layer for %q, got %q", path, got)
			}
		})
	}
}

func TestAssignDiscoveredLayers_AssignsLayerAndDependencyLayers(t *testing.T) {
	project := &model.ProjectModel{
		Files: []model.FileModel{
			{
				Path:     "src/server/api_router.cc",
				Language: model.LanguageCPP,
				Role:     model.FileRoleProduction,
			},
			{
				Path:     "src/controllers/device_orchestrator.h",
				Language: model.LanguageCPP,
				Role:     model.FileRoleProduction,
			},
		},
		Dependencies: []model.DependencyEdge{
			{
				FromFile: "src/server/api_router.cc",
				ToFile:   "src/controllers/device_orchestrator.h",
				Resolved: true,
			},
		},
	}

	AssignDiscoveredLayers(project)

	if project.Files[0].Layer != "server" {
		t.Fatalf("expected server layer, got %q", project.Files[0].Layer)
	}

	if project.Files[0].LayerSource != model.LayerAssignmentSourceDiscovered {
		t.Fatalf("expected discovered layer source, got %q", project.Files[0].LayerSource)
	}

	if project.Dependencies[0].FromLayer != "server" {
		t.Fatalf("expected dependency from layer server, got %q", project.Dependencies[0].FromLayer)
	}

	if project.Dependencies[0].ToLayer != "controllers" {
		t.Fatalf("expected dependency to layer controllers, got %q", project.Dependencies[0].ToLayer)
	}
}
