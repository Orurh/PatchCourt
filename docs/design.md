# PatchCourt Design

PatchCourt is a diff-aware architecture review tool for C++ projects.

The main product question is:

```text
Did this patch make the architecture better or worse?
```

PatchCourt does not try to replace a human architect or reviewer. It collects deterministic facts, compares before/after states, and turns a patch into an architecture case file.

## Product thesis

Large C++ systems rarely lose architecture in one commit. They drift gradually:

```text
API starts including vendor-specific implementation headers.
Domain starts depending on infrastructure details.
Public headers change without related tests.
Async callbacks capture raw pointers or `this`.
Shutdown order becomes implicit and fragile.
Reviewers cannot tell whether a patch added new risk or only touched old debt.
```

PatchCourt focuses on the patch, not on blaming the whole legacy codebase.

## Core review split

Every review should separate:

```text
Worse
  New risk introduced by this patch or old debt made worse.

Better
  Risk removed or old debt reduced by this patch.

Unchanged debt
  Existing problems that were already present and were not made worse.
```

This split is the core product value.

## Evidence-first model

Every important finding should have machine-readable evidence.

Evidence may include:

```text
file
line_start
line_end
snippet
from_file
to_file
from_layer
to_layer
dependency edge
symbol / contract id
finding id
confidence
```

UI, CLI, SARIF and LLM context should consume structured fields. They should not parse human-readable messages.

## Analysis pipeline

The intended pipeline is:

```text
ProjectSnapshot
  -> ProjectModel
  -> dependencies / symbols / findings / runtime sites
  -> diff
  -> ReviewResult
  -> review bundle
  -> renderers and clients
```

Review bundle is the current source of truth for interactive review.

## Current commands

Useful commands:

```bash
patchcourt check /path/to/project

patchcourt review \
  --base origin/main \
  --worktree \
  --root /path/to/project \
  --out /path/to/project/.patchcourt/out/latest

patchcourt open /path/to/project \
  --base origin/main \
  --worktree \
  --review-now

patchcourt edge \
  --model /path/to/project/.patchcourt/out/check/project-model.json \
  api cameras

patchcourt explain FINDING_ID \
  --model /path/to/project/.patchcourt/out/check/project-model.json
```

## Configuration philosophy

PatchCourt should be useful with zero config, then become more precise with policy.

Adoption levels:

```text
Level 0: no config
  Use auto/default analysis.

Level 1: suggested config
  Generate a starting .patchcourt.yaml.

Level 2: baseline
  Accept current architecture as legacy baseline.

Level 3: explicit policy
  Define layers, forbidden imports, ADR links and budgets.

Level 4: CI guardrails
  Fail only on explicitly configured new risk.
```

Default behavior should be report-only.

## C++ focus

C++ is the main focus because:

```text
include graph creates real compile-time coupling;
public headers are contract surface;
legacy C++ architecture debt is common;
AI-generated C++ is hard to review architecturally;
runtime lifecycle risks often matter more than folder structure.
```

Important C++ review dimensions:

```text
include/layer movement
public contract changes
header blast radius
ownership and async lifecycle
callback lifetime
thread boundaries
shutdown behavior
```

## Go support

Go support is useful for:

```text
dogfooding PatchCourt itself;
mixed repositories;
baseline import graph;
basic exported-symbol contract checks in the future.
```

Go is not the main market focus and PatchCourt should not become another Go linter.

## Runtime architecture risk

PatchCourt should detect review-relevant runtime architecture risks, not generic C++ bugs.

Examples:

```text
raw pointer captured into async task;
`this` captured into async callback;
callback may outlive owner;
shared mutable state in async object;
shutdown depends on sleep/polling;
thread boundary is unclear.
```

These findings should be phrased as evidence-backed review questions, not as proof of a bug.

## Local viewer

The local viewer is the main interactive UX.

It should help the reviewer inspect:

```text
overview
architecture graph
project tree
changed files
dependency movement
contract changes
runtime risks
findings
evidence
```

The viewer must read structured bundle/API data. It must not become the source of truth.

## LLM design

LLM integration is an assistant layer on top of PatchCourt evidence.

Allowed:

```text
summarize review;
explain a finding;
generate review questions;
draft PR comments from evidence;
answer questions about a review bundle.
```

Not allowed:

```text
invent files, symbols, dependencies or findings;
replace deterministic analysis;
decide architecture correctness;
act as the only source of severity/risk.
```

The long-term target is an evidence-bound assistant.
