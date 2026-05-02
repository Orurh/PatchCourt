# Architecture

PatchCourt is split into several areas:

- `core/go` — main Go implementation
- `internal/cli` — command-line adapter
- `internal/app` — application/usecase layer
- `internal/analysis` — project scanning, dependency resolution, rules, graphs, and discovery
- `internal/model` — shared data models
- `internal/output` — text, JSON, Markdown, DOT, Mermaid, and future LLM outputs
- `internal/platform` — infrastructure helpers such as git, logging, and path matching
- `analyzers/cpp-clang` — future deep C++ analyzer backend
- `examples` — demo projects
