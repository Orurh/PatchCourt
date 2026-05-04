package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/output/report"
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

			checkReport := app.BuildCheckReport(result)

			artifacts, err := report.WriteCheckArtifacts(report.CheckArtifactsInput{
				OutDir:     result.OutDir,
				Project:    result.Project,
				LayerGraph: result.LayerGraph,
				Report:     checkReport,
			})
			if err != nil {
				return err
			}

			result.Artifacts = convertCheckArtifacts(artifacts)
			checkReport = app.BuildCheckReport(result)

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

func convertCheckArtifacts(artifacts []report.CheckArtifact) []app.CheckArtifact {
	result := make([]app.CheckArtifact, 0, len(artifacts))

	for _, artifact := range artifacts {
		result = append(result, app.CheckArtifact{
			Name: artifact.Name,
			Path: artifact.Path,
		})
	}

	return result
}
