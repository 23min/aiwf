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

This is not an ADR: it captures the agreed planning direction and the remaining
artifact-layout and rendering decisions. This is not an exploration: the Claude-specific coupling
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

Given the same aiwf version, canonical definitions, and resolved host set:

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

Whenever Claude is selected, its current artifact paths, file formats, consent
gates, and generated contents remain unchanged. Host selection defaults to local
detection, so a repository with no explicit host configuration may additionally
receive Codex artifacts. That selection change is intentional; it does not
license incidental changes to the Claude renderer.

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

### Host selection defaults to local detection

`aiwf init` and `aiwf update` detect executable `claude` and `codex` commands
on the invoking process's `PATH`. Detection runs in the environment where aiwf
runs: inside the devcontainer for a container invocation. It does not invoke
the hosts, check login state, or contact an external service.

| Detected locally | Artifacts selected |
|---|---|
| Claude only | Claude skills, support files, and managed `CLAUDE.md` guidance |
| Codex only | Codex skills, support files, and managed `AGENTS.md` guidance |
| Both | Both host surfaces |
| Neither | Core aiwf setup; report that no supported host was detected |

The existing `hosts` field is the explicit override. An absent field enables
detection on every refresh; an explicit list selects exactly those hosts even
when their executables are absent. This supports CI, IDE-only installations,
and preparing another environment without searching editor extension internals.
An explicit empty list selects no host artifacts. Unknown host names are errors.

```yaml
# Pin Claude regardless of local detection
hosts: [claude-code]

# Pin both hosts regardless of local detection
hosts: [claude-code, codex]

# Pin Codex regardless of local detection
hosts: [codex]
```

The ledger reports which hosts were selected and whether selection came from
detection or configuration. Detection must not persist a machine's executable
inventory into shared project configuration. The config reader/writer must
preserve the distinction between an absent field and an explicit empty list.

A host disappearing from `PATH` is not a removal request. Preserve its existing
artifacts and guidance; report their retained, unselected state without claiming
they were refreshed. Host deselection likewise does not delete files implicitly.
Any future removal operation must be explicit and ownership-scoped.

Doctor and aiwf-created worktree refresh use the same host-resolution rules.
Tests control executable discovery at the process boundary; they must not depend
on which assistants happen to be installed on the test runner.

### User-owned guidance retains a consent boundary

Root `CLAUDE.md` and `AGENTS.md` files are user-owned instruction surfaces.
aiwf may manage only a clearly delimited block or an explicitly authorized
reference, must preserve surrounding bytes, and must detect malformed or
conflicting markers rather than guessing.

Claude's existing guidance behavior is governed by its current contract. For
selected Codex output, init/update automatically maintain a concise, marked
block of guidance directly in the root `AGENTS.md`, creating that file if
absent. A persistent configuration opt-out disables this wiring. The block is
derived from canonical guidance, not a separately maintained workflow source;
it does not depend on Claude-style `@file` imports. Changes appear in the ledger
and dry-run output.

Malformed or conflicting markers must leave user content untouched. Guidance
writers inspect the instruction path without following links. A symlink is
preserved and its guidance update skipped, with a diagnostic naming the path and
remediation; doctor must not call that skipped installation healthy. If the host
instruction paths alias the same file, preflight reports the conflict before
either host rewrites it. The user can wire guidance manually and opt out of
automatic wiring, or choose separate regular instruction files.

Apply this policy to both host guidance writers, addressing G-0501 as an explicit
exception to Claude behavior preservation. Keep `AtomicWriteFile` unchanged:
its replace-at-path contract protects against partial writes and is shared by
other callers. Guidance presence alone is not proof that a host loaded it:
fresh-session verification must also exercise instruction discovery, overrides,
and size limits, without treating model self-attestation as a mechanical guarantee.

## Capability and artifact matrix

