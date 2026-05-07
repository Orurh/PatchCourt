SHELL := /usr/bin/env bash

BIN := ./bin/patchcourt
CMD := ./cmd/patchcourt

# Target project under analysis.
#
# Example:
#   make state PROJECT=/home/user/work/my_repo
PROJECT ?= /Users/ardislor/Documents/work/photon_universal

# PatchCourt project config.
# By default we expect it inside the analyzed project.
CONFIG ?= $(PROJECT)/.patchcourt.yaml

# Generated artifacts directory.
OUT ?= $(PROJECT)/.patchcourt/out
MODEL ?= $(OUT)/project-model.json
REPORT ?= $(OUT)/report.html
REVIEW_TEXT ?= $(OUT)/review.txt
REVIEW_MD ?= $(OUT)/review.md
REVIEW_JSON ?= $(OUT)/review.json
REVIEW_HTML ?= $(OUT)/review.html
LLM_PACK ?= $(OUT)/review-context.md
SARIF ?= $(OUT)/patchcourt.sarif
SELF_OUT ?= ./.patchcourt/out/self-review
SELF_SARIF ?= $(SELF_OUT)/patchcourt.sarif
CAMERA_DEMO_OUT ?= ./.patchcourt/out/examples/camera-service
DOT_FILE ?= $(OUT)/layer-graph.dot
SVG_FILE ?= $(OUT)/layer-graph.svg

# Git review refs.
#
# Examples:
#   make git-review PROJECT=/repo BASE=origin/main HEAD=HEAD
#   make git-review PROJECT=/repo BASE=develop HEAD=feature/foo
#   make worktree-review PROJECT=/repo BASE=origin/main
BASE ?= origin/main
HEAD ?= HEAD

GO ?= go
DOT ?= dot
OPEN ?= xdg-open

.PHONY: help
help:
	@echo "PatchCourt dev targets:"
	@echo ""
	@echo "  make fmt"
	@echo "      Format Go code in ./cmd and ./internal."
	@echo ""
	@echo "  make test"
	@echo "      Run the full Go test suite."
	@echo ""
	@echo "  make build"
	@echo "      Build PatchCourt binary into $(BIN)."
	@echo ""
	@echo "  make clean"
	@echo "      Remove ./bin."
	@echo ""
	@echo "Project analysis targets:"
	@echo ""
	@echo "  make check PROJECT=/path/repo"
	@echo "      Analyze PROJECT and write artifacts into PROJECT/.patchcourt/out."
	@echo ""
	@echo "  make state PROJECT=/path/repo"
	@echo "      Run check and save PROJECT/.patchcourt/state/latest as local baseline."
	@echo ""
	@echo "  make review PROJECT=/path/repo"
	@echo "      Compare saved state/latest with current working tree."
	@echo ""
	@echo "  make update PROJECT=/path/repo"
	@echo "      Review since state/latest, then replace state/latest with current state."
	@echo ""
	@echo "Git review targets:"
	@echo ""
	@echo "  make git-review PROJECT=/path/repo BASE=origin/main HEAD=HEAD"
	@echo "      Compare two git refs using temporary detached worktrees."
	@echo ""
	@echo "  make worktree-review PROJECT=/path/repo BASE=origin/main"
	@echo "      Compare BASE git ref with current dirty working tree."
	@echo ""
	@echo "  make worktree-review-html PROJECT=/path/repo BASE=origin/main"
	@echo "      Compare BASE with current working tree and write text, HTML, and LLM context pack."
	@echo ""
	@echo "  make review-html PROJECT=/path/repo"
	@echo "      Compare saved state/latest with current tree and write text, HTML, and LLM context pack."
	@echo ""
	@echo "  make self-review BASE=HEAD"
	@echo "      Review PatchCourt's own dirty worktree and write artifacts into ./.patchcourt/out/self-review."
	@echo ""
	@echo "Report/navigation targets:"
	@echo ""
	@echo "  make html PROJECT=/path/repo"
	@echo "      Open generated HTML report."
	@echo ""
	@echo "  make open-review PROJECT=/path/repo"
	@echo "      Open generated review HTML report."
	@echo ""
	@echo "  make svg PROJECT=/path/repo"
	@echo "      Render DOT layer graph to SVG and open it."
	@echo ""
	@echo "  make edge PROJECT=/path/repo FROM=domain TO=application"
	@echo "      Drill down into one layer edge using project-model.json."
	@echo ""
	@echo "  make explain PROJECT=/path/repo ID=discovery.domain.depends_on.application"
	@echo "      Explain one finding from project-model.json."
	@echo ""
	@echo "Smoke targets:"
	@echo ""
	@echo "  make smoke PROJECT=/path/repo"
	@echo "      Format, test, build, save state, and review since state."
	@echo ""
	@echo "  make ci"
	@echo "      Format, test, and build PatchCourt itself."
	@echo ""
	@echo "  make release-check"
	@echo "      Run CI, camera demo, self-review, and verify release artifacts."
	@echo ""
	@echo "Demo targets:"
	@echo ""
	@echo "  make camera-demo"
	@echo "      Generate the camera-service demo and write bad/better review artifacts."
	@echo ""
	@echo "  make open-camera-demo"
	@echo "      Open bad and better camera-service review HTML reports."
	@echo ""
	@echo "Optional overrides:"
	@echo "  CONFIG=/path/.patchcourt.yaml"
	@echo "  OUT=/path/.patchcourt/out"
	@echo "  MODEL=/path/project-model.json"
	@echo "  BASE=origin/main"
	@echo "  HEAD=HEAD"
	@echo "  OPEN=gio\\ open"

