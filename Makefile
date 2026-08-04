# Convenience targets for ai-workflow development.
# CI runs `make ci`; everything else is for local dev.

# Targets here are ordered pipelines over shared files, not independent
# units of work: coverage, test-cov and coverage-gate all write the same
# coverage.out, and `ci` gates on the profile test-cov produces. Under -j
# make would start those concurrently, and the gate would read whatever
# coverage.out happened to be on disk — reporting green off a stale
# profile. Nothing here benefits from parallel make anyway; the suite
# already fans out internally via `go test -parallel 8`.
.NOTPARALLEL:

.PHONY: help build install diag-aiwf test check-fast test-race test-pins lint fmt vet coverage test-cov coverage-gate coverage-gate-only comment-history-audit growth-report mutate-diff selfcheck ci clean install-hooks e2e e2e-install stress stress-tests

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
	@echo "  check-fast - inner-loop gate: vet + lint + test (no race/coverage/selfcheck); run before each commit"
	@echo "  test-race - run unit tests with -race"
	@echo "  test-pins - run unit tests with -tags testpins (exercises Pin registry + bijection meta-test; M-0162/AC-2)"
	@echo "  lint      - run golangci-lint"
	@echo "  fmt       - apply gofumpt formatting"
	@echo "  vet       - run go vet"
	@echo "  coverage  - run tests with coverage; print summary"
	@echo "  test-cov  - combined race+coverage pass (one suite run); what 'ci' uses"
	@echo "  coverage-gate - diff-scoped coverage audit vs origin/main (G-0067); builds its own profile"
	@echo "  coverage-gate-only - the same gates against an existing coverage.out (what 'ci' uses)"
	@echo "  comment-history-audit - whole-tree scan for comments narrating a superseded state"
	@echo "  growth-report - snapshot the growth metrics docs/design/growth.md tracks (read-only; GROWTH_BASELINE=<rev> for a delta)"
	@echo "  mutate-diff - advisory diff-scoped mutation test: gremlins on internal/ packages changed vs origin/main (G-0267)"
	@echo "  selfcheck - build and run 'aiwf doctor --self-check' end-to-end"
	@echo "  ci        - the pre-push/CI gate (vet + lint + test-cov + coverage-gate-only + selfcheck); run once before pushing, not per commit"
	@echo "  install-hooks - symlink scripts/git-hooks/ into the .local hook chain (one-shot, idempotent)"
	@echo "  e2e-install - one-shot: install Playwright npm deps + Chromium browser"
	@echo "  e2e       - run the Playwright HTML-render browser tests (opt-in, requires e2e-install)"
	@echo "  stress    - run the on-demand correctness stress harness's whole scenario catalog (opt-in, dev-only; override STRESS_REPEAT=N)"
	@echo "  stress-tests - run the concurrency/fault scenario Go tests behind the 'stress' build tag (opt-in, dev-only)"
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

# check-fast is the inner-loop gate: vet + lint + the full test suite
# WITHOUT the race detector, coverage instrumentation, or the selfcheck
# binary build. Run it before each commit. The full `make ci` (race +
# coverage + selfcheck) is the pre-push / CI-parity gate — run it once
# before pushing, not per commit.
check-fast: vet lint test

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

# The tagged runs mirror go.yml's vet job. Tag-gated sources compile in
# no other target — `go test ./...` builds neither — so without them a
# type error in a `stress`- or `testpins`-tagged file passes the whole
# local gate and surfaces only in CI, or when someone runs
# `make stress-tests` by hand. Each is a compile, no scenario runs.
vet:
	go vet ./...
	go vet -tags stress ./...
	go vet -tags testpins ./...

# Lint cache is scoped per working tree (same rationale as the
# pre-push hook): the shared user-level cache replays issues carrying
# other worktrees' absolute paths, which fail open once that worktree
# is deleted. A pre-set GOLANGCI_LINT_CACHE is respected.
lint:
	GOLANGCI_LINT_CACHE="$${GOLANGCI_LINT_CACHE:-$$(git rev-parse --absolute-git-dir)/golangci-lint-cache}" golangci-lint run

fmt:
	gofumpt -l -w .

coverage:
	go test -exec=$(TEST_EXEC) -coverprofile=coverage.out -coverpkg=./internal/... -parallel 8 ./...
	go tool cover -func=coverage.out | tail -n 1

# test-cov is the combined race+coverage pass `make ci` uses: one suite
# run yields both the race signal and the coverage profile (-race forces
# -covermode=atomic), instead of the two separate full passes the split
# test-race + coverage targets cost. Mirrors CI's go.yml test step.
test-cov:
	go test -exec=$(TEST_EXEC) -race -covermode=atomic -coverprofile=coverage.out -coverpkg=./internal/... -parallel 8 ./...
	go tool cover -func=coverage.out | tail -n 1

