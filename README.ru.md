# PatchCourt

PatchCourt — это инструмент статического архитектурного ревью для C/C++ проектов.

Он строит include-level модель зависимостей, раскладывает файлы по архитектурным слоям, находит подозрительные связи и генерирует отчёты, удобные для разработчиков, CI и будущего LLM-assisted code review.

Текущий фокус проекта — анализ архитектуры C/C++ кода.

---

## Что делает PatchCourt

PatchCourt помогает отвечать на вопросы:

- Какие слои зависят друг от друга?
- Добавил ли патч новую запрещённую зависимость?
- Есть ли двунаправленные зависимости между слоями?
- Зависит ли domain-слой от внешнего implementation-кода?
- Есть ли потенциально неиспользуемые C++ include'ы?
- Какие конкретные файлы являются evidence для finding'а?
- Какие артефакты можно приложить к merge request?

PatchCourt разделён на несколько стадий:

```text
facts      -> include/import graph
discovery  -> metrics, clusters, suspicious dependencies
policy     -> explicit allowed dependencies
review     -> what changed in a patch
explain    -> why a finding exists
artifacts  -> markdown, json, dot, mermaid
```

---

## Текущий статус

Сейчас PatchCourt поддерживает:

- индексацию C/C++ файлов;
- граф C/C++ `#include` зависимостей;
- autodiscovery `compile_commands.json`;
- configured include paths;
- обработку system include paths;
- discovery архитектурных слоёв проекта;
- явную layer policy через `.patchcourt.yaml`;
- architecture violation findings;
- discovery hints;
- эвристику потенциально неиспользуемых include'ов;
- локальные suppressions через комментарии;
- scan reports в text/json/markdown;
- layer graph в dot/json/mermaid;
- before/after review reports;
- markdown review output;
- объяснение конкретного finding'а;
- команду `check`, которая одной командой создаёт стандартные артефакты проекта.

Go-анализ пока не является текущим фокусом.

---

## Установка

Из директории Go-реализации:

```bash
cd core/go
go build -o ./bin/patchcourt ./cmd/patchcourt
```

Запуск тестов:

```bash
go test ./...
```

---

## Быстрый старт

Запустить полный check проекта:

```bash
./bin/patchcourt check /path/to/project
```

Команда запишет стандартные артефакты в:

```text
/path/to/project/.patchcourt/out/
```

Пример вывода:

```text
PatchCourt check

Root: /path/to/project
Config: defaults
Out: /path/to/project/.patchcourt/out

Summary:
  production files: 101
  test files:       24
  dependencies:     686
  resolved:         301
  unresolved:       0
  findings:         9
  graph nodes:      10
  graph edges:      25

Artifacts:
  - project model: /path/to/project/.patchcourt/out/project-model.json
  - scan report: /path/to/project/.patchcourt/out/scan.md
  - layer graph json: /path/to/project/.patchcourt/out/layer-graph.json
  - layer graph dot: /path/to/project/.patchcourt/out/layer-graph.dot
  - layer graph mermaid: /path/to/project/.patchcourt/out/layer-graph.mmd
```

Сгенерировать SVG-граф:

```bash
dot -Tsvg /path/to/project/.patchcourt/out/layer-graph.dot \
  -o /path/to/project/.patchcourt/out/layer-graph.svg
```

---

## Команды

### `check`

Запускает scan + graph, пишет стандартные артефакты и печатает краткий summary.

```bash
./bin/patchcourt check /path/to/project
```

С явным конфигом и output directory:

```bash
./bin/patchcourt check /path/to/project \
  --config /path/to/project/.patchcourt.yaml \
  --out /tmp/patchcourt-out
```

Генерируемые артефакты:

```text
project-model.json
scan.md
layer-graph.json
layer-graph.dot
layer-graph.mmd
```

Это рекомендуемая команда для повседневного использования.

---

### `init`

Генерирует стартовый `.patchcourt.yaml`.

Baseline mode выводит текущие зависимости как разрешённые:

```bash
./bin/patchcourt init /path/to/project > .patchcourt.yaml
```

Strict mode обнаруживает слои, но оставляет `may_depend_on` пустым:

```bash
./bin/patchcourt init /path/to/project --strict > .patchcourt.yaml
```

Baseline mode полезен для legacy-проектов, где первая цель — не допустить новых архитектурных деградаций.

Strict mode полезен, когда нужно сразу увидеть существующие нарушения архитектурной политики.

Пример сгенерированного конфига:

