---
id: M-0122
title: 'Pass B first-principles: derive legal-workflow rules from entity model'
status: done
parent: E-0033
depends_on:
    - M-0120
tdd: advisory
acs:
    - id: AC-1
      title: Catalog file exists at canonical path with top-level sections
      status: met
    - id: AC-2
      title: Catalog has per-kind lifecycle section for each entity kind
      status: met
    - id: AC-3
      title: R-FP-NNNN rule rows have non-empty schema fields
      status: met
    - id: AC-4
      title: R-FP-NNNN ids unique and contiguous starting at R-FP-0001
      status: met
    - id: AC-5
      title: Open questions for Pass C section present and non-empty
      status: met
---
## Goal

Derive the legal-workflow surface from first principles — the entity model, the kernel's six kinds, their lifecycles, and the cross-entity relationships — without reading M-0121's audit catalog. The output is a parallel-structure markdown catalog that Pass C (M-0123) reconciles against M-0121.

The independence from M-0121 is *load-bearing*: if first-principles derivation produces a catalog that matches existing surfaces, we have high confidence. If it diverges, those divergences become explicit decisions in Pass C.

## Methodology

For each entity kind (epic, milestone, AC, ADR, gap, decision, contract):

1. **Lifecycle** — enumerate the legal states + transitions (independent of `internal/entity/transition.go`).
2. **Birth conditions** — what `aiwf add <kind>` requires (parent exists, kind-specific flags, naming rules).
3. **Terminal states** — which states are terminal? When does the entity become archive-eligible?
4. **Cross-entity invariants** — what does this kind's state imply for sibling/parent/child entities? E.g., "an AC's lifecycle is bounded by its parent milestone's lifecycle."
5. **Verb closure** — which kernel verbs operate on this kind, and what's each verb's pre/post condition expressed against the lifecycle?

For verbs that operate across kinds (`archive`, `promote`, `add ac`, etc.):

6. **Cross-kind preconditions** — what's true about the planning tree before the verb runs?
7. **Cross-kind post-conditions** — what's true after?

## Output

A markdown file under `docs/pocv3/design/legal-workflows-first-principles.md` with the row schema:

```
| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
```

Rule ids are `R-FP-NNNN` (4-digit, sequential; separate id-space from R-AUDIT) so Pass C can reference both during reconciliation. The "Reasoning" column is the explicit Pass B addition — Pass C uses it to distinguish principled agreement with Pass A from coincidental agreement.

## Acceptance criteria

### AC-1 — Catalog file exists at canonical path with top-level sections

### AC-2 — Catalog has per-kind lifecycle section for each entity kind

### AC-3 — R-FP-NNNN rule rows have non-empty schema fields

### AC-4 — R-FP-NNNN ids unique and contiguous starting at R-FP-0001

### AC-5 — Open questions for Pass C section present and non-empty

## Approach

- Author the catalog *without consulting* M-0121's output. Discipline matters here — if I peek, the cross-check loses its value.
- Use only:
  - The entity model from `docs/pocv3/design/design-decisions.md` (the six kinds, closed-set semantics).
  - Generic reasoning about lifecycles, ownership, and invariants.
  - ADRs that *define* the entity model (not ones that constrain workflows).
- Mark rules as *load-bearing* (this must hold or the model breaks) vs *conventional* (a sensible default but could be otherwise).

## What this milestone does *not* do

