# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

.PHONY: help build install test test-race lint fmt vet coverage selfcheck ci clean install-hooks e2e e2e-install

# Version embedded into the binary via -ldflags. Format: <branch>@<short-sha>[-dirty].
# Falls back to "dev" when not in a git checkout (e.g. an extracted source tarball).
AIWF_VERSION := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)@$(shell git describe --always --dirty 2>/dev/null)
LDFLAGS := -X main.Version=$(AIWF_VERSION)

help:
	@echo "Targets:"
	@echo "  build     - build the aiwf binary into ./bin/ (with embedded version)"
	@echo "  install   - go install the aiwf binary into \$$GOBIN (with embedded version)"
	@echo "  test      - run unit tests"
	@echo "  test-race - run unit tests with -race"
	@echo "  lint      - run golangci-lint"
	@echo "  fmt       - apply gofumpt formatting"
	@echo "  vet       - run go vet"
	@echo "  coverage  - run tests with coverage; print summary"
	@echo "  selfcheck - build and run 'aiwf doctor --self-check' end-to-end"
	@echo "  ci        - the full CI suite (vet + lint + test-race + coverage + selfcheck)"
	@echo "  install-hooks - point git at scripts/git-hooks/ via core.hooksPath (one-shot, idempotent)"
	@echo "  e2e-install - one-shot: install Playwright npm deps + Chromium browser"
	@echo "  e2e       - run the Playwright HTML-render browser tests (opt-in, requires e2e-install)"
	@echo "  clean     - remove build artifacts"

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/aiwf ./cmd/aiwf

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./cmd/aiwf

test:
	go test ./...

test-race:
	go test -race -parallel 8 ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofumpt -l -w .

coverage:
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -func=coverage.out | tail -n 1

# selfcheck builds the binary and drives every verb against a temp
# repo via `aiwf doctor --self-check`. Catches end-to-end regressions
# (broken commit trailers, hook installer drift, missing skills,
# `aiwf init` against a fresh git repo failing) that unit tests miss.
selfcheck: build
	./bin/aiwf doctor --self-check

ci: vet lint test-race coverage selfcheck

clean:
	rm -rf bin coverage.out

# install-hooks symlinks the tracked policy-lint hook into
# .git/hooks/pre-commit.local — the G45 chain target invoked by
# aiwf's chain-aware pre-commit hook. Idempotent: ln -sf overwrites
# any prior symlink and updates to scripts/git-hooks/pre-commit
# propagate immediately (the symlink resolves at hook-fire time).
#
# Run once after a fresh clone. The aiwf-managed hook itself is
# materialized by `aiwf init`/`aiwf update`, which write the
# chain-aware pre-commit hook at .git/hooks/pre-commit.
#
# Pre-G38 this target set `core.hooksPath = scripts/git-hooks`. That
# overrode git's default hooks dir, which collided with aiwf's own
# hook installer — see G48. The kernel now treats itself like any
# consumer: aiwf owns .git/hooks/<name>, kernel-specific logic lives
# at .git/hooks/<name>.local, both compose via G45's chain.
install-hooks:
	mkdir -p .git/hooks
	ln -sf ../../scripts/git-hooks/pre-commit .git/hooks/pre-commit.local
	@echo "Symlinked scripts/git-hooks/pre-commit -> .git/hooks/pre-commit.local"
	@echo "Run 'aiwf init' (if not already done) so the chain-aware aiwf hook calls it."

# Playwright browser-level tests for the HTML render. Opt-in: not
# run by `make ci` because they require Node + a 100MB Chromium
# install, and most contributors won't be touching the renderer's
# CSS. Run after `make e2e-install` (one-shot per machine).
#
# The fixture script (e2e/playwright/fixture.ts) builds the
# aiwf binary on each test process via `go build`, so there's no
# manual build step here.
e2e-install:
	cd e2e/playwright && npm install && npx playwright install chromium

e2e:
	cd e2e/playwright && npx playwright test
