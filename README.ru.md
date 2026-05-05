<div align="center">

# PatchCourt

### Diff-aware архитектурное ревью для C++ patch’ей

**Patch сделал архитектуру лучше или хуже?**

PatchCourt анализирует архитектурный эффект изменений в C++-проекте: отделяет **новый риск** от **старого legacy-долга** и показывает evidence до конкретных файлов, `#include`/`import` связей, слоёв, публичных контрактов, findings и review-вопросов.

<br/>

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![C++](https://img.shields.io/badge/Focus-C++_Architecture-00599C?style=for-the-badge&logo=cplusplus&logoColor=white)](https://isocpp.org/)
[![SARIF](https://img.shields.io/badge/SARIF-Code_Scanning-6f42c1?style=for-the-badge)](https://sarifweb.azurewebsites.net/)
[![Alpha](https://img.shields.io/badge/status-v0.2.0--alpha-orange?style=for-the-badge)](#фокус-релиза-v020-alpha)

[Русская версия](README.ru.md)
</div>

---

## Зачем нужен PatchCourt

В больших C++-проектах архитектура обычно ломается не взрывом, а дрейфом.

Сначала API подключает конкретную Sony-реализацию. Потом `domain` начинает знать про `infrastructure`. Потом публичный интерфейс меняется без тестов. Потом dependency cycle становится «ну он всегда был». А на review уже никто не понимает, кто принёс новый риск, а что было старым долгом.

PatchCourt нужен именно для этого места: показать, что изменил конкретный patch.

```text
Что стало хуже?
Что стало лучше?
Что было старым legacy-долгом?
Где конкретный evidence?
Что reviewer должен спросить в PR/MR?
```

PatchCourt — не компилятор, не clang-tidy, не замена человеку и не «AI reviewer».

Это deterministic evidence engine для архитектурного review.

```text
facts -> dependency graph -> architecture rules -> diff -> findings -> review artifacts
```

---

## Главная идея

PatchCourt отвечает на один вопрос:

> Сделал ли этот C++ patch архитектуру лучше или хуже?

И отвечает не общими словами, а структурированным evidence:

| Сигнал | Пример |
|---|---|
| Новая запрещённая зависимость | `api -> cameras/sony` |
| Новый layer edge | `domain -> infrastructure` |
| Изменился публичный контракт | `method::ICameraAdapter::RunPreflight` |
| Нет связанных тестовых изменений | public interface changed, tests did not |
| Старый долг | cycle уже был до patch’а |
| Улучшение | запрещённая зависимость удалена |

Самое важное — разделение:

```text
Worse          -> стало хуже из-за этого patch’а
Better         -> patch что-то улучшил
Unchanged debt -> долг уже был раньше
```

Вот это разделение и есть продуктовая ценность PatchCourt.

---

## Пример review

Плохой patch:

```cpp
// src/api/camera_routes.cc
#include "src/cameras/sony/sony_camera_manager.h"
```

PatchCourt может показать:

```text
Risk: HIGH

Worse:
  [HIGH] API layer now depends on Sony camera implementation
  [HIGH] Domain layer now depends on infrastructure
  [HIGH] Public interface changed: ICameraAdapter::RunPreflight
  [MEDIUM] Public contract changed without related test-like files

Better:
  none

Unchanged debt:
  existing unrelated architecture debt
```

И сразу дать evidence:

```text
Dependency:
  src/api/camera_routes.cc
    -> src/cameras/sony/sony_camera_manager.h

Layer edge:
  api -> cameras

Contract:
  method::ICameraAdapter::RunPreflight
```

PatchCourt не пытается доказать, что patch «правильный» или «неправильный».

Он делает архитектурный эффект patch’а видимым.

---

## Быстрый demo-сценарий

Склонировать проект и запустить camera-service demo:

```bash
git clone https://github.com/orurh/PatchCourt.git
cd PatchCourt/core/go

make camera-demo
```

Открыть HTML-отчёты:

```bash
make open-camera-demo
```

Demo генерирует bad/better отчёты:

```text
.patchcourt/out/examples/camera-service/bad-review.html
.patchcourt/out/examples/camera-service/bad-review.json
.patchcourt/out/examples/camera-service/bad-review.md
.patchcourt/out/examples/camera-service/bad-review.txt
.patchcourt/out/examples/camera-service/bad-context.md
.patchcourt/out/examples/camera-service/bad.sarif

.patchcourt/out/examples/camera-service/better-review.html
.patchcourt/out/examples/camera-service/better-review.json
.patchcourt/out/examples/camera-service/better-review.md
.patchcourt/out/examples/camera-service/better-review.txt
.patchcourt/out/examples/camera-service/better-context.md
.patchcourt/out/examples/camera-service/better.sarif
```

Смысл demo:

| Patch | Что должно быть видно |
|---|---|
| bad patch | новый architecture drift, высокий риск |
| better patch | меньше drift, ниже риск, часть проблем убрана |

---

## Что генерирует PatchCourt

| Артефакт | Зачем нужен |
|---|---|
| `review.html` | статичный человекочитаемый отчёт |
| `review.json` | machine-readable review result |
| `review.md` | markdown-версия отчёта |
| `review.txt` | вывод для терминала |
| `review-context.md` | context pack для LLM-review |
| `patchcourt.sarif` | экспорт для CI/code scanning |

SARIF — это integration/export layer.

Главные PatchCourt-артефакты:

```text
review.html
review.json
review-context.md
```

---

## Основной workflow

Review ветки относительно `origin/main`:

```bash
./bin/patchcourt review \
  --base origin/main \
  --head HEAD \
  --format json \
  --html-out .patchcourt/out/review.html \
  --llm-pack \
  --llm-pack-out .patchcourt/out/review-context.md \
  --sarif-out .patchcourt/out/patchcourt.sarif \
  > .patchcourt/out/review.json
```

Review текущего worktree относительно base ref:

```bash
./bin/patchcourt review \
  --base main \
  --worktree \
  --format json \
  --html-out .patchcourt/out/review.html \
  --llm-pack \
  --llm-pack-out .patchcourt/out/review-context.md \
  --sarif-out .patchcourt/out/patchcourt.sarif \
  > .patchcourt/out/review.json
```

Низкоуровневый режим `before/after`:

```bash
./bin/patchcourt review \
  --before-root /tmp/before \
  --after-root /tmp/after \
  --format markdown
```

---

## LLM context pack

PatchCourt умеет готовить компактный deterministic context pack для LLM-assisted review:

```bash
./bin/patchcourt review \
  --base origin/main \
  --head HEAD \
  --llm-pack \
  --llm-pack-out .patchcourt/out/review-context.md
```

Внутри context pack:

```text
patch summary
raw changed files
analyzed changed files
touched layers
architecture impact
contract changes
dependency changes
finding changes
risk reasons
review questions
```

Принцип:

```text
LLM может сжимать, объяснять и формулировать review questions.
LLM не должна выдумывать файлы, символы, зависимости или findings.
```

PatchCourt сначала собирает deterministic evidence. LLM работает поверх него.

---

## CI integration

PatchCourt можно запускать в CI как non-blocking architecture review assistant.

Рекомендуемый режим для alpha:

```text
generate review.html
upload review artifacts
upload SARIF where supported
do not fail CI by default
```

Примеры:

```text
core/go/docs/ci/github-actions.md
core/go/docs/ci/gitlab-ci.md
```

Минимальный GitHub Actions flow:

```yaml
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
```

Blocking mode лучше включать явно и отдельно, когда команда уже доверяет правилам проекта.

---

## Project check mode

PatchCourt умеет анализировать текущее состояние проекта:

```bash
./bin/patchcourt check /path/to/project
```

Типовые артефакты:

```text
.patchcourt/out/project-model.json
.patchcourt/out/scan.md
.patchcourt/out/layer-graph.json
.patchcourt/out/layer-graph.dot
.patchcourt/out/layer-graph.mmd
.patchcourt/out/report.html
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
  --model .patchcourt/out/project-model.json \
  api cameras
```

Пример формы вывода:

```text
Edge: api -> cameras
Count: 3

Top source files:
  src/api/camera_routes.cc

Top target files:
  src/cameras/sony/sony_camera_manager.h

Dependencies:
  src/api/camera_routes.cc
    -> src/cameras/sony/sony_camera_manager.h
```

Графы полезны. Evidence полезнее.

---

## Конфигурация

Архитектурные границы описываются в `.patchcourt.yaml`.

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

Сгенерировать стартовый конфиг:

```bash
./bin/patchcourt init /path/to/project > .patchcourt.yaml
```

Baseline mode полезен для legacy-проектов: принять текущие зависимости и не дать patch’ам ухудшать архитектуру дальше.

Strict mode полезен для greenfield или cleanup-работ: сразу подсветить существующие нарушения.

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
| public contract diff | alpha |
| test-like review questions | alpha |
| `review.html` | alpha |
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
web app
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
- Качество include resolution зависит от `compile_commands.json` или `.patchcourt.yaml`.
- CMake lightweight extraction не является полноценным CMake evaluator.
- Public contract extraction эвристический.
- Risk score — это приоритизация review, а не verdict корректности.
- SARIF — export/integration layer, не core model.
- Go support — baseline-level, не главный market focus.
- Возможны false positives; их нужно проверять по evidence.

---

## Фокус релиза v0.2.0-alpha

`v0.2.0-alpha` сфокусирован на:

```text
diff-aware C++ architecture review
review.html
review.json
review-context.md
patchcourt.sarif
camera-service bad/better demo
GitHub Actions / GitLab CI examples
release gates через make release-check
```

В этот релиз не входят:

```text
Clang backend
VS Code extension
web server
GitHub PR bot
GitLab native SAST JSON
deep cache
suppressions UI
широкое расширение Go/C++ risk rules
```

---

## Development

Из `core/go`:

```bash
make help
make ci
make camera-demo
make self-review BASE=HEAD
make release-check BASE=HEAD
```

Architecture guardrails проверяются тестами.

Core/usecase/analyzer пакеты должны возвращать structured results и не писать напрямую в stdout/stderr.

---

## License

TBD.
