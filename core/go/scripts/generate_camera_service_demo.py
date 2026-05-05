from pathlib import Path
import shutil
import textwrap

ROOT = Path("examples/camera-service")

def write(path: str, content: str) -> None:
    target = ROOT / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(textwrap.dedent(content).lstrip(), encoding="utf-8")

def main() -> None:
    if ROOT.exists():
        shutil.rmtree(ROOT)

    write(".patchcourt.yaml", """
        ignore:
          paths:
            - build/**
            - .patchcourt/**

        layers:
          api:
            paths:
              - src/api/**
            may_depend_on:
              - application
              - domain

          application:
            paths:
              - src/application/**
            may_depend_on:
              - domain

          domain:
            paths:
              - src/domain/**
            may_depend_on: []

          cameras:
            paths:
              - src/infrastructure/cameras/**
            may_depend_on:
              - domain

          tests:
            paths:
              - tests/**
            may_depend_on:
              - api
              - application
              - domain
              - cameras
    """)

    # before
    write("before/src/domain/interfaces/i_camera_adapter.h", """
        #pragma once

        class ICameraAdapter {
        public:
            virtual ~ICameraAdapter() = default;
            virtual bool RunPreflight() const = 0;
            virtual bool StartSession(int count) const = 0;
        };
    """)

    write("before/src/application/camera_service.h", """
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
    """)

    write("before/src/application/camera_service.cc", """
        #include "application/camera_service.h"

        CameraService::CameraService(ICameraAdapter& camera)
            : camera_(camera) {}

        bool CameraService::Preflight() const {
            return camera_.RunPreflight();
        }

        bool CameraService::Start(int count) const {
            return camera_.StartSession(count);
        }
    """)

    write("before/src/api/camera_routes.cc", """
        #include "application/camera_service.h"

        bool HandlePreflight(CameraService& service) {
            return service.Preflight();
        }
    """)

    write("before/src/infrastructure/cameras/sony/sony_camera_manager.h", """
        #pragma once

        #include "domain/interfaces/i_camera_adapter.h"

        class SonyCameraManager final : public ICameraAdapter {
        public:
            bool RunPreflight() const override;
            bool StartSession(int count) const override;
        };
    """)

    write("before/src/infrastructure/cameras/sony/sony_camera_manager.cc", """
        #include "infrastructure/cameras/sony/sony_camera_manager.h"

        bool SonyCameraManager::RunPreflight() const {
            return true;
        }

        bool SonyCameraManager::StartSession(int count) const {
            return count > 0;
        }
    """)

    # after-bad
    write("after-bad/src/domain/interfaces/i_camera_adapter.h", """
        #pragma once

        class ICameraAdapter {
        public:
            virtual ~ICameraAdapter() = default;
            virtual bool RunPreflight(int camera_index) const = 0;
            bool StartSession(int count);
            virtual bool StopSession() const = 0;
        };
    """)

    write("after-bad/src/application/camera_service.h", """
        #pragma once

        #include "domain/interfaces/i_camera_adapter.h"

        class CameraService {
        public:
            explicit CameraService(ICameraAdapter& camera);
            bool Preflight(int camera_index) const;
            bool Start(int count) const;

        private:
            ICameraAdapter& camera_;
        };
    """)

    write("after-bad/src/application/camera_service.cc", """
        #include "application/camera_service.h"

        CameraService::CameraService(ICameraAdapter& camera)
            : camera_(camera) {}

        bool CameraService::Preflight(int camera_index) const {
            return camera_.RunPreflight(camera_index);
        }

        bool CameraService::Start(int count) const {
            return camera_.StartSession(count);
        }
    """)

    write("after-bad/src/api/camera_routes.cc", """
        #include "application/camera_service.h"
        #include "infrastructure/cameras/sony/sony_camera_manager.h"

        bool HandlePreflight(CameraService& service, SonyCameraManager& sony) {
            if (!sony.RunPreflight(0)) {
                return false;
            }

            return service.Preflight(0);
        }
    """)

    write("after-bad/src/infrastructure/cameras/sony/sony_camera_manager.h", """
        #pragma once

        #include "domain/interfaces/i_camera_adapter.h"

        class SonyCameraManager final : public ICameraAdapter {
        public:
            bool RunPreflight(int camera_index) const override;
            bool StartSession(int count);
            bool StopSession() const override;
        };
    """)

    write("after-bad/src/infrastructure/cameras/sony/sony_camera_manager.cc", """
        #include "infrastructure/cameras/sony/sony_camera_manager.h"

        bool SonyCameraManager::RunPreflight(int camera_index) const {
            return camera_index >= 0;
        }

        bool SonyCameraManager::StartSession(int count) {
            return count > 0;
        }

        bool SonyCameraManager::StopSession() const {
            return true;
        }
    """)

    # after-better
    write("after-better/src/domain/interfaces/i_camera_adapter.h", """
        #pragma once

        class ICameraAdapter {
        public:
            virtual ~ICameraAdapter() = default;
            virtual bool RunPreflight(int camera_index) const = 0;
            virtual bool StartSession(int count) const = 0;
            virtual bool StopSession() const = 0;
        };
    """)

    write("after-better/src/application/camera_service.h", """
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
    """)

    write("after-better/src/application/camera_service.cc", """
        #include "application/camera_service.h"

        CameraService::CameraService(ICameraAdapter& camera)
            : camera_(camera) {}

        bool CameraService::Preflight(int camera_index) const {
            return camera_.RunPreflight(camera_index);
        }

        bool CameraService::Start(int count) const {
            return camera_.StartSession(count);
        }

        bool CameraService::Stop() const {
            return camera_.StopSession();
        }
    """)

    write("after-better/src/api/camera_routes.cc", """
        #include "application/camera_service.h"

        bool HandlePreflight(CameraService& service) {
            return service.Preflight(0);
        }
    """)

    write("after-better/src/infrastructure/cameras/sony/sony_camera_manager.h", """
        #pragma once

        #include "domain/interfaces/i_camera_adapter.h"

        class SonyCameraManager final : public ICameraAdapter {
        public:
            bool RunPreflight(int camera_index) const override;
            bool StartSession(int count) const override;
            bool StopSession() const override;
        };
    """)

    write("after-better/src/infrastructure/cameras/sony/sony_camera_manager.cc", """
        #include "infrastructure/cameras/sony/sony_camera_manager.h"

        bool SonyCameraManager::RunPreflight(int camera_index) const {
            return camera_index >= 0;
        }

        bool SonyCameraManager::StartSession(int count) const {
            return count > 0;
        }

        bool SonyCameraManager::StopSession() const {
            return true;
        }
    """)

    write("after-better/tests/i_camera_adapter_test.cc", """
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
    """)

    write("README.md", """
        # PatchCourt camera-service demo

        This example demonstrates diff-aware architecture review on a small C++ camera service.

        ## Layout

        - `before/` — clean baseline.
        - `after-bad/` — intentionally bad patch:
          - API layer directly includes concrete Sony infrastructure.
          - Public camera contract changes.
          - No test-like files changed.
        - `after-better/` — improved patch:
          - API remains behind the application boundary.
          - Infrastructure depends inward on the domain interface.
          - A test-like file is updated.

        ## Run

        ```bash
        mkdir -p .patchcourt/out/examples/camera-service

        ./bin/patchcourt review \\
          --before-root examples/camera-service/before \\
          --after-root examples/camera-service/after-bad \\
          --config examples/camera-service/.patchcourt.yaml \\
          --format text \\
          --llm-pack \\
          --llm-pack-out .patchcourt/out/examples/camera-service/bad-context.md \\
          --html-out .patchcourt/out/examples/camera-service/bad-review.html \\
          > .patchcourt/out/examples/camera-service/bad-review.txt

        ./bin/patchcourt review \\
          --before-root examples/camera-service/before \\
          --after-root examples/camera-service/after-better \\
          --config examples/camera-service/.patchcourt.yaml \\
          --format text \\
          --llm-pack \\
          --llm-pack-out .patchcourt/out/examples/camera-service/better-context.md \\
          --html-out .patchcourt/out/examples/camera-service/better-review.html \\
          > .patchcourt/out/examples/camera-service/better-review.txt
        ```
    """)

if __name__ == "__main__":
    main()
