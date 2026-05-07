#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BIN="$ROOT/bin/patchcourt"
OUT_BASE="$ROOT/.patchcourt/out/bench-runtime"

PHOTON="${PHOTON:-/home/orurh/Документы/VSCode/work/photon_universal}"
LIBGOPRO="${LIBGOPRO:-/home/orurh/Документы/VSCode/work/photon_universal/libs/libgopro}"
PHOTON_LIB="${PHOTON_LIB:-/home/orurh/Документы/VSCode/work/photon_universal/libs/photon_lib}"

mkdir -p "$OUT_BASE" "$ROOT/bin"

echo "== Build PatchCourt =="
go build -o "$BIN" ./cmd/patchcourt

write_summary() {
  local name="$1"
  local out="$2"
  local summary="$out/runtime-summary.json"

  jq --arg project "$name" '
    {
      project: $project,
      files: (.files | length),
      dependencies: (.dependencies | length),
      findings: (.findings | length),
      layers: ([.files[]?.layer] | map(select(. != null and . != "")) | unique),
      runtime_findings: (
        [
          .findings[]?
          | select(.id | test("cpp\\.lifetime|cpp\\.shutdown"))
          | {
              id,
              severity,
              confidence,
              title,
              evidence_count: (.evidence | length)
            }
        ]
      ),
      runtime_evidence_by_file: (
        [
          .findings[]?
          | select(.id | test("cpp\\.lifetime|cpp\\.shutdown"))
          | .id as $id
          | .evidence[]?
          | {
              id: $id,
              file,
              line: .line_start,
              snippet
            }
        ]
      )
    }
  ' "$out/project-model.json" > "$summary"
}

run_check() {
  local name="$1"
  local project_root="$2"
  local config="${3:-}"
  local out="$OUT_BASE/$name"

  echo
  echo "============================================================"
  echo "Benchmark: $name"
  echo "Root:      $project_root"
  echo "Out:       $out"
  echo "============================================================"

  if [ ! -d "$project_root" ]; then
    echo "SKIP: root does not exist: $project_root"
    return 0
  fi

  rm -rf "$out"
  mkdir -p "$out"

  if [ -n "$config" ]; then
    "$BIN" check "$project_root" --config "$config" --out "$out"
  else
    "$BIN" check "$project_root" --out "$out"
  fi

  write_summary "$name" "$out"
}

echo
echo "== Prepare libgopro nested config =="
mkdir -p "$OUT_BASE/configs"

if [ -d "$LIBGOPRO" ]; then
  "$BIN" init "$LIBGOPRO" --preset nested-cpp \
    > "$OUT_BASE/configs/libgopro.generated-nested.patchcourt.yaml"
fi

echo
echo "== Prepare camera-service demo =="
python3 scripts/generate_camera_service_demo.py
if [ -f scripts/add_camera_service_runtime_risk_demo.py ]; then
  python3 scripts/add_camera_service_runtime_risk_demo.py
fi

run_check "photon_universal" "$PHOTON"
run_check "libgopro" "$LIBGOPRO" "$OUT_BASE/configs/libgopro.generated-nested.patchcourt.yaml"
run_check "photon_lib" "$PHOTON_LIB"
run_check "camera_service_after_bad" "$ROOT/examples/camera-service/after-bad" "$ROOT/examples/camera-service/.patchcourt.yaml"
run_check "camera_service_after_better" "$ROOT/examples/camera-service/after-better" "$ROOT/examples/camera-service/.patchcourt.yaml"

SUMMARY_MD="$OUT_BASE/runtime-summary.md"

{
  echo "# PatchCourt Runtime Benchmark"
  echo
  echo "Generated: $(date -Is)"
  echo
  echo "| Project | Files | Deps | Findings | Runtime findings |"
  echo "|---|---:|---:|---:|---|"

  for summary in "$OUT_BASE"/*/runtime-summary.json; do
    [ -f "$summary" ] || continue

    jq -r '
      . as $s
      | [
          .project,
          (.files | tostring),
          (.dependencies | tostring),
          (.findings | tostring),
          (
            .runtime_findings
            | map("`" + .id + "`=" + (.evidence_count | tostring))
            | join("<br>")
          )
        ]
      | "| " + .[0] + " | " + .[1] + " | " + .[2] + " | " + .[3] + " | " + .[4] + " |"
    ' "$summary"
  done

  echo
  echo "## Runtime evidence by project"
  echo

  for summary in "$OUT_BASE"/*/runtime-summary.json; do
    [ -f "$summary" ] || continue

    project="$(jq -r '.project' "$summary")"
    echo "### $project"
    echo

    jq -r '
      .runtime_evidence_by_file
      | group_by(.id)
      | .[]
      | "#### `" + .[0].id + "`\n"
        + (
          .[:20]
          | map("- `" + .file + ":" + (.line | tostring) + "` — " + (.snippet // ""))
          | join("\n")
        )
        + "\n"
    ' "$summary"

    echo
  done
} > "$SUMMARY_MD"

echo
echo "Benchmark summary:"
echo "  $SUMMARY_MD"
echo
cat "$SUMMARY_MD"
