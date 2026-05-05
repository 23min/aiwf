# aiwf architecture (PoC)

This is the foundational reference for how aiwf is shaped. It is **synthesis only** — it does not introduce new design; it distills what is already true in the code so a new contributor can place a file, trace a verb, or reason about a boundary by reading one document.

For the *why* behind the choices, see [`design/design-decisions.md`](design/design-decisions.md) (the seven kernel commitments) and [`design/design-lessons.md`](design/design-lessons.md) (the three architectural principles distilled from them). This document is for the *what* and *how*.

---

## 1. System shape — the four layers

aiwf has four distinct layers, each living where its constraints are best served. Knowing which layer something belongs to is usually enough to know where to put it.

```
┌─────────────────────────────────────────────────────────────────────┐
│  Layer 1 — Engine binary (machine-installed, external to repo)      │
│  ─ go install github.com/23min/ai-workflow-v2/cmd/aiwf        │
│  ─ Single binary; no plugins; no per-project install                │
│  ─ Hardcodes the six entity kinds and their statuses                │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Layer 2 — Embedded skills (compiled into the binary)               │
│  ─ internal/skills/embedded/aiwf-{add,check,…}/SKILL.md       │
│  ─ The aiwf-* skills the binary materializes on demand              │
│  ─ Versioned with the binary; never edited in-repo by hand          │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  │  aiwf init / aiwf update
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Layer 3 — Materialized skill adapters (in-repo, gitignored)        │
│  ─ <consumer-repo>/.claude/skills/aiwf-*/                           │
│  ─ Regenerated only on explicit aiwf init / aiwf update             │
│  ─ Stable across git checkout by design                             │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Layer 4 — Consumer planning state (in-repo, committed)             │
│  ─ <consumer-repo>/work/{epics,gaps,decisions,contracts}/           │
│  ─ <consumer-repo>/docs/adr/                                        │
│  ─ <consumer-repo>/aiwf.yaml (per-project policy + contract bindings)│
│  ─ This is the source of truth; markdown files are canonical state  │
└─────────────────────────────────────────────────────────────────────┘
```

**Where to put a file:**

- New verb implementation → layer 1, under `internal/verb/` or `cmd/aiwf/`.
- New aiwf-* skill describing how an agent should use a verb → layer 2, under `internal/skills/embedded/<skill-name>/SKILL.md`.
- New entity kind, status, or schema field → layer 1, in `internal/entity/entity.go` (the canonical schema table).
- New per-project config knob → layer 4, in `aiwf.yaml` (and add a parser arm in `internal/aiwfyaml/`).
- A skill that wraps multiple aiwf verbs into a ritual (planning, wrap-epic, record-decision) → **not** part of aiwf core; lives in the rituals plugin (`ai-workflow-rituals` repo).

---

## 2. Data flow — load, check, project, apply, commit

Every aiwf invocation that touches state follows the same four-step pipeline. Read-only verbs (`check`, `status`, `history`, `schema`, `template`, `whoami`, `doctor`, `render` without `--write`) stop after step 1 or 2.

```
                     consumer repo on disk
                              │
                              │ ① tree.Load
                              ▼
            ┌──────────────────────────────────────┐
            │  tree.Tree {                          │
            │    Entities []*entity.Entity          │
            │    Stubs    []*entity.Entity   ← G14  │
            │    PlannedFiles map[string]struct{}   │
            │  }                                    │
            └──────────────────────────────────────┘
                              │
                              │ ② check.Run (read-only verbs end here)
                              ▼
                       []check.Finding
                              │
                              │ ③ verb projects mutation onto the tree
                              ▼
              projected *tree.Tree (in memory)
                              │
                              │ ④ check.Run on projection;
                              │    findings introduced by the verb block
                              ▼
                          *verb.Plan
                              │
                              │ ⑤ verb.Apply
                              │    — locks .git/aiwf.lock (POSIX flock)
                              │    — runs every OpMove via `git mv`
                              │    — runs every OpWrite directly to disk
                              │    — composes commit message + trailers
                              │    — git commit
                              ▼
                       single git commit with
                  aiwf-verb / aiwf-entity / aiwf-actor
                       trailers; fully atomic
```

**The contract this pipeline implements:**

