#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REPORT_DIR="${ROOT_DIR}/code/product/test-report"
ALLURE_RESULTS_DIR="${REPORT_DIR}/allure-results"
ALLURE_REPORT_DIR="${REPORT_DIR}/allure-report"

mkdir -p "${REPORT_DIR}"
mkdir -p "${ALLURE_RESULTS_DIR}"

run_module() {
  local name="$1"
  local dir="$2"

  echo "==> ${name}"
  (
    cd "${dir}"
    go test ./... -count=1 -json -coverprofile "${REPORT_DIR}/${name}-cover.out" | tee "${REPORT_DIR}/${name}-tests.json"
    "$(go env GOPATH)/bin/go-junit-report" -parser gojson -in "${REPORT_DIR}/${name}-tests.json" -out "${ALLURE_RESULTS_DIR}/${name}-junit.xml"
    go tool cover -html "${REPORT_DIR}/${name}-cover.out" -o "${REPORT_DIR}/${name}-coverage.html"
  )
}

run_module "sdk" "${ROOT_DIR}/code/sdk"
run_module "business" "${ROOT_DIR}/code/components/business"
run_module "data" "${ROOT_DIR}/code/components/data"

if command -v allure >/dev/null 2>&1; then
  allure generate "${ALLURE_RESULTS_DIR}" --clean -o "${ALLURE_REPORT_DIR}"
  echo "Allure report generated at ${ALLURE_REPORT_DIR}"
else
  echo "Allure CLI not found. Install with: npm install -g allure-commandline"
fi

echo "Reports saved to ${REPORT_DIR}"
