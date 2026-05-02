package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

type reviewOptions struct {
	beforePath string
	afterPath  string
	format     string
}

func (r *Runner) newReviewCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts reviewOptions

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review contract changes between two project model JSON snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := app.ReviewFormat(opts.format)

			result, err := r.newApp(rootOpts).RunReview(ctx, app.ReviewRequest{
				BeforePath: opts.beforePath,
				AfterPath:  opts.afterPath,
			})
			if err != nil {
				return err
			}

			return r.renderReviewResult(format, result)
		},
	}

	cmd.Flags().StringVar(&opts.beforePath, "before", "", "path to before project model JSON")
	cmd.Flags().StringVar(&opts.afterPath, "after", "", "path to after project model JSON")
	cmd.Flags().StringVar(&opts.format, "format", string(app.ReviewFormatText), "output format: text, json")

	_ = cmd.MarkFlagRequired("before")
	_ = cmd.MarkFlagRequired("after")

	return cmd
}
