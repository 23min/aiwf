---
id: M-0102
title: Repair Playwright e2e suite for current kernel state
status: draft
parent: E-0029
tdd: advisory
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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
