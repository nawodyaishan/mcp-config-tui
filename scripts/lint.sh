#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

if ! command -v golangci-lint >/dev/null 2>&1; then
	printf "ERROR: golangci-lint is required; install it to run the local lint guard.\n" >&2
	exit 1
fi

golangci-lint run ./...
