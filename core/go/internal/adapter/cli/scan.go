package cli

import (
	"context"

	"github.com/orurh/patchcourt/internal/usecase"
	"github.com/spf13/cobra"
)

// scanOptions хранит CLI-опции команды scan.
//
// Эти значения заполняются Cobra на основе флагов команды:
//
//	--config
//	--format
//	--json
//
// Опции используются только CLI-адаптером и не являются частью
// application-слоя PatchCourt.
type scanOptions struct {
	configPath string
	format     string
	jsonOutput bool
}

// newScanCommand создает Cobra-команду scan.
//
// Команда scan строит модель проекта для указанного пути.
// Если путь не передан, используется текущая директория.
//
// Команда не выполняет анализ напрямую. Она преобразует CLI-аргументы
// в usecase.ScanRequest, вызывает application-слой через Application interface,
// а затем выводит результат в выбранном формате.
func (r *Runner) newScanCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts scanOptions

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a project and build a project model",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := optionalRootArg(args)

			format := usecase.ScanFormat(opts.format)
			if opts.jsonOutput {
				format = usecase.ScanFormatJSON
			}

			result, err := r.newApp(rootOpts).RunScan(ctx, usecase.ScanRequest{
				Root:       root,
				ConfigPath: opts.configPath,
			})
			if err != nil {
				return err
			}

			return r.renderScanResult(format, result)
		},
	}

	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().StringVar(&opts.format, "format", string(usecase.ScanFormatText), "output format: text, markdown, json")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "shortcut for --format json")

	return cmd
}
