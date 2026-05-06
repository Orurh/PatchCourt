package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orurh/patchcourt/internal/model"
	"github.com/stretchr/testify/require"
)

func TestAnalyze_DetectsRawPointerCapturedIntoAsyncTask(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/gopro_manager.cc", `
void Start() {
  auto* camera_ptr = camera.get();
  boost::asio::post(*thread_pool_, [this, camera_id, camera_ptr]() {
    camera_ptr->HealthCheck();
  });
}
`)

	project := projectWithFile("src/gopro_manager.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingRawPointerCapture)
	require.Equal(t, model.SeverityHigh, finding.Severity)
	require.Equal(t, model.ConfidenceHigh, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 4, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "camera_ptr")
}

func TestAnalyze_DetectsThisCapturedIntoAsyncCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/gopro_manager.cc", `
void Start() {
  boost::asio::post(*thread_pool_, [this]() {
    OnCameraResult();
  });
}
`)

	project := projectWithFile("src/gopro_manager.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingThisCapture)
	require.Equal(t, model.SeverityHigh, finding.Severity)
	require.Equal(t, model.ConfidenceMedium, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 3, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "`this`")
}

func TestAnalyze_DetectsShutdownSleepPollingLoop(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/gopro_manager.cc", `
void DisconnectAll() {
  while (pending_disconnects.load() > 0) {
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
  }
}
`)

	project := projectWithFile("src/gopro_manager.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingShutdownPolling)
	require.Equal(t, model.SeverityMedium, finding.Severity)
	require.Equal(t, model.ConfidenceMedium, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 4, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "loop")
}

func TestAnalyze_DetectsDetachedDelayedShutdownCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/api_router.cc", `
void HandleShutdown() {
  if (shutdown_callback_) {
    std::thread([callback = shutdown_callback_]() {
      std::this_thread::sleep_for(std::chrono::milliseconds(100));
      callback();
    }).detach();
  }
}
`)

	project := projectWithFile("src/api_router.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingDetachedDelayedCallback)
	require.Equal(t, model.SeverityMedium, finding.Severity)
	require.Equal(t, model.ConfidenceMedium, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 5, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "detached thread")

	requireNoFinding(t, findings, findingShutdownPolling)
}

func TestAnalyze_DoesNotTreatSimpleBackoffSleepAsShutdownPolling(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/web_server.cc", `
void Run() {
  while (running) {
    if (would_block) {
      std::this_thread::sleep_for(backoff);
      continue;
    }
  }
}
`)

	project := projectWithFile("src/web_server.cc")
	findings := Analyze(root, project)

	requireNoFinding(t, findings, findingShutdownPolling)
	requireNoFinding(t, findings, findingDetachedDelayedCallback)
}

func TestAnalyze_IgnoresGeneratedAndExternalFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "generated/foo.pb.cc", `
void Start() {
  boost::asio::post(*thread_pool_, [this]() {});
}
`)
	writeFile(t, root, "third_party/lib.cc", `
void Start() {
  boost::asio::post(*thread_pool_, [this]() {});
}
`)

	project := &model.ProjectModel{
		Root: root,
		Files: []model.FileModel{
			{
				Path:     "generated/foo.pb.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleGenerated,
			},
			{
				Path:     "third_party/lib.cc",
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleExternal,
			},
		},
	}

	findings := Analyze(root, project)

	require.Empty(t, findings)
}

func TestAnalyze_DoesNotReportPlainLambdaWithoutAsyncContext(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/plain.cc", `
void Run() {
  auto fn = [this]() {
    DoWork();
  };
}
`)

	project := projectWithFile("src/plain.cc")
	findings := Analyze(root, project)

	require.Empty(t, findings)
}

func projectWithFile(path string) *model.ProjectModel {
	return &model.ProjectModel{
		Root: ".",
		Files: []model.FileModel{
			{
				Path:     path,
				Language: model.LanguageCPP,
				Kind:     model.FileKindSource,
				Role:     model.FileRoleProduction,
			},
		},
	}
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func requireFinding(t *testing.T, findings []model.Finding, id string) model.Finding {
	t.Helper()

	for _, finding := range findings {
		if finding.ID == id {
			return finding
		}
	}

	t.Fatalf("finding %s not found in %#v", id, findings)
	return model.Finding{}
}

func requireNoFinding(t *testing.T, findings []model.Finding, id string) {
	t.Helper()

	for _, finding := range findings {
		if finding.ID == id {
			t.Fatalf("unexpected finding %s in %#v", id, findings)
		}
	}
}
