#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

require_var V 'make release V=v1.2.0 MSG="release note"'
require_var MSG 'make release V=v1.2.0 MSG="release note"'
require_semver_tag "${V}"
require_missing_tag "${V}"

if [[ -n "$(git status --porcelain)" ]]; then
	printf "ERROR: release requires a clean worktree\n" >&2
	exit 1
fi

printf "Running release guard: make verify\n"
bash scripts/verify.sh
git tag -a "${V}" -m "${MSG}"
git push origin "${V}"
printf "Released %s. Monitor: https://github.com/nawodyaishan/usync/actions\n" "${V}"
