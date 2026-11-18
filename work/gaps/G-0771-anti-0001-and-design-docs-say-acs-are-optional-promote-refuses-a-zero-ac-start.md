---
id: G-0771
title: ANTI-0001 and design docs say ACs are optional; promote refuses a zero-AC start
status: open
---
## What's missing

D-0039 refuses `draft → in_progress` on a milestone with no acceptance criteria unless
`--force --reason` is passed, and `requireNonEmptyACsAtMilestoneStart` in
`internal/verb/promote.go` enforces it; before that, the `milestone-draft-incomplete-acs`
warning in `internal/check/acs.go` reports a draft milestone with none. These sources state that
the kernel does not require a milestone to have an AC:

- the anti-rule ANTI-0001 in `internal/workflows/spec/antirules.go` (line 23);
- the `AntiRule` doc comment in `internal/workflows/spec/spec.go` (line 165), which gives it as
  the first of three examples;
- `docs/design/design-decisions.md` (line 152, "What's not a kernel rule");
- R-FP-0166 in `docs/design/legal-workflows-first-principles.md` (line 425), the source
  ANTI-0001 cites;
- the comments in `internal/cellcoverage/fixture.go` (lines 76–78) and
  `internal/cellcoverage/fixture_test.go` (lines 95–99 and 197–199), which cite the anti-rule
  for the zero-AC `done` fixture.

Measured in this tree at bd818e14d:

```
$ grep -n "required to have" internal/workflows/spec/antirules.go internal/workflows/spec/spec.go docs/design/legal-workflows-first-principles.md
internal/workflows/spec/antirules.go:23:			Statement: "A milestone is NOT required to have >=1 AC. ACs are optional.",
internal/workflows/spec/spec.go:165://   - A milestone is NOT required to have ≥1 AC.
docs/design/legal-workflows-first-principles.md:425:| R-FP-0166 | anti-rule | A milestone is **not** required to have ≥1 AC. ACs are optional. | Codified in design-decisions.md §"What's not a kernel rule." The kernel guards the AC outcome, not its existence. | load-bearing | n/a |
$ grep -o 'No "milestone must have ≥1 AC" — ACs remain optional.' docs/design/design-decisions.md
No "milestone must have ≥1 AC" — ACs remain optional.
```

And on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d, in a scratch repository: E-0001 `active`, M-0001 `draft` with `tdd: none` and no AC,
on the epic branch. E-0001 and M-0001 here are the scratch repository's own; in this tree those
ids name unrelated archived entities.

```
$ aiwf check | grep milestone-draft-incomplete-acs
milestone-draft-incomplete-acs (warning) × 1 — draft milestone M-0001 has zero acceptance criteria; add them at plan time (aiwf add ac) so the contract is visible before the milestone lands on main
$ aiwf promote M-0001 in_progress
aiwf promote: cannot promote M-0001 to "in_progress": milestone has no acceptance criteria; add one with `aiwf add ac M-0001 --title "..."` first, or pass --force to override
exit 2
```

## Why it matters

ANTI-0001 lists the zero-AC case among the patterns the kernel deliberately does not police, so
the guard in `promote.go` reads as a breach of the spec, and a change that removed it would
agree with the spec. Nothing flags the contradiction: the tests that read `spec.AntiRules()`
check its count, id shape, uniqueness, non-empty statement and reasoning, and sources, and none
compares a statement with the kernel. G-0161, which asked for that comparison, is `wontfix`.

`docs/design/` is in the Normative tier, read as current truth: a planner who relies on
design-decisions.md writes a milestone with no AC, is warned at draft, and is refused at start.
