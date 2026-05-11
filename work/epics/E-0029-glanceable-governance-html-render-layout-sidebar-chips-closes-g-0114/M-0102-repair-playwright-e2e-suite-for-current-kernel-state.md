---
id: M-0102
title: Repair Playwright e2e suite for current kernel state
status: in_progress
parent: E-0029
tdd: advisory
acs:
    - id: AC-1
      title: Fixture builds and runs against current kernel
      status: met
    - id: AC-2
      title: All Playwright assertions match current rendered output
      status: met
---
# Repair Playwright e2e suite for current kernel state

## Goal

Restore the Playwright e2e suite at `e2e/playwright/` to green against the current kernel. The suite rotted across multiple kernel changes since it landed under E-0009 — repo reorg (`a137132`), hook-installation behavior change in `aiwf init`, the uniform 4-digit ID width migration (E-0023), and likely other drift — and no CI gate caught any of it. M-0098 / M-0099 / M-0100 / M-0101 all depend on Playwright as the load-bearing test mechanism per the epic Constraints; the suite must be usable before any layout / chip / sidebar / hierarchy AC can be tested.

## Context

Discovered mid-red-phase on M-0098/AC-1 (Layout fills viewport at widths above 768px). The intent was to write a failing Playwright test for the viewport-fill assertion against the current `style.css`; running `npx playwright test` against the existing 40-test suite surfaced three independent stale layers before any new test could even fail meaningfully:

1. **Path-rot from `a137132 chore(aiwf): repo reorg to Go-standard layout`** (2026-05-05). The fixture's `repoRoot = resolve(__dirname, "..", "..", "..")` resolves to the parent of the repo (correct when the file lived at `tools/e2e/playwright/`, wrong now). The `go build` target `./tools/cmd/aiwf` no longer exists — the binary lives at `./cmd/aiwf`.
2. **Hook-disable trick broken** since some kernel change to `aiwf init`. The fixture sets `core.hooksPath: /var/empty` to neuter hooks during e2e runs, but `aiwf init` now writes its pre-push hook to the configured `hooksPath` instead of `.git/hooks/`, so the trick triggers `open /var/empty/pre-push: operation not permitted` on the read-only system dir.
3. **Test assertions expect pre-canonical narrow IDs** (`E-01`, `M-001`). Post-E-0023 the kernel emits canonical 4-digit IDs (`E-0001`, `M-0001`); the fixture's `--epic E-01` calls still resolve (parsers tolerate narrow input per ADR-0008), but the rendered HTML carries canonical IDs and the tests' `getByRole("link", { name: "E-01" })` won't match.

Without this repair the epic's Playwright-as-chokepoint constraint is "operator promises" — exactly the failure mode the kernel rule *"correctness must not depend on LLM behavior"* forbids. CI gating of Playwright is **still deferred** per the epic Constraints (no gap filed; trade-off accepted), but the local suite needs to actually run.

This milestone is the precursor to all downstream E-0029 work. M-0098 stays at `in_progress` (it began work — added ACs, started the red phase) but is blocked pending M-0102; the M-0098 *Work log* records the pivot.

## Acceptance criteria

ACs added via `aiwf add ac M-0102` at start-milestone time. The observable-behavior space:

- Path-rot fixed: `fixture.ts`'s `repoRoot` resolves to the actual repo root; the `go build` target is `./cmd/aiwf`; the existing comment block is updated to reflect the post-reorg layout.
- Hook-disable strategy works against current `aiwf init` — the fixture either uses a writable empty hooks dir (mkdtemp + setting `hooksPath` to it), runs the `hooksPath` config *after* `aiwf init`, or another mechanism that doesn't trigger the `/var/empty` write failure. The chosen mechanism is documented at the call site so future kernel changes signal the right surface to revisit.
- Test assertions updated to current canonical 4-digit ID format. Every test that locates entities by ID name (e.g. `getByRole("link", { name: "E-01" })`) is updated to `"E-0001"` (and similarly for milestones, ACs, decisions, contracts, ADRs, gaps). The fixture's narrow-ID `--epic E-01` calls can stay (parsers tolerate them) or be updated to canonical for clarity — decided at red phase.
- Full existing 40-test suite runs green against the current kernel + current `style.css` (i.e. pre-M-0098 layout, current rendered output). No test is deleted; no test loses semantic intent. The repair is mechanical: paths + hook strategy + ID assertions.
- One spot-check assertion is added or confirmed that exercises a *CSS-driven computed-style* behavior (e.g. the existing tabs-via-`:target` tests, or a small new layout-measurement against current CSS). This confirms the suite remains usable for the layout / chip / sidebar / hierarchy work in M-0098..M-0101.

