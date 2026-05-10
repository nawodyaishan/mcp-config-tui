.PHONY: help tidy tidy-check mod-verify fmt vet lint test build run dry-run apply verify snapshot tag release clean gitignore-check

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
	  '/total:/ { gsub(/%/,"",$$3); if ($$3+0 < 65.0) \
	    { print "FAIL: total coverage " $$3 "% is below 65% gate"; exit 1 } \
	    else { print "PASS: total coverage " $$3 "%" } }'

build:
	@bash scripts/build.sh

run:
	@bash scripts/run.sh

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

clean:
	@bash scripts/clean.sh