| Capability | Claude today | Codex local target | First slice |
|---|---|---|---|
| Skills | `.claude/skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` | Yes |
| Persistent guidance | `CLAUDE.md` import plus `.claude/aiwf-guidance.md` | Automatically maintained root `AGENTS.md` block, with persistent opt-out | Yes |
| Shared templates | `.claude/templates/*.md` | `.agents/aiwf/templates/*.md`, an aiwf-owned support directory | Yes |
| Role agents | `.claude/agents/*.md` | `.codex/agents/*.toml` | Deferred |
| Hooks | Claude settings plus `.claude/hooks/*` | Codex hooks/config with project trust | Deferred |
| Statusline | `.claude/statusline.sh` plus settings wiring | No direct parity target established | Deferred |
| Project host config | Claude settings files | `.codex/config.toml` where needed | Deferred unless the first slice proves a required setting |
| Local CLI/IDE | Materialized ignored artifacts | Materialized ignored artifacts | Yes |
| Managed worktrees | aiwf-created worktree refresh | Codex-managed checkout and `.worktreeinclude` behavior | Deferred |
| Cloud/review | Not the current adapter contract | Tracked/setup-provisioned guidance and skills | Deferred |

The inventory checked on 2026-09-18 contains 39 skills and six template Markdown
files. Derive the implementation inventory from the embedded loaders rather
than pinning a hand-maintained count. Generated outputs are derived artifacts,
not independently maintained implementations.

Codex skills reference `.agents/aiwf/templates/` explicitly. This directory is
aiwf's support layout, not a Codex discovery convention. Each template has one
generated copy per host, derived from the same canonical embedded source.
Claude's `.claude/templates/` paths remain unchanged. The Codex directory gets
ownership tracking and reference-resolution checks alongside its skill output.

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

### Explicit host rendering

The canonical skill and guidance sources use explicit placeholders for values
such as template paths, and small named fragments for host-specific instructions
such as worktree entry and independent-review dispatch. Common workflow prose
is authored once. Host fragments carry only the operational differences, not
copies of entire rituals.

The renderer uses a closed, typed set of bindings and fragment names. It must
reject unknown placeholders and missing host bindings before writing artifacts.
It does not infer substitution sites from ordinary prose or broadly replace
every occurrence of `.claude`: that path can legitimately name the configured
worktree directory even in Codex output.

Rendering Claude must reproduce its characterized output bytes. Both hosts use
the same rendering entry point for materialization and expected-content health
checks. Tests exercise substitution, missing bindings, deterministic output,
and references to generated files; they do not pin prose phrases. No new runtime
dependency or general-purpose extension system is required.

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
- proof that explicit Claude selection preserves existing Claude behavior;
- all four executable-detection combinations, explicit overrides, absent versus
  empty host lists, and temporarily missing executables;
- Codex-only and side-by-side materialization;
- deterministic and idempotent repeated refresh;
- config parsing, validation, schema, and example generation;
- init/update/worktree integration for every supported host selection;
- doctor behavior for selected, absent, stale, and unselected host artifacts;
- preservation and conflict handling for user-owned `AGENTS.md` content;
- retention of existing host artifacts when detection or selection changes;
- every reachable branch introduced in target selection and rendering.

Tests should assert observable filesystem and CLI behavior. Internal renderer
calls are not the contract.

### Policies

The policy surface should be targeted. Expected additions or amendments are:

- supported config fields and values are discoverable;
- selecting Claude preserves the characterized Claude output;
- the scaffold and example config describe side-by-side selection;
- Claude characterization fixtures cannot drift unnoticed;
- no Codex artifacts appear unless Codex is detected or explicitly selected;
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

This repository is the first dogfooding environment. Its devcontainer must
provide Codex installation and durable state alongside Claude. Installation and
mount changes are present in the working tree; a fresh rebuild remains a
separate verification step. Consumers can explicitly select either host without
installing its CLI on the machine materializing the artifacts.

Switching assistants does not change the selected host set. The receiving
session reads aiwf records, the current branch and worktree state, and a handoff
describing unfinished work. Transferring provider chat transcripts is outside
this feature. Parallel implementation uses separate branches/worktrees, with
both surfaces available in each; two terminals in one checkout do not isolate
edits or Git operations. Git integration remains explicit, not an automatic
cross-worktree synchronization service.

