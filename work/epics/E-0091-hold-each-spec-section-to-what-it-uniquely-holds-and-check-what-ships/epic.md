---
id: E-0091
title: Hold each spec section to what it uniquely holds, and check what ships
status: active
---

## Goal

Make a milestone spec carry only what no other record holds, and make a shipped
change reach the changelog by check rather than by recall. Today the spec's
largest section duplicates records the kernel owns while hiding the one link it
uniquely holds, and a release's notes rest on whoever cut it remembering.

## Context

Three forces meet in the same five shipped surfaces, and each has been measured
rather than inferred.

The milestone spec's sections have no owner. The same rule — when a section is
filled, and what it holds — is stated independently in the template, both
milestone rituals, and two agent cards, and two of them already disagree about
`## Validation` (G-0636). The template is a weaker owner than it looks: its
per-section comments are consumed at scaffold time and reach 1 of 325 specs, so
a rule that binds during implementation cannot live there.

`## Work log` is where that costs most. It was designed when frontmatter did not
exist and a checkbox list was the only way to see progress; `acs[]`, the TDD
phase ladder and `aiwf history` have carried that since. What remains is
unbounded: 175 populated Work logs, 122 carrying prose beyond the stated
one-line entry, median 283 words against a stated shape of about fifteen
(G-0530). Its purpose was stated in no binding surface until the patch that
opened this epic, and the section has no downstream consumer — the epic wrap
that writes the changelog reads milestone titles and merge SHAs, never a Work
log.

It cannot simply be deleted, because one fact in it is real and unheld
elsewhere: the link from an acceptance criterion to the commit that implemented
it. `aiwf history` discards commits carrying an entity trailer with neither verb
nor actor — 44 such commits, including ones the `skill-edit-provenance-backstop`
rule mandates be written that way (G-0601). Until that projection sees them, the
Work log is the only index there is.

At the far end, nothing verifies that what shipped is described. `[Unreleased]`
is written at a patch's wrap and an epic's wrap and nowhere else, and the only
mechanical check confirms a heading matches a tag. Cutting v0.34.0 surfaced
three changes that reached the release undocumented, two of them on `docs(`
commits — a prefix that means "nothing user-visible" in most repos and the
opposite here, since guidance and rituals ship as product (G-0529).

## Scope

- Give `aiwf history` sight of a commit carrying an entity trailer alone, and a
  chokepoint that catches a missing entity trailer while it is still cheap.
- Add `## Release note` to the milestone spec: the milestone's user-visible
  delta, which is also the input the epic wrap's changelog entry has never had.
- Retire `## Work log` once its unique fact is derivable elsewhere. Adding the
  note does not wait on that — the two sections hold different facts, so they
  coexist until the retirement is safe.
- Give every milestone-spec section rule one owner, chosen by where the rule
  binds, and make the surfaces that restate it point at that owner instead.
- Settle the `## Validation` timing contradiction as a side effect of naming
  that owner.
- Enforce that an entity body carries its kind's required sections, so the
  section set this epic settles is held rather than remembered.
- Verify that a release's `[Unreleased]` names what shipped, including deltas in
  the embedded guidance and ritual trees.

## Out of scope

- Changing what `aiwf history` renders beyond entity-trailered commits. `aiwf
  show`'s dropped terminal reason (G-0590) and the CLI-layer trailer completion
  (G-0546) are adjacent and separately tracked.
- Retiring `## Dependencies`, `## Surfaces touched` or `## References`. G-0530
  names all four; only `## Work log` has a replacement designed here, and the
  other three turn on questions this epic does not answer.
- Rewriting the Work logs of milestones already terminal. Every non-conforming
  Work log measured sits on a `done` or `cancelled` milestone; the historical
  record stays as written.
- Any per-entry length ceiling. A ceiling that can be raised is raised, which
  this repo's own guidance line budget demonstrates.

## Constraints

- No fabricated `aiwf-verb` value. D-0071 settled that no aiwf verb commits
  source and the closed set carries no value meaning "I edited a shipped
  surface"; minting one reintroduces the defect G-0150 closed. The fix is to the
  projection, not the trailer set.
