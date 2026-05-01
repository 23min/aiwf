# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

.PHONY: help build install test test-race lint fmt vet coverage selfcheck ci clean hooks

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
	@echo "  hooks     - install repo git hooks (pre-commit refreshes STATUS.md)"
	@echo "  clean     - remove build artifacts"

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/aiwf ./tools/cmd/aiwf

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./tools/cmd/aiwf

test:
	go test ./tools/...

test-race:
	go test -race ./tools/...

vet:
	go vet ./tools/...

lint:
	golangci-lint run ./tools/...

fmt:
	gofumpt -l -w ./tools

coverage:
	go test -coverprofile=coverage.out -coverpkg=./tools/internal/... ./tools/...
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

# hooks installs the tracked git hooks under scripts/git-hooks/ into
# .git/hooks/. Idempotent. Run once after a fresh clone (or whenever
# the tracked hook scripts change).
hooks:
	install -m 0755 scripts/git-hooks/pre-commit .git/hooks/pre-commit
	@echo "Installed: .git/hooks/pre-commit"
