#include "infrastructure/cameras/sony/sony_camera_manager.h"

bool SonyCameraManager::RunPreflight() const {
    return true;
}

bool SonyCameraManager::StartSession(int count) const {
    return count > 0;
}
