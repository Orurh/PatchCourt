<div align="center">

# PatchCourt

### Diff-aware архитектурное ревью для C++ patch’ей

**Patch сделал архитектуру лучше или хуже?**

PatchCourt анализирует архитектурный эффект изменений в C++-проекте: отделяет **новый риск** от **старого legacy-долга** и показывает evidence до конкретных файлов, `#include`/`import` связей, слоёв, публичных контрактов, findings, runtime-risk сигналов и review-вопросов.

<br/>

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![C++](https://img.shields.io/badge/Focus-C++_Architecture-00599C?style=for-the-badge&logo=cplusplus&logoColor=white)](https://isocpp.org/)
[![SARIF](https://img.shields.io/badge/SARIF-Code_Scanning-6f42c1?style=for-the-badge)](https://sarifweb.azurewebsites.net/)
[![Alpha](https://img.shields.io/badge/status-v0.2.2--alpha-orange?style=for-the-badge)](#фокус-релиза-v022-alpha)

[English version](README.md)

</div>

---

## Зачем нужен PatchCourt

В больших C++-проектах архитектура обычно ломается не одним большим коммитом, а постепенным дрейфом.

Сначала API подключает конкретную vendor-реализацию. Потом `domain` начинает знать про `infrastructure`. Потом публичный интерфейс меняется без связанных тестов. Потом dependency cycle становится «ну он всегда был». А на review уже непонятно, patch принёс новый риск или просто задел старый долг.

PatchCourt нужен именно для этого места:

```text
Что стало хуже?
Что стало лучше?
Что было старым legacy-долгом?
Где конкретный evidence?
Что reviewer должен проверить первым?
```

PatchCourt — не компилятор, не clang-tidy, не security scanner, не замена человеку и не «AI reviewer».

Это deterministic evidence engine для архитектурного review.

```text
facts
  -> dependency graph
  -> architecture rules
  -> patch diff
  -> findings
  -> review bundle
  -> local viewer / JSON / LLM context / SARIF
```

---

## Главная идея

PatchCourt отвечает на один вопрос:

> Сделал ли этот C++ patch архитектуру лучше или хуже?

И отвечает не общими словами, а structured evidence:

| Сигнал | Пример |
|---|---|
| Новая запрещённая зависимость | `api -> cameras/sony` |
| Новый layer edge | `domain -> infrastructure` |
| Изменился публичный контракт | `method::ICameraAdapter::RunPreflight` |
| Нет связанных тестовых изменений | public interface changed, tests did not |
| Runtime architecture risk | `this` захвачен в async callback |
| Старый долг | cycle уже был до patch’а |
| Улучшение | запрещённая зависимость удалена |

Самое важное разделение:

```text
Worse          -> стало хуже из-за этого patch’а
Better         -> patch что-то улучшил
Unchanged debt -> долг уже был раньше
```

Вот это разделение и есть продуктовая ценность PatchCourt.

---

## Быстрый старт из исходников

```bash
git clone https://github.com/orurh/PatchCourt.git
cd PatchCourt/core/go

make build
make viewer-build
```

Запустить лёгкий анализ проекта с default/auto-discovered configuration:

```bash
./bin/patchcourt check . --out .patchcourt/out/check
```

Или через Makefile:

```bash
make check PROJECT=. CONFIG=
```

`CONFIG=` специально отключает явный `.patchcourt.yaml` и позволяет PatchCourt использовать default/auto-discovered configuration.

---

## Workflow с local viewer

Самый удобный review workflow сейчас — local viewer:

```bash
cd PatchCourt/core/go

./bin/patchcourt open . \
  --base origin/main \
  --worktree \
  --review-now
```

PatchCourt запускает локальный API/viewer server и создаёт initial review bundle.

Viewer assets ищутся автоматически:

```text
./viewer-dist
viewer-dist рядом с patchcourt binary
web/viewer/dist в dev checkout
```

Headless-режим:

```bash
./bin/patchcourt open . \
  --base origin/main \
  --worktree \
  --review-now \
  --no-browser
```

Проверки API, пока server запущен:

```bash
curl -s http://127.0.0.1:8787/api/health
curl -s http://127.0.0.1:8787/api/reviews/latest/graph | jq '{nodes: (.nodes | length), edges: (.edges | length)}'
```

---

## Workflow с review bundle

PatchCourt может записать self-contained review bundle:

```bash
./bin/patchcourt review \
  --base origin/main \
  --worktree \
  --root . \
  --out .patchcourt/out/latest
```

Bundle содержит:

```text
manifest.json
review.json
project-before.json
project-after.json
graph.json
runtime.json
tree.json
findings.json
contracts.json
dependencies.json
review-context.md
patchcourt.sarif
```

Этот bundle — source of truth для local viewer и integrations.

Старый HTML-first workflow больше не основной путь. Текущее направление продукта — **bundle/data-first + local viewer**.

---

## Release archive workflow

Собрать release archive с binary и viewer assets:

```bash
cd core/go
make release-archive RELEASE_VERSION=v0.2.2-alpha
```

Ожидаемая структура архива:

```text
patchcourt-v0.2.2-alpha/
  patchcourt
  viewer-dist/
    index.html
    assets/...
```

После распаковки:

```bash
./patchcourt open /path/to/project --review-now
```

Для release archive не нужно вручную передавать `--viewer-dir`.

---

## Встроенный camera-service demo

Запуск demo:

```bash
cd core/go
make camera-demo
```

Demo генерирует bad/better review outputs в:

```text
.patchcourt/out/examples/camera-service/
```

Актуальные bundle-oriented артефакты demo:

```text
bad-review.txt
bad-review.md
bad/review.json
bad/review-context.md
bad/patchcourt.sarif

better-review.txt
better-review.md
better/review.json
better/review-context.md
better/patchcourt.sarif
```

Открывать старые static demo reports имеет смысл только если они есть в checkout:

```bash
make open-camera-demo
```

---

## Что генерирует PatchCourt

| Артефакт | Зачем нужен |
|---|---|
| `manifest.json` | manifest bundle и schema version |
| `review.json` | machine-readable PatchCourt review result |
| `project-before.json` | модель проекта до patch’а |
| `project-after.json` | модель проекта после patch’а |
| `graph.json` | review graph для local viewer |
| `tree.json` | project tree для local viewer |
| `runtime.json` | runtime architecture risk report |
| `findings.json` | finding changes и evidence |
| `contracts.json` | public contract changes и impact |
| `dependencies.json` | dependency и layer-edge changes |
| `review-context.md` | deterministic LLM-ready context pack |
| `patchcourt.sarif` | CI/code scanning export |

SARIF — integration/export layer. Главный PatchCourt-артефакт сейчас — review bundle.

---

## LLM context pack

PatchCourt готовит deterministic context pack для LLM-assisted review:

```text
review-context.md
```

Внутри:

```text
patch summary
raw changed files
analyzed changed files
touched layers
architecture impact
contract changes
dependency changes
finding changes
runtime risks
risk reasons
review questions
```

Принцип:

```text
LLM может сжимать, объяснять и формулировать review questions.
LLM не должна выдумывать файлы, символы, зависимости или findings.
```

PatchCourt сначала собирает deterministic evidence. LLM работает поверх evidence.

---

## SARIF и CI

PatchCourt пишет SARIF внутрь review bundle:

```text
.patchcourt/out/latest/patchcourt.sarif
```

Минимальный CI flow:

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

      - name: Run PatchCourt review
        run: |
          patchcourt review \
            --base origin/main \
            --head HEAD \
            --root . \
            --out .patchcourt/out/latest

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: .patchcourt/out/latest/patchcourt.sarif

      - name: Upload PatchCourt artifacts
        uses: actions/upload-artifact@v4
        with:
          name: patchcourt-review-bundle
          path: .patchcourt/out/latest
```

Больше примеров:

```text
core/go/docs/ci/github-actions.md
core/go/docs/ci/gitlab-ci.md
```

Рекомендуемый alpha-режим:

```text
generate review bundle
upload SARIF where supported
upload review artifacts
do not fail CI by default
```

Blocking mode лучше включать явно, когда команда уже доверяет project policy.

---

## Project check mode

PatchCourt умеет анализировать текущее состояние проекта:

```bash
./bin/patchcourt check /path/to/project --out /path/to/project/.patchcourt/out/check
```

Типовые артефакты:

```text
project-model.json
scan.md
layer-graph.json
layer-graph.dot
layer-graph.mmd
```

Это полезно, чтобы:

```text
понять текущую структуру проекта
найти подозрительные layer edges
построить dependency graph
подготовить baseline config
```

---

## Edge drill-down

Граф сам по себе часто бесполезен, если нельзя провалиться в evidence.

PatchCourt позволяет объяснить конкретную layer dependency:

```bash
./bin/patchcourt edge \
  --model .patchcourt/out/check/project-model.json \
  api cameras
```

Пример формы вывода:

```text
PatchCourt edge

Edge: api -> cameras
Count: 3

Top source files:
  src/api/camera_routes.cc

Top target files:
  src/cameras/sony/sony_camera_manager.h

Dependencies:
  src/api/camera_routes.cc
    -> src/cameras/sony/sony_camera_manager.h [used]
```

Графы полезны. Evidence полезнее.

---

## Конфигурация

PatchCourt может стартовать без config.

```bash
./bin/patchcourt check . --out .patchcourt/out/check
./bin/patchcourt review --base origin/main --worktree --root . --out .patchcourt/out/latest
```

`.patchcourt.yaml` нужен, когда команда хочет явно описать architecture policy.

Пример:

```yaml
ignore:
  paths:
    - build/**
    - cmake-build-*/**
    - third_party/**
    - external/**
    - generated/**
    - "**/*.pb.cc"
    - "**/*.pb.h"

cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - src
    - include

layers:
  api:
    paths:
      - src/api/**
      - src/server/**
    may_depend_on:
      - controllers
      - domain

  controllers:
    paths:
      - src/controllers/**
    may_depend_on:
      - domain
      - cameras

  domain:
    paths:
      - src/domain/**
    may_depend_on: []

  cameras:
    paths:
      - src/cameras/**
    may_depend_on:
      - domain

forbidden_imports:
  - from_layer: api
    patterns:
      - src/cameras/sony/**
      - src/cameras/*_impl/**
```

Сгенерировать стартовый config:

```bash
./bin/patchcourt init /path/to/project > /path/to/project/.patchcourt.yaml
```

Для legacy-проектов лучше начинать с report-only review и добавлять policy постепенно.

---

## Что уже работает

| Область | Статус |
|---|---|
| C++ file indexing | works |
| C++ include graph | works |
| `compile_commands.json` discovery | works |
| configured include paths | works |
| Go import baseline | works |
| layer rules via `.patchcourt.yaml` | works |
| architecture findings | works |
| edge drill-down | works |
| before/after review | works |
| git base/head review | works |
| worktree review | works |
| review bundle output через `--out` | works |
| local viewer через `patchcourt open` | alpha |
| viewer asset auto-discovery | alpha |
| project tree / architecture graph viewer | alpha |
| public contract diff | alpha |
| runtime architecture risk signals | alpha |
| LLM context pack | alpha |
| SARIF export | alpha |

---

## Чем PatchCourt не является

PatchCourt — это не:

```text
C++ compiler frontend
clang-tidy replacement
proof of correctness
generic security scanner
Go linter replacement
full AI code reviewer
SaaS product в текущей alpha
```

PatchCourt — это:

```text
deterministic architecture-impact reviewer for patches
```

---

## Ограничения

PatchCourt сейчас в alpha-стадии.

Текущие ограничения:

- C++ анализ lightweight и пока без Clang AST.
- Качество include resolution зависит от `compile_commands.json`, структуры проекта или `.patchcourt.yaml`.
- CMake lightweight extraction не является полноценным CMake evaluator.
- Public contract extraction эвристический.
- Runtime risk rules намеренно осторожные и evidence-first.
- Risk score — это приоритизация review, а не verdict корректности.
- SARIF — export/integration layer, не core model.
- Go support — baseline-level, не главный market focus.
- Возможны false positives; их нужно проверять по evidence.

---

## Фокус релиза v0.2.2-alpha

`v0.2.2-alpha` сфокусирован на том, чтобы bundle/viewer workflow стал реально удобным:

```text
review bundle через --out
local viewer через patchcourt open
automatic viewer-dist discovery
release archive с binary + viewer-dist
полный review graph в open --review-now
SARIF и LLM context как bundle artifacts
make release-check
make release-archive
```

В этот релиз не входят:

```text
обязательный Clang backend
VS Code extension
SaaS/web platform
GitHub PR bot как основной workflow
deep cache
suppressions UI
широкое расширение Go/C++ risk rules
AI architect behavior
```

---

## Development

Из `core/go`:

```bash
make help
make ci
make viewer-build
make camera-demo
make open-self-nobrowser OPEN_REVIEW_NOW=true BASE=origin/main
make release-check
make release-archive RELEASE_VERSION=v0.2.2-alpha
```

Architecture guardrails проверяются тестами.

Core/usecase/analyzer пакеты должны возвращать structured results и не писать напрямую в stdout/stderr.

---

## License

Apache-2.0.
