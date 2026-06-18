# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

.PHONY: help build install diag-aiwf test test-race test-pins lint fmt vet coverage coverage-gate selfcheck ci clean install-hooks e2e e2e-install copy-skill-fixture

# Version embedded into the binary via -ldflags. Format: <branch>@<short-sha>[-dirty].
# Empty (so version.Current falls back to buildinfo) when not in a git checkout
# (e.g. an extracted source tarball).
AIWF_VERSION := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)@$(shell git describe --always --dirty 2>/dev/null)
LDFLAGS := -X github.com/23min/aiwf/internal/version.Stamp=$(AIWF_VERSION)

# Test-binary wrapper that ad-hoc signs Darwin test binaries before exec'ing
# them, to dodge the macOS Sonoma 14.8.x syspolicyd crash on unsigned Mach-O
# headers. No-op on Linux/CI. See work/gaps/G-0133.
TEST_EXEC := $(CURDIR)/scripts/sign-and-run.sh

help:
	@echo "Targets:"
	@echo "  build     - build the aiwf binary into ./bin/ (with embedded version)"
	@echo "  install   - go install the aiwf binary into \$$GOBIN (with embedded version)"
	@echo "  diag-aiwf - build a worktree-scoped binary at ./bin/aiwf-diag and print its absolute path (G-0147)"
	@echo "  test      - run unit tests"
	@echo "  test-race - run unit tests with -race"
	@echo "  test-pins - run unit tests with -tags testpins (exercises Pin registry + bijection meta-test; M-0162/AC-2)"
	@echo "  lint      - run golangci-lint"
	@echo "  fmt       - apply gofumpt formatting"
	@echo "  vet       - run go vet"
	@echo "  coverage  - run tests with coverage; print summary"
	@echo "  coverage-gate - diff-scoped coverage audit vs origin/main (G-0067); run after committing"
	@echo "  selfcheck - build and run 'aiwf doctor --self-check' end-to-end"
	@echo "  ci        - the full CI suite (vet + lint + test-race + coverage + selfcheck)"
	@echo "  install-hooks - point git at scripts/git-hooks/ via core.hooksPath (one-shot, idempotent)"
	@echo "  e2e-install - one-shot: install Playwright npm deps + Chromium browser"
	@echo "  e2e       - run the Playwright HTML-render browser tests (opt-in, requires e2e-install)"
	@echo "  copy-skill-fixture SKILL=<name> - copy embedded ritual skill into testdata (deprecated; testdata fixtures removed per G-0182)"
	@echo "  clean     - remove build artifacts"

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/aiwf ./cmd/aiwf

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" ./cmd/aiwf
	@if [ "$$(uname)" = "Darwin" ]; then \
		bin="$${GOBIN:-$$(go env GOPATH)/bin}/aiwf"; \
		echo "Ad-hoc signing $$bin for Darwin syspolicyd resilience (G-0134)"; \
		codesign --sign - --force "$$bin" 2>/dev/null || echo "  codesign failed; manually sign with: codesign -s - -f $$bin"; \
	fi

# Build a worktree-scoped aiwf binary at ./bin/aiwf-diag and print its
# absolute path. The convention (per CLAUDE.md *Worktree binary discipline*):
# when diagnosing aiwf behavior against the current worktree source, run
# `make diag-aiwf` and invoke the printed path. Avoids the silent-stale
# PATH-binary trap that prompted G-0147.
diag-aiwf:
	@mkdir -p bin
	@CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/aiwf-diag ./cmd/aiwf
	@echo "Built: $(CURDIR)/bin/aiwf-diag"
	@echo "Invoke as: $(CURDIR)/bin/aiwf-diag <verb> [args...]"

test:
	go test -exec=$(TEST_EXEC) -parallel 8 ./...

test-race:
	go test -exec=$(TEST_EXEC) -race -parallel 8 ./...

# Run tests with -tags testpins enabled, which compiles in the
# internal/workflows/spec/branch/branchtest Pin registry and
# (post-AC-4) the bijection meta-test. Without the tag, both are
# excluded; this target is the local-dev path to exercise the
# pin-calling tests and the bijection invariants. CI runs are
# expected to include this in the same shape per AC-4.
#
# `-count=1` forces a fresh test-binary build (bypass Go's test
# cache). Reviewer R1-T3 observed a non-reproducible ghost
# violation in the AC-4 post-hook where the registry contained
# entries from a TestSabotage_* function that no longer existed in
# any source file. Most plausible cause: the prior `go test` run
# included a temporary sabotage test file whose binary was cached;
# a subsequent run after the file's deletion re-executed the
# cached binary because the cache hash hadn't invalidated. With
# `-count=1`, every invocation rebuilds the test binary from the
# current source tree, so deleted tests cannot ghost-replay.
test-pins:
	go test -exec=$(TEST_EXEC) -tags testpins -race -parallel 8 -count=1 ./...

vet:
	go vet ./...

# Lint cache is scoped per working tree (same rationale as the
# pre-push hook): the shared user-level cache replays issues carrying
# other worktrees' absolute paths, which fail open once that worktree
# is deleted. A pre-set GOLANGCI_LINT_CACHE is respected.
lint:
	GOLANGCI_LINT_CACHE="$${GOLANGCI_LINT_CACHE:-$$(git rev-parse --absolute-git-dir)/golangci-lint-cache}" golangci-lint run

fmt:
	gofumpt -l -w .

