#pragma once

#include "src/domain/interfaces/i_camera_adapter.h"

namespace patchcourt::example {

class DeviceOrchestrator {
public:
    bool RunPreflight() const;
};

} // namespace patchcourt::example
