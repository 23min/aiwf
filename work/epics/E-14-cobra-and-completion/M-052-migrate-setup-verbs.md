---
id: M-052
title: Migrate setup verbs
status: in_progress
parent: E-14
acs:
    - id: AC-1
      title: init, update, upgrade migrated to Cobra
      status: met
    - id: AC-2
      title: Marker-based artifact regeneration preserved (skills, hook markers)
      status: met
    - id: AC-3
      title: aiwf doctor --self-check passes after init in fresh consumer repo
      status: met
    - id: AC-4
      title: Installed git hooks remain byte-identical or update goldens with rationale
      status: met
---

## Goal

Migrate `init`, `update`, and `upgrade` — verbs that touch the consumer repo's marker-managed artifacts (gitignored skills under `.claude/skills/aiwf-*` and hook markers under `.git/hooks/<hook>`). These are the verbs most likely to surprise downstream consumers if behavior drifts; extra care goes into hook idempotency and marker preservation.

## Approach

Test against a fresh tempdir consumer repo (per the existing pattern). After `init`, `aiwf doctor --self-check` must pass — that's the integration anchor. Hooks installed under `.git/hooks/` must be byte-identical to the pre-migration installer's output (or deliberately updated, in which case the change goes through the doctor goldens with rationale).

## Acceptance criteria

### AC-1 — init, update, upgrade migrated to Cobra

### AC-2 — Marker-based artifact regeneration preserved (skills, hook markers)

### AC-3 — aiwf doctor --self-check passes after init in fresh consumer repo

### AC-4 — Installed git hooks remain byte-identical or update goldens with rationale

