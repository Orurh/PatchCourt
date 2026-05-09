package bundle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/orurh/patchcourt/internal/render/reviewbundle"
	"github.com/orurh/patchcourt/internal/reportmodel"
	"github.com/orurh/patchcourt/internal/usecase"
)

type CreateReviewRequest struct {
	Base       string `json:"base"`
	Head       string `json:"head,omitempty"`
	Worktree   bool   `json:"worktree,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
}

type CreateReviewResponse struct {
	ID        string            `json:"id"`
	BundleDir string            `json:"bundle_dir"`
	Artifacts map[string]string `json:"artifacts"`
	Risk      any               `json:"risk"`
	Summary   any               `json:"summary"`
}

func registerReviewRoutes(mux *http.ServeMux, opts Options) {
	mux.HandleFunc("/api/reviews", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if opts.Root == "" {
			http.Error(w, "project root is required for review generation", http.StatusBadRequest)
			return
		}

		workspace, err := resolveWorkspaceOutsideRoot(opts.Root, opts.Workspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req CreateReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("decode review request: %v", err), http.StatusBadRequest)
			return
		}

		if err := validateCreateReviewRequest(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := runReview(r, opts.Root, req)
		if err != nil {
			http.Error(w, fmt.Sprintf("run review: %v", err), http.StatusInternalServerError)
			return
		}

		reviewID := time.Now().UTC().Format("20060102T150405Z")
		reviewDir := filepath.Join(workspace, "reviews", reviewID)
		latestDir := filepath.Join(workspace, "latest")

		if err := reviewbundle.Write(reviewDir, *result); err != nil {
			http.Error(w, fmt.Sprintf("write review bundle: %v", err), http.StatusInternalServerError)
			return
		}

		if err := replaceLatestBundle(latestDir, reviewDir, *result); err != nil {
			http.Error(w, fmt.Sprintf("write latest review bundle: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSONValue(w, http.StatusOK, CreateReviewResponse{
			ID:        reviewID,
			BundleDir: latestDir,
			Artifacts: latestReviewArtifacts(),
			Risk:      result.Risk,
			Summary:   result.Summary,
		})
	})

	registerLatestReviewArtifact(mux, "/api/reviews/latest/manifest", opts, "manifest.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/review", opts, "review.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/project/before", opts, "project-before.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/project/after", opts, "project-after.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/graph", opts, "graph.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/runtime", opts, "runtime.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/tree", opts, "tree.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/findings", opts, "findings.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/contracts", opts, "contracts.json")
	registerLatestReviewArtifact(mux, "/api/reviews/latest/dependencies", opts, "dependencies.json")
}

func validateCreateReviewRequest(req *CreateReviewRequest) error {
	if req == nil {
		return fmt.Errorf("review request is required")
	}

	req.Base = strings.TrimSpace(req.Base)
	req.Head = strings.TrimSpace(req.Head)
	req.ConfigPath = strings.TrimSpace(req.ConfigPath)

	if req.Base == "" {
		return fmt.Errorf("base is required")
	}

	if req.Worktree && req.Head != "" {
		return fmt.Errorf("head cannot be set when worktree=true")
	}

	if !req.Worktree && req.Head == "" {
		req.Head = "HEAD"
	}

	return nil
}

func runReview(r *http.Request, root string, req CreateReviewRequest) (*reportmodel.ReviewResult, error) {
	roots, err := materializeReviewRoots(r.Context(), root, req.Base, req.Head, req.Worktree)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(roots.TempDir)

	configPath := req.ConfigPath
	if configPath == "" {
		candidate := filepath.Join(root, ".patchcourt.yaml")
		if _, err := os.Stat(candidate); err == nil {
			configPath = candidate
		}
	}

	app := usecase.NewWithOptions(usecase.FactoryOptions{})

	return app.RunReview(r.Context(), usecase.ReviewRequest{
		BeforeRoot: roots.BeforeRoot,
		AfterRoot:  roots.AfterRoot,
		ConfigPath: configPath,
	})
}

func replaceLatestBundle(latestDir string, reviewDir string, result reportmodel.ReviewResult) error {
	if latestDir == "" {
		return fmt.Errorf("latest bundle directory is required")
	}

	if err := os.RemoveAll(latestDir); err != nil {
		return fmt.Errorf("remove old latest bundle: %w", err)
	}

	if err := reviewbundle.Write(latestDir, result); err != nil {
		return err
	}

	_ = reviewDir
	return nil
}

func registerLatestReviewArtifact(mux *http.ServeMux, route string, opts Options, filename string) {
	mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		workspace, err := resolveWorkspaceOutsideRoot(opts.Root, opts.Workspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		latestDir := filepath.Join(workspace, "latest")
		data, err := os.ReadFile(filepath.Join(latestDir, filename))
		if err != nil {
			http.Error(w, fmt.Sprintf("read latest %s: %v", filename, err), http.StatusNotFound)
			return
		}

		writeJSONBytes(w, http.StatusOK, data)
	})
}

func latestReviewArtifacts() map[string]string {
	return map[string]string{
		"manifest":       "manifest.json",
		"review":         "review.json",
		"project_before": "project-before.json",
		"project_after":  "project-after.json",
		"graph":          "graph.json",
		"runtime":        "runtime.json",
		"tree":           "tree.json",
		"findings":       "findings.json",
		"contracts":      "contracts.json",
		"dependencies":   "dependencies.json",
		"llm_context":    "review-context.md",
		"sarif":          "patchcourt.sarif",
	}
}
