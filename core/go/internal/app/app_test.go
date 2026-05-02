package app

import (
	"context"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/engine"
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
