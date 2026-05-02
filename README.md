# PatchCourt

PatchCourt is a static architecture review tool for C/C++ projects.

It builds an include-level dependency model, maps files to architectural layers, detects suspicious coupling, and produces review-friendly reports for humans, CI, and future LLM-assisted code review.

The current focus is C/C++ architecture analysis.

---

## What PatchCourt does

PatchCourt answers questions like:

- Which layers depend on each other?
- Did this patch introduce a new forbidden dependency?
- Are there bidirectional layer dependencies?
- Does a domain layer depend on outer implementation code?
- Are there possibly unused C++ includes?
- What evidence supports each finding?
- What artifacts can be attached to a merge request?

PatchCourt is intentionally split into several stages:

```text
facts      -> include/import graph
discovery  -> metrics, clusters, suspicious dependencies
policy     -> explicit allowed dependencies
review     -> what changed in a patch
explain    -> why a finding exists
artifacts  -> markdown, json, dot, mermaid
```

---

## Current status

PatchCourt currently supports:

- C/C++ file indexing
- C/C++ `#include` dependency graph
- `compile_commands.json` autodiscovery
- configured include paths
- system include handling
- project layer discovery
- explicit layer policy via `.patchcourt.yaml`
- architecture violation findings
- discovery hints
- possibly unused include detection
- local suppressions via comments
- scan reports in text/json/markdown
- layer graph output in dot/json/mermaid
- before/after review reports
- markdown review output
- finding explanation
- one-command project check with generated artifacts

Go project analysis is not the current focus yet.

---

## Installation

From the Go implementation directory:

```bash
cd core/go
go build -o ./bin/patchcourt ./cmd/patchcourt
```

Run tests:

```bash
go test ./...
```

---

## Quick start

Run a full project check:

```bash
./bin/patchcourt check /path/to/project
```

This writes standard artifacts to:

```text
/path/to/project/.patchcourt/out/
```

Typical output:

```text
PatchCourt check

Root: /path/to/project
Config: defaults
Out: /path/to/project/.patchcourt/out

Summary:
  production files: 101
  test files:       24
  dependencies:     686
  resolved:         301
  unresolved:       0
  findings:         9
  graph nodes:      10
  graph edges:      25

Artifacts:
  - project model: /path/to/project/.patchcourt/out/project-model.json
  - scan report: /path/to/project/.patchcourt/out/scan.md
  - layer graph json: /path/to/project/.patchcourt/out/layer-graph.json
  - layer graph dot: /path/to/project/.patchcourt/out/layer-graph.dot
  - layer graph mermaid: /path/to/project/.patchcourt/out/layer-graph.mmd
```

Generate an SVG graph:

```bash
dot -Tsvg /path/to/project/.patchcourt/out/layer-graph.dot \
  -o /path/to/project/.patchcourt/out/layer-graph.svg
```

---

## Commands

### `check`

Run scan + graph and write standard artifacts.

```bash
./bin/patchcourt check /path/to/project
```

With explicit config and output directory:

```bash
./bin/patchcourt check /path/to/project \
  --config /path/to/project/.patchcourt.yaml \
  --out /tmp/patchcourt-out
```

Generated artifacts:

```text
project-model.json
scan.md
layer-graph.json
layer-graph.dot
layer-graph.mmd
```

This is the recommended command for day-to-day use.

---

### `init`

Generate an initial `.patchcourt.yaml`.

Baseline mode infers current dependencies as allowed dependencies:

```bash
./bin/patchcourt init /path/to/project > .patchcourt.yaml
```

Strict mode discovers layers but leaves `may_depend_on` empty:

```bash
./bin/patchcourt init /path/to/project --strict > .patchcourt.yaml
```

Baseline mode is useful for legacy projects where the first goal is to prevent new architectural drift.

Strict mode is useful when you want PatchCourt to immediately report existing architecture violations.

Example generated config:

