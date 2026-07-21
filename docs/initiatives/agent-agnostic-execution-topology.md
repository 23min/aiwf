---
title: Agent-agnostic execution topology
status: captured
date: 2026-06-30
---

# Agent-agnostic execution topology

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind, so this file lives under `docs/initiatives/` as an umbrella capture.

This is not an ADR: it does not ratify a decision.
This is not research: it does not primarily survey or prove a thesis.
This is not an exploration: the need is no longer speculative.
This is not a plan: it intentionally avoids epics, milestones, sequencing, and
timeframes.

The purpose is to preserve the shape of a feature-sized concern so future epics
can be drafted from a coherent center instead of rediscovering the edges.

## Initiative statement

aiwf should become agent-agnostic at the execution-topology layer without
becoming an agent workspace manager.

The core posture:

> aiwf informs and validates execution topology; it does not prescribe or own the
> user's workflow.

The practical target is a configured, agent-readable, mechanically vetted model
for how aiwf work should be carried out across local checkouts, git worktrees,
devcontainers, and AI hosts such as Claude Code, Codex, and Copilot.

Worktrees are the immediate forcing function. Parallel work is materially better
when each concurrent session has its own checkout. But when worktree placement,
branch choreography, or host-specific sandbox behavior is left entirely to LLM
skill prose, the correctness of aiwf state depends on the agent remembering and
executing a fragile ritual. That violates one of aiwf's core principles:
framework correctness must not depend on LLM behavior.

The initiative is therefore not "make aiwf manage worktrees." It is:

- name the execution-topology contract;
- make it configurable;
- make it inspectable by humans and AI agents;
- make `doctor` able to verify the configured topology;
- give skills and agents one canonical aiwf answer for path, branch, and
  placement guidance;
- preserve user choice across operating models and AI hosts.

## Mission fit

aiwf's mission is to keep durable structural state honest inside the repo:
planning state, references, status transitions, provenance, audit history, and
mechanical validation. It is not a project-management tool and not a workflow
engine.

Execution topology is adjacent to that mission, not because aiwf should own
developer ergonomics, but because topology can affect the truth of aiwf state:

- a state-transition commit can land on the wrong branch;
- a delegated agent can act outside its authorized scope;
- a worktree can be unreachable inside a devcontainer;
- a worktree under `$HOME` can disappear on container rebuild;
- duplicate nested worktrees can confuse tree loaders and policy scans;
- sibling worktrees can allocate colliding ids if local refs are ignored;
- stale binaries can make `aiwf check` output look authoritative while running
  old code.

Those are not mere convenience failures. They affect state visibility,
provenance, branch binding, id allocation, and lost-work risk.

The boundary should be:

- aiwf does not dictate whether a project uses main-checkout work, in-repo
  worktrees, sibling worktrees, Codex-managed worktrees, or another host model.
- aiwf does declare the configured topology for aiwf-aware work.
- aiwf does validate whether that topology is currently safe.
- aiwf does provide enough structured output for Claude, Codex, Copilot, and a
  human shell to follow the same answer.

This preserves the "state, not workflow" philosophy. Workflow remains a render
or operating model. The execution-topology contract is a correctness boundary
around state mutation.

## Philosophical anchors

This initiative aligns with the existing doctrine:

- `README.md`: aiwf is a small git-native framework for validated planning
  state, not a project-management system.
- `docs/research/07-state-not-workflow.md`: state is canonical; workflow is an
  optional render.
- `docs/research/12-operating-model-agnostic.md`: aiwf sits underneath operating
  models and composes rather than competes.
- `CLAUDE.md`: framework correctness must not depend on LLM behavior.
- `work/gaps/G-0269-mutating-verbs-lack-a-head-drift-guard-against-shared-worktree-session-races.md`:
  separate worktrees are useful, but correctness must not require everyone to
  remember to use them.

The initiative sharpens those into an execution-topology doctrine:

> aiwf records structural state. Agents and humans may execute work in any
> operating model. When execution topology can affect aiwf correctness, aiwf
> exposes configured, validated, agent-readable guidance.

## Naming and placement direction

The current default worktree directory is `.claude/worktrees`, via ADR-0023.
That solved a real devcontainer durability problem, but it bakes a host name
into an aiwf convention.

A neutral in-repo default should be considered:

```yaml
worktree:
  placement: in-repo
  dir: .worktrees
```

The symmetry is intentional:

```text
work/        tracked aiwf entities and structural state
.worktrees/ ignored execution checkouts
```

