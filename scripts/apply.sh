#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

require_var KEYS_FILE "make apply KEYS_FILE=~/Downloads/exa_keys.txt"

go run "${CMD}" --keys-file "${KEYS_FILE}" --apply
