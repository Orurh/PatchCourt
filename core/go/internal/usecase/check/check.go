package check

import (
	"context"
	"fmt"
	"path/filepath"

	graphmodel "github.com/orurh/patchcourt/internal/analyzer/graph"
	"github.com/orurh/patchcourt/internal/model"
	"github.com/orurh/patchcourt/internal/platform/logx"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/state"
	scanusecase "github.com/orurh/patchcourt/internal/usecase/scan"
)

type Artifact = reportmodel.CheckArtifact
type Result = reportmodel.CheckResult

type Request struct {
	Root       string
	ConfigPath string
	OutDir     string
	SaveState  bool
}

type Service struct {
	Scan   scanusecase.Service
	Logger logx.Logger
}

func NewService(scan scanusecase.Service, logger logx.Logger) Service {
	if logger == nil {
		logger = logx.Nop()
	}

	return Service{
		Scan:   scan,
		Logger: logger,
	}
}

func (s Service) Run(ctx context.Context, req Request) (*Result, error) {
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

	s.Logger.Debug(
		"running check",
		logx.String("root", absRoot),
		logx.String("config_path", req.ConfigPath),
		logx.String("out_dir", outDir),
	)

	scanResult, err := s.Scan.Run(ctx, scanusecase.Request{
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

	s.Logger.Debug(
		"check completed",
		logx.Int("findings", len(scanResult.Project.Findings)),
		logx.Int("graph_nodes", len(layerGraph.Nodes)),
		logx.Int("graph_edges", len(layerGraph.Edges)),
	)

	return &Result{
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
