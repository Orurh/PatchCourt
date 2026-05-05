#pragma once

class ICameraAdapter {
public:
    virtual ~ICameraAdapter() = default;
    virtual bool RunPreflight(int camera_index) const = 0;
    bool StartSession(int count);
    virtual bool StopSession() const = 0;
};
