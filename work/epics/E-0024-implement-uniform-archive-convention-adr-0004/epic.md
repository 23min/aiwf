---
id: E-0024
title: Implement uniform archive convention (ADR-0004)
status: proposed
---

# E-0024 — Implement uniform archive convention (ADR-0004)

## Goal

Land the `aiwf archive` verb and the convergence machinery so terminal-status entities live under per-parent `archive/` subdirectories, decoupled from FSM promotion, with drift bounded by an advisory check finding plus an optional configurable threshold.

## Context

[ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) was accepted on 2026-05-09 but no implementation work was planned. Today `work/gaps/` carries 80+ entries, most terminal — every directory listing scans past archived noise. The same shape is coming for findings (ADR-0003 companion), decisions, and ADRs as the framework matures.

The ADR commits to: a sweep verb (dry-run default), per-kind storage layout (milestones ride with their epic), drift-control findings, and a one-direction model (no reverse sweep — file a new entity referencing the archived one if you need to revisit). Storage is "one level deep `archive/` subdirectory, alongside active entities of the same parent" per the ADR's storage table.

The first `aiwf archive --apply` invocation in this repo doubles as the historical migration: there is no separate migration verb. The id resolver already needs to span active+archive paths so `Resolves: G-0018`-style references stay live across moves. ADR-0001 (proposed) adds an `inbox/` subdirectory under each kind for pre-mint state — orthogonal to `archive/`; both can coexist.

## Scope

### In scope

