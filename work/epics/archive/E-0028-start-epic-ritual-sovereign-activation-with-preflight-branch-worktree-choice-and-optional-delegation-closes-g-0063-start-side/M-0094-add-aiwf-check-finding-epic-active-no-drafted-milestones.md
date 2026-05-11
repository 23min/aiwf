---
id: M-0094
title: Add aiwf check finding epic-active-no-drafted-milestones
status: done
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: rule fires warning when active epic has zero drafted milestones
      status: met
      tdd_phase: done
    - id: AC-2
      title: rule does not fire when active epic has at least one drafted milestone
      status: met
      tdd_phase: done
    - id: AC-3
      title: rule does not fire when epic is at status proposed, done, or cancelled
      status: met
      tdd_phase: done
    - id: AC-4
      title: finding hint text references the start-epic preflight role and G-0063
      status: met
      tdd_phase: done
---

# M-0094 — Add `aiwf check` finding `epic-active-no-drafted-milestones`

## Goal

Add a new `aiwf check` warning finding `epic-active-no-drafted-milestones` that fires when an epic at status `active` has zero milestones at status `draft`. The finding informs the `aiwfx-start-epic` skill's preflight: an epic is not ready for activation until at least one milestone is drafted.

## Context

E-0028's start-epic ritual needs a kernel-checkable signal for "is this epic actually ready to activate?" The body-completeness check (`entity-body-empty`, M-0066) already covers Goal/Scope/Out-of-scope prose. The remaining preflight gap per G-0063 is the drafted-milestone check — that an epic going active has work queued, not just text on a page.

This is a standalone kernel rule. It does not depend on M-0095 (sovereign-act enforcement); the two land in parallel and the skill in M-0096 consumes both.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- One new finding code: `epic-active-no-drafted-milestones`, severity warning.
- Trigger: any epic with `status: active` whose child milestone set contains zero entries at `status: draft`.
- Hint text points at G-0063's framing and the skill's preflight role.
- Implementation lives in `internal/check/` alongside the existing kind-scoped rules; tests under `internal/check/<name>_test.go` follow the fixture-tree convention used by `entity_body_test.go`.

## Dependencies

- None. Standalone kernel rule; lands in parallel with M-0095.

## References

- E-0028 epic spec.
- G-0063 — preflight checks table, row 2.
- `internal/check/entity_body.go` and `_test.go` — reference shape for new check rules.

### AC-1 — rule fires warning when active epic has zero drafted milestones

A fixture tree under `internal/check/testdata/` with an epic at status `active` and zero milestones at status `draft` (zero milestones at all is the simplest positive shape; "zero drafts among existing milestones" is the same case under reading A). Driving the test through `check.Run` (per CLAUDE.md *Test the seam, not just the layer*) produces a `findings.Finding` with code `epic-active-no-drafted-milestones`, severity `warning`, target the epic's id.

### AC-2 — rule does not fire when active epic has at least one drafted milestone

A sibling fixture with the same `active` epic plus ≥1 milestone at status `draft`. Driving through `check.Run`, the result set contains no finding with code `epic-active-no-drafted-milestones`. Covers the negative branch — drafts present, no warning.

### AC-3 — rule does not fire when epic is at status proposed, done, or cancelled

Table-driven test across each non-`active` epic status — `proposed`, `done`, `cancelled` — each with zero drafted milestones (the firing condition under reading A). For every status, the result set contains no finding with code `epic-active-no-drafted-milestones`. Covers the kind/status guard; ensures the rule is scoped exactly to `active` epics, not to all epics-with-no-drafts.

### AC-4 — finding hint text references the start-epic preflight role and G-0063

The finding's hint surface mentions G-0063 (gap framing) and the start-epic preflight role (so a reader who lands on the finding via `aiwf check` can navigate to the framing without re-deriving it). Structural assertion on the hint string (substring grep is acceptable here per CLAUDE.md *Substring assertions are not structural assertions* — the hint surface is a single short string, not a structured document where placement matters).

## Work log

<!-- Phase timeline lives in `aiwf history M-0094/AC-<N>`; the entries here capture
     one-line outcomes + the implementing commit's SHA (filled at wrap when the
     implementation lands as a single commit). -->

### AC-1 — rule fires warning when active epic has zero drafted milestones

Rule implemented in `internal/check/epic_active_drafts.go` as `epicActiveNoDraftedMilestones(*tree.Tree) []Finding`; wired into `check.Run` between `gapResolvedHasResolver` and the I2 AC checks. Seam-level test in `internal/check/epic_active_drafts_test.go` drives `check.Run` against an in-memory tree with one active epic and zero milestones, asserting the finding's code, severity (warning), and entity-id. · commit <wrap> · tests 1/1.

### AC-2 — rule does not fire when active epic has at least one drafted milestone

Negative-case test in the same file: tree has E-0001 active + M-0001 (in_progress, parent E-0001) + M-0002 (draft, parent E-0001). Drives through `check.Run`; asserts no finding with the new code. Covers branch C (`m.Status == StatusDraft → hasDraft = true`) and branch D-true (hasDraft → continue to next epic). · commit <wrap> · tests 1/1.

