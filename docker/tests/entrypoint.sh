#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-}"
REPO_REF="${REPO_REF:-}"
RUN_TRAFFIC_CAPTURE="${RUN_TRAFFIC_CAPTURE:-0}"

if [ -n "${REPO_URL}" ]; then
  if [ ! -d "/workspace/.git" ]; then
    tmp_dir="$(mktemp -d)"
    git clone "${REPO_URL}" "${tmp_dir}"
    if [ -n "${REPO_REF}" ]; then
      git -C "${tmp_dir}" fetch --all
      git -C "${tmp_dir}" checkout "${REPO_REF}"
    fi
    if [ -d "/workspace" ]; then
      find /workspace -mindepth 1 -maxdepth 1 ! -name code -exec rm -rf {} +
      if [ -d "/workspace/code" ]; then
        find /workspace/code -mindepth 1 -maxdepth 1 ! -name product -exec rm -rf {} +
      fi
      if [ -d "/workspace/code/product" ]; then
        find /workspace/code/product -mindepth 1 -maxdepth 1 ! -name test-report -exec rm -rf {} +
      fi
    fi
    tar -C "${tmp_dir}" --exclude './code/product/test-report' -cf - . | tar -C /workspace -xf -
    rm -rf "${tmp_dir}"
  else
    if [ -n "${REPO_REF}" ]; then
      git -C /workspace fetch --all
      git -C /workspace checkout "${REPO_REF}"
    fi
  fi
fi

cd /workspace

mkdir -p /workspace/code/product/test-report/otel
mkdir -p /workspace/code/product/test-report/monitoring
chmod 777 /workspace/code/product/test-report/otel /workspace/code/product/test-report/monitoring || true

START_TS=$(date +%s)

set +e
./code/product/run_ci_tests.sh
TEST_STATUS=$?
set -e

END_TS=$(date +%s)

if command -v python3 >/dev/null 2>&1; then
  python3 /workspace/benchmark/scripts/fetch_resources.py \
    --prom-url "http://prometheus:9090" \
    --project "" \
    --start "${START_TS}" \
    --end "${END_TS}" \
    --output "/workspace/code/product/test-report/monitoring/resources.json" \
    >/workspace/code/product/test-report/monitoring/export.log 2>&1 || true
fi

if [ "${TEST_STATUS}" -eq 0 ] && [ "${RUN_TRAFFIC_CAPTURE}" -eq 1 ]; then
  ./code/product/e2e_traffic_capture.sh
fi

exit "${TEST_STATUS}"
