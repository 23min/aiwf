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
      title: A hidden trailer block or an unowned ritual edit is refused at commit time
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

Measured at `d7a6598`, walking this branch's history rather than all refs:
10,814 commits, of which 8,445 carry an `aiwf-entity:` trailer and 59 carry it
alone — 31 naming a milestone, 26 a gap, 2 a decision. M-0326's own nine
implementation commits were among them, so `aiwf history M-0326` listed every
add, promote and body edit and none of the work.

The AC-level link is thinner still. 3,700 commits carry a composite
`M-NNNN/AC-N` trailer, and all but 23 were written by an aiwf verb — `promote`
and `add` account for most, with `retitle`, `cancel`, `rename` and `edit-body`
behind them. Of the 23, 20 carry a verb value the closed set does not recognize,
from before it was policed, and 3 carry no verb at all. So the shape exists and was never systematic, which is
what nothing asking for it produces: `aiwfx-start-milestone` step 6 specified the
subject `feat(<scope>): <summary> (M-NNNN/AC-N)` and no trailers. Retiring
`## Work log` (G-0530) needs that index to exist without the spec.

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

A row for such a commit has no verb and no actor to print. Both render `-`, the marker
`RenderTo` already uses for an absent target status in the same table. No
verb-shaped label is minted: the column would name something no aiwf verb did,
and the JSON field keeps the honest empty string.

Flipping the filter and running `go test ./...` on 2026-08-31 failed exactly
three tests, all in `internal/entityview/eventfromcommit_test.go`. Two hand the
constructor a trailer slice carrying no entity key at all — a shape no grepped
commit has — and are fixture gaps. The third was named for the false
positive but supplied a slice in which git's parser *did* return the entity
trailer, so it pinned the proxy rather than the case. It is retired; the
admit-rule table covers both directions, including the trailer-free slice a real
false positive produces.

### AC-2 — A commit whose subject names an AC carries that AC's entity trailer

`aiwf check --commit-msg` refuses a message whose subject names an
`(M-NNNN/AC-N)` scope while its `aiwf-entity` trailer names something else, or
nothing. The subject is the commit's own claim to have implemented that
criterion; the trailer is what makes the claim reachable from the criterion.

The link this creates is the one fact a `## Work log` holds that no other record
does, and the retirement milestone (G-0530) depends on it existing without the
spec. AC-1 made such a commit renderable; nothing yet makes it written. Measured
at `d7a6598` with the predicate the rule itself applies — the scope anchored at
the end of the subject — 369 commits name an AC that way and 22 carry the
matching composite trailer. A commit-msg rule judges only new commits, so the
remainder is not a baseline to clear.

The rule is universal rather than aiwf-scoped, which is what distinguishes it
from AC-3's other half. The subject convention ships with
`aiwfx-start-milestone`, so the rule binds wherever that ritual is followed and
stays silent where it is not — a consumer writing no AC-scoped subjects never
meets it. It invents no obligation for an AC that produces no commit at all,
which an AC met by an observation legitimately does not.

`aiwfx-start-milestone` step 6 carries the trailer on its per-AC commit
instruction, its anti-patterns name what the `## Work log` holds that an AC's
history cannot, and the `aiwf-history` skill states what selection by trailer
does and does not reach. The milestone-spec template and the changelog's
unreleased entry describe the section the same way. All are shipped prose, held at review under
D-0070; what is asserted mechanically is the refusal.

### AC-3 — A hidden trailer block or an unowned ritual edit is refused at commit time

`aiwf check --commit-msg` refuses when the staged change touches the ritual
authoring tree and git's parser returns no `aiwf-entity` trailer for the message.
The predicate is the directory — every file under it materializes into a
consumer's `.claude/`, entity templates and agent cards as much as skills — which
is wider than the CI-tier backstop's `SKILL.md`-only scan. The hook already has the message and can ask git for the staged
paths, and it already extracts the trailer block through git's own heuristic.

