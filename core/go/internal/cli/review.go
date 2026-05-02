package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

type reviewOptions struct {
	beforePath string
	afterPath  string

	beforeRoot string
	afterRoot  string
	configPath string

	format string
}

func (r *Runner) newReviewCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts reviewOptions

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review contract changes between two project model snapshots or project roots",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := app.ReviewFormat(opts.format)

			result, err := r.newApp(rootOpts).RunReview(ctx, app.ReviewRequest{
				BeforePath: opts.beforePath,
				AfterPath:  opts.afterPath,
				BeforeRoot: opts.beforeRoot,
				AfterRoot:  opts.afterRoot,
				ConfigPath: opts.configPath,
			})
			if err != nil {
				return err
			}

			return r.renderReviewResult(format, result)
		},
	}

	cmd.Flags().StringVar(&opts.beforePath, "before", "", "path to before project model JSON")
	cmd.Flags().StringVar(&opts.afterPath, "after", "", "path to after project model JSON")
	cmd.Flags().StringVar(&opts.beforeRoot, "before-root", "", "path to before project root")
	cmd.Flags().StringVar(&opts.afterRoot, "after-root", "", "path to after project root")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.format, "format", string(app.ReviewFormatText), "output format: text, json")

	return cmd
}
