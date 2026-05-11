# PatchCourt Architecture

PatchCourt is a deterministic-first architecture review tool.

The core pipeline is:

```text
project sources
  -> project facts
  -> dependencies / contracts / runtime signals
  -> diff
  -> ReviewResult
  -> bundle artifacts
  -> CLI / local viewer / SARIF / LLM context
```

PatchCourt should collect facts first and render them later. Core analysis code should not print directly to stdout/stderr.

## Repository layout

```text
core/go
  cmd/patchcourt
    CLI entrypoint.

  internal/adapter/cli
    Cobra commands and terminal-facing adapter code.
    This is where stdout/stderr handling belongs.

  internal/usecase
    Application/usecase layer.
    Commands such as check, scan, review, edge, explain, graph, init.
    Usecases return structured results.

  internal/analyzer
    Project analysis and language-specific fact extraction.

    project
      Project model builder.

    lang/cpp
      C++ file scanning, include resolution, compile_commands support,
      include usage and runtime-risk heuristics.

    lang/go
      Baseline Go import graph support for dogfooding and mixed repositories.

    discovery
      Architecture discovery hints.

    rules
      Rule evaluation.

    graph
      Graph-related analysis helpers.

    suppressions
      Finding suppression support.

  internal/diff
    Diff logic for projects, dependencies, contracts and findings.

  internal/model
    Core project model:
    files, dependencies, symbols, findings, evidence.

  internal/reportmodel
    Review/report-facing model:
    ReviewResult, impact sections, contract impacts, risk summary.

  internal/render
    Rendering and artifact generation.

    check
      check output and check artifacts.

    review
      text/markdown/html review rendering.

    reviewbundle
      data-first review bundle:
      manifest.json, review.json, project-before.json, project-after.json,
      graph.json, tree.json, dependencies.json, findings.json,
      contracts.json, runtime.json, review-context.md, patchcourt.sarif.

    llmpack
      deterministic LLM context pack.

    sarif
      SARIF export.

  internal/serve/bundle
    Local bundle/API server used by `patchcourt open` and `patchcourt serve`.
    Serves viewer assets and review bundle endpoints.

  internal/source
    Source abstractions for roots, git refs, worktrees and saved state.

  internal/state
    Local baseline state support.

  internal/platform
    Infrastructure helpers: filesystem, git, logging, path matching.

web/viewer
  React viewer for interactive architecture review.

examples
  Demo projects.

docs
  Product and developer documentation.

analyzers/cpp-clang
  Future optional Clang-based precision backend.
```

## Dependency direction

PatchCourt follows a pragmatic Clean Architecture style:

```text
adapter/cli, serve/bundle
  -> usecase
  -> analyzer / diff / model / reportmodel
  -> platform abstractions where needed
```

Rules:

```text
- core analysis should return structured results;
- renderers turn structured results into text, JSON, HTML, SARIF, or LLM context;
- CLI and server adapters are responsible for user interaction;
- LLM is not a source of truth;
- SARIF is an integration/export layer, not the core model;
- HTML/viewer are clients of review bundle data, not the source of truth.
```

## Bundle-first review

The current review workflow is data-first.

A review with `--out DIR` writes a bundle:

```text
manifest.json
review.json
project-before.json
project-after.json
graph.json
tree.json
dependencies.json
findings.json
contracts.json
runtime.json
review-context.md
patchcourt.sarif
```

The local viewer reads these artifacts through the bundle API.

The intended product flow is:

```bash
patchcourt open /path/to/project --base origin/main --worktree --review-now
```

`patchcourt open` starts the local API/viewer, creates a review bundle if requested, and serves it to the viewer.

## Deterministic-first principle

PatchCourt should not depend on LLM output for correctness.

Correct flow:

```text
deterministic facts
  -> deterministic findings
  -> deterministic evidence
  -> optional LLM explanation
```

Incorrect flow:

```text
LLM reads repository
  -> invents architecture findings
```

LLM integration should be evidence-bound and should cite PatchCourt evidence IDs, files, dependencies, contracts or findings.

## Clang strategy

Clang is not a required dependency.

PatchCourt should stay useful without Clang:

```text
include graph
layer graph
contract diff MVP
runtime-risk heuristics
review bundle
local viewer
SARIF
LLM context
```

A future Clang backend may be added as an optional precision adapter, for example:

```bash
patchcourt review --cpp-backend clang
```

The core model, review pipeline and renderers must not depend on Clang.
