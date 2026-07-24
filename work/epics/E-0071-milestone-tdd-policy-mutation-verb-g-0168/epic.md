---
id: E-0071
title: Milestone tdd-policy mutation verb (G-0168)
status: proposed
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

## References

- D-0048 — the governing decision (verb surface, uniform-ordinary gating, defer
  the rest).
- G-0168 — the originating gap (four set-at-create fields lacking mutation
  verbs).
- G-0442 — the split-out set-at-transition amend problem (out of scope here).
- `aiwf milestone depends-on` — the existing subverb precedent this verb mirrors.