## Constraints

- **No new test framework, no new dependencies.** Playwright 1.49.x stays. Chromium-only stays.
- **Repair, don't expand.** This milestone updates existing tests to match current rendered output; it does not add new behavioral tests. Layout / chip / sidebar / hierarchy tests land in M-0098..M-0101.
- **Don't change what tests check.** Test intent is preserved (e.g. a test asserting "epic page links navigate to per-epic page" still asserts that); only the *target string* (`E-0001` vs `E-01`) or the *fixture mechanism* (hook strategy) changes.
- **Deterministic.** Same discipline as the existing suite — no clock, no network, no flakes.

## Design notes

- The three rot layers are independently fixable; the milestone's ACs can be sequenced (paths → hooks → IDs → green) or parallelized (all three before re-running). Pick at start-milestone.
- For the hook-disable mechanism: a `mkdtempSync` empty hooks dir set via `core.hooksPath` after `aiwf init` is the cleanest fix. Setting before `aiwf init` to a writable empty dir also works. The "before vs after" decision affects whether `aiwf init` writes hooks into the empty dir (which we then ignore) or into the original `.git/hooks/` (where they sit unused). Either is fine; the writable-dir-after approach is slightly cleaner.
- The fixture's pre-canonical-ID strings (`E-01`, `M-001`) are inputs to the kernel verbs, which is still legal — but for consistency the fixture body could canonicalize them. Decide at red phase based on which produces cleaner test code.

## Surfaces touched

- `e2e/playwright/fixture.ts` (paths, hook-disable strategy, possibly verb input IDs)
- `e2e/playwright/tests/render.spec.ts` (ID assertion updates throughout)
- `e2e/playwright/playwright.config.ts` (no expected change; in scope if config-level workaround is needed)

## Out of scope

- **CI integration of Playwright.** Deferred per the epic Constraints; user decision (no gap filed).
- **New layout / chip / sidebar / hierarchy tests.** Those land in M-0098..M-0101.
- **Cross-browser support.** Chromium-only stays.
- **Restructuring test organization or splitting `render.spec.ts`.** Out of scope; future polish if needed.
- **Updating `e2e/playwright/`'s own README / docs.** Stretch goal but not required for "suite green."

## Dependencies

- None inside E-0029. This milestone is the precursor.

## References

- E-0029 (parent epic)
- G-0114 (gap closed by parent epic)
- `a137132 chore(aiwf): repo reorg to Go-standard layout` — surfaced the path-rot
- E-0023 (uniform 4-digit kernel ID width) — surfaced the ID-format drift
- `e2e/playwright/fixture.ts`, `e2e/playwright/tests/render.spec.ts` — the surfaces to repair
- `CLAUDE.md` — *Framework correctness must not depend on LLM behavior*

## Work log

### AC-1 — Fixture builds and runs against current kernel

Fixture executes cleanly against current kernel · commit `9cd5955` · tests 40/40 reach assertion phase. Three rot layers addressed in one commit: (1) `repoRoot` resolver changed from 3-levels-up to 2-levels-up to match post-`a137132` repo layout; (2) `go build` target changed from `./tools/cmd/aiwf` to `./cmd/aiwf`; (3) hook-disable strategy switched from `core.hooksPath: /var/empty` (read-only) to a `mkdtempSync` writable empty hooks dir set after `aiwf init`. Also caught a fourth rot layer mid-work: G-0055 chokepoint (E-0016) requires `--tdd` on `aiwf add milestone` — added `--tdd none` to both fixture milestone-add calls, preserving the original "M-001 tests both met-direct and phase-walked AC rendering" intent.

