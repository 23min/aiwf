---
id: G-0743
title: The repo lock lands in the project tree in a linked worktree
status: open
---
## What's missing

`lockfilePath` in `internal/repolock/repolock_unix.go` (lines 105-116) places the repository lock at `<root>/.git/aiwf.lock` only when `.git` is a directory. Its own comment states the intent: the lock sits under `.git/` "so the lockfile is naturally outside the project tree (and unaffected by gitignore rules)". In a linked worktree `.git` is a file, so the lock falls back to `<root>/.aiwf.lock`, inside the project tree.

The managed ignore block does not cover that path either. `ensureGitignore` (`internal/initrepo/initrepo.go:1206`) writes the patterns from `GitignorePatternsFor` (`internal/skills/skills.go:547`), and none of them names the lock. So after a mutating verb in a linked worktree the checkout holds an untracked `.aiwf.lock`; read-only verbs do not create it. Worktrees made by `aiwf worktree add` are affected as well as plain `git worktree add` ones. This repository's own `.gitignore` lists `.aiwf.lock` (line 73), which hides the defect here.

Reproduced 2026-10-25 with aiwf built from `1e8e17e6c`, git 2.51.1, Linux. Each run starts from a fresh `aiwf init` repository. In the first, one `aiwf add gap` ran in the main checkout, in a worktree from `git worktree add -b plain ../plain`, and in one from `aiwf worktree add viaaiwf --base main`; after each, the script printed `git status --short` and what `git add -A` would stage:

```
--- main checkout after a verb: status []  .git/aiwf.lock: present
--- plain: .git is a file; status after a verb: [?? .aiwf.lock ]
    git add -A would stage: [.aiwf.lock ]
--- viaaiwf: .git is a file; status after a verb: [?? .aiwf.lock ]
    git add -A would stage: [.aiwf.lock ]
```

In the second, inside a worktree from `aiwf worktree add wt --base main`, the script ran `aiwf status`, `aiwf list` and `aiwf check`, then `aiwf add gap`, then `git worktree remove` on that worktree from the main checkout:

```
after status, list, check: []
after aiwf add gap:        [?? .aiwf.lock ]
fatal: '…/.claude/worktrees/wt' contains modified or untracked files, use --force to delete it
git worktree remove exit: 128
```

M-0241/AC-4 records that the lock is scoped per worktree on purpose. This gap concerns where the file lands, not that scoping.

## Why it matters

`git worktree remove` refuses a worktree once any mutating verb has run in it. The shipped rituals close a worktree with that command (`wf-patch/SKILL.md:221` and `aiwfx-wrap-milestone/SKILL.md:294` under `internal/skills/embedded-rituals/`), so in a consumer repository cleanup at the end of a patch or a milestone wrap fails, and the way past it is `--force`, which also discards anything else left untracked.

`git add -A` in such a worktree stages the lock, and nothing between that commit and a merge refuses it.

`aiwf status --worktrees` marks every such worktree `dirty` from its first mutating verb on, so the marker no longer tells a reader whether the worktree holds uncommitted work.