# coverage-gate runs the profile-driven policy gates: the diff-scoped
# branch-coverage audit (G-0067 — every statement on a line changed
# since origin/main is tested or //coverage:ignore'd), the total
# firing-fixture-presence meta-gate (G-0259 — every non-grandfathered
# policy has a test that covers its firing branch) plus its no-stale
# allowlist check, and the skill-edit structural-test backstop (G-0220 —
# a ritual SKILL.md edit must be paired with a referencing structural
# test under internal/policies/). It generates a fresh atomic-mode
# profile, then delegates to coverage-gate-only. The diff-scoped gates
# compare the base against the working tree, so uncommitted changes are
# in scope and you need not commit first. CI runs the same gates in the
# test job.
coverage-gate:
	go test -exec=$(TEST_EXEC) -covermode=atomic -coverprofile=coverage.out -coverpkg=./internal/... -parallel 8 ./...
	$(MAKE) coverage-gate-only

# coverage-gate-only runs those same gates against a coverage.out that
# already exists, skipping the instrumented suite run that produces it.
# That run is the whole cost — 2 minutes against the gates' 3 seconds —
# and it is never served from the test cache, because every `go test` here
# passes -exec=$(TEST_EXEC), which is outside the cacheable flag set `go
# help test` defines. (The coverage flags themselves are all cacheable;
# -exec is what defeats it.) Splitting the gates out is what lets `make ci`
# run them against the profile `test-cov` just built instead of paying for
# the suite twice.
#
# AIWF_COVERAGE_BASE is honored when the caller sets it, so a range that
# has already landed on trunk stays auditable:
#   AIWF_COVERAGE_BASE=<ref> make coverage-gate
# Left unset it resolves to the merge-base with origin/main, which is the
# fork point on a branch and github.event.before's equivalent once a merge
# is staged for push.
#
# The three arms below each report a distinct outcome, because the gates
# do not treat them alike:
#   - unset, or the all-zero sha CI hands a brand-new branch: every
#     diff-scoped policy reads that as "no base" and no-ops.
#   - a base that names no commit (a typo, a deleted ref): the policies
#     do NOT no-op, they fail on git's bad-revision error. Saying "will
#     no-op" here would promise silence and then die three seconds later.
#   - a base resolving to HEAD: only uncommitted changes are in scope.
# The HEAD comparison goes through `git rev-parse --verify` so a symbolic
# base like HEAD or a branch name is caught as readily as a sha.
ZERO_SHA := 0000000000000000000000000000000000000000
coverage-gate-only:
	@test -s "$(CURDIR)/coverage.out" || { \
	  echo "coverage-gate-only: no usable coverage.out at $(CURDIR) — nothing to audit."; \
	  echo "                    Generate one with 'make coverage-gate' or 'make test-cov'."; \
	  exit 1; \
	}
	@base="$${AIWF_COVERAGE_BASE:-$$(git merge-base origin/main HEAD 2>/dev/null)}"; \
	resolved="$$(git rev-parse --verify --quiet "$$base^{commit}" 2>/dev/null)"; \
	if [ -z "$$base" ] || [ "$$base" = "$(ZERO_SHA)" ]; then \
	  echo "coverage-gate: no comparison point ($${base:-unset}) — the diff-scoped gates will no-op."; \
	  echo "               Set AIWF_COVERAGE_BASE=<ref> to give them one."; \
	elif [ -z "$$resolved" ]; then \
	  echo "coverage-gate: base '$$base' does not name a commit — the diff-scoped gates will fail."; \
	elif [ "$$resolved" = "$$(git rev-parse HEAD)" ]; then \
	  echo "coverage-gate: base resolves to HEAD — only uncommitted changes are in scope."; \
	  echo "               To audit a range already on trunk: AIWF_COVERAGE_BASE=<ref> make coverage-gate"; \
	fi; \
	AIWF_COVERAGE_PROFILE="$(CURDIR)/coverage.out" \
	AIWF_COVERAGE_BASE="$$base" \
	go test -exec=$(TEST_EXEC) -run '^TestPolicy_(BranchCoverageAudit|FiringFixturePresence|FiringFixtureNoStaleAllowlist|SkillEditStructuralTestBackstop|CommentHistoryAttrition|TestExecutableWrite)$$' -count=1 ./internal/policies/

# comment-history-audit is the focused whole-tree run of the comment
# history-attrition scan — the surface the wf-codebase-health rubric's
# comment-hygiene force reaches for during an audit.
#
# It is not advisory: PolicyCommentHistoryAttritionTree runs in the
# ordinary policy suite, so the same finding already fails `make
# check-fast`, `make ci`, and CI. This target is the fast focused way to
# ask the question on its own — after a large merge or an import — and it
# exits non-zero on a finding, because a target that reported green while
# the build went red would be lying.
comment-history-audit:
	@echo "Scanning every tracked Go file for comments narrating a superseded state..."
	go test -exec=$(TEST_EXEC) -run '^TestPolicy_CommentHistoryAttritionTree$$' -count=1 ./internal/policies/

