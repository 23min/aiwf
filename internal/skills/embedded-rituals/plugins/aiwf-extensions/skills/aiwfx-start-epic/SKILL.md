---
name: aiwfx-start-epic
description: Activates an aiwf epic — runs the preflight checks (epic body complete, drafted-milestone present, kernel `aiwf check` clean), asks the operator to choose a worktree placement and branch shape, optionally opens an `aiwf authorize` delegation scope, and lands the sovereign `aiwf promote E-NN active` commit. Use when the user says "start E-NN", "activate the auth epic", or "let's begin work on E-03". The promote step requires a `human/` actor unless `--force --reason "..."` is used; commit and any agent delegation require explicit human approval.
---

# aiwfx-start-epic

Activates an epic. Activation is a sovereign moment — the kernel treats `aiwf promote E-NN active` as a human-only act per M-0095, and the skill makes the surrounding deliberation explicit: preflight checks against the epic's readiness, an explicit worktree-placement choice, an explicit branch-shape choice, and an optional principal-to-agent delegation hand-off.

## Principles

- **Activation is sovereign.** The kernel refuses `aiwf promote E-NN active` from a non-`human/` actor unless `--force --reason "..."` is used. The skill's promotion step runs as the human; an AI assistant orchestrating the conversation hands the verb off to the operator.
- **Sovereign acts on `main`; branch cut afterwards.** Per [ADR-0010](../../../../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md), state-announcement commits (the promote at step 6 and, if delegating, the authorize at step 7) land on `main` BEFORE the epic branch is cut at step 8. The chokepoint behind this sequencing is M-0103's AI-target preflight on `aiwf authorize` — without ritual branch context the preflight refuses. The M-0104/AC-4 carve-out makes the `--branch epic/E-NN-<slug>` future-binding from `main` accept (the named branch is cut at step 8).
- **Preflight uses kernel signals.** Body completeness, drafted-milestone presence, and `aiwf check` cleanliness all surface through existing kernel rules (`entity-body-empty`, `epic-active-no-drafted-milestones`, the standard refusal-severity findings). The skill reads — it does not duplicate the rule.
- **Worktree placement is a deliberate choice.** Each option has different tradeoffs for parallel work, IDE state, and `aiwf check` blast radius. The skill surfaces it as a prompt rather than picking on the operator's behalf.
- **The promotion commit and any authorize commit are separate.** One verb = one commit. The skill orchestrates both in sequence; it never bundles them.

## Precondition

1. The epic spec exists at `work/epics/E-NN-<slug>/epic.md` with status `proposed`.
2. At least one milestone under the epic has status `draft` (the kernel's `epic-active-no-drafted-milestones` warning fires otherwise; the skill's step 2 surfaces it).
3. Working tree clean.

If any precondition fails, stop and report. Do not improvise around a half-planned epic.

## Workflow

### 1. Preflight: read the epic spec

Open `work/epics/E-NN-<slug>/epic.md`. Confirm the Goal, Scope (in / out), and Constraints sections are concrete prose, not template placeholders. The kernel's `entity-body-empty` finding catches the worst case (all-template body); this step catches the in-between case (body present but vague).

If any section is template-shaped, stop and return the operator to `aiwfx-plan-epic` to flesh it out.

### 2. Drafted-milestone check

Run `aiwf check` and look for the `epic-active-no-drafted-milestones` warning targeting this epic. If it fires, the epic has no `draft`-status milestone yet — the skill cannot proceed because there is nothing queued to start.

If it fires, hand the operator to `aiwfx-plan-milestones E-NN` to allocate at least one milestone, then re-enter `aiwfx-start-epic`.

### 3. `aiwf check` clean of refusal-level findings

The drafted-milestone check (step 2) is a warning; this step is the broader pass. Run `aiwf check` and confirm no error-severity findings touch this epic, its milestones, or files the operator is about to commit.

If error-severity findings exist, the skill stops. Resolve them before activation.

### 4. Project tests/build advisory pass

Run the project's tests and build. This step is advisory — a red baseline does not block activation, but the operator should know the state before committing to the work.

Report the result. If red, ask the operator whether to proceed or to fix the baseline first.

### 5. Delegation prompt (Q&A)

Ask the operator whether the work proceeds in-loop (the operator drives every milestone) or delegated (an `aiwf authorize` scope is opened to a named `ai/<id>` agent). The answer determines whether step 7 runs.

- **In-loop** — no scope opened. Step 7 is skipped.
- **Delegate to `ai/<id>`** — step 7 runs `aiwf authorize E-NN --to ai/<id> --branch epic/E-NN-<slug>`. The operator names the agent and the future epic branch (typically `epic/E-NN-<slug>` derived from the epic id and slug).

The delegation choice is asked BEFORE the sovereign acts because the authorize trailer (if delegating) binds the scope to a named branch, and the epic-branch name should be known when the authorize commit lands on `main`. Per ADR-0010, the authorize commit's `aiwf-branch:` trailer is a forward-binding — the named branch is cut at step 8.

### 6. Sovereign promotion

Confirm with the operator that the epic is on `main` (or the parent branch the sovereign acts will land on). Per ADR-0010, both this step and step 7 (if delegating) run with the operator's HEAD on `main` — the epic branch is cut afterwards at step 8.

Activation is the sovereign moment. The operator runs:

```bash
aiwf promote E-NN active
```

The kernel refuses this verb from a non-`human/` actor — per M-0095, an `ai/<id>` operator attempting to flip the epic gets a typed error pointing at the rule and the override path. The operator is human; an AI assistant orchestrating the conversation does not invoke the verb itself.

