# Epic wrap — E-0024

**Date:** 2026-05-11
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** worktree-agent-ae85cfc7baf987101
**Merge commit:** <filled at merge step>

## Milestones delivered

- M-0084 — Loader and id resolver span active and archive directories (promote-done `abd7bd0`). `stripArchiveSegment` recognizes the archive segment at the per-kind position; `PathKind`, `IDFromPath`, `refsResolve`, and `ForwardRefs` resolve ids identically across active and `archive/` without flag opt-in.
- M-0086 — Three new archive check-rule findings + existing shape/health rule scoping (promote-done `acb3382`). New findings: `archived-entity-not-terminal` (blocking), `terminal-entity-not-archived` (advisory), `archive-sweep-pending` (aggregate). Seven named shape/health rules skip `archive/`; tree-integrity rules (`ids-unique`, parse-level) traverse it.
- M-0085 — `aiwf archive` verb with dry-run default, `--apply`, `--kind` (promote-done `b40a557`). Single-commit invariant with `aiwf-verb: archive` trailer (no `aiwf-entity:`, multi-entity-sweep exception added to the trailer-keys policy). Idempotent re-runs. `--kind milestone` is a truthful no-op per ADR-0004.
- M-0087 — Display surfaces: status sweep-pending line, show archived indicator, render per-kind index segregation (promote-done `d24fb81`). Discovered and fixed a visibility regression in the first-pass render templates (see *Reviewer notes*).
- M-0088 — Configuration knob, embedded `aiwf-archive` skill, CLAUDE.md amendment (promote-done `827ef0f`). `archive.sweep_threshold` flips the advisory aggregate finding to blocking past N; CLAUDE.md "What aiwf commits to" §10 names the convention.

## Summary

Lands the uniform archive convention per ADR-0004. Terminal-status entities sweep into per-parent `archive/` subdirectories via the new `aiwf archive` verb (dry-run default, `--apply` for the single sweep commit). The loader and id resolver span both active and archive directories so cross-references stay live indefinitely; movement is decoupled from FSM promotion (`aiwf promote` and `aiwf cancel` are unchanged). Drift is policed by three new check rules — one blocking when a hand-edit moves an archived entity off-terminal, one advisory when a terminal entity hasn't been swept yet, and one aggregate (`archive-sweep-pending`) with an opt-in `archive.sweep_threshold` knob in `aiwf.yaml` that flips it to blocking past the named count. Reverse-sweep was deliberately not built (per ADR-0004 §"Reversal" — file a new entity referencing the archived one if a closed entity needs revisiting). The embedded `aiwf-archive` skill and CLAUDE.md §10 land alongside the implementation so AI discoverability moves in lockstep with the verb.

The first `aiwf archive --apply` invocation against the kernel's own tree doubles as the historical migration: dry-run currently previews 89 entities (21 epics, 1 ADR, 67 gaps) ready to sweep into their archive subdirs. The verb's existence makes that a deliberate one-commit action by the operator, not an epic-wrap side effect.

## ADRs ratified

None. The epic implemented the already-accepted ADR-0004 verbatim. Implementation discoveries — the visibility regression caught during human verification (M-0087), the shared `internal/verb/apply.go` MkdirAll-before-Mv change (M-0085), the location-keyed archive segment recognition (M-0084), the `IsArchivedPath` placement on the `entity` package (M-0086), the per-rule scope-list rationale (M-0086) — are captured in the per-milestone *Decisions made during implementation* sections.

## Decisions captured

None as ADR-shaped or D-NNN entries. Per-milestone implementation decisions are recorded in each milestone spec's *Decisions made during implementation* section.

## Validation

- `aiwf check` (run from worktree at HEAD with the freshly-built binary): 176 warnings (0 errors). Breakdown:
  - **174 `terminal-entity-not-archived` advisories** — one per currently-terminal entity in the active tree (21 epics + 1 ADR + 67 gaps + 85 milestones that ride with their parent epics). This is exactly the historical-migration backlog the new advisory rule is designed to surface; the count clears the moment an operator runs `aiwf archive --apply`. Per the rule's design (M-0086), the finding is advisory and stays non-blocking until `archive.sweep_threshold` is set in `aiwf.yaml`.
  - **`gap-resolved-has-resolver` on `G-0093`** — addressed but lacks `addressed_by` / `addressed_by_commit`. Pre-dates this epic and is explicitly out of scope per M-0086's *Decisions made during implementation* (the rule traverses archive deliberately; M-0086 named a specific seven-rule scope-list).
  - **`provenance-untrailered-scope-undefined`** — worktree branch has no upstream configured; expected.
- `golangci-lint run` — 0 issues.
- `go build -o /tmp/aiwf ./cmd/aiwf` — exit 0.
- `aiwf archive --dry-run` against the kernel previews 89 entities cleanly (21 epic, 1 adr, 67 gap), one move per archived item. The historical migration is now operator-deliberate, not an epic-wrap side effect.

Tests skipped at the user's request (recorded reason: G-0097, test-suite parallelism gap). All per-milestone validation sections recorded green test runs at the time each milestone wrapped.

## Follow-ups carried forward

None. The milestone wraps did not open new gaps. The two cross-cutting follow-ups named in the epic scope (G-0091 — body-prose path-form ref preventive check; G-0092 — doc-authority hierarchy across `docs/`) remain open as planned, not opened by this epic.

The out-of-scope rituals-side nudge (suggest `aiwf archive --dry-run` from `aiwfx-wrap-epic` / `aiwfx-wrap-milestone`) is not filed as a kernel gap because the work lives in the upstream `ai-workflow-rituals` plugin repo. The epic spec's *Out of scope* section names it as the deliberate carve-out.

## Reviewer notes

Two trade-offs worth surfacing:

1. **M-0086's per-rule scope-list is the AC-named seven, not an exhaustive sweep.** ADR-0004 §"Check shape rules" mandates "named explicitly per rule — no global 'skip if path contains archive' shortcut." M-0086 AC-4 enumerated the seven rules to scope (`acs-shape`, `entity-body-empty-ac`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`, `epic-body-empty-milestone-table`). Other plausibly-shape-and-health rules — `gap-resolved-has-resolver`, `titles-nonempty`, `adr-supersession-mutual`, `id-path-consistent`, `acs-title-prose`, `status-valid` — continue to traverse archive. This is a **deliberate non-scope**, not a defect: the AC pinned the seven, and the M-0086 spec explicitly named the carved-out rules in its *Decisions made during implementation* section. A future milestone may re-examine each individually. The `gap-resolved-has-resolver` warning on `G-0093` in the post-wrap `aiwf check` is the visible consequence and is correct behavior.

2. **The first apply against the kernel's tree is the operator's call.** Wrap closes the planning unit; it does not run `aiwf archive --apply`. The dry-run preview is the verification surface here. Whether to commit the 89-entity historical migration as one sweep or as several `--kind`-scoped commits is an operator decision left for after the wrap — both shapes are supported by the verb's design.

## Handoff

The archive convention is live. Operators can run `aiwf archive --dry-run` at any time; `--apply` lands the sweep as a single commit. The advisory `archive-sweep-pending` finding will keep the count visible until the first apply lands. Open epics and their milestones continue to live in the active per-kind directories until they reach terminal status, at which point the next sweep moves them.

Out-of-scope follow-ups (G-0091 body-prose path-form ref check, G-0092 doc-authority hierarchy, and the rituals-plugin wrap-nudge) remain available for separate planning.
