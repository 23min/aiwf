---
id: M-0197
title: Document aiwf-check finding codes + documented-superset chokepoint
status: in_progress
parent: E-0048
depends_on:
    - M-0196
tdd: required
acs:
    - id: AC-1
      title: Every emitted finding code is documented in the aiwf-check skill
      status: open
      tdd_phase: done
    - id: AC-2
      title: A documented-superset chokepoint binds the skill to the emission sites
      status: open
      tdd_phase: done
---
## Goal

The `aiwf-check` skill exists to document the finding codes `aiwf check` emits —
it is the channel an AI assistant or operator consults to interpret a finding it
did not recognize ("what does this code mean, how do I fix it"). Its documented
set has drifted: fourteen emitted codes are undocumented, including the
high-stakes branch-choreography findings `isolation-escape` (and its three
subcodes) and `promote-on-wrong-branch`, exactly the codes whose interpretation
guidance matters most. Nothing mechanical binds the skill's documented set to the
emission sites, so a new finding code can ship undocumented and no chokepoint
notices.

This milestone (a) documents every currently-undocumented emitted code in the
skill, and (b) adds a **documented-superset chokepoint**: a Go policy test that
enumerates the emitted finding-code set and fails if the skill omits any of it.
The enumerator is the one `PolicyFindingCodesHaveHints` already uses to walk
`Finding{}` emission sites and resolve `Code*` constants — extracted to a shared
helper, so "what is emitted" has a single source of truth and the two chokepoints
cannot drift from each other. That shared enumerator is hardened to also resolve
the typed `codespkg.Code{ID: …}` descriptors (referenced as `Code: CodeXxx.ID`
selector expressions), which the current version silently skips — this is why the
branch-choreography findings are invisible to it today.

The chokepoint lives as a Go policy test (CI tier), not an `aiwf check` finding,
because it enumerates Go `Code*` declarations by AST — meaningless in a consumer
tree where `internal/check/` is absent and the skill is materialized rather than
authored. The drift is one-directional (omissions only); the check enforces
documented ⊇ emitted, with a rationale-annotated opt-out for the synthetic
test-fixture codes.

Source: G-0283. Parent epic E-0048.

## Acceptance criteria

### AC-1 — Every emitted finding code is documented in the aiwf-check skill

The `aiwf-check` skill body (`internal/skills/embedded/aiwf-check/SKILL.md`)
carries a meaning + remediation entry for every finding code the check layer
emits. The fourteen codes undocumented at the milestone's start are each added to
the skill's finding-code reference: `acs-shape`, `acs-body-coherence`,
`acs-title-prose`, `refs-resolve`, `fsm-history-consistent`,
`id-path-consistent`, `body-prose-id`, `skill-body-id`,
`milestone-done-incomplete-acs`, `promote-on-wrong-branch`, `isolation-escape`,
`isolation-escape-shallow-clone`, `isolation-escape-oracle-failure`,
`isolation-escape-orphaned-ai-commit`, and `git-config-core-worktree-misset`.
The four `isolation-escape*` branch-choreography findings and
`promote-on-wrong-branch` are the high-stakes ones the skill most needs to
explain.

Mechanical evidence: the AC-2 chokepoint — which asserts the skill's documented
set is a superset of the emitted set — passes. Per CLAUDE.md "AC promotion
requires mechanical evidence", the AC-2 guard is the structural assertion that
fails if any code named here loses its skill entry; there is no separate
substring test for AC-1.

### AC-2 — A documented-superset chokepoint binds the skill to the emission sites

A Go policy test under `internal/policies/` enumerates the emitted finding-code
set and fails if any emitted code is absent from the aiwf-check skill's
documented set. The emitted set is the union of the `Code*` string constants and
the typed `codespkg.Code{ID: …}` descriptors used at `Finding{}` construction
sites across `internal/check/` and `internal/cli/check/`, enumerated through the
**same** walker `PolicyFindingCodesHaveHints` uses (extracted to a shared helper
so the two chokepoints read one source of truth). A rationale-annotated opt-out
list carves out the synthetic test-fixture codes (`a-err`, `z-warn`) that are
never surfaced to a user. Adding a new emitted code without documenting it in the
skill reddens the gate.

Test: (1) a firing fixture — drive the policy against a documented set with one
emitted code removed and assert exactly one violation naming that code (this also
lights the policy's `Violation` construction line for the G-0259 firing-fixture
meta-gate, so no new `grandfatherDark` entry is owed); (2) a seam test that the
shared enumerator, run over the real check packages, includes a typed-descriptor
code (`isolation-escape`) in its emitted set — proving the `Code: CodeXxx.ID`
selector resolution added here actually reaches the branch-choreography findings,
the gap that kept them undocumented; (3) `PolicyFindingCodesHaveHints` continues
to pass over the live tree after the enumerator refactor — no regression to the
hint chokepoint that shares the walker.
