package cli

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

// initOptions хранит CLI-опции команды init.
//
// По умолчанию команда печатает сгенерированный .patchcourt.yaml в stdout.
// С флагом --write команда записывает конфиг в файл.
type initOptions struct {
	strict     bool
	preset     string
	write      bool
	force      bool
	outputPath string
}

// newInitCommand создает Cobra-команду init.
//
// Команда init анализирует структуру проекта и генерирует стартовый
// .patchcourt.yaml. Результат можно вывести в stdout или записать в файл.
func (r *Runner) newInitCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts initOptions

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Generate initial .patchcourt.yaml for a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := optionalRootArg(args)

			result, err := r.newApp(rootOpts).RunInit(ctx, app.InitRequest{
				Root:       root,
				Strict:     opts.strict,
				Preset:     opts.preset,
				Write:      opts.write,
				Force:      opts.force,
				OutputPath: opts.outputPath,
			})
			if err != nil {
				return err
			}

			if result.Written {
				_, err = fmt.Fprintf(r.stdout, "PatchCourt init\n\nConfig written: %s\n", result.ConfigPath)
				return err
			}

			_, err = fmt.Fprint(r.stdout, result.ConfigYAML)
			return err
		},
	}

	cmd.Flags().BoolVar(&opts.strict, "strict", false, "generate strict config without inferred may_depend_on baseline")
	cmd.Flags().StringVar(&opts.preset, "preset", "", "init preset: auto, go-clean")
	cmd.Flags().BoolVar(&opts.write, "write", false, "write generated config to .patchcourt.yaml")
	cmd.Flags().BoolVar(&opts.force, "force", false, "overwrite existing config when used with --write")
	cmd.Flags().StringVar(&opts.outputPath, "out", "", "output config path when used with --write")

	return cmd
}