```yaml
ignore:
  paths:
    - ".git/**"
    - "build/**"
    - "libs/**"
    - "third_party/**"
    - "external/**"
    - "generated/**"
    - "**/*.pb.h"
    - "**/*.pb.cc"

cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - "src"

layers:
  server:
    paths:
      - "src/server/**"
    may_depend_on:
      - domain

  domain:
    paths:
      - "src/domain/**"
    may_depend_on: []
```

---

### `scan`

Build the project model and report findings.

Text output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format text
```

JSON output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format json > project-model.json
```

Markdown output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format markdown > scan.md
```

The scan model contains:

- files
- file roles
- symbols
- dependencies
- resolved/unresolved includes
- external dependencies
- layer assignments
- findings
- evidence

---

### `graph`

Build a layer graph from the project model.

DOT:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format dot > layer-graph.dot
```

Mermaid:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format mermaid > layer-graph.mmd
```

JSON:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format json > layer-graph.json
```

Generate SVG:

```bash
dot -Tsvg layer-graph.dot -o layer-graph.svg
```

---

### `review`

Compare before/after project models or before/after roots.

Review from two roots:

```bash
./bin/patchcourt review \
  --before-root /tmp/project-before \
  --after-root /tmp/project-after \
  --config .patchcourt.yaml \
  --format markdown
```

Review from two prebuilt models:

```bash
./bin/patchcourt review \
  --before before-model.json \
  --after after-model.json \
  --format text
```

Markdown review output is designed for merge requests:

```markdown
# PatchCourt Review

## Summary

- Risk: high, 11 points
- Dependency changes: 1
- Layer edge changes: 1
- Added findings: 1
- Added policy findings: 1

## Risk reasons

- +7 added high policy violation: architecture.server.cameras
- +1 dependency edge added: include|src/server/api_router.cc|src/cameras/camera_adapter_factory.h
- +3 layer edge added: server -> cameras
```

---

### `explain`

Explain a specific finding.

From root:

```bash
./bin/patchcourt explain architecture.server.cameras \
  --root /path/to/project \
  --config .patchcourt.yaml
```

From model:

```bash
./bin/patchcourt explain architecture.server.cameras \
  --model .patchcourt/out/project-model.json
```

Example output:

```text
PatchCourt explain

Finding: architecture.server.cameras
Title:   Include-level architecture boundary violation
Kind:    policy_violation
Severity: high
Confidence: high

Risk:
  Layer "server" includes a header from layer "cameras", which is not allowed by .patchcourt.yaml.

Evidence:
  - src/server/api_router.cc: includes src/cameras/camera_adapter_factory.h, creating include dependency server -> cameras
```

---

## Findings

PatchCourt currently emits two broad categories of findings.

### Policy violations

Policy violations come from explicit `.patchcourt.yaml` rules.

Example:

```yaml
layers:
  server:
    paths:
      - "src/server/**"
    may_depend_on:
      - domain

  cameras:
    paths:
      - "src/cameras/**"
    may_depend_on:
      - domain
```

If `src/server/api_router.cc` includes `src/cameras/camera_adapter_factory.h`, PatchCourt reports:

```text
architecture.server.cameras
```

### Discovery hints

Discovery hints are best-effort architecture smells found from the dependency graph.

Examples:

```text
discovery.bidirectional.application.cameras
discovery.bidirectional.domain.session
discovery.controllers.depends_on.server
discovery.domain.depends_on.application
discovery.cpp.unused_includes
```

These are not strict policy violations. They are review hints.

---

## Suppressions

PatchCourt supports local finding suppression comments.

Example:

```cpp
// patchcourt:ignore architecture.server.cameras reason: legacy direct adapter include
#include "src/cameras/camera_adapter_factory.h"
```

Suppressed findings are not reported for that file.

Use suppressions sparingly. Prefer fixing architecture boundaries or updating the policy when the dependency is intentional.

---

## C++ include resolution

PatchCourt resolves includes using several sources:

- direct project paths
- configured `cpp.include_paths`
- discovered `compile_commands.json`
- system include paths
- heuristic fallback

Example config:

```yaml
cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - "src"
    - "include"
```

