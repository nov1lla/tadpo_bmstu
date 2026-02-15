#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REPORT_DIR="${ROOT_DIR}/code/product/test-report"
ALLURE_RESULTS_DIR="${REPORT_DIR}/allure-results"
ALLURE_REPORT_DIR="${REPORT_DIR}/allure-report"
ALLURE_HISTORY_SRC="${REPORT_DIR}/allure-history"

mkdir -p "${REPORT_DIR}"
mkdir -p "${ALLURE_RESULTS_DIR}"
mkdir -p "${ALLURE_HISTORY_SRC}"
touch "${ALLURE_HISTORY_SRC}/.keep"

copy_history() {
  if [ -d "${ALLURE_HISTORY_SRC}" ]; then
    rm -rf "${ALLURE_RESULTS_DIR}/history"
    cp -R "${ALLURE_HISTORY_SRC}" "${ALLURE_RESULTS_DIR}/history"
  fi
}

run_module() {
  local stage="$1"
  local name="$2"
  local dir="$3"
  local tags="$4"
  local json_out="${REPORT_DIR}/${stage}-${name}-tests.json"

  echo "==> ${stage}/${name}"
  (
    cd "${dir}"
    if [ -n "${tags}" ]; then
      go test ./... -count=1 -json -tags "${tags}" | tee "${json_out}"
    else
      go test ./... -count=1 -json | tee "${json_out}"
    fi
  )
  return ${PIPESTATUS[0]}
}

write_skipped() {
  local stage="$1"
  local name="$2"
  local out="${ALLURE_RESULTS_DIR}/${stage}-${name}-skipped.xml"
  cat > "${out}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="${stage}-${name}" tests="1" skipped="1">
  <testcase classname="${stage}" name="${name}">
    <skipped message="skipped due to previous stage failure"/>
  </testcase>
</testsuite>
EOF
}

to_junit() {
  local stage="$1"
  local name="$2"
  local json_out="${REPORT_DIR}/${stage}-${name}-tests.json"
  local xml_out="${ALLURE_RESULTS_DIR}/${stage}-${name}-junit.xml"
  if [ -f "${json_out}" ]; then
    "$(go env GOPATH)/bin/go-junit-report" -parser gojson -in "${json_out}" -out "${xml_out}"
  fi
}

copy_history

unit_failed=0
run_module "unit" "sdk" "${ROOT_DIR}/code/sdk" "" || unit_failed=1
run_module "unit" "business" "${ROOT_DIR}/code/components/business" "" || unit_failed=1
run_module "unit" "data" "${ROOT_DIR}/code/components/data" "" || unit_failed=1

to_junit "unit" "sdk"
to_junit "unit" "business"
to_junit "unit" "data"

integration_failed=0
if [ "${unit_failed}" -eq 0 ]; then
  run_module "integration" "business" "${ROOT_DIR}/code/components/business" "integration" || integration_failed=1
  run_module "integration" "data" "${ROOT_DIR}/code/components/data" "integration" || integration_failed=1
  to_junit "integration" "business"
  to_junit "integration" "data"
else
  write_skipped "integration" "business"
  write_skipped "integration" "data"
fi

e2e_failed=0
if [ "${unit_failed}" -eq 0 ] && [ "${integration_failed}" -eq 0 ]; then
  run_module "e2e" "webapp" "${ROOT_DIR}/code/tests/e2e" "e2e" || e2e_failed=1
  to_junit "e2e" "webapp"
else
  write_skipped "e2e" "webapp"
fi

if command -v allure >/dev/null 2>&1; then
  allure generate "${ALLURE_RESULTS_DIR}" --clean -o "${ALLURE_REPORT_DIR}"
  rm -rf "${ALLURE_HISTORY_SRC}"
  mkdir -p "${ALLURE_HISTORY_SRC}"
  if [ -d "${ALLURE_REPORT_DIR}/history" ]; then
    cp -R "${ALLURE_REPORT_DIR}/history" "${ALLURE_HISTORY_SRC}"
  fi
  touch "${ALLURE_HISTORY_SRC}/.keep"
  echo "Allure report generated at ${ALLURE_REPORT_DIR}"
else
  echo "Allure CLI not found. Install with: npm install -g allure-commandline"
fi

echo "Reports saved to ${REPORT_DIR}"

if [ "${unit_failed}" -ne 0 ] || [ "${integration_failed}" -ne 0 ] || [ "${e2e_failed}" -ne 0 ]; then
  exit 1
fi
