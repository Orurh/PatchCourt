package main

import (
	"context"
	"fmt"
	"os"

	"github.com/orurh/patchcourt/internal/adapter/cli"
)

func main() {
	runner := cli.NewRunner(os.Stdout, os.Stderr)

	if err := runner.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "patchcourt failed: %v\n", err)
		os.Exit(1)
	}
}
