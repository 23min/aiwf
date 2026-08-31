---
id: M-0327
title: See entity-trailered commits in history, at AC granularity, guarded on commit
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: aiwf history lists a commit whose only aiwf trailer names the entity
      status: met
      tdd_phase: done
    - id: AC-2
      title: A commit whose subject names an AC carries that AC's entity trailer
      status: met
      tdd_phase: done
    - id: AC-3
      title: A staged shipped-surface edit without a parseable entity trailer is refused
      status: met
      tdd_phase: done
---

## Goal

Make a commit that names an entity visible in that entity's history, at the
granularity an acceptance criterion needs, and refuse at composition time a
commit whose aiwf trailers git cannot read.

## Closes

- G-0601 — the projection tests the entity trailer git parsed, so a commit
  carrying that trailer alone renders.
- G-0603 — `aiwf check --commit-msg` refuses a staged shipped-surface edit whose
  message carries no parseable entity trailer.

## Context

`aiwf history` greps `git log` for an `aiwf-entity:` trailer, then drops any
commit whose `aiwf-verb:` and `aiwf-actor:` trailers are both empty. That test is
a proxy for "git's trailer parser found a real trailer", and it fails on the case
D-0071 created: a shipped-surface edit proves its provenance with `aiwf-entity`
alone, so it satisfies the backstop and renders nowhere.

Measured 2026-08-31 over 10,801 commits across all refs: 8,430 carry an
`aiwf-entity:` trailer and 54 carry it alone — 26 naming a milestone, 26 a gap,
2 a decision. M-0326's own nine implementation commits are among them, so
`aiwf history M-0326` lists every add, promote and body edit and none of the work.

Every one of those 54 names a bare entity id. Of the 3,695 commits carrying a
composite `M-NNNN/AC-N` trailer, all carry a verb — they are `aiwf add ac` and
`aiwf promote` paperwork, without exception. No implementation commit carries the
composite trailer, because nothing asks it to: `aiwfx-start-milestone` step 6
specifies the subject `feat(<scope>): <summary> (M-NNNN/AC-N)` and no trailers,
and that skill states the consequence itself — an implementation commit normally
carries no kernel trailer, which is why the `## Work log` is the only index from
an AC to its commit. Retiring that section (G-0530) needs the index to exist
somewhere else first.

At the composition end, nothing catches a missing trailer while it is cheap. The
backstop is commit-scoped by design, so a forgotten `aiwf-entity` surfaces at the
next gate run, when the repair is an amend or a rebase.

## Acceptance criteria

### AC-1 — aiwf history lists a commit whose only aiwf trailer names the entity

Both drop sites test the entity trailer git's own parser returned, rather than
verb-or-actor: `ReadHistoryChain` in `internal/entityview/historyevent.go`, which
reads trailers through the `git log` pretty format, and `EventFromCommit` in
`internal/entityview/eventfromcommit.go`, which receives them already parsed. The
grep matches `aiwf-prior-entity:` too, so the test admits either key — a
reallocate's rename event must keep rendering when queried through the old id.

This separates a genuine trailer from a prose match by the mechanism the false
positive is about. That false positive is real and stays excluded: `--grep`
matches a wrapped body line beginning `aiwf-entity: <id>`, git's parser finds no
trailer there, and the commit is still dropped.

A row for such a commit has no verb and no actor to print. `RenderTo` already
renders an absent trailer as `-` in the same table, which is the convention to
reach for; what the two empty columns become is settled in-milestone, and no
verb-shaped label is minted for the JSON field, where the honest value is empty.

Flipping the filter and running `go test ./...` on 2026-08-31 failed exactly
three tests, all in `internal/entityview/eventfromcommit_test.go`. Two hand the
constructor a trailer slice carrying no entity key at all — a shape no grepped
commit has — and are fixture gaps. The third,
`TestEventFromCommit_ProseMentionSkipped`, is named for the false positive but
supplies a slice in which git's parser *did* return the entity trailer, so it
pins the proxy rather than the case; its fixture becomes the trailer-free slice
the real false positive produces.

