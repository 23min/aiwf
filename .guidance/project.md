# Project guidance

`CLAUDE.md` holds the rules for every task in this repository, and they take
precedence over the packs. Before the first edit a task governs, read the
document that matches it:

- [Testing](project/testing.md) — writing or changing Go tests, choosing an
  acceptance criterion's evidence, the coverage gates, running tests on macOS.
- [Verb design](project/verb-design.md) — adding or changing a verb, a check
  rule or the loader.
- [Go conventions](project/go-conventions.md) — Go code in this repository;
  where it differs from the Go pack, logging included, it overrides the pack.
- [Environment](project/environment.md) — setup, the devcontainer, subagent
  worktrees, diagnosing with a binary built from a worktree.
- [Shipped content](project/shipped-content.md) — editing embedded skills,
  rituals, templates or the guidance fragment.
- [Stress harness](project/stress-harness.md) — stress scenarios and their
  lanes.
