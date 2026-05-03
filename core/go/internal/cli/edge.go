package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

type edgeOptions struct {
	root       string
	configPath string
	modelPath  string
	format     string
	limit      int
}

func (r *Runner) newEdgeCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts edgeOptions

	cmd := &cobra.Command{
		Use:   "edge [path] FROM_LAYER TO_LAYER",
		Short: "Explain a layer edge by showing the include dependencies behind it",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := opts.root
			fromLayer := args[0]
			toLayer := args[1]

			if len(args) == 3 {
				root = args[0]
				fromLayer = args[1]
				toLayer = args[2]
			}

			result, err := r.newApp(rootOpts).RunEdge(ctx, app.EdgeRequest{
				Root:       root,
				ConfigPath: opts.configPath,
				ModelPath:  opts.modelPath,
				FromLayer:  fromLayer,
				ToLayer:    toLayer,
				Limit:      opts.limit,
			})
			if err != nil {
				return err
			}

			return r.renderEdgeResult(app.EdgeFormat(opts.format), result)
		},
	}

	cmd.Flags().StringVar(&opts.root, "root", ".", "project root to scan")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.modelPath, "model", "", "path to project model JSON produced by scan/check")
	cmd.Flags().StringVar(&opts.format, "format", string(app.EdgeFormatText), "output format: text, json")
	cmd.Flags().IntVar(&opts.limit, "limit", 50, "maximum number of dependencies to print")

	return cmd
}