The existing configurable `worktree.dir` and its `.claude/worktrees/` default
remain unchanged. The directory name does not assign a worktree to an AI host;
either host can operate there. Renaming or migrating worktree storage is outside
this feature.

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
skill bodies would increase both the immediate diff and permanent drift
cost and is outside this estimate.

Full local parity including custom agents, hooks, and deeper environment
onboarding is approximately 45–70 maintained files and 2,500–4,000 changed or
new LOC. Cloud/review distribution is a separate product slice because ignored
local artifacts do not automatically exist in a clean remote checkout.

## Proposed delivery shape

This is sequencing guidance for a future epic, not ratified milestones.

### Preserve Claude output through explicit rendering

Acceptance outline:

- Record generated Claude paths, modes, and bytes before changing canonical
  sources. Cover default and explicit-Claude init, refresh, and worktree setup,
  including surrounding user content and the existing hook consent choices.
- Introduce the typed host bindings and named fragments. Claude rendering is
  byte-equivalent; unknown placeholders and missing bindings fail before writes.
- Compare independent materializations and repeated refreshes. Assert observable
  files and outputs, respecting accepted D-0070; do not add prose-presence tests
  or treat proposed D-0072 as an accepted exception.

Exit property: the rendering boundary is working and tested while the public
host-selection behavior remains unchanged. No Codex support is advertised yet.

### Implement Codex artifacts and safe guidance ownership

Depends on the rendering boundary.

Acceptance outline:

- Render the skill corpus to `.agents/skills/`, shared templates to
  `.agents/aiwf/templates/`, and concise guidance for the root `AGENTS.md` block.
  All generated local references resolve. No Claude agent Markdown is emitted
  into a fictitious Codex directory.
- Preserve user bytes, modes where applicable, and foreign artifacts; detect
  ownership collisions and malformed markers before overwriting their content.
  Repeated generation converges, and interrupted writes are recoverable through
  another refresh. Per-file atomic writes do not claim transactionality across
  the whole artifact set.
- Skip and diagnose symlinked instruction files for both hosts without changing
  the shared atomic helper. Test broken links, external targets, loops, and
  aliased host paths as well as ordinary files and persistent wiring opt-outs.
- Render host-specific worktree and delegation instructions. A ritual requiring
  independent review has an executable Codex path; unavailable delegation is a
  reported limitation, never permission to substitute self-review.

Exit property: complete Codex artifact and guidance operations exist behind the
internal adapter boundary, with ownership behavior pinned. Automatic detection
is not exposed until lifecycle integration is complete.

### Enable detection across setup, refresh, worktrees, and diagnosis

Depends on both artifact adapters.

Acceptance outline:

- Resolve the four executable-detection combinations plus explicit configuration
  overrides. Preserve absent versus empty host lists through config rewrites;
  reject unknown names and deduplicate repeated valid names deterministically.
- Init, update, and aiwf-created worktrees use the same resolver. Schema, generated
  examples, command help, dry-run ledgers, and existing upgrade flows that invoke
  refresh describe the effective behavior. Dry-run performs no writes.
- Core Git hooks remain host-independent. Claude lifecycle hooks, settings, and
  statusline operations run only for selected Claude and retain their existing
  consent rules. Codex-only and no-host setups do not accidentally create Claude
  artifacts through secondary orchestration paths.
- Both detected hosts receive complete artifacts in one invocation. A missing
  executable or deselected host leaves its existing files untouched; the report
  distinguishes retained files from refreshed or healthy selected artifacts.
- Doctor compares selected-host skills, templates, and guidance against the
  expected rendered output. Mutating a generated artifact produces a named drift
  finding; an unrelated host or user-owned surrounding content does not. Treat
  availability, materialization, drift, and guidance-wiring conflicts separately.
- Ownership cleanup applies only to obsolete entries in an aiwf manifest for an
  actively refreshed host. Test foreign files, unowned-name collisions, unsafe
  manifest paths, and invalid configuration without destructive side effects.

Exit property: automatic host setup is usable through every supported local entry
point, including Codex-only installations. There is no public skills-only stage.

### Verify switching and parallel sessions in the dogfooding environment

Depends on the complete public lifecycle.

Acceptance outline:

