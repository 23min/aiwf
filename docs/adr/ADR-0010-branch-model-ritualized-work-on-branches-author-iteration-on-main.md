---
id: ADR-0010
title: 'Branch model: ritualized work on branches, author iteration on main'
status: accepted
---

# ADR-0010 — Branch model: ritualized work on branches, author iteration on main

> **Date:** 2026-05-11 · **Decided by:** Peter Bruinsma (human/peter)

## Context

The repo has carried two not-quite-reconciled stances on git branching since the PoC promoted to main:

- `CLAUDE.md` § "Working in this repo" (tightened today via G-0076) reads *"Trunk-based development on `main` for maintainers. Commit directly to trunk; no PR ceremony."* That's true for the author's day-to-day typing, but it doesn't account for delegated multi-commit work.
- Real practice has drifted toward branch isolation for substantial work: the parallel session driving E-0029 lives on `epic/E-0029-glanceable-render`; `aiwfx-start-epic` and `aiwfx-start-milestone` rituals already create epic and milestone branches; today's three patches (G-0069, G-0076, G-0079) used `fix/...` / `chore/...` branches per the `wf-patch` ritual.

The kernel commits to *"framework correctness must not depend on the LLM's behavior"* — but the inverse is also load-bearing: the human author is sovereign per the principal × agent × scope model and must be able to iterate without ceremony.

**Forces driving the decision:**

- **AI delegation amplifies the cost of un-isolated multi-commit work.** G-0059 cited M-0069 — 23 commits landing on `poc/aiwf-v3` with no nudge to isolate. Bisecting a regression across that span, reverting a single AC, or pausing mid-milestone is materially harder than on an isolated branch with a single merge point.
- **The harness's `isolation: "worktree"` kwarg is unreliable** (G-0099, partial-closed today by a PreToolUse hook). Worktrees imply branches; the convention needs to be codified for the kernel-side `isolation-escape` finding (E-0019) to enforce against something.
- **State-visibility matters for multi-session work.** Today's other session promoted E-0029 to active and opened the autonomous scope on a feature branch (G-0116); from main, the state was invisible. Operators running `aiwf status` from main saw the epic still proposed even while it was actively under delegation.
- **The PoC's "permissive ethos"** for solo human work is genuinely valuable — author iteration speed is part of why aiwf-the-tool exists. A hard PR-style flow would defeat it.

**Alternatives considered:**

