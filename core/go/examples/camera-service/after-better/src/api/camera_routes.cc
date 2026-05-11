#include "application/camera_service.h"

bool HandlePreflight(CameraService& service) {
    return service.Preflight(0);
}
