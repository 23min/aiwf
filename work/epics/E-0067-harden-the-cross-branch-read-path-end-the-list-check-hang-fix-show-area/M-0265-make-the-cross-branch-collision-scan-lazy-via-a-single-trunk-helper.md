---
id: M-0265
title: Make the cross-branch collision scan lazy via a single trunk helper
status: draft
parent: E-0067
tdd: required
acs:
    - id: AC-1
      title: One trunk helper composes the cross-branch scan for treeload, list, and show
      status: open
      tdd_phase: red
    - id: AC-2
      title: DetectCollisions runs only for ids absent from the local working tree
      status: open
      tdd_phase: red
    - id: AC-3
      title: Cross-branch list rows and check findings are unchanged before and after
      status: open
      tdd_phase: red
    - id: AC-4
      title: Zero DetectCollisions blob-stats when every id is present locally
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — One trunk helper composes the cross-branch scan for treeload, list, and show

### AC-2 — DetectCollisions runs only for ids absent from the local working tree

### AC-3 — Cross-branch list rows and check findings are unchanged before and after

### AC-4 — Zero DetectCollisions blob-stats when every id is present locally

