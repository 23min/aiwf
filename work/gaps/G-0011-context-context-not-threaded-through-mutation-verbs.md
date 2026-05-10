---
id: G-0011
title: '`context.Context` not threaded through mutation verbs'
status: addressed
addressed_by_commit:
  - 97283c0
---

Resolved in commit `97283c0` (refactor(aiwf): G11 — thread context.Context through every mutating verb). Every mutating verb (Add, Promote, Cancel, Rename, Move, Reallocate, Import, ContractBind, ContractUnbind, RecipeInstall, RecipeRemove) now takes ctx as its first argument. CLI dispatchers in `cmd/aiwf` already had ctx in scope; tests use `context.Background()` or the runner's `r.ctx`. Today the verb bodies are pure-projection (the IO is in Apply, gitops, tree.Load) so this is a discipline/future-proofing fix, but it aligns with `CLAUDE.md` and gives a clean cancellation handle when verbs grow IO-touching helpers.

---
