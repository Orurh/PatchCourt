package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/app"
	"github.com/spf13/cobra"
)

// graphOptions хранит CLI-опции команды graph.
//
// Эти значения заполняются Cobra на основе флагов команды:
//
//	--config
//	--format
//
// Опции используются только CLI-адаптером и не являются частью
// application-слоя PatchCourt.
type graphOptions struct {
	configPath string
	format     string
}

// newGraphCommand создает Cobra-команду graph.
//
// Команда graph строит граф проекта для указанного пути.
// Если путь не передан, используется текущая директория.
//
// Команда не строит граф напрямую. Она преобразует CLI-аргументы
// в app.GraphRequest, вызывает application-слой через Application interface,
// а затем выводит готовый граф в выбранном формате.
func (r *Runner) newGraphCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts graphOptions

	cmd := &cobra.Command{
		Use:   "graph [path]",
		Short: "Build a project graph",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := optionalRootArg(args)

			format := app.GraphFormat(opts.format)

			result, err := r.newApp(rootOpts).RunGraph(ctx, app.GraphRequest{
				Root:       root,
				ConfigPath: opts.configPath,
			})
			if err != nil {
				return err
			}

			return r.renderGraphResult(format, result)
		},
	}

	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.format, "format", string(app.GraphFormatMermaid), "output format: mermaid, dot, json")

	return cmd
}
