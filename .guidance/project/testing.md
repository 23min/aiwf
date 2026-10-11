# Testing

## Profile-driven gates

The profile-driven gates — the diff-scoped coverage audit chief among them — run inside `make ci` off the profile `test-cov` just built, so they cost seconds rather than the second instrumented suite run `make coverage-gate` pays for. That second run costs the full suite on a changed tree, and on Darwin costs it every time: `TEST_EXEC` names the Darwin-signing wrapper there and is empty elsewhere, and `-exec` with a program is outside the cacheable flag set `go help test` defines — the coverage flags themselves are all cacheable, replayed profile included. Gating off the profile `test-cov` already wrote is what keeps the marginal cost at seconds on either host, rather than at the mercy of cache state. The gates diff the **working tree** against the base, not `HEAD`, which is what lets `make ci` gate a change before the commit that carries it. The exceptions judge commits in the range, since their subject is a trailer or a message an uncommitted edit does not carry: the skill-edit provenance backstop, the co-author ban and the guidance fence.

## Evidence for a doc-shaped acceptance criterion

For doc-shaped ACs, the test is a structural assertion scoped to a named markdown section (see Test design rules below) — except over a shipped surface or this repository's development guidance, where [the shipped-prose decision](../../work/decisions/D-0070-retire-prose-content-assertions-over-shipped-surfaces.md) and [the development-guidance decision](../../work/decisions/D-0102-no-criterion-is-evidenced-by-a-sentence-pinned-in-development-guidance.md) rule that class out and the evidence has to be a relationship check instead.

## Conventions

- **Table-driven** when ≥2 cases share a function (subtests via `t.Run`); single-case tests stay flat.
- **Golden files** under `testdata/`, synthetic (obviously-fictional) content only.

### Running tests in the devcontainer (primary)

Inside the devcontainer (Linux) there is no signing requirement and `go test` runs unwrapped. Use `make test` / `make test-race` / `make coverage`, bare `go test ./pkg/...`, or focused `go test -run TestX -count=1 ./pkg/...`.

### Running tests on macOS host (fallback)

On the macOS host, `go test` must route the per-package test binary through [`scripts/sign-and-run.sh`](../../scripts/sign-and-run.sh) via `-exec` (it ad-hoc-signs on Darwin; no-op on Linux/CI). Skipping it lands you in a Sonoma 14.8.x syspolicyd crash loop that stalls every new process launch. **Do:** `make test`, `make test-race`, `make coverage` (CI carries `-exec=./scripts/sign-and-run.sh` equivalently). **Don't:** bare `go test ./...` (bypasses the wrap; `-race` also hits a fork/exec deadlock). Focused runs outside `make`: `export GOFLAGS="-exec=$(pwd)/scripts/sign-and-run.sh"` then bare `go test`. **Defaults, not a chokepoint** — nothing catches a bare `go test` on the host. (Installing the production binary on Darwin needs the same signing: `make install` and `aiwf upgrade` sign automatically; bare `go install …@ver` does not — sign manually with `codesign -s - -f "$(go env GOPATH)/bin/aiwf"`.)

## Test design rules

