---
id: D-0107
title: Commit the milestone wrap through edit-body
status: proposed
relates_to:
    - ADR-0058
    - D-0106
---
> **Date:** 2026-11-08 · **Decided by:** Peter Bruinsma

## Question

How the rituals' own commits fit [ADR-0058](../../docs/adr/ADR-0058-amend-the-unpushed-verb-commit-instead-of-adding-one.md). A verb amends only a commit a verb made, so a ritual's plain `git commit --trailer` ends a run of verbs. [D-0106](D-0106-stop-committing-roadmap-md-render-it-on-demand.md) removes the rituals' roadmap commits. Three ritual changes remained to settle:

- whether the wrap rituals' own commits go through verbs;
- whether evidence cites the code commit rather than the planning tip it ran at;
- whether spec edits that follow a code commit wait for the next run of verbs.

## Decision

- **The milestone wrap's prose goes through `aiwf edit-body`.** `aiwfx-wrap-milestone` edits the milestone spec in the working tree, shows the diff, and gates one `aiwf edit-body` on the milestone in place of its plain wrap commit.
- **`## Reviewer notes` names the commit the deciding review read.** The wrap prose then names HEAD whenever HEAD is still that commit, and ADR-0058 commits it anew, so history keeps the line between what the review read and what came after.
- **The epic wrap keeps its plain commit.** It writes `wrap.md` and `CHANGELOG.md`, which no verb writes.
- **Evidence cites the commit the check read,** as it does now.
- **Spec edits stay where the work puts them,** right after the code commit they describe.

## Reasoning

**What a hand-composed wrap commit lets through.** Measured 2026-11-08 with aiwf built from `d50379c2e` (an untracked directory outside the build marked it `+dirty`) and git 2.51.1 on Linux, in a scratch repository with a bare remote. Each case was the milestone wrap committed with `git commit --trailer` through the installed hooks, followed by `aiwf check` against the pushed base.

| Mistake in the hand-typed trailers | What happened |
|---|---|
| `aiwf-entity` names the epic, not the milestone | No finding; the milestone's history lacks the wrap |
| `aiwf-entity` left out | No finding; the milestone's history lacks the wrap |
| `aiwf-actor` names another person | No finding; history credits that person |
| `aiwf-actor` left out | No finding |
| key typed as `Aiwf-Verb` | Error at push, `provenance-untrailered-entity-commit` |
| verb value mistyped | The `commit-msg` hook refused it |
| no trailers | Error at push |

Through the verb, the trailers come from the verb and the actor from `git config`. The comparable slip, `aiwf edit-body` naming the epic while only the milestone's file had changed, was refused with exit 2. HEAD did not move, and the edit stayed in the working tree.

**What the plain wrap commit gave that the verb route drops:**

- *Its `feat(<scope>): …` subject.* Nothing reads it: the release ritual finds its build commit with `git log -- '**/*.go'`, the changelog workflow checks only a tag's section, and no Go code parses the subject. The commit holds only the spec, and `ROADMAP.md` until D-0106. Recent wrap commits used `feat(cli): …`, `chore(milestone): record …` and `chore(milestone): wrap …`, so there was no convention to keep. The milestone's merge keeps `chore(milestone): wrap <id> — <title>` and the `wrap-milestone` trailers.
- *The review boundary.* Naming the reviewed commit keeps it (Decision).

**What it is worth in commits.** The research's replay of 2026-08-26 to 2026-11-05, re-run 2026-11-08 with Python 3.12.3 against this repository, counts the planning commits amend leaves without the roadmap commits:

| Rituals | Planning commits left |
|---|---|
| As today | 608 |
| Milestone wrap through a verb | 589 |
| Both wrap rituals through verbs | 574 |

The 589 assumes the wrap prose always amends; naming the reviewed commit makes some of those wraps commit anew, by an amount not measured. The change is decided for its trailers and provenance, not its count. Putting the epic wrap through a verb as well would save 15 more, but it needs a new kernel verb that owns the changelog's `[Unreleased]` format, which a consumer may not use.

**Evidence citations.** In the window, 38 verb commits name the SHA of a parent that is itself a verb commit, which is what stops an amend:

| Where the citation sits | Count |
|---|---|
| A milestone's `## Validation` | 20 |
| A gap's reproduction | 13 |
| Elsewhere | 5 |

10 of the 20 Validation sections report `aiwf check`, which reads the planning files the later commits changed. For those, citing the last code commit would name a tree the check never read. Citing the commit the check read costs at most 46 commits in the window: 589 against 543 if no operation named HEAD.

**Spec edits after a code commit.** Under these choices, 82 of the window's 101 runs of only `edit-body` after a code commit could have joined the next run on their line; most are a milestone's `## Validation`, its decisions or its AC sections. Such an edit cannot wait in the working tree. With the milestone spec edited and left uncommitted, `aiwf promote` on one of its ACs, `aiwf add ac` and `aiwf retitle` each refused, and HEAD did not move. Waiting therefore means writing the edit later from memory, while the spec lags the code. `aiwfx-wrap-milestone` also requires the evidence committed before it sends the change to review, so part of the 82 could not wait in any case.

## Consequences

- **Every consumer's `aiwfx-wrap-milestone` changes.** Its stage, commit-gate and commit steps become one `aiwf edit-body` gate, and its `## Reviewer notes` guidance gains the reviewed commit. The CHANGELOG entry says so.
- **The milestone's history shows the wrap** as an `edit-body` event and as the merge's `wrap-milestone` event.
- **`aiwfx-wrap-epic` keeps hand-typed trailers on its wrap-artefact commit,** so the silent mistakes above remain possible there. [G-0744](../gaps/G-0744-the-untrailered-entity-audit-trusts-any-commit-carrying-an-aiwf-verb-trailer.md) records the audit that lets a wrong or missing `aiwf-entity` through.
- **What changes:**
  - `aiwfx-wrap-milestone` under `internal/skills/embedded-rituals/`, which is shipped content and so changes on a branch;
  - the Claude baseline's record of that ritual's bytes;
  - the CHANGELOG.
