---
id: M-050
title: Migrate read-only verbs
status: in_progress
parent: E-14
acs:
    - id: AC-1
      title: check, history, doctor, schema, template, render migrated to Cobra
      status: met
    - id: AC-2
      title: --format=json envelope output preserved byte-exact for each verb
      status: met
    - id: AC-3
      title: Exit codes preserved for each migrated read-only verb
      status: met
    - id: AC-4
      title: Subprocess integration tests pass for all six read-only verbs
      status: met
---

## Goal

Migrate `check`, `history`, `doctor`, `schema`, `template`, and `render` to the Cobra-shaped surface. Read-only verbs go first because they don't mutate state — recovery from failure is trivial and the migration risk is bounded.

## Approach

One verb at a time. Each verb's `--format=json` envelope is the contract; preserve byte-exact JSON output while letting Cobra control text/help formatting (the human surface can change; the machine surface cannot). Subprocess integration tests are the proof — if they pass, the migration is invisible to downstream consumers.

## Acceptance criteria

### AC-1 — check, history, doctor, schema, template, render migrated to Cobra

### AC-2 — --format=json envelope output preserved byte-exact for each verb

### AC-3 — Exit codes preserved for each migrated read-only verb

### AC-4 — Subprocess integration tests pass for all six read-only verbs

