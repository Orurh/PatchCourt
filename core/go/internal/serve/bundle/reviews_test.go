package bundle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCreateReviewRequest_RequiresBase(t *testing.T) {
	req := CreateReviewRequest{
		Worktree: true,
	}

	err := validateCreateReviewRequest(&req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "base is required")
}

func TestValidateCreateReviewRequest_RejectsHeadWithWorktree(t *testing.T) {
	req := CreateReviewRequest{
		Base:     "main",
		Head:     "HEAD",
		Worktree: true,
	}

	err := validateCreateReviewRequest(&req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "head cannot be set when worktree=true")
}

func TestValidateCreateReviewRequest_DefaultsHeadWhenWorktreeDisabled(t *testing.T) {
	req := CreateReviewRequest{
		Base: "main",
	}

	err := validateCreateReviewRequest(&req)

	require.NoError(t, err)
	require.Equal(t, "main", req.Base)
	require.Equal(t, "HEAD", req.Head)
	require.False(t, req.Worktree)
}

func TestValidateCreateReviewRequest_TrimsRefsAndConfigPath(t *testing.T) {
	req := CreateReviewRequest{
		Base:       " main ",
		Head:       " feature/test ",
		ConfigPath: " .patchcourt.yaml ",
	}

	err := validateCreateReviewRequest(&req)

	require.NoError(t, err)
	require.Equal(t, "main", req.Base)
	require.Equal(t, "feature/test", req.Head)
	require.Equal(t, ".patchcourt.yaml", req.ConfigPath)
}

func TestValidateCreateReviewRequest_AllowsWorktreeWithoutHead(t *testing.T) {
	req := CreateReviewRequest{
		Base:     "HEAD",
		Worktree: true,
	}

	err := validateCreateReviewRequest(&req)

	require.NoError(t, err)
	require.Equal(t, "HEAD", req.Base)
	require.Empty(t, req.Head)
	require.True(t, req.Worktree)
}

func TestLatestReviewArtifacts_DoesNotExposeGeneratedHTML(t *testing.T) {
	artifacts := latestReviewArtifacts()

	require.NotContains(t, artifacts, "html")
	require.NotContains(t, artifacts, "review.html")
	require.Equal(t, "review.json", artifacts["review"])
	require.Equal(t, "manifest.json", artifacts["manifest"])
}

func TestRegisterLatestReviewArtifact_ReadsFromWorkspaceLatest(t *testing.T) {
	projectRoot := t.TempDir()
	workspace := t.TempDir()
	latestDir := filepath.Join(workspace, "latest")

	require.NoError(t, os.MkdirAll(latestDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(latestDir, "review.json"), []byte(`{"ok":true}`+"\n"), 0o644))

	mux := http.NewServeMux()
	registerLatestReviewArtifact(mux, "/api/reviews/latest/review", Options{
		Root:      projectRoot,
		Workspace: workspace,
	}, "review.json")

	req := httptest.NewRequest(http.MethodGet, "/api/reviews/latest/review", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"ok":true}`, rec.Body.String())
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestRegisterLatestReviewArtifact_ReturnsNotFoundWhenMissing(t *testing.T) {
	projectRoot := t.TempDir()
	workspace := t.TempDir()

	mux := http.NewServeMux()
	registerLatestReviewArtifact(mux, "/api/reviews/latest/review", Options{
		Root:      projectRoot,
		Workspace: workspace,
	}, "review.json")

	req := httptest.NewRequest(http.MethodGet, "/api/reviews/latest/review", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRegisterLatestReviewArtifact_RejectsNonGET(t *testing.T) {
	projectRoot := t.TempDir()
	workspace := t.TempDir()

	mux := http.NewServeMux()
	registerLatestReviewArtifact(mux, "/api/reviews/latest/review", Options{
		Root:      projectRoot,
		Workspace: workspace,
	}, "review.json")

	req := httptest.NewRequest(http.MethodPost, "/api/reviews/latest/review", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Equal(t, http.MethodGet, rec.Header().Get("Allow"))
}

func TestRegisterReviewRoutes_PostWorktreeWritesLatestBundleToWorkspace(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for review route integration test")
	}

	projectRoot := t.TempDir()
	workspace := t.TempDir()

	writeTestFile(t, projectRoot, "src/app.cc", `#include <memory>

int main() {
  return 0;
}
`)

	runGitForTest(t, projectRoot, "init")
	runGitForTest(t, projectRoot, "config", "user.email", "patchcourt@example.test")
	runGitForTest(t, projectRoot, "config", "user.name", "PatchCourt Test")
	runGitForTest(t, projectRoot, "add", ".")
	runGitForTest(t, projectRoot, "commit", "-m", "base")

	writeTestFile(t, projectRoot, "src/app.cc", `#include <memory>
#include <thread>
#include <chrono>

class Service {
 public:
  void Stop() {
    while (pending_callbacks_.load()) {
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
  }

 private:
  std::atomic<bool> pending_callbacks_{false};
};

int main() {
  return 0;
}
`)

	mux := http.NewServeMux()
	registerReviewRoutes(mux, Options{
		Root:      projectRoot,
		Workspace: workspace,
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/reviews",
		strings.NewReader(`{"base":"HEAD","worktree":true}`),
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	latestDir := filepath.Join(workspace, "latest")
	require.FileExists(t, filepath.Join(latestDir, "manifest.json"))
	require.FileExists(t, filepath.Join(latestDir, "review.json"))
	require.FileExists(t, filepath.Join(latestDir, "project-before.json"))
	require.FileExists(t, filepath.Join(latestDir, "project-after.json"))
	require.FileExists(t, filepath.Join(latestDir, "runtime.json"))

	require.NoDirExists(t, filepath.Join(projectRoot, ".patchcourt"))

	var response CreateReviewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.NotEmpty(t, response.ID)
	require.Equal(t, latestDir, response.BundleDir)
	require.Equal(t, "review.json", response.Artifacts["review"])
	require.NotContains(t, response.Artifacts, "html")
}

func runGitForTest(t *testing.T, root string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
}

func writeTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
