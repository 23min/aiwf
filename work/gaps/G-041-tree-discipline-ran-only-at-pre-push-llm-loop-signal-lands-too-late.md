---
id: G-041
title: Tree-discipline ran only at pre-push — LLM-loop signal lands too late
status: addressed
---

Resolved in commit `(this commit)` (feat(aiwf): G41 — pre-commit gate + `aiwf check --shape-only`). G40 shipped the tree-discipline rule wired into the full `aiwf check` pipeline at pre-push only. That guarantees the bad state never *pushes*, but it does not give the LLM an in-loop signal — by the time pre-push fires, the stray commit has already landed locally, possibly been amended onto, or been bypassed via `git push --no-verify`. The user pushed back on two points:

1. **Agent-agnosticism.** A marker-managed CLAUDE.md fragment (the original early-warning proposal) ties aiwf to Claude Code; Cursor uses `.cursor/rules`, AGENTS.md is emerging, etc. Git hooks fire for any client that uses git — which is all of them. The hook is the agent-agnostic surface.
2. **Pre-commit beats pre-push for this rule.** Stray-file detection is fast and exact; there is no legitimate "WIP" state where a stray exists. Moving the check earlier costs nothing in correctness and gains the in-loop feedback signal. The kernel's existing "marker-managed framework artifacts" principle already covers git hooks, so no new surface is created.

The fix has three pieces, all in this commit:

1. **`aiwf check --shape-only` flag.** Runs only the tree-discipline rule (no trunk read, no provenance walk, no contract validation), reads `aiwf.yaml: tree.{allow_paths,strict}` the same way the full check does. Cheap enough to fire on every commit. Exit codes match the standard contract: 0 ok, 1 findings (only when tree.strict promotes the warning to error), 3 internal.
2. **Pre-commit hook gains the gate.** The aiwf-managed pre-commit hook now invokes `aiwf check --shape-only` *before* the existing STATUS.md regen step. The shape check is non-tolerant — non-zero exit blocks the commit (only fires when strict). The status step remains tolerant per the existing design.
3. **Skill + design doc updates.** `aiwf-check` SKILL documents the `--shape-only` flag and the pre-commit/pre-push split as a two-row table; `tree-discipline.md` records the chokepoint design rationale and explicitly rejects the marker-CLAUDE.md alternative with the agent-agnosticism reasoning.

Followup decoupling closed in G42 — see below. Severity: **High**.

---