The override path exists for genuine sovereign-act-shaped exceptions (a ratification run by a bot account, a recovery flow after a half-applied prior promote):

```bash
aiwf promote E-NN active --force --reason "<one-sentence justification>"
```

The standard provenance-coherence rule still requires the `--force` invocation itself to come from a `human/` actor, so the override remains human-sovereign by construction. Use it sparingly; the default path is the right one.

This is **commit 1** — the verb writes exactly one commit on `main` with the standard `aiwf-verb: promote`, `aiwf-entity: E-NN`, `aiwf-actor: human/<id>` trailers.

### 7. Sovereign authorize (only if delegating)

If step 5 chose delegation, the operator runs (still on `main`):

```bash
aiwf authorize E-NN --to ai/<id> --branch epic/E-NN-<slug> --reason "<one-sentence rationale>"
```

The `--branch` flag names the *future* epic branch — the one step 8 will cut. The branch does not yet exist when this verb runs. The M-0103 AI-target preflight permits this combination via the M-0104/AC-4 carve-out: from a checkout on `main`, an explicit `--branch` whose value matches the ritual shape (`epic/`/`milestone/`/`patch/` per `internal/branchparse/`) accepts even when the named branch does not yet exist. The commit's `aiwf-branch:` trailer carries the future ref; step 8's branch cut closes the binding.

This is a *separate* commit from step 6. The scope is `active` from this commit forward; the agent operates within it until the epic reaches a terminal status or the operator pauses the scope.

If the operator is NOT on `main` when this step runs (e.g. they jumped to a feature branch first), M-0103's preflight refuses with `branch-context-required` or `branch-not-found`. The override path is the same sovereign-act shape:

```bash
aiwf authorize E-NN --to ai/<id> --branch epic/E-NN-<slug> --force --reason "<one-sentence justification>"
```

The `--force` invocation requires a `human/` actor, so the override remains human-sovereign by construction. The default path (operator on `main`, no `--force`) is the right one.

If step 5 chose in-loop, skip.

### 8. Worktree placement and branch creation (Q&A)

Ask the operator where the work will live. The choice matters — each option has different tradeoffs for parallel work, IDE state, and `aiwf check` blast radius — so the skill surfaces it as a deliberate prompt rather than picking on the operator's behalf.

1. **No worktree, work directly on the epic branch in the main checkout.** The operator's existing checkout switches to `epic/E-NN-<slug>` via `git checkout -b`. Simplest; no extra checkout state to manage. Trade-off: no isolated playground if the epic gets contentious.
2. **`.claude/worktrees/<branch>/` (in-repo worktree).** A worktree under the repo's own `.claude/` tree. Survives `git checkout` on the main worktree; gitignored. Trade-off: lives inside the repo path so editor sessions rooted at the repo see it.
3. **`../aiwf-<branch>/` (sibling-directory worktree).** A worktree as a sibling of the repo root. Fully isolated path; editor sessions rooted at the sibling have a clean view. Trade-off: requires a deliberate `cd` to enter, and `find`-based tools rooted at the original repo do not see it.

The branch shape is settled by ADR-0010: ritualized work on `epic/E-NN-<slug>`. If step 7's authorize commit was produced (delegated case), the branch name is already in the trailer — this step cuts that exact ref. If step 5 chose in-loop, the operator still cuts `epic/E-NN-<slug>` (the same naming convention; no `aiwf-branch:` trailer was emitted upstream, but the convention is the same).

Execute the branch cut against the chosen worktree (or in the main checkout for option 1). The branch operation does not produce an aiwf commit; it is plain git plumbing.

### 9. Hand-off

The epic is now `active`, the branch is cut, and the operator's HEAD is on `epic/E-NN-<slug>` (in the chosen worktree). The natural next step is `aiwfx-start-milestone <first-M>` (typically the lowest-numbered `draft` milestone under this epic).

If a delegation scope was opened in step 7, the hand-off is to the named agent (the subagent-spawn mechanics are Claude Code surface, outside this skill's scope). The operator names the receiving agent and transmits the milestone id; the agent then enters `aiwfx-start-milestone` itself.

## Constraints

- 🛑 **Never commit or push without explicit human approval.** Step 6's promotion and step 7's authorize each require human confirmation.
- 🛑 **Sovereign promotion requires a `human/` actor.** Per M-0095, `aiwf promote E-NN active` from a non-human actor is refused unless `--force --reason "..."` is used. An AI assistant orchestrating the conversation does not run the verb itself.
- 🛑 **Sovereign acts land on `main` before the branch cut.** Per ADR-0010, steps 6 and 7 run with HEAD on `main`; step 8 cuts the epic branch afterwards. The M-0103 preflight enforces this for the authorize commit (the M-0104/AC-4 carve-out allows the `--branch <future>` form from `main`).
- The promotion commit and any authorize commit are separate. One verb = one commit.
- Worktree placement is a deliberate Q&A choice, not a default the skill picks on the operator's behalf. The branch shape is settled by ADR-0010 — `epic/E-NN-<slug>` — and is not surfaced as a prompt.

## Anti-patterns

- *Skipping the drafted-milestone check.* The epic activates with nothing queued; the next thing that happens is friction.
- *Letting an AI assistant run `aiwf promote E-NN active` directly.* The kernel refuses; the override path (`--force --reason`) is for genuine sovereign-act-shaped exceptions, not for routing around the rule.
- *Bundling the promote and authorize commits.* One verb = one commit. A combined commit is two acts at one timestamp and breaks `aiwf history`.
- *Defaulting the worktree placement.* The choice matters; surfacing it as a prompt is the point.

## Next step

→ `aiwfx-start-milestone <M-NNN>` for the first drafted milestone in the epic.
