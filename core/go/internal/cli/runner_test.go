package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type fakeApplication struct {
	initReq  app.InitRequest
	scanReq  app.ScanRequest
	graphReq app.GraphRequest

	initResult  *app.InitResult
	scanResult  *app.ScanResult
	graphResult *app.GraphResult

	initErr  error
	scanErr  error
	graphErr error
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
