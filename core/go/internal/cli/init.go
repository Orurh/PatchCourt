package cli

import (
	"context"
	"fmt"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

// initOptions хранит CLI-опции команды init.
//
// Сейчас команда init только печатает сгенерированный .patchcourt.yaml в stdout.
// Запись файла будет добавлена отдельным флагом позже.
type initOptions struct {
	strict bool
}

// newInitCommand создает Cobra-команду init.
//
// Команда init анализирует структуру проекта и генерирует стартовый
// .patchcourt.yaml. Результат выводится в stdout, чтобы пользователь мог
// проверить конфиг перед записью в репозиторий.
func (r *Runner) newInitCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts initOptions

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Generate initial .patchcourt.yaml for a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := optionalRootArg(args)

			result, err := r.newApp(rootOpts).RunInit(ctx, app.InitRequest{
				Root:   root,
				Strict: opts.strict,
			})
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(r.stdout, result.ConfigYAML)
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.strict, "strict", false, "generate strict config without inferred may_depend_on baseline")

	return cmd
}
