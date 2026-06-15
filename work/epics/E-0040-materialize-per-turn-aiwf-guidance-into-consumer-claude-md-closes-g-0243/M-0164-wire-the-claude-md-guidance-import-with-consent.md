---
id: M-0164
title: Wire the CLAUDE.md guidance import with consent
status: draft
parent: E-0040
depends_on:
    - M-0163
tdd: required
acs:
    - id: AC-1
      title: init and update wire the import by default; --no-wire-claudemd opts out
      status: open
      tdd_phase: red
    - id: AC-2
      title: Content outside the markers is preserved; CLAUDE.md is created if absent
      status: open
      tdd_phase: red
    - id: AC-3
      title: Re-running is idempotent; a removed import line is reported, not re-added
      status: open
      tdd_phase: red
    - id: AC-4
      title: A printed notice announces the CLAUDE.md edit and names the opt-out
      status: open
      tdd_phase: red
    - id: AC-5
      title: The inserted import line resolves to the materialized guidance file
      status: open
      tdd_phase: red
    - id: AC-6
      title: A damaged marker block is handled per the hook-marker policy
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — init and update wire the import by default; --no-wire-claudemd opts out

### AC-2 — Content outside the markers is preserved; CLAUDE.md is created if absent

### AC-3 — Re-running is idempotent; a removed import line is reported, not re-added

### AC-4 — A printed notice announces the CLAUDE.md edit and names the opt-out

### AC-5 — The inserted import line resolves to the materialized guidance file

### AC-6 — A damaged marker block is handled per the hook-marker policy

