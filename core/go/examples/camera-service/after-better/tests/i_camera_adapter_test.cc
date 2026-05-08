#include "domain/interfaces/i_camera_adapter.h"

class FakeCameraAdapter final : public ICameraAdapter {
public:
    bool RunPreflight(int camera_index) const override {
        return camera_index >= 0;
    }

    bool StartSession(int count) const override {
        return count > 0;
    }

    bool StopSession() const override {
        return true;
    }
};