.PHONY: require-project
require-project:
	@test -n "$(PROJECT)" || (echo "PROJECT is required. Usage: make check PROJECT=/path/to/repo" && exit 2)

.PHONY: require-edge
require-edge:
	@test -n "$(FROM)" || (echo "FROM is required. Usage: make edge PROJECT=/path/repo FROM=domain TO=application" && exit 2)
	@test -n "$(TO)" || (echo "TO is required. Usage: make edge PROJECT=/path/repo FROM=domain TO=application" && exit 2)

.PHONY: require-finding
require-finding:
	@test -n "$(ID)" || (echo "ID is required. Usage: make explain PROJECT=/path/repo ID=discovery.domain.depends_on.application" && exit 2)

.PHONY: require-model
require-model:
	@test -f "$(MODEL)" || (echo "Project model not found: $(MODEL). Run: make check PROJECT=$(PROJECT)" && exit 2)

.PHONY: fmt
fmt:
	$(GO)fmt -w ./cmd ./internal

.PHONY: test
test:
	$(GO) test ./...

.PHONY: build
build:
	mkdir -p ./bin
	$(GO) build -o $(BIN) $(CMD)

.PHONY: clean
clean:
	rm -rf ./bin

.PHONY: ci
ci: fmt test build

.PHONY: check
check: build require-project
	$(BIN) check "$(PROJECT)" \
		--config "$(CONFIG)" \
		--out "$(OUT)"

.PHONY: state
state: build require-project
	$(BIN) check "$(PROJECT)" \
		--config "$(CONFIG)" \
		--out "$(OUT)" \
		--save-state

.PHONY: review
review: build require-project
	$(BIN) review \
		--since-last "$(PROJECT)" \
		--config "$(CONFIG)"

.PHONY: update
update: build require-project
	$(BIN) review \
		--since-last "$(PROJECT)" \
		--config "$(CONFIG)" \
		--update-state

.PHONY: git-review
git-review: build require-project
	$(BIN) review \
		--base "$(BASE)" \
		--head "$(HEAD)" \
		--root "$(PROJECT)" \
		--config "$(CONFIG)"

.PHONY: worktree-review
worktree-review: build require-project
	$(BIN) review \
		--base "$(BASE)" \
		--worktree \
		--root "$(PROJECT)" \
		--config "$(CONFIG)"


