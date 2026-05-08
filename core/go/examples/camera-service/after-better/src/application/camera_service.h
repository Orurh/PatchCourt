#pragma once

#include "domain/interfaces/i_camera_adapter.h"

class CameraService {
public:
    explicit CameraService(ICameraAdapter& camera);
    bool Preflight(int camera_index) const;
    bool Start(int count) const;
    bool Stop() const;

private:
    ICameraAdapter& camera_;
};
