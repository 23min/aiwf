---
id: M-0204
title: 'Model 1: commit implementation per AC; live phase promotes'
status: draft
parent: E-0049
depends_on:
    - M-0203
tdd: advisory
---

## Goal

Model 1 commit model: milestone implementation commits land per acceptance
criterion on the milestone branch, and `tdd: required` phase promotes fire
live during the red→green→done cycle — never bundled into a single burst at
wrap. Closes [G-0293](../../gaps/archive/G-0293-promote-tdd-phase-live-not-in-a-burst-at-milestone-wrap.md).

## Context

This milestone's scope landed ahead of its own lifecycle, via a standalone
`wf-patch` rather than a milestone TDD cycle: `patch/G-0293-live-phase-promotes`
(commit `76829a69`, "commit milestone implementation per-AC, not bundled at
wrap"), with the streamlined cadence shipped as the guidance default at commit
`bc6b27d0` ("docs(guidance): ship streamlined TDD phase-promote cadence as the
default"). G-0293 is `addressed`. The current `aiwfx-start-milestone` embedded
skill (step 6) already commits per-AC and fires phase promotes live — verified
against the live SKILL.md content.

## Scope

No further work identified under this milestone; the patch delivered the full
scope described in E-0049's epic body item 1. Retained as a record of the
epic's original milestone plan. Whether to promote this milestone `done`
(referencing the patch commit as evidence) or `cancel` it as superseded
elsewhere — mirroring M-0203's disposition — is an open follow-up decision,
not resolved by this edit.

## Acceptance criteria

None drafted. The work landed via `patch/G-0293-live-phase-promotes` outside
this milestone's TDD cycle, so there is no in-milestone AC ladder to retrofit
after the fact.
