---
id: G-0270
title: Check finding for epic activation commit landing on a non-trunk branch
status: open
discovered_in: E-0043
---
## Problem

`aiwf check` has no rule that flags an epic `proposed → active` activation commit
that landed on a non-trunk branch. Per ADR-0010, epic activation is a sovereign
act that lands on `main` before the epic branch is cut. When a shared-worktree
HEAD-drift race (its sibling gap G-0269, the prevention half) or plain operator error
lands the activation on a feature branch instead, nothing detects it post-hoc —
the misplaced commit validates fine in isolation.

Observed: while starting E-0043, `aiwf promote E-0043 active` landed on a parallel
session's feature branch instead of `main`, and no finding fired. The misplacement
surfaced only by manual `git worktree list` / reflog inspection.

## Direction

A check rule that walks activation commits (those carrying `aiwf-verb: promote`,
`aiwf-entity: E-...`, transition `proposed → active`) and reports a finding when
the commit is not reachable from the trunk ref — or sits on a branch whose shape
is not trunk. Warning severity, mirroring M-0106's `isolation-escape`, which is
the analogous post-hoc detector for authorize-scope branch-binding drift.

Open design points to settle at the milestone: how the trunk ref is resolved
(configured vs. `origin/HEAD`), and whether the rule also covers other
trunk-only sovereign transitions or just epic activation to start.

## Provenance

Discovered while starting E-0043. This is the detection half of the lesson; the
prevention half is its sibling gap G-0269 (the HEAD-drift guard on mutating verbs).
