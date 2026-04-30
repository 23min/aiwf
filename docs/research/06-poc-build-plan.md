# A KISS PoC build plan

> **Status:** actionable-plan
> **Hypothesis:** At single-developer or small-team scale on a few-months horizon, the framework's value collapses to six entity kinds, stable IDs with collision resolution, a pre-push validator, a structured-commit history reader, and stable skills — implementable in a focused week or two, broken into a small number of sessions.
> **Audience:** anyone executing the PoC, or deciding whether the prior research is enough to start building.
> **Premise:** most of the framework's apparent complexity is paying for problems that don't fire at this scale; strip those costs, ship a tiny thing in a few focused sessions, try it on real projects, let real friction surface what — if anything — needs to grow.
> **Discipline:** any feature not on this list is out of scope until real use exposes a need for it.
> **Tags:** #aiwf #plan

---

## Abstract

This document concludes the `00`–`05` research arc by collapsing the synthesis into a buildable plan. A single Go binary `aiwf`, installed via `go install`. The repo gets one config file (`aiwf.yaml`), one planning directory (`work/`), one decisions directory (`docs/adr/`), and one gitignored materialized-skills directory (`.claude/skills/wf-*`). Six entity kinds (epic, milestone, ADR, gap, decision, contract), each with a closed status set and one Go function for legal transitions. A handful of verbs, each producing one git commit with a structured trailer. An `aiwf check` command that validates the tree and runs as a pre-push hook. An `aiwf history <id>` that renders `git log` for an entity. **No event log, no graph projection, no CRDTs, no FSM-as-data, no module system, no registry, no multi-host adapters, no tombstones beyond `cancelled`, no cross-branch merge handling.** Small enough to throw away — a focused week or two of work, broken into a small number of sessions, ready to use on a real project. Built on a branch (`poc/aiwf-v3`) so `main` stays open for other implementations. Future additions are deferred until real friction demonstrates need.

---

## 1. What is and isn't a problem at this scale

What the research worried about that **does not fire** at single-developer or small-team scale on a few-months horizon:

- Heavy concurrent-writer conflicts on planning state — at most a few writers, often one at a time.
- ID collisions across many branches — usually one or two branches in flight, rarely more.
- Cross-team governance and approval flows — no large review chain.
- "AI rules diverge across many long-lived branches" — branches are short-lived; rule changes are deliberate.
- Multi-many-machine sync — `git push`/`pull` handles whatever sync is needed for in-repo state.
- Server-side merge drivers, full CRDT registries, elaborate tombstone bookkeeping — unnecessary at this scale.
- Compliance / audit-grade provenance — not in scope for the PoC.

What **does** still fire and so must be addressed:

- **Referential stability** (`03`, `04`): the AI gets confused when `E-19` shifts meaning or disappears. At small scale this matters more, not less, because the AI is doing more of the planning load.
- **In-repo planning beats external** (`05` §3.3): bisectable, one working set for the AI, no API friction.
- **The forgetful-assistant problem** (`02`): the AI needs reliable places to look (decisions, current goal, recent changes) and a habit of writing back.
- **Validation as the chokepoint** (`03`): even with one or two people, things get missed. A pre-push hook catches what skills miss.
- **Plans are clay** (`02`, `04`): rename, insert, cancel must be cheap and not break references.
- **Skills must be stable across `git checkout`** (`05` §3.6): adapters materialized once, regenerated on explicit update only.
- **L1 binary external, L2/L3 in-repo, L6 gitignored** (`05` §3): the layer split applies at every scale.

That trims the design surface dramatically.

---

## 2. The PoC, in one paragraph

A single Go binary `aiwf`, installed via `go install`. The repo gets one config file (`aiwf.yaml`), one planning directory (`work/`), one decisions directory (`docs/adr/`), and one gitignored materialized-skills directory (`.claude/skills/wf-*`). Six entity kinds: `epic`, `milestone`, `adr`, `gap`, `decision`, `contract`. A handful of verbs that each produce one git commit with a structured trailer. A `aiwf check` command that validates the tree and is run by a `pre-push` git hook. A `aiwf history <id>` that renders `git log` for an entity. That's it. No event log, no graph projection, no CRDTs, no FSM-as-data, no module system, no registry, no multi-host adapters, no tombstones, no cross-branch merge handling.