- **Test the seam, not just the layer.** When a helper is wired into a caller, cover the helper's behavior *and* the integration seam (drive `run([]string{"<verb>", …})` or `check.Run`). When output depends on values only a real binary has (`ReadBuildInfo`, ldflags globals, `os.Args[0]`), add a binary-level subprocess integration test.
- **Contract tests for upstream-cached systems.** Pin "did we ask the right question," not just "did we parse the answer": derive the expected value through an *independent* code path (gated under `-short`).
- **Spec-sourced inputs for upstream-defined input spaces.** Cite the spec in a comment and cover the full enumerated grammar, not "the example I had in mind."
- **Substring assertions are not structural assertions.** A grep for a literal proves it exists *somewhere*, not in the right *place*. Parse HTML (`x/net/html`) / walk the heading hierarchy and assert presence inside the named section/attribute; for multi-section pages name *which* section. Use a structural assertion whenever the literal is short/generic. **Over a shipped surface the answer is not to scope the substring but to drop it** — [the shipped-prose decision](../../work/decisions/D-0070-retire-prose-content-assertions-over-shipped-surfaces.md) retires prose- and heading-presence assertions across the embedded skill, ritual, template, agent-card, guidance, and statusline trees, because scoping a phrase still pins a reading that rewording breaks and nothing catches. What survives there is the relationship check, comparing two artefacts so either one moving goes red: a citation resolved against the document it names, or an expectation derived by running the code. Enforced by `shipped-prose-assertion` (`internal/policies/shipped_prose_assertion.go`), whose two exemption maps carry the trigger phrases that drive skill dispatch and the derived-expectation checks. Development guidance is held by [the development-guidance decision](../../work/decisions/D-0102-no-criterion-is-evidenced-by-a-sentence-pinned-in-development-guidance.md): every test that reads `CLAUDE.md`, `AGENTS.md`, the project router or a document it routes to from the repository root is listed in `guidanceReaderList` as a relationship check, an absence check, or a test naming a guidance file without reading it, and `guidance-readers` fails on a reader the list does not name. Everywhere else — the design docs, generated output — this rule stands unchanged.
- **Render output must be human-verified before an iteration closes.** For rendered UI/docs, run the binary against a real fixture (the kernel's own planning tree), exercise every tab/link/conditional path, then mark done. If you can't run the binary, say so — tests don't stand in for the look.
- **Test every reachable branch.** Each `if`/`switch`/filter arm needs a test that traverses it, or a `//coverage:ignore <reason>`. Skim uncovered lines before committing; each is a missing test, dead code to delete, or genuinely unreachable (annotate).
- **Don't paper over a test failure — root-cause it.** A state error (lock contention, projection mismatch, not-found) is information about the system. Dump the state, read the cause. A "manual git commit to keep things clean" inside a test is a yellow flag — the production verb isn't making it.
- **Policy tests that read entity files resolve via the loader** — `tree.Load(ctx, root)` + `Tree.ByID(id)` + `entity.Path`, never a hardcoded `filepath.Join(root, "work", …)` literal (archive sweeps move files; the loader resolves across active and archive). Chokepoint: `PolicyNoHardcodedEntityPaths`.

## Test discipline

Test files run **parallel-by-default**, with `t.Parallel()` the first line of every parallelizable test, nested table subtests included; new test files follow suit. The six load-bearing rules:

- **`setup_test.go` per package** with a `func TestMain(m *testing.M)` that `os.Setenv`s the four GIT identity vars once (`GIT_AUTHOR_NAME`/`EMAIL`, `GIT_COMMITTER_NAME`/`EMAIL`) then `os.Exit(m.Run())`. `os.Setenv`, not `t.Setenv` (which panics under `t.Parallel`).
- **Serial skip-list** in `setup_test.go`'s comment block, one rationale line per test that must stay serial (`t.Setenv`/`t.Chdir`, mutates a package var, shares stdout/stderr capture).
- **`sync.Once` for expensive read-only shared fixtures** (cmd-binary build, live-repo `*Tree`), with a `// do not mutate` comment.
- **`-race -parallel 8` cap** uniform across `Makefile` + workflows (race + git-subprocess fan-out flakes at default parallelism). Chokepoint: `internal/policies/race_parallel_cap.go`.
- **`testsupport.HardenGitTestEnv()` in exec-bearing TestMains** (after identity seeding, before `m.Run()`): scrubs inherited git-locator env vars and disables auto-gc. Chokepoint: `internal/policies/git_test_env_harden.go`.
- **`testsupport.WriteExecutable` for every executable stand-in**, never a bare `os.WriteFile` with an exec bit. A plain write holds a writable descriptor the whole time it is open; a concurrent fork inherits it, and `execve` on a file held open for writing fails with `ETXTBSY`, so the fixture step fails and the property under test is never evaluated. The helper holds `syscall.ForkLock` across the write, which excludes exactly those forks. Writing to a temp name and renaming does *not* help — `ETXTBSY` is enforced against the inode, which the rename carries along. Chokepoint: `internal/policies/test_executable_write.go` (diff-scoped); `//exec:ok <reason>` when the mode itself is the subject.

Presence chokepoint: `internal/policies/test_setup_presence.go` (AST walk of `internal/*` test packages; fails CI if `setup_test.go`/`TestMain` is missing). Scope is `internal/*`; `cmd/aiwf/` uses a per-file skip-list in its own `setup_test.go`.

## Coverage

High coverage on `internal/...` (PoC target 90%; total-coverage check advisory). Exclusions: `cmd/aiwf/main.go` (integration-tested), generated code, `//coverage:ignore <reason>` lines.

- **Diff-scoped coverage gate.** Every statement on a line changed since the base ref must be tested or `//coverage:ignore`'d — an untested changed branch names the `file:line`. Engine: `internal/policies/branch_coverage_audit.go`; runs inside `make ci`, and standalone via `make coverage-gate` (which builds its own profile). It compares the base against the **working tree**, so uncommitted edits and untracked `.go` files are in scope and you need not commit first. Once a merge has landed the default base resolves to `HEAD`; name the landed range with `AIWF_COVERAGE_BASE=<ref> make coverage-gate`. It's *statement* coverage with the ignore-escape, not true per-arm branch coverage. One asymmetry to know: CI builds its profile with `-tags testpins` and `make test-cov` does not, so a statement reached only by a `testpins`-gated test reads as uncovered locally while CI sees it covered — local red, CI green.
- **Firing-fixture meta-gate.** Every policy's `Policy: "<id>"` construction line must be covered by some test (no vacuous chokepoints), else it fails unless its id is in the shrinking `grandfatherDark` ledger. Engine: `internal/policies/firing_fixture_presence.go`; runs with `make ci` / `make coverage-gate`. Fail-closed if the profile carries no `internal/policies` blocks.
- **Beyond line coverage.** Fuzz targets (`Fuzz*` in `internal/{entity,gitops,version,pathutil}/`, full runs on the `fuzz` workflow); property tests (`internal/entity/transition_property_test.go` + `PolicyFSMInvariants`); mutation testing (`mutate-hunt`, workflow_dispatch only, `--workers 1 --timeout-coefficient 15`; ignore equivalent-mutant / unreachable-branch noise).
