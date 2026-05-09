---
id: G-090
title: AC-8 materialisation drift-check has three branches not unit-tested; refactor lookup to take cache root as parameter for hermetic testing with synthetic temp dirs
status: open
discovered_in: M-079
---
M-079's `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` (in `internal/policies/aiwfx_whiteboard_test.go`) verifies that the rituals-plugin cache contains a copy of the `aiwfx-whiteboard` SKILL.md byte-for-byte matching the in-repo fixture. The success path (cache present + drift-free) and the "skill missing" path were exercised end-to-end during M-079's deploy cycle. Three other branches in the lookup logic remain unexercised by hermetic unit tests:

1. **Cache-root absent → skip.** When `~/.claude/plugins/cache/` doesn't exist (fresh install / sandboxed CI), the test should skip cleanly. Today the skip logic is asserted only by absence of failure on machines without the cache.
2. **Plugin-dir read error → skip.** When the cache exists but the specific plugin directory is unreadable (permissions, partial install), the skip path fires. Untested.
3. **Fixture↔cache drift → FAIL.** The test's *load-bearing* assertion. When the cache contains a SKILL.md that differs from the in-repo fixture, the test must fail with a clear drift message. This branch was exercised manually during M-079 deploy iteration (cache → mutate fixture → re-run shows drift); no unit test pins it.

## What's missing

**Hermetic unit-test coverage of the three runtime-conditional branches in the AC-8 cache-lookup logic.** The current shape of the lookup function reads from the home directory at call time, which makes hermetic testing (deterministic input, no `$HOME` side effects) impossible without process isolation.

The fix is structural: refactor the cache-root resolution to take the cache root as a parameter rather than discovering it from `$HOME`. The function signature changes from `lookupSkillInCache(name string) (string, error)` to `lookupSkillInCache(cacheRoot, name string) (string, error)`. Tests inject a synthetic temp dir (`t.TempDir()` returns a path; tests populate it with the fixtures the case needs), exercise each branch, and clean up automatically. Production callers pass `os.UserHomeDir()`-derived paths.

## Why it matters

Three downstream costs of the current shape:

1. **The drift-FAIL branch — the test's actual safety value — is unverified.** The success path passes today because the deploy is fresh; if the rituals-plugin cache ever drifts from the in-repo fixture without one of the skip conditions catching it (cache exists, plugin dir readable, but content differs), the test would FAIL as designed — *but only the success path proves the assertion machinery works*. The codebase is trusting a branch that has never executed.
2. **CLAUDE.md *"Test untested code paths"* discipline is unmet.** The Testing section's §"Test untested code paths before declaring code paths 'done'" requires every reachable branch to have a test or a `//coverage:ignore` rationale. The three branches above are reachable in production but unexercised by any unit test fixture.
3. **Future plugin-skill milestones repeat the mistake.** AC-8's lookup pattern is the prototype for any future "this plugin skill stays byte-for-byte aligned with the kernel-side fixture" check. If the prototype ships with three untested branches, every copy-paste descendant inherits the gap.

## Fix shape

**Refactor the cache-lookup helper to accept the cache root as a parameter.** Two-line signature change in production code; corresponding tests under `internal/policies/aiwfx_whiteboard_test.go` (or a new `_test.go` if the lookup is moved into its own file) exercise all three branches with synthetic temp dirs:

- **Branch A (cache-root absent):** test points lookup at a non-existent path → returns skip.
- **Branch B (plugin dir read error):** test creates a temp dir, makes its plugin subdir unreadable (`os.Chmod 0000`); lookup → returns skip.
- **Branch C (fixture↔cache drift):** test creates a temp dir, populates the plugin subdir with a SKILL.md that differs from the fixture; lookup → returns drift error with the specific assertion message.

After refactor, `go test -coverprofile=cov.out ./internal/policies/` shows 100% coverage on the lookup function. The success and skill-missing branches stay covered as before via the existing tests.

## Out of scope

- **Generalizing the parameterization to other home-directory-reading helpers.** Would be a broader CLAUDE.md *"No package-level mutable state"* sweep; tracked elsewhere if it becomes painful.
- **Adding a CI-side drift check that doesn't require local plugin install.** M-079's reviewer notes flagged this as a separate friction point in cross-repo deploy ergonomics. Could combine well with this gap's refactor but isn't strictly required to close it.

## References

- **M-079** work log AC-8 — the cycle where the test was added.
- **M-079** Deferrals section — explicit deferral of branch-coverage hardening; this gap is the deferred work item.
- `internal/policies/aiwfx_whiteboard_test.go` — current test location.
- CLAUDE.md *"Testing"* §"Test untested code paths before declaring code paths 'done'" — the discipline this gap captures non-compliance with.
- CLAUDE.md *"Coverage"* §"Beyond line coverage" — the broader testing discipline (fuzz / property / mutation) into which the AC-8 branches eventually feed.
