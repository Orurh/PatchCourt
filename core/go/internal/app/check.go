package app

import (
	"context"
	"fmt"
	"path/filepath"

	graphmodel "github.com/orurh/patchcourt/internal/analysis/graph"
	"github.com/orurh/patchcourt/internal/config"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
)

type CheckRequest struct {
	Root       string
	ConfigPath string
	OutDir     string
}

type CheckArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type CheckResult struct {
	Root       string                `json:"root"`
	ConfigPath string                `json:"config_path,omitempty"`
	OutDir     string                `json:"out_dir"`
	Project    *model.ProjectModel   `json:"project,omitempty"`
	Config     *config.Config        `json:"config,omitempty"`
	LayerGraph graphmodel.LayerGraph `json:"layer_graph"`
	Summary    model.ScanSummary     `json:"summary"`
	Artifacts  []CheckArtifact       `json:"artifacts"`
}

func (a *App) RunCheck(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("check canceled before start: %w", err)
	}

	root := req.Root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	outDir := req.OutDir
	if outDir == "" {
		outDir = filepath.Join(absRoot, ".patchcourt", "out")
	}

	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(absRoot, outDir)
	}

	a.logger.Debug(
		"running check",
		logx.String("root", absRoot),
		logx.String("config_path", req.ConfigPath),
		logx.String("out_dir", outDir),
	)

	scanResult, err := a.RunScan(ctx, ScanRequest{
		Root:       absRoot,
		ConfigPath: req.ConfigPath,
	})
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("check canceled after scan: %w", err)
	}

	layerGraph := graphmodel.BuildLayerGraph(scanResult.Project, scanResult.Config)
	summary := model.BuildScanSummary(scanResult.Project)

	a.logger.Debug(
		"check completed",
		logx.Int("findings", len(scanResult.Project.Findings)),
		logx.Int("graph_nodes", len(layerGraph.Nodes)),
		logx.Int("graph_edges", len(layerGraph.Edges)),
	)

	return &CheckResult{
		Root:       absRoot,
		ConfigPath: req.ConfigPath,
		OutDir:     outDir,
		Project:    scanResult.Project,
		Config:     scanResult.Config,
		LayerGraph: layerGraph,
		Summary:    summary,
	}, nil
}

func (r *CheckResult) ArtifactPathByName(name string) string {
	if r == nil {
		return ""
	}

	for _, artifact := range r.Artifacts {
		if artifact.Name == name {
			return artifact.Path
		}
	}

	return ""
}