1. **Status quo (pure trunk-based for maintainers).** Rejected — fails for AI delegation. M-0069 evidence already accumulated; G-0059 named the cost.
2. **Pure branch-based / PR-style for everything.** Rejected — defeats author iteration speed; runs into the principal-sovereign principle (the human can always override `--force`).
3. **Hybrid by who-does-it: solo human → main, delegated AI → branch.** Rejected — wrong axis. A solo human invoking `wf-patch` *should* still go on a branch (that's what the ritual is for), and an AI agent making a single state-announcement commit *can* land on main (single mutation, no scope, no isolation needed).
4. **Hybrid by ritual (adopted).** The axis that matches actual practice — rituals create branch context; without ritual context, you're on main.

## Decision

We adopt a **two-tier branch model with author override**:

### Tier 1 — Main (default surface)

The following land on `main` by default, when no ritual branch context is active (Tier 2's umbrella rule supersedes when one is):

- **Initial entity creation:** `aiwf add epic`, `aiwf add milestone`, `aiwf add gap`, `aiwf add decision`, `aiwf add contract`, `aiwf add adr`, `aiwf add ac`.
- **State-announcement transitions that *open* a scope:** `aiwf promote E-NN proposed → active`, `aiwf authorize E-NN --to ai/<agent>`. These commits announce to every operator and every parallel session that the project's state has changed; they must be visible from main *before* any ritual branch is cut against the entity.
- **Author iteration:** the human author can commit anything on `main` — typing, focused fixes, one-off mutations they choose not to ritualize. The author's discretion governs per-case whether to elevate a change to a ritual (`wf-patch`) or land it directly. This is the sovereign-override surface and is self-policed (no mechanical detection of *"this should have been a branch"*).

### Tier 2 — Ritual branches (planned multi-commit work)

The following ritual surfaces create named branches; their work lands on those branches:

| Ritual | Branch shape | Lifecycle |
|---|---|---|
| `aiwfx-start-epic` | `epic/E-NN-<slug>` | Integration branch; receives milestone merges; merges into main at epic wrap |
| `aiwfx-start-milestone` | `milestone/M-NNN-<slug>` | Work branch; merges into its parent epic branch on milestone done |
| `wf-patch` | `fix/...`, `patch/...`, `doc/...`, `chore/...` | Single-focused-change branch; merges into main when the patch lands |

**Once on a ritual branch, all mutations related to that work go there too** — status changes (e.g., `aiwf promote M-NNN draft → in_progress`), AC adds, body edits, gaps discovered during the work, scope `--pause` / `--resume`. The merge-to-main is the visibility event. The trade-off is deliberate: in-flight work isn't broadcast to other sessions until it's a coherent unit, which is the right shape for delegated multi-commit cycles.

### Sequencing rule for opening an epic

The state-announcement commits **must precede** the branch cut, not follow it:

1. `aiwf promote E-NN proposed → active` → commits to `main`
2. *(optional)* `aiwf authorize E-NN --to ai/<agent>` → commits to `main`
3. The ritual (`aiwfx-start-epic`) **then** cuts `epic/E-NN-<slug>` off main
4. All subsequent epic-level work lands on the epic branch

The symmetric rule applies to milestones: `aiwf promote M-NNN draft → in_progress` lands on the parent epic branch (which already exists), then `aiwfx-start-milestone` cuts the milestone work branch off the epic branch.

This sequencing is exactly what G-0116 names as broken in today's `aiwfx-start-epic` (the skill cuts the worktree at step 5 and promotes at step 8 — backwards). G-0116 is unblocked by this ADR.

### AI chokepoint

AI-actor multi-commit work **requires** a ritual branch context. The kernel will enforce this at the `aiwf authorize` surface: opening an autonomous scope on an AI agent will require a named branch context, with the scope-branch coupling recorded in the commit trailer. The exact mechanism (verb-level `--branch` flag, auto-creation, kernel finding) is a planning question downstream of this ADR — see G-0059's ladder (steps 2–5) for the surfaces under consideration.

### Human override

Humans are sovereign. The author can always commit on `main` regardless of model, including substantive multi-commit work. The model is the *default discipline*; the author's discretion is the *escape valve*. The kernel does not police human-actor commits against this convention — consistent with `--force` being human-only and the principal × agent × scope model documented in `docs/pocv3/design/provenance-model.md`.

## Consequences

**Positive:**

- **Closes G-0059's question.** The branch-model gap that's been open since M-0069 now has a recorded answer. Implementation (the chokepoint mechanism) is downstream planning.
- **Unblocks G-0116.** The `aiwfx-start-epic` sequencing fix has a clear target: reorder so promote + authorize fire on main before the worktree/branch is cut.
- **Clarifies G-0099's full-closure path.** The kernel-side `isolation-escape` finding (tracked under E-0019) can now police against a documented branch convention rather than an informal one.
- **Clarifies E-0019's shape.** Parallel-TDD-subagents need a place to land their cycles; the milestone-branch-under-epic-branch convention provides it.
- **Preserves author iteration speed.** Direct-to-main typing remains unencumbered.

**Negative:**

- **In-branch mutations sacrifice visibility to main until merge.** Operators running `aiwf status` from main see only the *opening* state for an in-flight epic; mid-flight milestone promotes, AC adds, and body refinements don't appear until the epic wraps. This is the conscious trade for atomicity of the unit-of-merge. Mitigations: encourage frequent epic-branch merges-down-from-main and/or design `aiwf status` (future) to surface in-flight ritual branches.
- **Two-tier discipline requires per-case author judgment.** *"Should I ritualize this?"* is a small but real cognitive tax. Mitigated by the rituals being optional — the author can always punt and commit on main.
- **Self-policed iteration boundary means no mechanical guard against author drift.** If the author starts landing 23-commit milestone-shaped work on main without ritualizing, nothing stops them. Consistent with sovereignty; the cost lands on the author.

**Follow-up work (sequenced downstream, not gated to this ADR's ratification):**

- **CLAUDE.md § "Working in this repo" rewrite** — replace the current single-paragraph trunk-based stance with the two-tier model. `wf-patch` shape.
- **G-0116 fix** — reorder `aiwfx-start-epic` steps so promote + authorize fire on main before the worktree/branch cut. `wf-patch` in the rituals repo, mirrored via the fixture pattern in `internal/policies/testdata/`.
- **Chokepoint implementation epic** — a focused epic (or milestone under E-0019) that lands the `aiwf authorize --branch` semantics, the scope-branch coupling in the trailer schema, and eventually a kernel finding for `isolation-escape` / branch-convention violations.
- **`aiwf status` surface for in-flight ritual branches** (later, after the model has lived for a few epics) — mitigate the in-branch-visibility trade-off.

## Validation

This ADR's contract should be revisited if:

- **Author iteration on main becomes a routine source of bisect or revert pain** — the self-policed boundary stopped working in practice and the kernel should police harder.
- **The in-branch-related-mutations rule produces persistent confusion** about which branch carries which planning state — possibly a sign that some mutations (e.g., milestone status changes mid-epic) should land on main after all.
- **AI-side ritualization friction outweighs the isolation value** — if every autonomous cycle pays a high coordination cost to set up its branch context, the chokepoint design needs revisiting.

Periodic touchpoint: at the next epic wrap (E-0029 or beyond), audit whether the sequencing and in-branch rules held up.

## References

- Related ADRs: ADR-0009 (Orchestration substrate; Decision 3 — isolation as parent-side precondition — is the neighbouring kernel-shape decision).
- aiwf gaps: G-0059 (question recorded here, answered by this ADR), G-0099 (worktree isolation; tier-3 partial-closed by `.claude/hooks/validate-agent-isolation.sh`; full closure waits on the kernel-side `isolation-escape` finding under E-0019), G-0116 (sequencing fix, unblocked by this ADR).
- Related epics: E-0019 (Parallel TDD subagents with finding-gated AC closure — the natural home for the chokepoint implementation).
- Provenance model: `docs/pocv3/design/provenance-model.md` (principal × agent × scope; sovereign override).
