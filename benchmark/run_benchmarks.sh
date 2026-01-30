#!/usr/bin/env bash
set -euo pipefail

RUNS=${RUNS:-15}
START_AT=${START_AT:-1}
RESULTS_DIR=${RESULTS_DIR:-benchmark/results}
PROJECT_PREFIX=${PROJECT_PREFIX:-bench}
WEBAPP_CPUS=${WEBAPP_CPUS:-1.0}
WEBAPP_MEM=${WEBAPP_MEM:-1g}
POSTGRES_CPUS=${POSTGRES_CPUS:-1.0}
POSTGRES_MEM=${POSTGRES_MEM:-1g}
K6_CPUS=${K6_CPUS:-1.0}
K6_MEM=${K6_MEM:-512m}
CLEAN_DOCKER=${CLEAN_DOCKER:-1}
# Extra safety to prevent disk from filling up on long benchmark sessions.
# These options never touch benchmark results; they only clean Docker artifacts.
PRUNE_DOCKER=${PRUNE_DOCKER:-1}
MIN_FREE_GB=${MIN_FREE_GB:-5}

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VENV_DIR=${VENV_DIR:-"${ROOT_DIR}/benchmark/.venv"}

mkdir -p "${RESULTS_DIR}"

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
else
  COMPOSE_CMD=(docker-compose)
fi

if docker buildx version >/dev/null 2>&1; then
  BUILDKIT_AVAILABLE=1
else
  BUILDKIT_AVAILABLE=0
fi

if [ ! -x "${VENV_DIR}/bin/python" ]; then
  python3 -m venv "${VENV_DIR}"
fi

PYTHON="${VENV_DIR}/bin/python"

${PYTHON} - <<'PY' 2>/dev/null || ${PYTHON} -m pip install -r "${ROOT_DIR}/benchmark/scripts/requirements.txt"
import matplotlib
print("matplotlib available")
PY

find_free_port() {
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("", 0))
print(s.getsockname()[1])
s.close()
PY
}

free_gb() {
  # df output format: ... Available 1K-blocks ...; convert to GB.
  df -Pk "${ROOT_DIR}" | awk 'NR==2 {printf "%.2f\n", $4/1024/1024}'
}

ensure_free_space() {
  local free
  free=$(free_gb)
  # Compare floats via python to avoid bc dependency.
  "${PYTHON}" - <<PY
free = float("${free}")
min_free = float("${MIN_FREE_GB}")
raise SystemExit(0 if free >= min_free else 1)
PY
}

prune_docker_artifacts() {
  if [ "${PRUNE_DOCKER}" -ne 1 ]; then
    return 0
  fi

  # Safe-ish pruning: removes only *stopped* containers and *dangling* images.
  # This prevents the machine from slowly filling up with leftovers between runs.
  docker container prune -f >/dev/null 2>&1 || true
  docker image prune -f >/dev/null 2>&1 || true
}

