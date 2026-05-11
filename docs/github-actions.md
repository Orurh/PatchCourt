# GitHub Actions integration

PatchCourt can run in GitHub Actions as a report-only architecture review step.

Recommended alpha workflow:

```text
run PatchCourt review
write a review bundle
upload the bundle as CI artifact
upload SARIF where GitHub Code Scanning is enabled
avoid failing CI by default until project policy is explicit
```

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

The most useful artifacts for PR review are:

```text
review.json
review-context.md
patchcourt.sarif
graph.json
dependencies.json
findings.json
```

If you use the local viewer workflow, the same bundle can be inspected through `patchcourt open` locally.

## Example workflow from source

This workflow builds PatchCourt from the repository and runs a review on pull requests.

Create `.github/workflows/patchcourt.yml`:

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

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build PatchCourt
        run: |
          cd core/go
          go build -o ./bin/patchcourt ./cmd/patchcourt

      - name: Run PatchCourt review
        run: |
          mkdir -p .patchcourt/out/latest
          core/go/bin/patchcourt review \
            --base origin/main \
            --head HEAD \
            --root . \
            --out .patchcourt/out/latest \
            --format json \
            > .patchcourt/out/latest/review.json

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: .patchcourt/out/latest/patchcourt.sarif

      - name: Upload PatchCourt bundle
        uses: actions/upload-artifact@v4
        with:
          name: patchcourt-review-bundle
          path: .patchcourt/out/latest
```

## Example workflow from GitHub Release archive

Release archives should contain:

```text
patchcourt
viewer-dist/
```

Install the Linux archive and run a bundle review:

```yaml
name: PatchCourt

on:
  pull_request:

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

      - name: Install PatchCourt
        run: |
          curl -L -o patchcourt.tar.gz \
            https://github.com/Orurh/PatchCourt/releases/download/v0.2.2-alpha/patchcourt-v0.2.2-alpha-linux-amd64.tar.gz
          tar -xzf patchcourt.tar.gz
          sudo mv patchcourt-v0.2.2-alpha/patchcourt /usr/local/bin/patchcourt

      - name: Run PatchCourt review
        run: |
          mkdir -p .patchcourt/out/latest
          patchcourt review \
            --base origin/main \
            --head HEAD \
            --root . \
            --out .patchcourt/out/latest \
            --format json \
            > .patchcourt/out/latest/review.json

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: .patchcourt/out/latest/patchcourt.sarif

      - name: Upload PatchCourt bundle
        uses: actions/upload-artifact@v4
        with:
          name: patchcourt-review-bundle
          path: .patchcourt/out/latest
```

## Worktree review for branch builds

For CI jobs where the checked-out branch represents the current worktree, use:

```bash
patchcourt review \
  --base origin/main \
  --worktree \
  --root . \
  --out .patchcourt/out/latest \
  --format json \
  > .patchcourt/out/latest/review.json
```

For PR builds where `HEAD` is the PR commit, use:

```bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --root . \
  --out .patchcourt/out/latest \
  --format json \
  > .patchcourt/out/latest/review.json
```

## Configuration

`--config` is optional.

Without `--config`, PatchCourt uses default/auto-discovered configuration:

```bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --root . \
  --out .patchcourt/out/latest
```

With explicit policy:

```bash
patchcourt review \
  --base origin/main \
  --head HEAD \
  --root . \
  --config .patchcourt.yaml \
  --out .patchcourt/out/latest
```

Use explicit config only when `.patchcourt.yaml` matches the current repository structure.

## Notes

PatchCourt should run in report-only mode by default during alpha.

SARIF is an integration layer. The source of truth is the PatchCourt review bundle.

Do not commit generated `.patchcourt/out` artifacts into the repository.
