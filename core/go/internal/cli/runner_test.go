package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/orurh/patchcourt/internal/analysis/findingdiff"
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

	initResult    *app.InitResult
	scanResult    *app.ScanResult
	graphResult   *app.GraphResult
	reviewResult  *app.ReviewResult
	explainResult *app.ExplainResult

	initErr    error
	scanErr    error
	graphErr   error
	reviewErr  error
	explainErr error
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
