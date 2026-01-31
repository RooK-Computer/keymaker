#!/usr/bin/env bash
set -euo pipefail

PORT="${SIM_PORT:-8098}"
LISTEN=":${PORT}"
BASE="http://127.0.0.1:${PORT}"

cd "$(dirname "$0")/.."

# Build simulator first (also refreshes embedded web assets).
make build-sim >/dev/null

./bin/keymaker-sim --listen "${LISTEN}" --scenario retropie --dev >/tmp/keymaker-sim-validate.log 2>&1 &
SIM_PID=$!
cleanup() {
  kill "${SIM_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

KEYMAKER_API_BASE="${BASE}" python3 ./tools/validate_sim_api.py