```yaml
ignore:
  paths:
    - ".git/**"
    - "build/**"
    - "libs/**"
    - "third_party/**"
    - "external/**"
    - "generated/**"
    - "**/*.pb.h"
    - "**/*.pb.cc"

cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - "src"

layers:
  server:
    paths:
      - "src/server/**"
    may_depend_on:
      - domain

  domain:
    paths:
      - "src/domain/**"
    may_depend_on: []
```

---

### `scan`

Строит модель проекта и выводит findings.

Text output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format text
```

JSON output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format json > project-model.json
```

Markdown output:

```bash
./bin/patchcourt scan /path/to/project \
  --config .patchcourt.yaml \
  --format markdown > scan.md
```

Scan model содержит:

- files;
- file roles;
- symbols;
- dependencies;
- resolved/unresolved includes;
- external dependencies;
- layer assignments;
- findings;
- evidence.

---

### `graph`

Строит граф слоёв проекта.

DOT:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format dot > layer-graph.dot
```

Mermaid:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format mermaid > layer-graph.mmd
```

JSON:

```bash
./bin/patchcourt graph /path/to/project \
  --config .patchcourt.yaml \
  --format json > layer-graph.json
```

Сгенерировать SVG:

```bash
dot -Tsvg layer-graph.dot -o layer-graph.svg
```

---

### `review`

Сравнивает before/after модели проекта или before/after директории.

Review по двум директориям:

```bash
./bin/patchcourt review \
  --before-root /tmp/project-before \
  --after-root /tmp/project-after \
  --config .patchcourt.yaml \
  --format markdown
```

Review по двум заранее построенным моделям:

```bash
./bin/patchcourt review \
  --before before-model.json \
  --after after-model.json \
  --format text
```

Markdown review output рассчитан на merge requests:

```markdown
# PatchCourt Review

## Summary

- Risk: high, 11 points
- Dependency changes: 1
- Layer edge changes: 1
- Added findings: 1
- Added policy findings: 1

## Risk reasons

- +7 added high policy violation: architecture.server.cameras
- +1 dependency edge added: include|src/server/api_router.cc|src/cameras/camera_adapter_factory.h
- +3 layer edge added: server -> cameras
```

---

### `explain`

Объясняет конкретный finding.

Из root директории:

```bash
./bin/patchcourt explain architecture.server.cameras \
  --root /path/to/project \
  --config .patchcourt.yaml
```

Из модели:

```bash
./bin/patchcourt explain architecture.server.cameras \
  --model .patchcourt/out/project-model.json
```

Пример вывода:

```text
PatchCourt explain

Finding: architecture.server.cameras
Title:   Include-level architecture boundary violation
Kind:    policy_violation
Severity: high
Confidence: high

Risk:
  Layer "server" includes a header from layer "cameras", which is not allowed by .patchcourt.yaml.

Evidence:
  - src/server/api_router.cc: includes src/cameras/camera_adapter_factory.h, creating include dependency server -> cameras
```

---

## Findings

PatchCourt сейчас генерирует две основные категории findings.

### Policy violations

Policy violations появляются из явных правил `.patchcourt.yaml`.

Пример:

```yaml
layers:
  server:
    paths:
      - "src/server/**"
    may_depend_on:
      - domain

  cameras:
    paths:
      - "src/cameras/**"
    may_depend_on:
      - domain
```

Если `src/server/api_router.cc` включает `src/cameras/camera_adapter_factory.h`, PatchCourt сообщает:

```text
architecture.server.cameras
```

### Discovery hints

Discovery hints — это best-effort архитектурные запахи, найденные по dependency graph.

Примеры:

```text
discovery.bidirectional.application.cameras
discovery.bidirectional.domain.session
discovery.controllers.depends_on.server
discovery.domain.depends_on.application
discovery.cpp.unused_includes
```

Это не строгие policy violations, а подсказки для ревью.

---

## Suppressions

PatchCourt поддерживает локальное подавление finding'ов через комментарии.

Пример:

```cpp
// patchcourt:ignore architecture.server.cameras reason: legacy direct adapter include
#include "src/cameras/camera_adapter_factory.h"
```

Suppressed findings не попадают в отчёт для этого файла.

Suppressions стоит использовать аккуратно. Лучше исправлять архитектурные границы или обновлять policy, если зависимость действительно intentional.

---

## Разрешение C++ include'ов

PatchCourt разрешает include'ы через несколько источников:

- direct project paths;
- configured `cpp.include_paths`;
- обнаруженный `compile_commands.json`;
- system include paths;
- heuristic fallback.

Пример конфига:

