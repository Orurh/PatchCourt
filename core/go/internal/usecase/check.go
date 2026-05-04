package usecase

import (
	"context"
	"fmt"
	"path/filepath"

	graphmodel "github.com/orurh/patchcourt/internal/analyzer/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/state"
)

type CheckArtifact = reportmodel.CheckArtifact
type CheckResult = reportmodel.CheckResult

type CheckRequest struct {
	Root       string
	ConfigPath string
	OutDir     string
	SaveState  bool
}

type CheckService struct {
	Scan   ScanService
	Logger logx.Logger
}

func NewCheckService(scan ScanService, logger logx.Logger) CheckService {
	return CheckService{
		Scan:   scan,
		Logger: logger,
	}
}

func (s CheckService) Run(ctx context.Context, req CheckRequest) (*CheckResult, error) {
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

	if s.Logger != nil {
		s.Logger.Debug(
			"running check",
			logx.String("root", absRoot),
			logx.String("config_path", req.ConfigPath),
			logx.String("out_dir", outDir),
		)
	}

	scanResult, err := s.Scan.Run(ctx, ScanRequest{
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

	statePath := ""

	if req.SaveState {
		if _, err := state.SaveState(state.SaveStateOptions{
			Root:       absRoot,
			ConfigPath: req.ConfigPath,
			Project:    scanResult.Project,
		}); err != nil {
			return nil, fmt.Errorf("save state: %w", err)
		}

		statePath = filepath.Join(state.StateDir(absRoot, state.LatestStateName), "project-model.json")
	}

	if s.Logger != nil {
		s.Logger.Debug(
			"check completed",
			logx.Int("findings", len(scanResult.Project.Findings)),
			logx.Int("graph_nodes", len(layerGraph.Nodes)),
			logx.Int("graph_edges", len(layerGraph.Edges)),
		)
	}

	return &CheckResult{
		Root:       absRoot,
		ConfigPath: req.ConfigPath,
		OutDir:     outDir,
		StatePath:  statePath,
		Project:    scanResult.Project,
		Config:     scanResult.Config,
		LayerGraph: layerGraph,
		Summary:    summary,
	}, nil
}

func (a *App) RunCheck(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	projects := NewProjectBuilder(a.analysis)
	scan := NewScanService(projects)

	return NewCheckService(scan, a.logger).Run(ctx, req)
}
