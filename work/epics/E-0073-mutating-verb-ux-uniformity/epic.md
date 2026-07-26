---
id: E-0073
title: Mutating-verb UX uniformity
status: active
---

## Goal

Converge the mutating-verb surface toward consistent operator UX. The PoCv3
health scorecard's C2 finding flagged two uniformity smells in the mutating-verb
surface: same-state inputs that return a Go error instead of a clean no-op, and
wide-blast-radius rewrites that offer no preview before they commit. This epic
addresses the first, uniformly across every affected verb; the second is
deliberately deferred with recorded rationale (see Out of scope). Addresses
G-0230, narrowed to its NoOp-convergence half.

## Scope

One milestone: same-state inputs converge to `verb.Result.NoOp` across the six
mutating verbs that still error today — `promote`, `cancel`, `move`,
`acknowledge-illegal`, `rename`, `retitle` — matching the convention already
shipped in `archive`, `rewidth`, `contract bind`, and `contract recipe install`.
The convergence is pinned by a new AST policy invariant so the discipline cannot
rot back to one-of as new verbs land.

## Out of scope

Dry-run / `--apply` on wide-blast-radius rewrite verbs, deferred here so the
decision is not silently re-derived:

- `reallocate` dry-run — deferred as YAGNI. It rewrites every cross-reference to
  a renumbered id, but fires only on id collisions, is rare, and has caused no
  actual incident. Refile if one occurs.
- `rename` dry-run — rejected. Making `rename` dry-run-by-default (as `archive`
  and `rewidth` are) is a regression for a hot, interactive, single-entity verb:
  it forces `--apply` on every existing invocation for little safety gain. The
  batch sweeps earned dry-run because they are rare and high-blast; `rename` is
  neither.
