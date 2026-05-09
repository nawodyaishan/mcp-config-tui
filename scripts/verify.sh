#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

bash scripts/mod-verify.sh
bash scripts/tidy-check.sh
bash scripts/vet.sh
bash scripts/lint.sh
bash scripts/test.sh
bash scripts/build.sh
bash scripts/gitignore-check.sh
