# PatchCourt v0.2.0-alpha

PatchCourt v0.2.0-alpha is focused on diff-aware architecture review for C++ patches.

PatchCourt helps answer one review question:

> Did this patch make the architecture better or worse?

It compares base and changed code, separates newly introduced risk from legacy architecture debt, and provides evidence down to files, include/import edges, layers, public contracts, findings, and review questions.

## Highlights

- Diff-aware architecture review for C++ patches.
- Architecture impact report: Worse / Better / Unchanged debt.
- Static `review.html` report.
- Machine-readable `review.json`.
- LLM-ready `review-context.md`.
- Layer dependency diff.
- Dependency edge diff.
- Public contract diff.
- Review questions for public contract changes.
- Camera-service bad/better demo.
- SARIF export for CI/code scanning integrations.
- GitHub Actions example.
- GitLab CI artifacts example.

## Main workflow

~~~bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --format json \
  --html-out .patchcourt/out/review.html \
  --llm-pack \
  --llm-pack-out .patchcourt/out/review-context.md \
  --sarif-out .patchcourt/out/patchcourt.sarif \
  > .patchcourt/out/review.json
~~~

## Demo

~~~bash
make camera-demo
make open-camera-demo
~~~

The demo generates reports for a bad patch and a better patch:

~~~text
.patchcourt/out/examples/camera-service/bad-review.html
.patchcourt/out/examples/camera-service/bad-review.json
.patchcourt/out/examples/camera-service/bad-context.md
.patchcourt/out/examples/camera-service/bad.sarif

.patchcourt/out/examples/camera-service/better-review.html
.patchcourt/out/examples/camera-service/better-review.json
.patchcourt/out/examples/camera-service/better-context.md
.patchcourt/out/examples/camera-service/better.sarif
~~~

## CI integration

PatchCourt can run in CI as a non-blocking review assistant.

Recommended alpha-mode behavior:

~~~text
- always generate review.html;
- always upload review artifacts;
- upload SARIF where supported;
- do not fail CI by default.
~~~

Blocking/gating options can be added later with explicit flags such as:

~~~text
--fail-on-risk high
--fail-on-new-policy-violation
~~~

## Current limitations

- C++ analysis is lightweight and does not use Clang AST yet.
- Include resolution quality depends on `compile_commands.json` or `.patchcourt.yaml`.
- Risk score is review prioritization, not a correctness verdict.
- SARIF is an integration/export format; `review.html`, `review.json`, and `review-context.md` remain the primary PatchCourt reports.
- Go support is baseline-level and mainly used for dogfooding and mixed repositories.
- False positives are possible and should be reviewed with the provided evidence.