.PHONY: review-html
review-html: build require-project
	mkdir -p "$(OUT)"
	$(BIN) review \
		--since-last "$(PROJECT)" \
		--config "$(CONFIG)" \
		--format text \
		--llm-pack \
		--llm-pack-out "$(LLM_PACK)" \
		--html-out "$(REVIEW_HTML)" \
		--sarif-out "$(SARIF)" \
		> "$(REVIEW_TEXT)"

.PHONY: worktree-review-html
worktree-review-html: build require-project
	mkdir -p "$(OUT)"
	$(BIN) review \
		--base "$(BASE)" \
		--worktree \
		--root "$(PROJECT)" \
		--config "$(CONFIG)" \
		--format text \
		--llm-pack \
		--llm-pack-out "$(LLM_PACK)" \
		--html-out "$(REVIEW_HTML)" \
		--sarif-out "$(SARIF)" \
		> "$(REVIEW_TEXT)"

.PHONY: self-review
self-review: build
	mkdir -p "$(SELF_OUT)"
	$(BIN) review \
		--base "$(BASE)" \
		--worktree \
		--root "." \
		--format text \
		--llm-pack \
		--llm-pack-out "$(SELF_OUT)/review-context.md" \
		--html-out "$(SELF_OUT)/review.html" \
		--sarif-out "$(SELF_SARIF)" \
		> "$(SELF_OUT)/review.txt"
	$(BIN) review \
		--base "$(BASE)" \
		--worktree \
		--root "." \
		--format markdown \
		> "$(SELF_OUT)/review.md"
	$(BIN) review \
		--base "$(BASE)" \
		--worktree \
		--root "." \
		--format json \
		> "$(SELF_OUT)/review.json"

.PHONY: open-review
open-review: require-project
	@test -f "$(REVIEW_HTML)" || (echo "Review HTML not found: $(REVIEW_HTML). Run: make worktree-review-html PROJECT=$(PROJECT)" && exit 2)
	$(OPEN) "$(REVIEW_HTML)" >/dev/null 2>&1 &

.PHONY: html
html: require-project
	@test -f "$(REPORT)" || (echo "HTML report not found: $(REPORT). Run: make check PROJECT=$(PROJECT)" && exit 2)
	$(OPEN) "$(REPORT)" >/dev/null 2>&1 &

.PHONY: svg
svg: require-project
	@test -f "$(DOT_FILE)" || (echo "DOT graph not found: $(DOT_FILE). Run: make check PROJECT=$(PROJECT)" && exit 2)
	$(DOT) -Tsvg "$(DOT_FILE)" -o "$(SVG_FILE)"
	$(OPEN) "$(SVG_FILE)" >/dev/null 2>&1 &

.PHONY: edge
edge: build require-project require-edge require-model
	$(BIN) edge \
		--model "$(MODEL)" \
		"$(FROM)" "$(TO)"

.PHONY: explain
explain: build require-project require-finding require-model
	$(BIN) explain "$(ID)" \
		--model "$(MODEL)"

.PHONY: smoke
smoke: fmt test build state review

