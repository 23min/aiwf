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

### Stable ids and rename ergonomics

- IDs are sequential within a kind, allocated by scanning the tree at allocation time and picking `max + 1`.
- The id is encoded in the file or directory path (`E-19-<slug>/`, `M-001-<slug>.md`).
- Renames preserve the id: `aiwf rename <id> <new-slug>` does `git mv` plus a title update.
- Removals are not deletions. `aiwf cancel <id>` flips status to the kind's terminal value (`cancelled`/`wontfix`/`rejected`/`retired`). The file stays. References stay valid.
- Collisions are detected by `aiwf check` (two paths starting with the same `E-NN-` prefix). Resolution is `aiwf reallocate <id>`, which picks the next free id, `git mv`s, walks every entity's frontmatter to update reference fields, and surfaces body-prose references as findings for human review.

### Markdown is the source of truth; git is the time machine

There is no separate event log file. There is no separate graph projection file. The markdown frontmatter is canonical state; `git log` is history; structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable. `aiwf history <id>` reads `git log` filtered by trailer.

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
| Materialized skill adapters (`.claude/skills/wf-*`) | In the consumer repo, gitignored | Composed from the binary on `aiwf init`/`update`; stable across `git checkout` |

The materialization invariant is load-bearing: skills are regenerated only on explicit `aiwf init` / `aiwf update`, never implicitly on `git checkout` or every verb invocation. This is what keeps the AI's behavior stable when switching branches.

### One git commit per mutating verb

Every mutating verb (`add`, `promote`, `cancel`, `rename`, `reallocate`) produces exactly one git commit. This gives per-mutation atomicity for free: if the working tree changes don't pass `aiwf check`, the verb aborts before the commit lands. There is no separate journal file, no two-phase commit ceremony, no event-log-then-confirm protocol. The git commit *is* the atomic boundary.

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
