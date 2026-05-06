#!/usr/bin/env python3
from pathlib import Path

ROOT = Path("examples/camera-service")

BAD_RUNTIME = r'''#include <chrono>
#include <memory>
#include <thread>

namespace boost::asio {
template <class Pool, class Fn>
void post(Pool&, Fn) {}
}

namespace camera_service {

class Camera {
 public:
  void HealthCheck() {}
};

class CameraManager {
 public:
  void StartHealthCheck();
  void DisconnectAll();

 private:
  std::unique_ptr<Camera> camera_;
  int thread_pool_ = 0;
  int pending_disconnects_ = 1;

  void OnCameraResult() {}
};

void CameraManager::StartHealthCheck() {
  auto* camera_ptr = camera_.get();

  boost::asio::post(thread_pool_, [this, camera_ptr]() {
    camera_ptr->HealthCheck();
    OnCameraResult();
  });
}

void CameraManager::DisconnectAll() {
  while (pending_disconnects_ > 0) {
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
  }
}

}  // namespace camera_service
'''

BETTER_RUNTIME = r'''#include <memory>

namespace boost::asio {
template <class Pool, class Fn>
void post(Pool&, Fn) {}
}

namespace camera_service {

class Camera {
 public:
  void HealthCheck() {}
};

class CameraManager : public std::enable_shared_from_this<CameraManager> {
 public:
  void StartHealthCheck();

 private:
  std::shared_ptr<Camera> camera_;
  int thread_pool_ = 0;

  void OnCameraResult() {}
};

void CameraManager::StartHealthCheck() {
  std::weak_ptr<CameraManager> self = weak_from_this();
  std::shared_ptr<Camera> camera = camera_;

  boost::asio::post(thread_pool_, [self, camera]() {
    if (auto locked = self.lock()) {
      camera->HealthCheck();
      locked->OnCameraResult();
    }
  });
}

}  // namespace camera_service
'''

def write(path: Path, data: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(data)

def main() -> None:
    write(
        ROOT / "after-bad/src/runtime/camera_async_lifecycle.cc",
        BAD_RUNTIME,
    )
    write(
        ROOT / "after-better/src/runtime/camera_async_lifecycle.cc",
        BETTER_RUNTIME,
    )

if __name__ == "__main__":
    main()
