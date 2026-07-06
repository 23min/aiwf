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
      status: met
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

Tracked in frontmatter `acs[]` and detailed in the `### AC-1` … `### AC-5` sections
below. All five are doc-shaped: each pins a `SKILL.md`/CLAUDE.md worktree-creation
call site against the embedded ritual snapshot bytes, scoped to the relevant
section per CLAUDE.md "Substring assertions are not structural assertions."

### AC-1 — aiwfx-start-milestone's cut-branch step invokes aiwf worktree add

`aiwfx-start-milestone` step 5's "isolate this milestone in its own worktree"
alternative now names a concrete `aiwf worktree add milestone/M-NNNN-<slug>
--base epic/E-NNNN-<slug>` invocation in place of the prior vague "default it
to in-repo... read with aiwf doctor" prose — the verb creates the worktree and
materializes rituals atomically. The default case (reusing the parent epic's
worktree, no new worktree) is unchanged: it never invoked worktree creation at
all, so there is nothing to rewire there.

Evidence: `TestM0234_AC1_StartMilestoneCutStepInvokesWorktreeAdd` in
`internal/policies/m0234_worktree_add_rewire_test.go` — asserts the step-5
subsection names `aiwf worktree add` and cross-references the `aiwf-worktree`
skill.

### AC-2 — wf-patch's branch-creation step invokes aiwf worktree add

`wf-patch` step 2 previously named no concrete worktree-creation command at
all — only branch-naming convention. It now opens with `aiwf worktree add
patch/G-NNNN-<short-slug> --base main`, citing CLAUDE.md's "Default to a
worktree for any branch work" convention directly.

Evidence: `TestM0234_AC2_WfPatchBranchStepInvokesWorktreeAdd` — asserts the
step-2 subsection names `aiwf worktree add` and cross-references the CLAUDE.md
section.

### AC-3 — aiwfx-start-epic's worktree-placement step invokes aiwf worktree add

Resolves the E-0059 epic's open question: `aiwfx-start-epic` step 8 *does*
create a worktree directly — it is the epic's own worktree-placement/creation
step, and milestone branches reuse that same worktree by default. Step 8 now
names `aiwf worktree add epic/E-NN-<slug>` for the in-repo (default) and
sibling placements, and confirms materialization afterward with `aiwf doctor
--root <path>`; the no-new-worktree (main-checkout) placement keeps plain `git
checkout -b`, since there is no worktree to materialize into.

Evidence: `TestM0234_AC3_StartEpicWorktreeStepInvokesWorktreeAdd` — asserts the
step-8 subsection names `aiwf worktree add` and `aiwf doctor`, and that `git
checkout -b` is retained for the main-checkout placement.

### AC-4 — CLAUDE.md worktree sections cite aiwf worktree add instead of raw git

Both CLAUDE.md sections now cite the verb: "Default to a worktree for any
branch work" gained a concrete `aiwf worktree add <branch> --base <base>`
citation (previously policy prose with no command at all), and "Subagent
worktree isolation" step 1 replaced the literal raw `git worktree add <path>
-b <branch> <base>` with `aiwf worktree add <branch> [<path>] --base <base>`;
step 2 now verifies with `aiwf doctor --root <path>` (materialization) rather
than only `git worktree list` (worktree existence). Step 3 (pass the absolute
path into the subagent's prompt rather than relying on `cd`) is unchanged,
since the new verb has no more ability to change a subagent's cwd than the raw
command did.

Evidence: `TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd` — asserts
both CLAUDE.md sections name `aiwf worktree add`, that the Subagent section
names `aiwf doctor` and the absolute-path instruction, and that the raw `git
worktree add` two-command sequence is gone.

### AC-5 — Every rewritten SKILL.md/CLAUDE.md edit has a referencing structural test

All three rewritten `SKILL.md` files (`aiwfx-start-milestone`,
`aiwfx-start-epic`, `wf-patch`) are referenced by `internal/policies/*_test.go`
path constants (pre-existing fixture loaders plus this milestone's new test
file), satisfying the `skill-edit-structural-test-backstop` policy; the
CLAUDE.md prose change is covered by AC-4's test per this repo's "AC promotion
requires mechanical evidence" rule, which applies to doc-shaped ACs regardless
of the mechanical backstop's narrower embedded-rituals scope.

Evidence: AC-1 through AC-4's tests above, plus a clean `make coverage-gate`
run confirming the backstop policy raises no violation for this diff.

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
