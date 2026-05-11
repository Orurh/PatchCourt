package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	DataDir   string
	Root      string
	Workspace string
	Addr      string
	ViewerDir string
	Stderr    io.Writer
}

func Serve(ctx context.Context, opts Options) error {
	if err := normalizeServeOptions(&opts); err != nil {
		return err
	}

	mux := http.NewServeMux()
	if err := registerServeRoutes(mux, opts); err != nil {
		return err
	}

	server := &http.Server{
		Addr:              opts.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", opts.Addr, err)
	}

	logServeStart(opts, listener)
	return runHTTPServer(ctx, server, listener)
}

func normalizeServeOptions(opts *Options) error {
	if opts.DataDir == "" && opts.Root == "" {
		return fmt.Errorf("either bundle data directory or project root is required")
	}

	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8787"
	}

	return nil
}

func registerServeRoutes(mux *http.ServeMux, opts Options) error {
	if err := registerBundleDataRoutes(mux, opts.DataDir); err != nil {
		return err
	}

	if err := registerProjectRoutes(mux, opts); err != nil {
		return err
	}

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBytes(w, http.StatusOK, []byte(`{"status":"ok"}`+"\n"))
	})

	if opts.ViewerDir != "" {
		return registerViewerRoutes(mux, opts.ViewerDir)
	}

	registerAPIRootRoute(mux)
	return nil
}

func registerBundleDataRoutes(mux *http.ServeMux, dataDir string) error {
	if dataDir == "" {
		return nil
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		return fmt.Errorf("read bundle data directory %s: %w", dataDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundle data path is not a directory: %s", dataDir)
	}

	registerJSONFile(mux, "/api/manifest", dataDir, "manifest.json")
	registerJSONFile(mux, "/api/review", dataDir, "review.json")
	registerJSONFile(mux, "/api/project/before", dataDir, "project-before.json")
	registerJSONFile(mux, "/api/project/after", dataDir, "project-after.json")
	registerJSONFile(mux, "/api/graph", dataDir, "graph.json")
	registerJSONFile(mux, "/api/runtime", dataDir, "runtime.json")
	registerJSONFile(mux, "/api/tree", dataDir, "tree.json")
	registerJSONFile(mux, "/api/findings", dataDir, "findings.json")
	registerJSONFile(mux, "/api/contracts", dataDir, "contracts.json")
	registerJSONFile(mux, "/api/dependencies", dataDir, "dependencies.json")

	return nil
}

func registerProjectRoutes(mux *http.ServeMux, opts Options) error {
	if opts.Root == "" {
		return nil
	}

	info, err := os.Stat(opts.Root)
	if err != nil {
		return fmt.Errorf("read project root %s: %w", opts.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project root is not a directory: %s", opts.Root)
	}

	registerGitRoutes(mux, opts.Root)
	registerReviewRoutes(mux, opts)
	return nil
}

func registerAPIRootRoute(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		writeJSONBytes(w, http.StatusOK, []byte(`{
  "name": "PatchCourt bundle API",
  "endpoints": [
    "/api/health",
    "/api/manifest",
    "/api/review",
    "/api/project/before",
    "/api/project/after",
    "/api/graph",
    "/api/runtime",
    "/api/tree",
    "/api/findings",
    "/api/contracts",
    "/api/dependencies",
    "/api/git/status",
    "/api/git/branches",
    "/api/git/refs",
    "/api/git/graph",
    "/api/git/commits",
    "/api/reviews",
    "/api/reviews/latest/manifest",
    "/api/reviews/latest/review",
    "/api/reviews/latest/graph",
    "/api/reviews/latest/runtime",
    "/api/reviews/latest/tree",
    "/api/reviews/latest/findings",
    "/api/reviews/latest/contracts",
    "/api/reviews/latest/dependencies"
  ]
}`+"\n"))
	})
}

func logServeStart(opts Options, listener net.Listener) {
	if opts.Stderr == nil {
		return
	}

	fmt.Fprintf(opts.Stderr, "PatchCourt bundle API listening on http://%s\n", listener.Addr().String())
	if opts.ViewerDir != "" {
		fmt.Fprintf(opts.Stderr, "Serving viewer from: %s\n", opts.ViewerDir)
	}
	if opts.DataDir != "" {
		fmt.Fprintf(opts.Stderr, "Serving bundle data from: %s\n", opts.DataDir)
	}
}

func runHTTPServer(ctx context.Context, server *http.Server, listener net.Listener) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown bundle API: %w", err)
		}

		return ctx.Err()

	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serve bundle API: %w", err)
	}
}

func registerViewerRoutes(mux *http.ServeMux, viewerDir string) error {
	info, err := os.Stat(viewerDir)
	if err != nil {
		return fmt.Errorf("read viewer directory %s: %w", viewerDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("viewer path is not a directory: %s", viewerDir)
	}

	indexPath := filepath.Join(viewerDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return fmt.Errorf("viewer index.html not found in %s: %w", viewerDir, err)
	}

	fileServer := http.FileServer(http.Dir(viewerDir))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." || cleanPath == "" {
			http.ServeFile(w, r, indexPath)
			return
		}

		fullPath := filepath.Join(viewerDir, cleanPath)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexPath)
	})

	return nil
}

func registerJSONFile(mux *http.ServeMux, route string, dir string, filename string) {
	mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		data, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			http.Error(w, fmt.Sprintf("read %s: %v", filename, err), http.StatusNotFound)
			return
		}

		writeJSONBytes(w, http.StatusOK, data)
	})
}

func writeJSONBytes(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
