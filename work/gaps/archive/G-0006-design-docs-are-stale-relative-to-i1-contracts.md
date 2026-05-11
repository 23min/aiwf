---
id: G-0006
title: Design docs are stale relative to I1 (contracts)
status: addressed
addressed_by_commit:
  - 221b9ff
---

Resolved in commit `221b9ff` (docs(poc): G6 — sync design decisions and plan with the I1 contract surface). `design-decisions.md` (then named `poc-design-decisions.md`) gains a "Contracts (added in I1)" subsection cross-referencing `contracts-plan.md`, the chokepoint section now mentions contract verification joining the same envelope, the `aiwf.yaml` table includes the `contracts:` row, the verb list reflects the current 14-verb surface (with G2's rollback and G4's lock noted), and the "deliberately not in the PoC" table drops the now-false "schema-aware contract validation" row. `poc-plan.md` gains an "Iteration I1 — Contracts" section listing all eight sub-iterations as done, the obsolete `contract-artifact-exists` and `add contract --format/--artifact-source` lines are annotated as superseded.

---
