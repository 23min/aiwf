# Epic wrap — E-0044

**Date:** 2026-06-28
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0044-harden-the-areas-feature-for-multi-project-1-1-monorepo-use
**Merge commit:** 15c01a7d

## Milestones delivered

- M-0176 — Partition totality and disjointness property test for areagroup (merged e41fa185)
- M-0177 — aiwf rename-area verb with atomic cross-entity rewrite (merged 7ded0d4c)
- M-0178 — areas.required knob promoting untagged entities to a blocking finding (merged 77dc0855)
- M-0179 — paths per-area config evolution with backward-compatible unmarshaler (merged db38130c)
- M-0180 — Area-path dead-glob and overlap checks (merged df3dbdeb)
- M-0181 — Mistag detection via aiwf-entity trailer with acknowledge path (merged f8894d80)
- M-0182 — Area discoverability skill and path-hint derivation at aiwf add (merged b44dc6ed)
- M-0183 — aiwf set-area verb to tag one entity to a declared area member (merged ad345c97)
- M-0184 — Reserved global area value: predicate, whitelist, and verb acceptance (merged b31c99aa)
- M-0185 — Area-path scoped-coverage check (unslotted-project detection) (merged 257b050d)
- M-0208 — rename-area preserves comments and sibling keys in the areas block on rename (merged 41496fe2)

## Summary

E-0044 hardened the E-0043 label-only `area` tag into a path-backed, trustworthy
filter for the 1:1 project↔area monorepo. It gave each area an optional `paths:`
glob — the oracle the kernel structurally lacked for a purely semantic label —
then built the checks that oracle unlocks: dead-glob, overlap, scoped-coverage
(the M-0185 unslotted-project catch), and mistag. Around them it added the
`areas.required` strictness knob, the `aiwf rename-area` / `aiwf set-area` verbs,
deterministic path-hint area derivation at `aiwf add`, the reserved `global`
cross-cutting sentinel, and the partition-totality property test that makes the
grouping view's silent-drop failure mechanically impossible rather than merely
hoped-for. Net effect: `aiwf list --area <name>` is now a reliable "all work for
that project," not a convenience a silent mislabel could betray. The epic seed
G-0278 is resolved by this work (promoted to `addressed`, resolver E-0044).

## ADRs ratified

- ADR-0020 — Dual-form areas.members schema: backward-compatible label+location evolution (accepted during M-0179)
- ADR-0021 — Sanctioned global area value for inherently-cross-cutting entities (ratified at this wrap)

## Decisions captured

- None as standalone D-NNN. Mid-flight choices were lightweight and recorded in each milestone's `## Decisions made during implementation` section; the area architecture is set by ADR-0020 / ADR-0021. (M-0185's universe model — Option A literal coverage roots, with Option B coverage-globs deferred as a dual-form evolution — is captured in that milestone's Design notes.)

## Follow-ups carried forward

- G-0280 — Pre-commit kernel-policy lint runs the full suite on every commit.
- G-0282 — Inverse-coverage policy: mechanical per-verb chokepoint for what-undoes-this.
- G-0288 — `areas:` config schema has no AI-discoverable doc surface (the full areas-block reference; M-0179/M-0180/M-0185 advanced it with per-field notes).
- G-0307 — top-level `aiwf.yaml` decode stays non-strict (only the `areas:` block rejects unknown keys). (Filed during M-0185 as G-0305; reallocated G-0305 → G-0307 at the epic→main merge to resolve a parallel-worktree id collision with an E-0047 statusline gap.)
- G-0306 — Consolidate the three ack-walker HEAD-walk loops into one primitive.

## Doc findings

Focused doc-lint sweep (the per-milestone doc-lints ran at each milestone wrap;
mechanical doc-integrity — every `aiwf <verb>` mention in a skill resolving,
entity cross-references — is CI-enforced via the `skill_coverage` policy and
`aiwf check`). Epic-level sweep over the area doc surfaces (the `aiwf-check` /
`aiwf-area` skills, ADR-0020 / ADR-0021, the E-0044 entity tree): **clean** — all
relative markdown links resolve; no removed-feature or broken-reference findings.

## Handoff

The 1:1 path-backed area feature is complete and trustworthy. Deliberately left
open: the full areas-schema doc surface (G-0288), the top-level config
strict-decode (G-0307), and the smaller refactors (G-0280, G-0282, G-0306). The
semantic-section (non-1:1) area case stays label-only by design (epic out-of-scope).
No release is bundled with this wrap; cut one via aiwfx-release if desired.

Note: the epic→main merge integrated the parallel E-0045 / E-0047 / E-0048 work
already on trunk (a semantic-merge fix to the new `aiwf check --fast` path's
pre-M-0179 `Areas.Members` API usage; `make ci` + `make coverage-gate` green on
the combined tree).
