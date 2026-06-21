---
title: Codex compatibility audit
status: audit
date: 2026-06-16
---

# Codex compatibility audit

## Purpose

This document audits the Claude-specific surfaces in `aiwf` and proposes a
Codex-first compatibility path. It is intentionally not an implementation plan
or ADR. The near-term goal is to make the existing Claude assumptions visible,
name the Codex-native equivalents, and identify the smallest product slice that
would let a Codex operator use `aiwf` without hand-translating the workflow.

The bigger agent-agnostic goal remains valid, but Codex is the right first
second host because it supports the same `SKILL.md` format for skills. The
hard parts are not the skill files. The hard parts are project guidance,
subagent roles, hooks/settings, status surfaces, cloud behavior, and avoiding
two divergent instruction corpora.

## Executive summary

`aiwf` is already structurally close to Codex compatibility at the skill layer.
ADR-0014 explicitly created an agent-target seam, and
`internal/skills.MaterializeTo` already has a test-only Codex-shaped target that
writes verbatim `SKILL.md` files to `.agents/skills/`. That is the low-risk
first step: ship a production Codex target for skills.

Full Codex compatibility is broader than that:

1. Codex repo skills live under `.agents/skills`, not `.claude/skills`.
2. Codex persistent project guidance is `AGENTS.md`, not `CLAUDE.md`.
3. Codex project configuration, hooks, and custom agents live under `.codex/`,
   and project `.codex/` layers only load after the project is trusted.
4. Codex custom agents are TOML files under `.codex/agents/`, not freeform
   Markdown files under `.claude/agents/`.
5. Codex subagents are explicit. Codex does not spawn subagents automatically;
   the operator or main agent must ask for a subagent workflow.
6. Codex cloud/review cannot rely on gitignored, locally materialized files
   unless a cloud setup step materializes them. Tracked `AGENTS.md` is the
   reliable cloud-facing guidance surface.

The recommended direction is therefore staged:

1. Ship local Codex CLI/IDE support by materializing `aiwf-*`, `aiwfx-*`, and
   `wf-*` skills to `.agents/skills/`.
2. Add an `AGENTS.md` guidance writer, separate from the `CLAUDE.md` writer,
   with small tracked guidance suitable for Codex cloud/review.
3. Treat Codex custom agents as a later, lossy conversion from the current
   Markdown role agents to `.codex/agents/*.toml`.
4. Do not port the Claude statusline or Claude `Agent` hook directly. Codex has
   different extension points; use hooks only for Codex-native lifecycle checks
   after the need is proven.
5. Allow Claude and Codex adapters to coexist, but keep their generated files
   separate and make shared guidance authority explicit.

## Source basis

This audit is based on the current repository state and the current Codex
manual fetched on 2026-06-16.

Relevant Codex documentation:

