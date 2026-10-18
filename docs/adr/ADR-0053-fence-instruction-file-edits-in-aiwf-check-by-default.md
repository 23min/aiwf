---
id: ADR-0053
title: Fence instruction-file edits in aiwf check by default
status: accepted
---
> **Date:** 2026-09-24 · **Decided by:** Peter Bruinsma (human), while starting E-0092

## Context

A repository's root instruction files — `CLAUDE.md` for Claude Code and `AGENTS.md` for Codex — are loaded before every task, so each line added to them is paid on every task. In practice they grow inside ordinary work: an edit rides in a commit that also changes code, and passes as that commit, so the growth is never put as its own question. G-0676 measures this in the aiwf repository itself.

aiwf operates in consumer repositories through defaults: the hooks `aiwf init` installs and the `aiwf check` rules the pre-push hook runs. Nothing there judges an instruction-file edit, so a consumer's instruction files grow unguarded by default. A shipped guidance rule telling agents to keep such edits separate holds only by vigilance, which is the failure being addressed.

Alternatives considered:

- **A size ceiling for consumers.** Rejected: a consumer has no principled basis for the number. It would be today's size, freezing past growth, or a guess raised the first time it bites — and it adds a setting, a marker for required reads, and a finding code to learn.
- **A size report in `aiwf doctor`.** Rejected: visibility without a guard, and one more output to learn.
- **Refusing at the `commit-msg` hook as well.** Deferred: an earlier catch, but a second call site and a change to installed hook scripts that takes effect only after `aiwf update`, so the pre-push check stays the guarantee either way.
- **Nested instruction files.** Out of aiwf's concern: aiwf neither covers nor plans to cover a `CLAUDE.md` or `AGENTS.md` below the root.

## Decision

- An edit to handwritten instruction content is its own commit. The content is the root `CLAUDE.md` and `AGENTS.md` outside aiwf's managed blocks, the project router `.guidance/project.md`, and the documents that router links to.
- `aiwf check` enforces it over unpushed commits, the range the pre-push hook judges. Such a commit may also carry only the other instruction files, aiwf's owned guidance outputs and their record, and `aiwf.yaml`; it names the entity it belongs to in an `aiwf-entity` trailer that resolves; and when it removes a handwritten line it records a disposition block.
- The finding is an error by default, so it blocks the push. An `aiwf.yaml` setting turns it off.
- A change confined to a managed block is generated output and is not judged. A merge commit is not judged; the commits it brings in are.
- Consumers get no size ceiling and no size report.
- The aiwf repository keeps two stricter checks internal: a ceiling on its own primed load, and D-0091's ban on new prose-presence assertions over its guidance.

## Consequences

- A consumer who upgrades and edits `CLAUDE.md` alongside code is blocked at push until the commit is split — a local amend or rebase, since the range is unpushed. The changelog and the `aiwf-check` skill state this and name the off switch.
- The rule needs a finding code, skill documentation, a config field, and discoverability wiring like any other `aiwf check` rule.
- The aiwf repository runs the same rule it ships. Its internal commit-range fence is replaced by the kernel rule; aiwf's own fragment source is no longer an allowed companion, so a commit editing handwritten `CLAUDE.md` and the fragment source together is split.
- Deliberate growth remains possible and is visible as its own trailered commit; aiwf does not limit its size.

## Validation

Revisit if consumers turn the setting off in numbers, or if pushes blocked by the rule show the companion set is too narrow for ordinary guidance updates.

## References

- E-0092 — the epic that ships the rule
- G-0676 — the growth this answers
- D-0091 — the internal pin scan
- ADR-0052 — project guidance independent of personal bootstrap