- Does not read M-0121's catalog.
- Does not reconcile (that's M-0123).
- Does not produce Go code.

## Operator notes

Executed by a subagent (general-purpose, no conversation history from the parent session) per the operator's decision to preserve A/B independence under time constraint. The parent session's framing of R1 ("use this file for Pass B") was overridden by the dispatch prompt: R1 is *still* Pass A's output and therefore off-limits to Pass B by ADR-0011 §Three-pass methodology. The subagent operated on `design-decisions.md`, `ADR-0004/0008/0010/0011`, `provenance-model.md`, and the M-0122 spec — no R1, no R2, no implementation source.

## Work log

### AC-1 — Catalog file exists at canonical path with top-level sections

`legal-workflows-first-principles.md` written under `docs/pocv3/design/` with 10 top-level sections in the order specified in this milestone's body. `TestM0122_AC1_CatalogExistsAndOrdered` walks the §1–§10 headings and asserts each appears at a monotonically-increasing offset.

### AC-2 — Catalog has per-kind lifecycle section for each entity kind

§1 of the catalog contains six kind-specific subsections (§1a Epic through §1f Contract), each producing R-FP rule rows. `TestM0122_AC2_PerKindSubsectionsPresent` walks the kind list and confirms each subsection is present and contains at least one rule.

### AC-3 — R-FP-NNNN rule rows have non-empty schema fields

176 R-FP rows total across §1–§10. Each row parses to exactly six cells (`Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated`) with non-empty content per cell. `TestM0122_AC3_SixColumnSchemaNonEmpty` uses the `splitTableRow` helper from `m0121_audit_catalog_test.go` (handles backtick-protected pipes and backslash-escapes).

### AC-4 — R-FP-NNNN ids unique and contiguous starting at R-FP-0001

176 distinct ids, contiguous `R-FP-0001` through `R-FP-0176`, all matching the 4-digit canonical shape. `TestM0122_AC4_IdsUniqueAndContiguous` parses every R-FP mention, dedupes, and verifies the sequence.

### AC-5 — Open questions for Pass C section present and non-empty

`## Open questions for Pass C` section at the end of the catalog carries 15 numbered questions (Q1–Q15). `TestM0122_AC5_OpenQuestionsSectionPresent` asserts the section exists and contains at least three numbered questions. The actual count (15) substantially exceeds the floor, and the questions are substantive — most have a sensible default but require a real call during Pass C.

## Decisions made during implementation

- **R1 also off-limits to Pass B.** The parent session's R1 header said "use this file for Pass B" — that was wrong per ADR-0011's "Pass B independent of Pass A's output." The dispatch prompt explicitly overrode the R1 header. Pass B operated on the *abstract* entity model (`design-decisions.md` + relevant ADRs) only. Recorded under §Operator notes.
- **Schema includes a "Reasoning" column.** Pass A's 6-column schema captured `Source / Citation / ...`. Pass B's 6-column schema replaces those with `Reasoning / Load-bearing?` — explicit derivation rationale per rule. Pass C uses this to distinguish principled vs coincidental agreement with Pass A.
- **Subagent dispatched without `isolation: "worktree"` kwarg.** Per CLAUDE.md §Subagent worktree isolation, the parent (this session) operates in the existing epic worktree; the subagent dispatches into that same worktree via absolute path in its prompt. The PreToolUse hook would deny the `isolation` kwarg in any case.

## Validation

- `aiwf check --root .` — 0 errors, 27 warnings. New `acs-tdd-audit` advisories for M-0122's five ACs at `phase: -` under `tdd: advisory` (expected); remainder pre-existing.
- `go test -parallel 8 -short ./...` — all packages green.
- `go test -race -parallel 8 -run TestM0122 ./internal/policies/` — 5/5 passing in ~1.4s. (The macOS test deadlock that previously made `-race` unusable on macOS was resolved by the merge from main's `360387cf` codesign fix; G-0127 is now `addressed`.)
- `golangci-lint run ./internal/policies/` — 0 issues.
- `go build -o /tmp/aiwf-e0033 ./cmd/aiwf` — green.

## Deferrals

None new from this milestone. The 15 open questions in the catalog's closing section are **inputs to M-0123 (Pass C)**, not deferrals — they're the cross-check material this milestone exists to surface.

## Reviewer notes

- **Independence verification.** Spot-checked the catalog for references to Pass A's content (R-AUDIT ids, R-RULE ids, citations from `legal-workflows-audit.md`'s body). Only header/footer framing references Pass A by name (acknowledging its existence and id-space); no content references. The subagent confirmed it did not open R1, R2, or implementation source files except the explicitly-permitted `m0121_audit_catalog_test.go` (used as test-pattern template). Independence held.
- **Rule count comparison.** Pass A produced 225 facets / 156 consolidated rules. Pass B produced 176 rules. The difference is structural (Pass A organizes by source, Pass B by entity-model concept) and content (Pass B may under-count where Pass A captures multiple chokepoints for one underlying claim). Pass C reconciles cell-by-cell, not by count; the comparison is informative but not load-bearing.
- **Schema divergence is deliberate.** Pass B's `Reasoning` and `Load-bearing?` columns don't have a Pass A equivalent. Pass C will need to map Pass A's `Source` + `Citation` against Pass B's `Reasoning` for matched rules.
- **R2 sed updates bundled here.** The merge from main brought `aiwf reallocate G-0128 → G-0130`, which auto-rewrote references in entity frontmatter and bodies but not in `legal-workflows-audit.md` (not an aiwf entity). The 5 catalog substitutions were applied by sed in the parent session post-reallocate; they were uncommitted when M-0122 work began and ride along in this wrap commit.
- **The 15 open questions** are the highest-value Pass B output. Most concern transitions the entity model doesn't pin (Q1 deferred→open, Q2 self-promote, Q3/Q4 direct accepted→rejected) or cascades the model doesn't define (Q5/Q6 cancellation cascades). Pass C decisions on these become discrete decision entities; the reasoning is captured in the catalog's Reasoning column for each affected R-FP rule.
- **No mid-milestone dialog.** The subagent was a one-shot. If Pass B had ambiguities, it captured them as open questions rather than asking. The 15 questions are Pass B's record of those.
