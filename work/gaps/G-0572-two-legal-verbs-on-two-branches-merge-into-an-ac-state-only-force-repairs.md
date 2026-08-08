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
the kernel reports at error severity and that no unforced verb can repair.

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

Every exit was measured:

| Attempt | Outcome |
|---|---|
| `--phase done` — what the finding's own hint recommends | refused by the phase FSM: `""` cannot transition to `done` |
| `--phase green` | refused by the phase FSM |
| `--phase red` | refused by the projection guard; exit 1, nothing written |
| `--phase red --force --reason ...` | same projection refusal |
| `promote <milestone>/AC-1 open` | refused by the AC FSM: `met` cannot transition to `open` |
| `promote <milestone>/AC-1 open --force --reason ...` | **succeeds** — and the phase ladder then walks unforced to a clean tree |
| `--phase done --force --reason ...` | succeeds, but stamps `done` on a ladder never walked |
| `milestone tdd <milestone> --policy advisory` | the error becomes a warning |
| `cancel <milestone>/AC-1` | the finding goes with the criterion |

## Why it matters

An honest repair exists, and it is sovereign-only. Force-demoting the criterion
to `open` clears the finding — `acs-tdd-audit` fires only on ACs whose status is
exactly `met` — after which `--phase red`, `green`, `done` and `promote met` all
succeed unforced and the tree ends at zero errors with evidence that was
actually earned. What the kernel does not offer is any unforced route out of a
state two ordinary verbs and a clean merge produced.

The hint compounds it. The finding names `--phase done` as the fix, and the
phase FSM refuses that transition from an absent phase, so an operator following
the error message's own instruction is told the move is illegal with nothing
pointing at what else to try. `internal/verb/milestone_tdd.go` refuses a
single-branch flip that would strand a met, phaseless AC, on the stated grounds
that "the phase ladder cannot be re-run on already-met work" — the demote-then-
recycle path above disproves that rationale, and the merge creates the state the
single-branch verb declines to.

The `--phase red` refusal has a cause worth separating from this gap: the
projection guard compares findings by an identity that includes the message
text, and `acs-tdd-audit`'s message interpolates the very phase value the verb
is mutating, so a pre-existing finding reads as newly introduced. That is a
general defect with its own instances, tracked separately.

This is the composition class G-0121 names, reached without any random walk: two
verbs, two branches, one merge. G-0569 is the neighbouring defect on the same
FSM from the opposite direction — there a phase survives a reset it should not,
here it cannot start when it must.
