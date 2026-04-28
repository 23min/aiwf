# PoC design decisions

This is a self-contained summary of what the PoC commits to and why. It is the design context that motivates the build plan in [`poc-plan.md`](poc-plan.md). The full research arc that produced these decisions lives on `main`; this branch deliberately does not include those documents to keep the working context focused.

If a proposed change conflicts with anything below, treat it as a kernel-level decision and surface it explicitly.

---

## What the framework needs to do

Eight things, and only these:

1. **Record planning state** — what epics, milestones, decisions, gaps, and contracts exist. Persistent, accessible from inside the repo.
2. **Express relationships** — milestones belong to epics, milestones depend on milestones, decisions ratify scope, gaps motivate new work, ADRs and contracts constrain code.
3. **Support evolution** — insert a milestone between two others, rewrite scope when dependencies change, spawn an epic from a discovered gap, supersede a decision. Plans are clay until they're committed to main.
4. **Keep history honest** — when a thing changed, who changed it, why. Provenance is queryable.
5. **Validate consistency** — references resolve, status transitions are legal, terminal states stay terminal, cycles don't form, ids don't collide. Mechanical, fast, deterministic. No AI judgment required.
6. **Generate human-readable views** — ROADMAP, history reports, status summaries. Derived on demand from the canonical state.
7. **Coordinate AI behavior** — skills for the AI host, kept versioned with the framework binary so behavior is reproducible.
8. **Survive parallel work** — multiple branches, multiple writers (within the small-team target). Merging is well-defined for structural state; semantic conflicts surface as findings.

Notice what is *not* on this list: an event log, a graph projection, a hash chain, a closed entity vocabulary fixed before use, trace-first writes as a permanent ledger, a globally totally-ordered sequence of mutations. Those are implementation choices. The PoC does not adopt them.

---

## Cross-cutting properties

These are quality bars every part of the framework must respect:

- **Enforcement does not depend on the LLM choosing to enforce.** Skills are advisory; the pre-push git hook and `aiwf check` are authoritative. If a guarantee depends on the LLM remembering, it is not a guarantee.
- **Referential stability is real.** An id like `E-19`, once allocated, always means the same entity, even after rename, move, or status-change to terminal. The id is the primary key; the slug is just display.
- **Honest about meaning.** The framework guarantees referential and structural stability. It does not pretend to guarantee semantic stability of prose; that is a property of human and AI understanding.
- **Engine is invocable without an AI.** Every verb takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes. Humans, CI scripts, and other tools drive it directly.
- **Soft in raw studio; AI-assisted strictness pre-push; mechanical strictness at the chokepoint.** Iteration on a personal branch is unconstrained. As work approaches readiness, the validators tighten the loop pre-push, while the AI is still in conversation. The pre-push git hook is the mechanical gate that doesn't depend on the LLM.
- **Layered location-of-truth.** Engine binary is machine-installed (external). Per-project policy and planning state are in-repo. Materialized skill adapters are in-repo but gitignored. Each layer lives where its constraints are best served.

---

## What the PoC commits to

### Six entity kinds

| Kind | Statuses | ID format |
|---|---|---|
| Epic | `proposed`, `active`, `done`, `cancelled` | `E-NN` |
| Milestone | `draft`, `in_progress`, `done`, `cancelled` | `M-NNN` |
| ADR | `proposed`, `accepted`, `superseded`, `rejected` | `ADR-NNNN` |
| Gap | `open`, `addressed`, `wontfix` | `G-NNN` |
| Decision | `proposed`, `accepted`, `superseded`, `rejected` | `D-NNN` |
| Contract | `draft`, `published`, `deprecated`, `retired` | `C-NNN` |

Hardcoded in Go for the PoC. Extensible to YAML-driven kinds later if real consumers need to customize the vocabulary.

### Frontmatter schema and body templates

Every entity is a markdown file (or, for epics and contracts, a directory containing `epic.md` / `contract.md`) with YAML frontmatter and a prose body. Frontmatter is canonical structured state; the body is human prose, not parsed.

**Common to every kind:**

| Field | Type | Required | Notes |
|---|---|---|---|
| `id` | string | yes | Matches the kind's id format. Primary key. |
| `title` | string | yes | Display name; freely renamable. |
| `status` | string | yes | Must be in the kind's status set. |

**Per-kind reference fields** (checked by `refs-resolve`; `parent` and `depends_on` also feed `no-cycles`):

| Kind | Field | Type | Required | Target |
|---|---|---|---|---|
| Milestone | `parent` | id | yes | epic |
| Milestone | `depends_on` | []id | no | other milestones |
| ADR | `supersedes` | []id | no | other ADRs |
| ADR | `superseded_by` | id | no | another ADR |
| Gap | `discovered_in` | id | no | milestone or epic |
| Gap | `addressed_by` | []id | no | any kind |
| Decision | `relates_to` | []id | no | any kind |
| Contract | `format` | string | yes | e.g., `openapi`, `json-schema`, `proto` |
| Contract | `artifact` | path | yes | relative to the contract directory; existence checked by `contract-artifact-exists` |

