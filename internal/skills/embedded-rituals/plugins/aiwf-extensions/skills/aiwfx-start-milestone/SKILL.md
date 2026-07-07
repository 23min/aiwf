---
name: aiwfx-start-milestone
description: Sets up and begins an aiwf milestone — preflight checks, branch setup, status promotion to in_progress, then iterative TDD via wf-tdd-cycle. Use when the user says "start milestone M-NNNN" or "implement M-NNNN" and a draft milestone spec exists. Commits and pushes require explicit human approval.
---

# aiwfx-start-milestone

Begins implementation of an existing milestone. Per the branch-model sequencing rule, the state-announcement commits (promote at step 3, optional authorize at step 4) land on the parent epic branch BEFORE the milestone branch is cut at step 5. AC progress lives in the milestone spec's frontmatter `acs[]` (kernel-validated via `aiwf check`).

## When to use

A milestone spec exists at `work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md` with status `draft`, AND the parent epic is `active` with its `epic/E-NNNN-<slug>` branch existing locally and currently checked out. The user says: *"start M-NNNN"*, *"implement the cache milestone"*, *"begin the auth milestone"*.

If the spec doesn't exist or isn't ready, use `aiwfx-plan-milestones` first. If the parent epic isn't active or its branch doesn't exist locally, use `aiwfx-start-epic E-NNNN` first.

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
- **Parent epic branch must exist locally and be the operator's current checkout.** The state-announcement commits at steps 3 and 4 land on the parent epic branch BEFORE the milestone branch is cut at step 5. If the parent epic branch does not exist locally, the parent epic has not been activated yet — stop and run `aiwfx-start-epic E-NNNN` first; do NOT improvise by creating the branch here. If the parent epic branch exists but is not currently checked out, switch to it before continuing (`git checkout epic/E-NNNN-<slug>`).
- Run the project's build. **Confirm green** before introducing any change.
- Run the project's tests. **Confirm green.**

If anything is red before you start, stop. Don't begin a milestone on a broken baseline.

### 2. Delegation prompt (Q&A)

Ask the operator whether the milestone proceeds in-loop (the operator drives every AC) or delegated (an `aiwf authorize` scope is opened to a named `ai/<id>` agent for the milestone). The answer determines whether step 4 runs.

- **In-loop** — no scope opened. Step 4 is skipped.
- **Delegate to `ai/<id>`** — step 4 runs `aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>`. The operator names the agent.

The delegation choice is asked BEFORE the sovereign acts (steps 3 and 4) because the authorize trailer (if delegating) binds the scope to the future milestone branch, and the milestone-branch name should be known when the authorize commit lands on the parent epic branch. The authorize commit's `aiwf-branch:` trailer is a forward-binding — the named branch is cut at step 5.

The milestone scope is independent of any epic scope opened at `aiwfx-start-epic` step 7. Kernel semantics: one scope per entity; the milestone's scope is opened, paused, and resumed on its own entity, with its own `aiwf-branch:` (the milestone branch).

### 3. Sovereign promote on parent epic branch

