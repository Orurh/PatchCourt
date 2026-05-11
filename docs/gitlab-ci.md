# GitLab CI integration

PatchCourt can run in GitLab CI as a report-only architecture review step.

The portable path is to upload the generated review bundle as a normal GitLab artifact. SARIF can be added where GitLab SARIF ingestion is available.

## What PatchCourt produces

A review bundle contains:

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

Recommended artifacts:

```text
.patchcourt/out/latest/review.json
.patchcourt/out/latest/review-context.md
.patchcourt/out/latest/patchcourt.sarif
.patchcourt/out/latest/graph.json
.patchcourt/out/latest/dependencies.json
.patchcourt/out/latest/findings.json
```

## Example `.gitlab-ci.yml` from GitHub Release archive

```yaml
stages:
  - review

patchcourt_review:
  stage: review
  image: alpine:latest

  before_script:
    - apk add --no-cache curl tar git
    - curl -L -o patchcourt.tar.gz "https://github.com/Orurh/PatchCourt/releases/download/v0.2.2-alpha/patchcourt-v0.2.2-alpha-linux-amd64.tar.gz"
    - tar -xzf patchcourt.tar.gz
    - mv patchcourt-v0.2.2-alpha/patchcourt /usr/local/bin/patchcourt

  script:
    - mkdir -p .patchcourt/out/latest
    - git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"
    - |
      patchcourt review \
        --base "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
        --head HEAD \
        --root . \
        --out .patchcourt/out/latest \
        --format json \
        > .patchcourt/out/latest/review.json

  artifacts:
    when: always
    paths:
      - .patchcourt/out/latest/
    expire_in: 1 week
```

## Example from source

Use this when PatchCourt is built from the same repository checkout.

```yaml
stages:
  - review

patchcourt_review:
  stage: review
  image: golang:1.24

  before_script:
    - apt-get update
    - apt-get install -y git

  script:
    - cd core/go
    - go build -o ./bin/patchcourt ./cmd/patchcourt
    - cd ../..
    - mkdir -p .patchcourt/out/latest
    - git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"
    - |
      core/go/bin/patchcourt review \
        --base "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
        --head HEAD \
        --root . \
        --out .patchcourt/out/latest \
        --format json \
        > .patchcourt/out/latest/review.json

  artifacts:
    when: always
    paths:
      - .patchcourt/out/latest/
    expire_in: 1 week
```

## Optional SARIF report

Some GitLab installations can ingest SARIF reports.

If your GitLab version/tier supports it, add:

```yaml
  artifacts:
    when: always
    paths:
      - .patchcourt/out/latest/
    reports:
      sarif: .patchcourt/out/latest/patchcourt.sarif
    expire_in: 1 week
```

## Worktree review

For branch pipelines where the checkout itself is the current worktree:

```bash
patchcourt review \
  --base origin/main \
  --worktree \
  --root . \
  --out .patchcourt/out/latest \
  --format json \
  > .patchcourt/out/latest/review.json
```

For merge request pipelines where `HEAD` is the MR commit:

```bash
patchcourt review \
  --base "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
  --head HEAD \
  --root . \
  --out .patchcourt/out/latest \
  --format json \
  > .patchcourt/out/latest/review.json
```

## Configuration

`--config` is optional.

Default/auto-discovered configuration:

```bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --root . \
  --out .patchcourt/out/latest
```

Explicit project policy:

```bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --root . \
  --config .patchcourt.yaml \
  --out .patchcourt/out/latest
```

Use explicit config only after checking that `.patchcourt.yaml` matches the current repository structure.

## Notes

For GitLab, normal CI artifacts are the most portable integration path.

PatchCourt should be report-only by default during alpha. Blocking mode should be introduced only after the team defines explicit architecture policy and budgets.

Do not commit generated `.patchcourt/out` artifacts into the repository.