---

## 3. Hard defaults — no configuration decisions

The PoC ships with these defaults baked in. Configuration to change any of them is **deferred**: when something hurts, then make it configurable.

### Directory layout

```
<repo-root>/
├── aiwf.yaml                          # the only config file (~10 lines)
├── work/
│   └── epics/
│       └── E-NN-<slug>/
│           ├── epic.md
│           └── M-NNN-<slug>.md       # milestones live inside their epic
├── work/
│   ├── gaps/
│   │   └── G-NNN-<slug>.md
│   ├── decisions/
│   │   └── D-NNN-<slug>.md
│   └── contracts/
│       └── C-NNN-<slug>/
│           ├── contract.md            # description, scope, status, links
│           └── schema/                # the actual artifact(s): OpenAPI, JSON Schema, .proto, etc.
├── docs/
│   └── adr/
│       └── ADR-NNNN-<slug>.md
├── .claude/skills/wf-*/              # gitignored; materialized by `aiwf init`
└── ROADMAP.md                        # rendered on demand by `aiwf render roadmap`
```

### Entity kinds and statuses (closed sets, hardcoded)

| Kind | Statuses | ID format |
|---|---|---|
| Epic | `proposed`, `active`, `done`, `cancelled` | `E-NN` (zero-padded width 2) |
| Milestone | `draft`, `in_progress`, `done`, `cancelled` | `M-NNN` (zero-padded width 3) |
| ADR | `proposed`, `accepted`, `superseded`, `rejected` | `ADR-NNNN` (zero-padded width 4) |
| Gap | `open`, `addressed`, `wontfix` | `G-NNN` (zero-padded width 3) |
| Decision | `proposed`, `accepted`, `superseded`, `rejected` | `D-NNN` (zero-padded width 3) |
| Contract | `draft`, `published`, `deprecated`, `retired` | `C-NNN` (zero-padded width 3) |

Gap and decision are intentionally lighter than ADR. **Gap** = "we discovered this was missing or wrong while doing the work" — created mid-flight, often tied to a specific milestone, resolves quickly. **Decision** = a small ratification that doesn't deserve a full ADR ("we decided to defer EU VAT for now"). ADR is for architecture-level decisions with reasoning, alternatives, and consequences.

