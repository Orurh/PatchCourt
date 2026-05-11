# Rule System

PatchCourt rules are evidence-based.

A rule should not print output directly. It should produce structured findings that can be rendered by CLI, HTML, local viewer, SARIF and LLM context pack.

## Finding model

A finding should contain enough information for both humans and machines.

Typical fields:

```text
id
kind
severity
confidence
title / message
evidence
risk
suggestion
related dependency edge
related layer edge
related contract
```

Evidence is required for important findings.

## Evidence

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
symbol
contract id
dependency edge
message
```

Renderers should use structured evidence. They should not parse human-readable text to recover file paths or edges.

## Rule output

Rules should add findings to the project model or review result.

Correct:

```text
rule evaluates facts
  -> creates finding
  -> attaches evidence
  -> renderer displays finding
```

Incorrect:

```text
rule prints text directly
```

Core/analyzer/usecase packages should not write directly to stdout/stderr.

## Current rule groups

Implemented or partially implemented rule groups include:

```text
architecture boundary checks
  configured layer dependency violations and forbidden imports/includes.

discovery hints
  suspicious edges, bidirectional dependencies and architecture drift hints.

C++ runtime risk heuristics
  raw pointer capture into async/thread task;
  `this` captured into async callback;
  sleep/polling shutdown patterns.

contract review signals
  public contract changes;
  review questions for contract changes;
  test-like impact hints.

review risk scoring
  risk reasons and points used to prioritize review attention.
```

## Diff-aware review

Rules are most useful when combined with before/after diff.

PatchCourt should classify review impact as:

```text
Worse
  finding/dependency/risk added or made worse by the patch.

Better
  finding/dependency/risk removed or reduced by the patch.

Unchanged debt
  existing finding/dependency/risk that was already present before the patch.
```

A rule should not make legacy debt noisy by default. Review should focus on what the patch changed.

## Severity and confidence

Severity describes potential review impact:

```text
critical
high
medium
low
info
```

Confidence describes how strong the evidence is:

```text
high
medium
low
```

For heuristic runtime rules, confidence is especially important.

Example:

```text
high confidence
  obvious raw pointer captured into boost::asio::post lambda.

medium confidence
  `this` captured inside async-looking callback API.

low confidence
  async-looking class with mutable state, access model unclear.
```

## Runtime risk philosophy

Runtime risk findings are review questions, not proof of a bug.

Example finding:

```text
Raw pointer captured into deferred async/thread task.
```

Good review question:

```text
What guarantees that this object outlives the posted callback?
```

Bad output:

```text
This code definitely crashes.
```

PatchCourt should help reviewers inspect ownership, callback lifetime, cancellation and shutdown contracts.

## SARIF mapping

SARIF is an export layer.

Findings should map to SARIF rules/results using stable IDs and locations.

SARIF should not be the internal source of truth. The source of truth is the structured PatchCourt model and review bundle.

## LLM usage

LLM should consume findings and evidence, not create unsupported findings.

Allowed:

```text
summarize findings;
explain why a rule matters;
turn a finding into a review question;
draft a PR comment with evidence references.
```

Not allowed:

```text
invent evidence;
invent findings;
override deterministic rule output;
act as a hidden rule engine.
```

## Future direction

Potential future rule-system improvements:

```text
typed rule definitions;
stable text keys for finding messages;
localized finding text;
ADR-aware rules;
architecture budgets;
baseline/suppression workflow;
rule documentation generated from definitions;
evidence-bound LLM explanations.
```
