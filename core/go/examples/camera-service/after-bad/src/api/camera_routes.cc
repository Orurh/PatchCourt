#include "application/camera_service.h"
#include "infrastructure/cameras/sony/sony_camera_manager.h"

bool HandlePreflight(CameraService& service, SonyCameraManager& sony) {
    if (!sony.RunPreflight(0)) {
        return false;
    }

    return service.Preflight(0);
}
