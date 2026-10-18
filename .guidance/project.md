# Project guidance

`CLAUDE.md` holds the rules for every task in this repository, and they take
precedence over the packs. Before designing or editing what a task governs,
read the matching document:

- [Testing](project/testing.md) — writing or changing tests in any language,
  choosing an acceptance criterion's evidence.
- [Verb design](project/verb-design.md) — designing, adding or changing a
  verb, a check rule or the loader.
- [Environment](project/environment.md) — setting up a clone or the
  devcontainer, dispatching a subagent into a worktree of its own, diagnosing
  aiwf with a binary built from a worktree.
- [Shipped content](project/shipped-content.md) — editing embedded skills,
  rituals, templates or the guidance fragment.
- [Stress harness](project/stress-harness.md) — stress scenarios and their
  lanes.
- [Writing docs](project/writing-docs.md) — documents under `docs/` and ADRs.
- [Release](project/release.md) — cutting a release.
- [Enforcement](project/enforcement.md) — where a rule is enforced, from
  pre-commit to the release tag.
- [Language packs](index.md) — after any matching document above, the pack
  for the task's language or toolchain; for Go, then
  [Go conventions](project/go-conventions.md), which add to and override it.
