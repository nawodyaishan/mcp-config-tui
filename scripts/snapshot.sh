#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

if ! command -v goreleaser >/dev/null 2>&1; then
	printf "ERROR: goreleaser is required; install it to build release snapshots.\n" >&2
	exit 1
fi

goreleaser release --snapshot --clean
