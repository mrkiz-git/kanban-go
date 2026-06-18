#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${KANBA_IMAGE:-kanba-go:local}"
CONTAINER="${KANBA_CONTAINER:-kanba-go}"
HOST_PORT="${HOST_PORT:-${PORT:-8080}}"
CONTAINER_PORT="${CONTAINER_PORT:-8080}"
DATA_VOLUME="${KANBA_DATA_VOLUME:-${CONTAINER}-data}"
LOG_DIR="${KANBA_LOG_DIR:-${ROOT}/data/logs}"
LOG_FILE="${LOG_FILE:-${LOG_DIR}/kanba.log}"
PID_FILE="${KANBA_PID_FILE:-${ROOT}/data/kanba.pid}"
BINARY="${KANBA_BINARY:-${ROOT}/bin/kanba}"

usage() {
  cat <<EOF
Usage: $(basename "$0") [verbose|background|container]

  verbose     Run the local Go server in the foreground with debug logs on stdout.
  background  Run the local Go server as a background process; logs go to ${LOG_FILE}.
  container   Build and run the Podman container in the background (default).

Stop a background server or container with: ./scripts/stop.sh
EOF
}

start_local() {
  local mode="$1"

  mkdir -p "${ROOT}/data" "${LOG_DIR}" "${ROOT}/bin"

  echo "Building local server..."
  (cd "$ROOT" && go build -o "$BINARY" ./cmd/kanba)

  export HOST="${HOST:-0.0.0.0}"
  export PORT="${PORT:-8080}"

  case "$mode" in
    verbose)
      export LOG_LEVEL="${LOG_LEVEL:-debug}"
      unset LOG_FILE
      unset LOG_STDOUT
      echo "Starting Kanba in verbose mode on http://localhost:${PORT}/api/health"
      cd "$ROOT"
      exec "$BINARY"
      ;;
    background)
      if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        echo "Kanba is already running in the background (pid $(cat "$PID_FILE"))."
        exit 1
      fi

      export LOG_LEVEL="${LOG_LEVEL:-info}"
      export LOG_FILE="${LOG_FILE}"
      export LOG_STDOUT=0

      echo "Starting Kanba in background on http://localhost:${PORT}/api/health"
      echo "Logs: ${LOG_FILE}"

      cd "$ROOT"
      nohup "$BINARY" </dev/null >/dev/null 2>&1 &
      echo $! >"$PID_FILE"
      echo "Started Kanba (pid $(cat "$PID_FILE"))."
      ;;
    *)
      echo "Unknown local mode: $mode" >&2
      exit 1
      ;;
  esac
}

start_container() {
  local mode="${1:-background}"

  cd "$ROOT"

  echo "Building container image ${IMAGE}..."
  podman build -f Containerfile -t "$IMAGE" .

  podman rm -f "$CONTAINER" 2>/dev/null || true

  local -a run_args=(
    --name "$CONTAINER"
    -e "PORT=${CONTAINER_PORT}"
    -p "${HOST_PORT}:${CONTAINER_PORT}"
    -v "${DATA_VOLUME}:/app/data"
  )

  case "$mode" in
    verbose)
      run_args+=(-e "LOG_LEVEL=debug")
      echo "Starting container ${CONTAINER} in verbose mode on host port ${HOST_PORT}..."
      podman run --rm "${run_args[@]}" "$IMAGE"
      ;;
    background)
      run_args+=(
        -d
        -e "LOG_LEVEL=${LOG_LEVEL:-info}"
        -e "LOG_FILE=/app/data/logs/kanba.log"
        -e "LOG_STDOUT=0"
      )
      echo "Starting container ${CONTAINER} in background on host port ${HOST_PORT}..."
      podman run "${run_args[@]}" "$IMAGE"
      echo "Kanba running at http://localhost:${HOST_PORT}/api/health"
      echo "Container logs: podman logs -f ${CONTAINER}"
      echo "Persistent app logs: podman exec ${CONTAINER} cat /app/data/logs/kanba.log"
      ;;
    *)
      echo "Unknown container mode: $mode" >&2
      exit 1
      ;;
  esac
}

MODE="${1:-container}"

case "$MODE" in
  verbose|background)
    start_local "$MODE"
    ;;
  container)
    start_container background
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unknown mode: $MODE" >&2
    usage
    exit 1
    ;;
esac
