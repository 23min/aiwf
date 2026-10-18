# Go conventions

This repository's additions to the Go pack, and the places it overrides the pack. Read the pack first.

Where a rule here repeats the pack, it is one this repository depends on.

## Formatting and linting

- **`golangci-lint`** is the only linter, and CI fails on any finding.
- No `//nolint` without a one-line rationale.
- **Never `var _ = x` to silence `unused`.** A blank-identifier alias kept solely to stop `unused`/`staticcheck` flagging reserved-for-future code defeats the check whose job is to find exactly that. No caller → not ready to land; delete it and re-add with its first real caller (the shipped form of this force is `wf-codebase-health` H2).

## Concurrency

- `context.Context` is the first argument of every IO-touching function, used for cancellation only.
- Don't stuff request data into a `context.Context` via `WithValue`, except across API boundaries.

## CLI conventions

- Each verb is a `cobra.Command` with a `runXCmd(...)` body called from `RunE`; wire from `newRootCmd`. Closed-set flag values bind completion via `cobra.FixedCompletions(...)`; entity-id flags use `completeEntityIDFlag(kind)`. Chokepoint: `internal/cli/integration/completion_drift_test.go`.
- The exit codes are `0` ok, `1` findings, `2` usage, `3` internal.
- **Exit codes:** a verb returns its code as an `int`, and the `RunE` adapter shuttles it via `*exitError`.
- Output is human-readable by default, and tool output goes to stdout.
- **Output:** `--format=json` emits `{ tool, version, status, findings, result, metadata }` (`status` ∈ `ok`/`findings`/`error`); `--pretty` indents. **Diagnostic/operator logging** is opt-in and default-off ([the logging ADR](../../docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md)): `log/slog`, configured via `AIWF_LOG`/`AIWF_LOG_FORMAT`/`AIWF_LOG_FILE` (env beats `aiwf.yaml`'s `logging:` block beats the default), writes structured records to a daily-rotated file under the XDG state home by default — never stderr unless explicitly configured there. `forbidigo` (backed by the `logging-chokepoint` AST policy, which also catches `fmt.Fprintln`/`fmt.Fprintf` to `os.Stdout`/`os.Stderr`) bans bare `fmt.Print*` call sites outside the sanctioned `cliutil` text wrappers, so operator-facing text can't silently drift back onto ad hoc prints; diagnostic logging goes through `log/slog` in `internal/logger`, never a bare print.
- **Flags:** `--help`, `--version`, `-v`, `--pretty` plus verb-specific. No global config files; config via flags, env, or `aiwf.yaml`.

## Dependencies

Minimize external deps (one-line justification per new one). `go 1.24` minimum in `go.mod` (the consumer-compatibility floor); CI toolchain is pinned separately to `go-version: "1.25"` (bump CI for stdlib CVEs, bump the floor only for a needed language feature). One `go.mod` for the module.

Binaries build with `CGO_ENABLED=0`, so they are static.

## Naming

Package names short, lowercase, singular (`entity` not `entities`). Avoid stuttering (`entity.Load`, not `entity.Entity`). Exported identifiers get a doc comment starting with the name.

Acronyms stay capitalized (`parseURL`, `jsonOut`).

## Testing

- Two or more cases of one function share one table, run as subtests through `t.Run`.
- A test with one case stays flat rather than table-driven.
- Golden files go under `testdata/`.
- **Policy tests that read entity files resolve via the loader** — `tree.Load(ctx, root)` + `Tree.ByID(id)` + `entity.Path`, never a hardcoded `filepath.Join(root, "work", …)` literal (archive sweeps move files; the loader resolves across active and archive). Chokepoint: `PolicyNoHardcodedEntityPaths`.

### Running tests in the devcontainer (primary)

Inside the devcontainer (Linux) there is no signing requirement and `go test` runs unwrapped. Use `make test` / `make test-race` / `make coverage`, bare `go test ./pkg/...`, or focused `go test -run TestX -count=1 ./pkg/...`.

### Running tests on macOS host (fallback)

On the macOS host, `go test` must route the per-package test binary through [`scripts/sign-and-run.sh`](../../scripts/sign-and-run.sh) via `-exec` (it ad-hoc-signs on Darwin; no-op on Linux/CI). Skipping it lands you in a Sonoma 14.8.x syspolicyd crash loop that stalls every new process launch. **Do:** `make test`, `make test-race`, `make coverage` (CI carries `-exec=./scripts/sign-and-run.sh` equivalently). **Don't:** bare `go test ./...` (bypasses the wrap; `-race` also hits a fork/exec deadlock). Focused runs outside `make`: `export GOFLAGS="-exec=$(pwd)/scripts/sign-and-run.sh"` then bare `go test`. **Defaults, not a chokepoint** — nothing catches a bare `go test` on the host. (Installing the production binary on Darwin needs the same signing: `make install` and `aiwf upgrade` sign automatically; bare `go install …@ver` does not — sign manually with `codesign -s - -f "$(go env GOPATH)/bin/aiwf"`.)

## Test discipline

`t.Parallel()` is the first line of every parallelizable test, and `TestMain` seeds shared state with `os.Setenv`, never `t.Setenv`, which panics under `t.Parallel`.

Test files run **parallel-by-default**, nested table subtests included; new test files follow suit. The six load-bearing rules:

- **`setup_test.go` per package** with a `func TestMain(m *testing.M)` that `os.Setenv`s the four GIT identity vars once (`GIT_AUTHOR_NAME`/`EMAIL`, `GIT_COMMITTER_NAME`/`EMAIL`) then `os.Exit(m.Run())`.
- **Serial skip-list** in `setup_test.go`'s comment block, one rationale line per test that must stay serial (`t.Setenv`/`t.Chdir`, mutates a package var, shares stdout/stderr capture).
- **`sync.Once` for expensive read-only shared fixtures** (cmd-binary build, live-repo `*Tree`), with a `// do not mutate` comment.
- **`-race -parallel 8` cap** uniform across `Makefile` + workflows (race + git-subprocess fan-out flakes at default parallelism). Chokepoint: `internal/policies/race_parallel_cap.go`.
- **`testsupport.HardenGitTestEnv()` in exec-bearing TestMains** (after identity seeding, before `m.Run()`): scrubs inherited git-locator env vars and disables auto-gc. Chokepoint: `internal/policies/git_test_env_harden.go`.
- **`testsupport.WriteExecutable` for every executable stand-in**, never a bare `os.WriteFile` with an exec bit. A plain write holds a writable descriptor the whole time it is open; a concurrent fork inherits it, and `execve` on a file held open for writing fails with `ETXTBSY`, so the fixture step fails and the property under test is never evaluated. The helper holds `syscall.ForkLock` across the write, which excludes exactly those forks. Writing to a temp name and renaming does *not* help — `ETXTBSY` is enforced against the inode, which the rename carries along. Chokepoint: `internal/policies/test_executable_write.go` (diff-scoped); `//exec:ok <reason>` when the mode itself is the subject.

Presence chokepoint: `internal/policies/test_setup_presence.go` (AST walk of `internal/*` test packages; fails CI if `setup_test.go`/`TestMain` is missing). Scope is `internal/*`; `cmd/aiwf/` uses a per-file skip-list in its own `setup_test.go`.

## Profile-driven gates

The profile-driven gates — the diff-scoped coverage audit chief among them — run inside `make ci` off the profile `test-cov` just built, so they cost seconds rather than the second instrumented suite run `make coverage-gate` pays for. That second run costs the full suite on a changed tree, and on Darwin costs it every time: `TEST_EXEC` names the Darwin-signing wrapper there and is empty elsewhere, and `-exec` with a program is outside the cacheable flag set `go help test` defines — the coverage flags themselves are all cacheable, replayed profile included. Gating off the profile `test-cov` already wrote is what keeps the marginal cost at seconds on either host, rather than at the mercy of cache state. The gates diff the **working tree** against the base, not `HEAD`, which is what lets `make ci` gate a change before the commit that carries it. The exceptions judge commits in the range, since their subject is a trailer or a message an uncommitted edit does not carry: the skill-edit provenance backstop, the co-author ban and the guidance fence.

## Coverage

High coverage on `internal/...` (PoC target 90%; total-coverage check advisory). Exclusions: `cmd/aiwf/main.go` (integration-tested), generated code, `//coverage:ignore <reason>` lines.

- **Diff-scoped coverage gate.** Every statement on a line changed since the base ref must be tested or `//coverage:ignore`'d — an untested changed branch names the `file:line`. Engine: `internal/policies/branch_coverage_audit.go`; runs inside `make ci`, and standalone via `make coverage-gate` (which builds its own profile). It compares the base against the **working tree**, so uncommitted edits and untracked `.go` files are in scope and you need not commit first. Once a merge has landed the default base resolves to `HEAD`; name the landed range with `AIWF_COVERAGE_BASE=<ref> make coverage-gate`. It's *statement* coverage with the ignore-escape, not true per-arm branch coverage. One asymmetry to know: CI builds its profile with `-tags testpins` and `make test-cov` does not, so a statement reached only by a `testpins`-gated test reads as uncovered locally while CI sees it covered — local red, CI green.
- **Firing-fixture meta-gate.** Every policy's `Policy: "<id>"` construction line must be covered by some test (no vacuous chokepoints), else it fails unless its id is in the shrinking `grandfatherDark` ledger. Engine: `internal/policies/firing_fixture_presence.go`; runs with `make ci` / `make coverage-gate`. Fail-closed if the profile carries no `internal/policies` blocks.
- **Beyond line coverage.** Fuzz targets (`Fuzz*` in `internal/{entity,gitops,version,pathutil}/`, full runs on the `fuzz` workflow); property tests (`internal/entity/transition_property_test.go` + `PolicyFSMInvariants`); mutation testing (`mutate-hunt`, workflow_dispatch only, `--workers 1 --timeout-coefficient 15`; ignore equivalent-mutant / unreachable-branch noise).
