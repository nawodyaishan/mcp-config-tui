#!/usr/bin/env bash
set -euo pipefail

cat <<'HELP'
Targets:
  make tidy            - sync module dependencies
  make tidy-check      - verify module files are tidy
  make mod-verify      - verify downloaded module checksums
  make fmt             - format Go sources
  make vet             - run go vet across packages
  make lint            - run golangci-lint
  make test            - run Go tests
  make build           - build the CLI into ./bin
  make run             - launch the TUI
  make record [RECORD_PATH=...] [HOME_DIR=...] - launch TUI with session recording
  make replay [TRANSCRIPT=...] [FIXTURE=happy-path-exa] [EMIT_MATRIX=1] - replay a transcript
  make dry-run KEYS_FILE=... - preview config changes
  make apply KEYS_FILE=...   - apply config changes
  make snapshot        - build a local GoReleaser snapshot
  make tag V=v1.2.0 MSG=... - create a local annotated release tag
  make release V=v1.2.0 MSG=... - verify, tag, and push release tag
  make gitignore-check - validate ignore rules
  make ux-explore     - run state-space explorer tests
  make clean           - remove local build artifacts
HELP