Timestamps (`created`, `updated`) are deliberately absent from frontmatter; `git log` carries them. Putting them in YAML would be redundant state and a future drift target.

**Body templates** are short section stubs written by `aiwf add`. They are starting points, not enforced structure:

| Kind | Body sections |
|---|---|
| Epic | `## Goal` / `## Scope` / `## Out of scope` |
| Milestone | `## Goal` / `## Acceptance criteria` |
| ADR | `## Context` / `## Decision` / `## Consequences` |
| Gap | `## What's missing` / `## Why it matters` |
| Decision | `## Question` / `## Decision` / `## Reasoning` |
| Contract | `## Purpose` / `## Stability` |

Bodies are not validated. The framework guarantees structural and referential stability of frontmatter; prose is the human's responsibility.

### Stable ids and rename ergonomics

- IDs are sequential within a kind, allocated by scanning the tree at allocation time and picking `max + 1`. There is no cross-branch coordination; the allocator only sees the current branch's tree.
- The id is encoded in the file or directory path (`E-19-<slug>/`, `M-001-<slug>.md`). Because the slug is part of the path, two parallel branches that allocate the same id for different titles produce different paths — git merges both files in cleanly. The collision is *semantic* (two files share an id in their frontmatter), not textual, and surfaces only when `aiwf check` runs. This slug-as-collision-buffer property is what makes the simple allocator viable without coordination.
- Renames preserve the id: `aiwf rename <id> <new-slug>` does `git mv` plus a title update.
- Removals are not deletions. `aiwf cancel <id>` flips status to the kind's terminal value (`cancelled`/`wontfix`/`rejected`/`retired`). The file stays. References stay valid.
- Collisions are detected by `aiwf check`'s `ids-unique` finding. The pre-push hook makes this fatal before push. Resolution is `aiwf reallocate`, which accepts either an id (when unambiguous) or a path (required when two entities collide on the same id). It picks the next free id (`max + 1` at call time), `git mv`s, walks every entity's frontmatter to rewrite reference fields, and surfaces body-prose references as findings for human review. The id format is never extended with suffixes (no `M-007a`/`M-007b`); collision recovery always renumbers.

### Markdown is the source of truth; git is the time machine

There is no separate event log file. There is no separate graph projection file. The markdown frontmatter is canonical state; `git log` is history; structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable. `aiwf history <id>` reads `git log` filtered by trailer.

The `reallocate` verb additionally writes an `aiwf-prior-entity: <old-id>` trailer alongside the new id's `aiwf-entity:`. This is the bridge that keeps both ids' histories complete: `aiwf history M-008` matches `aiwf-entity: M-008` and shows the reallocate as an event in M-008's life; `aiwf history M-007` matches `aiwf-prior-entity: M-007` and shows the reallocate as the terminal event in M-007's life ("renumbered to M-008"). Without this, querying the old id after a reallocation would silently come up empty — a footgun, since the user would think the entity was deleted.

This is a deliberate departure from designs that maintain a parallel transaction log. The research that motivates this departure is on `main`; the short version is that an append-only event log file fights git's branching model in ways that are expensive to fix and unnecessary at this scale.

### Validation is the chokepoint

`aiwf check` is a pure function from the working tree to a list of findings. It runs as a `pre-push` git hook installed by `aiwf init`. The hook is what turns the framework's guarantees from suggestions into mechanical enforcement.

`--no-verify` bypasses the hook (standard git behavior). The framework does not try to prevent that — bypassing is sometimes the right call. But the *default* behavior is that broken state cannot be pushed silently.

### Layered location-of-truth

| Layer | Location | Why |
|---|---|---|
| Engine binary (`aiwf`) | Machine-installed via `go install` | Standard tool distribution; same as `git`, `ripgrep`, `jq` |
| Per-project policy (`aiwf.yaml`) | In the consumer repo, git-tracked | Team-shared, CI-readable, travels with clone |
| Per-project planning state (`work/`, `docs/adr/`) | In the consumer repo, git-tracked | Co-evolves with code, bisectable, no API friction |
| Per-developer config | `~/.config/aiwf/` | Personal preferences and tool-path overrides |
| Materialized skill adapters (`.claude/skills/aiwf-*`) | In the consumer repo, gitignored | Composed from the binary on `aiwf init`/`update`; stable across `git checkout`. The `aiwf-` prefix is the namespace boundary; non-`aiwf-*` skill directories are untouched. |

The materialization invariant is load-bearing: skills are regenerated only on explicit `aiwf init` / `aiwf update`, never implicitly on `git checkout` or every verb invocation. This is what keeps the AI's behavior stable when switching branches. The on-disk files are a cache, not state: `aiwf update` wipes every `.claude/skills/aiwf-*/` directory and rewrites them from the binary's embedded skills; `aiwf doctor` reports drift via byte-compare against the embedded version. No state file, no manifest, no version stamp.

Skills are embedded in the `aiwf` binary via Go's `embed.FS` and copied out on `init` / `update`. This deliberately couples skill content to the binary version: skills are adapters that call binary-provided commands, so version-skew between them would silently break things. Distributing skills via a separate channel — e.g., as a Claude Code plugin — is a viable future *packaging* path for easier installation, but as an architecture choice it would re-introduce the version-skew problem that embedding avoids.

