#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUILD_DIR="${ROOT_DIR}/code/product/build"
TRAFFIC_DIR="${ROOT_DIR}/code/product/traffic"
STATIC_DIR="${BUILD_DIR}/static"
LOCAL_DIR="${BUILD_DIR}/localjson"
CONFIG_PATH="${BUILD_DIR}/data_local.json"

mkdir -p "${BUILD_DIR}" "${TRAFFIC_DIR}" "${STATIC_DIR}" "${LOCAL_DIR}"
printf '<html>ok</html>' > "${STATIC_DIR}/index.html"

cat > "${CONFIG_PATH}" <<EOF
{
  "backend": "local",
  "local": {
    "directory": "${LOCAL_DIR}"
  }
}
EOF

DATA_PLUGIN="${BUILD_DIR}/data.so"
BUSINESS_PLUGIN="${BUILD_DIR}/business.so"
WEBAPP_BIN="${BUILD_DIR}/webapp"

(cd "${ROOT_DIR}/code/components/data" && go build -tags plugin -buildmode=plugin -o "${DATA_PLUGIN}" ./cmd/plugin)
(cd "${ROOT_DIR}/code/components/business" && go build -tags plugin -buildmode=plugin -o "${BUSINESS_PLUGIN}" ./cmd/plugin)
(cd "${ROOT_DIR}/code/apps/webapp" && go build -o "${WEBAPP_BIN}" ./cmd/webapp)

PORT="$(python - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
ADDR="127.0.0.1:${PORT}"

export WEBAPP_ADDR="${ADDR}"
export DATA_PLUGIN_PATH="${DATA_PLUGIN}"
export BUSINESS_PLUGIN_PATH="${BUSINESS_PLUGIN}"
export DATA_SOURCE="${CONFIG_PATH}"
export WEBAPP_STATIC_DIR="${STATIC_DIR}"

"${WEBAPP_BIN}" &
WEBAPP_PID=$!

trap 'kill "${WEBAPP_PID}" >/dev/null 2>&1 || true' EXIT

sleep 0.5

TCPDUMP_FILE="${TRAFFIC_DIR}/e2e_traffic.pcap"
tcpdump -i lo port "${PORT}" -w "${TCPDUMP_FILE}" >/dev/null 2>&1 &
TCPDUMP_PID=$!
trap 'kill "${TCPDUMP_PID}" >/dev/null 2>&1 || true' EXIT

for _ in {1..20}; do
  if curl -s -o /dev/null -w "%{http_code}" "http://${ADDR}/api/users" | grep -q "405"; then
    break
  fi
  sleep 0.2
done

USER_JSON="$(curl -s -X POST "http://${ADDR}/api/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Traffic User"}')"
USER_ID="$(python - <<PY
import json
print(json.loads('''${USER_JSON}''')["id"])
PY
)"

curl -s "http://${ADDR}/api/users/${USER_ID}" >/dev/null

GAME_JSON="$(curl -s -X POST "http://${ADDR}/api/games" \
  -H "Content-Type: application/json" \
  -d '{"userId":"'"${USER_ID}"'","playerColor":"light","isPlayerFirst":true}')"
GAME_ID="$(python - <<PY
import json
print(json.loads('''${GAME_JSON}''')["game"]["id"])
PY
)"

curl -s -X POST "http://${ADDR}/api/games/${GAME_ID}/moves" \
  -H "Content-Type: application/json" \
  -d '{"start":{"row":5,"col":0},"steps":[{"row":4,"col":1}]}' >/dev/null

curl -s "http://${ADDR}/api/games/${GAME_ID}/moves" >/dev/null
curl -s -X DELETE "http://${ADDR}/api/games/${GAME_ID}" >/dev/null

kill "${TCPDUMP_PID}" >/dev/null 2>&1 || true
kill "${WEBAPP_PID}" >/dev/null 2>&1 || true

echo "Traffic capture saved to ${TCPDUMP_FILE}"
