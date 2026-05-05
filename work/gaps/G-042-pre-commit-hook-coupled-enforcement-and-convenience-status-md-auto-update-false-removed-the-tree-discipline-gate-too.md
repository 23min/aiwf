---
id: G-042
title: 'Pre-commit hook coupled enforcement and convenience — `status_md.auto_update: false` removed the tree-discipline gate too'
status: addressed
---

Resolved in commit `(this commit)` (feat(aiwf): G42 — decouple pre-commit hook responsibilities). G41 wired the tree-discipline gate into the pre-commit hook, but the hook installer was still gated by `aiwf.yaml: status_md.auto_update` — a flag whose original purpose was to opt out of *STATUS.md regeneration*, not enforcement. Flipping the flag removed the entire hook, which now meant losing the gate too. Pre-push still caught stray files, but the in-loop early-warning that motivated G41 disappeared.

The fix decouples the two responsibilities at the script level:

- The pre-commit hook now installs unconditionally when aiwf is adopted in the repo (the `SkipHooks` opt-out at init time remains the single escape hatch for "I want no aiwf hooks at all").
- `preCommitHookScript(execPath, regenStatus)` takes a bool for the regen step. When false, the script body contains only the tree-discipline gate; when true, it includes the gate followed by the existing tolerant STATUS.md regen.
- The `ensurePreCommitHook` action set is now {`Created`, `Updated`, `Skipped` (alien hook)}; `Removed` no longer occurs through this path. `aiwf doctor`'s pre-commit reporting is updated accordingly: missing-hook is always drift, present-with-mismatching-regen is drift, and the new "ok, gate-only" line marks the desired-and-actual-agree state under `auto_update: false`.
- `extractPreCommitExecPath` now handles the `if ! 'path' …` negation form introduced for the gate; without this, the doctor would have reported a malformed hook for the gate-only mode.

Tests cover both modes end-to-end:

- `TestEnsurePreCommitHook_RegenOff_FreshInstall` and `_RefreshDropsRegen` pin the new install/refresh contracts.
- `TestEnsurePreCommitHook_RegenOff_AlienHookPreserved` proves the always-install change does not weaken alien-hook preservation.
- `TestRefreshArtifacts_FlipFlagDropsRegenKeepsGate` and `TestRun_UpdateDropsRegenKeepsGateOnOptOut` exercise the canonical opt-out flow at the package and verb levels.
- `TestPreCommitHookScript_RegenStatus_Decoupling` pins the script-template invariant: gate always present, regen only when `regenStatus=true`.
- The doctor self-check repo's update round-trip is rewritten ("keeps gate, drops regen" + "reinstates regen") so a regression that re-couples the responsibilities surfaces in the self-check, not in the field.

Severity: **High**. The coupling silently negated G41's enforcement guarantee for any consumer who had touched the unrelated STATUS.md flag. Caught in review immediately after G41 shipped.

---