END_AT=$((START_AT + RUNS - 1))
for i in $(seq -w "${START_AT}" "${END_AT}"); do
  if ! ensure_free_space; then
    echo "Not enough free disk space to continue (need >= ${MIN_FREE_GB}GB). Free now: $(free_gb)GB" >&2
    echo "Hint: run 'make bench-prune-docker' (or set PRUNE_DOCKER=1) and retry." >&2
    exit 1
  fi

  RUN_ID="run_${i}_$(date +%s)"
  PROJECT_NAME="${PROJECT_PREFIX}_${USER:-user}_${i}_${RANDOM}"
  WEBAPP_IMAGE="tadpo-bench-webapp:${RUN_ID}"
  WEBAPP_PORT=$(find_free_port)
  PROMETHEUS_PORT=$(find_free_port)

  RUN_DIR="${RESULTS_DIR}/${RUN_ID}"
  mkdir -p "${RUN_DIR}"
  chmod 777 "${RUN_DIR}"

  echo "==> ${RUN_ID}: building image ${WEBAPP_IMAGE}"
  if [ "${BUILDKIT_AVAILABLE}" -eq 1 ]; then
    DOCKER_BUILDKIT=1 docker build \
      -f "${ROOT_DIR}/docker/benchmark/Dockerfile" \
      --build-arg BENCH_RUN_ID="${RUN_ID}" \
      --cache-from type=local,src="${ROOT_DIR}/benchmark/.docker-cache" \
      --cache-to type=local,dest="${ROOT_DIR}/benchmark/.docker-cache",mode=max \
      -t "${WEBAPP_IMAGE}" \
      "${ROOT_DIR}"
  else
    echo "BuildKit/buildx not available; building without cache."
    DOCKER_BUILDKIT=0 docker build \
      -f "${ROOT_DIR}/docker/benchmark/Dockerfile" \
      --build-arg BENCH_RUN_ID="${RUN_ID}" \
      -t "${WEBAPP_IMAGE}" \
      "${ROOT_DIR}"
  fi

  export WEBAPP_IMAGE
  export WEBAPP_PORT
  export PROMETHEUS_PORT
  export POSTGRES_DB="ppo_bench_${i}"
  export POSTGRES_USER="bench"
  export POSTGRES_PASSWORD="bench"
  export WEBAPP_CPUS
  export WEBAPP_MEM
  export POSTGRES_CPUS
  export POSTGRES_MEM

  cleanup() {
    "${COMPOSE_CMD[@]}" -f "${ROOT_DIR}/docker/benchmark/docker-compose.bench.yml" -p "${PROJECT_NAME}" down -v >/dev/null 2>&1 || true
    if [ "${CLEAN_DOCKER}" -eq 1 ]; then
      docker image rm -f "${WEBAPP_IMAGE}" >/dev/null 2>&1 || true
    fi
    prune_docker_artifacts
  }
  trap cleanup EXIT

  "${COMPOSE_CMD[@]}" -f "${ROOT_DIR}/docker/benchmark/docker-compose.bench.yml" -p "${PROJECT_NAME}" up -d

  echo "==> ${RUN_ID}: waiting for webapp"
  ready=0
  for _ in $(seq 1 60); do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${WEBAPP_PORT}/api/users" | grep -q "405"; then
      ready=1
      break
    fi
    sleep 1
  done
  if [ "${ready}" -ne 1 ]; then
    echo "webapp did not start in time" >&2
    exit 1
  fi

  START_TS=$(date +%s)

  echo "==> ${RUN_ID}: running k6"
  docker run --rm \
    --network "${PROJECT_NAME}_default" \
    --cpus "${K6_CPUS}" \
    --memory "${K6_MEM}" \
    -e BASE_URL="http://webapp:8080" \
    -v "${ROOT_DIR}/benchmark/k6:/scripts:ro" \
    -v "${ROOT_DIR}/${RUN_DIR}:/results" \
    grafana/k6:0.48.0 \
    run --summary-export "/results/k6_summary.json" --out "json=/results/k6.json" "/scripts/scenarios.js"

  END_TS=$(date +%s)

  echo "==> ${RUN_ID}: fetching resources"
  "${ROOT_DIR}/benchmark/scripts/fetch_resources.py" \
    --prom-url "http://localhost:${PROMETHEUS_PORT}" \
    --project "${PROJECT_NAME}" \
    --start "${START_TS}" \
    --end "${END_TS}" \
    --output "${ROOT_DIR}/${RUN_DIR}/resources.json"

  echo "==> ${RUN_ID}: analyzing"
  "${ROOT_DIR}/benchmark/scripts/analyze_run.py" \
    --k6-json "${ROOT_DIR}/${RUN_DIR}/k6.json" \
    --k6-summary "${ROOT_DIR}/${RUN_DIR}/k6_summary.json" \
    --resources "${ROOT_DIR}/${RUN_DIR}/resources.json" \
    --output-dir "${ROOT_DIR}/${RUN_DIR}"

  cleanup
  trap - EXIT

done

echo "==> aggregating results"
"${ROOT_DIR}/benchmark/scripts/aggregate_runs.py" \
  --results-dir "${ROOT_DIR}/${RESULTS_DIR}" \
  --output-dir "${ROOT_DIR}/${RESULTS_DIR}/summary"