`.aiwf/worktrees` is less attractive as a default because aiwf does not
currently have a broader private state directory. Creating `.aiwf/` just to hold
worktrees implies a private home that does not yet exist. `.worktrees` names the
thing directly.

Sibling worktrees remain valuable for users whose devcontainer or host topology
supports them:

```yaml
worktree:
  placement: sibling
  dir: ../worktrees/<repo>
```

The important distinction is that `sibling` is explicit. It depends on parent
folder mount discipline, path reachability, and host/container agreement.
`doctor` should vet those assumptions instead of letting the agent infer them.

## Inform, do not prescribe

The desired model is configured but not mandatory:

- `placement: in-repo` means aiwf expects worktrees under the repo root and can
  validate gitignore/loader exclusion and reachability.
- `placement: sibling` means aiwf intentionally allows a repo-escaping worktree
  root and validates that the path is writable, durable, and reachable.
- `placement: none` or direct main-checkout work should remain a legitimate
  operating choice for simple or sequential work.

The user or project chooses the operating model. aiwf makes the chosen model
legible and checkable.

This avoids a boolean such as `use-sibling-worktrees`. Placement is not a
single yes/no preference. It is a topology with different invariants.

## The middle path: plan, not create

A full `aiwf worktree create` verb would be a significant architectural step.
It would make aiwf own git worktree lifecycle behavior: branch existence,
collision behavior, dirty-state checks, cleanup semantics, nested worktree edge
cases, and possibly host-specific path translation.

That may eventually be justified, but it is not the smallest mission-aligned
step.

The better middle path is an informational/planning surface:

```bash
aiwf worktree plan <scope>
aiwf worktree plan <scope> --format=json
```

This surface would not mutate the filesystem. It would compute and report:

- intended branch name;
- intended worktree path;
- placement mode;
- base branch or base ref;
- expected current branch for state-changing aiwf verbs;
- exact `git worktree add ...` command or command family;
- post-create verification commands;
- warnings about devcontainer reachability, durability, ignored-file copying, or
  configured topology mismatch.

Skills and agents would call this surface instead of hand-rolling branch and
path math. A human can still inspect the plan and decide what to run. A future
mutating verb could reuse the same planning primitive if experience proves that
creation itself belongs in aiwf.

This is the cleanest expression of the initiative:

> aiwf is the source of truth for the safe topology; Git remains the tool that
> creates the worktree.

## Drop-in readiness audit

Before this initiative hardens into kernel behavior, a drop-in audit script
would be useful as a proving tool.

The script would answer two separate questions that are easy to conflate:

```text
Can this container/host support the desired topology?
Can this repo's aiwf installation express and validate that topology?
```

That is a two-stage rocket.

Stage 1 is environment readiness. A standalone script can be copied into any
devcontainer or local checkout and inspect the execution substrate before aiwf
itself has learned the full feature. It would report facts such as:

- current directory, git root, and repo basename;
- whether the checkout is already a linked git worktree;
- whether `/workspaces` or another parent directory appears to be host-mounted;
- whether only the repo root is mounted, making sibling worktrees unreachable;
- whether a sibling worktree directory can be created, entered, and removed;
- whether an in-repo ignored worktree directory is actually ignored by Git;
- whether `$HOME` looks durable or ephemeral;
- whether `CODEX_HOME`, `COPILOT_HOME`, Claude state, and GitHub CLI state point
  at durable locations;
- whether `git worktree add` works in the current repo;
- what `git worktree list --porcelain` already reports;
- whether common agent tools (`claude`, `codex`, `copilot`) and support tools
  (`git`, `rg`, `gh`, `aiwf`) are present.

Stage 2 is aiwf readiness. Once aiwf has topology-aware config and planner
surfaces, the same script can inspect whether this repo can consume them:

- `aiwf version`;
- `aiwf doctor`;
- current `aiwf.yaml` worktree config;
- whether the repo still uses the legacy `.claude/worktrees` default;
- whether `.worktrees` or the configured in-repo worktree root is ignored;
- whether generated guidance and agent surfaces are present;
- whether non-Claude surfaces such as `AGENTS.md` and `.agents/skills` exist;
- whether `aiwf worktree plan --help` exists;
- whether `aiwf worktree plan <scope> --format=json` emits the expected fields.

The output should be useful to humans and easy for agents to parse. For example:

