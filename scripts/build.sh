#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

mkdir -p ./bin
go build -ldflags="$(ldflags_value)" -o "${BIN}" "${CMD}"
