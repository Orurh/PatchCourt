# GitLab CI integration

PatchCourt can run in GitLab CI and upload the generated review artifacts.

The portable path is to upload `.patchcourt/out/` as normal CI artifacts. SARIF can be added where GitLab SARIF ingestion is available.

## Example `.gitlab-ci.yml`

~~~yaml
stages:
  - review

patchcourt_review:
  stage: review
  image: alpine:latest

  before_script:
    - apk add --no-cache curl tar git
    - curl -L -o patchcourt.tar.gz "https://github.com/orurh/patchcourt/releases/download/v0.2.0-alpha/patchcourt-linux-amd64.tar.gz"
    - tar -xzf patchcourt.tar.gz
    - mv patchcourt /usr/local/bin/patchcourt

  script:
    - mkdir -p .patchcourt/out
    - git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"
    - |
      patchcourt review \
        --base "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
        --head HEAD \
        --format json \
        --html-out .patchcourt/out/review.html \
        --llm-pack \
        --llm-pack-out .patchcourt/out/review-context.md \
        --sarif-out .patchcourt/out/patchcourt.sarif \
        > .patchcourt/out/review.json

  artifacts:
    when: always
    paths:
      - .patchcourt/out/
    expire_in: 1 week
~~~

## Optional SARIF report

Some GitLab installations can ingest SARIF reports.

If your GitLab version/tier supports it, add:

~~~yaml
  artifacts:
    when: always
    paths:
      - .patchcourt/out/
    reports:
      sarif: .patchcourt/out/patchcourt.sarif
    expire_in: 1 week
~~~

## Notes

For GitLab, `review.html` artifacts are the most portable integration path.

Recommended artifacts:

~~~text
.patchcourt/out/review.html
.patchcourt/out/review.json
.patchcourt/out/review-context.md
.patchcourt/out/patchcourt.sarif
~~~
