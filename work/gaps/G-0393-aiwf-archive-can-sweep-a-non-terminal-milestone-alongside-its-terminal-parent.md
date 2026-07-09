---
id: G-0393
title: aiwf archive can sweep a non-terminal milestone alongside its terminal parent
status: open
discovered_in: M-0243
---
## What's missing

`aiwf archive`'s per-kind sweep moves a milestone into `archive/` whenever
its parent epic reaches a terminal status — the milestone "rides with"
its parent (per the storage layout `aiwf archive --help` documents) and
carries no independent terminality check of its own. Promoting a parent
epic straight to a terminal status via `aiwf promote <epic> done` carries
no non-terminal-children guard (unlike `aiwf cancel`, whose
`epic-cancel-non-terminal-children` refusal exists specifically to
prevent this class of state), so a milestone can still be `in_progress`
(or any non-terminal status) when its parent epic is promoted to `done`
and then swept by `aiwf archive --apply`.

The result is a milestone living under `archive/` with a non-terminal
status — a tree state `aiwf check`/`aiwf show` then flags at error
severity (`archived-entity-not-terminal`, "archive is the structural
projection of FSM-terminality," per ADR-0004 §Reversal). The anomaly is
caught, but only after the fact: `aiwf archive` (or `aiwf promote <epic>
done`) has no guard preventing the sweep from creating this invalid state
in the first place.

Confirmed directly: an epic promoted straight to `done` while its
milestone remains `in_progress`, followed by `aiwf archive --apply`,
produces exactly this state — a non-terminal milestone under `archive/`
that `aiwf check` immediately flags as an error.

Separately confirmed as NOT broken: the milestone's own authorize scope
(if genuinely still active at sweep time) survives the sweep correctly —
`aiwf show`'s `scopes` array and `aiwf authorize --pause` both continue to
resolve it after the archive move. That specific fear (scope becomes
unresolvable once its holder crosses the archive boundary) does not hold.

## Why it matters

`cancel` already treats "parent terminal while a child is non-terminal"
as a refusal-worthy precondition. `promote <epic> done` reaching the
same parent-terminal state through a different, equally normal path
without the same guard is an inconsistency: the same invalid shape is
prevented on one path and produced-then-caught on the other. A guard
mirroring `epic-cancel-non-terminal-children` — refusing the promote (or
the archive sweep) until every child milestone is itself terminal — would
close this gap at its source rather than relying on `aiwf check` to
surface it after the tree is already in the invalid state.
