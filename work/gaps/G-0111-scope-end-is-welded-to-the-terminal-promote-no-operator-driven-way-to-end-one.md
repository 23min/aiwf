---
id: G-0111
title: Scope-end is welded to the terminal promote; no operator-driven way to end one
status: open
priority: high
discovered_in: M-0096
---
## What's missing

Ending an authorization scope is not something an operator can do. A scope opened with `aiwf authorize <id> --to ai/<agent>` has exactly one exit — the terminal promote or cancel of that same entity — and that exit fires as a side effect rather than as a deliberate act.

Where:

- `internal/verb/authorize.go` — `AuthorizeMode` is a closed three-value set: `AuthorizeOpen`, `AuthorizePause`, `AuthorizeResume`. There is no end, close, or revoke mode, and the CLI in `internal/cli/authorize/authorize.go` accepts exactly one of `--to`, `--pause`, `--resume`.
- `internal/cli/promote/promote.go` sets `IsTerminalPromote` when the target status has no outgoing FSM edges; `internal/cli/cliutil/provenance.go` then stamps one `aiwf-scope-ends: <auth-sha>` trailer onto the promote commit for each scope on that entity whose replayed state is active. `internal/cli/cancel/cancel.go` sets the same flag unconditionally.

So a single commit carries three distinct concerns: the FSM transition, the completion declaration, and the closure of the delegation.

The weld reaches past the two verbs. `aiwfx-wrap-epic` orders its whole commit bundle around it — §"Why promote is last among verb-driven commits" makes the epic's promote-to-`done` the last verb-driven commit precisely *because* that commit ends the scope, since any verb commit landing after it would carry an `aiwf-authorized-by:` naming an ended scope and fire `provenance-authorization-ended` on push. `internal/check/provenance.go` carries an equality carve-out for a terminal-promote commit that ends the very scope it acts under, and `isWrapBundleCommit` in the same file forgives wrap-epic and wrap-milestone commits landing after a same-entity terminal promote. Whatever an explicit end does to the auto-end, those three surfaces move with it.

## Why it matters

The kernel's provenance model separates who is accountable from who ran the verb and from the scope the work happened under. Closing a scope is an act of the principal — it withdraws a delegation. Today it is inferred from a status flip, so nothing in the log distinguishes *the human ended this delegation* from *the entity happened to reach a terminal status*, and a human who wants to withdraw a delegation without closing the entity has no verb to reach for at all.

The asymmetry with the start side is the visible symptom: opening a scope is a deliberate, reasoned invocation, and there is no matching gesture to close one.
