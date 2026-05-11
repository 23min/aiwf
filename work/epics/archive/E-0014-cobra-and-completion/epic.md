---
id: E-0014
title: Cobra and completion
status: done
---

## Goal

Migrate aiwf from stdlib `flag` to `github.com/spf13/cobra` so that every verb, subverb, flag, and closed-set value is tab-completable in bash and zsh — including dynamic enumeration of live entity ids. Establish "CLI surfaces must be auto-completion-friendly" as a load-bearing kernel principle, mechanically enforced by a drift-prevention test rather than reviewer vigilance.

## Scope

- Add Cobra to `go.mod` with one-line dep justification.
- Refactor `cmd/aiwf/main.go` from flag-based dispatch to a Cobra root command, preserving exit codes (0/1/2/3), `--format=json` envelope shape, single-commit-per-mutation discipline, and trailer-key behavior.
- Migrate every existing verb (`check`, `add`, `promote`, `cancel`, `rename`, `reallocate`, `init`, `update`, `upgrade`, `history`, `doctor`, `render`, `import`, `schema`, `template`, `version`).
- Ship `aiwf completion bash|zsh` (the kubectl/gh idiom — user evals it from their rc file; doesn't touch the consumer repo).
- Wire static completion for subverbs, kinds, statuses, format names. Wire dynamic completion for entity ids (`--epic=<TAB>` shells back to aiwf).
- Add the design principle to `CLAUDE.md` and a drift-prevention test in `internal/policies/` so a flag without completion wiring fails CI.

## Out of scope

- Verb semantics (this is a structural refactor only).
- Changes to the `--format=json` envelope shape.
- The `list` verb (its own future epic, depends on this one landing first).
- fish / powershell completion.
- Pagination, output formatting beyond what already exists.
- Deprecation grace period — this is a big-bang compatibility break. Help text describes the surface as it is, with no "previously was" / "renamed from" notes.
