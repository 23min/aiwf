---
title: Agent-host artifact adapters — side-by-side Claude and Codex generation
status: captured
date: 2026-08-04
---

# Agent-host artifact adapters — side-by-side Claude and Codex generation

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf
entity kind, so this file lives under `docs/initiatives/` as an umbrella
capture.

This is not an ADR: it does not ratify the remaining artifact-ownership and
rendering decisions. This is not an exploration: the Claude-specific coupling
and the viability of a Codex adapter have already been established. This is not
an executable plan: it bounds the work, records invariants, and proposes a
delivery shape from which an epic and milestones can be drafted.

The initiative is deliberately separate from
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md).
Execution topology describes where work happens and which checkout or branch is
authoritative. This initiative describes which host-facing files aiwf generates
and how it keeps those files derived from one canonical workflow definition.
They interact, but they are independently deliverable concerns.

## Initiative statement

aiwf should generate host-native artifacts for Claude Code and Codex side by
side without changing existing Claude behavior or creating a second workflow
implementation.

The desired architecture is:

```text
canonical aiwf definitions
          |
   host-aware rendering
      /             \
Claude artifacts   Codex artifacts
(existing output)  (additive output)
```

The core workflow model remains host-independent. Host adapters own filesystem
locations, file formats, and the small amount of operational wording that is
genuinely different between hosts.

The first product target is local Codex CLI and IDE use in a repository that may
also use Claude Code. Codex agents, hooks, status surfaces, cloud execution, and
Codex-managed worktree behavior are separate follow-on capabilities rather than
implicit parts of the first compatibility claim.

## Evidence and relationship to prior work

The dated
[`08-codex-compatibility-audit.md`](../explorations/08-codex-compatibility-audit.md)
is the historical inventory and first compatibility proposal. It correctly
identified skills as the easiest initial surface and guidance, role agents,
hooks, and cloud behavior as distinct problems. Its product facts should be
treated as an audit snapshot, not timeless Codex documentation.

The
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md)
initiative is adjacent. It should supply host-neutral facts about paths,
branches, worktrees, and execution placement. Artifact adapters should consume
those facts rather than independently encode topology rules in Claude and Codex
prose.

The current repository already contains a useful but incomplete seam:

- `internal/skills.Target` names output directories;
- `internal/skills.ClaudeTarget` is the only production target;
- `internal/skills.MaterializeTo` can write the shared skill corpus to another
  directory layout;
- `internal/skills/materialize_target_test.go` proves path redirection with a
  test-only Codex-shaped target;
- `aiwf.yaml` already has a `hosts` field, but the schema currently documents
  `claude-code` as its only supported value;
- init, update, doctor, hook synchronization, guidance wiring, and worktree
  refresh still select Claude directly.

The existing target seam proves that a second skills directory is feasible. It
does not yet model a host adapter: Codex guidance, agents, configuration, and
hooks use different artifact types, not merely different directory names.

## Current Codex surface, revalidated 2026-08-04

Current Codex documentation establishes the relevant native surfaces:

