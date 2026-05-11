# LLM Context Pack

PatchCourt can generate a deterministic LLM context pack from review evidence.

The purpose is not to let an LLM inspect the whole repository from scratch. PatchCourt first collects facts, diffs them, and prepares a compact context for review assistance.

## Current artifact

When a review bundle is written, PatchCourt generates:

```text
review-context.md
```

Example:

```bash
patchcourt review \
  --base origin/main \
  --worktree \
  --root /path/to/project \
  --out /path/to/project/.patchcourt/out/latest
```

The generated bundle contains:

```text
/path/to/project/.patchcourt/out/latest/review-context.md
```

The local viewer/review workflow can also create the bundle:

```bash
patchcourt open /path/to/project \
  --base origin/main \
  --worktree \
  --review-now
```

## What the pack contains

The LLM context pack may include:

```text
review summary
risk summary
raw changed files
analyzed changed files
architecture impact
Worse / Better / Unchanged debt
dependency changes
layer edge changes
contract changes
contract impacts
runtime risks
findings with evidence
risk reasons
review questions
```

The exact content may evolve, but the principle stays stable:

```text
LLM receives curated PatchCourt facts, not the whole repository.
```

## Design principle

PatchCourt is deterministic-first.

Correct flow:

```text
PatchCourt facts
  -> PatchCourt findings
  -> PatchCourt evidence
  -> LLM summary/explanation/questions
```

Incorrect flow:

```text
LLM scans repository
  -> LLM invents architecture findings
```

## LLM rules

An LLM using the context pack should:

```text
use only provided evidence;
reference files, findings, contracts and dependencies from the pack;
separate facts from interpretation;
say when evidence is missing;
produce review questions instead of unsupported claims.
```

An LLM must not:

```text
invent files;
invent symbols;
invent dependencies;
invent findings;
claim certainty without evidence;
replace reviewer judgment;
make architecture decisions for the team.
```

## Useful prompts

Summary prompt:

```text
Summarize this PatchCourt review.
Focus on what got worse, what got better, and what is unchanged legacy debt.
Use only the evidence in the context pack.
```

Review questions prompt:

```text
Generate review questions from this PatchCourt context.
Each question must reference a finding, dependency edge, contract change or file from the context.
Do not invent missing evidence.
```

PR comment prompt:

```text
Draft a concise PR review comment from this PatchCourt context.
Include the top risks and concrete files/edges/contracts to inspect.
Keep old unchanged debt separate from new risk.
```

Finding explanation prompt:

```text
Explain why this finding matters.
Use only the evidence in the PatchCourt context.
If the context does not prove a bug, phrase the result as a review question.
```

## Future direction

The planned assistant layer should work directly on review bundles.

Possible commands:

```bash
patchcourt ask \
  --review .patchcourt/out/latest/review.json \
  "Why is this dependency risky?"
```

or, later:

```bash
patchcourt ask \
  --bundle .patchcourt/out/latest \
  "What should I review first?"
```

The assistant must be evidence-bound:

```text
answer
  -> evidence refs
  -> files / findings / edges / contracts
```

Answers without evidence should be treated as advisory only or rejected.
