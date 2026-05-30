.PHONY: help tidy tidy-check mod-verify fmt vet lint test build run record replay dry-run apply verify snapshot tag release clean gitignore-check ux-matrix ux-fake-prod ux-explore

help:
	@bash scripts/help.sh

tidy:
	@bash scripts/tidy.sh

tidy-check:
	@bash scripts/tidy-check.sh

mod-verify:
	@bash scripts/mod-verify.sh

fmt:
	@bash scripts/fmt.sh

vet:
	@bash scripts/vet.sh

lint:
	@bash scripts/lint.sh

test:
	@bash scripts/test.sh

.PHONY: coverage-check
coverage-check:
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | awk \
	  '/total:/ { gsub(/%/,"",$$3); if ($$3+0 < 60.0) \
	    { print "FAIL: total coverage " $$3 "% is below 60% gate"; exit 1 } \
	    else { print "PASS: total coverage " $$3 "%" } }'

build:
	@bash scripts/build.sh

run:
	@bash scripts/run.sh

record: ## launch usync with session recording (transcript path overridable via RECORD_PATH=...)
	@bash scripts/record.sh

replay: ## replay a recorded transcript against a uxexplore fixture (TRANSCRIPT=... FIXTURE=happy-path-exa)
	@TRANSCRIPT="$(TRANSCRIPT)" FIXTURE="$(FIXTURE)" EMIT_MATRIX="$(EMIT_MATRIX)" bash scripts/replay.sh

dry-run:
	@KEYS_FILE="$(KEYS_FILE)" bash scripts/dry-run.sh

apply:
	@KEYS_FILE="$(KEYS_FILE)" bash scripts/apply.sh

verify:
	@bash scripts/verify.sh

snapshot:
	@bash scripts/snapshot.sh

tag:
	@V="$(V)" MSG="$(MSG)" bash scripts/tag.sh

release:
	@V="$(V)" MSG="$(MSG)" bash scripts/release.sh

gitignore-check:
	@bash scripts/gitignore-check.sh

ux-matrix:
	@USYNC_UX_MATRIX=1 go test -v ./pkg/tui -run TestDashboardFlowMatrix

ux-fake-prod:
	@bash tests/ux-fake-prod/docker-run.sh

ux-explore: ## run state-space explorer + coverage gates
	@go test ./pkg/uxexplore/...
	@go run ./cmd/ux-explore

clean:
	@bash scripts/clean.sh

.PHONY: test-e2e
test-e2e:
	@mkdir -p tests/e2e
	go test -v ./cmd/usync ./tests/e2e/...
