# Where state lives — in-repo, out-of-repo, or layered, and which is more successful

> **Status:** research / location-of-truth analysis. Not normative. Sixth in the series; reads on top of `00`–`04` and `KERNEL.md`.
> **Audience:** the user, after they pointed at a paper, asked whether there is empirical evidence about in-repo vs. out-of-repo planning state, and proposed `brew install aiwf` with agents/skills materialized into the repo (gitignored) and the framework being IDE- and PM-agnostic.
> **Premise:** "in repo or out of repo" is not one question but five questions stacked. The right answer puts each layer where its constraints are best served, and most of the user's instinct is right for some layers and wrong for others.

---

## 1. The paper the user pointed at

DOI `10.1145/3746059.3747646` resolves to:

**Tomas Petricek, "Denicek: Computational Substrate for Document-Oriented End-User Programming"** — UIST '25, the 38th ACM Symposium on User Interface Software and Technology.

What it actually is: a substrate that represents a program as **a series of edits that construct and transform a document consisting of data and formulas.** It is intended to make hard-to-build end-user programming experiences (collaborative editing, programming by demonstration, incremental recomputation, schema change control, end-user debugging, concrete programming) easier to implement.

What it tells us, indirectly:

- The lineage of "represent state as a sequence of edits over a document" continues to attract serious research. This is the same family as event sourcing, CRDTs, and patch-theory VCS, all of which surfaced earlier in this research series.
- "Schema change control" is named explicitly as one of the experiences Denicek supports — this is exactly the "plans change while implementing" problem the user has been articulating since `02`.
- The substrate is *agnostic to where the document lives*. The paper does not adjudicate in-repo vs. out-of-repo storage; it gives you primitives that work either way.

What it does *not* tell us, despite proximity:

- Whether project state for AI-coding teams is more successful in the repo or outside it.
- How a real team handles branching of a Denicek-style document.
- Anything about the CI / multi-machine / multi-user trade-offs that dominate the user's actual question.

It is good support for the **edits-over-document** representation choice (relevant to §5 of `04` and to any module that adopts CRDT primitives), but it is not on point for the location question. A worthwhile read; not a settling reference.

---

## 2. Is there empirical evidence about which model is more successful?

Honest scope note before answering: I will name what I know about with confidence, mark what I am inferring rather than citing, and explicitly flag the absence of solid empirical work.

### 2.1 The empirical literature is thin

I know of **no peer-reviewed study that directly measures the success rate of in-repo vs. out-of-repo project state** for AI-assisted software teams. The closest adjacent work:

