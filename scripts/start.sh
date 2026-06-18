#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${KANBA_IMAGE:-kanba-go:local}"
CONTAINER="${KANBA_CONTAINER:-kanba-go}"
# HOST_PORT: port on the host machine. PORT is kept as a backward-compatible alias.
HOST_PORT="${HOST_PORT:-${PORT:-8080}}"
# CONTAINER_PORT must match Containerfile ENV PORT and the port the Go server listens on.
CONTAINER_PORT="${CONTAINER_PORT:-8080}"
DATA_VOLUME="${KANBA_DATA_VOLUME:-${CONTAINER}-data}"

cd "$ROOT"

echo "Building container image ${IMAGE}..."
podman build -f Containerfile -t "$IMAGE" .

podman rm -f "$CONTAINER" 2>/dev/null || true

echo "Starting container ${CONTAINER} on host port ${HOST_PORT} (container port ${CONTAINER_PORT})..."
podman run -d \
  --name "$CONTAINER" \
  -e "PORT=${CONTAINER_PORT}" \
  -p "${HOST_PORT}:${CONTAINER_PORT}" \
  -v "${DATA_VOLUME}:/app/data" \
  "$IMAGE"

echo "Kanba running at http://localhost:${HOST_PORT}/api/health"
