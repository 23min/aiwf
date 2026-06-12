---
id: G-0221
title: 'Disk-level atomic writes: no central temp+fsync+rename helper'
status: addressed
addressed_by_commit:
    - 1aba9334
---
## What's missing

A central, audited `pathutil.AtomicWriteFile(path, data, perm)` helper that implements the "write to sibling temp file → `f.Sync()` → `os.Rename(tmp, path)`" pattern, and a discipline that every kernel write to a persistent file routes through it. Today the pattern is duplicated in two places (`internal/aiwfyaml/aiwfyaml.go:187-197` and `internal/skills/skills.go:496-512`), and the rest of the kernel writes directly via `os.WriteFile` / `os.Create` / `io.Copy` — meaning an OS crash or hard-kill mid-write can leave the file half-written.

Concrete write sites that currently bypass the temp+rename pattern:

- `internal/verb/apply.go:140` — the central mutation writer used by every mutating verb (`promote`, `cancel`, `archive`, `edit-body`, etc.). The verb-level transactional rollback shipped under G-0002 protects against mid-verb *logic* errors (Apply restores HEAD on partial failure), but it does not protect against an OS crash or `kill -9` between `os.WriteFile` and the subsequent `git add`. The file on disk is the durability boundary; rollback can only restore from HEAD if the on-disk state reaches a consistent point.
- Three `internal/config/config.go` rewrites (around lines 340, 397, 463) that update `aiwf.yaml` outside the doc-helper path.
- `internal/skills/settings.go:95` — the settings-json write for the statusline opt-in.
- The htmlrender per-file emit under `internal/render/htmlrender/` — N files written in sequence with no all-or-nothing guarantee.

Reconfirmed by the 2026-06-04 codebase health scorecard (C3 verdict: Weak; see `docs/pocv3/health-scorecard-2026-06-04.md`). Note: this gap is distinct from the archived G-0002 — that one covered *verb-level transactional rollback* on partial logic failure inside `Apply`; this one covers *disk-level write atomicity* against OS-crash-mid-write, which is a different layer of the same correctness story.

## Why it matters

The kernel's "every mutating verb produces exactly one git commit" guarantee (CLAUDE.md, design-decisions §7) gives per-mutation atomicity *at the git layer*. The atomicity does not extend to the filesystem step that precedes the commit. If the OS crashes between `os.WriteFile(path, newBody)` and `runGit("add", path)`, the worktree carries a half-written file that:

- `aiwf check` may now refuse to load (frontmatter truncated, body cut mid-line).
- Git sees as a dirty working-tree state that won't match HEAD.
- The next `aiwf` invocation may pick up the partial file as "current state" and double-write or misvalidate.

For an interactive CLI used inside an editor's terminal where Ctrl-C / window-close / sleep-mode are routine, this is not a theoretical concern. The two existing temp+rename sites (`aiwfyaml.go`, `skills.go`) demonstrate the pattern is understood — but the discipline is one-of, not architectural.

## Candidate path

1. Add `internal/pathutil/atomic.go` exposing `AtomicWriteFile(path string, data []byte, perm os.FileMode) error` with the canonical sequence: `os.CreateTemp(dir, base+".aiwf-tmp-")`, `f.Write(data)`, `f.Sync()`, `f.Close()`, `os.Rename(tmp, path)`. Clean up the temp on any error path.
2. Route the four named write sites through it. The htmlrender per-file emit is the biggest call site and the most exposed to "render killed mid-run leaves a half-finished site" — also the most natural fit.
3. For the SKILL.md-loop + manifest pair (`internal/skills/skills.go` writes both and they must agree), use a staging-directory pattern: build the new layout under `.claude/.aiwf-staging-<pid>/`, then a single `os.Rename` swap to `.claude/`. Falls back to per-file `AtomicWriteFile` if `os.Rename` on the directory fails (cross-device).
4. Add an `internal/policies/atomic_write_chokepoint_test.go` that AST-walks the kernel under `internal/` and flags any `os.WriteFile`, `os.Create`+`Write`, or `ioutil.WriteFile` call whose target path is determined at runtime to be a persistent (non-temp, non-`/tmp/`) file. Allowlist the few legitimate cases (testdata fixtures, golden-file regeneration) with a one-line rationale. Same chokepoint shape as `PolicyNoHardcodedEntityPaths`.

The policy test is the load-bearing piece — without it, the next reviewer reaches for `os.WriteFile` and the discipline rots back to one-of.
