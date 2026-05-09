#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

require_var V 'make tag V=v1.2.0 MSG="release note"'
require_var MSG 'make tag V=v1.2.0 MSG="release note"'
require_semver_tag "${V}"
require_missing_tag "${V}"

git tag -a "${V}" -m "${MSG}"
printf "Tagged %s locally. Run 'git push origin %s' to trigger the release workflow.\n" "${V}" "${V}"