- **Documentation-as-code studies** (Anne Gentle's writing; the rise of MkDocs / Docusaurus / docs-in-repo conventions) — measure documentation freshness and contributor velocity but with prose docs, not structured planning state. Findings generally favor in-repo for *technical* docs that change with code.
- **ADR adoption studies** — community case studies and trade-press reports (ThoughtWorks Tech Radar, various "we adopted ADRs" blog posts) consistently report ADRs-in-repo as net positive. None compare against an out-of-repo equivalent rigorously.
- **Spec-driven development tooling** (Spec Kit, Kiro, Tessl) — too new to have empirical literature; tool authors' marketing claims are not evidence.
- **Project-management tool effectiveness literature** (Atlassian, Forrester, McKinsey) — measures Jira/Linear/Asana adoption, but these are external-state tools by definition and there is no in-repo control.

### 2.2 Why solid evidence is hard to gather

- **Team size is a confound.** Teams that adopt heavy in-repo planning are usually small. Teams using Jira are usually large. You cannot tease apart "the tool worked better" from "the team was smaller."
- **Selection effects everywhere.** A team that adopts ADRs is already a team that values written reasoning. The result is endogenous.
- **Outcome is hard to define.** What's success? Velocity? Code quality? Team satisfaction? Audit-readiness? Different choices give different winners.
- **Time scales are long.** A planning-tool decision pays off (or doesn't) over months to years. Studies of that horizon are rare.
- **The AI angle is brand new.** Every empirical claim about "AI coding assistants and their context" is from the last 18 months and lacks the longitudinal data needed.

### 2.3 What we can say

In place of empirical findings, here are observed patterns from the field as of April 2026:

- **Long-horizon open-source projects almost universally keep architectural decisions in-repo** (ADRs, CONTRIBUTING.md, governance docs). They keep operational tickets external (GitHub Issues, Discussions). The split is by *kind of artifact*, not by tool ideology.
- **Solo and small-team AI-coding projects** trend toward all-in-repo (CLAUDE.md, in-repo specs, ROADMAP.md). The friction of an external system is high relative to the value for one person.
- **Mature companies with regulatory exposure** keep audit-bearing artifacts in systems with formal access controls and immutability guarantees (Jira, ServiceNow, Confluence with retention policies). They sometimes mirror summaries to the repo.
- **The fastest-growing AI-coding tools** (Cursor, Claude Code, Copilot, Aider, Continue) all add per-repo configuration files, which is empirical evidence of a kind — every tool reaches independently for in-repo configuration as the necessary surface, even when the rest of the system is external. The ergonomic pull toward in-repo *for some things* is consistent.

The pattern is not "in-repo wins" or "external wins." The pattern is **stratification**: different layers of the system gravitate toward different locations based on their constraints, and the successful designs explicitly stratify rather than picking one location for everything.

This is the answer to the user's question. The choice is not binary; it is layered. Section §3 unpacks the layers.

---

## 3. The five layers — what's actually at stake when we say "where does state live?"

Treating "where state lives" as one question conflates five layers with very different constraints:

| Layer | What it is | Update frequency | Cross-machine? | CI needs it? | Team-shared? |
|---|---|---|---|---|---|
| L1: Engine binary | The `aiwf` executable itself | At install/upgrade | Yes (each machine) | Yes (CI installs it) | Yes (everyone has it) |
| L2: Per-project policy | Which modules enabled, conventions, contracts, governance rules | Rare | Travels with project | Yes (CI reads policy) | Yes (team-wide) |
| L3: Per-project planning state | Epics, milestones, ADRs, decisions, gaps, roadmap | Daily during active work | Travels with project | Yes (CI validates) | Yes (team-wide) |
| L4: Per-developer ergonomic state | Local preferences, in-flight scratch, view filters | Per-session | Per-machine | No | No |
| L5: Skill content (multi-source) | Framework / project / installable skills (sources for adapter materialization) | On framework upgrade, project edit, or skill install | Mixed by source | Yes (CI may run skill-aware checks) | Mixed by source |
| L6: Materialized adapters + runtime cache | The composed `.claude/skills/wf-*` etc., indices, hashes | On `aiwf init` / `aiwf update` only | Per-machine | Regenerated | No |

The user's `brew install aiwf` proposal is a clear win at L1, correct at L4, and a clean partial answer at L5/L6 — but the original framing of "skills materialize from the binary" was too tight. Skills come from multiple sources; only the framework-skill subset is binary-bundled. §3.5 is rewritten below to reflect this.

Let me walk each.

### 3.1 L1 — The engine binary

**Where it should live: outside the repo, on the machine, package-managed.**

This is the right answer and the framework currently mishandles it. The architecture's "framework as a git submodule under `.ai/`" pattern (per `architecture.md` §11) was a workaround for not having a build/distribution story. With a real binary distributed via brew, apt, scoop, or a `go install` step, the submodule disappears.

Properties this gets right:

- The binary is versioned per-machine, like every other developer tool (git, ripgrep, jq, gh).
- Upgrade is `brew upgrade aiwf`, like every other tool.
- CI installs the binary in its container setup, like every other tool.
- The repo carries no copy of the framework's source.
- The "framework version diverges across branches" problem (from `00` and `04`) **disappears at this layer** because the binary version is a property of the machine, not the branch.

The only subtlety: the binary's *required version* is a per-project policy decision (L2), so the repo needs *something* that says "this project requires aiwf >= 0.7." That something is a few lines of YAML, not the binary itself.

### 3.2 L2 — Per-project policy

**Where it should live: in the repo, small, YAML.**

This is the layer the user's proposal under-specifies. "PM-agnostic" and "process-agnostic" are properties; they still need to be *configured* somewhere, and that somewhere is per-project, team-shared, and travels with clones.

What L2 contains:

- Which modules are enabled (`epics-milestones`, `adr`, `gh-sync`, `crdt-registry`, etc.).
- Which conventions apply (id format, branch-naming convention, FSM rules per kind).
- Which external systems sync (`gh`, `linear`, `jira`, or none).
- Which governance rules apply ("milestone promotion to complete requires two reviewers").
- Minimum required `aiwf` version.

Why in-repo: every developer on the team needs the same policy; CI needs it; new clones need it without separate setup; PR review needs it. Without an in-repo file, there's no shared truth about what the framework should be doing.

This is small (~50–200 lines of YAML for a typical project) and changes rarely. It is *not* the heavy planning state; it is the configuration of the engine for this project.

A minimal `aiwf.yaml` (or `.ai-repo/config.yaml`) at the project root.

### 3.3 L3 — Per-project planning state

**This is the actually contested layer.** The user's proposal puts it outside the repo. The prior research argued for in-repo markdown. Walk both, ruthlessly.

#### Case for L3 outside the repo

- No git fight. The state never enters git, so merge conflicts on planning state are impossible.
- No "framework state diverges across branches" problem at L3 either.
- No tombstone bookkeeping required to handle deletions (the external store can just delete; no append-only constraint).
- Schema migrations are easier (one place to migrate, no per-branch versions in flight).
- Easier to share state across multiple repos for the same logical project (rare but real for some teams).

#### Case for L3 in the repo

- **Co-evolution with code.** A PR can land "M-005 done" and the code that closes it in the same atomic unit. CI can verify the link.
- **Bisectability of the plan.** `git checkout HEAD~50` shows the plan as it was 50 commits ago. With external L3, the plan is current; you cannot reconstruct what was believed-then.
- **Multi-machine without sync.** State travels with `git clone`. No additional sync layer for the developer to operate.
- **Multi-user without a server.** `git push` / `git pull` is the sync. The framework does not become a multi-master sync engine.
- **CI reads it for free.** Fresh container with the repo cloned has full state. No credentials, no API quotas, no flaky network.
- **Provenance ties to commits.** Structured commit trailers (`aiwf-verb:`, `aiwf-entity:`) link state changes to commits in a way external state cannot.
- **AI working set is one filesystem.** When the assistant is implementing M-005, it reads M-005's spec from the same checkout as the code, no API call.

#### Where they break down

The case for outside-the-repo collapses on these facts:

- **Project identity.** "This project" needs an id that is stable across clones, forks, transfers, and renames. `git config remote.origin.url` is fragile (forks, transfers); the repo path is fragile (move the directory); a `.aiwf-project-id` file is back to in-repo. There is no clean answer that doesn't put *something* in the repo.
- **Multi-machine without a server.** Engineer works on desktop and laptop. State on desktop only. Solutions: a sync service (which is a server, the thing we were avoiding), Dropbox-like external sync (fragile, partial-sync issues, conflict resolution still needed), or manual rsync (terrible UX). For the brew-install model to work multi-machine, *some* server is required somewhere.
- **Multi-user without a server.** Two engineers on the same project. Each has their own local state. To collaborate they must sync. You are now building Linear from scratch but worse, because Linear has had years to harden.
- **CI in fresh containers.** CI clones the repo, installs the binary, and runs `aiwf check`. With L3 in-repo, this works. With L3 external, CI must fetch state from somewhere — credentials, network, race conditions, vendor lock-in, API rate limits.
- **Bisectability.** Permanently lost. A code regression that traces to "we changed our minds about this milestone two months ago" cannot be bisected if the milestone state is current-only.
- **Compliance for regulated industries.** "Show me the planning state at the time this commit was authored" is a compliance question some teams must answer. Out-of-repo current-only state cannot.

#### The honest verdict on L3

For a **solo developer on a single machine**, out-of-repo is *almost* viable, and the convenience is real. For anything else, the costs of out-of-repo dominate. And designing the framework for solo-on-single-machine and then trying to scale up later is the bad direction; designing for the harder case and degrading gracefully to the easy one is the good direction.

**L3 belongs in the repo.** This is the conclusion the prior research arrived at and the user's proposal does not (yet) overturn.

### 3.4 L4 — Per-developer ergonomic state

**Where it should live: outside the repo, in the developer's home directory or XDG config.**

What L4 contains:

- Local preferences ("show me ROADMAP filtered to my-assigned milestones").
- In-flight scratch ("I'm thinking about this; not ready to commit").
- View filters, sort orders, color schemes.
- Personal credentials for external integrations.

This is uncontroversial. Per-developer state stays per-developer. `~/.config/aiwf/` (or `$XDG_CONFIG_HOME/aiwf/`).

### 3.5 L5 — Skill content from three distinct sources

The earlier framing — "skills materialize from the binary" — was too narrow. Skills are not a single category. The user pointed out three reasons the binary alone can't be the source of truth: skills must evolve independently of the framework binary because LLMs evolve; they need configurability for local needs (different OS, different tool paths, different environments); and projects legitimately have their own skills that no upstream framework knows about.

So L5 splits into three sources, each with different semantics:

**L5a — Framework skills.** Bundled with the `aiwf` binary. Covers the verbs the framework itself provides: `wf-add-milestone`, `wf-promote`, `wf-verify`, `wf-prepush`, etc. These evolve with the binary; `brew upgrade aiwf` updates them. They are the universal core.

**L5b — Project skills.** Written by the team for this project. Live **in-repo, git-tracked**, in a known directory (`.ai-repo/skills/` or similar). Examples: a project-specific code-review skill that knows the team's conventions; a deployment runbook expressed as a skill; a domain-specific skill for the product's quirks. These travel with the repo and evolve with the project. They *can* differ across branches (a feature branch experimenting with a new approach can introduce a project skill); see "branch survival" below for the resolution.

**L5c — Installable third-party skills.** Pulled from a registry on demand: `aiwf install skill postgres-db@1.2`, `aiwf install skill react-testing-library`. Versioned independently of both the framework and the project. Cached locally (`~/.cache/aiwf/skills/`); declared and pinned in the project's `aiwf.yaml` + `aiwf.lock` so every developer and CI gets the same versions. Like npm packages, but for skills.

And a fourth axis cuts across all three:

**L5d — Local skill configuration.** Per-machine overrides for tool paths, environment defaults, OS-specific bits. The postgres skill might need to know whether the local client is `psql`, `pgcli`, or `docker exec`. The deploy skill might default to a different environment on a developer machine vs. CI. These overrides live at `~/.config/aiwf/skill-local.yaml`, never in-repo. They patch the materialized adapters at composition time.

### 3.6 L6 — Materialized adapters

The end-state files the AI host actually reads — `.claude/skills/wf-*`, `.github/skills/<name>/`, the host-shaped equivalents — are **the composed result** of L5a + L5b + L5c with L5d overrides applied. They are derived. They are gitignored. They are regenerated only by an explicit `aiwf init` or `aiwf update` step.

#### Branch survival — the load-bearing invariant

The user named the right invariant: **the materialized adapter set should stay the same regardless of what branch is checked out.** This is not an automatic consequence of how adapters are stored; it is an active design decision about *when* materialization runs.

Materialization runs on:
- `aiwf init` (first time in this checkout)
- `aiwf update` (explicit, deliberate)
- Detection of a stale lockfile (with prompt, not auto-apply)

Materialization does **not** run on:
- `git checkout` of another branch
- `git pull`
- Implicitly during normal `aiwf` verb invocation

This matches how every other dev tool works. You don't reinstall your IDE on every branch switch. You don't reinstall npm packages unless `package.json` changed and you ran `npm ci`. The framework follows the same pattern: a project-level manifest (`aiwf.yaml`) and lockfile (`aiwf.lock`) declare the resolved skill set; an explicit update step rematerializes; no automatic regeneration on checkout.

When a branch *legitimately* needs a different project-skill set (the feature branch experimenting with a new review approach), the developer runs `aiwf update` on that branch consciously. The framework can detect this drift via `aiwf doctor`, which reports when materialized adapters disagree with the resolved manifest on the current branch, and offers to update.

This means: branches *can* diverge in their skill *source* (L5b is git-tracked, so source files do switch with the tree). But the *materialized adapters* the AI sees do not switch automatically. The default behavior is "skills are stable across checkouts"; divergence is opt-in and deliberate.

#### The manifest and lockfile

For composition to be deterministic across machines:

- **`aiwf.yaml`** declares: minimum aiwf version, enabled framework modules (selecting which L5a skills are wanted), installed third-party skills with version ranges (L5c), and the path to project-skill source (L5b, default `.ai-repo/skills/`).
- **`aiwf.lock`** pins: exact resolved versions and content hashes for L5c, plus a hash of the resolved skill set as a whole.
- **`aiwf update`** resolves the manifest, updates the lockfile, materializes adapters into L6.

CI runs `aiwf check --skills-current` to verify the lockfile resolves cleanly with the binary version available. If it doesn't, CI fails with a clear message: which skill, which version, what's the mismatch.

#### Properties this gets right

- Framework skills evolve with the binary — `brew upgrade aiwf && aiwf update` brings in new framework verbs without project intervention.
- Project skills evolve with the project — git-tracked, reviewed in PRs, can be team-specific.
- Third-party skills evolve independently — like any package dependency.
- Local environment differences are handled where they belong — per-machine config — without polluting the team-shared layers.
- Materialized adapters are stable across branches by default — the AI's behavior doesn't change when you `git checkout` to verify something on main.
- Materialized adapters are reproducible — same manifest + lockfile + binary version = same adapters.
- Skill divergence across branches is *possible* but *deliberate* — when a branch introduces project-skill changes, `aiwf update` is the explicit step.
- Cross-developer drift is detectable — `aiwf doctor` and CI both check.

#### What this resolves and what it doesn't

Resolves:
- The current architecture's coupling of skill content to git (skills committed in `.claude/skills/wf-*`) and the AI-rule-divergence-across-branches problem from `04` §3.3 / §5.
- The user's concern that skills must evolve independently of the framework binary — they do, via L5b (project) and L5c (third-party).
- The local-tool-difference concern — handled in L5d, never team-shared.

Does not fully resolve:
- Different `aiwf` binary versions across team members still produce different L5a content; `aiwf doctor` and CI's required-version check are the mitigation.
- A project skill (L5b) that depends on a third-party skill (L5c) needs a clear declaration mechanism in `aiwf.yaml`; the design of that is open (§12).
- Whether skill *capabilities* (the verbs they expose) should match across hosts (Claude Code, Cursor, Copilot) is a downstream question — the framework provides the content; each host's materializer shapes it for that host. Some loss of fidelity is inevitable across hosts.

---

## 4. The user's proposal, scored layer by layer

| Layer | User's proposal | Verdict |
|---|---|---|
| L1: Engine binary | `brew install aiwf` | ✅ Correct. Replace the submodule pattern. |
| L2: Per-project policy | Implicit; under-specified | ⚠️ Must be in-repo. Small YAML config (`aiwf.yaml`) plus lockfile (`aiwf.lock`). |
| L3: Per-project planning state | Outside the repo | ❌ Breaks on multi-machine, multi-user, CI, bisectability, regulation. Stay in-repo. |
| L4: Per-developer ergonomic | (Implicit) outside | ✅ Correct. |
| L5a: Framework skill content | Bundled with binary, regenerated | ✅ Correct as a *subset* of skills, not all of them. |
| L5b: Project skill content | (Not in original proposal) | ✅ In-repo, git-tracked, evolves with the project. Added by user's revision. |
| L5c: Installable third-party skills | (Not in original proposal) | ✅ Cached locally, declared and pinned in `aiwf.yaml` + `aiwf.lock`. Added by user's revision. |
| L5d: Local skill config | (Not in original proposal) | ✅ Per-machine `~/.config/aiwf/skill-local.yaml`. Added by user's revision. |
| L6: Materialized adapters | "skills (versioned, gitignored) in my repo folder" | ✅ Correct and novel. Composed from L5a/b/c with L5d overrides; gitignored; only regenerated on explicit `aiwf init` / `aiwf update`. Stable across branch switches by design. |

The proposal is right at L1, L4, L5a, L6; under-specified at L2; wrong at L3. With the user's revision (separating skill sources L5a / L5b / L5c / L5d), the picture is more honest about how skills actually evolve, while still preserving the most important property: **the materialized adapter set the AI host reads is stable across branch switches** because materialization is an explicit step, not an implicit consequence of checkout.

The interesting move: **the proposal's strengths are at L1, L5a/b/c (each in its own way), and especially L6.** The framework adopts the multi-source skill model and the explicit-materialization invariant without adopting the L3 stance.

---

## 5. The "isn't this how LLMs work?" intuition

The user wrote: *"Of course we already have a lot which is outside the repo (many of the tools themselves are on the host OS or the container OS and located by path and versioned). What is so wrong about having the framework being installed as a tool just like any other tool via brew or apt etc and being always available? And isn't that how even LLMs work?"*

This intuition is right for L1 and not for L2/L3. Walk it precisely:

- **The LLM itself is external** — the model weights are on Anthropic's servers, called via API. Yes.
- **The LLM's invocation context** (system prompt fragments, rules, tools available) is partly external (what the host like Claude Code adds) and partly **in-repo** (`CLAUDE.md`, `.claude/skills/`, `.cursorrules`, `.github/copilot-instructions.md`). This is empirical: every major AI coding host has converged on a per-repo configuration file. **Even LLMs don't keep their per-project configuration external.**
- **The LLM's per-project planning state** is whatever the human or the host gives it. Today there is no good answer — which is exactly why this framework is being designed.

So the analogy supports "framework binary external" (L1) but actively undermines "framework state external" (L3). The empirical pattern across every AI coding tool in 2026 is: external binary, in-repo configuration. The framework should follow that pattern.

---

## 6. The successful real-world pattern

Restating §2.3 as a positive design statement, the convergent pattern from successful adjacent tools is:

| Layer | Where it lives | Why |
|---|---|---|
| Binary / engine | External (PATH, package-managed) | Standard tool distribution |
| Per-project policy / config / lockfile | In-repo (small YAML/JSON) | Team-shared, CI-readable, travels with clone |
| Per-project content (planning, decisions, specs) | In-repo (markdown, structured) | Co-evolves with code, bisectable, CI-readable |
| Project-specific skills | In-repo (markdown) | Team's own conventions; evolves with project |
| Per-project content (operational tickets, comments, sprints) | External (GitHub/Linear/Jira) — *if* the project uses one | Multi-user, notifications, mature UX |
| Framework skill content | External (binary-bundled) | Evolves with the engine |
| Third-party skill content | External (registry, cached locally, lockfile-pinned) | Independent versioning; like dependencies |
| Per-developer state + skill local config | External (XDG config) | Personal, no team coupling |
| Materialized skill adapters + runtime cache | External (XDG cache or gitignored repo path) | Composed from above; regeneratable; machine-specific; stable across branch switches by design |

This is **not novel** in its broad strokes — successful tools have arrived at this stratification independently — but the user's revision adds two things that are genuinely good design contributions: separating the three skill sources (framework / project / third-party / local-config), and making the materialized adapter set stable across branch switches via explicit-only `aiwf update`. Adopt those. Keep the rest of the pattern.

---

## 7. The sub-question: IDE-agnostic, PM-agnostic

The user emphasized that the framework should be IDE-agnostic and PM-agnostic. Both are good goals; both have specific design implications.

### 7.1 IDE-agnostic

The framework must not require any specific IDE. Test: can a developer do everything via CLI from a vanilla terminal? If yes, IDE-agnostic. If anything is IDE-only, that's a leak.

Implications:

- All verbs are CLI-first. IDE integration is a thin wrapper over the CLI.
- Skills are text files that any AI host can read (Claude Code, Cursor, Copilot, Continue, Aider). The framework provides skill *content* and lets each host's adapter pattern materialize it.
- No required keybindings, no required language servers, no required extensions.

The current architecture mostly satisfies this; the L5 skill-materialization shift makes it cleaner (each host gets its adapter shape from the binary).

### 7.2 PM-agnostic

The framework must not require Linear, Jira, GitHub Projects, or any specific external system. Test: can a project use the framework with no external PM at all? If yes, PM-agnostic.

Implications:

- Planning state is in-repo. External PM is opt-in via sync modules.
- The kernel works with no network, no credentials, no API keys.
- Sync modules (`gh-sync`, `linear-sync`, etc.) are clearly opt-in and clearly bidirectional or unidirectional as configured.
- The framework defines its own vocabulary (epics, milestones, decisions, etc.) that maps onto external systems via the sync modules — not the other way around.

This is consistent with the prior research and reinforces the L3-in-repo conclusion. Out-of-repo L3 with PM-agnostic = "we have our own PM tool now," which is the trap §3.3 warned about.

---

## 8. The "but what about the in-between" question revisited

`04` proposed the Pre-PR Workshop as a first-class tier. With the layer model from this document, that gets sharper:

- **Pre-PR Workshop** is mostly L3 work (validating planning state) + some L5 work (regenerating skill adapters if the binary or config changed). The work happens locally with the binary on PATH.
- **PR Workshop** is the same checks rerun in CI, plus async review.
- **Museum** is `main` after merge, sealed.

The package-managed binary at L1 makes Pre-PR Workshop tooling much cleaner: the developer's local `aiwf prepush` is bit-for-bit the same code as CI's `aiwf check`. No "did I have the right submodule pinned" friction. Same binary, same answers.

---

## 9. Implications for the framework's near-term plan

If the user adopts this layered view, several concrete changes follow:

### 9.1 Switch L1 from submodule to binary distribution

- Build `aiwf` as a single static Go binary (already the direction; `tools/cmd/aiwf/`).
- Distribute via brew tap, apt repo, `go install`, and as a downloadable binary from GitHub Releases.
- Retire the `framework/` submodule pattern from `architecture.md` Appendix A.
- Replace it with: at install time, the binary writes a tiny `aiwf.yaml` to the project root; everything else (skills, contracts) is shipped in the binary or fetched from per-project config.

### 9.2 Move materialized skill adapters out of git; keep skill *sources* layered

- The binary bundles framework skills (L5a). `aiwf` ships them.
- Project skills (L5b) live in `.ai-repo/skills/` (or similar) — git-tracked. Teams write their own here.
- Third-party skills (L5c) install from a registry: `aiwf install skill <name>@<version>`. Cached locally; declared in `aiwf.yaml`; pinned in `aiwf.lock`.
- Local skill config (L5d) lives at `~/.config/aiwf/skill-local.yaml` for per-machine tool paths and environment defaults.
- Materialized adapters (L6) — `.claude/skills/wf-*`, `.github/skills/<name>/`, etc. — are composed from L5a/b/c with L5d patched in. Gitignored. Written by `aiwf init` (first time) and `aiwf update` (subsequently).
- `.gitignore` for materialized adapter paths is added by `aiwf init`.
- **Materialization runs only on `aiwf init` and `aiwf update`**, not on `git checkout` or every verb invocation. This is the load-bearing invariant: AI behavior is stable across branch switches because the adapters the host reads don't change implicitly.
- `aiwf doctor` reports drift: lockfile out of sync with manifest, materialized adapters older than declared sources, binary version below the project's required minimum.
- Skill divergence across branches is *possible* (a feature branch can edit `.ai-repo/skills/`) but not *implicit* — divergence only takes effect after `aiwf update` runs on that branch.

### 9.3 Define L2 explicitly

- A single `aiwf.yaml` at project root.
- Contains: required binary version, enabled framework modules, declared third-party skills with version ranges, path to project skills directory, conventions, governance rules, sync configuration.
- Companion `aiwf.lock` pins exact resolved versions and content hashes for third-party skills, plus a hash of the resolved skill set.
- Both tracked in git, reviewed in PRs, small and stable.

### 9.4 Keep L3 in-repo, as prior research concluded

- No change from `01` / `03` / `04`'s direction.
- Markdown specs as canonical.
- CRDT-modeled metadata layer (per `04` §5) for the merge-sensitive parts.
- Tombstones, stable ids, structured commit trailers.

### 9.5 Cache directory

- `~/.cache/aiwf/<project-hash>/` for derived/cached state.
- Project hash from absolute repo path or from `aiwf.yaml`'s declared project id.
- Wipeable safely; auto-regenerated.

This is a meaningful but contained change to the architecture: a redistribution between layers, not a rethink of what the framework is.

---

## 10. What this changes about prior research

- **`00-fighting-git.md`**: still valid where it is. The new layer model further isolates the "fighting git" risk to L3 alone (binary, skills, cache no longer fight git because they're not in git).
- **`01-git-native-planning.md`**: confirmed for L3. The L1/L5 reshuffling does not affect the L3 conclusion.
- **`02-do-we-need-this.md`**: the brew-install model makes "do we need this?" a more tractable question — the framework becomes a *tool you install*, not a *system you adopt*. Lower commitment threshold; easier to try; easier to abandon.
- **`03-discipline-where-the-llm-cant-skip-it.md`**: the chokepoint argument is unchanged. CI still runs the binary on the PR. With the binary on PATH instead of via submodule, CI setup is simpler.
- **`04-governance-provenance-and-the-pre-pr-tier.md`**: the modular opt-in cross-cutting property is reinforced. Modules are now clearly L1+L2 things (binary capability + project policy). Pre-PR tier benefits from L1's simpler distribution.
- **`KERNEL.md`**: should pick up an additional cross-cutting property about layer separation. Proposed wording in §11 below.

---

## 11. Proposed update to KERNEL.md

Adding one cross-cutting property:

> **Layered location-of-truth.** The framework distinguishes engine binary (machine-installed), per-project policy and lockfile (in-repo, small YAML), per-project planning state (in-repo, markdown + metadata layer), per-project skills (in-repo, markdown), framework and third-party skills (binary-bundled or registry-cached), per-developer ergonomic state and skill local config (machine-local), and materialized skill adapters plus runtime cache (machine-local, composed from the skill sources, stable across branch switches by design — only regenerated on explicit `aiwf init` / `aiwf update`). Each layer lives where its constraints are best served. (See `05-where-state-lives.md` §3.)

Not edited from this document — flagged for a deliberate edit.

---

## 12. Open questions this document does not close

1. **What's the exact format of `aiwf.yaml` and `aiwf.lock`?** Schema work for L2 plus lockfile semantics (resolution algorithm, content hashing, integrity).
2. **How does `aiwf init` interact with existing repos that have hand-managed `.claude/skills/`?** Migration path needs design.
3. **Binary distribution — brew tap from where?** Anthropic-hosted, user-hosted, GitHub Releases? Affects update cadence and trust model.
4. **What is the project identity for L6 cache?** Repo absolute path is fragile; `aiwf.yaml` declared id is rigid; some hash of `git config remote.origin.url` plus repo root is fragile in different ways.
5. **Multi-binary scenarios.** Two devs with different `aiwf` versions on the same project. Does CI gate this? Does `aiwf doctor` warn? How loud?
6. **What about Windows?** `brew` is macOS/Linux. Windows needs scoop / winget / chocolatey support. Distribution multiplies.
7. **Is L6 adapter materialization compatible with skill-host evolution?** If `.claude/skills/` format changes in a future Claude Code release, the binary needs to know the new shape. Coupling concern.
8. **Skill registry for L5c — where does it live?** Anthropic-hosted, community-run, or no central registry (just URL-based installs)? Affects discoverability, trust, and the social shape of the ecosystem.
9. **Project-skill (L5b) dependencies on third-party skills (L5c).** If a project skill calls into a third-party skill, how is that declared? Does the resolver follow transitive dependencies? Open.
10. **Cross-host fidelity.** A skill written once should work in Claude Code, Cursor, Copilot, Continue, Aider. How much cross-host fidelity is the framework's job, and how much is the host adapter's job? Likely the materializer per host owns the translation; the skill author writes in a host-agnostic format.
11. **Skill conflict at composition.** If a project skill (L5b) and a third-party skill (L5c) both want to provide the same verb name, what wins? Manifest-declared precedence; clear conflict reporting from `aiwf update`.
12. **`aiwf update` semantics.** Strict (fail on any drift)? Permissive (warn but proceed)? Per-environment? Open.

These belong in subsequent research docs or in implementation proposals.

---

## 13. The honest answer to the user's headline question

> *Is there any information out there regarding which model is more successful or more promising?*

No rigorous empirical comparison exists for "all in-repo" vs. "all out-of-repo" planning state in AI-assisted teams. What does exist is **convergent practice across successful tools**, which is itself a kind of evidence: the answer is **layered, not binary**. Successful patterns put the binary outside, the policy and content (including project skills) inside, third-party skills cached locally and lockfile-pinned, the personal and materialized adapters outside. The user's proposal is right where it matches that pattern (L1, L5a, L6, and the user's revision adding L5b/c/d) and wrong where it deviates (L3).

> *What's so wrong about having the framework installed as a tool via brew?*

Nothing. That part is right and should be adopted. The error is conflating "framework binary external" (correct) with "framework state external" (wrong for almost any team).

> *Isn't that how LLMs work?*

The model is external. The per-project configuration is in-repo. Every successful AI coding tool in 2026 confirms this split. The framework should match.

> *I'd `brew install aiwf` and get agents and skills (versioned, gitignored) in my repo folder.*

Yes — with the user's revision sharpening this. Skills come from three sources (binary-bundled framework skills, in-repo project skills, registry-installed third-party skills) plus per-machine local config; the materialized adapters that the AI host actually reads are composed from those sources, gitignored, and **regenerated only on explicit `aiwf init` / `aiwf update`** — so the AI's behavior is stable across `git checkout`. This cleanly fixes the AI-rule-divergence problem identified in `04` while honoring the reality that skills must evolve independently of the framework binary, that projects have their own skills, and that local environments differ.

The framework that emerges from this layering is *smaller* than the current architecture, *more deployable* than the current architecture (brew install vs. submodule wrangling), and *more honest* about which problems live where.

---

## Sources

- [Denicek: Computational Substrate for Document-Oriented End-User Programming](https://dl.acm.org/doi/full/10.1145/3746059.3747646?download=true) — the paper the user cited; substrate for edits-over-document programming, relevant to the representation question but not directly to the location question.
- [Proceedings of the 38th Annual ACM Symposium on User Interface Software and Technology (UIST '25)](https://dl.acm.org/doi/proceedings/10.1145/3746059) — venue context.
