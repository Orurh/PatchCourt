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

	finding := requireFinding(t, findings, findingRawPointerAsyncCapture)
	require.Equal(t, model.SeverityHigh, finding.Severity)
	require.Equal(t, model.ConfidenceHigh, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 4, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "camera_ptr")
	require.Contains(t, finding.Evidence[0].Message, string(contextAsioPost))
}

func TestAnalyze_DetectsThisCapturedIntoAsioPost(t *testing.T) {
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

	finding := requireFinding(t, findings, findingThisCaptureAsync)
	require.Equal(t, model.SeverityHigh, finding.Severity)
	require.Equal(t, model.ConfidenceMedium, finding.Confidence)
	require.Len(t, finding.Evidence, 1)
	require.Equal(t, 3, finding.Evidence[0].LineStart)
	require.Contains(t, finding.Evidence[0].Message, "`this`")
	require.Contains(t, finding.Evidence[0].Message, string(contextAsioPost))
}

func TestAnalyze_DetectsThisCapturedIntoAsioAsyncCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/tcp_server.cc", `
void Accept() {
  acceptor_.async_accept(socket_, [this](boost::system::error_code ec) {
    OnAccept(ec);
  });
}
`)

	project := projectWithFile("src/tcp_server.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingThisCaptureAsync)
	require.Len(t, finding.Evidence, 1)
	require.Contains(t, finding.Evidence[0].Message, string(contextAsioAsyncOp))
}

func TestAnalyze_DetectsThisCapturedIntoStoredCallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/ftp_coordinator.cc", `
void Wire() {
  ftp_server_->SetUploadCompletedCallback([this](const std::string& path) {
    OnUploadCompleted(path);
  });
}
`)

	project := projectWithFile("src/ftp_coordinator.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingThisCaptureStoredCallback)
	require.Equal(t, model.SeverityMedium, finding.Severity)
	require.Len(t, finding.Evidence, 1)
	require.Contains(t, finding.Evidence[0].Message, string(contextStoredCallback))
}

func TestAnalyze_DetectsThisCapturedIntoThread(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/worker.cc", `
void Start() {
  thread_ = std::thread([this] {
    Run();
  });
}
`)

	project := projectWithFile("src/worker.cc")
	findings := Analyze(root, project)

	finding := requireFinding(t, findings, findingThisCaptureThread)
	require.Equal(t, model.SeverityHigh, finding.Severity)
	require.Len(t, finding.Evidence, 1)
	require.Contains(t, finding.Evidence[0].Message, string(contextThreadStart))
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

func TestAnalyze_DoesNotReportConditionVariableWaitPredicate(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/device.cc", `
bool WaitConnected() {
  return connection_ready_cv_.wait_for(lock, timeout, [this]() {
    return connected_callback_received_ && this->is_connected();
  });
}
`)

	project := projectWithFile("src/device.cc")
	findings := Analyze(root, project)

	require.Empty(t, findings)
}

func TestAnalyze_DoesNotReportLocalAlgorithmLambda(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/algorithm.cc", `
void Sort() {
  std::sort(items.begin(), items.end(), [this](const auto& left, const auto& right) {
    return Score(left) < Score(right);
  });
}
`)

	project := projectWithFile("src/algorithm.cc")
	findings := Analyze(root, project)

	require.Empty(t, findings)
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

func TestAnalyze_DoesNotReportPlainLambdaWithoutRecognizedDeferredContext(t *testing.T) {
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

func TestAnalyze_DoesNotLetNearbyThreadStealAsioPostContext(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/storage_space_worker.cc", `
void Start() {
  boost::asio::post(strand_, [this] {
    Fetch();
    ScheduleTick();
  });

  thread_ = std::thread([this] {
    ioc_.run();
  });
}
`)

	project := projectWithFile("src/storage_space_worker.cc")
	findings := Analyze(root, project)

	asyncFinding := requireFinding(t, findings, findingThisCaptureAsync)
	require.Len(t, asyncFinding.Evidence, 1)
	require.Equal(t, 3, asyncFinding.Evidence[0].LineStart)
	require.Contains(t, asyncFinding.Evidence[0].Message, string(contextAsioPost))

	threadFinding := requireFinding(t, findings, findingThisCaptureThread)
	require.Len(t, threadFinding.Evidence, 1)
	require.Equal(t, 8, threadFinding.Evidence[0].LineStart)
	require.Contains(t, threadFinding.Evidence[0].Message, string(contextThreadStart))
}

func TestAnalyze_DoesNotTreatStructuredBindingAsLambdaCapture(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/manager.cc", `
void ChangeAll() {
  for (const auto& [camera_id, camera_ptr] : cameras_to_change) {
    Do(camera_id, camera_ptr);
  }
}
`)

	project := projectWithFile("src/manager.cc")
	findings := Analyze(root, project)

	require.Empty(t, findings)
}
