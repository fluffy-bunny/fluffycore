#!/usr/bin/env bash
# Run the NATS integration tests against a throwaway local NATS server (docker compose).
#
# Usage:
#   ./scripts/test-nats.sh                 # go test ./nats/...
#   ./scripts/test-nats.sh -run TestNATS_  # extra args go to go test
#   KEEP_NATS=1 ./scripts/test-nats.sh     # leave the server running afterwards
#
# NATS_TEST_PORT overrides the host port (default 4299).
set -euo pipefail

cd "$(dirname "$0")/.."
compose=(docker compose -f docker/docker-compose.nats.yml)
port="${NATS_TEST_PORT:-4299}"

cleanup() {
  if [ -z "${KEEP_NATS:-}" ]; then
    "${compose[@]}" down -v >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

"${compose[@]}" up -d --wait
export NATS_TEST_URL="nats://localhost:${port}"
echo "NATS_TEST_URL=${NATS_TEST_URL}"
go test -count=1 -v "${@:-./nats/...}"
