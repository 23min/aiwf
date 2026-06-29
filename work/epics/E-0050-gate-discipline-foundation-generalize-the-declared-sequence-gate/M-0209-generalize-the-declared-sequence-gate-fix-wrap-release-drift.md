---
id: M-0209
title: Generalize the declared-sequence gate; fix wrap/release drift
status: in_progress
parent: E-0050
tdd: advisory
acs:
    - id: AC-1
      title: Generalized declared-sequence gate documented in CLAUDE.md and guidance
      status: met
    - id: AC-2
      title: aiwfx-release splits the two origin pushes into separate push gates
      status: open
    - id: AC-3
      title: aiwfx-wrap-milestone batches its terminal local steps in one gate
      status: open
    - id: AC-4
      title: 'wrap-epic: one declared-sequence gate for merge+commit+promote; split deletes'
      status: open
---

## Goal

Generalize the wf-patch declared-sequence gate (CLAUDE.md §"Gate discipline
survives compaction") into a general capability for any sequence of local,
reversible mutations — one gate that enumerates every action verbatim, binds
approval to exactly that list (subset approval allowed), and aborts + re-gates on
any deviation. Document the standing rule in CLAUDE.md's gate-discipline section
and `.claude/aiwf-guidance.md`, rewriting the false "wf-patch only; milestone and
epic wraps keep per-action gates" scope sentence.

Then fix the three rituals that violate it today: `aiwfx-release` (split the
bundled two-push gate into two separate push gates), `aiwfx-wrap-milestone` and
`aiwfx-wrap-epic` (replace the ungated promote / merge / cleanup steps with the
declared-sequence gate, push excluded). The bright line — batch local, reversible
mutations; exclude outward / irreversible actions and timing-bearing mutations
(`tdd: required` phase promotes fire live) — is the load-bearing safety claim,
pinned by structural tests under `internal/policies/`.

Source: G-0295. Extracted from E-0049 into foundation epic E-0050 so both E-0048
and E-0049 milestone wraps inherit the corrected gate.

## Acceptance criteria

### AC-1 — Generalized declared-sequence gate documented in CLAUDE.md and guidance

Rewrote CLAUDE.md's gate-discipline paragraph from the false "wf-patch only;
milestone and epic wraps keep per-action gates" into the generalized
declared-sequence gate (any local, reversible sequence; enumerate-verbatim;
subset-approvable) with the bright line excluding outward/irreversible and
timing-bearing actions. Mirrored the rule into the embedded guidance source.
Evidence: `TestM0209_AC1_GeneralizedGateInClaudeMd` + `…InGuidance`.

### AC-2 — aiwfx-release splits the two origin pushes into separate push gates

Split the bundled step-6 push into two gates — push-commit (step 6) and push-tag
(step 7) — removed the bundled "Push the commit and the tag" prompt, renumbered
the trailing steps. Evidence: `TestM0209_AC2_ReleaseSplitsPushGates`.

### AC-3 — aiwfx-wrap-milestone batches its terminal local steps in one gate

Folded the ungated promote + ungated merge into one declared-sequence gate over
the terminal local sequence (merge → promote-done → cleanup); push and origin
delete are separate outward gates. Preserved the existing trailered-merge contract
test. Evidence: `TestM0209_AC3_WrapMilestoneDeclaredSequenceGate`.

### AC-4 — wrap-epic: one declared-sequence gate for merge+commit+promote; split deletes

Replaced the separate merge-gate + commit-gate + ungated-promote with one
declared-sequence gate over merge → wrap-artefact commit → promote-done; split the
batched origin-branch deletes into per-action gates. Preserved AC-2/AC-6 merge
contract and the G-0119 promote-last ordering (locator updated). Evidence:
`TestM0209_AC4_WrapEpicDeclaredSequenceGate`.

## Work log

- AC-1 — CLAUDE.md + `internal/skills/embedded-guidance/aiwf-guidance.md` · new test `m0209_declared_sequence_gate_test.go`
- AC-2 — `aiwfx-release/SKILL.md` · tests 4/4 of M-0209 green
- AC-3 — `aiwfx-wrap-milestone/SKILL.md` · existing merge-step test preserved
- AC-4 — `aiwfx-wrap-epic/SKILL.md` + `aiwfx_wrap_epic_test.go` locator update

Commit SHAs recorded at wrap.

## Validation

- `go test ./internal/policies/` — pass (full package, incl. all `TestM0209_*` and
  the preserved `TestAiwfxWrapEpic_*` / `TestAiwfxWrapMilestone_*`).
- `go test ./internal/skills/` — pass.
- `golangci-lint run ./internal/policies/` — 0 issues.
- `aiwf check` (worktree) — 0 errors.
- Diff-scoped coverage gate: n/a — no production Go lines changed (markdown +
  test files only).

## Reviewer notes

- `tdd: advisory`, but every AC carries a structural test (red→green verified) per
  the mechanical-evidence rule; the test-discipline obligation is not waived.
- Tests are section-scoped structural assertions, not flat greps, per CLAUDE.md
  §"Substring assertions are not structural assertions".
- Edits are to the *embedded* ritual source; consumers see them after `aiwf
  update`. The "stage implementation, not just the spec" wrap-ritual fix is
  G-0293/E-0049, out of scope here — so this wrap stages the implementation
  explicitly.
- Independent fresh-context review (Sonnet subagent) over the diff returned
  REQUEST-CHANGES: 3 blocking (a stale `(step 8)`→`(step 7)` back-reference, and
  two prose orderings that listed promote before merge, contradicting the
  G-0119-mandated merge→promote order) + 2 non-blocking (AC-1 guidance test was a
  file-wide grep; AC-3 test didn't pin merge-before-promote). All five fixed as
  corrective edits before this wrap commit; mechanical fixes re-confirmed by green
  tests + clean lint.

