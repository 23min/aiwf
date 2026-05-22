---
id: M-0121
title: 'Pass A audit: catalog legal-workflow rules from existing surfaces'
status: done
parent: E-0033
depends_on:
    - M-0120
tdd: advisory
acs:
    - id: AC-1
      title: Audit catalog exists at canonical path with per-source sections in spec order
      status: met
      tdd_phase: done
    - id: AC-2
      title: All nine audit sources covered with at least one rule or explicit no-rules note
      status: met
      tdd_phase: done
    - id: AC-3
      title: Each rule row has the six-column schema with non-empty fields
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Catalog schema internally consistent: unique sequential ids, totals match'
      status: met
---
## Goal

Walk every source of pre-existing legality statements in the repo and extract them into a single draft catalog with citations. The output is **evidence**, not a spec — Pass C (M-0123) reconciles this catalog with M-0122's first-principles catalog into the canonical Go spec.

## Sources to audit

In order of mechanical authority (most rigorous first):

| Source | Format | What to extract |
|---|---|---|
| `internal/entity/transition.go` | Go maps | Per-kind FSM tables; (state, event, next-state) triples |
| `internal/policies/*.go` | Go tests | Each policy = a legality rule already mechanized in CI |
| `internal/check/*.go` | Go code | Each finding code = a class of illegal state |
| Cobra verb definitions under `cmd/aiwf/` and `internal/cli/<verb>/` | Go code | Per-verb pre/postconditions surfaced in flag validation and RunE bodies |
| ADRs under `docs/adr/` | Markdown | Cross-cutting workflow constraints (ADR-0004 archive, ADR-0008 ids, etc.) |
| `docs/pocv3/design/design-decisions.md` | Markdown | The 10 kernel commitments |
| `CLAUDE.md` | Markdown | Workflow constraints (trunk-based, AC mechanical evidence, etc.) |
| Skills under `.claude/skills/` + rituals plugin | Markdown | Narrative workflows (start-milestone, wrap-milestone, etc.) — lower authority but useful for cross-checking |
| `aiwf <verb> --help` | CLI text | Per-verb terse pre/postconditions |

## Output

A markdown file under `docs/pocv3/design/legal-workflows-audit.md` with one row per legality rule:

```
| Rule id | Source | Citation | Scope | Statement | Severity if violated |
```

The rule ids are sequential within the audit doc (R-AUDIT-001..N) — they are *not* an aiwf entity kind. They're internal references that Pass C will map onto spec cells.

## Acceptance criteria

### AC-1 — Audit catalog exists at canonical path with per-source sections in spec order

### AC-2 — All nine audit sources covered with at least one rule or explicit no-rules note

### AC-3 — Each rule row has the six-column schema with non-empty fields

### AC-4 — Catalog schema internally consistent: unique sequential ids, totals match

## Approach