- repository guidance is discovered through
  [`AGENTS.md`](https://learn.chatgpt.com/docs/agent-configuration/agents-md);
- repository skills are discovered under
  [`.agents/skills`](https://learn.chatgpt.com/docs/build-skills);
- project configuration is stored in `.codex/config.toml` and project `.codex/`
  layers apply only after the project is trusted;
- project custom agents are TOML files under `.codex/agents/`, not Markdown
  role files;
- lifecycle hooks use Codex hook configuration rather than Claude settings;
- Codex-managed worktrees may need ignored local artifacts copied through
  [`.worktreeinclude`](https://learn.chatgpt.com/docs/environments/git-worktrees#copy-ignored-local-files-into-managed-worktrees).

These facts define separate adapter capabilities. A target with
`SkillsDir`, `AgentsDir`, `TemplatesDir`, and `HooksDir` strings cannot express
which capabilities a host supports or how their formats differ.

## Desired future property

Given the same aiwf version and configuration:

1. a project configured for Claude produces exactly the Claude artifacts it
   produces before this initiative;
2. a project configured for Codex produces a complete, internally consistent
   local Codex skill and guidance surface;
3. a project configured for both produces both independent surfaces in one
   refresh;
4. regenerating either surface is deterministic and idempotent;
5. shared workflow facts have one canonical source;
6. doctor reports health against the hosts selected by the project rather than
   against Claude unconditionally;
7. no claim of Codex agent, hook, cloud, review, or managed-worktree support is
   made until that capability has its own implementation and tests.

## Non-negotiable compatibility invariants

### Claude remains the compatibility anchor

Absent new configuration, existing repositories continue to behave as Claude
repositories. The current Claude artifact paths, file formats, consent gates,
and generated contents remain unchanged.

Before a shared renderer is refactored, characterization tests must pin the
current Claude output. The compatibility condition is generated-byte
equivalence, not merely semantic similarity. Intended future Claude changes can
then update those fixtures explicitly as ordinary Claude changes, independent
of Codex support.

### Canonical content is not duplicated by host

Verb definitions, ritual intent, templates, workflow contracts, and shared
guidance remain canonical once. Adapters derive host output from those sources.
Maintaining a complete Claude corpus and a complete Codex corpus independently
would create two unvalidated sources of truth.

Small host-specific fragments are appropriate where a host has different tool
names, instruction discovery, subagent semantics, or filesystem layout. A full
host-specific override should be exceptional and should name why shared
rendering is insufficient.

### Host selection is explicit

The existing `hosts` field is the natural configuration boundary. The
compatibility-preserving direction is:

```yaml
# Existing/default behavior
hosts: [claude-code]

# Side-by-side behavior
hosts: [claude-code, codex]

# Intentional Codex-only behavior
hosts: [codex]
```

Adding Codex support must not silently enable Codex output in every existing
consumer. Deselecting a host may remove only artifacts proven to be aiwf-owned;
it must not remove or rewrite user-owned host files.

### User-owned guidance retains a consent boundary

Root `CLAUDE.md` and `AGENTS.md` files are user-owned instruction surfaces.
aiwf may manage only a clearly delimited block or an explicitly authorized
reference, must preserve surrounding bytes, and must detect malformed or
conflicting markers rather than guessing.

Claude's existing guidance behavior is governed by its current contract. Codex
guidance needs its own contract because Codex does not document Claude-style
`@file` imports as an `AGENTS.md` mechanism.

## Capability and artifact matrix

| Capability | Claude today | Codex local target | First slice |
|---|---|---|---|
| Skills | `.claude/skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` | Yes |
| Persistent guidance | `CLAUDE.md` import plus `.claude/aiwf-guidance.md` | Marker-managed `AGENTS.md` content or another explicitly decided native shape | Yes |
| Shared templates | `.claude/templates/*.md` | Adapter-owned support/reference location; Codex has no equivalent `.agents/templates` convention | Yes, after location decision |
| Role agents | `.claude/agents/*.md` | `.codex/agents/*.toml` | Deferred |
| Hooks | Claude settings plus `.claude/hooks/*` | Codex hooks/config with project trust | Deferred |
| Statusline | `.claude/statusline.sh` plus settings wiring | No direct parity target established | Deferred |
| Project host config | Claude settings files | `.codex/config.toml` where needed | Deferred unless the first slice proves a required setting |
| Local CLI/IDE | Materialized ignored artifacts | Materialized ignored artifacts | Yes |
| Managed worktrees | aiwf-created worktree refresh | Codex-managed checkout and `.worktreeinclude` behavior | Deferred |
| Cloud/review | Not the current adapter contract | Tracked/setup-provisioned guidance and skills | Deferred |

The embedded corpus currently contains 19 verb skills, 19 ritual skills, four
role agents, four substantive templates, one guidance fragment, one hook, and
one statusline script. The first Codex slice therefore generates approximately
43–47 Codex files or directories depending on the chosen support-file layout,
but those outputs are derived artifacts rather than individually maintained
implementations.

## Architectural boundary

The implementation should introduce a host capability model, not continue to
grow a bag of path strings. A host adapter needs to answer questions such as:

- which artifact capabilities it supports;
- where each supported artifact is written;
- whether an artifact can be copied verbatim, rendered from shared content, or
  requires a host-native encoder;
- which user-owned guidance file it may wire and under what consent contract;
- how ownership and stale-artifact cleanup are recorded;
- which health checks apply to that host.

The core should pass typed canonical definitions to adapters. Adapters should
not parse another host's generated files as their source. The dependency
direction remains:

```text
config + canonical definitions
            |
      materialization service
       /                \
Claude adapter       Codex adapter
       |                 |
host-owned paths and host-native encodings
```

The smallest implementation may keep adapters inside `internal/skills` while
there are only two and their behavior remains cohesive. A generalized plugin
system or public provider interface is not justified by this initiative.

## Change surface

### Production and configuration

Expected concentration:

- `internal/skills`: typed host/capability selection, Codex skill rendering,
  ownership manifests, and host-specific content fragments;
- `internal/initrepo`: refresh all configured hosts and wire applicable
  guidance;
- `internal/config`: validate `claude-code`, `codex`, and their combination;
- `internal/config/schema.go` and `aiwf.example.yaml`: make host selection
  discoverable;
- `internal/cli/initcmd`: initialize selected host artifacts;
- `internal/cli/update`: reconcile all selected host artifacts;
- `internal/cli/doctor`: report per-host materialization and guidance health;
- `internal/cli/worktree`: reproduce configured artifacts in aiwf-created
  worktrees.

### Tests

Tests are a material part of the feature rather than cleanup after the
renderer. Expected coverage includes:

- golden characterization of current Claude output;
- proof that the existing/default configuration stays Claude-only;
- Codex-only and side-by-side materialization;
- deterministic and idempotent repeated refresh;
- config parsing, validation, schema, and example generation;
- init/update/worktree integration for every supported host selection;
- doctor behavior for selected, absent, stale, and unselected host artifacts;
- preservation and conflict handling for user-owned `AGENTS.md` content;
- ownership-safe cleanup when a host is deselected;
- every reachable branch introduced in target selection and rendering.

Tests should assert observable filesystem and CLI behavior. Internal renderer
calls are not the contract.

### Policies

The policy surface should be targeted. Expected additions or amendments are:

- supported config fields and values are discoverable;
- default initialization remains Claude-compatible;
- the scaffold and example config describe side-by-side selection;
- Claude characterization fixtures cannot drift unnoticed;
- no Codex artifacts appear unless Codex is selected;
- host-specific wording does not enter the wrong host's output;
- tracked guidance and generated guidance do not become competing authorities.

Ordinary entity, FSM, verb, reference, and repository-tree policies do not
change merely because another host can invoke the same aiwf behavior.

### Documentation and dogfooding

Documentation should distinguish these claims:

- local Codex CLI/IDE compatibility;
- Codex custom-agent compatibility;
- Codex hook integration;
- Codex-managed worktree compatibility;
- Codex cloud/review compatibility.

The repository's devcontainer needs Codex installation and durable state only
if aiwf chooses to dogfood Codex as a supported development host. That is an
onboarding consequence, not a prerequisite for consumers to materialize Codex
artifacts with the aiwf binary.

## Kernel impact

The domain kernel should not change for this initiative:

- entity kinds and schemas;
- epic and milestone state machines;
- verb semantics and legality;
- dependency and reference rules;
- tree loading and canonical planning state;
- ordinary `aiwf check` findings.

Configuration, initialization, materialization, doctor, update, and worktree
refresh are kernel-adjacent application plumbing and do change. This boundary
is load-bearing: Codex support is another presentation/execution adapter, not a
second workflow engine.

No persisted planning-state migration, external service, runtime dependency, or
network integration is required for the local CLI/IDE slice.

## Size estimate

For a coherent local CLI/IDE slice:

| Surface | Estimated maintained change |
|---|---:|
| Production and configuration | 10–14 files, 400–800 LOC |
| Tests | 12–20 files, 500–900 LOC |
| Policies | 2–6 files, 100–250 LOC |
| Docs, examples, and optional devcontainer wiring | 5–9 files, 200–400 LOC |
| Core domain kernel | Approximately zero |
| **Total** | **30–45 files, 1,200–2,200 LOC** |

The range assumes host differences are represented as small fragments or
renderers over shared definitions. Copying and independently maintaining all
38 skill bodies would increase both the immediate diff and permanent drift
cost and is outside this estimate.

Full local parity including custom agents, hooks, and deeper environment
onboarding is approximately 45–70 maintained files and 2,500–4,000 changed or
new LOC. Cloud/review distribution is a separate product slice because ignored
local artifacts do not automatically exist in a clean remote checkout.

## Proposed delivery shape

This is sequencing guidance for a future epic, not ratified milestones.

### 1. Freeze the Claude compatibility contract

- inventory generated Claude artifacts and consent behavior;
- add golden or digest-backed characterization scenarios at the materializer
  and refresh seams;
- prove default init, update, doctor, and worktree behavior before refactoring.

Exit property: a later adapter refactor cannot accidentally change Claude
output without an explicit fixture change.

### 2. Activate host selection and Codex skills

- validate `codex` in the existing `hosts` field;
- replace unconditional `ClaudeTarget` selection at orchestration seams with
  configured host iteration;
- materialize the 38 shared skills under `.agents/skills`;
- keep agents, hooks, statusline, and project Codex config out of this step;
- prove Claude-only, Codex-only, and side-by-side behavior.

Exit property: a local Codex session can discover aiwf skills while Claude
output remains unchanged.

### 3. Add Codex-native guidance and support files

- settle the `AGENTS.md` ownership/consent contract;
- render concise Codex guidance without assuming Claude import syntax;
- select a support/reference layout for the four shared templates;
- host-neutralize shared skill wording or add narrow Codex fragments where
  tools and semantics differ;
- keep canonical workflow facts single-sourced.

Exit property: a local Codex session can follow aiwf's standing operating rules
without reading or translating Claude artifacts.

### 4. Complete lifecycle integration

- make init, update, doctor, and aiwf-created worktrees converge on the selected
  host set;
- make dry-run and stale-artifact behavior host-aware;
- update schema, examples, operator documentation, and targeted policies;
- dogfood side-by-side generation in this repository if desired.

Exit property: supported local entry points agree about installed hosts and can
diagnose incomplete or stale Codex materialization.

### Deferred capability slices

Each of these needs its own forcing use case and compatibility contract:

- render four Claude Markdown roles as Codex TOML custom agents;
- define Codex-native hook behavior and project-trust consent;
- determine whether any status surface is useful rather than seeking cosmetic
  parity;
- support Codex-managed worktrees and ignored-artifact propagation;
- distribute required artifacts to Codex cloud and code review;
- support additional hosts beyond Claude and Codex.

## Risks and controls

### Claude behavior drifts during abstraction

Control: characterize output before extracting shared rendering. Treat fixture
changes as intentional product changes, not incidental refactor updates.

### Shared content accumulates unreadable host conditionals

Control: keep shared workflow statements host-neutral; render named fragments
at explicit seams; use a full override only when semantics genuinely differ.

### Two generated surfaces become two authorities

Control: generated files carry ownership/provenance; source content remains
embedded and canonical; generated files are never read back as definitions.

### A path-only target abstraction lies about compatibility

Control: model capabilities and encoders explicitly. Do not populate fictitious
directories such as `.agents/agents` or `.agents/templates` merely to preserve
the current struct shape.

### Guidance installation overwrites user instructions

Control: use an explicit consent contract, marker ownership, byte preservation,
conflict detection, dry-run visibility, and ownership-safe removal.

### “Codex support” overclaims unsupported surfaces

Control: expose and document capabilities separately. Local skills and guidance
do not imply custom agents, hooks, managed worktrees, cloud, or review support.

### Additional hosts trigger a speculative framework

Control: implement two concrete adapters with the smallest typed boundary that
fits them. Generalize only when a third real host demonstrates repeated shape.

## Open decisions, in dependency order

These should be decided one at a time before or during epic drafting:

1. Confirm the `hosts` compatibility contract: absent means Claude, selection
   is explicit, and `[claude-code, codex]` means side-by-side reconciliation.
2. Choose the Codex guidance ownership shape: marker-managed root `AGENTS.md`
   block, generated tracked file, or another native arrangement.
3. Choose the shared template/reference location for Codex without inventing a
   false Codex convention.
4. Choose the smallest host-fragment mechanism that keeps the 38 skill sources
   canonical and readable.
5. Decide whether this repository's devcontainer becomes an explicit Codex
   dogfooding environment in the first epic or in follow-on onboarding work.

Agent TOML, hooks, status surfaces, managed worktrees, and cloud distribution
remain later decisions unless the local slice uncovers a hard dependency.

## Ready-for-epic condition

This initiative is ready to become an executable epic when decisions 1–4 above
are settled and the Claude characterization fixture strategy is demonstrated on
the current materializer. The epic should then encode the four proposed slices
as acceptance-driven milestones, preserving the deferred capability boundary.