- `aiwf archive [--apply] [--kind <kind>] [--root <path>]` verb with dry-run default, single-commit invariant, idempotent re-runs, `aiwf-verb: archive` trailer with no `aiwf-entity:` (multi-entity-sweep exception added to the trailer-keys policy).
- `tree.Tree` loader reads both `<kind>/` and `<kind>/archive/` (and `docs/adr/` + `docs/adr/archive/`).
- `refsResolve` and `internal/entity/refs.go::ForwardRefs` resolve ids across active+archive.
- Three new check-rule findings: `archived-entity-not-terminal` (blocking), `terminal-entity-not-archived` (advisory), `archive-sweep-pending` (aggregate).
- Existing shape/health rules (`acs-shape`, `entity-body-empty-ac`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`) skip `archive/`. Tree-integrity rules (`ids-unique`, parse-level errors) traverse it.
- `archive.sweep_threshold` knob in `aiwf.yaml` schema and parsing — flips `archive-sweep-pending` to blocking past N.
- Display surfaces: `aiwf status` adds a tree-health one-liner *"Sweep pending: N terminal entities not yet archived (run `aiwf archive --dry-run` to preview)"* hidden when 0; `aiwf show <id>` resolves regardless of location and indicates archived state in the render; `aiwf render --format=html` segregates per-kind index pages (active-only default at the home nav; `<kind>/all.html` renders the full set; per-entity pages render regardless of status).
- Embedded `aiwf-archive` skill under `internal/skills/embedded/aiwf-archive/SKILL.md` per the per-verb-skill default in ADR-0006.
- CLAUDE.md "What aiwf commits to" amendment naming the convention.
- Verb help text covering all flags, examples, and the no-reverse-sweep design rule.

### Out of scope

- Wrap-skill nudges in `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` to suggest `aiwf archive --dry-run`. These live in the upstream `ai-workflow-rituals` plugin repo. File a follow-up gap to track the rituals-side work.
- **G-0091** — preventive check for body-prose path-form refs to entity files. Separate concern; the existing post-hoc lychee CI workflow remains the safety net. ADR-0004 explicitly carves it out.
- **G-0092** — doc-authority hierarchy across `docs/`. Filed as a natural follow-up to ADR-0004's doc-archive-scope clarification; does not block this epic.
- A reverse `aiwf reactivate` / `aiwf un-archive` verb. Deliberately omitted per the ADR's Reversal section — the canonical pattern is to file a new entity referencing the archived one.
- A per-id `aiwf archive G-018` verb. Sweep is by status, not by id. Per-id housekeeping was rejected in the ADR's Alternatives section.
- Time-based archive partitioning (`archive/2026-q2/`). Premature for current scale; revisit when archive directories themselves grow unwieldy.
- `aiwf render --format=html` filter chips / JS view-switching. Render-implementation decision deferred to a downstream render milestone per the ADR.

## Constraints

- **No file-move side effect on `aiwf promote` or `aiwf cancel`.** Promotion verbs stay one-purpose. Tests for promote/cancel must not grow archive-aware branches.
- **Single commit per `--apply` invocation.** Per kernel principle #7, one verb invocation produces one commit. Trailer is `aiwf-verb: archive` with no `aiwf-entity:` — the trailer-keys policy in `internal/policies/trailer_keys.go` must learn this exception.
- **Idempotent re-runs.** `aiwf archive --apply` on a clean tree is a no-op (no commit produced).
- **`internal/entity/transition.go::IsTerminal` is the source of truth** for which statuses are terminal per kind. Verb and check rules must consult it; no parallel definition.
- **References stay valid across archive moves.** Loader and resolver MUST find archived entities by id without flag opt-in; only display surfaces (active default, `--archived` opt-in) treat archive as second-class.
- **Forget-by-default for archived entities.** Shape/health rules skip archive; only tree-integrity rules traverse. The kernel does not police per-rule cleanliness inside archive.
- **No reverse sweep verb.** If a contributor hand-edits frontmatter to move a status off terminal, `archived-entity-not-terminal` fires; remediation is to revert the hand-edit, not to relocate the file.

## Success criteria

- [ ] Running `aiwf archive --apply` in this repo sweeps all currently-terminal gaps, decisions, and ADRs into their archive subdirs in one commit; `ls work/gaps/` shrinks to the open set; `aiwf check` reports zero `archive-sweep-pending`.
- [ ] `aiwf show <id>` resolves any terminal entity (now under `archive/`) without `--archived` ceremony; `aiwf history <id>` traces the sweep commit and the entity's earlier history continuously.
- [ ] `aiwf list` defaults to active; `aiwf list --archived` includes archived; `aiwf status` is strictly active-only and prints the sweep-pending tree-health line when applicable.
- [ ] Hand-editing a frontmatter status off-terminal under `archive/` fires `archived-entity-not-terminal` with a remediation message naming the revert path.
- [ ] Setting `archive.sweep_threshold` in `aiwf.yaml` to a value below the current pending count makes `aiwf check` exit non-zero with a `archive-sweep-pending` blocking finding.
- [ ] `aiwf promote` and `aiwf cancel` source code carries no archive-aware branches and their tests carry no archive-aware fixtures.
- [ ] Every milestone listed under *Milestones* below ships with its ACs met and TDD evidence per CLAUDE.md *AC promotion requires mechanical evidence*.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the `aiwf-verb: archive` trailer-without-entity exception require a new `principal_write_sites` policy entry, or is the existing multi-entity allowance sufficient? | no | Resolve in M-2 during verb implementation; the trailer-keys policy test will surface the missing exception if any. |
| Should `aiwf render --format=html`'s archive segregation be a separate milestone or roll into M-4? | no | Held in M-4 for now; if scope creeps the milestone can split during planning. |
| Does the embedded `aiwf-archive` skill need a fixture-validation test alongside the skill-coverage policy? | no | Resolve in M-5 when the skill body is authored; precedent is the M-074/M-079 pattern for embedded-skill drift checks. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Merge edge case: branch A archives `G-0018` (rename to `archive/`) while branch B edits the same file in place. | low | Standard kernel pattern of "merge, run check, fix findings" handles rename+modify conflicts mechanically. Document in the verb's skill body. |
| Existing shape/health rules silently start ignoring archived entities — a regression-class change for any rule that should traverse archive (e.g. `ids-unique`). | medium | M-3 explicitly enumerates which rules skip and which traverse; tests cover both behaviors per rule. |
| First `aiwf archive --apply` in this repo (the historical migration) produces a large commit that's hard to review. | low | Dry-run output is the review surface; the commit body lists per-kind counts and affected ids. Operators can scope via `--kind` to break the migration into smaller commits if desired. |

## Milestones

- [M-0084](work/epics/E-0024-implement-uniform-archive-convention-adr-0004/M-0084-loader-and-id-resolver-span-active-and-archive-directories.md) — Loader and id resolver span active+archive (foundational; nothing else lands cleanly without it) · depends on: —
- [M-0085](work/epics/E-0024-implement-uniform-archive-convention-adr-0004/M-0085-aiwf-archive-verb-dry-run-default-apply-kind.md) — `aiwf archive` verb (dry-run default, `--apply`, `--kind`) with single-commit + trailer-keys exception · depends on: M-0084
- [M-0086](work/epics/E-0024-implement-uniform-archive-convention-adr-0004/M-0086-three-new-archive-check-rule-findings-and-existing-rule-scoping.md) — Three new archive check-rule findings + existing shape/health rules skip `archive/` · depends on: M-0085
- [M-0087](work/epics/E-0024-implement-uniform-archive-convention-adr-0004/M-0087-display-surfaces-for-archived-entities-status-show-render.md) — Display surfaces: `aiwf status` sweep-pending line, `aiwf show` archived indicator, `aiwf render` per-kind index segregation · depends on: M-0086
- [M-0088](work/epics/E-0024-implement-uniform-archive-convention-adr-0004/M-0088-configuration-knob-embedded-skill-and-claude-md-amendment.md) — Configuration (`archive.sweep_threshold` in `aiwf.yaml`) + embedded `aiwf-archive` skill + CLAUDE.md amendment · depends on: M-0087

## ADRs produced

- None. This epic implements the already-accepted ADR-0004; the design decisions are pinned. New ADR-shaped decisions surfaced during implementation will be filed via `aiwfx-record-decision`.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — Uniform archive convention for terminal-status entities (the spec this epic implements).
- [ADR-0006](../../../docs/adr/ADR-0006-skills-policy-per-verb-default-topical-multi-verb-when-concept-shaped-no-skill-when-help-suffices.md) — Skills policy: per-verb default; M-5's embedded `aiwf-archive` skill follows this rule.
- ADR-0001 (proposed) — Mint entity ids at trunk integration; orthogonal to archive but shares the per-kind subdirectory shape.
- ADR-0003 (referenced by ADR-0004) — `finding` (F-NNNN) as a seventh entity kind; co-evolved as the highest-volume archive consumer.
- CLAUDE.md "What aiwf commits to" §2 (stable ids), §3 (pre-push hook is the chokepoint), §5 (correctness must not depend on LLM behavior), §7 (one verb = one commit).
- CLAUDE.md "Designing a new verb" — verb-design rule answered by ADR-0004's Reversal section.
- `internal/entity/transition.go::IsTerminal` — terminal-status source of truth.
- `internal/check/check.go::refsResolve`, `internal/entity/refs.go::ForwardRefs` — id resolution; both extended in M-1.
- `internal/policies/trailer_keys.go` — trailer-keys policy; M-2 adds the `archive` verb's no-entity exception.
- G-0091 (open follow-up) — body-prose path-form ref preventive check; out of scope here.
- G-0092 (open follow-up) — doc-authority hierarchy across `docs/`; out of scope here.
