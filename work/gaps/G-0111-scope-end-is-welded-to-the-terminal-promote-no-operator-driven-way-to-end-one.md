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

## Why it matters

The kernel's provenance model separates who is accountable from who ran the verb and from the scope the work happened under. Closing a scope is an act of the principal — it withdraws a delegation. Today it is inferred from a status flip, so nothing in the log distinguishes *the human ended this delegation* from *the entity happened to reach a terminal status*, and a human who wants to withdraw a delegation without closing the entity has no verb to reach for at all.

The asymmetry with the start side is the visible symptom: opening a scope is a deliberate, reasoned invocation, and there is no matching gesture to close one.

## What a fix has to account for

The current bundling is load-bearing, not incidental — three surfaces are built on top of it:

- `aiwfx-wrap-epic` orders its whole commit bundle around it. Its §"Why promote is last among verb-driven commits" makes the epic's promote-to-`done` the last verb-driven commit precisely *because* that commit ends the scope: any verb commit landing after it would carry an `aiwf-authorized-by:` naming an ended scope and fire `provenance-authorization-ended` on push.
- `internal/check/provenance.go` carries an equality carve-out for the case where a terminal-promote commit ends the very scope it acts under — allowed, because the scope was active when the commit landed.
- `isWrapBundleCommit`, in the same file, forgives wrap-epic and wrap-milestone commits that land after a same-entity terminal promote has ended the scope.

An operator-driven end therefore lands alongside a decision about what the terminal promote does afterwards: keep the auto-end as a fallback for the un-ended case, or drop it and re-order the ritual and both carve-outs with it. Dropping it is observable to any caller relying on today's behavior, so it ships as a documented behavior change in `CHANGELOG.md`.

## Resolution path

The ADR comes first, because the ritual and both check carve-outs all follow from the single choice of what the terminal promote does once an explicit end exists. Then the new mode on `aiwf authorize`, then the `aiwfx-wrap-epic` re-ordering.

Ritual content is authored directly in the embedded snapshot at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`, pinned as `aiwfxWrapEpicFixturePath` in `internal/policies/aiwfx_wrap_epic_test.go` — one commit in this repo, no cross-repo copy.
