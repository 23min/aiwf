---
id: G-0346
title: Wrap rituals merge onto mainline without reconciling a diverged trunk first
status: open
---
## What's missing

The wrap rituals merge a ritual branch **onto** a checkout of mainline, so when
mainline has advanced past the branch's fork point any conflict is resolved on
mainline itself, mid-merge, with no validated gate. `aiwfx-wrap-epic` step 5 is
the clearest instance:

```
git checkout main
git pull --ff-only origin main
git merge --no-ff --no-commit epic/<epic-branch>
```

If the epic branch diverged from `main`, that `git merge` resolves conflicts on
the `main` checkout, and precondition 5 ("full local gate green on the epic
branch") validated a tree that predates mainline's newer commits — so the
integrated result is never gated as a whole. `wf-patch` step 9 (merge to
mainline) and `aiwfx-wrap-milestone` (merge into the epic branch) share the
shape whenever the parent branch has moved.

The reliable practice — confirmed in the G-0344 wrap, where `main` had advanced
three commits under a patch branch — is the inverse: **integrate mainline into
the ritual branch first, resolve conflicts and re-run the full gate there, then
merge back** (a clean fast-forward for a linear-trunk patch, or the trailered
`--no-ff` merge for an epic). Mainline only ever receives an already-validated
result.

## Why it matters

Resolving a merge conflict directly on `main` is the exact state the rituals'
gate discipline exists to prevent: an unvalidated, half-integrated tree sitting
on the integration target. It also lets the "gate green before merge"
precondition pass vacuously — green on a branch tree that omits mainline's newer
commits. The failure is quiet: the merge completes and the push may even pass
CI, but the local pre-merge validation never covered the combined tree, and a
mid-merge conflict resolution on `main` has no branch to fall back to.

## Direction

Ship the reconcile-first practice in the ritual bodies (operating guidance, so
it materializes into consumer repos and is dogfooded here — one source, no
fork):

- **`aiwfx-wrap-epic`**: add a reconcile step before step 5 — when mainline is
  not an ancestor of the epic branch (`git merge-base --is-ancestor
  origin/main <branch>` is false), integrate mainline into the epic branch,
  resolve conflicts, and re-run the full local gate there, so the step-5 merge
  is clean. Tighten precondition 5 to require the gate green on the epic branch
  **after integrating current mainline**.
- **`wf-patch`** step 9 and **`aiwfx-wrap-milestone`**: the same note, scoped to
  their integration targets (mainline; the epic branch).
- Keep the repo-specific merge *mechanism* (linear-trunk fast-forward for
  patches, trailered `--no-ff` merge for epics) where it already lives — this
  adds the mechanism-neutral reconcile-first practice, not a mechanism opinion.

Each edited `SKILL.md` under the embedded-rituals tree needs a referencing
structural test under `internal/policies/` (the
`skill-edit-structural-test-backstop` policy) as the mechanical evidence.

Stronger follow-up (separate, if the prose proves insufficient): a mechanical
wrap preflight that asserts `git merge-base --is-ancestor origin/main <branch>`
and blocks the merge step when mainline has diverged — the framework's
"correctness must not depend on LLM behavior" principle argues the durable form
is a check, not advice.
