#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

cp go.mod "${tmp_dir}/go.mod"
cp go.sum "${tmp_dir}/go.sum"

bash scripts/tidy.sh

if ! cmp -s go.mod "${tmp_dir}/go.mod" || ! cmp -s go.sum "${tmp_dir}/go.sum"; then
	printf "ERROR: go.mod or go.sum is not tidy. Run 'make tidy'.\n" >&2
	diff -u "${tmp_dir}/go.mod" go.mod || true
	diff -u "${tmp_dir}/go.sum" go.sum || true
	exit 1
fi