- Walk sources top-down (most-mechanical first), so the lower-authority sources can cross-reference the higher.
- For each source, produce a per-source section in the audit doc with its extracted rules.
- Mark rules that are *implicit* (we believe the source intends X but doesn't state it directly) as such, so Pass C can flag them for explicit decision.
- No spec authorship in this milestone — only extraction.

## What this milestone does *not* do

- Does not invent new legality rules (that's M-0122).
- Does not reconcile contradictions between sources (that's M-0123).
- Does not produce Go code (test scaffolding excepted).

## Work log

### AC-1 — Audit catalog exists at canonical path with per-source sections in spec order

Catalog created at `docs/pocv3/design/legal-workflows-audit.md` (`legal-workflows-audit-r1.md` snapshot also created — see Reviewer notes for rationale). Source headings `### 1. FSM tables` through `### 9. Verb help text` appear in spec order; the order-walk in `TestM0121_AC1_AuditCatalogExistsAndOrdered` asserts each heading appears at a monotonically-increasing offset.

### AC-2 — All nine audit sources covered with at least one rule or explicit no-rules note

All nine sources produce at least one `R-AUDIT-NNNN` rule row. §2 (mechanical policies) acknowledges 11 out-of-scope CI-hygiene policies in its trailer block (matched by the test's case-insensitive substring search for `"out-of-scope"`). §5 includes one explicit out-of-scope row (R-AUDIT-0150, the ADR-0011 self-reference).

### AC-3 — Each rule row has the six-column schema with non-empty fields

All 225 R-AUDIT rows in §§1–9 parse to exactly 6 cells. `TestM0121_AC3_SixColumnSchemaNonEmpty` walks the row pattern via regex and asserts column count + non-empty content per cell. The test's `splitTableRow` helper handles two pipe-escape mechanisms: backslash-escaped (`\|`) and inline-code-protected (pipes inside backtick spans).

### AC-4 — Catalog schema internally consistent: unique sequential ids, totals match

R-AUDIT ids run R-AUDIT-0001 through R-AUDIT-0226 contiguously; no gaps. Each id matches the canonical 4-digit shape per ADR-0008. The acknowledgment row at R-AUDIT-0150 (ADR-0011 self-reference) is counted in the sequence. `TestM0121_AC4_SchemaConsistent` parses every `R-AUDIT-NNNN` mention, dedupes, and verifies the 1..max(ids) range is fully populated.

## Decisions made during implementation

- **External review integration (Revision 2, 2026-05-18).** A separate session reviewed the audit catalog and surfaced 8 conflicts. All were addressed: FSM-as-tree-invariant adopted, state-aware CancelTarget endorsed, conditional-severity schema, sovereign rationale, reallocate exception, self-transitions explicit, two-step `--force` pattern, R-RULE-019 restated. Two new gaps filed (G-0132, G-0131). Documented in §10's revision banner.
- **Pass A independence preservation (R1 snapshot).** A second-session review pointed out that Revision 2's interpretive amendments compromised the methodology ADR's load-bearing A/B independence. Resolution: snapshot the pristine pre-revision Pass A as `legal-workflows-audit-r1.md` (§§1-9 only); keep R2 (`legal-workflows-audit.md`) as the working catalog including reconciliation work. Pass B (M-0122) reads R1 only. Pass C (M-0123) reconciles R1 + Pass B's first-principles + R2's pre-reconciliation work as inputs.
- **G-0132 absorbed as M-0130; G-0131 absorbed as M-0131.** Both new milestones inserted between M-0123 (Pass C) and M-0124/M-0125 (cell coverage), so the cell tests run against the actually-enforced spec. M-0124 and M-0125 dependencies updated accordingly.
- **Conditional-severity schema design constraint** captured in M-0123's body: the `Rule` struct must model conditional severity natively (e.g., `[]ConditionalSeverity{Predicate, Escalated}`), not via duplicate rows or Notes-column smuggling. Affects ≥4 rules (acs-tdd-audit, unexpected-tree-file, archive-sweep-pending, validator-unavailable).

## Validation

- `aiwf check --root .` — 0 errors, 25 warnings. The 7 new `acs-tdd-audit` warnings (M-0121's 4 ACs at `phase: -` under tdd: advisory) are expected per CLAUDE.md's "advisory severity flips warning, not error" rule. Other warnings pre-existing.
- `go test -parallel 8 -short ./...` — all packages green.
- `go test -run TestM0121 -v ./internal/policies/` — 4/4 passing.
- `golangci-lint run ./internal/policies/` — 0 issues.
- `go build -o /tmp/aiwf-e0033 ./cmd/aiwf` — green.

## Deferrals

- **G-0132** — `fsm-history-consistent` check implementation. Absorbed as **M-0130** in E-0033 (inserted before M-0124/M-0125). Without it, R-RULE-001..018 in the catalog read as target severity rather than current. M-0124 and M-0125 depend on M-0130.
- **G-0131** — state-aware `CancelTarget` for Contract. Absorbed as **M-0131** in E-0033 (same insertion point). Without it, `aiwf cancel C-NNN` on a deprecated contract fails with an FSM-illegal-transition error. M-0124 and M-0125 depend on M-0131.
- **ADR-0001 + ADR-0009 ratification status.** Two long-proposed ADRs whose `unenforced (ADR proposed)` severity in the catalog (R-RULE-145, R-RULE-147) is doing real semantic work. Process call surfaced to the operator separately (not catalog work); each ADR should be ratified or rejected per CLAUDE.md's "decision is decision" rule.

## Reviewer notes

- **R2 catalog is not the spec.** It is M-0121's *evidence*. The methodology ADR (ADR-0011) commits Pass C to producing the canonical Go spec table; M-0123 owns that work. R2's §10 dedup is a useful *step toward* Pass C but is not authoritative.
- **R1 vs R2 distinction is load-bearing for Pass B.** When M-0122 starts, the operator (human or LLM) reads `legal-workflows-audit-r1.md` only. R2's §10 (and especially the Revision-2 amendments inside §§10.1, 10.5, 10.7) makes interpretive choices that should be Pass C's call, not Pass B's. R1's header makes this constraint explicit.
- **§10.1's "hard-reject" severity is target, not current.** The catalog's enforcement-status legend at the top of §10.1 explains the verb-time-vs-history-walk distinction. M-0130's job is to close the gap; until then, R-RULE-001..018's manual-edit cases are caught only as warnings via the partial coverage of `provenance-untrailered-entity-commit`.
- **The 5 sweep additions (R-RULE-152..156) include `aiwf milestone depends-on`** — a real mutating verb that §4's initial extraction missed because §4 worked from `aiwf --help` rather than directly reading verb-source files. The lesson for future audits: extract from source, not just from help text.
- **The audit catalog's `R-AUDIT-NNNN` namespace is internal to this document.** These ids are not aiwf entities; they are a numbering scheme for Pass C reconciliation. R-AUDIT ids 0001-0226 are the per-source facets; R-RULE ids 001-156 are the consolidated rules in §10. Do not confuse with any future kernel id schema.
- **The conditional-severity notation** (`base [predicate → escalated]`) used in §10's table is provisional. M-0123 will replace it with a typed Go field; see M-0123's body §Design constraint.
- **No `t.Skip` markers in this milestone's tests.** M-0121's ACs are about the catalog file's shape, which is fully testable today. The downstream milestones M-0124/M-0125 will carry `t.Skip` markers for cells that depend on M-0130/M-0131 — those are tagged at test-write time.
