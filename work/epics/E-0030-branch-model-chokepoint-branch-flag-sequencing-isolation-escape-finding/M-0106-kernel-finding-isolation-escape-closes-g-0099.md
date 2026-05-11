---
id: M-0106
title: Kernel finding isolation-escape (closes G-0099)
status: draft
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
---

## Goal

Add a kernel finding `isolation-escape` that fires at `aiwf check` (pre-push) when an AI-actor's commits violate [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s branch convention — i.e., when commits made under an active AI scope land on a branch that doesn't match the scope's `aiwf-branch:` trailer. Closes [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) fully.

## Context

The session-layer PreToolUse hook (landed today) denies the `isolation: "worktree"` Agent kwarg, preventing one failure mode. M-0102 + M-0103 prevent another: AI dispatch without a named branch is refused. This milestone adds the *third* layer — post-hoc detection of drift that slips through both gates (e.g., a subagent that escapes its assigned branch via `cd ..` or `git -C <other-path>`, or a manual cherry-pick that violates the scope-branch coupling).

Together the three surfaces give defense in depth: pre-dispatch (session-layer hook), at-dispatch (preflight), and at-push (kernel finding). The finding is the unbypassable layer — it fires regardless of which dispatch path the parent used.

The finding polices AI-actor commits only (per ADR-0010's sovereignty principle); human-actor commits, including manual cherry-picks between branches, are not policed.

## Out of scope

- Rituals updates (M-0104 / M-0105).
- Author iteration — the finding fires only on AI-actor commits.
- Non-aiwf commits — only commits carrying an `aiwf-entity:` trailer are inspected.
- Retroactive enforcement — only commits made under active scopes after this milestone lands are policed.

## Dependencies

- **M-0102** — provides the `aiwf-branch:` trailer the finding reads from.
- **M-0103** — the preflight chokepoint that this finding is the post-hoc complement of.

## Open questions for AC drafting

- **Finding scope:** Per-commit or per-scope-lifetime? Per-commit gives clear signal (each violating commit fires once) but can be noisy on a runaway subagent; per-scope-lifetime is one finding per misbehaving scope but less specific. Tentatively per-commit; revisit if noise is a real problem.
- **Severity:** Blocking error or warning? Tentatively *warning* on first land (let real usage tune), with a roadmap to flip to *error* once the policy is settled and the false-positive rate is known.
- **Detection algorithm:** For each `aiwf-entity:`-trailered commit, resolve the active scope on that entity at commit time; if `aiwf-actor:` is `ai/<id>` and the commit's branch doesn't match the scope's `aiwf-branch:`, fire. Confirm at AC-drafting time.
- **What about scope `paused` state?** Should commits made while a scope is paused be policed at all, or skipped? Decide here.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0106` time. -->
