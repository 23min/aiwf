---
id: M-0120
title: Ratify legal-workflow spec methodology in ADR
status: done
parent: E-0033
tdd: advisory
acs:
    - id: AC-1
      title: ADR exists, is loaded as adr kind, initially proposed, cross-references E-0033
      status: met
    - id: AC-2
      title: Structural test asserts seven decision-point sections with non-empty content
      status: met
    - id: AC-3
      title: ADR promoted to accepted via aiwf promote
      status: met
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

### AC-1 — ADR exists, is loaded as adr kind, initially proposed, cross-references E-0033

### AC-2 — Structural test asserts seven decision-point sections with non-empty content

### AC-3 — ADR promoted to accepted via aiwf promote

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

## Work log

### AC-1 — ADR allocation and cross-reference

ADR-0011 allocated via `aiwf add adr --title "Legal-workflow spec methodology"` (commit `ddcdce4f`). Body cross-references E-0033 in both prose and the trailing References block. Verified by `TestADR0011_AC1_AllocationAndCrossReference` which asserts frontmatter `id: ADR-0011` matches and the body contains the string `E-0033`.

### AC-2 — Seven decision-section structural test

`internal/policies/adr_0011_test.go` added with three test functions, three helpers (`extractSubsection`, `hasNonEmptyProse`, `countLevel3Headings`), and the canonical seven section names enumerated in `required []string`. `TestADR0011_AC2_SevenDecisionSections` walks the markdown section hierarchy under `## Decision`, asserts each named subsection exists with non-empty prose, and includes a drift-guard count of `### `-level headings (`countLevel3Headings == 7`) so an 8th decision can't be added silently without updating the test.

### AC-3 — Promotion to accepted

`aiwf promote ADR-0011 accepted` ran cleanly through the FSM (`proposed → accepted`); commit `dae719ed`. `TestADR0011_AC3_StatusAccepted` regex-matches the frontmatter `status: accepted` line.

## Decisions made during implementation

No mid-flight architectural decisions surfaced beyond what the ADR itself ratifies. The three minor choices made during the work — package layout, test-helper signatures, the prose-vs-list shape of the ADR — were all routine implementation details, not architectural calls that warrant a separate decision entity.

## Validation

- `aiwf check --root .` — 0 errors, 21 warnings. Three new `acs-tdd-audit` warnings are advisory consequences of `tdd: advisory` policy with ACs at `phase: -` (expected; the audit fires as warning, not error). Remaining 18 warnings are pre-existing and unrelated.
- `go test -parallel 8 -short ./...` — all packages green. ~25 seconds total. Race detector deliberately omitted on macOS per G-0127 — race coverage runs in CI on Linux only.
- `golangci-lint run ./internal/policies/` — 0 issues.
- `go build -o /tmp/aiwf-e0033 ./cmd/aiwf` — green.
- Targeted run `go test -run TestADR0011 ./internal/policies/` — 3/3 passing, ~0.3s.

## Deferrals

- **G-0127** — opened during milestone preflight. Integration tests fork/exec deadlock on macOS under `-race -parallel 8`. Documented mitigation (don't run `-race` on macOS; CI runs Linux-race coverage). Root-cause investigation deferred. Cross-references this milestone via `discovered-in`.

## Reviewer notes

- The ADR deliberately commits to Go-as-canonical-form at the methodology level, not at M-0123 (when the schema lands). This is a load-bearing choice: deferring it to M-0123 would mean each downstream milestone reconsiders representation, defeating the "fixed point" property the ADR exists to provide. The trade-off: M-0123 has less design freedom than it would otherwise.
- The structural test for AC-2 walks the markdown section hierarchy with a hand-rolled scanner rather than a full markdown AST library. The existing `extractMarkdownSection` helper from `adr_0007_test.go` already does this; the new `extractSubsection`, `hasNonEmptyProse`, and `countLevel3Headings` helpers follow the same idiom. If future ADR-test work warrants it, these could be promoted to a shared `mdsection` package, but YAGNI for now — three helpers, one consumer.
- The drift-guard (`countLevel3Headings == 7`) intentionally fails CI when an 8th decision is added without updating the test. That's a feature, not friction: it forces explicit ratification of any new commitment rather than letting one slip in via a silent body edit.
- ADR-0011 dropped the `## References` markdown anchor convention used in some older ADRs — the cross-references are written as bullets at the bottom of the document, matching ADR-0010's more recent shape. No mechanical drift involved.
- The pre-existing `entity-body-empty` warning for M-0102 is unrelated to this milestone; leaving it alone.
