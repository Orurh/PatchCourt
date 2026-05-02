#pragma once

namespace patchcourt::example {

class ICameraAdapter {
public:
    virtual ~ICameraAdapter() = default;
    virtual bool RunPreflight() const = 0;
};

} // namespace patchcourt::example
