---
id: G-0025
title: Pre-commit policy hook is per-clone, install-by-copy — drifts silently
status: addressed
addressed_by_commit:
  - 40c3d2d
---

Resolved in commit `40c3d2d` (build(repo): G25 — adopt core.hooksPath for the tracked pre-commit hook). The policy gate that enforces G21's discoverability rule (and every other policy under `internal/policies/`) lived in `.git/hooks/pre-commit` — installed per clone via `make hooks` (install-by-copy of `scripts/git-hooks/pre-commit`). The model has two failure modes:

1. **Drift.** The installed copy can fall behind the tracked source between `make hooks` runs. Concrete reproducer at gap-filing time: this very repo's tracked `scripts/git-hooks/pre-commit` (May 1) only regenerated `STATUS.md`; the installed `.git/hooks/pre-commit` had drifted ahead with the policies test gate. Nothing detected this — the only signal would have been a contributor running `make hooks` and noticing the file change in the diff.
2. **First-clone footgun.** A new contributor who clones and starts committing without running `make hooks` skips the policy gate entirely. CI catches it eventually, but every PR that lands in that window is one the contributor could have caught locally.

The fix is structural, not just procedural: switch from install-by-copy to `git config core.hooksPath scripts/git-hooks`. Git then executes the tracked file directly — no `.git/hooks/<name>` copy exists, no drift can occur, and `git pull` updates everyone's hook in sync with the policy it enforces. The `make hooks` target is renamed to `make install-hooks`; the README's new "Contributing to aiwf" section instructs new contributors to run it once after cloning. The hook itself stays tolerant (missing `go` is silently skipped) so doc-only commits from a non-Go environment aren't blocked.

Severity: Medium. Doesn't break correctness when the hook is current, but the safety net the policies package was designed to be is only real for contributors who have the up-to-date hook installed — and the install-by-copy model gave no signal when that wasn't true.

This gap is repo-internal: it applies to the kernel-development repo (`ai-workflow-v2`), not to consumer repos using `aiwf init`. Consumer-side hooks are managed by the kernel binary and refreshed by `aiwf update`; that path has its own drift-detection story under G12 (pre-push) and is out of scope here.

---

<a id="g24"></a>
