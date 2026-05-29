---
name: aiwfx-start-milestone
description: Sets up and begins an aiwf milestone — preflight checks, branch setup, status promotion to in_progress, then iterative TDD via wf-tdd-cycle. Use when the user says "start milestone M-NNNN" or "implement M-NNNN" and a draft milestone spec exists. Commits and pushes require explicit human approval.
---

# aiwfx-start-milestone

Begins implementation of an existing milestone. Promotes status, sets up the branch, and hands off to `wf-tdd-cycle` for each acceptance criterion. AC progress lives in the milestone spec's frontmatter `acs[]` (kernel-validated via `aiwf check`); the v1 separate tracking-doc convention is gone.

## When to use

A milestone spec exists at `work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md` with status `draft`. The user says: *"start M-NNNN"*, *"implement the cache milestone"*, *"begin M-0007"*.

If the spec doesn't exist or isn't ready, use `aiwfx-plan-milestones` first.

## Workflow

### 1. Preflight

- Read the milestone spec. Confirm every AC is concrete and testable. If any AC is vague, stop and ask the user to refine before starting work.
- Read the parent epic's spec for context.
- Read prior milestone specs in the same epic if this milestone builds on them.
- Confirm the spec has its ACs landed via `aiwf add ac` (frontmatter `acs[]` populated, body `### AC-N — <title>` headings present). If the spec was hand-written and `acs[]` is empty, ask the user whether to add them now via:

  ```bash
  aiwf add ac M-NNNN --title "<observable behavior>"
  ```

  Each invocation appends one AC and scaffolds the body heading; `aiwf check` will surface drift between frontmatter and body if the two disagree.

- Confirm the milestone's `tdd:` policy is intentional. `tdd: required` makes the audit `met requires phase: done` an error (blocks pre-push); `tdd: advisory` makes it a warning; `tdd: none` or absent skips it. If the user wants TDD discipline tracked mechanically, set `tdd: required` in the spec's frontmatter before starting.
- Run the project's build. **Confirm green** before introducing any change.
- Run the project's tests. **Confirm green.**

If anything is red before you start, stop. Don't begin a milestone on a broken baseline.

### 2. Promote status to `in_progress`

```bash
aiwf promote M-NNNN in_progress
```

aiwf validates the transition (`draft → in_progress` is legal), rewrites frontmatter, produces one commit with `aiwf-verb: promote` trailers.

### 3. Branch setup

If the project uses an epic integration branch:

```bash
git checkout -b epic/E-NNNN-<slug> origin/main      # if missing
git push -u origin epic/E-NNNN-<slug>
```

Then the milestone branch:

```bash
git checkout -b milestone/M-NNNN-<slug>            # from epic branch if epics are integration-batched, otherwise from main
```

If the project lands milestones directly on `main` via PR (no epic-integration branch), skip the epic-branch step and create `milestone/M-NNNN-<slug>` from `main`.

### 4. Implementation — iterate via `wf-tdd-cycle`

AC progress lives inside the milestone spec itself (frontmatter `acs[]` plus body `## Work log` section). There is no separate tracking doc — `templates/milestone-spec.md` carries the full set of sections (Work log, Decisions made during implementation, Validation, Deferrals, Reviewer notes).

For each AC, in sequence:

- Invoke `wf-tdd-cycle` (red → green → refactor → done). When the milestone is `tdd: required`, `wf-tdd-cycle` drives `aiwf promote M-NNNN/AC-<N> --phase <p>` at each phase transition; the timeline shows up in `aiwf history M-NNNN/AC-<N>` automatically.
- After the cycle ends green and clean, advance the AC status:

  ```bash
  aiwf promote M-NNNN/AC-<N> met
  ```

  Under `tdd: required`, the kernel audit refuses `met` without `phase: done` — keep them in this order. The kernel records both events in `aiwf history`.

- Append a Work log entry to the milestone spec's `## Work log` section: `### AC-<N> — <short title>` followed by `<one-line outcome> · commit <SHA> · tests <N/M>`. Don't duplicate the phase timeline — `aiwf history M-NNNN/AC-<N>` is the authoritative record.

If a decision surfaces mid-implementation that wasn't pre-locked in the spec, invoke `aiwfx-record-decision` to capture it. Mirror the decision id under the spec's `## Decisions made during implementation` section.

If a piece of work surfaces that's deferred, open a gap (`aiwf add gap --title "..." --discovered-in M-NNNN`) and mirror the resulting `G-NNN` id under the spec's `## Deferrals` section.

### 5. Self-review before declaring complete

Run a self-review pass before invoking `aiwfx-wrap-milestone`:

- Re-read the milestone spec; confirm every AC has at least one passing test.
- Run `aiwf check` (or `aiwf show M-NNNN`); confirm zero error-severity findings on the milestone. The `acs-tdd-audit`, `milestone-done-incomplete-acs`, and `acs-shape` codes are the AC-related ones to watch for.
- Run the **branch-coverage audit** from `wf-tdd-cycle` — every reachable conditional branch in the diff has an explicit test. This is a hard rule.
- Run through the `wf-review-code` checklist mentally (correctness, edge cases, conventions, no unrelated changes).
- If the project has its own end-to-end smoke procedure, run it.

Fix anything you find before declaring done.

### 6. Hand off to wrap

When self-review is clean, declare:

> *"Implementation complete. <N> tests passing, build green, branch-coverage audit clean, self-review passed. Ready for `aiwfx-wrap-milestone`."*

Do not commit the implementation yet — `aiwfx-wrap-milestone` bundles the implementation, the wrap-side spec updates (Validation, Reviewer notes, Deferrals), and the milestone-status closure into a single approved sequence.

## Constraints

- 🛑 **Never commit or push without explicit human approval.** Every commit gate is the human's, not the AI's.
- 🛑 **Branch-coverage hard rule** (see `wf-tdd-cycle`). Audit runs before declaring complete, not after the human asks.
- Tests must be deterministic. No clock, no network, no flakes shipped.
- Build must be green before declaring done.
- Follow existing code conventions. Prefer minimal changes — don't refactor unrelated code along the way.

## Anti-patterns

- *Promoting to `in_progress` before preflight passes.* If the baseline is broken, fix it under a `wf-patch` first, then start the milestone.
- *Skipping the Work log section.* It's the audit trail of mid-flight context next to each AC's commits. Don't reconstruct it after the fact.
- *Hand-editing `acs[]` in frontmatter.* Use `aiwf add ac` / `aiwf promote M-NNNN/AC-<N>` / `aiwf rename M-NNNN/AC-<N>` instead — the verbs preserve position-stability and the body-coherence pairing.
- *Mixing milestones.* One milestone per branch. Don't fold "while I was here" work into the diff.
- *Skipping the branch-coverage audit.* "I'll catch it in review" doesn't catch it.

## Next step

→ `aiwfx-wrap-milestone M-NNNN` after self-review is clean.
