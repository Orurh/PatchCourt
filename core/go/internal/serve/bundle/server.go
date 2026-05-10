package bundle

import (
	"context"
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
	if opts.DataDir == "" && opts.Root == "" {
		return fmt.Errorf("either bundle data directory or project root is required")
	}

	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8787"
	}

	mux := http.NewServeMux()

	if opts.DataDir != "" {
		info, err := os.Stat(opts.DataDir)
		if err != nil {
			return fmt.Errorf("read bundle data directory %s: %w", opts.DataDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("bundle data path is not a directory: %s", opts.DataDir)
		}

		registerJSONFile(mux, "/api/manifest", opts.DataDir, "manifest.json")
		registerJSONFile(mux, "/api/review", opts.DataDir, "review.json")
		registerJSONFile(mux, "/api/project/before", opts.DataDir, "project-before.json")
		registerJSONFile(mux, "/api/project/after", opts.DataDir, "project-after.json")
		registerJSONFile(mux, "/api/graph", opts.DataDir, "graph.json")
		registerJSONFile(mux, "/api/runtime", opts.DataDir, "runtime.json")
		registerJSONFile(mux, "/api/tree", opts.DataDir, "tree.json")
		registerJSONFile(mux, "/api/findings", opts.DataDir, "findings.json")
		registerJSONFile(mux, "/api/contracts", opts.DataDir, "contracts.json")
		registerJSONFile(mux, "/api/dependencies", opts.DataDir, "dependencies.json")
	}

	if opts.Root != "" {
		info, err := os.Stat(opts.Root)
		if err != nil {
			return fmt.Errorf("read project root %s: %w", opts.Root, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("project root is not a directory: %s", opts.Root)
		}

		registerGitRoutes(mux, opts.Root)
		registerReviewRoutes(mux, opts)
	}

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBytes(w, http.StatusOK, []byte(`{"status":"ok"}`+"\n"))
	})

	if opts.ViewerDir != "" {
		if err := registerViewerRoutes(mux, opts.ViewerDir); err != nil {
			return err
		}
	} else {
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
		if opts.ViewerDir != "" {
			fmt.Fprintf(opts.Stderr, "Serving viewer from: %s\n", opts.ViewerDir)
		}
		if opts.DataDir != "" {
			fmt.Fprintf(opts.Stderr, "Serving bundle data from: %s\n", opts.DataDir)
		}
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