Presence is all it can enforce. Resolving the named id against the tree needs a
load the hook does not do, so the CI-tier backstop keeps that half; the two are
different questions and both are worth asking.

Parseable, not merely present, is the operative word, and it closes a second hole
in the same seam. Git reads trailers only from a message's last paragraph, so a
blank line between the aiwf block and a trailing `Co-Authored-By:` line hides the
whole block. Before this milestone a message carrying `aiwf-verb: feat` written
as one paragraph was refused with exit 1 naming the value, while the same message
split by that blank line exited 0 — a fabricated verb is exactly what
`trailer-verb-unknown` exists to refuse and what G-0150 closed. So the guard
covers any message whose aiwf block git did not read, not only a shipped-surface
edit.

Hidden is narrower than trailer-shaped, and what separates them is how many aiwf
lines the paragraph carries. Two make a block. One does not in general, because
a single trailer-shaped line is textually identical to prose opening with a key
and a colon, and git's parser returns nothing for either — refusing every such
line rejects ordinary English with no recourse, which is how a check comes to be
disabled. The exception is a lone entity trailer whose value names an entity:
that is the shape a shipped-surface edit carries, since no aiwf verb commits
source, so it is the block most likely to be written from here on, and an id is
not a sentence.

The guard runs before the other two, because a hidden block makes the trailers
invisible to them and reporting one of their findings first states something
untrue about the message in front of the operator.

A subject git composed is not a claim its author made. `git commit --fixup` and
`--squash` copy the target commit's subject verbatim, so an AC scope in one
belongs to that commit; refusing here would break `rebase --autosquash`, whose
pairing depends on the subject matching exactly.

