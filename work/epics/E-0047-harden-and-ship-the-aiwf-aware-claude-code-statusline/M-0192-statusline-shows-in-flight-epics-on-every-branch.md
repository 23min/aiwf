---
id: M-0192
title: Statusline shows in-flight epics on every branch
status: in_progress
parent: E-0047
depends_on:
    - M-0191
tdd: required
---
## Deliverable

The statusline entity segment shows in-flight epics on **every** branch (G-0188), not just ritual branches — today the most informative slot is blank in the commonest context (`main`).

- On `main` / non-ritual branches: render all non-terminal epics (proposed/active) with the canonical glyph/color language from `aiwf status --worktrees` (`→` active yellow, `○` proposed blue; terminal `done`/`cancelled` filtered out).
- On ritual branches: accentuate the current epic (the one the branch belongs to) and show its milestone/gap inline; other in-flight epics render visually secondary.
- Cap at ~3 shown with `+N` overflow to keep the line scannable.

Tested against the M1 behavioral harness. ACs to be defined at milestone start.
