#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER="${KANBA_CONTAINER:-kanba-go}"
PID_FILE="${KANBA_PID_FILE:-${ROOT}/data/kanba.pid}"
BINARY="${KANBA_BINARY:-${ROOT}/bin/kanba}"
PORT="${PORT:-8080}"

# shellcheck source=common.sh
source "${ROOT}/scripts/common.sh"

stopped=0

if [[ -f "$PID_FILE" ]]; then
  pid="$(cat "$PID_FILE")"
  if pid_is_kanba "$pid"; then
    kill "$pid"
    echo "Stopped local Kanba server (pid ${pid})."
    stopped=1
  else
    echo "Removing stale pid file for pid ${pid}."
  fi
  rm -f "$PID_FILE"
fi

if stop_kanba_on_port "$PORT"; then
  stopped=1
fi

if podman ps -q --filter "name=^${CONTAINER}$" | grep -q .; then
  podman stop "$CONTAINER" >/dev/null
  podman rm "$CONTAINER" >/dev/null 2>&1 || true
  echo "Stopped container ${CONTAINER}."
  stopped=1
fi

if [[ "$stopped" -eq 0 ]]; then
  echo "No running Kanba server or container found."
fi
