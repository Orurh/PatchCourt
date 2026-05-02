# PatchCourt Design

PatchCourt is a diff-aware architecture and risk analyzer for Go/C++ codebases.

The core idea is to build a structured project model first, then use that model to detect architecture drift, dependency violations, risky patterns, and review-relevant evidence.

Current focus:

- repository inventory
- C++ include graph
- layer graph
- architecture rules
- baseline/strict config generation
- evidence-based reports
