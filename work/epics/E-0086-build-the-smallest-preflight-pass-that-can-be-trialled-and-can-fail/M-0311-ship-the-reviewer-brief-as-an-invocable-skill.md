---
id: M-0311
title: Ship the reviewer brief as an invocable skill
status: draft
parent: E-0086
tdd: required
---
## Goal

Ship the instruction sheet a dispatched preflight reviewer receives, as a skill
any session can invoke, so what a review asks for stops varying with who
dispatches it.

## Context

The pass has only ever been dispatched with a brief retyped from the
specification by the session doing the dispatching. The specification states the
exposure: "independence is what makes the audit's breadth affordable, and the
brief is where independence is established or lost."

The brief is built first because both seams dispatch it, and neither seam can be
written before the thing it dispatches exists.

## Acceptance criteria

### AC-1 — The brief ships as a skill carrying a section per part

`wf-preflight-brief` ships under `internal/skills/embedded-rituals/`, with one
section for each of the parts the specification fixes:

1. The two snapshots — corpus ref and subject, named separately
2. Selecting sources, and recording the selection with a reason per source
3. The prohibition on measuring, and recording each measurable question with the
   command it would run
4. The three questions — contradiction, complication, silent assumption
5. Checking each attribution against its source, reported supported or unsupported
6. Reporting the selected sources the subject cites nowhere
7. The report — the ledger's columns, the four reading states, no verified value
8. What whoever dispatches it must do — withhold the outcome

A structural test asserts a heading exists for each. D-0050 names this form:
"a heading exists, a section holds N sub-headings, a labelled paragraph opens a
line, a cross-reference resolves to a target that exists."

An empty section passes, and no assertion here reads a section's content. That
floor is the criterion's, not an oversight: no rule in the corpus forces a shape
on the prose inside, and the epic's constraint is that where none does, the
criterion says so rather than faking one. Content is held at the wrap review.

### AC-2 — The brief names no location absent from a consumer tree

A structural test asserts the brief carries no path token outside the set aiwf
itself creates in a consumer repository — `work/`, `docs/adr/`, `.claude/`,
`aiwf.yaml`. The brief states its corpus by property instead, in the form
ADR-0044 supplies: "a reader falls back to describing the corpus by property —
documents the project maintains as current truth, excluding generated output and
archived snapshots."

The test is scoped to this one file, frontmatter included. G-0548 owns the
tree-wide sweep and records the population this scoping leaves untouched:
"design docs under `docs/`, package directories under `internal/`, source files
under `cmd/`."

## Constraints

- **The initiative is the specification.** A claim here it does not carry is a
  defect in one of the two.
- **This milestone's own preflight pass ran before it started** — challenge,
  three blind sweeps at a pinned corpus, and a lab on the rows the sweeps
  returned unknown. Its ledger is below.
- **The brief carries no rationale, no development history, and no real entity
  id or path**, per the shipped-surface rule.
- **The mandate AC-1 creates retires with the sweep.** When `wf-preflight-sweep`
  ships and absorbs the reading method, parts 2 and 4 move to it and this
  milestone's heading list is superseded rather than extended. Owner: the epic.
- **No kernel change.** No finding code, no config field, no schema entry.
- **Every count written down names the command that produced it**, or is dropped.

## Design notes

- The brief ships in the `wf-rituals` plugin. M-0308 fixed the operative test —
  "What belongs in `aiwf-extensions` is a step with no meaning outside aiwf" —
  and states the disposition for the rest: "An aiwf-specific step in
  `wf-rituals` is conditional, not absent." The brief's aiwf-shaped steps (the
  entity index, the `## Spec measurement` destination) are written as conditions
  on a project that carries them, in the house form M-0308 names —
  `wf-tdd-cycle`'s "If the project uses aiwf and the milestone is
  `tdd: required`".
- ADR-0007 carries no placement rule for the `wf-*` plugin; its decision covers
  the kernel-embedded and `aiwf-extensions` boundary. It is not cited for this.
- `skill-author-guide.md` rule 5 — "Before your skill returns success, run
  `aiwf check`" — does not bind the brief. Its trigger is a skill returning
  success, and its purpose is catching findings from a tree change; the brief's
  reviewer changes nothing, runs nothing, and returns rows rather than success.
  `aiwf check` belongs where the dispatching session writes the ledger. This
  disposes the epic ledger row recorded as unresolved between the two documents.
- ADR-0019 is the precedent for shipping a ritual skill that advises rather than
  gates; the `wf-*` genus predates it.
