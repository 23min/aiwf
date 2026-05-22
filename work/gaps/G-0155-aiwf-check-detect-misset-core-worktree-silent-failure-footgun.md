---
id: G-0155
title: 'aiwf check: detect misset core.worktree (silent failure footgun)'
status: open
---
## What's missing

`core.worktree` in `.git/config` can be set to a path that doesn't match the directory containing `.git` — typically by some prior tool that mishandled `git worktree add` semantics, but in principle any process can write it. When that happens, every `git` invocation from the affected directory silently operates on the *configured* worktree, not the directory's actual checked-out files. There is no warning, no error, no diff in `git status` output that reveals the misdirection; the planning tree, the index, the file lookups all transparently route through the wrong path.

`aiwf check` does not currently inspect this configuration, so the same silent failure can corrupt every aiwf verb: `aiwf status` lists the wrong worktree's entities, `aiwf add` writes into the wrong worktree, `aiwf check` itself loads the wrong tree and validates it. The operator only discovers the mismatch when something obviously breaks — typically much later, after several commits or checks have already been made against the wrong state.

This is exactly the silent-failure pattern aiwf's `aiwf check` chokepoint exists to prevent: a precondition of correctness (the kernel reading the right tree) is left unenforced, and the kernel's "framework correctness must not depend on the LLM's behavior" rule is implicitly violated.

## How it bites

Surfaced during the M-0124 wrap session:

1. `core.worktree = /workspaces/aiwf-M-0124-positive-cell-coverage` got set in `/workspaces/aiwf/.git/config` (source unknown — probably an earlier `git worktree add` invocation that mishandled it).
2. The main session running in `/workspaces/aiwf` saw `git status` reporting 102 uncommitted changes — none of which were in `/workspaces/aiwf`'s actual working tree. They were the M-0124 session's WIP, visible through the misdirection.
3. `git add .claude/statusline.sh` succeeded but staged from the wrong path. `git diff` reported empty for files actually edited in the main worktree. Edits made in this session went *to disk in the right place* but were *invisible to git*.
4. The pre-commit hook ran tests against the misdirected worktree's source tree, failing on M-0124's in-progress test code.
5. Diagnosis required reading `.git/config` by hand, recognizing the misset, and unsetting it via `git config --local --unset core.worktree`.

The whole session burned ~30 min on diagnosis. With a kernel-level check rule, the first `aiwf` invocation would have emitted a clear, named finding pointing at the misset.

## The right place for the rule

`aiwf check` is the authoritative chokepoint (pre-push hook, runs on every push), but the rule should fire even earlier — every verb that loads the tree depends on the right tree being loaded. Two surfaces benefit:

1. **`aiwf check`** (canonical): adds a finding named `git-config-core-worktree-misset`. Fires when `git config --get core.worktree` returns a value that doesn't match the directory containing `.git`. Severity: error (blocks the push).
2. **`aiwf doctor`** (secondary): same probe, surfaced in the doctor's narrative output. Doctor is the verb operators run when "something feels off" — having the finding here too helps the operator who never gets as far as a `check`.

The finding lives in `internal/cli/check/` (CLI layer with git access), not `internal/check/` (pure-tree layer). Wired into the `RunE` of `aiwf check` next to the other CLI-layer findings (provenance, contracts, metrics).

## Fix shape

New file `internal/cli/check/git_config.go`:

```go
package check

import (
    "context"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/23min/aiwf/internal/check"
)

// RunGitConfigCheck verifies that core.worktree, if set, resolves to
// the directory containing .git. A mismatch silently redirects every
// git operation against the wrong working tree and is a near-perfect
// footgun (no error, no warning, no obvious symptom).
func RunGitConfigCheck(ctx context.Context, root string) []check.Finding {
    cmd := exec.CommandContext(ctx, "git", "config", "--local", "--get", "core.worktree")
    cmd.Dir = root
    out, err := cmd.Output()
    if err != nil {
        // git config exits 1 when the key is not set — the normal
        // case for a healthy repo. No finding.
        return nil
    }
    configured := strings.TrimSpace(string(out))
    if configured == "" {
        return nil
    }
    abs, _ := filepath.Abs(configured)
    expected, _ := filepath.Abs(root)
    if abs == expected {
        return nil
    }
    return []check.Finding{{
        Code:     "git-config-core-worktree-misset",
        Severity: check.SeverityError,
        Message:  "core.worktree=" + configured + " does not resolve to the repository root " + expected,
        Path:     ".git/config",
        Hint:     "Run `git config --local --unset core.worktree` from the repo root unless you specifically need the override (rare; bare repos only). See gap G-0155.",
    }}
}
```

Wire it into the check verb (`internal/cli/check/check.go`) right after the metrics check:

```go
findings = append(findings, RunGitConfigCheck(ctx, resolved)...)
```

ACs (suggested):

- AC-1: When `core.worktree` is unset (the healthy state), `RunGitConfigCheck` returns no findings.
- AC-2: When `core.worktree` is set to the directory containing `.git` (legitimate but rare; e.g., bare-repo workflows), no finding.
- AC-3: When `core.worktree` is set to a path that does not resolve to the directory containing `.git`, a `git-config-core-worktree-misset` error finding is emitted with the configured path and the expected root in the message.
- AC-4: The pre-push hook blocks the push when the finding fires.

## Test approach

Three subtests under `internal/cli/check/git_config_test.go`, each setting up a temp git repo and writing the relevant `core.worktree` value via `git config --local`. The test exercises the live `git config` shell (since the rule fundamentally depends on it) and asserts the finding's presence/absence + the message shape.

## Future expansion

This is the first finding in a likely "git worktree configuration sanity" family. Sibling findings worth adding when they bite:

- `git-worktree-orphan-pruned-dir`: `.git/worktrees/<name>/` exists but the worktree path is gone.
- `git-worktree-gitdir-mismatch`: linked worktree's `gitdir` pointer doesn't match its actual location.
- `git-worktree-locked-stale`: worktree locked indefinitely but never released.

Each gets its own finding code; this gap is scoped to `core.worktree` only (KISS / YAGNI).

## History

Surfaced during the M-0124 wrap session when the operator (with the AI assistant) burned ~30 min diagnosing a confusing "102 uncommitted changes" report that turned out to be the M-0124 session's WIP visible through the misdirected `core.worktree`. The diagnostic walk is preserved in the session transcript; the relevant chunks were folded into the discussion that produced this gap.
