#include "application/camera_service.h"

CameraService::CameraService(ICameraAdapter& camera)
    : camera_(camera) {}

bool CameraService::Preflight(int camera_index) const {
    return camera_.RunPreflight(camera_index);
}

bool CameraService::Start(int count) const {
    return camera_.StartSession(count);
}

bool CameraService::Stop() const {
    return camera_.StopSession();
}
