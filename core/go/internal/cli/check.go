package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/output/report"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	configPath string
	outDir     string
	format     string
}

func (r *Runner) newCheckCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts checkOptions

	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "Run scan and graph, write standard artifacts, and print a short summary",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := optionalRootArg(args)

			result, err := r.newApp(rootOpts).RunCheck(ctx, app.CheckRequest{
				Root:       root,
				ConfigPath: opts.configPath,
				OutDir:     opts.outDir,
			})
			if err != nil {
				return err
			}

			artifacts, err := r.writeCheckArtifacts(result)
			if err != nil {
				return err
			}

			result.Artifacts = artifacts

			checkReport := app.BuildCheckReport(result)

			switch strings.ToLower(opts.format) {
			case "", "text":
				report.WriteCheckReportText(r.stdout, checkReport)
				return nil
			case "json":
				encoder := json.NewEncoder(r.stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(checkReport)
			default:
				return fmt.Errorf("unsupported check format %q", opts.format)
			}
		},
	}

	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.outDir, "out", "", "output directory for generated artifacts")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text, json")

	return cmd
}

func (r *Runner) writeCheckArtifacts(result *app.CheckResult) ([]app.CheckArtifact, error) {
	if result == nil {
		return nil, fmt.Errorf("check result is nil")
	}

	if err := os.MkdirAll(result.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("create check output dir: %w", err)
	}

	artifacts := make([]app.CheckArtifact, 0, 6)

	writeArtifact := func(name string, filename string, write func(*os.File) error) error {
		path := filepath.Join(result.OutDir, filename)

		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create artifact %s: %w", path, err)
		}

		writeErr := write(file)
		closeErr := file.Close()

		if writeErr != nil {
			return fmt.Errorf("write artifact %s: %w", path, writeErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close artifact %s: %w", path, closeErr)
		}

		artifacts = append(artifacts, app.CheckArtifact{
			Name: name,
			Path: path,
		})
		return nil
	}

	if err := writeArtifact("project model", "project-model.json", func(file *os.File) error {
		return writeJSON(file, result.Project)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("scan report", "scan.md", func(file *os.File) error {
		report.WriteScanMarkdown(file, result.Project)
		return nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph json", "layer-graph.json", func(file *os.File) error {
		return writeJSON(file, result.LayerGraph)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph dot", "layer-graph.dot", func(file *os.File) error {
		report.WriteLayerGraphDOT(file, result.LayerGraph)
		return nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph mermaid", "layer-graph.mmd", func(file *os.File) error {
		report.WriteLayerGraphMermaid(file, result.LayerGraph)
		return nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("html report", "report.html", func(file *os.File) error {
		checkReport := app.BuildCheckReport(result)

		return report.WriteCheckHTML(file, report.CheckHTMLInput{
			Report:     checkReport,
			Project:    result.Project,
			LayerGraph: result.LayerGraph,
		})
	}); err != nil {
		return nil, err
	}

	return artifacts, nil
}
