#include <memory>

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
