#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${KANBA_IMAGE:-kanba-go:local}"
CONTAINER="${KANBA_CONTAINER:-kanba-go}"
PORT="${PORT:-8080}"

cd "$ROOT"

echo "Building container image ${IMAGE}..."
podman build -f Containerfile -t "$IMAGE" .

podman rm -f "$CONTAINER" 2>/dev/null || true

echo "Starting container ${CONTAINER} on port ${PORT}..."
podman run -d \
  --name "$CONTAINER" \
  -p "${PORT}:8080" \
  "$IMAGE"

echo "Kanba running at http://localhost:${PORT}/api/health"