- Shipped surfaces stay project-agnostic. The rituals and templates materialize
  into consumer repos, so a rule may name a category but not this repo's own
  tooling, ids, or paths.
- No prose-content assertion over a shipped surface. D-0070 rules the class out;
  what this epic ships as prose is held at review, and what it ships as
  behaviour is held by a check.
- Changing the history projection changes existing output. The commits it begins
  to surface are the defect being fixed, not a regression, but tests pinning
  history output move with it.
- The spec's section set is settled once and then enforced. A section added or
  retired without the enforcement rule following it re-opens the drift this epic
  closes.

## Success criteria

- [ ] `aiwf history <id>` lists a commit carrying an entity trailer and nothing
      else, and the commit that implemented a gap appears in that gap's history.
- [ ] A milestone spec no longer carries `## Work log`, and the link from an
      acceptance criterion to its implementation commit is answerable without
      reading the spec.
- [ ] A milestone spec carries `## Release note`, and the epic wrap's changelog
      entry is written from those notes rather than from milestone titles alone.
- [ ] Every section rule named in G-0636's surface inventory is stated once, and
      each restating surface points at the owner instead.
- [ ] `## Validation` has one answer about when it is filled.
- [ ] An entity body missing a section its kind requires is reported.
- [ ] A release whose `[Unreleased]` omits a shipped delta is reported before
      the tag, including a delta that lands only in the embedded guidance or
      ritual trees.
- [ ] Every gap listed in *References* is terminal or has its residual recorded.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| How does `aiwf history` label a row for a commit with no verb to name? | yes, for the history milestone | Decided in that milestone; a recorded decision if the choice has consequences for other trailer consumers. |
| Is `## Validation` filled during implementation or at wrap? | yes, for the spec sweep | Follows from naming its owner. Two shipped surfaces say in-flight, two say at wrap; the rituals that drive the work say at wrap. |
| Does the epic wrap read each milestone's `## Release note`, or do the notes accumulate somewhere the wrap copies from? | yes, for the release-note milestone | Settled there; both shapes satisfy the criterion. |
| Which surfaces count as "consumer-visible" for the changelog check? | no | G-0529 names finding codes, verbs, config keys and exit codes; the v0.34.0 evidence requires the embedded trees too. Enumerated in that milestone. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Retiring `## Work log` loses the AC-to-commit link if the history fix lands incomplete | high | Sequence the history work ahead of the retirement; the retirement milestone depends on it and does not start until the link is answerable without the spec. |
| The changelog check fires on correct trees and gets disabled | med | Land it at warning severity against a measured baseline; escalate only once the baseline is clean. |
| The spec sweep and the section-enforcement rule disagree about the section set | med | One milestone owns both, or the enforcement milestone depends on the sweep. |

## Milestones

<!-- Allocated one at a time as each is started. An entry with no id is the
     dependency shape this epic assumes, not an allocation. -->

- `M-0326` — `## Release note` joins the milestone spec and feeds the epic wrap's
  changelog entry · depends on: —
- `M-0327` — the history projection sees entity-trailered commits, at AC
  granularity, and a chokepoint catches an unparseable trailer while it is cheap
  · depends on: —
- `## Work log` retires, its unique fact derivable without it · depends on: the
  history milestone
- Every milestone-spec section rule gets one owner; `## Validation` resolves;
  required sections are enforced · depends on: the retirement milestone
- A release's `[Unreleased]` is checked against what shipped · depends on: —

## References

- G-0530 — milestone specs mandate four sections that duplicate structured data
- G-0636 — milestone-spec section rules are restated across five surfaces with no owner
- G-0601 — `aiwf history` hides skill edits owned by an entity trailer alone
- G-0603 — no chokepoint catches a missing entity trailer while it is still cheap
- G-0571 — nothing enforces that an entity body carries its kind's required sections
- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never checked
- G-0613 — the wrap changelog category set omits Removed, which practice uses
- G-0657 — commits whose trailer block is split from `Co-Authored-By:` are
  invisible to git's parser, so neither history nor the verb check sees them
- D-0070 — prose-content assertions over shipped surfaces are retired
- D-0071 — no aiwf verb commits source, so no verb value names a shipped-surface edit