```yaml
cpp:
  compile_commands:
    auto_discover: true
  include_paths:
    - "src"
    - "include"
```

PatchCourt автоматически ищет compile database в типичных местах:

```text
compile_commands.json
build/compile_commands.json
```

---

## Ignored paths

PatchCourt применяет default ignores, если конфиг не передан.

Default ignored paths включают:

```text
.git/**
build/**
cmake-build-debug/**
cmake-build-release/**
node_modules/**
vendor/**
libs/**
third_party/**
external/**
generated/**
**/*.pb.h
**/*.pb.cc
**/*.grpc.pb.h
**/*.grpc.pb.cc
```

Это убирает vendor/generated/build artifacts из архитектурного анализа проекта.

---

## File roles

PatchCourt классифицирует файлы как:

```text
production
test
generated
external
config
unknown
```

Architecture findings, layer graphs и review risk игнорируют зависимости, исходящие из test/generated/external файлов.

Цель — сфокусировать архитектурное ревью на production-коде.

---

## Пример workflow для C++ проекта

Сгенерировать baseline config:

```bash
./bin/patchcourt init /path/to/project > /path/to/project/.patchcourt.yaml
```

Запустить check:

```bash
./bin/patchcourt check /path/to/project \
  --config /path/to/project/.patchcourt.yaml
```

Открыть сгенерированный граф:

```bash
dot -Tsvg /path/to/project/.patchcourt/out/layer-graph.dot \
  -o /path/to/project/.patchcourt/out/layer-graph.svg

xdg-open /path/to/project/.patchcourt/out/layer-graph.svg
```

Объяснить top finding:

```bash
./bin/patchcourt explain discovery.bidirectional.application.cameras \
  --model /path/to/project/.patchcourt/out/project-model.json
```

---

## Пример review workflow

Подготовить before/after копии:

```bash
cp -a /path/to/project /tmp/project-before
cp -a /path/to/project /tmp/project-after
```

Внести изменение в `/tmp/project-after`.

Запустить review:

```bash
./bin/patchcourt review \
  --before-root /tmp/project-before \
  --after-root /tmp/project-after \
  --config /path/to/project/.patchcourt.yaml \
  --format markdown > review.md
```

`review.md` можно приложить к merge request или вставить в обсуждение code review.

---

## Design principles

PatchCourt строится вокруг нескольких принципов:

1. Facts first. Include graph собирается до применения policy.
2. Discovery is not policy. Подозрительные зависимости являются hints, пока они явно не запрещены.
3. Policy is explicit. `.patchcourt.yaml` задаёт разрешённые зависимости между слоями.
4. Review is evidence-based. У каждого finding'а должны быть конкретные file-level evidence.
5. Low noise matters. Test/generated/external/vendor/build файлы не должны доминировать в архитектурном отчёте.
6. C++ include dependencies are compile-time dependencies. Даже unused include'ы могут увеличивать coupling и build cost.

---

## Текущие ограничения

PatchCourt пока ранний инструмент.

Известные ограничения:

- C++ parsing лёгкий и синтаксический.
- Symbol usage detection эвристический.
- Macro-heavy и template-heavy код может давать false positives.
- Header-only библиотеки могут путать unused-include detection.
- Go-анализ пока не является текущим фокусом.
- `review --base main --head HEAD` ещё не реализован.
- Interactive UI / 3D graph viewer ещё не реализован.

---

## Roadmap

Ближайшие задачи:

- `check --format json`;
- более удобные CI-oriented exit codes;
- `review --base main --head HEAD`;
- LLM review context pack;
- более строгая config validation;
- улучшенные include resolution diagnostics;
- улучшение confidence для unused include;
- HTML/interactive graph report.

Возможное будущее:

- local web UI;
- VS Code extension;
- graph exploration с подсветкой findings;
- trend/baseline tracking;
- per-team architecture presets;
- Go package graph analysis.

---

## Development

Запустить все тесты:

```bash
go test ./...
```

Собрать CLI:

```bash
go build -o ./bin/patchcourt ./cmd/patchcourt
```

Запустить на самом проекте:

```bash
./bin/patchcourt check .
```

---

## Структура репозитория

```text
cmd/patchcourt                 CLI entrypoint

internal/app                   use cases: scan, graph, review, explain, check
internal/analysis              analyzers and domain logic
internal/config                config loading, validation, defaults
internal/model                 project model and shared data structures
internal/output/report         text/json/markdown/dot/mermaid renderers
internal/platform              filesystem, path, git, logging helpers
```

---

## License

TBD.
