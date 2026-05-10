package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	bundleserve "github.com/orurh/patchcourt/internal/serve/bundle"
	"github.com/spf13/cobra"
)

type openOptions struct {
	root      string
	workspace string
	addr      string
	viewerDir string
	noBrowser bool

	base       string
	head       string
	worktree   bool
	configPath string
	reviewNow  bool
}

func (r *Runner) newOpenCommand(ctx context.Context, rootOpts *rootOptions) *cobra.Command {
	var opts openOptions

	cmd := &cobra.Command{
		Use:   "open [project-root]",
		Short: "Start PatchCourt viewer/API for a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.root = args[0]
			}
			if strings.TrimSpace(opts.root) == "" {
				opts.root = "."
			}

			return r.runOpen(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&opts.addr, "addr", "127.0.0.1:8787", "address for the local PatchCourt server")
	cmd.Flags().StringVar(&opts.workspace, "workspace", "", "directory for generated review bundles")
	cmd.Flags().StringVar(&opts.viewerDir, "viewer-dir", "", "directory with built PatchCourt viewer assets")
	cmd.Flags().BoolVar(&opts.noBrowser, "no-browser", false, "do not open browser automatically")

	cmd.Flags().StringVar(&opts.base, "base", "HEAD", "base git ref for initial review")
	cmd.Flags().StringVar(&opts.head, "head", "", "head git ref for initial review")
	cmd.Flags().BoolVar(&opts.worktree, "worktree", true, "include worktree changes in initial review")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .patchcourt.yaml")
	cmd.Flags().BoolVar(&opts.reviewNow, "review-now", false, "create an initial review after server starts")

	return cmd
}

func (r *Runner) runOpen(ctx context.Context, opts openOptions) error {
	root, err := filepath.Abs(opts.root)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}

	if info, err := os.Stat(root); err != nil {
		return fmt.Errorf("read project root %s: %w", root, err)
	} else if !info.IsDir() {
		return fmt.Errorf("project root is not a directory: %s", root)
	}

	viewerDir := resolveViewerDir(opts.viewerDir)
	baseURL := serverBaseURL(opts.addr)

	if viewerDir == "" && r.stderr != nil {
		fmt.Fprintln(r.stderr, "PatchCourt viewer assets were not found.")
		fmt.Fprintln(r.stderr, "Run `cd ../../web/viewer && npm run build`, or pass --viewer-dir.")
		fmt.Fprintln(r.stderr, "Starting API server only.")
	}

	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- bundleserve.Serve(serverCtx, bundleserve.Options{
			Root:      root,
			Workspace: opts.workspace,
			Addr:      opts.addr,
			ViewerDir: viewerDir,
			Stderr:    r.stderr,
		})
	}()

	if err := waitForHealth(ctx, baseURL, errCh); err != nil {
		cancel()
		return err
	}

	if r.stderr != nil {
		fmt.Fprintf(r.stderr, "PatchCourt project: %s\n", root)
		fmt.Fprintf(r.stderr, "PatchCourt URL:     %s\n", baseURL)
	}

	if opts.reviewNow {
		review, err := createInitialReview(ctx, baseURL, opts)
		if err != nil {
			if r.stderr != nil {
				fmt.Fprintf(r.stderr, "Initial review failed: %v\n", err)
			}
		} else if r.stderr != nil {
			fmt.Fprintf(r.stderr, "Initial review created: %s\n", review.ID)
			fmt.Fprintf(r.stderr, "Review bundle:          %s\n", review.BundleDir)
			fmt.Fprintf(r.stderr, "Latest review API:      %s/api/reviews/latest/review\n", baseURL)
		}
	}

	if !opts.noBrowser {
		if err := openBrowser(ctx, baseURL); err != nil && r.stderr != nil {
			fmt.Fprintf(r.stderr, "Open browser failed: %v\n", err)
		}
	}

	if r.stderr != nil {
		fmt.Fprintln(r.stderr, "Press Ctrl+C to stop PatchCourt.")
	}

	return <-errCh
}

func resolveViewerDir(explicit string) string {
	if explicit != "" {
		if hasViewerIndex(explicit) {
			return explicit
		}
		return ""
	}

	candidates := []string{
		filepath.Join("..", "..", "web", "viewer", "dist"),
		filepath.Join("web", "viewer", "dist"),
		filepath.Join("..", "web", "viewer", "dist"),
	}

	for _, candidate := range candidates {
		if hasViewerIndex(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}

	return ""
}

func hasViewerIndex(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}

func serverBaseURL(addr string) string {
	if addr == "" {
		addr = "127.0.0.1:8787"
	}

	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}

	return "http://" + addr
}

func waitForHealth(ctx context.Context, baseURL string, errCh <-chan error) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
			return fmt.Errorf("server stopped before becoming healthy")

		case <-ctx.Done():
			return ctx.Err()

		case <-deadline:
			return fmt.Errorf("server did not become healthy at %s", baseURL)

		case <-ticker.C:
			resp, err := client.Get(baseURL + "/api/health")
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}
			}
		}
	}
}

func createInitialReview(ctx context.Context, baseURL string, opts openOptions) (*bundleserve.CreateReviewResponse, error) {
	req := bundleserve.CreateReviewRequest{
		Base:       opts.base,
		Head:       opts.head,
		Worktree:   opts.worktree,
		ConfigPath: opts.configPath,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal review request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/reviews", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create review request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("post review request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("review API returned %s: %s", resp.Status, message)
	}

	var review bundleserve.CreateReviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&review); err != nil {
		return nil, fmt.Errorf("decode review response: %w", err)
	}

	return &review, nil
}

func openBrowser(ctx context.Context, url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}

	return cmd.Start()
}
