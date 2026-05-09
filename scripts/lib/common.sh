#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

APP="usync"
CMD="./cmd/${APP}"
BIN="./bin/${APP}"
PKG_VERSION="github.com/nawodyaishan/usync/pkg/version"

export GOCACHE="${REPO_ROOT}/.cache/go-build"
export GOMODCACHE="${REPO_ROOT}/.cache/go-mod"
export GOLANGCI_LINT_CACHE="${REPO_ROOT}/.cache/golangci-lint"

cd "${REPO_ROOT}"

version_value() {
	git describe --tags --always --dirty 2>/dev/null || printf "dev"
}

commit_value() {
	git rev-parse --short HEAD 2>/dev/null || printf "none"
}

date_value() {
	date -u +%Y-%m-%dT%H:%M:%SZ
}

go_version_value() {
	go version | awk '{print $3}'
}

ldflags_value() {
	printf -- "-s -w -X %s.Version=%s -X %s.Commit=%s -X %s.Date=%s -X %s.GoVersion=%s" \
		"${PKG_VERSION}" "$(version_value)" \
		"${PKG_VERSION}" "$(commit_value)" \
		"${PKG_VERSION}" "$(date_value)" \
		"${PKG_VERSION}" "$(go_version_value)"
}

require_var() {
	local name="$1"
	local usage="$2"
	if [[ -z "${!name:-}" ]]; then
		printf "ERROR: %s is required. Usage: %s\n" "${name}" "${usage}" >&2
		exit 2
	fi
}

require_semver_tag() {
	local version="$1"
	if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		printf "ERROR: V must be a semver tag like v1.2.0 (got '%s')\n" "${version}" >&2
		exit 2
	fi
}

require_missing_tag() {
	local version="$1"
	if git rev-parse "${version}" >/dev/null 2>&1; then
		printf "ERROR: tag %s already exists\n" "${version}" >&2
		exit 1
	fi
}
