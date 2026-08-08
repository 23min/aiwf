---
id: G-0572
title: Two legal verbs on two branches merge into an AC state only --force repairs
status: open
priority: high
discovered_in: M-0300
---
## What's missing

Nothing pins what a *merge* does to an acceptance criterion's TDD evidence. Two
verbs, each legal where it ran, compose across a branch boundary into a state
the kernel reports at error severity and that no non-sovereign verb can repair.

Reproduced end to end in a disposable repo, with no hand-edited frontmatter and
no `--force`:

```
aiwf add epic --title "..."
aiwf add milestone --epic <epic> --tdd advisory --title "..."
aiwf add ac <milestone> --title "..."

git checkout -b branch-a
aiwf milestone tdd <milestone> --policy required   # legal here; exit 0

git checkout main && git checkout -b branch-b
aiwf promote <milestone>/AC-1 met                  # legal under tdd: advisory; exit 0

git checkout main
git merge --no-ff branch-a                         # clean
git merge --no-ff branch-b                         # clean, auto-merged
```

`aiwf check` on the merged tree then reports, at error severity:

```
acs-tdd-audit: <milestone>/AC-1 status: met under tdd: required
but tdd_phase is (absent) (expected done)
```

Each branch was internally consistent. Git had no textual conflict to raise —
the two verbs wrote different frontmatter fields of the same file. The error
exists only in the merged tree, and it blocks the push.

Every exit from that state was measured:

| Attempt | Outcome |
|---|---|
| `--phase done` — what the finding's own hint recommends | refused by the phase FSM: `""` cannot transition to `done` |
| `--phase green` | refused by the phase FSM |
| `--phase red` | refused by the projection guard — the post-state still carries the error, since `met` with `red` is not `done`. Exit 1, nothing written |
| `--phase red --force --reason ...` | same projection refusal; sovereignty does not help |
| `promote <milestone>/AC-1 open` — demote, then walk the ladder honestly | refused by the AC FSM: `met` cannot transition to `open` |
| `--phase done --force --reason ...` | **succeeds** |
| `milestone tdd <milestone> --policy advisory` | the error becomes a warning |
| `cancel <milestone>/AC-1` | the finding goes with the criterion |

## Why it matters

The one path that keeps the criterion is the sovereign one, and what it writes
is `tdd_phase: done` on a ladder that was never walked — fabricating exactly the
evidence `acs-tdd-audit` exists to demand. The rule's purpose is to make "the
test came before the code" mechanically checkable; the only repair the kernel
offers for a state it produced itself is to assert that claim without it. The
two remaining exits keep the record honest by discarding the guarantee instead:
weaken the milestone's policy, or cancel the criterion.

The hint compounds it. The finding names `--phase done` as the fix, and the
phase FSM refuses that transition from an absent phase — so an operator
following the error message's own instruction is told the move is illegal, with
nothing pointing at what else to try.

This is the composition class G-0121 names, reached without any random walk: two
verbs, two branches, one merge. It is also the concrete instance E-0080 was
chartered to catch and could not have — that epic's harness walks a single
branch, so no sequence it generates reaches a state which only exists after a
merge.

G-0569 is the neighbouring defect on the same FSM from the opposite direction:
there a phase survives a reset it should not, here it cannot start when it must.
