---
id: G-0134
title: aiwf init bakes absolute aiwf path in hooks; survives multi-context dev poorly
status: open
---
## What's missing

`aiwf init` writes git hooks with an absolute path to the current
shell's `aiwf` binary baked in: `'/Users/.../go/bin/aiwf check ...'`
on macOS host, `'/go/bin/aiwf check ...'` inside a Linux
devcontainer. The same `.git/hooks/` directory is shared between
host and container via the workspace bind-mount, but only one
absolute path can live in the hook at a time. Whichever
environment last ran `aiwf init` "wins" until the other side
re-runs init — at which point the first side breaks until *it*
re-runs init. Recurring tug-of-war on every context switch.

Aggravated for worktrees on devcontainers: `aiwf init` correctly
writes a per-worktree hook under `.git/worktrees/<n>/hooks/`, but
git fires commit/pre-push hooks from the **common** `.git/hooks/`
directory for both main and worktree contexts. The per-worktree
hook is never consulted for commits, so writing there silently
fails to do anything for the operator's commit flow.

## Why it matters

Every cross-context commit fails its pre-commit hook with
`<wrong-path>/aiwf: not found` or similar. The user discovers
this only at commit time, after they've finished typing the
message and answered the editor. Re-running `aiwf init` from the
current context unblocks the immediate commit but re-introduces
the failure for the other context. Across this session (M-0132
implementation) the hook seesaw blocked several commits and we
had to hand-patch the common-dir hooks with a path-probe pattern
just to land the wrap sequence.

The structural fix is for `aiwf init` to write hooks that probe
multiple known paths (or fall back to PATH lookup with a
deterministic chain) instead of baking one absolute path:

```sh
for AIWF in /go/bin/aiwf "$HOME/go/bin/aiwf" /usr/local/bin/aiwf; do
    [ -x "$AIWF" ] && break
done
[ -x "$AIWF" ] || { echo "aiwf binary not found" >&2; exit 1; }
exec "$AIWF" check ...
```

This is what's in place locally in this repo's `.git/hooks/`
after the hand-patch — survives host ↔ container ↔ worktree
context switches without re-init dance.

Same flavor as the plugin-state path-strictness issue surfaced
during M-0132 (filed as a sibling gap; both bite the same
multi-context dev pattern). The proper fix landing across both
gaps is the natural scope for a sibling milestone under E-0035
after the devcontainer skeleton wraps.

Related context:
- Discovered during M-0132 implementation, in particular the
  Reopen-in-Container retry loop where `init.sh` ran `aiwf init`
  in the container and broke subsequent host-side commits.
- The recovery section of M-0132's body documents the
  symptom-and-immediate-fix; this gap captures the structural
  fix needed in the kernel.
- The hand-patched hooks in `.git/hooks/{pre-commit,pre-push,post-commit}`
  are local-only (`.git/hooks/` is not version-controlled) and
  will be overwritten on the next `aiwf init` from either side.
  That's acceptable until the kernel-side fix lands; until then,
  the contributor knows the recovery dance from M-0132's body.
