package cli

import (
	"context"
	"io"

	"github.com/orurh/patchcourt/internal/usecase"
)

// Runner запускает CLI-адаптер PatchCourt.
//
// Runner хранит потоки стандартного вывода и ошибок, создает корневую
// Cobra-команду и передает ей аргументы командной строки.
//
// Runner не содержит бизнес-логики анализа проекта. Его задача — связать
// CLI-вызов с application-слоем и вывести результат команды в нужный поток.
type Runner struct {
	stdout     io.Writer
	stderr     io.Writer
	appFactory AppFactory
}

// NewRunner создает новый CLI runner.
//
// stdout используется только для результата команды: text, JSON, Markdown,
// DOT и других пользовательских форматов.
//
// stderr используется для диагностического вывода: ошибок, verbose/debug-логов
// и служебных сообщений.
func NewRunner(stdout io.Writer, stderr io.Writer) *Runner {
	return &Runner{
		stdout: stdout,
		stderr: stderr,
		appFactory: func(opts AppFactoryOptions) Application {
			return usecase.NewWithOptions(usecase.FactoryOptions{
				Verbose: opts.Verbose,
				Stderr:  stderr,
			})
		},
	}
}

// WithAppFactory подменяет factory application-слоя.
//
// Метод нужен для тестов CLI и для будущих сценариев, где CLI должен работать
// с другой реализацией usecase-слоя.
func (r *Runner) WithAppFactory(factory AppFactory) *Runner {
	if factory != nil {
		r.appFactory = factory
	}

	return r
}

// Run выполняет CLI-команду PatchCourt.
//
// args ожидаются без имени бинарника, то есть обычно это os.Args[1:].
// Метод создает корневую Cobra-команду, подключает stdout/stderr,
// передает аргументы и запускает выполнение команды.
//
// Конкретные команды внутри Cobra вызывают application-слой через Application interface.
func (r *Runner) Run(ctx context.Context, args []string) error {
	rootCmd := r.newRootCommand(ctx)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(r.stdout)
	rootCmd.SetErr(r.stderr)

	return rootCmd.Execute()
}
