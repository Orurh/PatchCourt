#pragma once

class ICameraAdapter {
public:
    virtual ~ICameraAdapter() = default;
    virtual bool RunPreflight(int camera_index) const = 0;
    virtual bool StartSession(int count) const = 0;
    virtual bool StopSession() const = 0;
};
