---
id: G-050
title: Pre-commit hook aborts when STATUS.md is gitignored — violates 'tolerant by design' contract, orphans .git/index.lock
status: addressed
addressed_by_commit:
  - 572bc96
---

## What's missing

The pre-commit hook template that `aiwf init` installs regenerates `STATUS.md` and stages it via `git add "$repo_root/STATUS.md"`. The hook header declares STATUS regeneration "tolerant by design — never blocks commits". The actual code contradicts that contract: when `STATUS.md` is gitignored in the consumer repo, `git add` (without `--force`) refuses the path and exits non-zero. Combined with `set -e` at the top of the hook, this aborts the entire pre-commit run and fails the commit — even though the regeneration was meant to be best-effort.

The breakage cascades into every path that triggers the pre-commit hook: every `aiwf promote` / `aiwf add` / `aiwf cancel` invocation (each writes a commit), every developer `git commit`, and `git stash push` (stash invokes pre-commit internally). When the hook aborts mid-flight, git commonly leaves a zero-byte `.git/index.lock` behind, which then blocks subsequent git operations until the lock is manually removed. **This is the root cause of recurring stale `.git/index.lock` issues that masquerade as aiwf bugs.**

Reproduction (in a consumer repo where `STATUS.md` is gitignored):
```bash
touch some-file && git add some-file && git commit -m "test"
# → hook fires, git add STATUS.md fails (gitignored), set -e aborts, commit fails,
#   .git/index.lock may be orphaned.
```

Why a consumer would gitignore `STATUS.md`: the file is regenerated on every commit and produces churn (every commit modifies it, every diff includes it, every PR has it as a noisy change). Gitignoring it is a reasonable consumer choice — and one we should expect.

## Why it matters

- **Productivity drag.** Every aiwf state transition (and every developer commit) is at risk of failure in the affected consumer repo. The downstream report (flowtime-vnext, 2026-05-05) describes "hitting and clearing the lock manually for days."
- **Hidden state corruption.** A stale `index.lock` blocks `aiwf add` / `aiwf promote`, surfacing as confusing error messages that look like aiwf bugs rather than hook bugs. Time wasted in misdiagnosis.
- **Hook contract violation.** The hook's own header comment promises "tolerant by design — never blocks commits". The actual behavior contradicts that contract.
- **The fix is one character.** Append ` 2>/dev/null || true` to the `git add` line, matching the tolerant-by-design semantics already documented in the hook header.

## Resolution path

The fix lives in the kernel's hook template (wherever the pre-commit body is generated and written by `aiwf init` / `aiwf update`). Three options, ordered by simplicity:

1. **Suppress the failure** — append ` 2>/dev/null || true` to the `git add` invocation. Matches the hook header's documented tolerance and is one line.
2. **Detect gitignored state first** — `git check-ignore --quiet "$STATUS_PATH"` before attempting `git add`; skip the staging step if the path is ignored.
3. **Honor consumer config** — read `aiwf.yaml`'s `status_md.auto_update` (or equivalent) and skip the staging step entirely when STATUS.md is gitignored. Largest scope; not warranted unless we want a broader knob.

Lean: option (1). It's the smallest change that respects the hook's own contract and handles every reason `git add` might fail (not just gitignore — also missing file, lock contention, etc.). The hook is meant to be best-effort; failure should never abort the commit.

Tests: per the kernel's "test the seam, not just the layer" rule, exercise the installed hook against a fixture repo where `STATUS.md` is gitignored — assert the hook exits 0 and the commit succeeds.

## References

- Downstream report: flowtime-vnext consumer repo, 2026-05-05; commit `7e9cc97` ("stop tracking generated STATUS.md") gitignored the path; symptom thread covered M-066 wrap and E-26 epic creation.
- Hook header contract: the "tolerant by design — never blocks commits" promise on the kernel's pre-commit template.
- Adjacent: G-045 (hook chaining via `.local` siblings) — same hook surface, different concern.
