#pragma once

class ICameraAdapter {
public:
    virtual ~ICameraAdapter() = default;
    virtual bool RunPreflight() const = 0;
    virtual bool StartSession(int count) const = 0;
};
