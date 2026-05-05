# GitHub Actions integration

PatchCourt can run on pull requests and upload:

- `review.html` as a build artifact;
- `review.json` as machine-readable PatchCourt output;
- `review-context.md` as an LLM-ready context pack;
- `patchcourt.sarif` to GitHub Code Scanning.

## Example workflow

Create `.github/workflows/patchcourt.yml`:

~~~yaml
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

      - name: Install PatchCourt
        run: |
          curl -L -o patchcourt.tar.gz \
            https://github.com/orurh/patchcourt/releases/download/v0.2.0-alpha/patchcourt-linux-amd64.tar.gz
          tar -xzf patchcourt.tar.gz
          sudo mv patchcourt /usr/local/bin/patchcourt

      - name: Run PatchCourt review
        run: |
          mkdir -p .patchcourt/out
          patchcourt review \
            --base origin/main \
            --head HEAD \
            --format json \
            --html-out .patchcourt/out/review.html \
            --llm-pack \
            --llm-pack-out .patchcourt/out/review-context.md \
            --sarif-out .patchcourt/out/patchcourt.sarif \
            > .patchcourt/out/review.json

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: .patchcourt/out/patchcourt.sarif

      - name: Upload PatchCourt artifacts
        uses: actions/upload-artifact@v4
        with:
          name: patchcourt-report
          path: .patchcourt/out
~~~

## Notes

PatchCourt should run in report-only mode by default during alpha.

Recommended artifacts to inspect:

~~~text
.patchcourt/out/review.html
.patchcourt/out/review.json
.patchcourt/out/review-context.md
.patchcourt/out/patchcourt.sarif
~~~

SARIF is an integration layer. The primary human report is `review.html`.
