---
id: G-0735
title: Comments, a test message and an ADR cite CLAUDE.md rules it does not carry
status: open
discovered_in: M-0335
---
## What's missing

Code comments, a test failure message and one ADR cite `CLAUDE.md` for a rule under a name or number `CLAUDE.md` does not carry, so a reader following the citation finds nothing there.

- "Test untested code paths [before declaring code paths 'done']": `internal/cli/integration/render_archive_unit_test.go:15`, `internal/policies/aiwfx_plan_milestones_test.go:184`, `internal/policies/aiwfx_start_epic_test.go:229`, `internal/policies/envelope_version_source_test.go:15`, `internal/policies/m0123_ac7_rules_encapsulation_test.go:70`, `internal/policies/no_dangling_entity_refs_test.go:46`, `internal/policies/skill_coverage.go:67`, `internal/render/render_test.go:328`, `internal/verb/archive.go:160`, `internal/verb/archive_test.go:16`. The rule these name is "Test every reachable branch" in `.guidance/project/testing.md`.
- "scaffolding writes files", cited as `CLAUDE.md`'s in the failure message at `internal/cli/integration/single_commit_invariant_test.go:70`; no guidance document states it.
- "CLAUDE.md's self-explaining-error" rule at `internal/cli/integration/sovereign_force_coherence_test.go:387`; the rule is "Self-explaining operator errors" in `.guidance/packs/code-health/guide.md`.
- "the operator-discipline gap CLAUDE.md documents" at `internal/check/id_rename_untrailered_test.go:20` and `internal/cli/integration/id_rename_untrailered_scenarios_test.go:84`; the rule against `git mv` for an id is in the shipped guidance, `internal/skills/embedded-guidance/aiwf-guidance.md:39`.
- `docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md:73` and `:158` cite `CLAUDE.md` §5, "What aiwf commits to", for "framework correctness must not depend on LLM behavior"; that is an engineering principle in `CLAUDE.md`, and commitment 5 is selected-host artifacts.

```
$ git grep -nE 'Test untested|untested code paths before|scaffolding writes files|self-explaining-error|operator-discipline gap CLAUDE.md|CLAUDE.md §5|commits to" §5' -- internal/ cmd/ docs/adr/ | wc -l
16
$ grep -cE 'Test untested|scaffolding writes|self-explaining' CLAUDE.md
0
```

Expected: no citation of `CLAUDE.md` naming a rule it does not carry. Observed 2026-10-11 in the aiwf devcontainer, as above: the first count is the sixteen sites listed, the second shows `CLAUDE.md` carries none of those names.

Where a fix would land: the files listed.

## Why it matters

A comment or failure message that justifies code by a rule names a source where the rule is not found, so whoever next changes the code cannot check the reason it gives.
