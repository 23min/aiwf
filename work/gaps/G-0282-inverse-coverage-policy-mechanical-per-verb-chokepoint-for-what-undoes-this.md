---
id: G-0282
title: 'Inverse-coverage policy: mechanical per-verb chokepoint for what-undoes-this'
status: open
discovered_in: M-0183
---
## Problem

aiwf's "what verb undoes this?" rule (CLAUDE.md §"Designing a new verb") requires every new verb to have a named inverse — self-inverse (A), terminal transition (B), deliberate documented one-way (C), or new-entity-for-the-inverse (D) — with "we'll figure it out later" explicitly unacceptable. But the rule is:

1. **Advisory only.** Enforced by code review; there is no `internal/policies/` chokepoint. This is exactly the "guarantee depends on reviewer vigilance" pattern the kernel philosophy distrusts — contrast `internal/policies/skill_coverage.go`, which mechanically requires a skill-or-allowlist entry per verb.
2. **New-verb-design-time only.** It never audits the existing verb surface, so a verb that shipped without a clean inverse stays unflagged.
3. **Missing the LLM-UX sharpening.** It asks "does an inverse *exist*?" not "is the inverse *discoverable and trap-free*?" For an AI assistant operating aiwf, an inverse that exists only as a hand-edit of frontmatter / `aiwf.yaml` is a trap: that untrailered commit trips `provenance-untrailered-entity-commit` (`internal/check/provenance.go` ~488-490), so the agent's default undo (git revert / hand-edit) is a universal trap for entity state, and the verb effectively has no usable inverse.

## Evidence (audit, discovered during M-0183 design)

A full sweep of every verb and sub-verb against the rule, with the LLM-UX lens, found two real gaps the advisory gate had missed:

- **`add --area` had no post-create change/clear path** — moving or clearing an entity's area forced a hand-edit → audit trip. This is the gap M-0183 (`aiwf set-area … [--clear]`) closes; the same audit confirmed `--clear` is load-bearing (without it `set-area` re-creates an asymmetric half-trap: tag/retag but no untag).
- **`authorize --to` has no discoverable scope revoke** — `--pause` only parks a scope; reaching `ended` needs a terminal-promote of the scope-entity or `--force`. Already tracked as G-0022 item 1 (explicit revoke verb), YAGNI-deferred.

Both are the *same shape*: a mutating verb whose conceptual inverse is neither a discoverable verb/flag nor a documented deliberate one-way. The rule named the shape; nothing mechanical caught it.

## Proposed fix

A lightweight **per-verb inverse-classification registry** in `internal/policies/`, modeled on `skill_coverage.go`:

- Walk the cobra command tree the same way `skill_coverage.go` already does; require every mutating verb/sub-verb to carry a registry entry `{class: A|B|C|D, note: "<one line>"}`. A verb without an entry fails CI — converting the advisory "what undoes this?" gate into a blocking one (the answer must be *written down* at the chokepoint).
- Read-only verbs are exempt (no state to undo); the registry records that classification too, so coverage is provably complete.
- **Sharpest mechanical wall (catches the untag shape):** a class-A (self-inverse) entry must *name its reversing flag/input*, and the policy asserts that flag exists on the command. A "tag-without-untag" verb (`set-area` without `--clear`) is then either mis-declared (a reviewer reading the required note catches it) or declares a reversing flag that doesn't exist (caught mechanically).

Deliberately NOT in scope: a policy that *reasons about whether the inverse actually works* (semantic FSM / side-effect judgment — infeasible). The registry forces the human-authored answer at the chokepoint, exactly as the skill-coverage allowlist forces a one-line rationale.

## Adjacent cleanups this could fold in (soft caveats from the audit)

- A class-C (deliberate one-way) entry should require an irreversibility note that also appears in the verb's `--help` / Long. `archive` already does this (cites ADR-0004); `rewidth` is one-way but does NOT say so in `--help` — the policy would flag it.
- Optional: the registry note for `add` / `import` could point at "undo = `cancel` (terminal), never delete" so the discoverable-inverse story is reachable from the chokepoint, not only CLAUDE.md commitment #2.

## References

- CLAUDE.md §"Designing a new verb" — the advisory rule this mechanizes.
- `internal/policies/skill_coverage.go` — the per-verb-coverage model to mirror (cobra-tree walk + per-verb entry-or-allowlist).
- `internal/check/provenance.go` (~488-490) — the untrailered-entity audit that makes a hand-edit inverse a trap.
- G-0022 (item 1) — the `authorize` scope-revoke gap this policy would flag (YAGNI-deferred).
- M-0183 — the `set-area --clear` work whose design surfaced this audit.
