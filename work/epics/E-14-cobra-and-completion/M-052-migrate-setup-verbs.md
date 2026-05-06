---
id: M-052
title: Migrate setup verbs
status: draft
parent: E-14
---

## Goal

Migrate `init`, `update`, and `upgrade` — verbs that touch the consumer repo's marker-managed artifacts (gitignored skills under `.claude/skills/aiwf-*` and hook markers under `.git/hooks/<hook>`). These are the verbs most likely to surprise downstream consumers if behavior drifts; extra care goes into hook idempotency and marker preservation.

## Approach

Test against a fresh tempdir consumer repo (per the existing pattern). After `init`, `aiwf doctor --self-check` must pass — that's the integration anchor. Hooks installed under `.git/hooks/` must be byte-identical to the pre-migration installer's output (or deliberately updated, in which case the change goes through the doctor goldens with rationale).

## Acceptance criteria