The two halves scope differently. The parseability guard needs no scoping: a message whose aiwf
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
- `internal/skills/embedded/aiwf-history/SKILL.md` — what selection by trailer reaches
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md`
- `internal/cli/check/check.go`, `internal/cli/show/show.go`,
  `internal/initrepo/initrepo.go` and its hook golden
- `internal/cli/render/resolver.go` and the history rows in
  `internal/htmlrender/embedded/*.tmpl`
- `CHANGELOG.md` — the unreleased entry's account of the same link

## Out of scope

- Rewriting or re-trailering commits already landed. G-0657 records the commits
  whose trailer block git cannot read — 50 under the predicate this milestone
  ships, spanning 2026-04-28 to 2026-06-29 and not growing since. The gap records
  the same 50 under a predicate that is looser than the shipped one and actually
  yields 53, catching a prose mention as well, so its figure is wrong rather
  than merely broader. AC-3 closes the
  path that admits the shape; the landed population stays as written, under the
  same reasoning that leaves terminal Work logs untouched.
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

`aiwf history <id>` now lists a commit whose only aiwf trailer names the entity —
the implementation commits and shipped-surface edits that no verb wrote, which it
previously discarded. A row for one carries `-` where a verb and actor would be,
matching the marker an absent target status already uses.

The `commit-msg` hook gained three refusals, each catching at composition what
was previously found at a later gate or not at all:

- a subject naming an `(M-NNNN/AC-N)` scope whose `aiwf-entity` trailer names
  something else, or nothing — the link that makes an implementation commit
  findable from the criterion it implements;
- an aiwf trailer block git will not read, because a blank line leaves it out of
  the message's final paragraph. Such a block is invisible to `aiwf history` and
  carries an unrecognized `aiwf-verb` value straight past the check that exists
  to refuse it;
- a staged edit to the shipped ritual tree whose message names no entity.

The per-AC commit instruction in `aiwfx-start-milestone` now writes the
`aiwf-entity` trailer, so an acceptance criterion's history shows the commit that
implemented it.

## Work log

### AC-1 — The projection admits a commit by its parsed entity trailer

Both drop sites read the entity trailer git returned; an absent verb or actor renders `-` on every surface that shows one · commits 8a321fb, d7a6598, 02ad406 · check-fast and coverage gate green

### AC-2 — A subject claiming an AC carries that AC's trailer

`aiwf check --commit-msg` refuses the mismatch; the per-AC commit instruction writes the trailer · commit 2fc6a80 · check-fast and coverage gate green

### AC-3 — A hidden trailer block or an unowned ritual edit is refused at commit time

Both guards reached through `aiwf check --commit-msg`; a block is hidden when git's own parse does not return its trailers · commits fc5501d, b1c020b, a0c99a0, 1b4d7a1, c02e431 · check-fast and coverage gate green

## Decisions made during implementation

- D-0083 — ship the commit-msg shipped-surface guard unscoped. The subprocess it
  costs runs in every consumer repo and can only ever fire in this one; the
  reasoning, the two alternatives, and what would reopen it are recorded there.

## Validation

Run on the milestone branch after the corrective round, in the devcontainer
(Linux, no signing wrapper needed). The figures below are the state that ships;
the per-AC runs recorded in `## Work log` predate the corrections and are
superseded here:

- `make check-fast` — exit 0. `go vet` across the default, `stress` and
  `testpins` tag sets; `golangci-lint run` reporting 0 issues; `go test
  -parallel 8 ./...` with no failures.
- `AIWF_COVERAGE_BASE=<ac-base> make coverage-gate` — exit 0 for each AC, after
  the AC-3 run named three uncovered changed lines that were closed with tests
  rather than annotations.
- `aiwf check` — 0 errors, two warnings.
  `provenance-untrailered-scope-undefined` predates this milestone.
  `epic-active-no-drafted-milestones` began firing at this milestone's own
  `draft -> in_progress` promote, which left the epic with no drafted milestone;
  it clears when the next one is planned.
- `go build ./...` — green.

Two observations either side of the change, re-runnable against this repo:

- `aiwf history M-0326` gained the nine implementation commits it had been
  omitting, and `aiwf history M-0327/AC-1` lists the commit that made the
  change, where the v0.34.0 binary lists only the paperwork.
- A message carrying a real wrap commit's trailer block split from its
  `Co-Authored-By:` line, and the same shape carrying a fabricated `aiwf-verb`
  value, both exited 0 before and exit 1 after. The second is the sharper
  result: the split block carried a value the closed set refuses straight past
  the check that exists to refuse it.

Mutation probes were run against the changed logic throughout, reverted by
capture-and-restore and verified byte-identical afterwards. Seven survivors were
found across four review rounds and all seven are now killed or removed: an
entity trailer whose value is only whitespace; the verb column in `aiwf show`;
the two call sites feeding the rendered site; the prior-entity arm of the
block test; the all-lines trailer gate; and the four page templates, which the
move of the display rules had made load-bearing in the opposite direction. An
eighth, a CRLF normalisation, proved equivalent once paragraphs split on any
all-whitespace line, and was deleted rather than pinned. What the next review
finds is for that review to record.

## Deferrals

- G-0657 — the commits already carrying a trailer block git cannot parse. AC-3
  closes the path that admits the shape; the landed population is historical
  record and is not rewritten, on the same reasoning the epic applies to the
  Work logs of terminal milestones.

## Reviewer notes

Independent review ran repeatedly over the full change-set and every round
returned request-changes. The behaviour was found sound throughout; almost every
finding was a claim about it, and corrective rounds kept introducing claim
defects of their own. Each finding's record is the check that pins it and the
commit body that says why it changed — not repeated here.

What held under attack, so a later round need not re-spend it: the commit-msg
guard was driven over every commit message in this history, sixteen synthetic
edge shapes and an end-to-end hook run, with no false positive and no skip that
loses a trailer git could not read; a sweep of every changed production file
killed all but one mutant, and that one is G-0658.

One fact nothing pins: no aiwf verb reaches this hook, because `CommitVerbChange`
writes through `commit-tree` and `update-ref`, which fire no hooks. The subject
shapes are independently safe, but a move back to the `git commit` porcelain
would leave that the only thing holding.

The guard's cost in consumer repos, where it can never fire, is settled in
D-0083.
