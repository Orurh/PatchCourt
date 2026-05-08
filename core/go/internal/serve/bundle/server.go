package bundle

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	DataDir string
	Addr    string
	Stderr  io.Writer
}

func Serve(ctx context.Context, opts Options) error {
	if opts.DataDir == "" {
		return fmt.Errorf("bundle data directory is required")
	}

	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8787"
	}

	info, err := os.Stat(opts.DataDir)
	if err != nil {
		return fmt.Errorf("read bundle data directory %s: %w", opts.DataDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundle data path is not a directory: %s", opts.DataDir)
	}

	mux := http.NewServeMux()

	registerJSONFile(mux, "/api/manifest", opts.DataDir, "manifest.json")
	registerJSONFile(mux, "/api/review", opts.DataDir, "review.json")
	registerJSONFile(mux, "/api/project/before", opts.DataDir, "project-before.json")
	registerJSONFile(mux, "/api/project/after", opts.DataDir, "project-after.json")
	registerJSONFile(mux, "/api/graph", opts.DataDir, "graph.json")
	registerJSONFile(mux, "/api/runtime", opts.DataDir, "runtime.json")
	registerJSONFile(mux, "/api/tree", opts.DataDir, "tree.json")

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBytes(w, http.StatusOK, []byte(`{"status":"ok"}`+"\n"))
	})

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
    "/api/tree"
  ]
}`+"\n"))
	})

	server := &http.Server{
		Addr:              opts.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", opts.Addr, err)
	}

	if opts.Stderr != nil {
		fmt.Fprintf(opts.Stderr, "PatchCourt bundle API listening on http://%s\n", listener.Addr().String())
		fmt.Fprintf(opts.Stderr, "Serving bundle data from: %s\n", opts.DataDir)
	}

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
		if err == nil || err == http.ErrServerClosed {
			return nil
		}

		return fmt.Errorf("serve bundle API: %w", err)
	}
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
