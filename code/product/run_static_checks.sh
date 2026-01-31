#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LINT_BIN="$(go env GOPATH)/bin/golangci-lint"

if [ ! -x "${LINT_BIN}" ]; then
  echo "golangci-lint not found at ${LINT_BIN}"
  echo "Install it with: make tools"
  exit 1
fi

echo "==> gofmt (check)"
unformatted=$(find "${ROOT_DIR}/code" -name '*.go' -not -path '*/vendor/*' -not -path '*/benchmark/*' -print0 | xargs -0 gofmt -l || true)
if [ -n "${unformatted}" ]; then
  echo "gofmt required for:"
  echo "${unformatted}"
  exit 1
fi

echo "==> golangci-lint (gofmt + gocyclo<=10)"
modules=(
  "${ROOT_DIR}/code/components/business"
  "${ROOT_DIR}/code/components/data"
  "${ROOT_DIR}/code/sdk"
  "${ROOT_DIR}/code/apps/webapp"
)

for mod in "${modules[@]}"; do
  echo "--> ${mod}"
  (cd "${mod}" && "${LINT_BIN}" run --config "${ROOT_DIR}/.golangci.yml" ./...)
done

echo "==> halstead (report)"
(cd "${ROOT_DIR}/tools/halstead" && go run . --paths "${ROOT_DIR}/code" --output "${ROOT_DIR}/code/product/test-report/static/halstead.json")

echo "Static checks OK"

