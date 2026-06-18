---
id: G-0206
title: BranchOracle false-positive on branch renames after authorize
status: addressed
discovered_in: M-0158
addressed_by:
    - M-0161
---
M-0103's `aiwf authorize --branch <name>` records the named branch
in the `aiwf-branch:` trailer on the authorize commit. The trailer
is a STRING — the branch's name at scope-open time. If the
operator later renames the branch via `git branch -m oldname
newname`, the trailer still says `oldname`.

## Failure mode

1. Operator runs `aiwf authorize E-0030 --to ai/claude --branch epic/E-0030-foo`.
   Trailer records `aiwf-branch: epic/E-0030-foo`.
2. Operator renames `git branch -m epic/E-0030-foo epic/E-0030-bar`.
3. AI commits land on `epic/E-0030-bar`.
4. `aiwf check` runs: oracle indexes `epic/E-0030-bar` (the current
   branch); finds the AI commit there. Compares to bound
   branch `epic/E-0030-foo` from the trailer. **Mismatch**.
5. **`isolation-escape` fires falsely** for every AI commit on
   the renamed branch.

This is the opposite failure mode from the other gaps: false
positive, not silent escape. But the friction is the same — the
operator sees a flood of warnings and either disables the rule
or files away the false positives.

## What's needed

Options:

1. **Treat `aiwf-branch:` trailer as a SHA-equivalent reference**:
   record the branch's tip SHA at authorize time in addition to
   the name; oracle/rule compares via SHA reachability when the
   name no longer resolves.

2. **Add a rename verb** (`aiwf scope rebind --to <new-branch>`)
   that records a new trailer linking old to new; the rule walks
   the rebind chain.

3. **Document the limitation** and add a kernel-level check
   (`aiwf-branch-trailer-unresolvable`) that surfaces when a
   trailer's named branch is missing; the operator chooses to
   either rename back, rebind, or accept the warnings.

Option 3 is the simplest fix; options 1 + 2 are more correct but
heavier.

## Why parked

The M-0158 honest-scope audit surfaced this. Branch renames are
common (operators reshape ritual names mid-epic); the current
behavior is silently wrong. Address as part of the real-world
hardening milestone.
