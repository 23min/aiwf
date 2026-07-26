---
id: M-0281
title: Same-state mutating-verb inputs return NoOp
status: draft
parent: E-0073
tdd: required
acs:
    - id: AC-1
      title: promote to current status returns NoOp instead of an error
      status: open
      tdd_phase: red
    - id: AC-2
      title: cancel of an already-terminal entity returns NoOp
      status: open
      tdd_phase: red
    - id: AC-3
      title: move to the current parent epic returns NoOp
      status: open
      tdd_phase: red
    - id: AC-4
      title: acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit
      status: open
      tdd_phase: red
    - id: AC-5
      title: rename to current slug and retitle to current title return NoOp
      status: open
      tdd_phase: red
    - id: AC-6
      title: verb_result_noop_invariant policy pins same-state NoOp across mutating verbs
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — promote to current status returns NoOp instead of an error

### AC-2 — cancel of an already-terminal entity returns NoOp

### AC-3 — move to the current parent epic returns NoOp

### AC-4 — acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit

### AC-5 — rename to current slug and retitle to current title return NoOp

### AC-6 — verb_result_noop_invariant policy pins same-state NoOp across mutating verbs

