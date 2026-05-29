#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

NO_COLOR=1 TERM=xterm-256color go test ./...
