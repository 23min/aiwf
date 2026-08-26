---
id: G-0640
title: Nothing surfaces an existing gap at filing time, so duplicates land and rot
status: open
priority: medium
---
## What's missing

`aiwf add` allocates an id and writes the entity without ever surfacing what the
tree already holds on the same subject. `Add` (`internal/verb/add.go`) reads the
tree four times — the ritual-branch guard, `--depends-on` referents, `AllocateID`,
and path construction — and none of them looks at what existing entities are
about. There is no flag and no output for it.

The manual route is no better: `aiwf list` carries no title-text filter and no
search flag, so a filer's alternative is reading every open gap title.

The tree carries the result. Measured 2026-08-26:

- **G-0562 and G-0578 are the same defect.** Both name
  `internal/policies/worktree_rituals_check_hook_test.go` writing an executable
  through a bare `os.WriteFile` rather than `testsupport.WriteExecutable`. Filed
  2026-08-06 and 2026-08-11, neither referencing the other. The defect was fixed
  on 2026-08-19 by `793b1ad97`, which routed that fixture through the helper.
  Neither closed against it for a week; both closed on 2026-08-26. A re-filing is
  not a copy — the later gap was the thinner one, and the earlier carried a second
  question it did not, which moved to G-0641 rather than closing alongside it.
  Closing the pair as duplicates would have dropped both the analysis and the
  question.
- **G-0580 and G-0618 name one hole across two predicates.** Both describe the
  skill-edit backstop reaching only `SKILL.md` under the embedded-rituals subtree,
  leaving agent cards and verb skills outside it. G-0580 names the structural-test
  backstop, which D-0071 retired — `skill_edit_provenance_backstop_test.go` now
  asserts that mandate stays absent from `CLAUDE.md`. G-0618 names the live
  `PolicySkillEditProvenanceBackstop`. Filed eleven days apart.

Filing dates above come from:

```
git log --diff-filter=A --format=%ad --date=short -- work/gaps/<id>-*.md
```

The tree has not been swept for further pairs, and co-citation does not find them:
51 source files are cited by more than one open gap, and the top of that ranking is
distinct defects sharing a file rather than duplicates.

Nothing in the tree computes string distance or text similarity, so anything built
here starts from no existing helper.

## What the fix looks like

`aiwf add <kind> --dry-run`: resolve everything the verb would do, print the id it
would allocate and the path it would write, and — for gaps — the nearest open gaps.
Commit nothing. The seam this needs is already there: `verb.Add` returns a `Result`
carrying a plan and `verb.Apply` commits it as a separate call, which is how
`aiwf archive` already offers the same flag.

`aiwf add` sits behind a human gate. The dry-run output is what the proposal put to
that gate is composed from, so the check runs at the moment the decision is made
rather than depending on anyone remembering a separate lookup beforehand.

The nearest list prints on a real add as well. The computation is identical either
way, and it backstops a filing that skipped the dry-run: the remedy there is
cancelling a fresh entity, which is worse than not creating it and better than the
two open duplicates it replaces.

**Rank, do not judge.** Measured 2026-08-26 over the open gaps, scoring each
candidate at 0.6 x title-token overlap + 0.4 x body-token overlap: filing G-0578,
its duplicate G-0562 ranks 1 of 178 at 0.462 against a runner-up of 0.148; filing
G-0618, its neighbour G-0580 ranks 2 of 178 at 0.114, inside a band of unrelated
gaps scoring 0.10 to 0.16. A threshold separates the first pair and not the second,
so a rule deciding "this is a duplicate" catches one and misses the other, while a
ranked list of five surfaces both. The runner-up is not noise to be suppressed
either: filing G-0578 it is G-0497, which sweeps the same class of defect.

Two shapes that do not work. A `[y/N]` confirm before the write fires only when
stdin is a terminal, and the path that files gaps here is not one, so the gate would
skip exactly the filings it exists to catch. A standalone search verb binds only
when someone remembers to run it first, which is the discipline both pairs above
record failing.

## Why it matters

A duplicate costs more than the wasted filing, because each copy then ages alone.
G-0562 and G-0578 both survived the commit that fixed them: closing one would not
have surfaced the other, and nothing walks the tree asking which open gaps a commit
just satisfied. Two records now describe a defect that no longer exists, and anyone
planning from either plans work already done.

It compounds with the backlog it sits in — 178 open gaps on 2026-08-26, 25 of them
high priority. Every mechanism proposed against gap quality so far inspects records
after they exist. Filing is the only point in the lifecycle before the record is
there to inspect.
