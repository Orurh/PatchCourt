package cli

import (
	"context"

	bundleserve "github.com/orurh/patchcourt/internal/serve/bundle"
	"github.com/spf13/cobra"
)

type serveOptions struct {
	dataDir string
	addr    string
}

func (r *Runner) newServeCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts serveOptions

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve PatchCourt analysis bundle API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return bundleserve.Serve(ctx, bundleserve.Options{
				DataDir: opts.dataDir,
				Addr:    opts.addr,
				Stderr:  r.stderr,
			})
		},
	}

	cmd.Flags().StringVar(&opts.dataDir, "data", "", "path to PatchCourt analysis bundle directory")
	cmd.Flags().StringVar(&opts.addr, "addr", "127.0.0.1:8787", "address for the bundle API server")

	_ = cmd.MarkFlagRequired("data")

	return cmd
}
