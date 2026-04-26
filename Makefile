# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

.PHONY: help build test test-race lint fmt vet coverage ci clean

help:
	@echo "Targets:"
	@echo "  build     - build the aiwf binary into ./bin/"
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
	CGO_ENABLED=0 go build -o bin/aiwf ./tools/cmd/aiwf

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