- The heading helpers under `internal/policies/` are fence-aware
  (`wf_ritual_honesty_test.go:43` — "Lines inside a fenced code block are 0
  whatever their leading hashes"), so a brief carrying a fenced example is safe
  for AC-1's assertion.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-preflight-brief/`
- `internal/policies/` — one test file

## Out of scope

- **The `reviewer` role card**, and every other clearance site — G-0585, which
  the epic excludes wholly.
- **Both seam instructions.** Nothing here says when the brief is dispatched.
- **The sweep's reading method, the lab, and the criterion challenge.**
- **The tree-wide path sweep** — G-0548.
- **A kernel rule for the path ban.** Whether one is earned is a question the
  trial answers.

## Dependencies

- None. No prior milestone under this epic.

## References

- `docs/initiatives/milestone-preflight-as-independent-review.md` — the specification
- ADR-0019 — the advisory-ritual precedent
- ADR-0044 (`proposed`) — describing a corpus by property
- D-0050 — tests over ritual prose assert shape, not wording
- D-0066 — where a completed pass is recorded
- G-0263 — the author-written brief as the exposure
- G-0548 — the tree-wide path population
- G-0580, G-0317 — what the skill-edit backstop reaches, and what it asserts
- G-0587 — the shipped-path collision
- G-0592, G-0593, G-0594 — the spec genre, attribution by extraction, and the
  record of both being breached in this epic
- M-0308 — the `wf-rituals` placement test

## Spec measurement

The criterion challenge, then three blind sweeps against a draft of this spec at
a pinned corpus, then a lab on the rows the sweeps returned unknown. Readings are
the sweeps'; the measured column is the lab's; dispositions are this session's.
Rows the three sweeps returned in common are recorded once.

| claim | reading | measured | evidence | disposition |
|---|---|---|---|---|
| A criterion may derive a shipped artifact's shape from the specification | contradicted |  | `CLAUDE.md:47` relocates the document to an archive on promotion; M-0308 refused the analogous coupling — "pinning it against this repo's configured ref would fix a value the document should not be carrying" | repaired; the derivation is dropped |
| The `reviewer` role card belongs in this milestone | contradicted |  | G-0585 — the change "wants its own branch and its own review rather than riding along with other work" | repaired in the epic; the card is out of scope |
| The card's replacement verdict is the ledger's disposition set | contradicted |  | the initiative — "It returns rows; the session that dispatched it triages them" | moot; the card is out of scope |
| Banning `docs/` and `work/` bans this repo's own layout | contradicted |  | `skill-author-guide.md` rule 1 treats both as consumer-side; G-0548 names the violated class — "design docs under `docs/`, package directories under `internal/`, source files under `cmd/`" | repaired; an allowlist replaces the ban list |
| The heading helpers miscount inside fenced blocks | contradicted | false | `wf_ritual_honesty_test.go:43` — "Lines inside a fenced code block are 0 whatever their leading hashes" | measured; the note states current behavior |
| A path scan over one file is the only mechanical form the collision admits | contradicted |  | G-0548 describes a near-mirror of `skill-body-id` over the same corpus | repaired; the claim is dropped |
| ADR-0044 states the tiering is a precondition | contradicted |  | ADR-0044 — "it is not a precondition for using any surface that needs the answer"; the initiative states the opposite | recorded; two live records disagree, and this spec cites ADR-0044 only for the by-property form |
| `aiwfx-record-decision` bans a shipped skill from naming a repository path | contradicted |  | that record bans embedding a markdown link; `CLAUDE.md` bans the bare path *with* a markdown-link carve-out | repaired; neither is cited as the ban's source |
| ADR-0007 carries a placement rule for the `wf-*` plugin | contradicted |  | its decision covers the kernel-embedded and `aiwf-extensions` boundary | repaired; the citation is dropped and M-0308 supplies the test |
| The epic's risk row and a ledger row cite a blocking question | contradicted |  | the Open questions table holds two rows, both `Blocking? = no` | repaired in the epic |
| Permanent tests ship without a retirement trigger | contradicted |  | the epic's constraint, and H3 — a mandate "lands with a named owner and what retires it, or it is a permanent tax" | repaired; AC-1's mandate retires with the sweep |
| A bare `docs/` path in a shipped skill trips no check | unknown | true | `aiwf check` from a binary built from this worktree, over a tree where `agents/planner.md` carries bare `docs/adr/` and `work/decisions/`: expected 0 errors if unenforced, observed 0 errors | measured; carried in the epic's ledger |
| `findAllVerbs` survives the withdrawal of the ritual it was written for | unknown | true | `grep -rn "func findAllVerbs" internal/policies/` → `skill_coverage.go:500` | measured |
| The skill-edit backstop reaches a role-agent card | unknown | false | `skillRitualsDir = "internal/skills/embedded-rituals"`, every reference scoped to `SKILL.md`; G-0580 states the same | measured; moot once the card left scope |
| Every part of the brief addresses the reviewer | complicated |  | two parts instruct whoever dispatches it, not whoever reads it | repaired; part 8 is the dispatcher-facing section |
| `skill-author-guide.md` rule 5 binds the brief | complicated |  | rule 5's trigger is a skill returning success, and its purpose is catching findings from a tree change | recorded in Design notes; disposes the epic row left unresolved between the two documents |
| Acceptance-criterion bodies carry argued rationale, and attributions are asserted rather than extracted | complicated |  | G-0592 and G-0593 ship both rules; G-0594 records the previous spec in this epic breaching both | repaired; third occurrence, tracked as G-0594 |
| The recorded measurement meets the bar for a settled claim | complicated |  | the initiative requires the command, its expected result, its observed output and its environment together | repaired; the cell above carries all four |
| Proposed decisions are cited as though they bind | complicated |  | D-0053 and D-0056 carry `status: proposed` | repaired; both dropped |
| One `## Spec measurement` section holds two passes' ledgers | complicated |  | D-0066 places a completed pass, singular | moot; one pass, one ledger |
| An empty section satisfying AC-1 is the shape G-0584 files as the defect | complicated |  | G-0584 — "Gut a lens to a heading with an empty body and both the heading count and the substring checks still pass" | recorded; AC-1 states the floor and the epic's constraint licenses it |
| `wf-review-code` ships the same verdict the card would lose | complicated |  | `wf-review-code:113` | moot; the card is out of scope |
