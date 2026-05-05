#pragma once

#include "domain/interfaces/i_camera_adapter.h"

class SonyCameraManager final : public ICameraAdapter {
public:
    bool RunPreflight(int camera_index) const override;
    bool StartSession(int count) const override;
    bool StopSession() const override;
};
