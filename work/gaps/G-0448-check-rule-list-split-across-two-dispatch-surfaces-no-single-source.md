---
id: G-0448
title: check rule list split across two dispatch surfaces, no single source
status: open
---
## What's missing

"The set of all `aiwf check` rules" has no single source of truth. Rules are dispatched from two parallel surfaces, split by an accident of function signature rather than a declared boundary:

- **Pure in-memory rules** (those needing only a `*tree.Tree`) are a hardcoded append-list in `internal/check/check.go` (~lines 109-166).
- **Git/history/area rules** (those needing `context.Context`, config, or git-derived arguments) are individually invoked and appended in `internal/cli/check/check.go` (~lines 127-305).

Whether a rule lands in surface one or surface two is decided purely by "does it need git/ctx/config" — not by any intentional layering. There is no `Rule` interface or registry that both surfaces feed; each rule is a bespoke free function returning `[]Finding`.

## Why it matters

- **No single enumeration.** Nothing in the codebase can answer "what are all the checks?" from one place. Adding a rule means editing whichever of the two orchestrators matches its signature, and the author has to know which.
- **Wired-into-only-one risk.** A rule can be defined and appended to one surface while silently absent from the other's conceptual list, with no mechanical cross-check. (No such orphan exists today — every rule is reachable — but nothing prevents the next one from being half-wired.)
- It is the package's clearest A2/C1 finding at the *architecture* level (as opposed to the copy-paste convergence tracked separately): one conceptual list, two physical homes.

Counterweight: real single-source wins already exist in this package and should be preserved — finding codes are typed `const`s (G-0129), the hint table is one Code+Subcode→hint map with a chokepoint test, and `DisputedTrunkIDs` is a deliberately-shared predicate. This gap is narrowly about the rule-enumeration split.

## Resolution shape

Introduce one ordered registry (or a minimal `Rule` shape) that both pure and git-fed rules register into, so "all checks" has a single source. The git-needing rules keep their richer dependency signature — the registry entry can carry the projection of config/ctx each needs — but enumeration and ordering live in one place. A follow-on chokepoint test could then assert every defined rule is registered (closing the wired-into-only-one risk mechanically).

Scope check against design-decisions.md: this is *not* the deliberately-excluded "module/capability registry" (that's about external capability discovery) — it's an internal refactor collapsing two enumeration sites into one. KISS applies: the smallest change is a single ordered slice both surfaces append through, not a plugin system.

## Where to fix

`internal/check/check.go` (`Run` / the in-memory append-list) and `internal/cli/check/check.go` (the git/history/area invocation block) — converge onto one registration surface.

## Related

- `wf-codebase-health` — the ritual that surfaced this (A2/C1).
- The convergence-tax gap — the copy-paste sibling finding in the same packages (this one is architectural, that one is line-level).