coverage:
	go test -exec=$(TEST_EXEC) -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -func=coverage.out | tail -n 1

# coverage-gate is the diff-scoped coverage audit (G-0067): every
# statement on a line changed since origin/main must be exercised by a
# test or annotated //coverage:ignore. It generates a fresh atomic-mode
# profile, resolves the base as the merge-base with origin/main, then
# runs the branch-coverage-audit policy with that profile + base. Run
# this after committing your work; it compares committed HEAD to the
# base, so uncommitted changes are not seen. CI runs the same gate in
# the test job.
coverage-gate:
	go test -exec=$(TEST_EXEC) -covermode=atomic -coverprofile=coverage.out -coverpkg=./internal/... ./...
	AIWF_COVERAGE_PROFILE="$(CURDIR)/coverage.out" \
	AIWF_COVERAGE_BASE="$$(git merge-base origin/main HEAD)" \
	go test -exec=$(TEST_EXEC) -run '^TestPolicy_BranchCoverageAudit$$' -count=1 ./internal/policies/

# selfcheck builds the binary and drives every verb against a temp
# repo via `aiwf doctor --self-check`. Catches end-to-end regressions
# (broken commit trailers, hook installer drift, missing skills,
# `aiwf init` against a fresh git repo failing) that unit tests miss.
selfcheck: build
	./bin/aiwf doctor --self-check

ci: vet lint test-race coverage selfcheck

clean:
	rm -rf bin coverage.out

# install-hooks symlinks the tracked kernel hooks into their
# .git/hooks/<name>.local chain targets — the G45 seam invoked by
# aiwf's chain-aware hooks. Idempotent: ln -sfn overwrites any
# prior symlink and updates to scripts/git-hooks/* propagate
# immediately (the symlink resolves at hook-fire time).
#
#   pre-commit — kernel policy lint + gitleaks path-leak gate.
#   pre-push   — golangci-lint boundary gate on pushed Go changes
#                (G-0179); runs before aiwf's `aiwf check`.
#
# Run once after a fresh clone. The aiwf-managed hooks themselves
# are materialized by `aiwf init`/`aiwf update`, which write the
# chain-aware hooks at .git/hooks/<name>.
#
# Pre-G38 this target set `core.hooksPath = scripts/git-hooks`. That
# overrode git's default hooks dir, which collided with aiwf's own
# hook installer — see G48. The kernel now treats itself like any
# consumer: aiwf owns .git/hooks/<name>, kernel-specific logic lives
# at .git/hooks/<name>.local, both compose via G45's chain.
install-hooks:
	@HOOKS_DIR=$$(git rev-parse --git-path hooks); \
	mkdir -p "$$HOOKS_DIR"; \
	ln -sfn ../../scripts/git-hooks/pre-commit "$$HOOKS_DIR/pre-commit.local"; \
	ln -sfn ../../scripts/git-hooks/pre-push "$$HOOKS_DIR/pre-push.local"; \
	echo "Symlinked scripts/git-hooks/pre-commit -> $$HOOKS_DIR/pre-commit.local"; \
	echo "Symlinked scripts/git-hooks/pre-push   -> $$HOOKS_DIR/pre-push.local"
	@echo "Run 'aiwf init' (if not already done) so the chain-aware aiwf hooks call them."

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

# Copy a SKILL.md fixture from this repo's testdata into the sibling
# ai-workflow-rituals repo at the path Claude Code expects. Closes
# the cross-repo flow's "construct a 6-segment path by hand" step
# per CLAUDE.md *Cross-repo plugin testing* and
# .devcontainer/README.md *Cross-repo plugin testing (rituals repo)*.
#
# Usage:  make copy-skill-fixture SKILL=aiwfx-start-epic
#
# Refuses (exit 2) with a clear stderr message if: SKILL is unset,
# the fixture doesn't exist, the sibling rituals repo isn't
# reachable at ../ai-workflow-rituals, or the destination skill
# directory isn't present in the rituals repo. No partial copies.
#
# G-0146 / E-0035 deferral closure (half-step). End-to-end smoke
# is deferred to a successor gap.
copy-skill-fixture:
	@test -n "$(SKILL)" || { echo "ERROR: SKILL=<name> required (e.g. make copy-skill-fixture SKILL=aiwfx-start-epic)" >&2; exit 2; }
	@test -f internal/policies/testdata/$(SKILL)/SKILL.md || { echo "ERROR: fixture missing: internal/policies/testdata/$(SKILL)/SKILL.md" >&2; exit 2; }
	@test -d ../ai-workflow-rituals || { echo "ERROR: sibling rituals repo not reachable at ../ai-workflow-rituals — see .devcontainer/README.md *Cross-repo plugin testing*" >&2; exit 2; }
	@TARGET=$$(find ../ai-workflow-rituals -path "*/skills/$(SKILL)/SKILL.md" -type f | head -n 1); \
	if [ -z "$$TARGET" ]; then \
		echo "ERROR: target skill dir not found under ../ai-workflow-rituals/plugins/*/skills/$(SKILL)/ — does the skill exist in the rituals repo?" >&2; \
		exit 2; \
	fi; \
	cp internal/policies/testdata/$(SKILL)/SKILL.md "$$TARGET"; \
	echo "Copied internal/policies/testdata/$(SKILL)/SKILL.md -> $$TARGET"; \
	echo "Next: cd ../ai-workflow-rituals && git diff && git commit + git push"

