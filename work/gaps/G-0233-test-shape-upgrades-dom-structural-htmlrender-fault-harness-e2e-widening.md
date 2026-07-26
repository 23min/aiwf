---
id: G-0233
title: 'Test-shape upgrades: DOM-structural htmlrender, fault harness, e2e widening'
status: open
priority: medium
---
## What's missing

The htmlrender test suite asserts rendered-HTML structure with `strings.Contains` — a substring search that proves a literal appears *somewhere* in the output, not that it appears in the right element, id, or section. CLAUDE.md's §"Substring assertions are not structural assertions" names this exact anti-pattern. Two changes make the discipline structural:

1. **Parse-and-traverse HTML assertions.** Adopt `golang.org/x/net/html`; add a small `internal/testutil/htmlassert` helper (`findInside(node, pred)`); migrate the ~13 `strings.Contains` assertions in `internal/htmlrender/htmlrender_test.go` (plus the handful in `markdown_test.go`) to parse the DOM and assert presence inside the named element / id / section.
2. **`internal/policies/dom_structural_assertions.go`** — an AST-level policy test forbidding `strings.Contains` against the result of any function returning HTML bytes / `template.HTML` / `[]byte` known to be HTML. Allowlist the few free-text CLI human-output checks where substring is genuinely correct, each with a one-line rationale. This is the load-bearing piece: without it the pattern creeps back as new render tests land.

Scope is small — an afternoon's `wf-patch`, not a milestone.

## Out of scope

**No synthetic-fault harness.** The ~35 fault-shaped `//coverage:ignore` sites (ENOSPC, lock-contention, TOCTOU races between two syscalls) mark defensive error-wrap-and-return lines that carry no branch logic. Exercising them would require injecting fake-filesystem / fake-syscall seams through production code — adding shipping-code complexity to test lines that only wrap an error, against KISS and the repo's real-dependencies-over-mocks rule. The `//coverage:ignore` annotation is already the correct disposition for a genuinely-unreachable line, so this is not a gap to close.

**Playwright fixture widening is deferred.** `e2e/playwright/tests/render.spec.ts` is a single 55-test spec over one fixture tree, so a fixture break invalidates every assertion — a real but bounded tail-risk. A second spec over a distinct tree (archive-heavy / contract-heavy / multi-epic) belongs in its own gap, filed if the tail-risk bites.

## Why it matters

D1's verdict was Strong but flagged the substring-assertion pattern as a known weakness CLAUDE.md already names by hand. Turning it into a parse-and-traverse discipline plus an AST tripwire finishes the habit and keeps it finished.

## Source

`docs/archive/pocv3/health-scorecard-2026-06-04.md` §D1 (structural-assertion move).
