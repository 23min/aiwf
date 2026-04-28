# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

.PHONY: help build install test test-race lint fmt vet coverage ci clean

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
	@echo "  ci        - the full CI suite (vet + lint + test-race + coverage)"
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

ci: vet lint test-race coverage

clean:
	rm -rf bin coverage.out