### AC-3 — rule does not fire when epic is at status proposed, done, or cancelled

Table-driven test across the three non-active statuses; each subcase has zero drafted milestones (the firing condition under reading A). Asserts no finding fires. Covers branch A (epic status guard — `Status != StatusActive → continue`). · commit <wrap> · tests 3/3.

### AC-4 — finding hint text references the start-epic preflight role and G-0063

Hint added to `internal/check/hint.go`'s `hintTable` under key `epic-active-no-drafted-milestones`. Test asserts (via substring grep on the hint string, justified per CLAUDE.md since the hint is a short single string) that "G-0063" and "start-epic" both appear. · commit <wrap> · tests 1/1.

### Branch-coverage extra — ignores milestones under other epics

Added `TestEpicActiveNoDraftedMilestones_IgnoresMilestonesUnderOtherEpics` covering branch B's skip arm (a draft milestone whose parent differs from the epic under consideration must not satisfy that epic's "has draft" check). Closes the last reachable conditional branch in the rule. · commit <wrap> · tests 1/1.

### Sibling artefact updates

- `internal/skills/embedded/aiwf-check/SKILL.md` — new row in the warnings table for `epic-active-no-drafted-milestones`. Required by the discoverability policy (`TestPolicy_FindingCodesAreDiscoverable`) which fires whenever a kernel finding code is undocumented in an AI-discoverable channel.
- `internal/check/testdata/messy/work/epics/E-02-no-drafts/epic.md` — new fixture entity so the messy fixture's expected-codes assertion exercises the rule (the existing E-01 collision shared an id and cross-pollinated milestones).
- `internal/check/fixtures_test.go` — `epic-active-no-drafted-milestones` added to the `expected` list documenting messy-fixture coverage.
- `internal/verb/projection_test.go` — `TestProjectionFindings_PreExistingFiltered`'s fixture extended with a drafted milestone so the verb-projection test's premise (an unrelated active epic is benign) survives the new rule.

## Decisions made during implementation

- Reading A (strict-literal: "fires whenever active epic has zero `draft` milestones") chosen over reading B ("no forward motion") and reading C ("activation-moment only, skill enforces"). Rationale and tradeoffs in the conversation; the rule's name and the kernel chokepoint principle both favor A.

## Validation

- `go test -race -count=1 ./...` — all packages green; zero FAIL lines.
- `golangci-lint run ./...` — 0 issues.
- `go test -coverprofile=… ./internal/check/` — `epicActiveNoDraftedMilestones` at 100.0% statement coverage; branch-coverage audit walked the rule's `if/continue/break` arms and matched each to a named test (AC-1 positive, AC-2 has-draft, AC-3 non-active status, AC-4 hint, plus the `IgnoresMilestonesUnderOtherEpics` branch-coverage extra).
- `aiwf check` (kernel planning tree from the worktree) — clean modulo the benign `provenance-untrailered-scope-undefined` warning (worktree branch has no upstream; expected pre-push).
- Discoverability gate — `TestPolicy_FindingCodesAreDiscoverable` passes against the new code's row in `internal/skills/embedded/aiwf-check/SKILL.md`.
- Cross-binary regression — `cmd/aiwf` binary tests (`TestBinary_CheckVerbose_ByteIdenticalToBaseline`, `TestBinary_CheckJSON*`, `TestBinary_CheckDefault_SummarizesWarnings`) pass with regenerated goldens + updated expected-counts (10 errors, 9 warnings, 19 findings for the messy fixture after the E-02-no-drafts addition).
- `wf-doc-lint` (scoped to the change-set) — no broken references, no removed-feature docs, no doc-side drift.

## Deferrals

- (none)

## Reviewer notes

- **Reading-A semantics.** The rule fires whenever an `active` epic has zero `draft` milestones, *not* "zero non-terminal." Late in an epic (all milestones in flight or done, none drafted) the warning persists as a forward-motion nudge — drafting the next milestone or wrapping the epic clears it. This was an explicit design choice (planning conversation, Decision 1 of the M-0094 readiness Q&A); reading-B and reading-C were rejected for, respectively, rule-name/semantics drift and the kernel-chokepoint principle.
- **Cross-pollination in the messy fixture.** Before this milestone, the messy fixture had two epics sharing id `E-01` (the `ids-unique` collision case). My rule's parent-match logic treats both epics as having the *same* drafted milestones (M-001/M-002 with `parent: E-01`), so neither fires. A separate `E-02-no-drafts/epic.md` was added so the rule has a clean firing case. This is the right shape: the kernel correctly avoids overfiring on the ambiguous-id case; the fixture's job is to give every documented code a non-ambiguous trigger.
- **Trailer convention for the wrap commit.** Per the M-0089 precedent (commit `de94cf9`), the wrap commit's trailers are `aiwf-verb: implement`, `aiwf-entity: M-0094`, `aiwf-actor: human/<id>`. The `implement` verb is a synthetic trailer-only marker (no CLI surface) signaling "this commit is the implementation work for this entity." Documented here so the next milestone doesn't re-derive it.
