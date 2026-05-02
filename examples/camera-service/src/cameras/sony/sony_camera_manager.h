#pragma once

#include "src/domain/interfaces/i_camera_adapter.h"

namespace patchcourt::example {

class SonyCameraManager final : public ICameraAdapter {
public:
    bool RunPreflight() const override;
};

} // namespace patchcourt::example