**Contract** is different in shape from the others: it points at a concrete machine-readable artifact (an OpenAPI spec, a JSON Schema, a `.proto` file, a SQL migration, a TypeScript type definition — whatever the boundary's natural format is). The `contract.md` is the description and metadata; the actual schema lives next to it under `schema/` (path is open-ended on purpose — different boundaries have different formats). Contracts get their own kind because (a) brownfield projects in particular involve reading and writing API/data contracts constantly, and even greenfield projects benefit from giving such artifacts a stable identity, and (b) other entities cite them: a milestone can be "implement endpoints described in C-003"; an ADR can say "this is enforced by C-007"; a gap can flag "C-005 is missing examples for the error response."

The kinds compose: a gap can spawn an ADR or a contract; an ADR can ratify a decision and reference a contract; a decision can record a deferred gap; a contract can be the artifact a milestone produces or constrains.

### Frontmatter (minimal, per kind)

**Epic:**
```yaml
---
id: E-01
kind: epic
title: Payments rewrite
status: active
created: 2026-04-27
---
```

**Milestone:**
```yaml
---
id: M-001
kind: milestone
title: Extract pricing service
status: in_progress
parent: E-01
depends_on: []
created: 2026-04-27
---
```

**ADR:** standard MADR-style. `id`, `kind: adr`, `title`, `status`, `created`, optional `supersedes`, `superseded_by`.

**Gap:**
```yaml
---
id: G-001
kind: gap
title: Existing pricing service has no audit log
status: open
discovered_in: M-002        # optional; the milestone where this surfaced
addressed_by:               # optional; ADR / milestone / decision id once resolved
created: 2026-04-27
---
```

**Decision:**
```yaml
---
id: D-001
kind: decision
title: Defer EU VAT integration to a later milestone
status: accepted
relates_to: [E-01, M-002]   # optional; the entities this decision affects
created: 2026-04-27
---
```

**Contract:**
```yaml
---
id: C-001
kind: contract
title: Pricing service public API
status: draft
artifact: schema/openapi.yaml   # path relative to the contract's directory
format: openapi-3.1             # one of: openapi, json-schema, proto, sql, typescript, other
relates_to: [E-01, M-002]       # optional; entities that produce or constrain this contract
created: 2026-04-27
---
```

The `format` field is a hint, not enforced — the engine treats the artifact as opaque (it does not parse OpenAPI or validate `.proto`). What it can check is: the `artifact:` path resolves to a file that exists in the contract's directory. That alone is worth having and costs ten lines of code.

### ID allocation

Scan the tree at allocation time, pick `max(seen ids of this kind) + 1`. No counter file, no allocator state. At single-developer or small-team scale on a primary branch, contention is rare; the simplest implementation is the right one. The collision-handling section below covers the cases where it does happen.

### ID collision handling (when contention does happen)

The above is true on the linear path. But the user's real workflow includes splitting epics, inserting milestones between existing ones, and occasionally experimenting on side branches. So collisions can arise:

- **Side-branch experiments** — branch A and branch B both allocate `E-19` from the same merge base.
- **Manual creation followed by automated allocation** — you create `E-19` by hand, then `aiwf add` doesn't see it yet because of timing or scope and allocates `E-19` again.
- **Cherry-pick / rebase scenarios** — replaying a commit that allocated `E-19` onto a tree where `E-19` already means something else.

Strategy: **detect collisions on the id, not on the slug.** The slug-in-path encoding (`E-19-foo/`, `E-19-bar/`) makes detection trivial — two paths starting with `E-19-` is the signal. `aiwf check` reports an `ids-collide` finding listing both paths. Resolution is a verb, not automatic on first sight (auto-action without consent is a foot-gun):

`aiwf reallocate <id>` picks the next free id for the kind, performs:
1. `git mv` the directory or file to the new id.
2. Update the entity's own `id` field in frontmatter.
3. Walk every other entity's frontmatter and rewrite reference fields (`parent`, `depends_on`, `supersedes`, `superseded_by`, `discovered_in`, `addressed_by`, `relates_to`) that pointed at the old id.
4. **Do not** rewrite prose body references. Surface them as a finding ("references in body text not updated; review and fix manually") so the human (or AI) can review.
5. Commit with a structured trailer recording old-id → new-id.

This keeps the rule simple: **the id is the primary key; collisions are a finding; resolution is one verb that updates references atomically.** It costs about half a session of additional work in session 2 and removes a real foot-gun.

### Status transitions (hardcoded; one Go function per kind)

- Epic: `proposed → active → done`; any non-terminal → `cancelled`.
- Milestone: `draft → in_progress → done`; any non-terminal → `cancelled`.
- ADR: `proposed → accepted`; `accepted → superseded`; `proposed → rejected`.
- Gap: `open → addressed`; `open → wontfix`.
- Decision: `proposed → accepted`; `accepted → superseded`; `proposed → rejected`.
- Contract: `draft → published → deprecated → retired`; `draft` can also go straight to `retired` if the contract was never published.

Anything else is rejected by `aiwf promote` and reported by `aiwf check`.

### Removals

There is no delete verb. To remove a thing, `aiwf cancel <id>` flips status to `cancelled`. The file stays. References stay valid. This is the cheapest possible tombstone protocol — the entity is in the tree, just terminal.

### Renames

`aiwf rename <id> <new-slug>` does `git mv` (directory for epics and contracts, file for milestones/ADRs/gaps/decisions) and updates the title in frontmatter. The id never changes. References elsewhere never break because they reference the id, not the slug.

### Skills

The binary embeds skill files for Claude Code. `aiwf init` materializes them to `.claude/skills/wf-*/SKILL.md` and adds those paths to `.gitignore`. `aiwf update` regenerates. No regeneration on `git checkout`; that's by design.

Skills shipped:
- `wf-add` — how to create epics, milestones, ADRs, gaps, decisions, contracts (with prompts to fill out frontmatter properly per kind).
- `wf-promote` — how to advance status legally.
- `wf-rename` — how to rename without breaking references.
- `wf-reallocate` — how to resolve id collisions when they're detected.
- `wf-history` — how to ask "what happened here?".
- `wf-check` — what `aiwf check` reports and how to fix common findings.

No project-skills directory for the PoC. If real use surfaces a need for one, add it then.

### Pre-push hook

`aiwf init` writes `.git/hooks/pre-push` that runs `aiwf check` and blocks on findings of severity `error`. `--no-verify` bypasses, like for any hook.

---

## 4. The verbs (complete list for the PoC)

| Verb | What it does | Commit message format |
|---|---|---|
| `aiwf init` | Create `aiwf.yaml`, scaffold directories, materialize skills, install pre-push hook | `chore(aiwf): init` (only if init produces changes) |
| `aiwf update` | Re-materialize skills (after `go install` upgrade) | (no commit; updates gitignored files) |
| `aiwf add epic --title "..."` | Allocate `E-NN`, create directory + `epic.md` | `feat(plan): add E-NN <slug>` |
| `aiwf add milestone --epic E-NN --title "..."` | Allocate `M-NNN`, create file under epic | `feat(plan): add M-NNN <slug>` |
| `aiwf add adr --title "..."` | Allocate `ADR-NNNN`, create file | `docs(adr): add ADR-NNNN <slug>` |
| `aiwf add gap --title "..." [--discovered-in M-NNN]` | Allocate `G-NNN`, create file | `feat(plan): add G-NNN <slug>` |
| `aiwf add decision --title "..." [--relates-to E-NN,M-NNN]` | Allocate `D-NNN`, create file | `feat(plan): add D-NNN <slug>` |
| `aiwf add contract --title "..." --format <openapi\|json-schema\|proto\|...> --artifact <path>` | Allocate `C-NNN`, create directory + `contract.md`; `--artifact` may point at an existing file (move it into `schema/`) or imply a placeholder | `feat(plan): add C-NNN <slug>` |
| `aiwf promote <id> <status>` | Edit frontmatter, validate transition | `feat(plan): promote <id> to <status>` |
| `aiwf cancel <id>` | Set status to `cancelled` (or `wontfix` for gaps, `rejected` for decisions, `retired` for contracts) | `feat(plan): cancel <id>` |
| `aiwf rename <id> <new-slug>` | `git mv` + frontmatter title update | `chore(plan): rename <id> to <slug>` |
| `aiwf reallocate <id>` | Resolve an id collision: pick next free id, `git mv`, update references in frontmatter, surface body-prose references as a finding | `chore(plan): reallocate <old-id> to <new-id>` |
| `aiwf check` | Validate the tree, report findings, exit non-zero on errors | (no commit) |
| `aiwf history <id>` | Render `git log` for the entity, formatted | (no commit) |
| `aiwf render roadmap` | Print a markdown table of all epics + milestones (write to `ROADMAP.md` if `--write`) | `docs(roadmap): refresh` if `--write` |
| `aiwf doctor` | Check binary version against `aiwf.yaml`, check skill drift, check id-collision health | (no commit) |

Every commit-producing verb writes a structured trailer:

```
aiwf-verb: promote
aiwf-entity: M-001
aiwf-actor: human/peter
```

`aiwf history` reads this trailer to render structured timelines.

`aiwf cancel` is a thin shorthand over `aiwf promote` that picks the right terminal status for the kind: `cancelled` for epics/milestones, `wontfix` for gaps, `rejected` for decisions/ADRs (in the `proposed` state), `retired` for contracts.

That's a small set of verbs — roughly a dozen and a half. The binary should stay small enough to read end-to-end in an afternoon; chasing a precise line count is the wrong target.

---

## 5. The checks `aiwf check` runs

Pure functions of the working tree. Fast. Deterministic.

1. **`ids-unique`** — no two entities share an id. Severity: error. (Detected via path prefix collision; e.g. two directories `E-19-foo/` and `E-19-bar/`.)
2. **`refs-resolve`** — every `parent`, `depends_on`, `supersedes`, `superseded_by`, `discovered_in`, `addressed_by`, `relates_to` resolves to an existing id. Severity: error.
3. **`status-valid`** — every entity's status is in the allowed set for its kind. Severity: error.
4. **`frontmatter-shape`** — required fields present, types correct. Severity: error.
5. **`no-cycles`** — no cycle in `depends_on` or `parent`. Severity: error.
6. **`contract-artifact-exists`** — for every contract, the file at `artifact:` exists relative to the contract's directory. Severity: error.
7. **`titles-nonempty`** — title field is set and non-empty. Severity: warning.
8. **`adr-supersession-mutual`** — if `A.superseded_by = B`, then `B.supersedes ⊇ {A}`. Severity: warning.
9. **`gap-resolved-has-resolver`** — a gap with status `addressed` has a non-empty `addressed_by`. Severity: warning.

Nine checks. Each is a small function. Together they deliver the referential-stability property from `03` and `04`.

---

## 6. The `aiwf.yaml` (entire schema)

```yaml
aiwf_version: ">= 0.1.0"   # required minimum; updated by aiwf init
project_id: payments-platform  # arbitrary string, used for cache pathing
hosts:
  - claude-code              # which AI hosts to materialize skills for; PoC: just Claude Code
```

That's it. No module list, no convention overrides, no governance rules, no sync configuration. Add fields when something hurts.

No lockfile for the PoC — there are no third-party skills.

---

## 7. Build sequence — a handful of sessions

The work breaks naturally into four deliverable-shaped chunks. Each is a focused work block of a few hours; in practice a chunk may take more than one sitting, especially the first. Stop when the chunk's deliverable runs end-to-end with at least one happy-path test.

### Session 1 — Foundations and `aiwf check`

Goal: an executable that loads the tree, validates it, reports findings. No mutating verbs yet.

Tasks:
- Go module skeleton (`tools/cmd/aiwf/main.go`, `tools/internal/`).
- Frontmatter parser (use `gopkg.in/yaml.v3` + the existing parsing code if any is already in this repo's `tools/` that's safe to reuse; otherwise write 50 lines).
- Tree loader: walks `work/epics/**` and `docs/adr/**`, parses every entity, returns an in-memory model.
- `aiwf check` with the nine checks from §5. JSON output (`--format=json`) and human-readable text (default).
- Tests: a synthetic-tree fixture for each finding type.

Deliverable: `aiwf check` runs against a hand-crafted `work/` directory and reports findings. Exit code 0 for clean, 1 for errors.

### Session 2 — Mutating verbs and commit trailers

Goal: the verbs in §4 that produce commits.

Tasks:
- `aiwf add epic|milestone|adr` — allocate id, write file, `git add` + `git commit` with structured trailer.
- `aiwf promote <id> <status>` — read entity, validate transition (one Go function per kind), edit frontmatter, commit.
- `aiwf cancel <id>` — special case of promote.
- `aiwf rename <id> <new-slug>` — `git mv` + frontmatter title update + commit.
- `aiwf check` runs as a guard inside each mutating verb (post-mutation): if the check fails, abort and roll back the file changes (the un-committed working tree changes; we haven't committed yet).
- Tests: round-trip for each verb against a fresh repo.

Deliverable: end-to-end planning workflow works. `aiwf init && aiwf add epic ... && aiwf add milestone ... && aiwf promote ...` produces a sensible git history.

### Session 3 — Skills, history, hooks

Goal: the AI can actually use it; `git log` becomes queryable.

Tasks:
- Embed skill markdown files via `embed.FS`.
- `aiwf init` writes `aiwf.yaml`, scaffolds `work/`, `docs/adr/`, materializes skills to `.claude/skills/wf-*/SKILL.md`, adds those paths to `.gitignore`, installs the pre-push hook.
- `aiwf update` re-materializes skills.
- `aiwf history <id>` reads `git log` filtered for `aiwf-entity: <id>` trailers; pretty-prints.
- `aiwf doctor` checks binary version vs. `aiwf.yaml`'s `aiwf_version`, plus skill freshness.
- Pre-push hook script generated by `aiwf init`.
- Tests: `aiwf init` in a fresh repo produces the expected layout; `aiwf history` returns the expected events for a multi-step fixture.

Deliverable: in a fresh repo, `aiwf init` sets you up; the AI host (Claude Code) sees the skills; the pre-push hook catches errors before push.

### Session 4 — Polish for real use

Goal: ready for real use.

Tasks:
- `aiwf render roadmap` — print a markdown table; with `--write` updates `ROADMAP.md` and commits.
- A short `CLAUDE.md` template `aiwf init` writes (only if no `CLAUDE.md` exists), explaining the conventions to the AI.
- `README.md` for the framework itself: what it is, how to install, the verbs.
- A self-test: `aiwf doctor --self-check` runs all the verbs against a temp directory.
- Light error-message polish — every finding should be one line, name the file:line, and suggest a fix.

Deliverable: usable on a real project.

Total: a focused week or two of work, depending on how clean the chunks land on the first try. After the polish chunk, the framework is good enough to start using.

---

## 8. What's deliberately not in the PoC

In rough order of "if needed, here's how to add it":

| Feature | When to add | What's required |
|---|---|---|
| Project-specific skills | When a project starts wanting its own domain skill | A `.ai-repo/skills/` directory; `aiwf update` includes it in materialization |
| Multi-host adapters (Cursor, Copilot) | When you start using a second host | Per-host materializer functions; `hosts:` in `aiwf.yaml` already supports it |
| Third-party skill registry | When a useful community skill emerges | `aiwf install`, `aiwf.lock`, registry resolution |
| FSM-as-YAML | When you need a seventh kind, or want to customize transitions per project | Move the hardcoded transition functions to `framework/contracts/*.yaml` |
| Module system | When the framework has more than ~20 verbs | `modules:` in `aiwf.yaml` controlling which verb groups load |
| Schema-aware contract validation | When a contract's format actually needs validation (parse OpenAPI, run JSON-schema-meta-schema, etc.) | Per-format validators; PoC just checks the artifact file exists |
| Tombstones beyond `cancelled` | When you need to actually remove an entity from view | A `removed: true` flag + render filter |
| GitHub Issues sync | When a project wants tickets mirrored or linked | An opt-in `gh-sync` module |
| CRDT registry, custom merge driver | When team work produces real merge conflicts on planning state | `04` §5; `00` Tier 1 |
| Pre-PR Workshop tooling | When PR review becomes a regular workflow | `04` §6 (`aiwf prepush`, `aiwf preview-merge`) |
| Hash-verified projection | Probably never, given the target scale | `00` discusses why this is expensive |

Each of these is a clean future addition. None of them blocks PoC value.

---

## 9. The patch-vs-rebuild decision

Honest framing of the choice between continuing on top of the earlier framework versus starting fresh with this PoC:

**Patch the existing framework**: known shape, known friction. Patches likely accumulate over time. Risk: the friction surfaces at the worst time (mid-project, when attention should be on the actual work).

**Build this PoC fresh**: a focused week or two of work up front, before serious use. Risk: that effort lost if the framework turns out not to be needed at all.

The break-even point is roughly: if patching the existing framework will cost more than a similar block of attention over a real project, build fresh. If the existing framework's failure modes (those described in `00`–`05`) don't fire at the target scale, patching is fine.

A reasonable approach: **timebox the PoC at a focused week or two.** If the third chunk's deliverable (init + history + hooks working) is not reached within the timebox, abandon and patch the existing framework instead. If it is reached cleanly, the project is ahead and on solid ground.

The PoC is small enough that the risk is bounded. The existing framework's friction is unbounded if it bites at the wrong moment. On expected-value, the PoC wins for projects in the target range unless the existing framework is already known to be working well at this scale.

---

## 10. Build the PoC on a branch — yes, this is the right move

The user's instinct: develop the PoC on a branch in this repo (`ai-workflow-v2`), never merged to `main`, discardable. **This is straightforwardly correct and complicates nothing.** Concretely:

```
git checkout -b poc/aiwf-v3
# all PoC sessions land commits here
# main stays untouched
```

Why this is the right shape:

- **Main stays an aspirational reference.** The research arc (`00`–`05`, `KERNEL.md`) and the existing `architecture.md` / `build-plan.md` on `main` remain as the long-term north star. The PoC branch is the practical experiment that runs alongside.
- **Discardable.** If the PoC turns out wrong, `git branch -D poc/aiwf-v3` and there's no cleanup on main. If it turns out right, you can later cherry-pick selectively, rewrite into a clean PR, or even rename the branch to `main` and `main` to `archive/pre-poc`.
- **Other implementations stay possible.** Nothing on the PoC branch forecloses a future Tier-2 (Automerge-substrate) or Tier-3 (patch-theory) implementation. They could each get their own branch off of the same base.
- **`go install` works fine from a branch.** When the binary is needed for real use, `go install github.com/<user>/ai-workflow-v2/tools/cmd/aiwf@<branch-or-commit>` installs from the branch's tip. Or build locally with `go build -o ~/bin/aiwf ./tools/cmd/aiwf` and put `~/bin` on PATH. Either way, the installed binary is bit-for-bit the PoC code, no submodule, no surprise.

What does *not* work and would complicate things — and is a different idea you might be conflating with this one:

- **Running the PoC framework against a *consumer repo* on a branch that never merges to that repo's main.** Don't do that. A consumer repo's main is the consumer repo's main; the PoC framework's metadata (`work/`, `docs/adr/`, `aiwf.yaml`) lives there like any other tool's output. Splitting planning state and code across separate branches in the consumer repo destroys bisectability and forces constant branch-hopping.

So: PoC framework's *source code* lives on `poc/aiwf-v3` in this (`ai-workflow-v2`) repo. Consumer repos use the *built binary* from that branch and treat `aiwf` as just another installed CLI tool. The two concerns are clean.

One small concrete: when the binary needs to be available across multiple machines, tag commits on the PoC branch when they're stable (`git tag poc-v0.1`, `poc-v0.2`, …) and `go install ...@poc-v0.2` to pin. Branch-tip is fine for a single machine; tags are friendlier for multi-machine.

---

## 11. First-use checklist

When the PoC is built, a typical first session inside a consumer repo looks like:

1. `cd ~/Projects/<consumer-repo>`
2. `aiwf init`
3. `aiwf add epic --title "<the first epic appropriate to this project>"` and promote to `active`
4. `aiwf add milestone --epic E-01 --title "<the first milestone>"`
5. `aiwf add adr --title "<a decision worth recording up front>"` if relevant
6. Open Claude Code, ask it to read the skills, start working.

For brownfield projects, the first epic is often "Discovery and ramp-up" with milestones around mapping the existing system, identifying deployment paths, and surfacing gaps. For greenfield projects, the first epic is more often "Foundations" with milestones around scaffolding, baseline conventions, and the first real feature. Either way, the first use of the framework gives the first signal of whether it is enough. Friction surfaces as `aiwf add gap --discovered-in M-001` and similar one-liners; the decision to extend the framework or live with the friction can be made later.

---

## 12. The principle behind the plan

The research arc landed on this: **the framework's value is the smallest set of mechanical guarantees that lets a forgetful AI and a busy human not lose track of what's planned, decided, and done.** At single-developer or small-team scale, on weeks-to-months projects of either greenfield or brownfield shape, that smallest set is:

- Six kinds — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for legal transitions.
- Stable ids that don't break under rename, cancel, or collision (resolved by `aiwf reallocate`).
- A `check` that runs pre-push.
- A `history` that answers "what happened here?".
- Skills that the AI can find and that don't drift across branches.
- A PoC branch (`poc/aiwf-v3`) that keeps the experiment isolated from `main` so the road stays open for other implementations.

Nothing else is load-bearing. Build that. Ship in a week. Use it. Let real friction tell you what to add.

---

## In this series

- Previous: [05 — Where state lives](https://proliminal.net/theses/where-state-lives/)
- Next: [07 — State, not workflow](https://proliminal.net/theses/state-not-workflow/)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
