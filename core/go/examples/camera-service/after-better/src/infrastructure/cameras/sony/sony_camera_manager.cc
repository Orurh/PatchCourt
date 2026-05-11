#include "infrastructure/cameras/sony/sony_camera_manager.h"

bool SonyCameraManager::RunPreflight(int camera_index) const {
    return camera_index >= 0;
}

bool SonyCameraManager::StartSession(int count) const {
    return count > 0;
}

bool SonyCameraManager::StopSession() const {
    return true;
}
