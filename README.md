<div align="center">

# PatchCourt

### Diff-aware architecture review for C++ patches

**Did this patch make your architecture better or worse?**

PatchCourt reviews the architectural impact of C++ changes, separates **new risk** from **legacy debt**, and produces evidence down to files, include/import edges, layers, public contracts, findings, runtime-risk signals, and review questions.

<br/>

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![C++](https://img.shields.io/badge/Focus-C++_Architecture-00599C?style=for-the-badge&logo=cplusplus&logoColor=white)](https://isocpp.org/)
[![SARIF](https://img.shields.io/badge/SARIF-Code_Scanning-6f42c1?style=for-the-badge)](https://sarifweb.azurewebsites.net/)
[![Alpha](https://img.shields.io/badge/status-v0.2.2--alpha-orange?style=for-the-badge)](#v022-alpha-release-focus)

[Русская версия](README.ru.md)

</div>

---

## Why PatchCourt exists

Large C++ systems rarely lose architecture in one dramatic commit.

They drift one patch at a time:

```text
API code starts including vendor-specific implementation headers.
Domain code pulls infrastructure details.
Public interfaces change without related tests.
Old dependency cycles become “it has always been like that”.
Reviewers cannot tell whether the patch introduced new risk or only touched old debt.
```

Most dependency tools show the whole existing architecture mess.

PatchCourt focuses on the patch:

```text
What got worse?
What got better?
What was already legacy debt?
Where exactly is the evidence?
What should the reviewer inspect first?
```

PatchCourt is not a compiler, clang-tidy replacement, generic linter, security scanner, or AI reviewer.

It is a deterministic evidence engine for architecture review.

```text
project facts
  -> dependency graph
  -> architecture rules
  -> patch diff
  -> findings
  -> review bundle
  -> local viewer / JSON / LLM context / SARIF
```

---

## Core idea

PatchCourt answers one review question:

> Did this C++ patch make the architecture better or worse?

And it answers with structured evidence:

| Signal | Example |
|---|---|
| New forbidden dependency | `api -> cameras/sony` |
| New layer edge | `domain -> infrastructure` |
| Public contract changed | `method::ICameraAdapter::RunPreflight` |
| Missing test-like changes | public interface changed, tests did not |
| Runtime architecture risk | `this` captured into async callback |
| Existing debt | old cycle was already present before the patch |
| Better change | forbidden dependency removed |

The key split:

```text
Worse          -> introduced or amplified by this patch
Better         -> improved by this patch
Unchanged debt -> already existed before this patch
```

That split is the product.

---

## Quick start from source

```bash
git clone https://github.com/orurh/PatchCourt.git
cd PatchCourt/core/go

make build
make viewer-build
```

Run a lightweight project check with default/auto-discovered configuration:

```bash
./bin/patchcourt check . --out .patchcourt/out/check
```

Or through Makefile:

```bash
make check PROJECT=. CONFIG=
```

`CONFIG=` intentionally disables an explicit `.patchcourt.yaml` and lets PatchCourt use default/auto-discovered configuration.

---

## Local viewer workflow

The easiest review workflow is the local viewer:

```bash
cd PatchCourt/core/go

./bin/patchcourt open . \
  --base origin/main \
  --worktree \
  --review-now
```

PatchCourt starts a local API/viewer server and creates an initial review bundle.

By default it tries to find built viewer assets automatically:

```text
./viewer-dist
viewer-dist next to the patchcourt binary
web/viewer/dist in development checkout
```

For headless use:

```bash
./bin/patchcourt open . \
  --base origin/main \
  --worktree \
  --review-now \
  --no-browser
```

Useful API checks while the server is running:

```bash
curl -s http://127.0.0.1:8787/api/health
curl -s http://127.0.0.1:8787/api/reviews/latest/graph | jq '{nodes: (.nodes | length), edges: (.edges | length)}'
```

---

## Review bundle workflow

PatchCourt can write a self-contained review bundle:

```bash
./bin/patchcourt review \
  --base origin/main \
  --worktree \
  --root . \
  --out .patchcourt/out/latest
```

The bundle contains:

```text
manifest.json
review.json
project-before.json
project-after.json
graph.json
runtime.json
tree.json
findings.json
contracts.json
dependencies.json
review-context.md
patchcourt.sarif
```

This bundle is the source of truth for the local viewer and integrations.

The old static-HTML-first workflow is no longer the primary path. The current product direction is **bundle/data-first + local viewer**.

---

## Release archive workflow

Build a release archive with the binary and viewer assets:

```bash
cd core/go
make release-archive RELEASE_VERSION=v0.2.2-alpha
```

Expected archive structure:

```text
patchcourt-v0.2.2-alpha/
  patchcourt
  viewer-dist/
    index.html
    assets/...
```

After unpacking:

```bash
./patchcourt open /path/to/project --review-now
```

No manual `--viewer-dir` should be needed for release archives.

---

## Built-in camera-service demo

Run the demo:

```bash
cd core/go
make camera-demo
```

The demo generates bad/better review outputs under:

```text
.patchcourt/out/examples/camera-service/
```

Current bundle-oriented demo artifacts include:

```text
bad-review.txt
bad-review.md
bad/review.json
bad/review-context.md
bad/patchcourt.sarif

better-review.txt
better-review.md
better/review.json
better/review-context.md
better/patchcourt.sarif
```

Open the old static demo reports only if they are present in your checkout:

```bash
make open-camera-demo
```

---

## What PatchCourt generates

| Artifact | Purpose |
|---|---|
| `manifest.json` | Bundle manifest and schema version |
| `review.json` | Machine-readable PatchCourt review result |
| `project-before.json` | Before-project model |
| `project-after.json` | After-project model |
| `graph.json` | Review graph for the local viewer |
| `tree.json` | Project tree for the local viewer |
| `runtime.json` | Runtime architecture risk report |
| `findings.json` | Finding changes and evidence |
| `contracts.json` | Public contract changes and impact |
| `dependencies.json` | Dependency and layer-edge changes |
| `review-context.md` | Deterministic LLM-ready context pack |
| `patchcourt.sarif` | CI/code scanning export |

SARIF is an integration layer. The primary PatchCourt artifact is the review bundle.

---

## LLM context pack

PatchCourt prepares a deterministic context pack for LLM-assisted review:

```text
review-context.md
```

It contains:

```text
patch summary
raw changed files
analyzed changed files
touched layers
architecture impact
contract changes
dependency changes
finding changes
runtime risks
risk reasons
review questions
```

Principle:

```text
LLM may summarize, explain, and generate review questions.
LLM must not invent files, symbols, dependencies, or findings.
```

PatchCourt collects facts first. LLM assistance should work on top of evidence.

---

## SARIF and CI

PatchCourt writes SARIF into the review bundle:

```text
.patchcourt/out/latest/patchcourt.sarif
```

Minimal CI flow:

```yaml
name: PatchCourt

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  security-events: write

jobs:
  patchcourt:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Run PatchCourt review
        run: |
          patchcourt review \
            --base origin/main \
            --head HEAD \
            --root . \
            --out .patchcourt/out/latest

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: .patchcourt/out/latest/patchcourt.sarif

      - name: Upload PatchCourt artifacts
        uses: actions/upload-artifact@v4
        with:
          name: patchcourt-review-bundle
          path: .patchcourt/out/latest
```

More examples:

```text
core/go/docs/ci/github-actions.md
core/go/docs/ci/gitlab-ci.md
```

Recommended alpha behavior:

```text
generate review bundle
upload SARIF where supported
upload review artifacts
do not fail CI by default
```

Blocking mode should be explicit and project-owned.

---

## Project check mode

PatchCourt can inspect the current project architecture:

```bash
./bin/patchcourt check /path/to/project --out /path/to/project/.patchcourt/out/check
```

Typical artifacts:

```text
project-model.json
scan.md
layer-graph.json
layer-graph.dot
layer-graph.mmd
```

Use this for:

```text
understanding project structure
finding suspicious layer edges
generating dependency graphs
building an initial baseline config
```

---

## Edge drill-down

Explain a concrete layer dependency:

```bash
./bin/patchcourt edge \
  --model .patchcourt/out/check/project-model.json \
  api cameras
```

Example output shape:

```text
PatchCourt edge

Edge: api -> cameras
Count: 3

Top source files:
  src/api/camera_routes.cc

Top target files:
  src/cameras/sony/sony_camera_manager.h

Dependencies:
  src/api/camera_routes.cc
    -> src/cameras/sony/sony_camera_manager.h [used]
```

Graphs are useful. Evidence is better.

---

## Configuration

PatchCourt can run without a config first.

```bash
./bin/patchcourt check . --out .patchcourt/out/check
./bin/patchcourt review --base origin/main --worktree --root . --out .patchcourt/out/latest
```

Use `.patchcourt.yaml` when the project wants explicit architecture policy.

Example:

```yaml
ignore:
  paths:
    - build/**
    - cmake-build-*/**
    - third_party/**
    - external/**
    - generated/**
    - "**/*.pb.cc"
    - "**/*.pb.h"

cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - src
    - include

layers:
  api:
    paths:
      - src/api/**
      - src/server/**
    may_depend_on:
      - controllers
      - domain

  controllers:
    paths:
      - src/controllers/**
    may_depend_on:
      - domain
      - cameras

  domain:
    paths:
      - src/domain/**
    may_depend_on: []

  cameras:
    paths:
      - src/cameras/**
    may_depend_on:
      - domain

forbidden_imports:
  - from_layer: api
    patterns:
      - src/cameras/sony/**
      - src/cameras/*_impl/**
```

Generate a starting config:

```bash
./bin/patchcourt init /path/to/project > /path/to/project/.patchcourt.yaml
```

For legacy projects, start with report-only review and add policy gradually.

---

## What works today

| Area | Status |
|---|---|
| C++ file indexing | works |
| C++ include graph | works |
| `compile_commands.json` discovery | works |
| configured include paths | works |
| Go import baseline | works |
| layer rules via `.patchcourt.yaml` | works |
| architecture findings | works |
| edge drill-down | works |
| before/after review | works |
| git base/head review | works |
| worktree review | works |
| review bundle output via `--out` | works |
| local viewer via `patchcourt open` | alpha |
| viewer asset auto-discovery | alpha |
| project tree / architecture graph viewer | alpha |
| public contract diff | alpha |
| runtime architecture risk signals | alpha |
| LLM context pack | alpha |
| SARIF export | alpha |

---

## What PatchCourt is not

PatchCourt is not:

```text
a C++ compiler frontend
a clang-tidy replacement
a proof of correctness
a generic security scanner
a Go linter replacement
a full AI code reviewer
a SaaS product in the current alpha
```

PatchCourt is:

```text
a deterministic architecture-impact reviewer for patches
```

---

## Current limitations

PatchCourt is alpha-stage software.

Current limitations:

- C++ analysis is lightweight and does not use Clang AST yet.
- Include resolution quality depends on `compile_commands.json`, project layout, or `.patchcourt.yaml`.
- CMake lightweight extraction is not a full CMake evaluator.
- Public contract extraction is heuristic.
- Runtime risk rules are intentionally conservative and evidence-first.
- Risk score is review prioritization, not a correctness verdict.
- SARIF is an export/integration layer, not the core PatchCourt model.
- Go support is baseline-level and not the main product focus.
- False positives are possible and should be reviewed with the provided evidence.

---

## v0.2.2-alpha release focus

`v0.2.2-alpha` focuses on making the bundle/viewer workflow usable:

```text
review bundle via --out
local viewer via patchcourt open
automatic viewer-dist discovery
release archive with binary + viewer-dist
full review graph in open --review-now
SARIF and LLM context generated as bundle artifacts
make release-check
make release-archive
```

Not included in this release:

```text
mandatory Clang backend
VS Code extension
SaaS/web platform
GitHub PR bot as the main workflow
deep cache
suppressions UI
broad Go/C++ risk-rule expansion
AI architect behavior
```

---

## Development

From `core/go`:

```bash
make help
make ci
make viewer-build
make camera-demo
make open-self-nobrowser OPEN_REVIEW_NOW=true BASE=origin/main
make release-check
make release-archive RELEASE_VERSION=v0.2.2-alpha
```

Architecture guardrails are enforced by tests.

Core/usecase/analyzer packages return structured results and must not write directly to stdout/stderr.

---

## License

Apache-2.0.