.PHONY: camera-demo
camera-demo: build
	python3 scripts/generate_camera_service_demo.py
	python3 scripts/add_camera_service_runtime_risk_demo.py
	mkdir -p "$(CAMERA_DEMO_OUT)"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-bad \
		--config examples/camera-service/.patchcourt.yaml \
		--format text \
		--llm-pack \
		--llm-pack-out "$(CAMERA_DEMO_OUT)/bad-context.md" \
		--html-out "$(CAMERA_DEMO_OUT)/bad-review.html" \
		--sarif-out "$(CAMERA_DEMO_OUT)/bad.sarif" \
		> "$(CAMERA_DEMO_OUT)/bad-review.txt"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-bad \
		--config examples/camera-service/.patchcourt.yaml \
		--format markdown \
		> "$(CAMERA_DEMO_OUT)/bad-review.md"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-bad \
		--config examples/camera-service/.patchcourt.yaml \
		--format json \
		> "$(CAMERA_DEMO_OUT)/bad-review.json"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-better \
		--config examples/camera-service/.patchcourt.yaml \
		--format text \
		--llm-pack \
		--llm-pack-out "$(CAMERA_DEMO_OUT)/better-context.md" \
		--html-out "$(CAMERA_DEMO_OUT)/better-review.html" \
		--sarif-out "$(CAMERA_DEMO_OUT)/better.sarif" \
		> "$(CAMERA_DEMO_OUT)/better-review.txt"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-better \
		--config examples/camera-service/.patchcourt.yaml \
		--format markdown \
		> "$(CAMERA_DEMO_OUT)/better-review.md"
	$(BIN) review \
		--before-root examples/camera-service/before \
		--after-root examples/camera-service/after-better \
		--config examples/camera-service/.patchcourt.yaml \
		--format json \
		> "$(CAMERA_DEMO_OUT)/better-review.json"
	@echo "Camera demo artifacts:"
	@echo "  $(CAMERA_DEMO_OUT)/bad-review.html"
	@echo "  $(CAMERA_DEMO_OUT)/better-review.html"
	@echo "  $(CAMERA_DEMO_OUT)/bad-context.md"
	@echo "  $(CAMERA_DEMO_OUT)/better-context.md"
	@echo "  $(CAMERA_DEMO_OUT)/bad.sarif"
	@echo "  $(CAMERA_DEMO_OUT)/better.sarif"

.PHONY: open-camera-demo
open-camera-demo:
	@test -f "$(CAMERA_DEMO_OUT)/bad-review.html" || (echo "Bad review HTML not found. Run: make camera-demo" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-review.html" || (echo "Better review HTML not found. Run: make camera-demo" && exit 2)
	$(OPEN) "$(CAMERA_DEMO_OUT)/bad-review.html" >/dev/null 2>&1 &
	$(OPEN) "$(CAMERA_DEMO_OUT)/better-review.html" >/dev/null 2>&1 &

.PHONY: release-check
release-check: ci camera-demo self-review
	@test -f "$(CAMERA_DEMO_OUT)/bad-review.html" || (echo "missing $(CAMERA_DEMO_OUT)/bad-review.html" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/bad-review.json" || (echo "missing $(CAMERA_DEMO_OUT)/bad-review.json" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/bad-review.md" || (echo "missing $(CAMERA_DEMO_OUT)/bad-review.md" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/bad-review.txt" || (echo "missing $(CAMERA_DEMO_OUT)/bad-review.txt" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/bad-context.md" || (echo "missing $(CAMERA_DEMO_OUT)/bad-context.md" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/bad.sarif" || (echo "missing $(CAMERA_DEMO_OUT)/bad.sarif" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-review.html" || (echo "missing $(CAMERA_DEMO_OUT)/better-review.html" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-review.json" || (echo "missing $(CAMERA_DEMO_OUT)/better-review.json" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-review.md" || (echo "missing $(CAMERA_DEMO_OUT)/better-review.md" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-review.txt" || (echo "missing $(CAMERA_DEMO_OUT)/better-review.txt" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better-context.md" || (echo "missing $(CAMERA_DEMO_OUT)/better-context.md" && exit 2)
	@test -f "$(CAMERA_DEMO_OUT)/better.sarif" || (echo "missing $(CAMERA_DEMO_OUT)/better.sarif" && exit 2)
	@test -f "$(SELF_OUT)/review.html" || (echo "missing $(SELF_OUT)/review.html" && exit 2)
	@test -f "$(SELF_OUT)/review.json" || (echo "missing $(SELF_OUT)/review.json" && exit 2)
	@test -f "$(SELF_OUT)/review.md" || (echo "missing $(SELF_OUT)/review.md" && exit 2)
	@test -f "$(SELF_OUT)/review.txt" || (echo "missing $(SELF_OUT)/review.txt" && exit 2)
	@test -f "$(SELF_OUT)/review-context.md" || (echo "missing $(SELF_OUT)/review-context.md" && exit 2)
	@test -f "$(SELF_SARIF)" || (echo "missing $(SELF_SARIF)" && exit 2)
	@echo "PatchCourt release-check passed."

.PHONY: bench-runtime
bench-runtime:
	./scripts/bench_runtime.sh
