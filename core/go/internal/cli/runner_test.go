package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
	"github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/analysis/risk"
	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/stretchr/testify/require"
)

type fakeApplication struct {
	initReq    app.InitRequest
	scanReq    app.ScanRequest
	graphReq   app.GraphRequest
	reviewReq  app.ReviewRequest
	explainReq app.ExplainRequest
	edgeReq    app.EdgeRequest
	checkReq   app.CheckRequest

	initResult    *app.InitResult
	scanResult    *app.ScanResult
	graphResult   *app.GraphResult
	reviewResult  *app.ReviewResult
	explainResult *app.ExplainResult
	edgeResult    *app.EdgeResult
	checkResult   *app.CheckResult

	initErr    error
	scanErr    error
	graphErr   error
	reviewErr  error
	explainErr error
	edgeErr    error
	checkErr   error
}

func (f *fakeApplication) RunInit(ctx context.Context, req app.InitRequest) (*app.InitResult, error) {
	f.initReq = req
	return f.initResult, f.initErr
}

func (f *fakeApplication) RunScan(ctx context.Context, req app.ScanRequest) (*app.ScanResult, error) {
	f.scanReq = req
	return f.scanResult, f.scanErr
}

func (f *fakeApplication) RunGraph(ctx context.Context, req app.GraphRequest) (*app.GraphResult, error) {
	f.graphReq = req
	return f.graphResult, f.graphErr
}

func (f *fakeApplication) RunReview(ctx context.Context, req app.ReviewRequest) (*app.ReviewResult, error) {
	f.reviewReq = req
	return f.reviewResult, f.reviewErr
}

func (f *fakeApplication) RunExplain(ctx context.Context, req app.ExplainRequest) (*app.ExplainResult, error) {
	f.explainReq = req
	return f.explainResult, f.explainErr
}

func (f *fakeApplication) RunCheck(ctx context.Context, req app.CheckRequest) (*app.CheckResult, error) {
	f.checkReq = req
	return f.checkResult, f.checkErr
}

func (f *fakeApplication) RunEdge(ctx context.Context, req app.EdgeRequest) (*app.EdgeResult, error) {
	f.edgeReq = req
	return f.edgeResult, f.edgeErr
}