```text
Environment topology:
  repo-root: /workspaces/aiwf
  parent-mounted: yes (/workspaces)
  sibling-worktrees: supported (/workspaces/worktrees/aiwf)
  home-durability: unknown
  codex-home: not durable (/home/vscode/.codex)
  recommended-placement: sibling

aiwf readiness:
  aiwf-version: v0.x.y
  worktree-config: legacy (.claude/worktrees)
  neutral-default-ready: no
  worktree-plan-command: missing
  agents-surface: partial (.agents/skills absent, AGENTS.md present)

Result:
  environment: ready for sibling topology
  aiwf: not yet feature-ready
```

The script should probably live outside the kernel at first: a portable shell
script that can be dropped into arbitrary containers and run before any aiwf
feature work exists. As the checks prove their value, the stable subset can move
into `aiwf doctor`, and the script can become a compatibility probe for older
aiwf versions or host setups.

This supports the initiative without collapsing it into implementation too
early. The audit script gathers empirical topology evidence; aiwf later absorbs
the parts that are clearly part of the durable correctness contract.

## Agent-host compatibility

Claude Code, Codex, and Copilot all have enough local capability to execute
worktree rituals:

- they can read repository guidance;
- they can run shell commands in a devcontainer or local checkout;
- they can follow skills or instruction files;
- they can operate from a given cwd when the host exposes that directory.

The problem is not raw capability. The problem is reliability and drift. Today,
the detailed worktree behavior lives primarily in Claude-specific skills and
`CLAUDE.md` prose. That makes the workflow harder to port to Codex or Copilot
without copying and reinterpreting host-shaped instructions.

The desired adapter story:

- Claude skills, Codex skills, Copilot skills, and human docs all ask aiwf for
  the configured topology.
- Host-specific files explain how to operate that host, not what the canonical
  aiwf worktree math is.
- `AGENTS.md` and `.agents/skills` become first-class shippable surfaces for
  non-Claude agents.
- `.claude/*` remains a Claude adapter, not the place where neutral aiwf
  doctrine lives.

Codex-specific considerations:

- Codex-managed app worktrees live under `$CODEX_HOME/worktrees`.
- Those are useful for Codex's own app workflow, but they are not neutral aiwf
  topology.
- If durability matters in a devcontainer, `CODEX_HOME` should be on a durable
  mount, such as `/workspaces/.ai-state/codex`.
- Manual git worktrees created from aiwf's topology remain the preferred neutral
  path for aiwf-scoped work.

Copilot-specific considerations:

- Copilot CLI can consume project instructions and skills, including
  `.agents/skills` and `.github` surfaces.
- That makes it a plausible future adapter, but it should consume the same
  aiwf topology output rather than learning a separate Copilot-specific ritual.

## Outside-tool context

Current spec-driven or SDD-adjacent tools tend not to own local worktree
topology as a first-class core abstraction.

Observed pattern:

- GitHub Spec Kit owns spec and branch workflow, but worktree management appears
  to remain user/agent discipline.
- Kiro is spec-aware and can run cloud tasks in isolated branches or sandboxes,
  but local worktree topology is not the central spec abstraction.
- OpenSpec community workflows discuss git worktrees explicitly, but as workflow
  guidance rather than a kernel-like planning/validation primitive.
- BMAD users report parallel-run friction and use manual git worktrees as a
  workaround.

The pattern is:

```text
Spec tools own spec/task artifacts.
Agent tools may own execution sandboxes.
Users and skills usually own local worktree discipline.
```

aiwf can do better without overreaching by owning the informational contract:
configured topology, doctor validation, and agent-readable worktree plans.

Relevant external examples:

- GitHub Spec Kit: https://github.com/github/spec-kit
- Kiro specs: https://kiro.dev/docs/specs/
- OpenSpec worktree workflow discussion:
  https://intent-driven.dev/blog/2026/04/01/openspec-git-worktrees-opencode/
- BMAD worktree friction example:
  https://github.com/bmad-code-org/BMAD-METHOD/issues/1750

## Existing aiwf surfaces this touches

### ADRs and docs

- `docs/adr/ADR-0023-default-to-in-repo-worktree-placement-under-claude-worktrees.md`
  currently records the devcontainer forcing function and `.claude/worktrees`
  default. This initiative may eventually supersede or amend the host-branded
  part while preserving the durable in-repo rationale.
- `docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md`
  is the branch-model backdrop for where state-announcement commits and
  implementation commits belong.
- `docs/research/07-state-not-workflow.md` provides the state-vs-workflow
  boundary.
- `docs/research/12-operating-model-agnostic.md` provides the compose-don't-absorb
  boundary.
- `docs/explorations/08-codex-compatibility-audit.md` identifies the Codex
  adapter surfaces and the need to avoid divergent instruction corpora.
