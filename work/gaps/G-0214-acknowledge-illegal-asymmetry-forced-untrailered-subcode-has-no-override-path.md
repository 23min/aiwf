---
id: G-0214
title: 'acknowledge-illegal asymmetry: forced-untrailered subcode has no override path'
status: open
discovered_in: M-0159
---
## What's missing

`aiwf acknowledge-illegal <sha>` covers the `fsm-history-consistent/illegal-transition` subcode but NOT the `fsm-history-consistent/forced-untrailered` subcode. The forward-ack mechanism (empty commit with `aiwf-force-for: <sha>` trailer) is recognized only by the illegal-transition arm of `fsm_history_consistent.go`; the forced-untrailered arm has no retroactive override path.

Real consumer impact (history-mining subagent §1.1 + §5.2; cross-references G-0196): a consumer hit this exact gap when a manual frontmatter edit (`proposed → active`) after `aiwf promote` hit git-lock contention. The resulting commit was forced-untrailered (no `aiwf-verb:` trailer because the verb invocation didn't complete its commit). The fsm-history-consistent rule fired on the forced-untrailered subcode. `aiwf acknowledge-illegal` was the operator's expected escape — but the subcode is not in the verb's recognized set.

The fix is roughly a one-line thread-through: extend the rule's exemption walk to recognize `aiwf-force-for:` trailers for the forced-untrailered subcode the same way it does for illegal-transition. The architectural primitive (the walk, the trailer, the verb) already exists; only the per-subcode dispatch needs to land.

## Why it matters

Asymmetric override surfaces are the canonical "looks complete but isn't" pattern. The verb name (`acknowledge-illegal`) implies coverage of every illegal-shaped finding; the implementation covers only one subcode. The operator's mental model breaks at the moment they need the escape, with no clear workaround.

This finding has real consumer evidence: it has already caught at least one operator. Fold into M-0159 alongside the Path B helper-lift work — the same `walkAcknowledgedSHAs` helper that gets extracted for shared use across `fsm-history-consistent`, `isolation-escape`, and `trailer-verb-unknown` will be the right home for the forced-untrailered exemption. The lift consolidates the override surface.

Discovered by the history-mining subagent §1.1 + §5.2 during M-0159 planning (2026-06-02). Cross-references G-0196 (the original symmetry-gap filing). M-0159 closes both.
