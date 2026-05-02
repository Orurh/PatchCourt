# Rule System

PatchCourt rules are evidence-based.

A rule should not directly print output. It should add findings to the project model.

A finding contains:

- id
- severity
- title
- evidence
- risk
- suggestion
- confidence

Current implemented rule group:

- architecture boundary checks based on configured layers and dependency edges