### AC-2 — A commit whose subject names an AC carries that AC's entity trailer

`aiwf check --commit-msg` refuses a message whose subject names an
`(M-NNNN/AC-N)` scope while its `aiwf-entity` trailer names something else, or
nothing. The subject is the commit's own claim to have implemented that
criterion; the trailer is what makes the claim reachable from the criterion.

The link this creates is the one fact a `## Work log` holds that no other record
does, and the retirement milestone (G-0530) depends on it existing without the
spec. AC-1 made such a commit renderable; nothing yet makes it written. Measured
2026-08-31 over 10,801 commits: 368 name an AC in the subject and 20 carry the
matching composite trailer. A commit-msg rule judges only new commits, so the
remainder is not a baseline to clear.

The rule is universal rather than aiwf-scoped, which is what distinguishes it
from AC-3's other half. The subject convention ships with
`aiwfx-start-milestone`, so the rule binds wherever that ritual is followed and
stays silent where it is not — a consumer writing no AC-scoped subjects never
meets it. It invents no obligation for an AC that produces no commit at all,
which an AC met by an observation legitimately does not.

`aiwfx-start-milestone` step 6 gains the trailer on its per-AC commit
instruction, and the anti-pattern describing an implementation commit as
carrying no kernel trailer stops being true and is rewritten. The `aiwf-history`
skill's stated limitation that history shows only verb-driven events is likewise
false after AC-1 and is rewritten here. All three are shipped prose, held at
review under D-0070; what is asserted mechanically is the refusal.

### AC-3 — A staged shipped-surface edit without a parseable entity trailer is refused

`aiwf check --commit-msg` refuses when the staged change touches a surface the
skill-edit backstop watches and git's parser returns no `aiwf-entity` trailer for
the message. The hook already has the message and can ask git for the staged
paths, and it already extracts the trailer block through git's own heuristic.

Presence is all it can enforce. Resolving the named id against the tree needs a
load the hook does not do, so the CI-tier backstop keeps that half; the two are
different questions and both are worth asking.

Parseable, not merely present, is the operative word, and it closes a second hole
in the same seam. Git reads trailers only from a message's last paragraph, so a
blank line between the aiwf block and a trailing `Co-Authored-By:` line hides the
whole block. Measured 2026-08-31: a message carrying `aiwf-verb: feat` written as
one paragraph is refused with exit 1 naming the value; split by that blank line,
the same message exits 0. A fabricated verb is exactly what `trailer-verb-unknown`
exists to refuse and what G-0150 closed, so the guard covers any message whose
aiwf trailer lines git did not parse, not only a shipped-surface edit.

The two halves scope differently, and the difference is the first thing to settle
in-milestone. The parseability guard needs no scoping: a message whose aiwf
trailer lines git cannot read is broken in any repo, because `aiwf history` there
will not see them either. The shipped-surface half watches a path that exists
only in this repo, and no mechanism for that is available to copy —
`PolicySkillEditProvenanceBackstop` is restricted to the aiwf repo by being a Go
policy test that never ships, which a materialized hook cannot imitate. The
commit-msg hook does ship: `aiwf init` installs it, and it runs wherever an
`aiwf.yaml` sits beside it.

## Constraints

- No fabricated `aiwf-verb` value, and no verb-shaped label standing in for one
  in rendered output. D-0071 settled that no aiwf verb commits source; the fix is
  to the projection and the guard, never to the trailer set.
- No prose-content assertion over a shipped surface (D-0070). AC-2's ritual edit
  and the `aiwf-history` skill's revised limitation are held at review; what is
  asserted is the relationship a test can re-derive.
- The projection change alters existing output. The commits that begin appearing
  are the defect being fixed; tests pinning history output move with it.
- Shipped surfaces stay project-agnostic — a rule may name a category, not this
  repo's ids or paths.

