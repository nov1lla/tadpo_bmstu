#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-}"
REPO_REF="${REPO_REF:-}"
RUN_TRAFFIC_CAPTURE="${RUN_TRAFFIC_CAPTURE:-0}"

if [ -n "${REPO_URL}" ]; then
  if [ ! -d "/workspace/.git" ]; then
    git clone "${REPO_URL}" /workspace
  fi
  if [ -n "${REPO_REF}" ]; then
    git -C /workspace fetch --all
    git -C /workspace checkout "${REPO_REF}"
  fi
fi

cd /workspace

set +e
./code/product/run_ci_tests.sh
TEST_STATUS=$?
set -e

if [ "${TEST_STATUS}" -eq 0 ] && [ "${RUN_TRAFFIC_CAPTURE}" -eq 1 ]; then
  ./code/product/e2e_traffic_capture.sh
fi

exit "${TEST_STATUS}"
