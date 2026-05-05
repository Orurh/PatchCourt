#pragma once

#include "domain/interfaces/i_camera_adapter.h"

class CameraService {
public:
    explicit CameraService(ICameraAdapter& camera);
    bool Preflight() const;
    bool Start(int count) const;

private:
    ICameraAdapter& camera_;
};