- Codex skills use the open agent skills format and are discovered from
  `.agents/skills` in the current directory, parent directories, repo root,
  user home, admin, and system scopes. See the Codex manual sections
  [Agent Skills](https://developers.openai.com/codex/skills) and
  [Customization](https://developers.openai.com/codex/concepts/customization).
- Codex project instructions are `AGENTS.md`; discovery walks from repo root to
  current working directory and applies closer files later. See
  [Custom instructions with AGENTS.md](https://developers.openai.com/codex/guides/agents-md).
- Codex project config is `.codex/config.toml`, loaded only for trusted
  projects; CLI and IDE share config layers. See
  [Config basics](https://developers.openai.com/codex/config-basic).
- Codex hooks are configured through `.codex/hooks.json` or `[hooks]` in
  config; project-local hooks require trust review. See
  [Hooks](https://developers.openai.com/codex/hooks).
- Codex custom agents are standalone TOML files under `.codex/agents/` or
  `~/.codex/agents/`, with required `name`, `description`, and
  `developer_instructions`. See
  [Subagents](https://developers.openai.com/codex/subagents).
- Codex plugins use `.codex-plugin/plugin.json`; marketplaces may live under
  `.agents/plugins/marketplace.json`. See
  [Build plugins](https://developers.openai.com/codex/plugins/build).
- Codex managed worktrees live under `$CODEX_HOME/worktrees`, and ignored local
  files do not move into managed worktrees unless included through
  `.worktreeinclude`. See
  [Worktrees](https://developers.openai.com/codex/app/worktrees).
- `codex exec` exists for non-interactive/scripted workflows, but defaults to a
  read-only sandbox and has distinct automation safety guidance. See
  [Non-interactive mode](https://developers.openai.com/codex/noninteractive).

Repository evidence:

- ADR-0014 already names Codex `.agents/skills/` as the first non-Claude
  target shape and says the obstacle is output location, not skill format:
  `docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md`.
- G-0178 is the open gap to prove the Codex target:
  `work/gaps/G-0178-prove-a-non-claude-agent-target-codex-for-the-ritual-materializer.md`.
- `internal/skills/skills.go` has a `Target` abstraction and a production
  `ClaudeTarget`.
- `internal/skills/materialize_target_test.go` already proves a test-only
  Codex-shaped target at `.agents/skills`, `.agents/agents`, and
  `.agents/templates`.

## Compatibility target matrix

Codex is not one surface. A useful compatibility design should explicitly say
which Codex surfaces it supports.

| Surface | What needs to work | Minimum aiwf support |
|---|---|---|
| Codex CLI in local repo | Skills, project guidance, local commands | `.agents/skills` materialization plus `AGENTS.md` guidance |
| Codex IDE extension | Same local repo state as CLI | Same as CLI; CLI and IDE share config |
| Codex app local checkout | Same tracked repo guidance; local generated files if present | `AGENTS.md`; optional local `.agents/skills` |
| Codex app managed worktree | Fresh Git worktree under `$CODEX_HOME/worktrees` | Tracked guidance; materialized ignored files only if setup copies or regenerates them |
| Codex cloud task | Hosted checkout, no local gitignored generated adapters by default | Tracked `AGENTS.md`; cloud setup that installs/runs `aiwf update`, or committed/package-distributed skills |
| Codex code review | Review follows `AGENTS.md` guidance | Tracked `AGENTS.md` review guidance; skills are secondary |
| `codex exec` automation | Noninteractive task with explicit sandbox | `AGENTS.md`; optional command recipes or CI action patterns |

The local CLI/IDE slice is the smallest viable slice. Cloud/review support is a
different product question because gitignored generated adapters do not travel
with a clean remote checkout.

## Current Claude-specific surfaces

### Materialized adapter directories

Current production target:

- `.claude/skills/<skill>/SKILL.md`
- `.claude/agents/*.md`
- `.claude/templates/*.md`
- `.claude/aiwf-guidance.md`
- `.claude/statusline.sh`
- `.claude/settings.json` and `.claude/settings.local.json` for Claude Code
  statusline wiring
- `.claude/hooks/validate-agent-isolation.sh`

These are protected through `.gitignore` plus `.aiwf-owned` manifests. Most are
local generated state. That is appropriate for Claude Code local operation, but
it matters for Codex cloud: ignored files are absent unless regenerated.

### Root instruction file

The repository's primary operator doctrine is `CLAUDE.md`. It contains both:

- general aiwf doctrine that Codex should also know, such as "framework
  correctness must not depend on the LLM's behavior"; and
- Claude-specific operational details, such as `.claude/` locations,
  `CLAUDE_JOB_DIR`, Claude Code statusline behavior, Claude plugin index
  workarounds, and the Claude `Agent` hook.

Codex does not use `CLAUDE.md` as its primary documented instruction file.
Codex uses `AGENTS.md`. A Codex operator can read `CLAUDE.md` manually, but that
is not a compatibility story.

### Guidance injection

ADR-0018 authorizes automatic marker-managed edits to consumer `CLAUDE.md`,
adding an import line for `.claude/aiwf-guidance.md`. That is Claude-specific
in two ways:

- the filename is `CLAUDE.md`;
- the fragment path is `.claude/aiwf-guidance.md`.

The current Codex manual documents `AGENTS.md` discovery, but does not document
Claude-style `@file` imports in `AGENTS.md`. Therefore the Codex guidance path
should not assume imports. It should either write a small marker-managed block
directly into `AGENTS.md`, or write an `AGENTS.md` that points human readers to
tracked docs while keeping the operational rules inline.

### Skills

The embedded `aiwf-*`, `aiwfx-*`, and `wf-*` skills are the easiest piece.
Codex supports `SKILL.md` with `name` and `description` frontmatter and
progressive disclosure. The current skill bodies already satisfy the core
format.

Potential Codex-specific skill issues:

- Some skill descriptions mention Claude Code discovery or behavior.
- Some skill bodies cite `CLAUDE.md` as the standing authority.
- Some skill steps mention `.claude/worktrees`.
- Some skill steps mention "subagent-spawn mechanics are Claude Code surface."
- Some skill flows assume role agents as Markdown files under `.claude/agents`.

These do not block materializing skills to Codex, but they reduce quality. The
first Codex release can ship the same bodies with known caveats; a polished
Codex release should audit and host-neutralize the skill prose.

### Role agents

The current role agents are Markdown files:

- `.claude/agents/planner.md`
- `.claude/agents/builder.md`
- `.claude/agents/reviewer.md`
- `.claude/agents/deployer.md`

Codex custom agents are not Markdown files. Codex custom agents are TOML files
under `.codex/agents/` or `~/.codex/agents/` with at least:

- `name`
- `description`
- `developer_instructions`

Therefore the current `Target{AgentsDir: ".agents/agents"}` test fixture is not
the right production Codex shape. It proves the materializer seam, but not the
actual Codex agent format.

Codex also does not spawn subagents automatically. The operator must explicitly
ask for subagents or parallel agent work. That fits aiwf's sovereignty model,
but any ritual that relies on implicit role-agent selection needs rewriting.

### Templates

The current templates are materialized to `.claude/templates/*.md`. Codex has
skills and custom prompts, but the current manual says custom prompts are
deprecated in favor of skills. There is no direct documented Codex equivalent
for `.claude/templates`.

Recommendation: keep templates as aiwf-owned support files only if a Codex skill
uses them by path. A Codex target could materialize templates under
`.agents/templates/`, but that is an aiwf convention, not a Codex convention.
The skill body must explicitly read that path.

### Settings, hooks, and statusline

Claude:

- `settings.json` has a `statusLine` key.
- `.claude/statusline.sh` is an optional scaffold.
- `.claude/hooks/validate-agent-isolation.sh` blocks a Claude `Agent` tool
  misuse pattern.

Codex:

- settings live in `.codex/config.toml` or `~/.codex/config.toml`;
- hooks live in `.codex/hooks.json` or inline `[hooks]`;
- project `.codex/` config and hooks only load in trusted projects;
- hook trust is explicit for non-managed command hooks;
- the documented hook events are Codex lifecycle/tool events, not Claude Code
  `Agent` JSON payloads.

There is no direct "port the Claude statusline" step in the current Codex docs.
The statusline should remain Claude-only until Codex has a documented analogous
UI surface, or until aiwf defines a separate Codex-native status/reporting
workflow.

The Claude isolation hook should not be ported literally. Codex subagents
inherit sandbox policy, are explicitly requested, and have different lifecycle
hooks. A future Codex hook might enforce aiwf-specific subagent policy, but that
should be derived from Codex's `SubagentStart`, `SubagentStop`, `PreToolUse`,
or `PermissionRequest` events, not copied from Claude.

### Devcontainer and plugin workarounds

The devcontainer documentation and recent changelog entries include
Claude-specific plugin-index shadow-mount workarounds. These are not relevant
to Codex because the proposed Codex path should not depend on the retired
Claude marketplace channel.

Do not generalize those workarounds. Treat them as Claude compatibility debt,
not as an agent-agnostic substrate.

### Provenance actor strings

The kernel provenance model already supports open actor strings like
`ai/<id>`. Examples often use `ai/claude`. Codex compatibility does not require
a schema change. It requires documentation and examples to use `ai/codex` where
appropriate, and maybe a convention that host names are lowercase stable ids:

- `ai/claude`
- `ai/codex`
- `ai/cursor`
- `bot/github-actions`

Do not special-case Codex in the kernel unless a real permission or audit rule
needs it.

## Recommended target model

### Separate adapter targets from logical artifacts

The logical artifacts are:

- kernel verb skills (`aiwf-*`);
- lifecycle and engineering rituals (`aiwfx-*`, `wf-*`);
- role agents (`planner`, `builder`, `reviewer`, `deployer`);
- entity body templates;
- per-turn guidance;
- optional host integrations such as statusline and hooks.

The host adapters are:

- Claude Code: `.claude/skills`, `.claude/agents`, `.claude/templates`,
  `CLAUDE.md`, `.claude/settings*.json`, `.claude/hooks`.
- Codex: `.agents/skills`, `AGENTS.md`, `.codex/agents/*.toml`,
  `.codex/config.toml`, `.codex/hooks.json`.

The implementation should keep those as separate target writers. Avoid a
single generic directory layout like `.agents/agents` unless Codex documents it
as a host-native path. For Codex, `.agents/skills` is host-native;
`.codex/agents` is host-native; `.agents/templates` would be aiwf-native.

### Adapter selection

The likely operator surface is one of:

```bash
aiwf init --agent claude
aiwf init --agent codex
aiwf init --agent claude,codex
aiwf update --agent codex
```

or config:

```yaml
adapters:
  targets:
    - claude
    - codex
```

The config shape is better for "both active at the same time" because
`aiwf update` can keep both surfaces fresh without remembering flags. CLI flags
can still override for one-off materialization.

Backward compatibility suggests:

- default target remains `claude` for existing consumers;
- new consumers can opt into `codex` or `both`;
- a future major version could default to `auto` or `both` if the generated
  surfaces are low-risk.

### Gitignore policy

For local CLI/IDE parity with Claude, generated Codex adapter files should be
gitignored:

- `.agents/skills/aiwf-*/`
- `.agents/skills/aiwfx-*/`
- `.agents/skills/wf-*/`
- `.agents/skills/.aiwf-owned`
- `.agents/skills/README.md`
- optional `.agents/templates/*` if templates are materialized there
- `.codex/agents/*.toml` only if generated and marker-owned
- `.codex/hooks.json` only if generated and local-only

But `AGENTS.md` should be tracked when Codex cloud/review support matters.
`AGENTS.md` is the one Codex surface that reliably travels with clean hosted
checkouts.

This is an important asymmetry:

- local Codex skills can be generated and ignored;
- cloud/review guidance must be tracked, or regenerated in the cloud setup;
- committing generated skills would help cloud discoverability but would
  violate aiwf's current "materialized adapters are cache, not state" posture.

The first Codex milestone should choose explicitly which slice it targets:
"local Codex" or "local plus cloud."

## Staged path

### Stage 0: document the target split

No code change. Update docs to state:

- `CLAUDE.md` is a Claude adapter file, not the universal doctrine file.
- `AGENTS.md` is the Codex adapter file.
- `SKILL.md` is the portable skill source.
- `.codex/agents` is a separate conversion target, not the same as
  `.claude/agents`.
- cloud/review support depends on tracked guidance or setup regeneration.

This audit can be the seed. A later ADR should decide the exact target config.

### Stage 1: production Codex skill target

Implement `CodexTarget` for skills:

```go
var CodexTarget = Target{
    Name:         "codex",
    SkillsDir:    ".agents/skills",
    AgentsDir:    "",
    TemplatesDir: ".agents/templates",
}
```

This should probably set `AgentsDir` empty in the first production slice,
because Codex role agents require TOML conversion. Materializing Markdown agents
to `.agents/agents` would create files Codex does not document as loadable.

Acceptance checks:

- `aiwf init/update` can materialize both Claude and Codex skill sets.
- `aiwf doctor` can report Codex skill materialization separately from Claude.
- `.gitignore` gets Codex materialized patterns.
- No Codex run needs `.claude/` to discover `aiwf-*`, `aiwfx-*`, or `wf-*`
  skills.
- Skill frontmatter remains valid.

Quality sweep:

- Replace unnecessary "Claude Code" wording inside generic skills.
- Replace `CLAUDE.md` citations with "project guidance" where the rule is not
  actually Claude-specific.
- Preserve Claude-specific instructions only in Claude-only sections.

### Stage 2: Codex `AGENTS.md` guidance

Add a Codex guidance writer. Do not assume `@.claude/...`-style imports.

Recommended shape:

- Materialize a small marker-managed block directly into root `AGENTS.md`.
- Keep it short enough to respect Codex's default project-doc budget.
- Mention `aiwf check`, `aiwf status`, `aiwf list`, and the mutating-verb
  provenance discipline.
- Point to full docs for human readers, but do not rely on the model following
  a link it has not read.
- Add `guidance.wire_agentsmd: true|false` or a target-specific equivalent in
  `aiwf.yaml`.

Consent model:

- Use ADR-0018 as precedent, but revisit the risk. `AGENTS.md` is tracked,
  shared guidance used by Codex cloud/review. It is more cross-host than
  `CLAUDE.md`, not less.
- Default-on may still be justified, but the docs should say why.

Open design question:

- Should `AGENTS.md` become the primary host-neutral guidance file, with
  `CLAUDE.md` reduced to a Claude adapter that imports or mirrors common
  guidance? That is attractive, but it is a repo-doctrine migration, not just a
  Codex target writer.

### Stage 3: Codex custom agents

Convert the role agents to `.codex/agents/*.toml` only after the skill layer is
working.

Mapping:

```toml
name = "reviewer"
description = "Review code for correctness, security, regressions, and missing tests."
developer_instructions = """
<converted body from reviewer.md>
"""
```

Important differences:

- Codex custom agents are configuration layers for spawned sessions.
- They can override model, reasoning, sandbox, MCP, and skills config.
- Codex only spawns subagents when explicitly asked.
- The current Markdown role-agent bodies may be too broad for Codex custom
  agents; narrow them during conversion.

Potential generated files:

- `.codex/agents/planner.toml`
- `.codex/agents/builder.toml`
- `.codex/agents/reviewer.toml`
- `.codex/agents/deployer.toml`

Do not emit these as a blind Markdown-to-TOML dump without tests. The conversion
needs semantic review because role agents are an execution-policy surface.

### Stage 4: Codex hooks and config, only for proven needs

Codex hooks are powerful but have trust and project-loading semantics. They are
not needed for basic compatibility.

Candidate future hooks:

- `Stop`: remind or check that `aiwf check` has been run after entity edits.
- `PreToolUse` or `PermissionRequest`: block obviously wrong destructive git
  commands in aiwf-managed branches.
- `SubagentStart`: enforce aiwf branch/scope discipline once Codex subagent
  orchestration is actively supported.

Do not port:

- Claude statusline settings.
- Claude `Agent` kwarg isolation hook.
- Claude plugin-index workarounds.

### Stage 5: Codex plugin packaging, if distribution needs it

Direct `.agents/skills` materialization is enough for local repo compatibility.
Codex plugins become useful when aiwf wants a reusable installable package
outside `aiwf init/update`, or when skills should be bundled with MCP servers,
hooks, app integrations, or marketplace metadata.

If pursued, a Codex plugin package needs:

- `.codex-plugin/plugin.json`
- `skills/` folder
- optional MCP/hook/assets declarations
- optional repo marketplace under `.agents/plugins/marketplace.json`

This should not replace embedded materialization in the first Codex slice.
ADR-0014 deliberately moved away from a marketplace-only distribution channel.

### Stage 6: cloud/review support

Codex code review follows `AGENTS.md`. It should not depend on generated
gitignored skills.

For Codex cloud tasks, choose one:

1. Track minimal `AGENTS.md` only and let cloud tasks use the `aiwf` CLI if
   installed by environment setup.
2. Add a cloud setup step that installs `aiwf` and runs `aiwf update --agent
   codex`.
3. Commit generated `.agents/skills` files, accepting that adapter files become
   tracked state.
4. Use a Codex plugin/marketplace path.

Option 1 is the least invasive. Option 2 is probably the best long-term fit for
cloud tasks. Option 3 conflicts with the current cache-not-state posture.
Option 4 is useful for distribution, but not necessary for a repo that already
has `aiwf`.

## Can Claude and Codex be active at the same time?

Yes, with caveats. There is no inherent file-level conflict if the adapters are
kept separate:

- Claude reads `.claude/*` and `CLAUDE.md`.
- Codex reads `.agents/skills`, `AGENTS.md`, and `.codex/*`.
- Git hooks installed by `aiwf` are shared and host-agnostic.
- The planning tree under `work/` is shared state, as intended.

The caveats are operational:

1. Shared doctrine can diverge. If `CLAUDE.md` says one thing and `AGENTS.md`
   says another, each agent may behave "correctly" according to its own file
   while violating the project intent.
2. Git branches can conflict. Codex-managed worktrees, Claude-managed
   worktrees, and manually created worktrees all share Git's one-branch-per-
   worktree rule.
3. Generated ignored files do not follow between checkouts unless regenerated.
   This affects both `.claude` and `.agents` adapters.
4. Hooks are host-specific. A Claude hook cannot police Codex behavior, and a
   Codex hook cannot police Claude behavior.
5. Subagent semantics differ. Claude role agents and Codex custom agents are not
   the same abstraction.

The safest coexistence model:

- Treat `work/`, `aiwf.yaml`, `docs/`, `README.md`, and tracked guidance as
  shared source of truth.
- Treat `.claude/` and `.agents/` as local generated adapter caches.
- Put common behavior rules in a small tracked common guidance source, then
  generate host-specific wrappers from it.
- Make `aiwf doctor` report adapter status per host:
  `claude: ok`, `codex: missing skills`, `codex: AGENTS.md unwired`, etc.

## Risks

### Risk: two instruction files drift

Codex needs `AGENTS.md`; Claude currently needs `CLAUDE.md`. Copy-pasting common
rules into both files creates drift.

Mitigation: define a common aiwf guidance source and generate host-specific
blocks. If that is too much for the first slice, keep the Codex `AGENTS.md`
block tiny and point most operational detail to skills and CLI help.

### Risk: generated Codex skills are absent in cloud

Gitignored `.agents/skills` works locally but not in a clean hosted checkout.

Mitigation: distinguish local support from cloud support in docs. For cloud,
use tracked `AGENTS.md` plus setup regeneration or plugin distribution.

### Risk: Markdown role agents are mistaken for Codex agents

The existing materializer test writes `.agents/agents/*.md`, but Codex custom
agents live in `.codex/agents/*.toml`.

Mitigation: production `CodexTarget` should set `AgentsDir` empty until a
Codex-specific agent converter exists.

### Risk: host-specific hooks create false confidence

The Claude `validate-agent-isolation.sh` hook is about a Claude `Agent` tool
failure mode. Porting it by path would not protect Codex.

Mitigation: keep hooks host-specific and only add Codex hooks derived from
Codex lifecycle events.

### Risk: "agent agnostic" becomes lowest-common-denominator

Codex and Claude share skills, but differ in agents, config, hooks, UI, cloud,
and review behavior. A single flattened abstraction could erase useful host
capabilities.

Mitigation: use a host adapter model. Share source artifacts where formats are
actually shared; write host-native adapters where formats differ.

## Proposed acceptance criteria for a first Codex milestone

A narrow first milestone should target local Codex CLI/IDE compatibility:

1. `aiwf init/update` can materialize Codex skills to `.agents/skills`.
2. The materialized Codex skill set includes all embedded `aiwf-*`, `aiwfx-*`,
   and `wf-*` skills.
3. `.gitignore` gains marker-managed Codex generated-adapter patterns.
4. `aiwf doctor` reports whether Codex skills are materialized and in sync.
5. No production Codex target writes `.agents/agents/*.md`.
6. No production Codex target writes `.codex/config.toml` or hooks by default.
7. Documentation says this is local Codex support, not full cloud support.
8. Existing Claude behavior is unchanged.

A second milestone should target Codex guidance:

1. `aiwf init/update` can maintain a marker-managed block in `AGENTS.md`.
2. The block is concise and does not assume import syntax.
3. `aiwf.yaml` can opt out.
4. `aiwf doctor` reports unwired Codex guidance separately from Claude
   guidance.
5. The root docs explain precedence between `CLAUDE.md` and `AGENTS.md`.

Only after those should Codex custom agents or hooks be attempted.

## Open questions

1. Should adapter targets be configured in `aiwf.yaml`, flags, or both?
2. Should new repos default to `claude`, `codex`, or `both` once Codex support
   exists?
3. Should `AGENTS.md` become the host-neutral primary guidance file for aiwf
   itself, with `CLAUDE.md` as a Claude compatibility wrapper?
4. Should Codex skills remain gitignored local caches, or should there be an
   optional committed-skills mode for cloud environments?
5. Should the `wf-*` rituals be packaged as a Codex plugin independent of
   aiwf, preserving their repo-agnostic value?
6. Should role agents be host-neutral source files with generated Claude
   Markdown and Codex TOML outputs?
7. What is the authoritative setup story for Codex cloud tasks that need the
   `aiwf` binary installed?
8. Should `aiwf authorize --to ai/codex` gain examples or convenience docs, or
   remain just an open actor string?

## Bottom line

Codex compatibility is feasible and should start with the boring part:
materialize the existing skills to `.agents/skills`. That validates the
agent-target seam ADR-0014 already created and closes the open G-0178 proof
without destabilizing the Claude path.

Do not call that full agent-agnostic support. Full support requires a Codex
guidance story (`AGENTS.md`), a cloud/review story, and a host-native answer for
custom agents and hooks. The safest direction is a multi-target adapter model:
shared `SKILL.md` where the standard is shared, host-native generated files
where it is not.
