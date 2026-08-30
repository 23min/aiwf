---
id: G-0111
title: Scope-end is welded to the terminal promote; no operator-driven way to end one
status: open
priority: high
discovered_in: M-0096
---
## What's missing

The automatic scope-end is welded to the status flip. A promote or cancel that takes a scope-entity terminal also closes every delegation on it, in the same commit, as a side effect of the transition.

Where:

- `internal/cli/promote/promote.go` sets `IsTerminalPromote` when the target status has no outgoing FSM edges; `internal/cli/cliutil/provenance.go` then stamps one `aiwf-scope-ends: <auth-sha>` trailer onto that commit per non-ended scope on the entity. `internal/cli/cancel/cancel.go` sets the same flag unconditionally.

So a single commit carries three distinct concerns: the FSM transition, the completion declaration, and the closure of the delegation.

An operator now has a deliberate gesture of their own — `aiwf authorize <id> --end` retires one named scope without touching the entity's status — but that is an addition beside the weld, not a replacement for it. The automatic end fires exactly where it always did.

The weld reaches past the two verbs. `aiwfx-wrap-epic` orders its whole commit bundle around it — §"Why promote is last among verb-driven commits" makes the epic's promote-to-`done` the last verb-driven commit precisely *because* that commit ends the scope, since any verb commit landing after it would carry an `aiwf-authorized-by:` naming an ended scope and fire `provenance-authorization-ended` on push. `internal/check/provenance.go` carries an equality carve-out for a terminal-promote commit that ends the very scope it acts under, and `isWrapBundleCommit` in the same file forgives wrap-epic and wrap-milestone commits landing after a same-entity terminal promote. Whatever an explicit end does to the auto-end, those three surfaces move with it.

## Why it matters

A commit that means three things is a commit no rule can read for one of them. The kernel's own surfaces work around it rather than through it: `aiwfx-wrap-epic` orders its entire commit bundle so the epic's promote-to-`done` lands last among verb-driven commits, because that commit ends the scope and anything verb-driven after it would carry an `aiwf-authorized-by:` naming an ended scope; and `internal/check/provenance.go` carries both an equality carve-out for a terminal-promote that ends the scope it acts under, and `isWrapBundleCommit` forgiving wrap commits that land after a same-entity terminal promote.

Each of those exists because the closure is inferred from the transition rather than stated. They are the cost the weld imposes, and they stay until the two are separable.
