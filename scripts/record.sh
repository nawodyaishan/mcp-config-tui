#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

args=("--record")
if [[ -n "${RECORD_PATH:-}" ]]; then
	args+=("--record-path" "${RECORD_PATH}")
fi
if [[ -n "${HOME_DIR:-}" ]]; then
	args+=("--home-dir" "${HOME_DIR}")
fi

printf "▶ launching usync with session recording\n" >&2
printf "  transcript: %s\n" "${RECORD_PATH:-artifacts/journeys/usync-<timestamp>.jsonl}" >&2
printf "  press [q] to quit and flush the transcript\n\n" >&2

go run "${CMD}" "${args[@]}"