- `docs/design/provenance-model.md` provides the principal x agent x
  scope model that execution topology must preserve.
- `docs/workflows.md` currently frames AI prompts around Claude Code "or
  any AI host"; this initiative gives that phrase a stronger technical basis.

### Open gaps

- `G-0099` worktree isolation parent-side precondition: establishes that
  isolation cannot depend on a host kwarg being honored.
- `G-0116` start-epic creates worktree before promote/authorize: records the
  branch-ordering hazard when state-announcement commits land in the wrong
  place.
- `G-0121` legal workflows and verb composition not pinned mechanically:
  explicitly calls out missing branch/worktree choreography.
- `G-0178` non-Claude target for ritual materializer: validates the adapter seam,
  initially via Codex.
- `G-0269` HEAD-drift guard: says correctness should not require everyone to
  remember separate worktrees.
- `G-0270` epic activation commit on a non-trunk branch: covers post-hoc
  detection for one wrong-branch choreography failure.
- `G-0272` allocator misses sibling worktree heads: shows topology affects id
  correctness.
- `G-0277` status/worktree stale view: shows worktree state visibility matters
  to user-facing status.
- `G-0281` opt-in gaps inbox on a never-checked-out ref: another example of
  reducing worktree desync hazards by moving correctness-sensitive state
  mutation out of the live checkout.
- `G-0313` consumer operating guidance drifts into `CLAUDE.md`: relevant because
  neutral topology doctrine must ship to consumers, not live only in this repo's
  Claude-specific guidance.

### Current skills

- `aiwfx-start-epic` currently performs worktree placement and branch creation
  in skill prose, reading `worktree.dir` from `aiwf doctor`.
- `aiwfx-start-milestone` inherits or repeats the same placement doctrine.
- Future agent-agnostic skills should cite an aiwf topology/plan surface instead
  of re-describing placement mechanics.

## Risks and boundaries

### Risk: aiwf becomes a workflow manager

Avoid by keeping the first-class surface informational and diagnostic. `doctor`
and `worktree plan` fit the existing aiwf style: expose state and findings,
leave execution to Git and the user unless a mutating verb is clearly justified
later.

### Risk: a neutral default breaks existing Claude users

Avoid with migration and compatibility:

- keep reading existing `worktree.dir`;
- warn before changing generated skill wording;
- treat `.claude/worktrees` as a legacy-valid configured value;
- make any default switch explicit in an ADR.

### Risk: sibling worktrees recreate devcontainer reachability failures

Avoid by making sibling placement opt-in and doctor-vetted. The old ADR-0023
lesson remains valid: a sibling path is only good when the container mount makes
it reachable and durable.

### Risk: too much agent-specific adapter drift

Avoid by placing neutral doctrine in `AGENTS.md`/shippable guidance and by using
skills only as host-specific adapters over the same aiwf output.

### Risk: `worktree plan` becomes a half-implemented create verb

Avoid by being explicit: it emits commands and checks, but does not run them.
If a future create verb is added, it should consume the same planner internally.

## Open design questions

These are intentionally not answered here.

- What is the exact config shape for placement?
- Should the neutral in-repo default become `.worktrees`, and if so how should
  `.claude/worktrees` migrate?
- Should repo-escaping worktree dirs be allowed only under
  `placement: sibling`?
- What should `aiwf doctor` report for placement health?
- What should `aiwf worktree plan --format=json` contain?
- Should the worktree planner be scope-aware for epics, milestones, gaps, and
  patch-shaped work?
- How should branch policy interact with trunk-based versus PR-style projects?
- Which guidance belongs in always-on shippable instructions versus on-demand
  skills?
- What is the minimal Codex target that proves the neutral topology contract?
- Does Copilot need a distinct adapter, or can it consume the same `.agents`
  materialization plus `AGENTS.md` guidance?

## Desired future property

A future human or AI agent should be able to enter any aiwf repo and ask:

```bash
aiwf doctor
aiwf worktree plan E-0042 --format=json
```

From those outputs, the agent should know:

- where aiwf expects isolated implementation work to happen;
- whether that location is durable in the current environment;
- what branch should exist;
- what command would create the worktree;
- what post-create checks prove the worktree is actually present;
- what risks remain for the chosen topology.

That is the initiative's center. aiwf should not replace Git, Claude, Codex,
Copilot, VS Code, or the user's operating model. It should make the aiwf-safe
execution topology explicit enough that all of them can cooperate without
copying fragile ritual prose.
