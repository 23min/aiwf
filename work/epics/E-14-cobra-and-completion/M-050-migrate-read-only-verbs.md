---
id: M-050
title: Migrate read-only verbs
status: draft
parent: E-14
---

## Goal

Migrate `check`, `history`, `doctor`, `schema`, `template`, and `render` to the Cobra-shaped surface. Read-only verbs go first because they don't mutate state — recovery from failure is trivial and the migration risk is bounded.

## Approach

One verb at a time. Each verb's `--format=json` envelope is the contract; preserve byte-exact JSON output while letting Cobra control text/help formatting (the human surface can change; the machine surface cannot). Subprocess integration tests are the proof — if they pass, the migration is invisible to downstream consumers.

## Acceptance criteria
