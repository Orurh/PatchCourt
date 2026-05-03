package cli

import (
	"context"

	"github.com/spf13/cobra"
)

// rootOptions хранит глобальные CLI-опции.
//
// Эти опции доступны всем подкомандам PatchCourt через persistent flags
// корневой Cobra-команды.
type rootOptions struct {
	verbose bool
}

// newRootCommand создает корневую Cobra-команду PatchCourt.
//
// Корневая команда отвечает за глобальные флаги, регистрацию подкоманд
// и общую справочную информацию CLI.
//
// Метод не запускает анализ проекта напрямую. Конкретная работа выполняется
// внутри подкоманд scan, graph, review и других будущих команд.
func (r *Runner) newRootCommand(ctx context.Context) *cobra.Command {
	var opts rootOptions

	cmd := &cobra.Command{
		Use:   "patchcourt",
		Short: "PatchCourt analyzes Go/C++ projects and reviews architecture risks",
		Long:  "PatchCourt is a diff-aware architecture and risk analyzer for Go/C++ codebases.",
	}

	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose logs on stderr")

	cmd.AddCommand(r.newInitCommand(ctx, &opts))
	cmd.AddCommand(r.newScanCommand(ctx, &opts))
	cmd.AddCommand(r.newGraphCommand(ctx, &opts))
	cmd.AddCommand(r.newReviewCommand(ctx, &opts))
	cmd.AddCommand(r.newExplainCommand(ctx, &opts))
	cmd.AddCommand(r.newEdgeCommand(ctx, &opts))
	cmd.AddCommand(r.newCheckCommand(ctx, &opts))

	return cmd
}

// newApp создает application/usecase-слой для выполнения CLI-команды.
//
// Метод выбирает подходящий logger на основе глобальных CLI-опций.
// По умолчанию используется no-op logger, чтобы не загрязнять stdout.
// При включенном verbose-режиме используется structured logger,
// который пишет диагностический вывод в stderr.
func (r *Runner) newApp(opts *rootOptions) Application {
	return r.appFactory(AppFactoryOptions{
		Verbose: opts != nil && opts.verbose,
	})
}

//
// Логи пишутся только в stderr, чтобы не ломать машинно-читаемый stdout,
// например JSON, DOT или Markdown-вывод команд.
