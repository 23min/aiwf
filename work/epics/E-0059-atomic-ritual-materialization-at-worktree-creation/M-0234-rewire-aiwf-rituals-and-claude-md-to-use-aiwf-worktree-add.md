---
id: M-0234
title: Rewire aiwf rituals and CLAUDE.md to use aiwf worktree add
status: in_progress
parent: E-0059
depends_on:
    - M-0233
tdd: none
acs:
    - id: AC-1
      title: aiwfx-start-milestone's cut-branch step invokes aiwf worktree add
      status: open
    - id: AC-2
      title: wf-patch's branch-creation step invokes aiwf worktree add
      status: open
    - id: AC-3
      title: aiwfx-start-epic's worktree-placement step invokes aiwf worktree add
      status: open
    - id: AC-4
      title: CLAUDE.md worktree sections cite aiwf worktree add instead of raw git
      status: open
    - id: AC-5
      title: Every rewritten SKILL.md/CLAUDE.md edit has a referencing structural test
      status: open
---

## Goal

Replace every raw `git worktree add` call site aiwf's own rituals and CLAUDE.md
instruct with the `aiwf worktree add` verb (M-0233), so aiwf-initiated worktree
creation actually materializes rituals atomically in practice, not just as an
unused capability.

## Context

M-0233 lands the verb; this milestone wires it into the real call sites. Doc-shaped
work — no new Go logic beyond what M-0233 already delivers. Same pattern as
M-0190 (E-0046), which rewrote ritual worktree-placement text and pinned the
result with structural tests against the embedded `SKILL.md` bytes.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0234 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — `aiwfx-start-milestone`'s cut-branch step invokes `aiwf
  worktree add` instead of raw `git worktree add`; a structural test asserts the
  step-scoped section names the verb.
- **AC-2 candidate** — `wf-patch`'s worktree setup step does the same, structurally
  asserted.
- **AC-3 candidate** — Resolve the open question from the E-0059 epic spec: does
  `aiwfx-start-epic` create worktrees directly, or only via
  `aiwfx-start-milestone`'s cut-branch step? If it creates them directly, rewire
  its worktree-placement step (step 8) the same way and add the matching
  structural test; if not, record the finding in this milestone's Work log and
  skip the rewrite.
- **AC-4 candidate** — CLAUDE.md's "Default to a worktree for any branch work" and
  "Subagent worktree isolation" sections cite the new verb instead of the raw
  two-command sequence. The subagent-dispatch procedure explicitly still passes
  the absolute worktree path into the subagent's prompt rather than relying on
  `cd` — unchanged by this milestone, since the new verb has no more ability to
  change a subagent's cwd than the raw command did.
- **AC-5 candidate** — Each rewritten `SKILL.md` (and any CLAUDE.md prose change)
  lands with its own referencing structural test under `internal/policies/`, per
  the `skill-edit-structural-test-backstop` policy.

### AC-1 — aiwfx-start-milestone's cut-branch step invokes aiwf worktree add

### AC-2 — wf-patch's branch-creation step invokes aiwf worktree add

### AC-3 — aiwfx-start-epic's worktree-placement step invokes aiwf worktree add

### AC-4 — CLAUDE.md worktree sections cite aiwf worktree add instead of raw git

### AC-5 — Every rewritten SKILL.md/CLAUDE.md edit has a referencing structural test

## Constraints

- Doc-shaped ACs use structural assertions scoped to the named section (per
  CLAUDE.md "Substring assertions are not structural assertions"), not flat greps
  for the verb name.
- Rewritten `SKILL.md` bodies carry no real entity ids, filesystem paths, or
  inline lifecycle status (the shipped-surface rule).

## Out of scope

- The verb itself and its flags/tests (M-0233).
- The session-start detection backstop (M-0235).

## Dependencies

- M-0233 — the verb this milestone wires into the call sites. Cannot start before
  M-0233 lands.

## References

- G-0374 — the gap this epic closes.
- M-0190 (E-0046) — the structural-test precedent for rewriting ritual
  worktree-placement content; `skill-edit-structural-test-backstop` policy.