### AC-2 — All Playwright assertions match current rendered output

Full suite green · commit `c5a80e6` · tests 40/40. Eight find-replace sweeps canonicalized narrow-legacy ID strings in `render.spec.ts` (`"E-01"` → `"E-0001"`, `"M-001"` → `"M-0001"`, and the analogous `E-02` / `M-002` and `.html`-suffixed forms). AC-* identifiers (single-digit, not padded) and the fixture's verb-invocation side (narrow input still tolerated per ADR-0008 parser) were intentionally left alone. `npx playwright test` from `e2e/playwright/` — 40 passed in 24.7s on a clean run.

## Decisions made during implementation

- **Hook-disable strategy: writable empty dir set AFTER `aiwf init`.** Considered alternatives: (a) set writable dir BEFORE init (works but causes init to write hooks into the empty dir, where they sit unused — wasteful), (b) skip the hooks-disable entirely (slow, risks pre-commit reentrancy). The after-init writable-empty-dir approach lets `aiwf init` populate `.git/hooks/` as normal (where the hooks are simply ignored because hooksPath redirects elsewhere) and subsequent commits skip them cleanly. Documented at the call site.
- **`--tdd none` preserves original M-001 intent.** Under tdd: required the fixture's direct-promote of `M-001/AC-1` to `met` triggers `acs-tdd-audit` as error. The original (pre-G-0055) fixture had no `--tdd` flag → effectively `none` → audit didn't fire. Passing `--tdd none` verbatim post-G-0055 chokepoint preserves that behavior exactly. The M-001 fixture continues to test BOTH the "AC met without phase walk" path (AC-1) and the "AC phase-walked" path (AC-2) in one milestone, which is the rendering scenario the test suite cares about.
- **Fixture's verb-input narrow IDs stay unchanged.** Per ADR-0008 the parser tolerates narrow legacy widths on input. The fixture continues to send `--epic E-01`, `--epic M-001` etc.; the kernel canonicalizes on emit; tests assert canonical. Asymmetry is intentional — updating the verb inputs would be churn for no behavioral benefit.

## Validation