## Design notes

- D-0071 — no aiwf verb commits source, so the entity trailer alone is the
  provenance a shipped-surface edit carries. It is why the projection must read
  that trailer rather than infer from a verb.
- D-0070 — settles that the ritual and skill edits here are held at review rather
  than pinned by a phrase assertion.

## Surfaces touched

- `internal/entityview/historyevent.go`, `internal/entityview/eventfromcommit.go`
- `internal/cli/history/history.go` — the text row for a verb-less event
- `internal/cli/check/commit_msg.go` — the composition-time guard
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
- `internal/skills/embedded/aiwf-history/SKILL.md` — its stated limitation that
  history shows only verb-driven events

## Out of scope

- Rewriting or re-trailering commits already landed. G-0657 records 50 commits
  whose trailer block git cannot parse, spanning 2026-04-28 to 2026-06-29 and not
  growing; AC-3 closes the path that admits the shape, and the landed population
  stays as written, under the same reasoning that leaves terminal Work logs
  untouched.
- Retiring `## Work log`. This milestone makes its unique fact derivable
  elsewhere; the retirement is its own milestone and depends on this one.
- Any other change to what `aiwf history` renders. `aiwf show`'s dropped terminal
  reason and the CLI-layer trailer completion are separately tracked.

## Dependencies

- None. The epic sequences this ahead of the `## Work log` retirement, which
  depends on it.

## References

- G-0601 — `aiwf history` hides skill edits owned by an entity trailer alone
- G-0603 — no chokepoint catches a missing entity trailer while it is cheap
- G-0657 — commits whose trailer block is split from `Co-Authored-By:`
- G-0530 — milestone specs mandate four sections that duplicate structured data
- G-0150 — `aiwf-verb` trailer values unpoliced against the registered-verb set
- G-0220 — ritual `SKILL.md` edits with no mechanical owner
- D-0070 — retire prose-content assertions over shipped surfaces
- D-0071 — enforce provenance, not content, at the skill-edit backstop
- M-0312 — re-pointed the backstop to provenance, creating the entity-alone commit

---

## Release note

<!-- The user-visible delta of this milestone. Written at wrap. -->

## Work log

### AC-1 — The projection admits a commit by its parsed entity trailer

Both drop sites read the entity trailer git returned, and an absent verb or actor
renders `-` · commit 8a321fb · full suite green, diff-scoped coverage gate green,
six mutation probes all killed

Measured against this repo on 2026-08-31, the same query either side of the
change: `aiwf history M-0326` gained the nine implementation commits it had been
omitting, and `aiwf history M-0327/AC-1` lists `8a321fb` — the commit that made
the change — where the v0.34.0 binary lists only the paperwork.

`RenderActor` keeps an arm for a principal recorded without an actor. No commit
in this repo's 10,801 has that shape and no verb writes it, so the branch is
reachable only from a hand-written commit. It stays because rendering `-` over a
principal the commit does record would discard provenance from an audit view,
which is the defect class this milestone exists to close; removing it costs no
crash, so the usual keep-the-guard asymmetry is not what decides it.

### AC-2 — A subject claiming an AC carries that AC's trailer

`aiwf check --commit-msg` refuses the mismatch, and the ritual's per-AC commit
instruction writes the trailer · commit 2fc6a80 · `make check-fast` exit 0,
coverage gate exit 0, five mutation probes killed

The guard runs before the empty-trailer-block early return, so a subject claiming
an AC while carrying no trailers at all is caught rather than passed. A probe
moving it after that return stays green on every other case and red only on this
one, which is what pins the ordering.

aiwf's own verbs commit through `git commit`, so this hook judges them too. Every
verb subject has the shape `aiwf <verb> <id> …` with no parenthesised scope, so
the anchored regex cannot match one. That was checked against the subject
constructors in `internal/verb/`, not inferred from history: a verb whose own
commits the hook refused would be unrunnable.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
