---
id: G-0539
title: A plain commit staging entity files passes pre-commit and hides until push
status: open
priority: medium
---
## What's missing

Two absences in one story: the bad commit is never blocked, and the finding it
produces then hides.

**The commit is never blocked.** `aiwf edit-body` and the other structured verbs
exist so an entity edit rides a verb and carries `aiwf-verb:` / `aiwf-entity:`
trailers. Nothing stops `git add -A && git commit` from staging an entity file
alongside code and committing it. The pre-commit hook runs
`aiwf check --shape-only`, and the rule that would catch it —
`provenance-untrailered-entity-commit` — walks history, so it is not in the
shape-only set and could not be: at pre-commit the commit does not exist yet.

What does exist at pre-commit is the index, and the index carries the whole
condition. A staged file under the entity tree, in a commit no verb is driving,
is the defect in full.

**The finding then hides.** On a branch with no upstream and no `--since`, the
provenance audit does not run. It reports `provenance-untrailered-scope-undefined`
at warning severity and moves on, so `aiwf check` returns 0 errors over a tree
that has two. Only an explicit `aiwf check --since <ref>` surfaces them. A branch
cut by `aiwf worktree add` has no upstream until it is pushed, which is the
normal state for a patch branch's entire life.

## Why it matters

Observed in this repo: a patch branch committed two entity bodies via
`git add -A`. Pre-commit passed. `aiwf check` reported 0 errors.
`aiwf check --since main` reported 2. The defect would have reached the push
before anything said so, and the push is where unwinding costs most — the
recovery paths there are `aiwf acknowledge illegal` or `promote --audit-only`,
both of which record a sovereign exemption rather than repair the history.

The kernel holds that framework correctness must not depend on an assistant
remembering to invoke the right verb. Here the verb exists and the discipline is
in the always-on guidance, and neither is a chokepoint. That is the shape the
principle exists to reject.

## Resolution shape

The pre-commit half prevents rather than detects, and needs one distinction to
be safe: a verb's own commit also stages entity files. The verbs drive git
themselves, so they can mark their own commits — an environment variable set
across the commit call is the cheap version, with the hook skipping when it is
present.

Three flows stage entity files without being one verb's edit and must not fire:
a merge commit carrying another branch's entity changes, `aiwf archive`'s sweep,
and `aiwf reallocate`'s cross-reference rewrite. The first is distinguishable by
`MERGE_HEAD`; the second and third are verbs, covered by the same marker.

Severity is the open question. A hard block is the honest reading of the rule,
but a pre-commit block that misfires is one an operator learns to `--no-verify`
past, and that habit costs more than this rule saves. Starting at warning is
defensible; what is not defensible is silence.

The silent-skip half is smaller and independent. With no upstream configured the
audit should either fall back to a base ref — the trunk ref is the obvious
candidate, since `allocate.trunk` already names one for a different purpose — or
report the skip at error severity. Reporting 0 errors over a tree that has two
is the part that misleads, and it is what let this reach the push.
