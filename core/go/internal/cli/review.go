package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

// reviewOptions хранит CLI-опции команды review.
//
// Эти значения заполняются Cobra на основе флагов команды:
//
//	--before
//	--after
//	--before-root
//	--after-root
//	--config
//	--format
//
// Опции используются только CLI-адаптером и не являются частью
// application-слоя PatchCourt.
type reviewOptions struct {
	beforePath string
	afterPath  string

	beforeRoot string
	afterRoot  string
	configPath string

	format string
}

// newReviewCommand создает Cobra-команду review.
//
// Команда review сравнивает два project model snapshots или два project roots.
// Результат выводится в stdout в выбранном формате.
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

			return r.renderReviewResult(format, app.ReviewRequest{
				BeforePath: opts.beforePath,
				AfterPath:  opts.afterPath,
				BeforeRoot: opts.beforeRoot,
				AfterRoot:  opts.afterRoot,
				ConfigPath: opts.configPath,
			}, result)
		},
	}

	cmd.Flags().StringVar(&opts.beforePath, "before", "", "path to before project model JSON")
	cmd.Flags().StringVar(&opts.afterPath, "after", "", "path to after project model JSON")
	cmd.Flags().StringVar(&opts.beforeRoot, "before-root", "", "path to before project root")
	cmd.Flags().StringVar(&opts.afterRoot, "after-root", "", "path to after project root")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.format, "format", string(app.ReviewFormatText), "output format: text, json, markdown")

	return cmd
}
