package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/output/report"
	"github.com/orurh/patchcourt/internal/platform/files"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	configPath string
	outDir     string
	format     string
	saveState  bool
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
				SaveState:  opts.saveState,
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
	cmd.Flags().BoolVar(&opts.saveState, "save-state", false, "save current project model as .patchcourt/state/latest")

	return cmd
}

func (r *Runner) writeCheckArtifacts(result *app.CheckResult) ([]app.CheckArtifact, error) {
	if result == nil {
		return nil, fmt.Errorf("check result is nil")
	}

	artifacts := make([]app.CheckArtifact, 0, 6)

	writeArtifact := func(name string, filename string, render func() ([]byte, error)) error {
		path := filepath.Join(result.OutDir, filename)

		data, err := render()
		if err != nil {
			return fmt.Errorf("render artifact %s: %w", path, err)
		}

		if err := files.WriteFileAtomic(path, data, 0o644); err != nil {
			return fmt.Errorf("write artifact %s: %w", path, err)
		}

		artifacts = append(artifacts, app.CheckArtifact{
			Name: name,
			Path: path,
		})
		return nil
	}

	if err := writeArtifact("project model", "project-model.json", func() ([]byte, error) {
		return encodeJSON(result.Project)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("scan report", "scan.md", func() ([]byte, error) {
		var buf bytes.Buffer
		report.WriteScanMarkdown(&buf, result.Project)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph json", "layer-graph.json", func() ([]byte, error) {
		return encodeJSON(result.LayerGraph)
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph dot", "layer-graph.dot", func() ([]byte, error) {
		var buf bytes.Buffer
		report.WriteLayerGraphDOT(&buf, result.LayerGraph)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("layer graph mermaid", "layer-graph.mmd", func() ([]byte, error) {
		var buf bytes.Buffer
		report.WriteLayerGraphMermaid(&buf, result.LayerGraph)
		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	if err := writeArtifact("html report", "report.html", func() ([]byte, error) {
		var buf bytes.Buffer
		checkReport := app.BuildCheckReport(result)

		if err := report.WriteCheckHTML(&buf, report.CheckHTMLInput{
			Report:     checkReport,
			Project:    result.Project,
			LayerGraph: result.LayerGraph,
		}); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}); err != nil {
		return nil, err
	}

	return artifacts, nil
}

func encodeJSON(value any) ([]byte, error) {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
