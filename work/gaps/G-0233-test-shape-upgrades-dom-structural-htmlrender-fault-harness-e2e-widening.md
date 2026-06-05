---
id: G-0233
title: 'Test-shape upgrades: DOM-structural htmlrender, fault harness, e2e widening'
status: open
---
## What's missing

Three independent test-shape upgrades that the audit surfaced under different principles but share a common shape ("the test discipline is partial; tighten it"):

1. **htmlrender DOM-structural assertions.** Today `internal/htmlrender/htmlrender_test.go:61-181` and several integration tests use `strings.Contains` to assert anchor/id/section presence in rendered HTML. CLAUDE.md's §"Substring assertions are not structural assertions" specifically names this anti-pattern. Adopt `golang.org/x/net/html` for parse-and-traverse; add a small `internal/testutil/htmlassert/findInside(node, pred)` helper; migrate the existing substring assertions to structural ones (one PR per page-shape is fine).
2. **`internal/policies/dom_structural_assertions.go`** — AST-level policy test forbidding `strings.Contains` against the result of any function returning HTML bytes / `template.HTML` / `[]byte` known to be HTML. Allowlist the few tests where substring is genuinely correct (free-text CLI human-output checks) with rationales.
3. **Synthetic-fault test harness.** Today 70+ `//coverage:ignore` markers carry the rationale "requires concurrent FS mutation / requires ENOSPC / requires syscall race." Add a small fault-injection harness (`internal/testutil/fault/`) that can simulate `ENOSPC`, `EAGAIN`, mid-write process kill, lock-contention — and migrate the highest-leverage `//coverage:ignore` sites into executed tests.
4. **Widen Playwright e2e.** `e2e/playwright/tests/render.spec.ts` is the only Playwright spec (55 tests). All 55 share one fixture project tree; a fixture-break invalidates every assertion. Add at least one second spec exercising a distinct fixture (an archive-heavy tree, a contract-heavy tree, or a multi-epic tree) so the tail-risk is bounded.

## Why it matters

D1's verdict was Strong but flagged the substring-assertion pattern as a "known weakness" CLAUDE.md already names; D4 noted the single-spec Playwright shape; E2 noted the synthetic-fault gap. All three are "the discipline exists, here are the places it isn't applied yet."

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §D1 (all three moves), §D4 (move 2: widen Playwright), §E2 (move 1: synthetic-fault harness).
