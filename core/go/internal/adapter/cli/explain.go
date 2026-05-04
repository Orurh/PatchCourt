package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/usecase"
	"github.com/spf13/cobra"
)

type explainOptions struct {
	root       string
	configPath string
	modelPath  string
	format     string
}

func (r *Runner) newExplainCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts explainOptions

	cmd := &cobra.Command{
		Use:   "explain FINDING_ID",
		Short: "Explain a specific finding",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := usecase.ExplainFormat(opts.format)

			result, err := r.newApp(rootOpts).RunExplain(ctx, usecase.ExplainRequest{
				FindingID:  args[0],
				Root:       opts.root,
				ConfigPath: opts.configPath,
				ModelPath:  opts.modelPath,
			})
			if err != nil {
				return err
			}

			return r.renderExplainResult(format, result)
		},
	}

	cmd.Flags().StringVar(&opts.root, "root", ".", "project root to scan")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.modelPath, "model", "", "path to project model JSON produced by scan --format json")
	cmd.Flags().StringVar(&opts.format, "format", string(usecase.ExplainFormatText), "output format: text, json")

	return cmd
}
