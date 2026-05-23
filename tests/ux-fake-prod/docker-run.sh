#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE="${USYNC_UX_IMAGE:-usync-ux-fake-prod:latest}"
ARTIFACTS="$ROOT/artifacts/ux-fake-prod"

mkdir -p "$ARTIFACTS"

docker build -f "$ROOT/tests/ux-fake-prod/Dockerfile" -t "$IMAGE" "$ROOT"
docker run --rm -t \
  -v "$ROOT/artifacts:/repo/artifacts" \
  "$IMAGE" "$@"

if [[ -s "$ARTIFACTS/flows/DM-P31.ansi" && ! -s "$ARTIFACTS/flows/DM-P31.txt" ]]; then
  cp "$ARTIFACTS/flows/DM-P31.ansi" "$ARTIFACTS/flows/DM-P31.txt"
fi

echo "Artifacts written to $ARTIFACTS"