- `npx playwright test` from `e2e/playwright/` — **40 passed (24.7s)**, chromium-only, headless. No flakes, no skips.
- `aiwf check` — 0 errors, 2 `acs-tdd-audit` warnings on M-0102/AC-1 and AC-2 (expected; under tdd: advisory, met-without-phase-done is advisory-by-design, not a regression). Other warnings (G-0082, G-0083 archive-pending; provenance-untrailered-scope-undefined on the worktree branch's no-upstream) are pre-existing and unrelated to this milestone.
- `go test -race ./...` — clean (pre-wrap run from the worktree).
- Spot-check coverage: the existing `:target + :has()` tabs tests at `render.spec.ts:104..138` exercise CSS-driven computed-style behavior end-to-end; passing means the suite remains usable as the test surface for the upcoming layout / chip / sidebar / hierarchy milestones.

## Deferrals

- (none)

## Reviewer notes

- The `acs-tdd-audit` warnings on AC-1 and AC-2 are intentional under tdd: advisory. The work was repair-shaped, not new-behavior-shaped — there is no meaningful "write failing test first" red phase for fixing assertions to match current rendered output. The audit fires as warning (not error) by design for `tdd: advisory`; accepting it is the right call.
- The fixture's verb-input narrow IDs were deliberately left unchanged; this is a documented asymmetry, not an oversight (see *Decisions made*).
- CI integration of the Playwright suite remains **deferred** per the epic Constraints. The suite is local-only; operator discipline is the chokepoint until a follow-up wires CI.
- Future kernel changes that affect emitted output (rendered HTML structure, page filenames, ID format) will require parallel updates to this suite. The pattern of "kernel changes silently break Playwright" is the root issue surfaced here; a CI gate would catch it, but that gate is out of scope.

### AC-1 — Fixture builds and runs against current kernel

**Pass criterion**: `renderRichFixture()` in `e2e/playwright/fixture.ts` completes successfully against the current kernel — the kernel binary builds, the fixture repo's `aiwf init` succeeds, the 14 verb invocations all return zero exit codes, and the final `aiwf render --format html --out <dir>` produces a populated rendered site at the returned out-dir path. Verified by running `npx playwright test --reporter=line` from `e2e/playwright/` — if AC-1 holds, all tests reach their assertion phase (they may still fail because of AC-2, but no test errors before assertions during fixture setup).

**Edge cases**: The `repoRoot` resolver must work whether the file is read from `/Users/peterbru/Projects/aiwf` or from the worktree at `/Users/peterbru/Projects/aiwf-E-0029-glanceable-render` — it derives from `__dirname` which is fixture-file-local, not invocation-cwd-local, so both should work once paths are correct. The hook-disable strategy must survive `aiwf init`'s current behavior of writing pre-push (and possibly pre-commit) hooks to the configured `core.hooksPath`. Edge: if `aiwf init` writes both pre-push and pre-commit, and the test runs commit-shaped verbs (add / promote), the writable-empty-hooks-dir must accept those too — i.e. it's a real writable temp dir, not `/var/empty`.

**Code references**: `e2e/playwright/fixture.ts` — three concrete sites: (1) line 21 `repoRoot = resolve(__dirname, "..", "..", "..")` needs to become `resolve(__dirname, "..", "..")` (per the post-reorg layout); (2) line 31 `"./tools/cmd/aiwf"` needs to become `"./cmd/aiwf"`; (3) line 93 `runGit(repoDir, "config", "core.hooksPath", "/var/empty")` needs to switch to either a `mkdtempSync` writable hooks dir set before init, or to be reordered to run after `aiwf init` with a writable target. The accompanying comment at line 20 (`tools/e2e/playwright/`) and the comments at lines 90–92 (about hook strategy) get refreshed in step with the code edits.

### AC-2 — All Playwright assertions match current rendered output

**Pass criterion**: All 40 tests in `e2e/playwright/tests/render.spec.ts` pass against the rendered output produced by `renderRichFixture()` running through the current kernel. `npx playwright test` from `e2e/playwright/` exits 0; the reporter shows all tests passed; no test is skipped or flaky. Verified by a clean run from a fresh `node_modules/` state.

**Edge cases**: Tests that locate entities by literal ID name (`getByRole("link", { name: "E-01" })`, `name: "M-001"`, etc.) need their assertion targets updated to canonical 4-digit form (`"E-0001"`, `"M-0001"`). This applies to every kind the kernel canonicalizes: epics, milestones, gaps, decisions, contracts, ADRs, ACs. The fixture's verb-invocation side (`["add", "milestone", "--epic", "E-01", ...]`) can stay (kernel parsers tolerate narrow input per ADR-0008) or be updated for consistency — either choice is valid; pick the one that yields the cleanest diff. Tests asserting URL fragments / anchors (e.g. `#tab-build`, `#ac-2`) are kind-independent and should still pass — verify they do. Tests asserting CSS classes (`.status-met`, `.scope-active`) are kind-independent and should still pass. The `console.error` collector in `beforeEach` must remain green — no template / asset 404s introduced by canonical IDs (e.g. `M-0001.html` is the new filename, the fixture's directory layout must produce that).

**Code references**: `e2e/playwright/tests/render.spec.ts` — every `getByRole`, `getByText`, `locator(..., { hasText: "..." })`, `toHaveAttribute("href", "...")` assertion that targets an entity ID name. Approximate count via `grep -c "E-01\|M-001\|AC-1\|AC-2" e2e/playwright/tests/render.spec.ts`; each match site gets its narrow-ID target replaced with the canonical form. The `playwright-report/` directory shows per-test trace.zip files for any remaining red — useful for diagnosing assertions that fail for *other* reasons (template structure changes from E-0017's body-prose chokepoint, M-074's milestone tabs layout, etc. — out of scope for M-0102 but flagged if encountered).

