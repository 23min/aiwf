---
id: G-0045
title: aiwf-managed git hooks don't compose with consumer-written hooks
status: addressed
addressed_by_commit:
  - 49e7764
---

Resolved in commit `49e7764` (feat(aiwf): G45 — hook chaining via `.local` siblings + auto-migration). The marker-managed `pre-push` and `pre-commit` hooks now invoke a `<hook-name>.local` sibling (if present and executable) before running aiwf's own work. `aiwf init` / `aiwf update` auto-migrate a pre-existing non-marker hook to `<hook-name>.local`, preserving its content byte-for-byte and its executable bit, then install aiwf's chain-aware hook. New `ActionMigrated` step result. `HookConflict` now signals only the rare `.local`-already-exists collision (refuse to clobber a deliberate `.local`). `aiwf doctor` reports the chain shape per hook: absent, present + executable (`chains to ...`), or present + non-executable (error). Tests cover migration, the load-bearing collision case, the chain runtime semantics (`.local` exits 0 / non-zero / non-executable), and doctor's three states.

`aiwf init` / `aiwf update` install marker-managed hooks at `.git/hooks/pre-push` and `.git/hooks/pre-commit`. When a consumer already has a non-marker hook in place, init refuses to overwrite (correct, by design — see [`internal/initrepo/initrepo.go`](../../internal/initrepo/initrepo.go) `ensurePreHook` / `ensurePreCommitHook`). The user is left with three choices: remove their hook, manually compose it with `aiwf check`, or run `aiwf init --skip-hook` and lose the chokepoint. None of these match the kernel's "framework should add to the consumer's flow, not demand the consumer dismantle their own" stance.

This is the load-bearing collision once the kernel itself dogfoods aiwf (G38): the kernel's existing pre-commit hook (`scripts/git-hooks/pre-commit`, run via `core.hooksPath`) collides with aiwf's marker-managed hook. The same collision happens for any consumer using husky / lefthook / pre-commit.com that has hand-written hooks under `.git/hooks/`.

**Resolution:** Hook chaining. The marker-managed hook learns to invoke a `<hook-name>.local` sibling before running aiwf's own work. Specifically:

1. **Naming.** `.git/hooks/pre-commit.local` and `.git/hooks/pre-push.local`. Git itself ignores the `.local` suffix — only aiwf's hook ever invokes it. No risk of git running the user's hook behind aiwf's back.
2. **Chain order: user-first.** The aiwf hook runs `<hook-name>.local` if present and executable, then (only on exit 0) runs aiwf's own work (`aiwf check --shape-only` for pre-commit, `aiwf check` for pre-push, plus the optional STATUS.md regen for pre-commit). User-first matches the convention in chaining tools (pre-commit.com, etc.) and means the user's iteration loop isn't gated on aiwf's check.
3. **Auto-migration on `aiwf init`.** When init detects an existing non-marker hook, it `mv`s it to `<hook-name>.local`, preserves its executable bit, then installs aiwf's chain-aware hook. The user wakes up with a working composition. Init prints a clear ledger line naming what moved where. The `--skip-hook` flag still bypasses the entire dance for users who manage hooks via husky/lefthook.
4. **Collision guard.** If `<hook-name>.local` already exists when init wants to migrate (consumer has both a non-marker hook *and* a prior `.local`), refuse with a clear error rather than overwrite — the user has clearly engaged with chain plumbing on purpose. This is the one case where init still requires manual resolution.
5. **Non-executable `.local`: fail loud.** If the chain script finds `.local` exists but `! -x`, it fails the commit/push with a clear remediation message (`chmod +x`). A non-executable hook script is almost always a configuration mistake; silent skip would let the user think they have hook coverage when they don't.
6. **`aiwf doctor` reports chain shape.** New rows: `pre-commit hook: ok (aiwf-managed; chains to .git/hooks/pre-commit.local)` when the sibling exists and is executable; `(aiwf-managed; no .local sibling)` when absent; `error (... is not executable — chmod +x to enable)` when present but non-executable. The error case increments doctor's problem count.
7. **`aiwf update` is the redeployment vector.** Existing consumers pick up the chain plumbing automatically when they next run `aiwf update`; their `<hook-name>.local` (if any) is left untouched.

Severity: **Medium**. Not blocking the PoC, but blocks G38 (dogfooding the kernel against itself) cleanly, and blocks any consumer with a pre-existing hook from a friction-free `aiwf init`. Filed as the chokepoint that has to land before the dogfood migration.

---
