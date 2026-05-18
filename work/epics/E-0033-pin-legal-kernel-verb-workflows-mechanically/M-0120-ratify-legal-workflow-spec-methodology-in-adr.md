---
id: M-0120
title: Ratify legal-workflow spec methodology in ADR
status: draft
parent: E-0033
tdd: advisory
acs:
    - id: AC-1
      title: ADR exists, is loaded as adr kind, initially proposed, cross-references E-0033
      status: open
---
## Goal

Land an ADR ratifying the methodology for building and maintaining the legal-workflow specification. The ADR is the **fixed point** that downstream milestones (M-0121..M-0125) build against — without it, each subsequent milestone risks re-litigating the methodology.

## Decision points the ADR commits to

Seven decisions, each its own section in the ADR body:

1. **Independence** — the spec is human-authored, not generated from the implementation. Spec-from-impl extraction is rejected because it makes tests rubber-stamp impl behavior.
2. **Three-pass methodology** — audit existing surfaces (Pass A) → first-principles derivation (Pass B, blind to A) → reconcile into spec (Pass C). The independence of A and B is load-bearing; reading B's prior output during A (or vice versa) defeats the cross-check.
3. **Canonical form: Go data structures** — the spec lives under `internal/workflows/spec/` (exact package name to be settled in M-0123). Markdown catalogs from A and B are working artifacts, not the spec. Go is the canonical form because (a) the test harness consumes Go, (b) AI-discoverability prefers Go over markdown for typed data, (c) the drift policy needs a single source of truth that CI can inspect.
4. **Cell-coverage commitment** — every spec entry has paired positive + negative tests under `internal/policies/`. A meta-test asserts the coverage is complete.
5. **Drift policy** — the spec is closed-set against the impl. No impl FSM transition, Cobra verb, or finding code may exist that the spec doesn't reference. A `internal/policies/` test enforces this. When the impl grows, the spec grows in the same PR.
6. **Scope** — kernel-verb workflows only. Branch choreography (which git branch verbs run on) is out-of-scope; E-0030 covers it. Rituals-plugin orchestration (skills composing kernel verbs into named workflows) is out-of-scope; per "framework correctness must not depend on LLM behavior," only the kernel can be mechanically verified.
7. **Future-change handling** — when the impl adds a verb, kind, or state, the same PR must update the spec or the drift test blocks. This is the load-bearing maintainability property; without it, the spec rots.

## Acceptance criteria

(Populated by `aiwf add ac` after this body lands.)

## Approach

1. Allocate the ADR via `aiwf add adr --title "Legal-workflow spec methodology"` (it gets the next ADR-NNNN id).
2. Draft the ADR body with the seven decision-point sections, an executive summary, context, alternatives considered, and consequences.
3. Cross-reference: ADR body cites E-0033; epic body cites the ADR.
4. Write a structural test under `internal/policies/` that walks the ADR's markdown and asserts the seven section headings are present with non-empty content. This is the mechanical evidence per CLAUDE.md's "AC promotion requires mechanical evidence" rule, even though `tdd: advisory`.
5. Promote ADR-NNNN to `accepted` via `aiwf promote ADR-NNNN accepted`.

## What this milestone does *not* do

- Does not enumerate workflows or write any catalog content (M-0121 + M-0122).
- Does not commit to a specific Go schema for the spec table (M-0123 — schema follows from what the catalogs surface).
- Does not bind specific severity tiers for illegal cells (M-0123 — also schema-dependent).

## Risk surface and mitigations

- **Risk: the ADR over-commits.** Mitigation: each of the seven decisions is at the load-bearing level (what shape vs. what content). Schema details, package names, exact rule-row layouts are deferred to M-0123. The ADR should be readable in 5 minutes; if it grows to 20 minutes, it's accreting detail that belongs elsewhere.
- **Risk: the structural test is too lenient.** Mitigation: assert section *content* is non-empty, not just heading presence. Use `golang.org/x/text` or markdown-AST walking — not `strings.Contains` (per CLAUDE.md: substring assertions are not structural assertions).
- **Risk: the methodology ADR introduces gate language.** Mitigation: the ADR ratifies the *methodology* now; M-0121..M-0125 act on it on their own schedule. No "ratify after X happens" phrasing in the body.

### AC-1 — ADR exists, is loaded as adr kind, initially proposed, cross-references E-0033

