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

./bin/patchcourt review \
  --before-root examples/camera-service/before \
  --after-root examples/camera-service/after-bad \
  --config examples/camera-service/.patchcourt.yaml \
  --format text \
  --llm-pack \
  --llm-pack-out .patchcourt/out/examples/camera-service/bad-context.md \
  --html-out .patchcourt/out/examples/camera-service/bad-review.html \
  > .patchcourt/out/examples/camera-service/bad-review.txt

./bin/patchcourt review \
  --before-root examples/camera-service/before \
  --after-root examples/camera-service/after-better \
  --config examples/camera-service/.patchcourt.yaml \
  --format text \
  --llm-pack \
  --llm-pack-out .patchcourt/out/examples/camera-service/better-context.md \
  --html-out .patchcourt/out/examples/camera-service/better-review.html \
  > .patchcourt/out/examples/camera-service/better-review.txt
```
