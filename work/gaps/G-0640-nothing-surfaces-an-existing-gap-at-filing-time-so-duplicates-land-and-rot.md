---
id: G-0640
title: Nothing surfaces an existing gap at filing time, so duplicates land and rot
status: open
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
  Both gaps are still `open`.
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
