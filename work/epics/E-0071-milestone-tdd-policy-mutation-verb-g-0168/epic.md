---
id: E-0071
title: Milestone tdd-policy mutation verb (G-0168)
status: active
---

# E-0071 — Milestone tdd-policy mutation verb (G-0168)

## Goal

Give a milestone's `tdd:` policy a proper post-creation mutation verb, closing
the `tdd:` portion of G-0168's verb-chokepoint hole. Changing the policy today
requires a hand-edit that bypasses the kernel's one-verb-per-mutation
convention (a fictional `aiwf-verb:` trailer, a path `--help` never reveals);
this epic makes it a first-class, trailered, discoverable act.

## Context

G-0168 identified four frontmatter fields set only at `aiwf add` time with no
post-creation mutation verb: milestone `tdd:`, gap `discovered_in:`, decision
`relates_to:`, contract `linked_adrs:`. D-0048 settled the design: build the
`tdd:` verb now, defer the three relation-field editors (per-kind subverbs,
when friction appears), and split the set-at-transition amend problem
(`addressed_by` / `superseded_by`) to G-0442.

`tdd:` is first because it is the only one with demonstrated friction — twice:
the M-0120 downgrade, and a later re-discovery from the upgrade direction. The
prerequisites that would have blocked a `milestone` subverb are all resolved:
G-0285 (root-banner drift guard), G-0284 (skill-coverage for namespace
subverbs), and G-0286 (relaxed `acs-shape/tdd-phase` so an absent phase is
legal until an AC is `met`).

## Scope

### In scope

- `aiwf milestone tdd <M-id> --policy none|advisory|required [--reason "..."]`
  — a post-creation mutator for a milestone's TDD policy, following the
  `aiwf milestone depends-on` subverb idiom.
- **Uniform-ordinary gating** (per D-0048): any actor (human, or an authorized
  `ai/` with a principal), an optional `--reason`, standard trailers, and no
  directional or sovereign carve-out — weakening is treated identically to
  strengthening.
- Policy-value validation against the closed set `{none, advisory, required}`.
- **Refuse-with-hint** when a flip to `required` would strand an already-`met`
  AC without `tdd_phase: done`: the verb names the offending ACs and aborts,
  never auto-seeding a phase.
- Discoverability: `aiwf milestone --help`, the root `--help` banner, `--policy`
  shell completion, and skill coverage.

### Out of scope

- The three relation-field editors — `discovered_in` / `relates_to` /
  `linked_adrs`. Deferred per D-0048 until real friction; their shape is
  already fixed (per-kind subverbs, not a generic verb).
- The set-at-transition amend verbs for `addressed_by` / `superseded_by` — a
  distinct problem tracked in G-0442.
- Any generic `aiwf relate --field <name>` multiplexer (rejected in D-0048).
- Graduating the uniform-ordinary / check-layer-governance principle from
  D-0048 to an ADR — a D-0048 follow-up, only if it proves load-bearing across
  more verbs.

## Constraints

- Uniform-ordinary gating is non-negotiable: no directional or sovereign
  carve-out. The sovereign-act tier stays keyed on FSM status edges only — this
  verb does not add a data-field entry to it.
- The verb never auto-seeds an AC's `tdd_phase`; it refuses with an actionable
  hint instead. Manufacturing a phase (`red` or `done`) on an untouched or
  already-passed AC would record false state.
- Standard kernel conventions: exactly one commit per mutation with
  `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers; completion wiring; skill
  coverage per ADR-0006.

## Success criteria

- [ ] An operator — human or authorized agent — can change a milestone's TDD
      policy in either direction with a single trailered `aiwf milestone tdd`
      command, and `aiwf history` renders it as a real verb.
- [ ] No path to change a milestone's TDD policy requires hand-editing
      frontmatter.
- [ ] The verb is reachable via `--help`, the root banner, shell completion,
      and a skill — discoverable without grepping source.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Verb spelling: `milestone tdd --policy <x>` vs `milestone set-tdd <x>` | no | Settled — `milestone tdd --policy <x>`: mirrors the `milestone depends-on` subverb precedent and completes a closed-set value in a flag rather than a bare positional. |

## Milestones

- `M-0277` — the `aiwf milestone tdd` verb: mutation + policy validation +
  uniform-ordinary gating + refuse-with-hint + discoverability. · depends on: —

## Cross-gap coordination (at wrap)

Three related gaps interact with this epic. None expands its build scope; each
is a wrap-time action recorded here so the wrap ritual carries it forward. The
two gap-body edits below only become true once M-0277 lands, so they are made at
wrap, not now.

- **G-0168** (originating parent) — this epic closes only the `tdd:` slice of
  its four set-at-create fields. The three relation editors (`discovered_in`,
  `relates_to`, `linked_adrs`) stay deferred per D-0048; G-0442 is a separate
  sibling (the set-at-transition pair), not G-0168's residual tracker. At wrap:
  leave G-0168 `open` — do not promote it to `addressed` — and note that the
  `tdd:` slice landed via M-0277.
- **G-0121** (legal-workflow composition) — M-0277's stress-walker acceptance
  criterion adds `milestone tdd` to the verb-sequence walker as an always-legal
  op. That is the shallow stepping-stone, not G-0121's named invariant ("no AC
  `met` under `tdd: required` with `tdd_phase ≠ done` after any legal
  sequence"): the walker seeds no ACs, so it cannot reach that state. At wrap:
  update G-0121's body to record that the walker now exercises tdd-policy flips,
  and name the AC-composition invariant-fuzz as the follow-on this epic unblocks
  — a standalone G-0121 milestone, deliberately not folded here.
- **G-0166** (verb-time rejection layer) — M-0277's refuse-with-hint acceptance
  criterion is a new verb-time rejection: the milestone-policy-side mirror of
  the AC-side `acs-tdd-audit` cell. It is not registered in the
  `internal/workflows/spec/` table by this epic — that table keys on
  `(Kind, FromState, Verb)` with an FSM status `FromState` and cannot yet model
  a data-field-mutation rejection; extending it is G-0166's own work. At wrap:
  note that refuse-with-hint criterion in G-0166 as a candidate cell for that
  systematization. Its own evidence stays a standalone verb-time test in M-0277.

## References

- D-0048 — the governing decision (verb surface, uniform-ordinary gating, defer
  the rest).
- G-0168 — the originating gap (four set-at-create fields lacking mutation
  verbs).
- G-0442 — the split-out set-at-transition amend problem (out of scope here).
- G-0121 — legal-workflow composition; the verb-sequence walker this epic
  extends (see Cross-gap coordination).
- G-0166 — verb-time rejection-layer systematization; AC-4 adds a mirror cell
  (see Cross-gap coordination).
- `aiwf milestone depends-on` — the existing subverb precedent this verb mirrors.
