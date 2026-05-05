#include "application/camera_service.h"

CameraService::CameraService(ICameraAdapter& camera)
    : camera_(camera) {}

bool CameraService::Preflight() const {
    return camera_.RunPreflight();
}

bool CameraService::Start(int count) const {
    return camera_.StartSession(count);
}
