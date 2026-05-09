---
id: M-082
title: Implement aiwf rewidth verb and apply to this repo's tree
status: draft
parent: E-23
depends_on:
    - M-081
tdd: required
---
## Goal

Implement `aiwf rewidth`, the migration verb that takes a consumer's narrow-width tree to canonical 4-digit form. Distributed with the kernel binary; idempotent; one commit per `--apply`. Active-tree only; archives untouched. Apply the verb to this repo's tree as M-B's wrap deliverable, producing a single rename + reference-rewrite commit.

After M-B ships, this repo runs at canonical width. Downstream consumers gain a tested distribution of the verb to migrate their own trees on demand. New consumers post-graduation continue to be born canonical (per M-A's allocator).

## Context

ADR-0008 specifies the verb shape: top-level Cobra command, dry-run default, `--apply` commits one transaction, active-tree only, idempotent. M-A made the parser tolerate narrow widths (so the verb can read a narrow tree) and made the allocator emit canonical (so post-migration files are uniform). With those in place, M-B implements the verb's logic and proves it against real data — this repo's own tree.

The verb's reference-rewrite engine handles three concrete patterns: id-form mentions in prose, composite ids (`M-NN/AC-N`), and markdown links targeting active-tree paths. Code fences and archive paths are excluded — the rule is "rewrite active-tree references in active-tree files, leave everything else alone."

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **TDD: required.** Each AC drives a red→green→refactor cycle. AC-3's reference-rewrite engine in particular needs careful test coverage of edge cases (code fences, trailing-digit guards, archive-path exclusion).
- **Pure forward motion.** The verb takes narrow → canonical. No "narrow it back" path. Reversal is `git revert` on the migration commit.
- **Single commit per `--apply` invocation.** Per kernel principle #7. Multi-entity sweeps are a special case in the trailer-keys policy (same shape as `aiwf archive`); trailer is `aiwf-verb: rewidth` with no `aiwf-entity:` trailer.
- **Active tree only.** `<kind>/archive/` files are skipped for renaming; archive paths are skipped for rewriting. ADR-0004's forget-by-default principle for archives is preserved.
- **Idempotent.** Running on an already-canonical or empty tree is a no-op; no commit produced.
- **Skill coverage allowlisted, not per-verb skill.** ADR-0006 case "no skill when --help suffices" applies — the verb is one-shot and self-documenting.
- **Cobra completion drift test passes.** New verb threads through `cmd/aiwf/completion_drift_test.go`.

## Design notes

### Walk order and determinism

The verb walks kinds in a fixed sequence (`epic, milestone, gap, decision, contract, adr` — composite-parent kinds last) and entities within a kind in alphabetical order by current filename. This determinism makes idempotence testable: a second invocation on the same tree visits files in the same order and produces no operations.

### Reference-rewrite scope

Three patterns rewritten:

- **Id-form mentions in prose.** Regex matches `\b[EMGDCF]-[0-9]{1,3}\b` (narrow forms only) and rewrites to canonical 4-digit. Trailing-digit guard via word boundaries: `E-22` matches but `E-220` does not; `E-2200` doesn't match either. Composite-id mentions (`M-NN/AC-N`) are detected separately to avoid double-rewriting.
- **Composite ids.** Regex `\bM-[0-9]{1,3}/AC-[0-9]+\b` rewrites the milestone portion to canonical; AC portion preserved.
- **Markdown links to active-tree paths.** Regex matches `\(work/<kind>/[EMGDCF]-[0-9]{1,3}-<slug>(?:\.md)?\)` and rewrites the embedded id to canonical. Links to `<kind>/archive/...` paths excluded.

**Code fences excluded.** A markdown parser identifies fenced code blocks; content inside fences is not rewritten. Inline backtick spans are also excluded (id mentions inside `` `E-22` `` stay as-is — they typically denote literal id text in documentation).

**Archive paths excluded.** Markdown links targeting `work/<kind>/archive/...` are not rewritten regardless of the embedded id width.

### Apply to this repo's tree

The wrap PR for M-B contains both the verb's source code + tests AND the result of running `aiwf rewidth --apply` against this repo's tree. The diff includes:

- File renames in `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/` from narrow to canonical width.
- Body content rewrites in active-tree markdown files (id mentions, composite ids, markdown links).

Manual diff review is a named checkpoint in the wrap commit body. `aiwf check` and lychee CI green are gates.

## Surfaces touched

- `cmd/aiwf/rewidth_cmd.go` (new) — Cobra command definition.
- `internal/verb/rewidth.go` (new) — verb implementation: walk, rename, rewrite.
- `internal/verb/rewidth/` (new package, optional) — reference-rewrite engine if it grows substantial.
- `cmd/aiwf/completion_drift_test.go` — entry for new verb (auto-discovered or explicit).
- `internal/policies/skill_coverage.go` — allowlist entry for `rewidth` with rationale.
- `internal/policies/<test>` — drift-prevention test if appropriate.
- This repo's `work/`, `docs/adr/` — file renames + body rewrites at wrap time.

## Out of scope

- The drift-check rule `entity-id-narrow-width` — that's M-C.
- ADR-0003 amendment — that's M-C.
- CLAUDE.md commitment #2 update — that's M-C.
- Embedded skill content refresh — that's M-C.
- Rituals plugin coordination — that's M-C.
- Doc-tree narrow-id sweep beyond what `aiwf rewidth` handles automatically — M-C handles `docs/`, `README.md`, `CHANGELOG.md` updates if they're outside the active-tree scope of `aiwf rewidth`.
- Reverse path (`canonical → narrow`). No use case; not implemented.
- Width 5 or 6 future-proofing — YAGNI per ADR-0008.
