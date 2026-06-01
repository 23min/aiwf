---
id: G-0196
title: acknowledge-illegal does not cover forced-untrailered subcode
status: open
discovered_in: E-0030
---
## What's missing

`aiwf acknowledge-illegal <sha>` only exempts the `fsm-history-consistent/illegal-transition` subcode. The `ackedSHAs` map built by `walkAcknowledgedSHAs` (which reads `aiwf-force-for:` trailers from HEAD's history) is threaded into `illegalTransitionFindings` but not into `forcedUntraileredFindings`. The `forced-untrailered` subcode — which fires on sovereign-act-shape transitions (e.g. epic `proposed → active`) by non-human actors or commits lacking an `aiwf-actor:` trailer entirely — has no retroactive fix path.

A consumer repo hit this on commit `423ffdfb`: the human manually edited the epic frontmatter to flip `proposed → active` because `aiwf promote` hit git-lock contention during a start-milestone ritual. The commit is a legitimate sovereign act by a named human, but it carries no `aiwf-force:` trailer and no `aiwf-actor: human/...` trailer (the edit bypassed the verb pipeline entirely). The operator cannot rewrite history (shared trunk, force-push forbidden), cannot `acknowledge-illegal` (scoped to `illegal-transition`), and cannot `--audit-only` (scoped to `manual-edit`). Pre-push is blocked with no clean resolution path.

## Why it matters

The three `fsm-history-consistent` subcodes — `illegal-transition`, `manual-edit`, `forced-untrailered` — each have a retroactive-acceptance mechanism except `forced-untrailered`. This asymmetry means a legitimate historical sovereign act that happened to miss the verb pipeline (lock contention, binary crash, operator unfamiliarity) permanently blocks pre-push with no documented fix short of history rewrite or a `check.suppress` escape hatch that defeats the rule's purpose.

The fix is narrow: thread `ackedSHAs` into `forcedUntraileredFindings` the same way it already feeds `illegalTransitionFindings`. The verb surface (`aiwf acknowledge-illegal <sha> --reason "..."`) stays unchanged — the `aiwf-force-for:` trailer already carries enough information, and the human-actor + reason gates already enforce the sovereign-act discipline. The skill docs and hint text need updating to reflect the widened scope.
