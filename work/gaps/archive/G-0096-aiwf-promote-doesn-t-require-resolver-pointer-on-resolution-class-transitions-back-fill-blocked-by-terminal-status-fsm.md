---
id: G-0096
title: aiwf promote doesn't require resolver pointer on resolution-class transitions; back-fill blocked by terminal-status FSM
status: addressed
addressed_by_commit:
    - 48fbd45
---
## Problem

Two layered holes in the M-059 resolver-pointer rules:

### 1. Verb-level enforcement gap (proximate cause)

`aiwf promote <gap-id> addressed` and `aiwf promote <adr-id> superseded` both have *optional* resolver-pointer flags (`--by` / `--by-commit` and `--superseded-by` respectively). The verb accepts a missing flag silently — it writes the resolver if present, doesn't reject if absent.

The downstream check rules `gap-resolved-has-resolver` and `adr-supersession-mutual` fire as `SeverityWarning`. Warnings don't block `aiwf check`, including the pre-push hook. So the typical path is:

1. Operator promotes a gap to `addressed` without `--by`.
2. Verb succeeds, single commit lands, status flips.
3. `aiwf check` warns at next push but doesn't block.
4. State persists.

This is the kernel-correctness failure mode CLAUDE.md warns about: a guarantee that depends on the operator (human or AI) remembering to add a flag is not a guarantee.

### 2. Back-fill is blocked by terminal-status FSM

`addressed` is terminal for gaps; `superseded` is terminal for ADRs. The legal-transitions FSM rejects both:

- `addressed → open` (revert: terminal blocks)
- `addressed → addressed` (same-status: not a transition)

So once a gap lands in the no-resolver-pointer state, *no verb path exists to fix it without `--force`*. M-059's design assumed resolver pointers would always ride the status-change commit; the back-fill case wasn't covered.

## Discovered

After the E-0023 wrap (2026-05-10), G-0093 had been promoted to `addressed` without `--by`/`--by-commit` (commit `8367951`); the warning surfaced on every subsequent push. Attempting to back-fill via either same-status promote or revert-then-re-promote both failed with terminal-status-block errors.

## Fix

Two changes to `aiwf promote` in one wf-patch:

1. **Require resolver on resolution-class transitions.** Reject `aiwf promote <gap-id> addressed` with `exitUsage` if neither `--by` nor `--by-commit` is set. Parallel rule for `aiwf promote <adr-id> superseded` requiring `--superseded-by`. Closes the proximate hole — new acquisitions of the bad state become impossible.

2. **Allow same-status back-fill carve-out.** When promoting to a terminal status that already matches the entity's current status, and the entity's resolver field is currently empty, allow the promote to proceed iff a resolver flag is provided. Writes only the resolver, not changing status. Single commit. Lets G-0093 (and any other legacy stragglers) be cleaned without `--force`.

Rule severity (warning → error) is deliberately *not* changed here. Once Fix (1) closes the new-acquisition path, severity is a migration question, not a correctness one.

## After-fix sequence

- G-0093 back-filled with `aiwf promote G-0093 addressed --by E-0023` under the new same-status carve-out.
- G-0096 (this gap) promoted to addressed with the wf-patch commit as `--by-commit`.