Confirm the operator is on the parent epic branch (per step 1's preflight). The promote lands here:

```bash
aiwf promote M-NNNN in_progress
```

aiwf validates the transition (`draft → in_progress` is legal), rewrites frontmatter, produces one commit on the parent epic branch with `aiwf-verb: promote` trailers.

If the promote needs to land via a sovereign-act override (e.g. a recovery flow after a half-applied prior promote, or a bot account ratification), the override path is:

```bash
aiwf promote M-NNNN in_progress --force --reason "<one-sentence justification>"
```

The standard provenance-coherence rule still requires the `--force` invocation itself to come from a `human/` actor, so the override remains human-sovereign by construction. Use sparingly; the default path is the right one.

This is **commit 1** of the start ritual — landed on the parent epic branch.

### 4. Sovereign authorize on parent epic branch (only if delegating)

If step 2 chose delegation, the operator runs (still on the parent epic branch):

```bash
aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug> --reason "<one-sentence rationale>"
```

The `--branch` flag names the *future* milestone branch — the one step 5 will cut. The branch does not yet exist when this verb runs. The kernel's AI-target preflight permits this combination via the ritual-current carve-out: from a ritual-shape current checkout (here `epic/E-NNNN-<slug>` satisfies that), an explicit `--branch` whose value matches the ritual shape (`milestone/` / `patch/`) accepts even when the named branch does not yet exist. The commit's `aiwf-branch:` trailer carries the future milestone ref; step 5's branch cut closes the binding.

This is a *separate* commit from step 3, landed on the same parent epic branch. The scope is `active` from this commit forward; the agent operates within it until the milestone reaches a terminal status or the operator pauses the scope.

If the operator is NOT on the parent epic branch when this step runs (e.g. they jumped to a feature branch first), the preflight classifies the current checkout's rung against the `--branch` target's rung; a pair that isn't a legal ritual flow (here `epic → milestone`) refuses with `rung-pair-illegal`, naming both branches' rungs. (Omitting `--branch` from a non-ritual checkout instead refuses with `branch-context-required`.) The override path is the same sovereign-act shape:

```bash
aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug> --force --reason "<one-sentence justification>"
```

The `--force` invocation requires a `human/` actor. Use sparingly; the default path is right.

If step 2 chose in-loop, skip.

### 5. Cut the milestone branch

Now that the state-announcement commits (steps 3 and optionally 4) have landed on the parent epic branch, cut the milestone work branch off it:

```bash
git checkout -b milestone/M-NNNN-<slug>
```

The branch operation does not produce an aiwf commit; it is plain git plumbing. If a delegated `aiwf authorize` commit was produced at step 4, the named branch now resolves and the binding closes — the trailer's forward-reference becomes a live ref.

**Worktree placement.** By default the milestone branch is cut in the parent epic's worktree, which is already in-repo under the configured `worktree.dir` (default `.claude/worktrees/`) when the epic was activated via `aiwfx-start-epic`'s default. In-repo is the default because a Claude Code session in a sandboxed devcontainer is confined to the workspace folder — a sibling or `$HOME` worktree is unreachable as the session's cwd and a `$HOME`-placed one is wiped on container rebuild.

If you instead isolate this milestone in its own worktree (e.g. for parallel milestone work), use `aiwf worktree add` in place of the plain `git checkout -b` above — it creates the linked worktree and materializes rituals (skills, agents, templates, guidance) into it atomically, in one step, in-repo under the same `worktree.dir` by default:

```bash
aiwf worktree add milestone/M-NNNN-<slug> --base epic/E-NNNN-<slug>
```

Pass an explicit path as the verb's second argument for a sibling-directory placement instead. The per-invocation override (main-checkout / sibling) stays available; in-repo is the recommendation, not a lock. See the `aiwf-worktree` skill for the full verb reference.

### 6. Implementation — iterate via `wf-tdd-cycle`

AC progress lives inside the milestone spec itself (frontmatter `acs[]` plus body `## Work log` section); `templates/milestone-spec.md` carries the full set of sections (Work log, Decisions made during implementation, Validation, Deferrals, Reviewer notes).

For each AC, in sequence:

- Invoke `wf-tdd-cycle` (red → green → refactor → done). When the milestone is `tdd: required`, `wf-tdd-cycle` drives `aiwf promote M-NNNN/AC-<N> --phase <p>` at each phase transition — **live, the moment each transition happens, never deferred or bursted at wrap.** The timeline in `aiwf history M-NNNN/AC-<N>` is the evidence the test came before the code; a phase ladder stamped in a batch later is indistinguishable from one back-stamped after the fact.
- After the cycle ends green and clean, advance the AC status:

  ```bash
  aiwf promote M-NNNN/AC-<N> met
  ```

  Under `tdd: required`, the kernel audit refuses `met` without `phase: done` — keep them in this order. The kernel records both events in `aiwf history`.

- 🛑 **Commit the AC's implementation code now** — the changed source and test files, on the milestone branch — before starting the next AC. This is a real commit, not deferred to wrap: `feat(<scope>): <AC summary> (M-NNNN/AC-<N>)`. Every commit is the human's gate; wait for explicit approval. The resulting SHA is what the Work log entry below cites.
- Append a Work log entry to the milestone spec's `## Work log` section: `### AC-<N> — <short title>` followed by `<one-line outcome> · commit <SHA> · tests <N/M>`. Don't duplicate the phase timeline — `aiwf history M-NNNN/AC-<N>` is the authoritative record.
- At this AC boundary, if the user asks for a handoff or context is getting long before the next AC, invoke `aiwfx-handoff` to emit a paste-ready `/compact` prime block. Emission here is on-demand — every-AC is noise.

If a decision surfaces mid-implementation that wasn't pre-locked in the spec, invoke `aiwfx-record-decision` to capture it. Mirror the decision id under the spec's `## Decisions made during implementation` section.

If a piece of work surfaces that's deferred, open a gap (`aiwf add gap --title "..." --discovered-in M-NNNN`) and mirror the resulting `G-NNN` id under the spec's `## Deferrals` section.

### 7. Readiness check before handoff

Before invoking `aiwfx-wrap-milestone`, confirm the change is *ready to be reviewed* — not that it *has been* reviewed. These are gates you clear yourself so a broken or noisy diff never reaches an independent reviewer; none of them is the review:

- Re-read the milestone spec; confirm every AC has at least one passing test.
- Run `aiwf check` (or `aiwf show M-NNNN`); confirm zero error-severity findings on the milestone. The `acs-tdd-audit`, `milestone-done-incomplete-acs`, and `acs-shape` codes are the AC-related ones to watch for.
- Run the **branch-coverage audit** from `wf-tdd-cycle` — every reachable conditional branch in the diff has an explicit test. This is a hard rule.
- Tidy the diff: remove debug output, unrelated changes, and stale comments so the reviewer's attention lands on substance, not lint. Reading your own code for correctness catches little — which is exactly why the review that follows is *independent*, not another pass by you.
- If the project has its own end-to-end smoke procedure, run it.

Fix anything you find before handing off. **The review itself runs at `aiwfx-wrap-milestone`: an independent, fresh-context two-lens pass — code-quality (`wf-review-code`) and design-quality (`wf-rethink`) — dispatched before the milestone closes. It is the authoritative check; this readiness pass never stands in for it.**

### 8. Hand off to wrap

When the readiness checks are clean, declare:

> *"Implementation complete. <N> tests passing, build green, branch-coverage audit clean, diff tidied. Ready for `aiwfx-wrap-milestone` — which runs the independent review before closing."*

The implementation is already committed, per-AC, from step 6 — there is nothing left to bundle. `aiwfx-wrap-milestone` dispatches the independent two-lens review, then commits only the wrap-side spec updates (Work log, Validation, Reviewer notes, Deferrals) and closes the milestone via its own declared-sequence gate.

## Constraints

- 🛑 **Never commit or push without explicit human approval.** Every commit gate is the human's, not the AI's.
- 🛑 **Branch-coverage hard rule** (see `wf-tdd-cycle`). Audit runs before declaring complete, not after the human asks.
- 🛑 **Sovereign acts land on the parent epic branch before the milestone-branch cut.** Steps 3 and 4 run with HEAD on `epic/E-NNNN-<slug>`; step 5 cuts `milestone/M-NNNN-<slug>` afterwards. The kernel's preflight enforces this for the authorize commit (the ritual-current carve-out allows the `--branch milestone/...` future-binding from a ritual-shape current checkout).
- 🛑 **Parent epic branch must exist locally and be the current checkout before this skill runs.** If it doesn't exist, the parent epic has not been activated — `aiwfx-start-epic E-NNNN` is the right entry point, not this skill. No silent fallthrough that materializes the parent branch on the operator's behalf.
- Tests must be deterministic. No clock, no network, no flakes shipped.
- Build must be green before declaring done.
- Follow existing code conventions. Prefer minimal changes — don't refactor unrelated code along the way.

## Anti-patterns

- *Promoting to `in_progress` before preflight passes.* If the baseline is broken, fix it under a `wf-patch` first, then start the milestone.
- *Improvising the parent epic branch when it doesn't exist.* The previous version of this skill silently fell through to `git checkout -b epic/E-NNNN-<slug> origin/main # if missing`. That masks the precondition failure (the parent epic wasn't activated) and produces a parent branch with no `aiwf promote E-NNNN active` commit on it. Stop and run `aiwfx-start-epic` instead.
- *Bundling the promote and authorize commits.* One verb = one commit. The promote (step 3) and authorize (step 4) each land on the parent epic branch in their own commit.
- *Cutting the milestone branch before the sovereign acts.* The kernel's preflight refuses authorize-on-milestone-branch with `branch-context-required` at the verb layer; the `isolation-escape` kernel finding catches the same shape post-hoc at `aiwf check` (warning severity). Branch cut belongs at step 5, after the trailers have landed on the parent.
- *Skipping the Work log section.* It's the audit trail of mid-flight context next to each AC's commits. Don't reconstruct it after the fact.
- *Hand-editing `acs[]` in frontmatter.* Use `aiwf add ac` / `aiwf promote M-NNNN/AC-<N>` / `aiwf rename M-NNNN/AC-<N>` instead — the verbs preserve position-stability and the body-coherence pairing.
- *Mixing milestones.* One milestone per branch. Don't fold "while I was here" work into the diff.
- *Skipping the branch-coverage audit.* "I'll catch it in review" doesn't catch it.

## Next step

→ `aiwfx-wrap-milestone M-NNNN` after the readiness check is clean.