# growth-report snapshots the apparatus-growth metrics that
# docs/design/growth.md interprets: test-to-production ratio, policy-corpus
# share, entity and gap counts, and the same-day gap-closure share.
#
# Read-only and advisory by design. It reports a rate of increase, which is a
# judgment call about what the project is spending its effort on — gating on a
# growth budget would add a chokepoint to the problem the doc measures. Every
# metric derives from git history, so GROWTH_BASELINE=<rev> reconstructs any
# earlier point for comparison.
growth-report:
	@scripts/growth-report.py $(if $(GROWTH_BASELINE),--baseline $(GROWTH_BASELINE),)

# mutate-diff runs diff-scoped mutation testing (G-0267): gremlins on
# just the internal/ packages changed since the merge-base with
# origin/main, the wf-vacuity / mutate-hunt companion scoped to your
# diff instead of the whole kernel. Advisory — it prints surviving
# mutants for triage and always exits 0; mutation is slow and
# equivalent-mutant noise makes "0 survivors" un-gateable. Override the
# base with MUTATE_DIFF_BASE=<ref> and the per-mutant timeout with
# MUTATE_DIFF_COEFFICIENT=<n>. Requires gremlins + jq (absence is
# reported, not fatal). See scripts/mutate-diff.sh.
mutate-diff:
	@scripts/mutate-diff.sh

# selfcheck builds the binary and drives every verb against a temp
# repo via `aiwf doctor --self-check`. Catches end-to-end regressions
# (broken commit trailers, hook installer drift, missing skills,
# `aiwf init` against a fresh git repo failing) that unit tests miss.
selfcheck: build
	./bin/aiwf doctor --self-check

# ci is the gate the wrap rituals run before a merge or a push. It covers
# go.yml's build/vet/lint/test matrix and its profile-driven gate step;
# coverage-gate-only reads the profile test-cov just wrote, so the
# diff-scoped coverage gate costs seconds here rather than a second
# instrumented suite run.
#
# It is not literally everything CI does: go.yml additionally runs the
# suite under -tags testpins, the govulncheck job, the build job, and the
# linter-config rule-firing harness. Those stay CI-side. The tagged vet
# runs are covered — `vet` mirrors go.yml's vet job.
#
# The prerequisites are an ordered pipeline — coverage-gate-only reads
# what test-cov writes — which is safe only because .NOTPARALLEL: at the
# top of this file forbids -j from interleaving them.
ci: vet lint test-cov coverage-gate-only selfcheck

clean:
	rm -rf bin coverage.out

# install-hooks symlinks the tracked kernel hooks into their
# .git/hooks/<name>.local chain targets — the G-0045 seam invoked by
# aiwf's chain-aware hooks. Idempotent: ln -sfn overwrites any
# prior symlink and updates to scripts/git-hooks/* propagate
# immediately (the symlink resolves at hook-fire time).
#
#   pre-commit — kernel policy lint (gated on Go/build inputs, G-0280).
#   pre-push   — golangci-lint boundary gate on pushed Go changes
#                (G-0179) + gitleaks secret-scan over the pushed
#                range (G-0291); runs before aiwf's `aiwf check`.
#
# Run once after a fresh clone. The aiwf-managed hooks themselves
# are materialized by `aiwf init`/`aiwf update`, which write the
# chain-aware hooks at .git/hooks/<name>.
#
# The kernel treats itself like any consumer: aiwf owns
# .git/hooks/<name>, kernel-specific logic lives at
# .git/hooks/<name>.local, and the two compose via G-0045's chain.
# The destination is whatever `git rev-parse --git-path hooks`
# resolves, so a clone with `core.hooksPath` set installs there
# instead — the same directory `aiwf init` honors (G-0048), which is
# what keeps the two halves of the chain in one place.
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

# stress runs cmd/stresstest's whole scenario catalog, STRESS_REPEAT
# attempts per scenario, against a freshly built aiwf binary (see
# CLAUDE.md "Stress-test harness"). Opt-in and dev-only: not part of
# `make ci`. Override the per-scenario repeat count with STRESS_REPEAT=N.
STRESS_REPEAT ?= 5

stress:
	go run ./cmd/stresstest run --scenario all --repeat $(STRESS_REPEAT)

# stress-tests runs the Go tests behind the `stress` build tag: the
# concurrency and fault-injection scenario drivers, plus the
# whole-catalog runner tests. Their oracles assert timing and
# observation-window properties of the machine they run on, so they own
# the runner here rather than sharing it with `go test ./...`. Opt-in
# and dev-only, same as `stress` above; the workflow-legality drivers
# (verb-sequence and its siblings) carry no tag and run on every push.
stress-tests:
	go test -exec=$(TEST_EXEC) -tags stress -parallel 8 -count=1 ./internal/stresstest/ ./cmd/stresstest/
