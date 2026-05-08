APP := exa-mcp-manager
CMD := ./cmd/$(APP)
BIN := ./bin/$(APP)
GOCACHE := $(CURDIR)/.cache/go-build
GOMODCACHE := $(CURDIR)/.cache/go-mod
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)
GO := env $(GOENV) go

.PHONY: help tidy fmt test build run dry-run apply clean gitignore-check

help:
	@printf "Targets:\n"
	@printf "  make tidy            - sync module dependencies\n"
	@printf "  make fmt             - format Go sources\n"
	@printf "  make test            - run Go tests\n"
	@printf "  make build           - build the CLI into ./bin\n"
	@printf "  make run             - launch the TUI\n"
	@printf "  make dry-run KEYS_FILE=... - preview config changes\n"
	@printf "  make apply KEYS_FILE=...   - apply config changes\n"
	@printf "  make gitignore-check - validate ignore rules\n"
	@printf "  make clean           - remove local build artifacts\n"

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	mkdir -p ./bin
	$(GO) build -o $(BIN) $(CMD)

run:
	$(GO) run $(CMD)

dry-run:
ifndef KEYS_FILE
	$(error KEYS_FILE is required, example: make dry-run KEYS_FILE=~/Downloads/exa_keys.txt)
endif
	$(GO) run $(CMD) --keys-file $(KEYS_FILE) --dry-run

apply:
ifndef KEYS_FILE
	$(error KEYS_FILE is required, example: make apply KEYS_FILE=~/Downloads/exa_keys.txt)
endif
	$(GO) run $(CMD) --keys-file $(KEYS_FILE) --apply

gitignore-check:
	bash tests/gitignore_test.sh

clean:
	rm -rf ./bin ./coverage.out ./.cache
