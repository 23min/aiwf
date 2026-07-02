---
id: M-0225
title: 'aiwf health producer: doctor --write-health + lifecycle refresh'
status: draft
parent: E-0055
depends_on:
    - M-0224
tdd: required
---
## Deliverable

Make aiwf a health producer. Map the doctor findings model onto the fixed
ai-dotfiles health-file schema and atomic-write `.claude/health.aiwf.json`,
resolved to the main checkout even from a linked worktree, and refresh it on
every installation-state-changing lifecycle event. Delivers the aiwf-producer
half of G-0305.

## Acceptance criteria (formalized at milestone start)

- **Schema-mapped atomic write.** `aiwf doctor --write-health` writes a
  schema-valid `health.aiwf.json` (`source: "aiwf"`; blocking → error, advisory →
  warn, informational → info); a healthy repo yields an empty `findings` array;
  the write is atomic (temp + rename). Evidence: a seeded-problem repo produces
  the mapped error finding; a healthy repo produces an empty array; no temp file
  remains after the write.
- **Main-checkout resolution.** Invoked from a linked worktree, the file lands in
  the main checkout's `.claude/`, not the worktree's. Evidence: a worktree test
  asserting the resolved destination path.
- **Schema contract.** A contract test pins the emitted JSON against the fixed
  ai-dotfiles schema (required keys, ISO8601-UTC `generated_at`, the severity
  enum), with the expected shape derived independently and gated under `-short`.
  The timestamp is injected at the CLI edge.
- **Lifecycle refresh.** `aiwf init`, `aiwf update`, and `aiwf upgrade` each
  refresh `health.aiwf.json` on success. Evidence: a table-driven test with one
  subtest per verb.
