---
name: aiwfx-start-epic
description: Activates an aiwf epic — runs the preflight checks (epic body complete, drafted-milestone present, kernel `aiwf check` clean), asks the operator to choose a worktree placement and branch shape, optionally opens an `aiwf authorize` delegation scope, and lands the sovereign `aiwf promote E-NN active` commit. Use when the user says "start E-NN", "activate the auth epic", or "let's begin work on E-03". The promote step requires a `human/` actor unless `--force --reason "..."` is used; commit and any agent delegation require explicit human approval.
---

# aiwfx-start-epic

Activates an epic. Activation is a sovereign moment — the kernel treats `aiwf promote E-NN active` as a human-only act per M-0095, and the skill makes the surrounding deliberation explicit: preflight checks against the epic's readiness, an explicit worktree-placement choice, an explicit branch-shape choice, and an optional principal-to-agent delegation hand-off.

## Principles

- **Activation is sovereign.** The kernel refuses `aiwf promote E-NN active` from a non-`human/` actor unless `--force --reason "..."` is used. The skill's promotion step runs as the human; an AI assistant orchestrating the conversation hands the verb off to the operator.
- **Preflight uses kernel signals.** Body completeness, drafted-milestone presence, and `aiwf check` cleanliness all surface through existing kernel rules (`entity-body-empty`, `epic-active-no-drafted-milestones`, the standard refusal-severity findings). The skill reads — it does not duplicate the rule.
- **Worktree placement and branch shape are deliberate choices.** Each is its own Q&A step. Defaults that hide either choice paper over a decision the operator should make consciously.
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

### 5. Worktree placement (Q&A)

Ask the operator where the work will live. The choice matters — each option has different tradeoffs for parallel work, IDE state, and `aiwf check` blast radius — so the skill surfaces it as a deliberate prompt rather than picking on the operator's behalf.

1. **No worktree, work on `main` directly.** Suitable for short, focused epics where milestone branches stay merged-back-quickly. The operator commits straight on `main` (or the epic branch if one is created in step 6); no extra checkout state to manage. Trade-off: no isolated playground if the epic gets contentious.
2. **`.claude/worktrees/<branch>/` (in-repo worktree).** A worktree under the repo's own `.claude/` tree. Survives `git checkout` on the main worktree; gitignored. Trade-off: lives inside the repo path so editor sessions rooted at the repo see it.
3. **`../aiwf-<branch>/` (sibling-directory worktree).** A worktree as a sibling of the repo root. Fully isolated path; editor sessions rooted at the sibling have a clean view. Trade-off: requires a deliberate `cd` to enter, and `find`-based tools rooted at the original repo do not see it.

Record the operator's choice. If they choose a worktree (options 2 or 3), the skill creates it as part of step 6 (branch creation), since the worktree and the branch land together.

### 6. Branch shape (Q&A)

Ask the operator which branch the work lands on. This is **deliberately a prompt, not a default** — G-0059 frames the open question of which branch-model convention aiwf should bless (per-epic integration branch / per-milestone branch / direct-to-`main` / something else), and the answer has not landed yet. Until G-0059 resolves, the skill surfaces the choice rather than presuming.

1. **Stay on the current branch.** Suitable if the current branch is the right landing point (e.g. you're already on `main` for a small epic, or already on an `epic/<slug>` branch from a prior session).
2. **Create branch `<name>` and switch to it.** The operator names the branch. Common shapes today are `epic/E-NN-<slug>` (integration branch covering all milestones) or `wf/<task>` (per-task) — neither is canonical pending G-0059. If step 5 chose a worktree, the branch is created in that worktree's path; otherwise it's a plain `git checkout -b`.

Record the operator's choice and execute the branch creation if option 2 was picked. The branch operation does not produce an aiwf commit; it is plain git plumbing.

When G-0059 resolves, this step's default can tighten and the prompt can shrink (or disappear). Until then the explicit Q&A is the right shape.

### 7. Delegation prompt (Q&A)

Ask the operator whether the work proceeds in-loop (the operator drives every milestone) or delegated (an `aiwf authorize` scope is opened to a named `ai/<id>` agent).

- **In-loop** — no scope opened. Step 9 is skipped.
- **Delegate to `ai/<id>`** — step 9 runs `aiwf authorize E-NN --to ai/<id>`. The operator names the agent.

### 8. Sovereign promotion

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

This is **commit 1** — the verb writes exactly one commit with the standard `aiwf-verb: promote`, `aiwf-entity: E-NN`, `aiwf-actor: human/<id>` trailers.

### 9. Optional `aiwf authorize` (only if delegating)

If step 7 chose delegation:

```bash
aiwf authorize E-NN --to ai/<id>
```

This is a *separate* commit from step 8. The scope is `active` from this commit forward; the agent operates within it until the epic reaches a terminal status or the operator pauses the scope.

If step 7 chose in-loop, skip.

### 10. Hand-off

The epic is now `active`. The natural next step is `aiwfx-start-milestone <first-M>` (typically the lowest-numbered `draft` milestone under this epic).

If a delegation scope was opened in step 9, the hand-off is to the named agent (the subagent-spawn mechanics are Claude Code surface, outside this skill's scope). The operator names the receiving agent and transmits the milestone id; the agent then enters `aiwfx-start-milestone` itself.

## Constraints

- 🛑 **Never commit or push without explicit human approval.** Step 8's promotion and step 9's authorize each require human confirmation.
- 🛑 **Sovereign promotion requires a `human/` actor.** Per M-0095, `aiwf promote E-NN active` from a non-human actor is refused unless `--force --reason "..."` is used. An AI assistant orchestrating the conversation does not run the verb itself.
- The promotion commit and any authorize commit are separate. One verb = one commit.
- Worktree placement and branch shape are deliberate Q&A choices, not defaults the skill picks on the operator's behalf.

## Anti-patterns

- *Skipping the drafted-milestone check.* The epic activates with nothing queued; the next thing that happens is friction.
- *Letting an AI assistant run `aiwf promote E-NN active` directly.* The kernel refuses; the override path (`--force --reason`) is for genuine sovereign-act-shaped exceptions, not for routing around the rule.
- *Bundling the promote and authorize commits.* One verb = one commit. A combined commit is two acts at one timestamp and breaks `aiwf history`.
- *Defaulting the worktree placement.* The choice matters; surfacing it as a prompt is the point.

## Next step

→ `aiwfx-start-milestone <M-NNN>` for the first drafted milestone in the epic.
