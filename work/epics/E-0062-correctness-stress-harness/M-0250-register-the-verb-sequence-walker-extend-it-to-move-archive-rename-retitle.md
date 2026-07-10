---
id: M-0250
title: Register the verb-sequence walker; extend it to move/archive/rename/retitle
status: draft
parent: E-0062
depends_on:
    - M-0249
tdd: required
acs:
    - id: AC-1
      title: cmd/stresstest registers and can run the verb-sequence walker standalone
      status: open
      tdd_phase: red
    - id: AC-2
      title: the walker's legal-transition set includes move, archive, rename, and retitle
      status: open
      tdd_phase: red
    - id: AC-3
      title: a post-step invariant cross-checks aiwf list's output against ground truth
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — cmd/stresstest registers and can run the verb-sequence walker standalone

### AC-2 — the walker's legal-transition set includes move, archive, rename, and retitle

### AC-3 — a post-step invariant cross-checks aiwf list's output against ground truth