- Run fresh Claude and Codex sessions in a disposable consumer checkout. Record
  host versions, launched paths, discovered skills/instruction sources where the
  host exposes them, and a representative planning, worktree, and review flow.
  Model narration alone is observational evidence, not a delivery guarantee.
- Exercise a Claude-to-Codex-to-Claude handoff using the same worktree's files and
  aiwf records. Exercise simultaneous sessions in separate branches/worktrees;
  verify that edits and pending Git state stay in the intended checkout. Keep
  shared Git hooks and the installed binary version consistent across sessions.
- Supply this repository's development instructions to Codex without independently
  maintaining a second copy of the engineering rules in `CLAUDE.md`. This is
  separate from the consumer guidance block; verify the actual loaded sources
  and instruction-size budget in a fresh Codex session.
- Complete the devcontainer install/persistence change and validate a fresh
  rebuild when explicitly authorized. Exercise build, lint, race tests, coverage
  gates, and selfcheck before release, with every reachable new branch exercised.
- Document supported capabilities, host overrides, guidance opt-outs, skipped
  symlinks, handoff steps, and the difference between installation health and
  observed host discovery. Keep old-binary downgrade protection and universal
  guidance delivery claims outside this epic.

Exit property: the local compatibility claim has recorded session evidence for
both hosts. A Claude session blocked by account limits or a rebuild not run stays
an outstanding verification item, not an assumed pass.

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

## Agreed planning direction

The local CLI/IDE scope includes automatic host detection, explicit configuration
overrides, automatic marker-managed guidance with an opt-out, shared Codex
templates at `.agents/aiwf/templates/`, explicit placeholders and named host
fragments over canonical sources, refusal to write through symlinked instruction
files, preservation when a host disappears, switching between hosts, parallel
worktree sessions, and dogfooding in this repository's devcontainer. Planning
remains on main; implementation runs in an isolated worktree, following D-0073.

## Related work and scope boundaries

- G-0178 is the original second-host materializer gap; this work implements its
  proof within a complete local host lifecycle.
- G-0501 supplies the symlink preservation defect addressed by both guidance
  writers. This is an intentional Claude behavior correction.
- G-0504 motivates content-aware installation checks. This epic covers the
  selected-host artifact families it materializes; assess the gap's full scope
  before claiming closure.
- G-0523 concerns a mechanical delivery channel independent of instruction-file
  discovery. Fresh-session observations here do not close that gap.
- G-0600 concerns stale binaries downgrading newer artifacts. Development uses a
  source-built binary; a general downgrade guard remains separate work.
- E-0092 reduces always-on guidance. Reuse its results if it lands first; do not
  duplicate that reduction project or require it to enable Codex.
- E-0019 concerns orchestrated parallel TDD subagents. Two human-operated host
  sessions in separate terminals do not require that orchestration feature.

Agent TOML, hooks, status surfaces, managed worktrees, and cloud distribution
remain later decisions unless the local slice uncovers a hard dependency.

## Ready-for-epic condition

The design choices are settled for epic drafting. Encode the delivery outline as
acceptance-driven milestones through aiwf's entity verbs after reviewing the
proposed epic body. Entity allocation creates commits; the draft itself does not
allocate ids or activate implementation. Before implementation, bring the
approved planning commits into the isolated worktree with an explicitly approved
local integration step.

### Characterization strategy evidence

On 2026-09-18, a binary built from source at `2d30d384e` exercised
`aiwf init --root <temporary-repository> --actor human/planning-probe --no-prompt
--skip-hook` twice in each of two independent temporary Git repositories. Each
started with a user-authored `CLAUDE.md`. A manifest of relative paths, modes,
and SHA-256 digests matched for 57 files across roots and repeated invocations;
the original user content remained intact.

The compared set comprised the generated Claude skills, agents, templates,
guidance fragment, root `CLAUDE.md`, `.gitignore`, and generated example config.
This establishes that a deterministic artifact fixture is practical. It does
not replace the permanent compatibility tests, cover Git hook installation, or
claim live host discovery. The first milestone adds the missing lifecycle and
consent cases before the renderer changes.
