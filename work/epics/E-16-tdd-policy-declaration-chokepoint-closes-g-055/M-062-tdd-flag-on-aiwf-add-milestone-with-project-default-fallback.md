---
id: M-062
title: tdd flag on aiwf add milestone with project-default fallback
status: draft
parent: E-16
tdd: required
---

## Goal

Add `--tdd required|advisory|none` to `aiwf add milestone`, the load-bearing chokepoint for the epic. The verb resolves the policy in this order: explicit flag > `aiwf.yaml: tdd.default` > refuse with a clear error pointing the operator at the new flag. The resolved value is written to the new milestone's frontmatter and becomes the per-milestone source of truth that future audits and verbs read from.

After this milestone, `aiwf add ac` against a `tdd: required` parent continues to seed `tdd_phase: red` (existing behavior — no change required). The `aiwf-add` skill documents the new flag and the resolution order so an LLM following the skill produces well-specified milestones.

## Approach

The flag is added to the existing Cobra command in `cmd/aiwf/add_cmd.go`; the resolver lives next to the existing `aiwf.yaml` consumer code (`internal/configyaml/` or the package that owns the loaded config struct) and is called from the verb body before the entity is allocated. Static completion via `cobra.FixedCompletions` for the closed set, registered the same way the existing `--format`/`--status` completions are. Subprocess integration test exercises every resolution path including the refusal cases, per CLAUDE.md "test the seam." Aggressively reuse the existing project-config load path — do not introduce a parallel reader.

The error message for the no-default-no-flag case is part of the contract: it must name the flag (`--tdd`), the closed-set values, the config field (`aiwf.yaml: tdd.default`), and recommend `--tdd required` for code milestones.

## Acceptance criteria
