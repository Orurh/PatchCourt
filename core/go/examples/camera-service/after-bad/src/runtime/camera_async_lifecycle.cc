#include <chrono>
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
