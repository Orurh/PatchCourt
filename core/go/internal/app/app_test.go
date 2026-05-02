package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/engine"
	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type fakeAnalysisService struct {
	req    engine.AnalyzeRequest
	result *engine.AnalyzeResult
	err    error
}

func (f *fakeAnalysisService) Analyze(ctx context.Context, req engine.AnalyzeRequest) (*engine.AnalyzeResult, error) {
	f.req = req
	return f.result, f.err
}

func TestApp_RunScanUsesInjectedAnalysisService(t *testing.T) {
	fakeAnalysis := &fakeAnalysisService{
		result: &engine.AnalyzeResult{
			Project: &model.ProjectModel{
				Root: "/repo",
			},
			Config: &config.Config{},
		},
	}

	application := NewWithAnalysis(logx.Nop(), fakeAnalysis)

	result, err := application.RunScan(context.Background(), ScanRequest{
		Root:       "/repo",
		ConfigPath: "/repo/.patchcourt.yaml",
	})
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	if fakeAnalysis.req.Operation != "scan" {
		t.Fatalf("expected scan operation, got %q", fakeAnalysis.req.Operation)
	}

	if fakeAnalysis.req.Root != "/repo" {
		t.Fatalf("expected root /repo, got %q", fakeAnalysis.req.Root)
	}

	if fakeAnalysis.req.ConfigPath != "/repo/.patchcourt.yaml" {
		t.Fatalf("unexpected config path: %q", fakeAnalysis.req.ConfigPath)
	}

	if result.Project.Root != "/repo" {
		t.Fatalf("unexpected project root: %q", result.Project.Root)
	}
}

func TestApp_RunGraphUsesInjectedAnalysisService(t *testing.T) {
	fakeAnalysis := &fakeAnalysisService{
		result: &engine.AnalyzeResult{
			Project: &model.ProjectModel{
				Root: "/repo",
				Files: []model.FileModel{
					{
						Path:  "src/server/api_router.cc",
						Layer: "api",
					},
				},
			},
			Config: &config.Config{},
		},
	}

	application := NewWithAnalysis(logx.Nop(), fakeAnalysis)

	result, err := application.RunGraph(context.Background(), GraphRequest{
		Root:       "/repo",
		ConfigPath: "/repo/.patchcourt.yaml",
	})
	if err != nil {
		t.Fatalf("RunGraph failed: %v", err)
	}

	if fakeAnalysis.req.Operation != "graph" {
		t.Fatalf("expected graph operation, got %q", fakeAnalysis.req.Operation)
	}

	if len(result.LayerGraph.Nodes) != 1 {
		t.Fatalf("expected 1 graph node, got %d", len(result.LayerGraph.Nodes))
	}

	if result.LayerGraph.Nodes[0] != "api" {
		t.Fatalf("expected api node, got %q", result.LayerGraph.Nodes[0])
	}
}

func TestApp_RunGraphUsesDiscoveredLayersWithoutConfig(t *testing.T) {
	root := t.TempDir()

	writeGraphDiscoveryTestFile(t, root, "src/server/api_router.cc", `#include "src/controllers/device_orchestrator.h"
`)
	writeGraphDiscoveryTestFile(t, root, "src/controllers/device_orchestrator.h", `#pragma once
`)

	application := New(logx.Nop())

	result, err := application.RunGraph(context.Background(), GraphRequest{
		Root: root,
	})
	if err != nil {
		t.Fatalf("RunGraph failed: %v", err)
	}

	if len(result.LayerGraph.Nodes) != 2 {
		t.Fatalf("expected 2 graph nodes, got %d: %#v", len(result.LayerGraph.Nodes), result.LayerGraph.Nodes)
	}

	if !graphHasNode(result.LayerGraph.Nodes, "server") {
		t.Fatalf("expected server node in %#v", result.LayerGraph.Nodes)
	}

	if !graphHasNode(result.LayerGraph.Nodes, "controllers") {
		t.Fatalf("expected controllers node in %#v", result.LayerGraph.Nodes)
	}

	if !graphHasEdge(result.LayerGraph.Edges, "server", "controllers") {
		t.Fatalf("expected server -> controllers edge in %#v", result.LayerGraph.Edges)
	}

	if result.LayerGraph.Edges[0].Violation {
		t.Fatalf("discovered graph without policy must not mark violations")
	}
}

func writeGraphDiscoveryTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func graphHasNode(nodes []string, want string) bool {
	for _, node := range nodes {
		if node == want {
			return true
		}
	}

	return false
}

func graphHasEdge(edges []graphmodel.LayerEdge, from string, to string) bool {
	for _, edge := range edges {
		if edge.From == from && edge.To == to {
			return true
		}
	}

	return false
}
