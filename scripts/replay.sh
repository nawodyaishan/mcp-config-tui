#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

if [[ -z "${TRANSCRIPT:-}" ]]; then
	latest="$(ls -1t artifacts/journeys/usync-*.jsonl 2>/dev/null | head -n 1 || true)"
	if [[ -z "${latest}" ]]; then
		printf "ERROR: TRANSCRIPT=... is required and no transcripts found under artifacts/journeys/\n" >&2
		exit 2
	fi
	TRANSCRIPT="${latest}"
	printf "▶ replaying most recent transcript: %s\n" "${TRANSCRIPT}" >&2
fi

fixture="${FIXTURE:-happy-path-exa}"
args=("replay" "--against-fixture" "${fixture}")
if [[ "${EMIT_MATRIX:-}" == "1" || "${EMIT_MATRIX:-}" == "true" ]]; then
	args+=("--emit-matrix")
fi
args+=("${TRANSCRIPT}")

go run "${CMD}" "${args[@]}"
