Move with `cd`, not the `EnterWorktree` tool: a session entered that way cannot reach mainline's worktree, and steps 11 through 15 run there.

Stay inside the repository while you work: a `cd` to a directory outside it returns the session to the directory it started in, not to this worktree. Run anything that needs another directory in a subshell, `( cd <dir> && … )`.
