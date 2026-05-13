---
id: G-0120
title: provenance audit rules cascade unfix-ably on post-promote wrap commits
status: addressed
discovered_in: E-0029
addressed_by_commit:
    - a87d7138ba5fe6cedff20131d4ec85bf2d337e4c
---
## What's missing

The provenance audit's three rules — `provenance-authorization-ended`, `provenance-trailer-incoherent`, and `provenance-no-active-scope` — fire in cascade on legitimate wrap-time commits that carry stale authorize trailers (per G-0119). Surgical trailer removal doesn't unblock: dropping `aiwf-authorized-by:` reveals the paired `aiwf-on-behalf-of:` finding; dropping that pair reveals the actor-needs-active-scope finding on `aiwf-actor: ai/claude`. The union of the three rules forbids any consistent trailer set for "LLM operator continued a wrap after the scope-ending promote."

This means the historical bad commits produced by G-0119's ritual order cannot be remediated by trailer surgery alone — they would have to be (a) rewritten with a different actor claim (revising who acted), (b) rewritten to drop all aiwf-* trailers (warning-level untrailered-entity-commit instead), or (c) bypassed via `--no-verify` push. Each violates a different kernel principle.

## Why it matters

G-0119 is the forward fix — re-order the ritual so future wraps stay clean. But forward fixes do not reach **historical commits already in `main`**. Without a reader-side accommodation, every push that includes commits from before the ritual fix lands continues to fail until either the bad commits are rewritten out of history (force-push main, breaks `aiwf history`) or every push uses `--no-verify`. Neither is sustainable; both violate kernel design pillars.

The cleanest reader-side fix: refine the three rules to recognize "the wrap operation is atomic across commit boundaries" — when a commit's `aiwf-verb:` is wrap-related (e.g. `wrap-epic`) AND its `aiwf-entity:` matches the entity that ended the scope, treat the post-promote window as still-authorized for the same actor. This parallels G-0118's `prior_ids`-walk approach: the kernel state already records what's needed; the audit just has to consult it.

## Likely fix

1. Identify the rule set: `provenance-authorization-ended`, `provenance-no-active-scope`, and any related provenance rule that fires on the cascade. Likely in `internal/check/provenance.go` near the existing scope-resolution code.
2. Add a per-rule exception: when the commit's `aiwf-verb` is in a small allow-list of wrap-related verbs (`wrap-epic`, `wrap-milestone`, possibly `reallocate`) AND its `aiwf-entity` matches the entity that terminated the scope, treat the commit as within-scope despite the post-promote timestamp.
3. Test: fixture-tree case where a wrap-bundle commit lands after its scope-terminating promote, and the audit no longer fires.

This is analogous to G-0118's reader-side fix (walk `prior_ids` to resolve renamed entities). Same shape: extend the audit to consult kernel state that already encodes the legitimate case.

Related: G-0118 (analogous reader-side fix for `out-of-scope` after rename), G-0119 (forward fix in the ritual; this gap covers historical commits and future commits if the ritual fix is delayed).