PatchCourt searches common compile database locations automatically:

```text
compile_commands.json
build/compile_commands.json
```

---

## Ignored paths

PatchCourt applies default ignores when no config is provided.

Default ignored paths include:

```text
.git/**
build/**
cmake-build-debug/**
cmake-build-release/**
node_modules/**
vendor/**
libs/**
third_party/**
external/**
generated/**
**/*.pb.h
**/*.pb.cc
**/*.grpc.pb.h
**/*.grpc.pb.cc
```

This keeps vendor, generated, and build artifacts out of project architecture analysis.

---

## File roles

PatchCourt classifies files as:

```text
production
test
generated
external
config
unknown
```

Architecture findings, layer graphs, and review risk ignore dependencies originating from test/generated/external files.

The goal is to focus architecture review on production code.

---

## Example workflow for a C++ project

Generate baseline config:

```bash
./bin/patchcourt init /path/to/project > /path/to/project/.patchcourt.yaml
```

Run check:

```bash
./bin/patchcourt check /path/to/project \
  --config /path/to/project/.patchcourt.yaml
```

Open generated graph:

```bash
dot -Tsvg /path/to/project/.patchcourt/out/layer-graph.dot \
  -o /path/to/project/.patchcourt/out/layer-graph.svg

xdg-open /path/to/project/.patchcourt/out/layer-graph.svg
```

Explain the top finding:

```bash
./bin/patchcourt explain discovery.bidirectional.application.cameras \
  --model /path/to/project/.patchcourt/out/project-model.json
```

---

## Example review workflow

Prepare before/after copies:

```bash
cp -a /path/to/project /tmp/project-before
cp -a /path/to/project /tmp/project-after
```

Make a change in `/tmp/project-after`.

Run review:

```bash
./bin/patchcourt review \
  --before-root /tmp/project-before \
  --after-root /tmp/project-after \
  --config /path/to/project/.patchcourt.yaml \
  --format markdown > review.md
```

Attach `review.md` to a merge request or paste it into a code review discussion.

---

## Design principles

PatchCourt is designed around a few constraints:

1. Facts first. The include graph is collected before policy is applied.
2. Discovery is not policy. Suspicious dependencies are hints unless explicitly forbidden.
3. Policy is explicit. `.patchcourt.yaml` defines allowed layer dependencies.
4. Review is evidence-based. Every finding should have concrete file-level evidence.
5. Low noise matters. Test, generated, external, vendor, and build files should not dominate architecture reports.
6. C++ include dependencies are compile-time dependencies. Even unused includes can increase coupling and build cost.

---

## Current limitations

PatchCourt is still early.

Known limitations:

- C++ parsing is lightweight and syntactic.
- Symbol usage detection is heuristic.
- Macro-heavy and template-heavy code can produce false positives.
- Header-only libraries can confuse unused-include detection.
- Go analysis is not the current focus.
- `review --base main --head HEAD` is not implemented yet.
- Interactive UI / 3D graph viewer is not implemented yet.

---

## Roadmap

Near-term:

- `check --format json`
- better CI-oriented exit codes
- `review --base main --head HEAD`
- LLM review context pack
- stricter config validation
- better include resolution diagnostics
- improved unused include confidence
- HTML/interactive graph report

Possible future:

- local web UI
- VS Code extension
- graph exploration with highlighted findings
- trend/baseline tracking
- per-team architecture presets
- Go package graph analysis

---

## Development

Run all tests:

```bash
go test ./...
```

Build CLI:

```bash
go build -o ./bin/patchcourt ./cmd/patchcourt
```

Run against this project:

```bash
./bin/patchcourt check .
```

---

## Repository layout

```text
cmd/patchcourt                 CLI entrypoint

internal/app                   use cases: scan, graph, review, explain, check
internal/analysis              analyzers and domain logic
internal/config                config loading, validation, defaults
internal/model                 project model and shared data structures
internal/output/report         text/json/markdown/dot/mermaid renderers
internal/platform              filesystem, path, git, logging helpers
```

---

## License

TBD.