### `aiwf.yaml` config

A short YAML file at the consumer repo root. Read by `aiwf` on every invocation; written by `aiwf init`. The file's presence is also how `aiwf` discovers the repo root: it walks up from the current working directory until it finds one.

| Field | Type | Required | Notes |
|---|---|---|---|
| `aiwf_version` | string | yes | Engine version the repo expects (e.g., `0.1.0`). `aiwf doctor` warns on mismatch. |
| `actor` | string | yes | Default value of the `aiwf-actor:` commit trailer (e.g., `human/peter`). Format: `<role>/<identifier>` — must match `^[^\s/]+/[^\s/]+$` (exactly one `/`, no whitespace, neither side empty; otherwise freeform). Override on a single invocation via `--actor`. `aiwf init` derives a default of `human/<local-part-of-git-config-user.email>` when not explicitly provided. |
| `hosts` | []string | no | Hosts to materialize skills for. PoC default and only supported value: `[claude-code]`. |

Example (the typical file):

```yaml
aiwf_version: 0.1.0
actor: human/peter
```

That's the entire file in normal use. `hosts` is omitted to take the default. No project-name field, no per-project skill paths, no policy knobs in the PoC; the kind FSM, id formats, and status sets are hardcoded in the engine — see *Six entity kinds*.

### One git commit per mutating verb

Every mutating verb (`add`, `promote`, `cancel`, `rename`, `reallocate`) produces exactly one git commit, or no change at all. Verbs are *validate-then-write*: the verb computes the projected new tree in memory (an overlay on top of the loaded tree), runs `aiwf check` against the projection, and only when the projection is clean writes files (and `git mv`s) and creates the commit. On findings the working tree is never touched. This gives per-mutation atomicity for free without a rollback path, and lets verbs run safely while the user has unstaged edits in flight. There is no separate journal file, no two-phase commit ceremony, no event-log-then-confirm protocol. The git commit *is* the atomic boundary.

Verbs only block on findings *introduced* by the projection — pre-existing tree errors (e.g., a broken reference left over from a prior hand-edit) do not refuse an unrelated `aiwf add`. The diff is by `code + subcode + path + entity + message`. This lets users incrementally fix a partially broken tree with `aiwf` itself rather than first having to clean up by hand. To see the full set of current problems regardless of any verb, run `aiwf check` directly.

---

## What is deliberately not in the PoC

In rough order of "if needed, here's how to add it":

| Feature | When to add | What's required |
|---|---|---|
| Project-specific skills | When a project starts wanting its own domain skill | A `.ai-repo/skills/` directory; `aiwf update` includes it in materialization |
| Multi-host adapters (Cursor, Copilot) | When a second host is in use | Per-host materializer functions; `hosts:` already accommodated in `aiwf.yaml` |
| Third-party skill registry | When a useful community skill emerges | `aiwf install`, lockfile, registry resolution |
| FSM-as-YAML | When kinds need per-project customization | Move the hardcoded transition functions to YAML |
| Module system | When the framework grows past ~20 verbs | Module loader controlled by `aiwf.yaml` |
| Tombstones beyond status-cancel | When entities need to be hidden from view, not just terminal | A `removed: true` flag plus render filter |
| Automatic migration from prior frameworks | When a second concrete migration source emerges | A bespoke transform script (likely external to `aiwf`); `aiwf check`'s findings are the diagnostic |
| GitHub Issues / Linear sync | When a project wants tickets mirrored | Opt-in sync module |
| CRDT registry, custom merge driver | When concurrent branches produce real merge conflicts on planning state | Modeled in research; not built in PoC |
| Pre-PR Workshop tooling | When PR review becomes a regular workflow | `aiwf prepush`, `aiwf preview-merge`, etc. |
| Hash-verified projection | Probably never, given the target scale | Modeled in research as expensive |
| Schema-aware contract validation | When a contract's format actually needs parsing | Per-format validators; PoC just checks file existence |

Each is a clean future addition. None blocks PoC value.

---

## On future versions

The PoC is deliberately discardable. The branch is not planned to merge back to `main`. A future version is free to take a different shape — possibly closer to the earlier event-sourced design, possibly something different — and the on-disk format is simple enough (markdown files with frontmatter, conventional directory layout, structured commit trailers) that a v2 reader could import a v1 tree mechanically.

The door to a backwards-compatible successor is left open. The door to a *different* successor is also left open. The PoC commits to the smallest set of mechanical guarantees that delivers value now; it does not commit to being the framework's permanent shape.

---

## The principle behind the plan

The framework's value is the smallest set of mechanical guarantees that lets a forgetful AI and a busy human not lose track of what's planned, decided, and done. At the target scale, that smallest set is:

- Six kinds with closed status sets.
- Stable ids that survive rename, cancel, and collision.
- An `aiwf check` that runs as a pre-push hook.
- An `aiwf history` that answers "what happened here?".
- Skills the AI can find that don't drift across branches.
- An on-disk format simple enough to grow into something larger or migrate away from.

Build that. Use it. Let real friction tell you what to add.