func TestRunner_RunInitUsesInjectedApplication(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeApp := &fakeApplication{
		initResult: &app.InitResult{
			ConfigYAML: "generated: true\n",
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"init",
		"/repo",
		"--strict",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if fakeApp.initReq.Root != "/repo" {
		t.Fatalf("expected root /repo, got %q", fakeApp.initReq.Root)
	}

	if !fakeApp.initReq.Strict {
		t.Fatalf("expected strict init request")
	}

	if stdout.String() != "generated: true\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunner_RunScanUsesInjectedApplication(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeApp := &fakeApplication{
		scanResult: &app.ScanResult{
			Project: &model.ProjectModel{
				Root: "/repo",
			},
			Config: &config.Config{},
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"scan",
		"/repo",
		"--config",
		"/repo/.patchcourt.yaml",
		"--format",
		"text",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if fakeApp.scanReq.Root != "/repo" {
		t.Fatalf("expected root /repo, got %q", fakeApp.scanReq.Root)
	}

	if fakeApp.scanReq.ConfigPath != "/repo/.patchcourt.yaml" {
		t.Fatalf("unexpected config path: %q", fakeApp.scanReq.ConfigPath)
	}

	output := stdout.String()

	if !strings.Contains(output, "PatchCourt scan") {
		t.Fatalf("expected scan report in stdout, got:\n%s", output)
	}

	if !strings.Contains(output, "Root: /repo") {
		t.Fatalf("expected root in stdout, got:\n%s", output)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunner_RunExplainUsesInjectedApplication(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeApp := &fakeApplication{
		explainResult: &app.ExplainResult{
			Source: "/repo",
			Finding: model.Finding{
				ID:         "architecture.api.cameras",
				Kind:       model.FindingKindPolicyViolation,
				Severity:   model.SeverityHigh,
				Title:      "Include-level architecture boundary violation",
				Confidence: model.ConfidenceHigh,
				Risk:       "Layer api includes cameras.",
				Suggestion: "Move dependency behind an interface.",
				Evidence: []model.Evidence{
					{
						File:    "src/server/api_router.cc",
						Message: "includes src/cameras/sony.h",
					},
				},
			},
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"explain",
		"architecture.api.cameras",
		"--root",
		"/repo",
		"--config",
		"/repo/.patchcourt.yaml",
		"--format",
		"text",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if fakeApp.explainReq.FindingID != "architecture.api.cameras" {
		t.Fatalf("expected finding id architecture.api.cameras, got %q", fakeApp.explainReq.FindingID)
	}

	if fakeApp.explainReq.Root != "/repo" {
		t.Fatalf("expected root /repo, got %q", fakeApp.explainReq.Root)
	}

	if fakeApp.explainReq.ConfigPath != "/repo/.patchcourt.yaml" {
		t.Fatalf("unexpected config path: %q", fakeApp.explainReq.ConfigPath)
	}

	output := stdout.String()

	if !strings.Contains(output, "PatchCourt explain") {
		t.Fatalf("expected explain report in stdout, got:\n%s", output)
	}

	if !strings.Contains(output, "architecture.api.cameras") {
		t.Fatalf("expected finding id in stdout, got:\n%s", output)
	}

	if !strings.Contains(output, "includes src/cameras/sony.h") {
		t.Fatalf("expected evidence in stdout, got:\n%s", output)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunner_RunReviewRendersMarkdown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeApp := &fakeApplication{
		reviewResult: &app.ReviewResult{
			Summary: app.ReviewSummary{
				FindingChanges:      1,
				AddedFindings:       1,
				AddedHighFindings:   1,
				AddedPolicyFindings: 1,
			},
			Risk: risk.Score{
				Points: 7,
				Level:  risk.LevelMedium,
				Reasons: []risk.Reason{
					{
						Points:  7,
						Message: "added high policy violation: architecture.api.cameras",
					},
				},
			},
			FindingChanges: []findingdiff.FindingChange{
				{
					Kind: findingdiff.FindingChangeKindAdded,
					ID:   "architecture.api.cameras",
					After: &model.Finding{
						ID:         "architecture.api.cameras",
						Kind:       model.FindingKindPolicyViolation,
						Severity:   model.SeverityHigh,
						Title:      "Include-level architecture boundary violation",
						Confidence: model.ConfidenceHigh,
					},
				},
			},
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"review",
		"--before-root",
		"/repo/before",
		"--after-root",
		"/repo/after",
		"--config",
		"/repo/.patchcourt.yaml",
		"--format",
		"markdown",
	})
	require.NoError(t, err)

	got := stdout.String()

	require.Contains(t, got, "# PatchCourt Review")
	require.Contains(t, got, "patchcourt explain architecture.api.cameras --root /repo/after --config /repo/.patchcourt.yaml")
	require.Empty(t, stderr.String())
	require.Equal(t, "/repo/after", fakeApp.reviewReq.AfterRoot)
	require.Equal(t, "/repo/.patchcourt.yaml", fakeApp.reviewReq.ConfigPath)
}

func TestRunner_RunCheckUsesInjectedApplication(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outDir := t.TempDir()

	fakeApp := &fakeApplication{
		checkResult: &app.CheckResult{
			Root:   "/repo",
			OutDir: outDir,
			Project: &model.ProjectModel{
				Root: "/repo",
				Files: []model.FileModel{
					{Path: "src/server/api_router.cc", Layer: "server", Role: model.FileRoleProduction},
					{Path: "src/domain/status.h", Layer: "domain", Role: model.FileRoleProduction},
				},
				Dependencies: []model.DependencyEdge{
					{
						FromFile:  "src/server/api_router.cc",
						ToFile:    "src/domain/status.h",
						Target:    "src/domain/status.h",
						Kind:      model.DependencyKindInclude,
						Resolved:  true,
						FromLayer: "server",
						ToLayer:   "domain",
					},
				},
				Findings: []model.Finding{
					{
						ID:       "discovery.cpp.unused_includes",
						Kind:     model.FindingKindDiscoveryHint,
						Severity: model.SeverityLow,
						Title:    "Possibly unused C++ includes",
					},
				},
			},
			Summary: model.ScanSummary{
				ProductionFiles: 2,
				TotalEdges:      1,
				Resolved:        1,
			},
			LayerGraph: graph.LayerGraph{
				Nodes: []string{"domain", "server"},
				Edges: []graph.LayerEdge{
					{From: "server", To: "domain", Count: 1},
				},
			},
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"check",
		"/repo",
		"--config",
		"/repo/.patchcourt.yaml",
		"--out",
		outDir,
	})
	require.NoError(t, err)

	require.Equal(t, "/repo", fakeApp.checkReq.Root)
	require.Equal(t, "/repo/.patchcourt.yaml", fakeApp.checkReq.ConfigPath)
	require.Equal(t, outDir, fakeApp.checkReq.OutDir)

	output := stdout.String()
	require.Contains(t, output, "PatchCourt check")
	require.Contains(t, output, "Artifacts:")
	require.Contains(t, output, filepath.Join(outDir, "project-model.json"))
	require.Contains(t, output, "discovery.cpp.unused_includes")
	require.Empty(t, stderr.String())

	requireFileExists(t, filepath.Join(outDir, "project-model.json"))
	requireFileExists(t, filepath.Join(outDir, "scan.md"))
	requireFileExists(t, filepath.Join(outDir, "layer-graph.json"))
	requireFileExists(t, filepath.Join(outDir, "layer-graph.dot"))
	requireFileExists(t, filepath.Join(outDir, "layer-graph.mmd"))
}

func requireFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
}

func TestRunner_RunEdgeUsesInjectedApplication(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeApp := &fakeApplication{
		edgeResult: &app.EdgeResult{
			Source:    "/repo/.patchcourt/out/project-model.json",
			FromLayer: "application",
			ToLayer:   "cameras",
			Count:     1,
			Usage: app.EdgeUsageSummary{
				Used: 1,
			},
			Dependencies: []model.DependencyEdge{
				{
					FromFile: "src/application/bootstrapper.cc",
					ToFile:   "src/cameras/camera_adapter_factory.h",
					Usage:    model.DependencyUsageUsed,
				},
			},
		},
	}

	runner := NewRunner(&stdout, &stderr).WithAppFactory(func(logger logx.Logger) Application {
		return fakeApp
	})

	err := runner.Run(context.Background(), []string{
		"edge",
		"--model",
		"/repo/.patchcourt/out/project-model.json",
		"application",
		"cameras",
	})
	require.NoError(t, err)

	require.Equal(t, "/repo/.patchcourt/out/project-model.json", fakeApp.edgeReq.ModelPath)
	require.Equal(t, "application", fakeApp.edgeReq.FromLayer)
	require.Equal(t, "cameras", fakeApp.edgeReq.ToLayer)

	output := stdout.String()
	require.Contains(t, output, "PatchCourt edge")
	require.Contains(t, output, "Edge: application -> cameras")
	require.Contains(t, output, "src/application/bootstrapper.cc")
	require.Empty(t, stderr.String())
}
