---
id: G-0524
title: Devcontainer mounts the repo's parent, exposing siblings and a rival CLAUDE.md
status: open
priority: medium
---
## What's missing

`.devcontainer/devcontainer.json` binds the **parent** of the clone, not the
clone:

```json
"workspaceFolder": "/workspaces/${localWorkspaceFolderBasename}",
"workspaceMount": "source=${localWorkspaceFolder}/..,target=/workspaces,type=bind,consistency=cached"
```

The workspace folder is the repo; the bind source is one level up. Every sibling
directory beside the clone is therefore visible and writable inside the
container, and so is any file sitting at that parent level.

The widening is deliberate. `.devcontainer/README.md` §"Reopen in Container"
states it: the mount goes one level up so sibling repos under the same parent are
reachable inside. `.devcontainer/init.sh` codes against it — when the workspace
folder is itself a git worktree of a *sibling* repo, it rewrites the worktree's
`.git` gitdir to a relative `../<main>/.git/worktrees/<name>` pointer, plus a
reverse pointer four levels up. That path arithmetic resolves only because
siblings share the mounted parent.

What the mount also does, unintended: it makes a parent-level `CLAUDE.md`
reachable inside the container. A session then has two candidate
project-instruction files — the repo's own and the parent's — and which one the
harness resolves is outside aiwf's control or observation. Without the widened
mount there is no second candidate at all.

## Why it matters

Two costs, one measured and one latent.

**Measured.** A session working in this repo carried the parent-level file as its
project instructions; this repo's `CLAUDE.md` and the guidance import it wires
never entered context. Every always-on operating rule was inactive, and three
review passes over a patch ran without the repo's own conventions before the
omission was caught by hand.

**Latent.** A container whose filesystem view is 30-plus unrelated repositories
gives every agent session in it write reach far outside the project it was opened
for. Nothing in aiwf's own discipline — gate-per-mutation, worktree isolation,
the check chokepoints — extends past this repository, so a stray path in a
generated command has no boundary to stop it.

## Fix shape

Narrow the bind to the clone:

```json
"workspaceMount": "source=${localWorkspaceFolder},target=/workspaces/${localWorkspaceFolderBasename},type=bind,consistency=cached"
```

`workspaceFolder` needs no change — it already resolves to the repo.

This is not a one-line edit, because it retires a documented capability. It lands
with:

- the sibling-worktree `.git` rewrite in `.devcontainer/init.sh` removed, since
  it becomes unreachable once siblings are unmounted — dead code with an
  explanatory comment is worse than no code;
- `.devcontainer/README.md`'s statement of the one-level-up rationale corrected
  to describe the narrowed mount;
- the replacement path recorded for the case the widening served: add an explicit
  entry to the `mounts` array when a session genuinely needs a specific sibling or
  worktree, which is narrower than the blanket parent bind and visible in the
  config rather than implicit in it.

## The trade-off this gap exists to settle

The parent mount buys cross-repo reach in one container — developing aiwf against
a downstream consumer tree, or opening the container directly on a sibling
worktree. Narrowing trades that for a container whose filesystem view equals the
project.

The lean at filing: the reach is rarely exercised, aiwf's own worktree convention
puts worktrees inside the repo rather than beside it, and the cases that do need a
sibling can be served by an explicit per-case mount. Recorded as a lean, not a
settled decision — the capability is documented and contributor-facing, so
dropping it belongs in a decision entity when the work is scheduled.

## Related

- **G-0523** — always-on guidance delivery rides `CLAUDE.md` discovery and can
  fail unobserved. Narrowing the mount removes one way that resolution goes wrong
  in this repo's own container; it does nothing for a consumer repo, which is what
  G-0523 covers. Neither substitutes for the other.

## Provenance

Filed after the parent-level `CLAUDE.md` was observed displacing this repo's own
in a live session. Deferred deliberately at that point rather than patched inline,
because the sibling-worktree handling and the contributor-facing README make it
larger than a config nudge.
