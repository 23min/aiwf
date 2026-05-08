---
id: G-057
title: Stray aiwf binary in repo root from local builds is not gitignored
status: open
---

## What's missing

`aiwf init` and `aiwf update` reconcile a fixed set of patterns into the consumer's `.gitignore`: skill cache via `internal/skills/skills.go`'s `GitignorePatterns()`, html render output dir via `internal/initrepo/initrepo.go`'s `ensureGitignore` and `htmlOutDirIgnore` helpers. Neither emits `/aiwf` — the local-build artifact a bare `go build ./cmd/aiwf` drops at the repo root. Any consumer that builds the binary in-tree gets no protection from the kernel.

This kernel repo's own `.gitignore` carries `/aiwf` since `ba52ba2` (May 7), but that's a maintainer hand-edit. Running `aiwf init` / `aiwf update` here would not have produced it. The symptom is mitigated for this repo only.

## Why it matters

- **`git add -A` captures the binary.** A several-megabyte ELF/Mach-O lands in a commit, the push goes out, and the repo's history carries a binary nobody wanted. Rewriting history to remove it is expensive.
- **Combined with the G-056-class pollution** (a render run on a stale init), `git status` becomes unreadable — burying real changes.

The kernel's "framework correctness must not depend on operator behavior" rule applies: the `/aiwf` line should ship from `aiwf init`, not from a build incantation operators are expected to remember.

## The real fix

Extend `internal/skills/skills.go`'s `GitignorePatterns()` (or carve out a sibling helper for engine-level artifacts — the current name is skill-specific) to emit `/aiwf` alongside the skill patterns. `ensureGitignore` already iterates the returned slice and reconciles each entry idempotently; no other call site needs to change. Add a unit test asserting `/aiwf` appears post-`init` on a fresh consumer tree.

The leading slash is load-bearing — it anchors to repo root so `cmd/aiwf/` and any future package named `aiwf` stay trackable.

Once shipped, promote G-057 to `addressed` with the implementing commit as `--by-commit`.

## Out of scope

- A `doctor --self-check` hint about a stray `./aiwf` in the working tree. The init-time write is sufficient; a runtime hint is noise.
- Centralizing build-artifact path conventions across all aiwf-style binaries. Current scope: just `aiwf`.

## Related

- [G-056](G-056-aiwf-render-output-site-is-not-gitignored-pollutes-consumer-working-tree.md) — same class of defect (framework-produced artifact without gitignore coverage); html out_dir reconciliation shipped in `056139d`.
