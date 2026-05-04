package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/orurh/patchcourt/internal/render/llmpack"
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

	sinceLast   string
	updateState bool

	gitRoot  string
	baseRef  string
	headRef  string
	worktree bool

	llmPack     bool
	llmPackPath string

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
				BeforePath:    opts.beforePath,
				AfterPath:     opts.afterPath,
				BeforeRoot:    opts.beforeRoot,
				AfterRoot:     opts.afterRoot,
				ConfigPath:    opts.configPath,
				SinceLastRoot: opts.sinceLast,
				UpdateState:   opts.updateState,
				GitRoot:       opts.gitRoot,
				BaseRef:       opts.baseRef,
				HeadRef:       opts.headRef,
				Worktree:      opts.worktree,
			})
			if err != nil {
				return err
			}

			if opts.llmPack {
				if err := r.writeReviewLLMPack(opts, result); err != nil {
					return err
				}
			}

			return r.renderReviewResult(format, app.ReviewRequest{
				BeforePath:    opts.beforePath,
				AfterPath:     opts.afterPath,
				BeforeRoot:    opts.beforeRoot,
				AfterRoot:     opts.afterRoot,
				ConfigPath:    opts.configPath,
				SinceLastRoot: opts.sinceLast,
				UpdateState:   opts.updateState,
				GitRoot:       opts.gitRoot,
				BaseRef:       opts.baseRef,
				HeadRef:       opts.headRef,
				Worktree:      opts.worktree,
			}, result)
		},
	}

	cmd.Flags().StringVar(&opts.beforePath, "before", "", "path to before project model JSON")
	cmd.Flags().StringVar(&opts.afterPath, "after", "", "path to after project model JSON")
	cmd.Flags().StringVar(&opts.beforeRoot, "before-root", "", "path to before project root")
	cmd.Flags().StringVar(&opts.afterRoot, "after-root", "", "path to after project root")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.sinceLast, "since-last", "", "compare saved .patchcourt/state/latest with current project root")
	cmd.Flags().BoolVar(&opts.updateState, "update-state", false, "save current project model as .patchcourt/state/latest after successful review")
	cmd.Flags().StringVar(&opts.gitRoot, "root", ".", "git repository root or any path inside it for --base/--head review")
	cmd.Flags().StringVar(&opts.baseRef, "base", "", "base git ref for review, for example main or origin/main")
	cmd.Flags().StringVar(&opts.headRef, "head", "", "head git ref for review, for example HEAD")
	cmd.Flags().BoolVar(&opts.worktree, "worktree", false, "compare --base git ref with current working tree")
	cmd.Flags().BoolVar(&opts.llmPack, "llm-pack", false, "write deterministic LLM review context pack")
	cmd.Flags().StringVar(&opts.llmPackPath, "llm-pack-out", "", "path to write LLM review context pack")
	cmd.Flags().StringVar(&opts.format, "format", string(app.ReviewFormatText), "output format: text, json, markdown")

	return cmd
}

func (r *Runner) writeReviewLLMPack(opts reviewOptions, result *app.ReviewResult) error {
	if result == nil {
		return fmt.Errorf("review result is nil")
	}

	outPath := opts.llmPackPath
	if outPath == "" {
		root := opts.gitRoot
		if root == "" {
			root = opts.afterRoot
		}
		if root == "" {
			root = "."
		}

		outPath = filepath.Join(root, ".patchcourt", "review-context.md")
	}

	if err := llmpack.WriteReviewContextFile(outPath, llmpack.ReviewContextInput{
		Result:   *result,
		MaxItems: 10,
	}); err != nil {
		return err
	}

	if r.stderr != nil {
		fmt.Fprintf(r.stderr, "LLM context pack written: %s\n", outPath)
	}

	return nil
}