- *Validate-then-write.* No verb mutates the working tree before its projection has passed `check.Run`. Failed validation aborts before any file is touched.
- *One commit per verb.* No verb composes multiple commits. Atomicity is bounded by the commit; rollback is "we did not commit," not "we are unwinding partial state."
- *Errors are findings, not parse failures* (root engineering principle). The loader is deliberately tolerant: parse failures become `LoadError` findings; the tree is loaded as far as it can go. G14 added stub registration for failed-parse entities so reference resolution does not cascade.
- *Hooks are advisory, the verb is authoritative.* The pre-push hook runs `aiwf check`. It is a fast-fail courtesy; the verb's projection check is the load-bearing enforcement. If the hook were uninstalled tomorrow, aiwf would still be correct.

**One deliberate carve-out: `contractverify` is hook-only.** `aiwf check` runs both `contractcheck.Run` (config correspondence: bound schema/fixtures paths exist, ids reference actual contract entities, paths don't escape the repo) and `contractverify.Run` (executes the validator binary against fixtures to confirm the schema actually accepts the valid set and rejects the invalid set). Mutating verbs (`aiwf contract bind`, `aiwf add contract --validator …`) run **`contractcheck.Run` on their projection** (G18) — so a typo in `--schema` or `--fixtures` is caught at verb time, not push time. They do **not** run `contractverify.Run`. Three reasons make this defensible: (1) validator availability is per-machine — a contributor without `cue` installed gets a `validator-unavailable` warning, not a hard error, by design; (2) running every validator on every verb invocation would be expensive and brittle; (3) the actual schema-vs-fixtures verification is a runtime semantic check, not a structural invariant the engine owns. The hook is the right place for it.

**Concurrency.** Mutating verbs acquire `<root>/.git/aiwf.lock` via `flock(2)` before reading the tree, so two `aiwf` invocations on the same repo cannot race on id allocation. Read-only verbs do not lock; they may run concurrently with mutations and tolerate seeing a snapshot from before or after a commit.

---

## 3. Boundaries — what's in the engine, what's in the skills, what's in the consumer

Three boundaries that any change should respect.

### Engine ↔ skills

The engine produces the **mechanical guarantees** — id allocation, atomic commit, schema validation, reference resolution, status transitions, trailer protocol. These are enforced inside the verb at projection time and re-enforced by the pre-push hook.

The skills produce **agent guidance** — when to call which verb, how to phrase the user-confirmation step, how to read the verb's output back to a human. Skills are advisory: nothing in aiwf's correctness depends on the LLM remembering to invoke them. **If a guarantee depends on the LLM remembering to invoke a skill, it is not a guarantee.**

Concretely: if a behavior must hold for `aiwf` to be correct, it lives in the engine (and a unit test pins it). If a behavior is "what a well-behaved agent should do," it lives in a skill (and an unhelpful agent that ignores the skill produces a worse experience but not an incorrect one).

### Engine ↔ consumer repo

The engine is **stateless across invocations** — it reads the consumer's tree on every verb, computes everything in memory, writes back, exits. There is no engine cache, no engine state file, no engine-owned database. The consumer's git repo is the sole persistent state.

The engine's only consumer-side state-shaping action is `aiwf init`: it writes `aiwf.yaml`, scaffolds `work/` directories, materializes `.claude/skills/aiwf-*/`, and installs the pre-push hook. After that, every verb is "read tree, validate, write entity files, git commit, exit" — no engine-owned bookkeeping accumulates.

### Per-machine ↔ per-project

aiwf is split deliberately:

| Concern                                    | Where it lives                              | Why                                              |
|--------------------------------------------|---------------------------------------------|--------------------------------------------------|
| The engine binary                          | per-machine (via `go install`)              | the same binary serves every consumer repo       |
| Per-project policy and planning state      | in-repo (committed)                         | the team's choices belong with the team's code   |
| Materialized skill adapters                | in-repo (gitignored)                        | regenerated by `aiwf init`/`update`; cache, not state |
| Validator binaries (e.g. `cue`, `jq`)      | per-machine                                 | a contributor without `cue` gets a warning, not a hard error (see [`design/design-decisions.md`](design/design-decisions.md) for the `strict_validators` rationale) |

Adding a "per-organization" or "per-host" layer is **not** in the PoC. The two layers (per-machine engine, per-project state) are sufficient until a third demonstrates need.

---

## 4. The published surface

What a contributor or skill author can rely on. This is the public contract; everything else is internal and may change.

| Surface                          | Access                                       | Stability                  |
|----------------------------------|----------------------------------------------|----------------------------|
| The verb surface                 | `aiwf <verb> [args]` (`aiwf help` lists all) | additive; verbs aren't removed |
| Per-kind frontmatter contract    | `aiwf schema [kind]` (text or `--format=json`) | pinned to the engine; runtime check enforces the same contract |
| Per-kind body template           | `aiwf template [kind]`                       | pinned; matches what `aiwf add` scaffolds |
| Trailer keys                     | `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`, `aiwf-prior-entity:`, `aiwf-prior-parent:` | locked; renaming would invalidate `aiwf history` for all existing commits |
| `aiwf check` finding codes       | `aiwf check --format=json`                   | additive; subcodes may be added, existing codes preserved |
| The JSON envelope                | `{tool, version, status, findings, result, metadata}` | shape locked; `result` payload varies per verb |
| Exit codes                       | `0` ok / `1` findings / `2` usage / `3` internal | locked |

**Not** part of the published surface: the layout of `internal/`, the names of internal functions, the wire format of `aiwf.lock`, the implementation of stub registration, the body of any test fixture.

For agents and AI scaffolders, the published surface plus the [skill-author guide](skill-author-guide.md) is the contract. Skills that hand-edit `aiwf.yaml`, hand-write commit trailers, or invent fields outside the schema are violating the contract regardless of whether they happen to work today.

---

## 5. Load-bearing principles

The three architectural principles distilled in [`design/design-lessons.md`](design/design-lessons.md), as one-line rules. They are part of the engineering surface, not just research notes — any change that violates one of them must be deliberate and surfaced.

1. **Identity is not location.** Cross-references anchor on stable ids (`E-19`, `M-002`, `D-014`), never on snapshot coordinates (commit hashes, projection hashes, sequence numbers). Coordinates are valid only as time-anchors *in addition to* an identity. Watch-points: every trailer takes an entity id; no future hash-based addressing layer may take a hash as input to `aiwf-entity:` or any cross-reference field. See [`design/design-lessons.md`](design/design-lessons.md) §1.

2. **Atomicity is a unit, not a sequence.** Every verb has exactly one atomicity boundary, drawn around the whole operation: the commit. There is no observable half-state. Verbs go through `Apply`; rollback logic lives inside `Apply`. Hooks are advisory; the verb must be correct without them. See [`design/design-lessons.md`](design/design-lessons.md) §2.

3. **Don't fight the substrate's vocabulary.** Adding aiwf vocabulary for genuinely new concepts (entity, projection, contract, gap, recipe) is fine. Replacing substrate vocabulary with framework-private synonyms is not — a commit is a commit, a branch is a branch, a file is a file. The deliberate exception is *event*, an aiwf-private aggregate meaning "trailer-bearing commit"; this carve-out is documented in `design/design-lessons.md` §6 sweep findings. See [`design/design-lessons.md`](design/design-lessons.md) §6.

If a proposed change forces a violation of any of these, treat it as a kernel-level decision and surface it explicitly — not a quiet refactor.

---

## 6. Pointers

- [`design/design-decisions.md`](design/design-decisions.md) — the seven kernel commitments and the "deliberately not in the PoC" list. The *why* behind §1–§4 above.
- [`design/design-lessons.md`](design/design-lessons.md) — the three principles and the vocabulary sweep findings. The *why* behind §5.
- Gap entities under `work/gaps/` (post-G38 dogfood migration). `aiwf status --kind gap` for open gaps; `aiwf show G-NNN` for one. Pre-migration history is archived at [`archive/gaps-pre-migration.md`](archive/gaps-pre-migration.md).
- [`plans/poc-plan.md`](archive/poc-plan-pre-migration.md) — the four sessions of work that produced the engine, plus the I1 contracts iteration that built on top.
- [`skill-author-guide.md`](skill-author-guide.md) — the contract for AI skill scaffolders. The published surface from §4 above, expressed as rules and a worked example.
- Root [`CLAUDE.md`](../../CLAUDE.md) — the engineering principles (KISS, YAGNI, no half-finished implementations).
- [`CLAUDE.md`](../../CLAUDE.md) — Go-specific rules (formatting, testing, error handling, CLI conventions, commit-trailer convention).
