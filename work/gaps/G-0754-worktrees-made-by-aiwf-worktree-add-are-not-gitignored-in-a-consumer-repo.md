---
id: G-0754
title: Worktrees made by aiwf worktree add are not gitignored in a consumer repo
status: open
---
## What's missing

`aiwf worktree add` places worktrees under `worktree.dir`, by default `.claude/worktrees` (`DefaultWorktreeDir`, `internal/config/config.go:832`), and aiwf states in several places that the directory is ignored: the constant's comment ("gitignored via `.claude/*`", line 831); the context of [ADR-0023](../../docs/adr/ADR-0023-default-to-in-repo-worktree-placement-under-claude-worktrees.md) ("In-repo worktrees are reachable, persistent, and gitignored (`.claude/*`)", line 37); the guidance fragment shipped for Claude Code (`internal/skills/embedded-guidance/claude/epic-worktree-placement.md:1`); and the `aiwfx-start-epic` skill ("gitignored", `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md:113`). No ignore pattern aiwf writes covers it. `ensureGitignore` (`internal/initrepo/initrepo.go:1206`) writes the patterns `GitignorePatternsFor` returns (`internal/skills/skills.go:547`), and none of them names `.claude/worktrees` or `.claude/*`; `README.md:538` says "The entire `.claude/` or `.agents/` directory is not ignored wholesale." This repository's own `.gitignore` carries `.claude/*` at line 38, which hides the defect here.

Reproduced 2026-10-31 with aiwf built from `c06ffdaf9` (`vcs.modified=false`, the code `main` holds at `35d7cf109`), git 2.51.1, Linux. A fresh `git init` repository with `hosts: []` in `aiwf.yaml`, then `aiwf init`, which installed the `commit-msg`, `post-commit`, `pre-commit` and `pre-push` hooks, a commit, and a push to a bare remote; with `hosts: [claude-code]` the status and staging output was the same:

```
$ aiwf worktree add feature/x --base main --print-path
…/.claude/worktrees/feature/x
$ git status --porcelain
?? .claude/
$ git check-ignore -v .claude/worktrees/feature/x; echo "exit $?"
exit 1
$ aiwf status --worktrees | grep "⎇ main"
  ⎇ main  •  last commit just now  •  dirty
$ git add -A
warning: adding embedded git repository: .claude/worktrees/feature/x
$ git diff --cached --raw
:000000 160000 0000000 29787ed A	.claude/worktrees/feature/x
$ git commit -qm "chore: tidy"; echo "exit $?"
exit 0
$ git push -q origin main; echo "exit $?"
exit 0
$ git ls-tree -r origin/main -- .claude
160000 commit 29787ed504005f259147e05b185850231988973e	.claude/worktrees/feature/x
```

`git add -A` went on to print git's embedded-repository hint, cut above. In a fresh clone of the remote:

```
$ ls -A .claude/worktrees/feature/x | wc -l
0
$ git submodule update --init; echo "exit $?"
fatal: No url found for submodule path '.claude/worktrees/feature/x' in .gitmodules
exit 128
```

## Why it matters

In a consumer repository, every worktree made at the default placement shows as untracked in the main checkout, and `aiwf status --worktrees` marks the main checkout `dirty` for as long as the worktree exists, so the marker no longer says whether the main checkout holds uncommitted work.

`git add -A` in the main checkout stages each worktree as a gitlink, and with aiwf's hooks installed the commit and the push both go through. The default branch then carries an entry that a clone checks out as an empty directory and that `git submodule` cannot initialize, since no `.gitmodules` names it.

## Related

[G-0743](G-0743-the-repo-lock-lands-in-the-project-tree-in-a-linked-worktree.md) records an untracked lock file inside the worktree's own checkout; this gap concerns the main checkout holding the worktree.
