---
id: G-0501
title: aiwf init and update replace a symlinked CLAUDE.md with a copy
status: open
priority: high
discovered_in: M-0284
---
## What's missing

The guidance-import wiring reads `CLAUDE.md` with `os.ReadFile` (which follows a
symlink), and writes it back with `pathutil.AtomicWriteFile`, whose documented
semantics replace the path rather than write through it. The existence probe uses
`os.Stat`, which follows too, so the step sees a file and reports the import wired.

Measured, on a repo where `CLAUDE.md` is a tracked symlink to `AGENTS.md`:

    before:  120000 blob …  CLAUDE.md    ->  lrwxrwxrwx CLAUDE.md -> AGENTS.md
    aiwf init                             ->  exit 0, no warning
    after:   -rw-r--r-- CLAUDE.md              git status:  T CLAUDE.md

The link is gone and `CLAUDE.md` is a frozen copy. A later edit to `AGENTS.md` no
longer reaches it — verified by appending a line to `AGENTS.md` and finding it
absent from `CLAUDE.md`. `aiwf update` destroys the link again after the operator
restores it.

## Why it matters

Symlinking `CLAUDE.md` to a shared `AGENTS.md` so one file serves several agent
tools is a common convention, and this fires on the first `aiwf init` in such a
repo. Nothing reports it: exit 0, no finding, and the step's own message says the
guidance import was wired.

The consequence is a silent fork. The operator keeps editing the file they think is
canonical; the agent keeps reading a copy that stopped tracking it. Neither surface
says the two have diverged.

This is the defect class E-0075 closed at the commit seam, reached through a writer
that does not use that seam. `checkCarriedSymlinks` guards `verb.Apply`; the
guidance wiring never goes through `verb.Apply`, so the guard cannot see it.

## Scope

Whether to refuse or to write through is a decision, not a mechanical fix.

- **Refuse.** `os.Lstat` before the write; report the link and leave it alone.
  Consistent with the commit-seam guard, and the operator keeps what they built.
- **Write through.** Resolve the link and write to its target, so the shared file
  gains the marker block and every tool that reads it sees the same content.

Writing through is likely what the operator wanted, but it edits a file outside the
repo's own tree when the target is a relative path pointing elsewhere — which is
why this is a decision rather than an obvious correction.

Any non-`verb.Apply` writer that touches an operator-owned path shares this shape;
the fix should say which writers were audited.

## References

- `internal/initrepo/initrepo.go` — the read, the existence probe, and the write
- `internal/pathutil/atomic.go` — replace-not-write-through semantics
- `internal/verb/apply.go` — `checkCarriedSymlinks`, the equivalent guard at the commit seam
- ADR-0018 — the guidance-import wiring this affects