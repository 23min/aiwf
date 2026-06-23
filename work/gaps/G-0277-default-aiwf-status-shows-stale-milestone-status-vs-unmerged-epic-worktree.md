---
id: G-0277
title: Default aiwf status shows stale milestone status vs unmerged epic worktree
status: open
---
## Problem

`aiwf status` (the default, non-`--worktrees` view) renders each in-flight
epic's milestone list from the **current worktree's branch frontmatter**. When
the epic's work lives on an unmerged epic branch checked out in a *sibling*
worktree, the milestone statuses the default view prints are stale — and it
presents them as authoritative, with only a soft footer hint
(`for the full per-worktree view: aiwf status --worktrees`) rather than any
inline signal on the milestone lines themselves.

Observed (2026-06-23): running `aiwf status` from the `main` checkout
(`/workspaces/aiwf`) reported every milestone of the active epic E-0043 as
`draft`. In fact M-0171 was already `done` on the epic branch in the sibling
worktree `/workspaces/aiwf-E-0043-area-tag`; its done-promote commit had not
merged to `main`, so `main`'s copy of the milestone frontmatter still read
`draft`. Only `aiwf status --worktrees` showed the true state. A direction
synthesis built on the default view recommended "start M-0171" — a milestone
that was already complete.

## Why this is a correctness problem, not just ergonomics

The default `aiwf status` is the surface an operator (or an AI assistant)
reaches for to answer "what's next?". When it shows `draft` for a `done`
milestone it doesn't merely omit information — it actively contradicts the
authoritative per-branch state on the epic's own worktree, and nothing on the
milestone line flags the divergence. The footer hint is advisory and easy to
miss; the milestone list reads as ground truth.

This is the **read-side** facet of the shared-worktree family. Its siblings
G-0269 (HEAD-drift guard) and G-0270 (epic-activation-on-non-trunk-branch
finding) cover the *mutation* side — a verb landing on the wrong branch; this
gap covers a *display* surface misleading the reader. Distinct from G-0157
(perf: batch the git fan-out in the worktree view) and G-0188 (statusline, not
`aiwf status`).

## Direction (to converge at the milestone)

Invariant: the default `aiwf status` must never present a milestone status that
contradicts the authoritative state on the epic's own worktree without flagging
the divergence. Candidate mechanisms:

- For an active epic with a sibling worktree checked out on its epic branch,
  read that worktree's milestone frontmatter — the machinery already exists
  (`aiwf status --worktrees` does exactly this) — and either reconcile the
  statuses into the default view or annotate divergence inline, e.g.
  `M-0171 [draft here / done on the epic/E-0043 worktree]`.
- At minimum, replace the soft footer hint with a per-epic inline banner when a
  sibling epic-branch worktree exists, so the operator knows the milestone list
  below is branch-local and possibly stale.
- Decide whether the default view should *merge* sibling-worktree state (richer,
  but blurs "what's on this branch") or *flag* it (cheaper, preserves
  branch-local semantics). The whiteboard ritual and any "what's next?" consumer
  want the merged/true picture; a branch-scoped audit wants the local one.

## Provenance

Discovered during an `aiwfx-whiteboard` synthesis (2026-06-23): the default
`aiwf status` rooted in the `main` checkout reported E-0043's M-0171 as `draft`
when it was `done` on the unmerged epic branch in a sibling worktree; the
operator caught the wrong "start M-0171" recommendation. Read-side sibling of
the shared-worktree cluster (G-0269, G-0270).
