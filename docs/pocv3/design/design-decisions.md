# PoC design decisions

This is a self-contained summary of what the PoC commits to and why. It is the design context that motivates the build plan in [`poc-plan.md`](../archive/poc-plan-pre-migration.md). The full research arc that produced these decisions lives on `main`; this branch deliberately does not include those documents to keep the working context focused.

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
- **Kernel functionality is AI-discoverable.** Every verb, flag, JSON envelope field, body-section name, finding code, trailer key, and YAML field is reachable through channels an AI assistant routinely consults: `aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, the kernel's CLAUDE.md, or the design docs cross-referenced from it. If an AI assistant has to grep source to learn a kernel capability, the capability is undocumented. New capabilities ship with their `--help` text and skill-level documentation alongside the implementation, not after.
- **Principal/agent provenance, scope as a first-class FSM, `--force` is human-only.** The kernel separates *who is accountable* (principal) from *who ran the verb* (operator/actor). When an agent acts under a human's authorization, the act references an authorize commit (a typed scope with its own lifecycle: `active | paused | ended`). Verb dispatch composes the entity FSM with the scope FSM (gating, not containment). `--force` requires a human actor — sovereign acts always trace to a named human. See [`provenance-model.md`](provenance-model.md) for the full model.

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
| Contract | `proposed`, `accepted`, `deprecated`, `retired`, `rejected` | `C-NNN` |

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
| Contract | `linked_adrs` | []id | no | ADRs that motivate the contract |

Timestamps (`created`, `updated`) are deliberately absent from frontmatter; `git log` carries them. Putting them in YAML would be redundant state and a future drift target.

**Body templates** are short section stubs written by `aiwf add`. They are starting points, not enforced structure:

| Kind | Body sections |
|---|---|
| Epic | `## Goal` / `## Scope` / `## Out of scope` |
| Milestone | `## Goal` / `## Acceptance criteria` (per-AC `### AC-N — <title>`) / `## Work Log` / `## Decisions made during implementation` / `## Validation` / `## Deferrals` / `## Reviewer notes` (expanded in I2) |
| ADR | `## Context` / `## Decision` / `## Consequences` |
| Gap | `## What's missing` / `## Why it matters` |
| Decision | `## Question` / `## Decision` / `## Reasoning` |
| Contract | `## Purpose` / `## Stability` |

Bodies are not validated. The framework guarantees structural and referential stability of frontmatter; prose is the human's responsibility.

### Stable ids and rename ergonomics

- IDs are sequential within a kind, allocated by scanning the tree at allocation time and picking `max + 1`. The allocator reads two trees: the caller's working tree and the configured trunk ref (default `refs/remotes/origin/main`, overridable via `aiwf.yaml: allocate.trunk`). Scanning trunk closes the dominant collision case ("forgot trunk has moved on"). The remaining residual — two feature branches that diverged before either pulled the other — is caught at pre-push by the `ids-unique` check (which also reads the trunk ref) and resolved by `aiwf reallocate`, with rename history preserved in a `prior_ids: []` frontmatter list so audit trails migrate readably with the entity. The model aligns with how developers think about code merge conflicts: trunk is the coordination point; branch-to-branch divergence is the operator's responsibility to merge, just like code. Full mechanics in [`id-allocation.md`](id-allocation.md).
- The id is encoded in the file or directory path (`E-19-<slug>/`, `M-001-<slug>.md`). Because the slug is part of the path, two parallel branches that allocate the same id for different titles produce different paths — git merges both files in cleanly. The collision is *semantic* (two files share an id in their frontmatter), not textual, and surfaces only when `aiwf check` runs. This slug-as-collision-buffer property is what makes the simple allocator viable without coordination.
- Renames preserve the id: `aiwf rename <id> <new-slug>` does `git mv` plus a title update.
- Removals are not deletions. `aiwf cancel <id>` flips status to the kind's terminal value (`cancelled`/`wontfix`/`rejected`/`retired`). The file stays. References stay valid.
- Collisions are detected by `aiwf check`'s `ids-unique` finding. The pre-push hook makes this fatal before push. Resolution is `aiwf reallocate`, which accepts either an id (when unambiguous) or a path (required when two entities collide on the same id). It picks the next free id (`max + 1` at call time), `git mv`s, walks every entity's frontmatter to rewrite reference fields, and surfaces body-prose references as findings for human review. The id format is never extended with suffixes (no `M-007a`/`M-007b`); collision recovery always renumbers.

### Markdown is the source of truth; git is the time machine

There is no separate event log file. There is no separate graph projection file. The markdown frontmatter is canonical state; `git log` is history; structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable. `aiwf history <id>` reads `git log` filtered by trailer.

The `reallocate` verb additionally writes an `aiwf-prior-entity: <old-id>` trailer alongside the new id's `aiwf-entity:`. This is the bridge that keeps both ids' histories complete: `aiwf history M-008` matches `aiwf-entity: M-008` and shows the reallocate as an event in M-0008's life; `aiwf history M-007` matches `aiwf-prior-entity: M-007` and shows the reallocate as the terminal event in M-0007's life ("renumbered to M-0008"). Without this, querying the old id after a reallocation would silently come up empty — a footgun, since the user would think the entity was deleted.

This is a deliberate departure from designs that maintain a parallel transaction log. The research that motivates this departure is on `main`; the short version is that an append-only event log file fights git's branching model in ways that are expensive to fix and unnecessary at this scale.

### Validation is the chokepoint

`aiwf check` is a pure function from the working tree to a list of findings. It runs as a `pre-push` git hook installed by `aiwf init`. The hook is what turns the framework's guarantees from suggestions into mechanical enforcement.

When the consumer repo's `aiwf.yaml` declares contract bindings, `aiwf check` also runs the contract verify+evolve passes — see *Contracts (added in I1)* below for the model. The pre-push hook is still the single chokepoint; contracts are an additional source of findings inside the same envelope, not a separate gate.

`--no-verify` bypasses the hook (standard git behavior). The framework does not try to prevent that — bypassing is sometimes the right call. But the *default* behavior is that broken state cannot be pushed silently.

### Contracts (added in I1)

Contracts ship as part of the PoC's mechanical-validation surface. The full design lives in [`contracts-plan.md`](../plans/contracts-plan.md); the load-bearing decisions, in summary:

- **Two things that share the word "contract."** The *contract entity* (`C-NNN`) is a registry record in `work/contracts/` — pure planning state, six kinds row 6. The *contract binding* lives in `aiwf.yaml.contracts.entries[]` — operational state pointing at a schema path, a fixtures-tree root, and a named validator.
- **The engine owns orchestration; the user owns validators.** aiwf never ships a `cue` or `ajv` binary. Validators are *declared* in `aiwf.yaml.contracts.validators` (a name → command + argv-template mapping); the user installs the binary via their normal toolchain. **Recipes** are embedded markdown content for common languages (CUE, JSON Schema, …) opt-in via `aiwf contract recipe install`, never selected as defaults.
- **Verify and evolve.** Each binding's fixtures live under `<fixtures>/<version>/{valid,invalid}/`. The verify pass runs every `valid/` fixture (must pass) and every `invalid/` fixture (must fail) at the current version. The evolve pass runs every historical `valid/` fixture against the *current* schema, catching silent breakage from schema changes.
- **Validator availability is a per-machine concern.** A contributor without `cue` installed gets a `validator-unavailable` *warning*, not a hard error — the framework's enforcement should not depend on every developer's local toolchain. Teams that want stricter behavior set `aiwf.yaml.contracts.strict_validators: true`. See G3 in [`gaps.md`](../archive/gaps-pre-migration.md) for the rationale.
- **Verb surface for contract bindings.** No hand-editing of `aiwf.yaml` is required. `aiwf contract bind/unbind` mutate entries; `aiwf contract recipe install/show/remove` mutate validators; `aiwf contract verify` runs the passes; `aiwf add contract --validator … --schema … --fixtures …` does atomic add+bind in one commit. Every mutation produces exactly one git commit with the standard trailers.

### Acceptance criteria and TDD (added in I2)

ACs are first-class but namespaced inside their milestone, addressable as `M-NNN/AC-N`. They are not a seventh entity kind; they are structured sub-elements of the milestone with composite ids, validated by `aiwf check` and reachable from `aiwf history`. The full design lives in [`acs-and-tdd-plan.md`](../plans/acs-and-tdd-plan.md); the load-bearing decisions, in summary:

- **Why namespaced, not a seventh kind.** ACs are composed by exactly one milestone — they cannot be moved between milestones without losing meaning, they are numbered relative to the milestone (not globally allocated), and their lifecycle is bounded by it. That's composition, not reference. A seventh peer-level kind with global ids (`AC-1234`) would invert the relationship; sub-scoping under the milestone preserves it.
- **Composite id grammar:** `M-NNN/AC-N`. AC ids are allocated per-milestone starting at 1; no global allocator. The slash extends the existing prefix-letter id grammar by one alternation; YAML-safe unquoted; CLI-clean.
- **AC ids are position-stable.** `acs[i].id == "AC-{i+1}"` for every index. Cancelled ACs stay in `acs[]` at their original position (status flip, not deletion); the allocator picks `max+1` over the full list including cancelled entries. Mirrors the milestone/epic id-stability model: composite-id references always resolve.
- **Frontmatter additions on milestone:**
  - `tdd: required | advisory | none` (default `none` when absent) — opt-in policy.
  - `acs: [{id, title, status, tdd_phase}]` — structured list. Ids are `AC-1`, `AC-2`, …, sequential per milestone (cancelled entries count toward position). The body carries a matching `### AC-N — <title>` heading per AC for prose.
  - On the Go side, every field is a plain `string` with `omitempty`; empty == absent. Closed-set membership rules out `""` as a legal value, so the sentinel is unambiguous.
- **Closed status sets:**
  - AC status: `open | met | deferred | cancelled`. `deferred` and `cancelled` are terminal; `met` may move to `deferred`/`cancelled` if scope changes after the fact.
  - TDD phase: `red | green | refactor | done`. Linear; required only when milestone `tdd: required`; tolerated as absent or any value when `tdd: none`.
- **`aiwf add ac` seeds the initial phase.** When the parent milestone is `tdd: required`, the verb writes `tdd_phase: red` as part of the same commit that creates the AC; otherwise it leaves `tdd_phase` absent. The kernel never makes an implicit TDD-policy decision — it just writes the only legal starting state under the FSM.
- **One audit rule across the two.** When milestone `tdd: required`, AC `status: met` requires `tdd_phase: done`. The kernel guards the *outcome*; the rituals plugin's `wf-tdd-cycle` skill drives the *flow*. A human or any AI can satisfy the kernel without that skill installed.
- **`--force --reason "<text>"` allows any-to-any transition.** Reason required, not optional. Lands as a `aiwf-force: <reason>` trailer alongside the standard trailers. Force relaxes only the *transition* rule; coherence checks (id format, closed-set membership, ref resolution) still run.
- **Trailer schema extension.** All `promote` events (milestone *and* AC) now carry `aiwf-to: <state>` so the target state is in the structured trailer rather than buried in the commit subject. AC events reuse the existing verb: `aiwf-verb: promote / aiwf-entity: M-007/AC-1 / aiwf-to: met / aiwf-actor: …`. Rollout is forward-only: pre-I2 commits stay as they are, the reader does not parse subjects to infer a target, and `aiwf history` renders the target-state column as a dash for trailer-less rows. No backfill, no history rewrite.
- **Composite ids as reference targets.** Open-target fields (`gap.addressed_by`, `decision.relates_to`) accept `M-NNN/AC-N`. Closed-target fields (`milestone.parent → epic`, `adr.supersedes → adr`, etc.) are unchanged. The bare milestone id (`aiwf history M-007`) shows milestone events plus all AC events via prefix match anchored on the literal `/` boundary.
- **Milestone-done implies AC progress.** A milestone may not transition to `done` while any AC has `status: open`; `deferred` and `cancelled` are acceptable terminal AC states for a done milestone, with the body explanation as the documentation. Surfaces as the `milestone-done-incomplete-acs` finding (error), which runs on every `aiwf check` pass — not just on verb projection — so a milestone that became `done` via `--force --reason` while ACs were still open keeps surfacing the inconsistency until the ACs reach a terminal state. `--force --reason` overrides the verb-time refusal; the standing check still reports.
- **`aiwf rename` accepts composite ids.** `aiwf rename M-NNN/AC-N "<new-title>"` updates `acs[].title` in the parent milestone's frontmatter and rewrites the matching `### AC-N — <title>` body heading in one commit. The bare-id form keeps the existing path-rename behavior; the verb dispatches on composite-vs-bare.
- **What `aiwf check` enforces (new findings):**
  - `acs-shape` (error) — frontmatter `acs[]` items have valid `id` (`AC-N`, position-equal including cancelled entries), `status` in the closed set, `tdd_phase` in the closed set when present.
  - `acs-body-coherence` (warning) — every frontmatter AC has a matching `### AC-<N>` heading in body, and vice versa. Pairs by id only, not by title text — body title is prose and remains kernel-blind. The body heading regex is permissive: em-dash, hyphen, colon, or id-only forms all parse.
  - `acs-tdd-audit` (error when `tdd: required`; warning when `advisory`) — every AC with `status: met` has `tdd_phase: done`.
  - `acs-transition` (error, on verb projection) — refuses illegal AC-status or `tdd_phase` transitions unless `--force --reason` is supplied.
  - `milestone-done-incomplete-acs` (error) — fires on every `aiwf check` pass when a milestone has `status: done` and at least one AC has `status: open`.
- **What's not a kernel rule.** No "milestone must have ≥1 AC" — ACs remain optional. No "milestone can't enter `in_progress` without all ACs in `red`" — the kernel guards the outcome (`met` requires `done`), not the entry. No global AC allocator. No AC tombstone beyond status-cancel. No `aiwf doctor` warnings about deleted skill names from the rituals shrink — the kernel stays uncoupled from rituals plugin internals.

`STATUS.md` and `aiwf show <id>` render AC progress per milestone; `aiwf history M-NNN/AC-N` filters trailers by composite id and shows the AC's red-green-refactor sequence and status changes.

The rituals plugin (`ai-workflow-rituals`) shrinks to match: `wf-tdd-cycle` stays (process prose for red-green-refactor; now drives `aiwf promote --phase` so its work shows up in `aiwf history`); `aiwfx-track` is removed (its job is now a kernel-validated section of the milestone doc); `aiwfx-start-milestone` and `aiwfx-wrap-milestone` absorb its workflow guidance; the separate `tracking-doc.md` template merges into `milestone-spec.md`.

### Provenance model (added in I2.5)

I2.5 sits between I2 (acceptance criteria + TDD) and I3 (governance HTML render). It addresses a gap that pre-I2.5 design conflated: the operator (who ran the verb) and the accountability-bearer (whose judgment authorized it). With humans, this is the same person; with LLM-mediated work, it isn't. The full design lives in [`provenance-model.md`](provenance-model.md); the build plan in [`provenance-model-plan.md`](../plans/provenance-model-plan.md). Load-bearing decisions, in summary:

- **Identity is runtime-derived, not stored.** `aiwf.yaml.actor` is removed; the operator identity comes from `git config user.email` (with `--actor` override). This fixes the multi-clone bug where every developer inherited whoever ran `aiwf init`'s identity. Each checkout, each worktree, each CI runner naturally gets the right actor.
- **Three-layer trailer set.** `aiwf-actor:` (operator, unchanged) + `aiwf-principal:` (accountability-bearer, required when actor is non-human) + `aiwf-on-behalf-of:` and `aiwf-authorized-by:` (scope membership, required-together). Plus `aiwf-scope:` (state event on authorize verbs) and `aiwf-scope-ends:` (auto-end markers). `aiwf-actor:` keeps its existing meaning — operator — so pre-I2.5 history reads unchanged.
- **Scope is a first-class FSM.** Closed states `active | paused | ended`. Authorization is a typed grant created by `aiwf authorize <id> --to <agent>`; pause/resume are first-class transitions. Multiple parallel scopes are supported. End is automatic when the scope-entity reaches a terminal status (recorded by `aiwf-scope-ends:` trailer on the terminal-promote commit). Strict end-on-terminal: un-canceling a scope-entity does not resurrect ended scopes.
- **Gating, not containment.** Entity FSMs and scope FSMs are orthogonal. A verb is allowed iff the entity-FSM transition is legal AND, for non-human actors, at least one active scope's reachability check passes. Reachability uses the same reference-graph index built for `aiwf show`'s `referenced_by`. For human actors with no `--principal`, scope checks are skipped — humans need no authorization to act.
- **`--force` is human-only.** Sovereign acts always trace to a named human. The kernel refuses `--force` from any actor whose role is not `human/...`. Forced acts always carry only the existing `aiwf-actor: human/...` + `aiwf-force:` trailers; no principal, no on-behalf-of. A future delegated-force flag is filed as G23 and intentionally deferred.
- **Provenance findings are first-class.** New `provenance-*` family of `aiwf check` codes catches malformed trailer combinations, stale authorization SHAs (three sub-cases: missing, out-of-scope, ended), non-human force, and out-of-scope agent acts. Verb-side refusals + standing-rule audits compose: every rule is enforced twice, catching both kernel-internal bugs and externally-authored commits.
- **Render integration.** Governance HTML's Provenance tab renders scopes-as-section: a top-level table listing every scope that touched the entity (auth SHA, agent, principal, opened, state, end date, event count); below, a chronological event timeline with `[scope-id]` chips. `aiwf history` text output uses `principal via agent` syntax in the actor column when they differ, with trailing scope chips.
- **Open extensions deliberately deferred.** Explicit revoke verb, time-bound scopes, verb-set restrictions, pattern scopes, sub-agent delegation, bulk-import per-entity attribution, and delegated `--force` are filed as G22 (extension surface) and G23 (delegated force). YAGNI for the PoC; revisit when real friction shows up.

### Governance HTML render (added in I3)

A static-site generator (`aiwf render --format=html --out site/`) produces a per-repo governance page from canonical planning state. The render is a thin templating layer over `aiwf show <id> --format=json`; the kernel remains the truth layer. The full design lives in [`governance-html-plan.md`](../plans/governance-html-plan.md); the load-bearing decisions, in summary:

- **Static files only.** No `aiwf serve`, no runtime, no auth, no database. Open the directory locally, host on GitHub Pages, drop in S3 — portable to any web server. Read-only by design; the kernel is the source of truth and edits go through verbs.
- **Pure HTML + CSS, deterministic output.** No JavaScript framework, no client-side rendering, no build toolchain. Single Go binary writes a single static directory. Hand-written stylesheet under 5 KB, embedded in the binary via `embed.FS`. Output is byte-identical for the same input tree (no wall-clock timestamps; sorted map iteration; sorted directory walk) — pinned by a "render twice, byte-compare" test.
- **Output goes to `site/` at repo root, with the gitignore as a derived artifact.** Standard static-site-generator convention. Configurable via `aiwf.yaml.html.out_dir` and `--out`. User intent is expressed in `aiwf.yaml.html.commit_output` (bool, default `false`). When `false`, `aiwf init`/`update` add `out_dir` to the framework-managed gitignore block; when `true`, both verbs remove it from the block. The gitignore is a derived artifact — a user manually toggling the line is replaced by the YAML field. Generated artifacts never live under `work/`.
- **Site shape:** `index.html` (epics list), `<epic-id>.html` (epic overview with milestones table + dependency DAG + linked entities), `<milestone-id>.html` (six tabs: Overview, Manifest, Build, Tests, Commits, Provenance). Per-AC content is rendered inline in the milestone Manifest tab and addressable via `#ac-N` anchors; a separate per-AC page is out of scope until real friction shows up.
- **Tabs use `:target`-driven CSS show/hide.** Each tab is a `<section>` with an `id`; navigation is a strip of in-page anchor links; the standard CSS-default-tab sibling trick shows Overview on the bare URL. Bookmarkable per-tab URLs; browser back/forward switches tabs naturally; no JS.
- **Tabs render existing data.** Provenance is actor-centric audit (who did what, with `--force` overrides highlighted with reason); Build is the per-AC TDD timeline from phase trailers; Tests renders metrics from the new `aiwf-tests:` trailer with a visible `strict` / `advisory` policy badge. No new entity kinds.
- **`aiwf-tests:` trailer (added in I3) — kernel-owned write path.** Optional commit trailer carrying per-cycle test metrics: `pass=N fail=N skip=N`. Loose key=value format on read; write-strict (recognized keys + non-negative integers) at the kernel verb. Phase-promoting verbs accept `--tests "key=value …"` and write the trailer in the same commit; the rituals plugin's `wf-tdd-cycle` skill calls into this rather than constructing the trailer itself, keeping a single write path. Solo users and CI scripts use the kernel flag directly. Aggregation is "first commit returned by `aiwf history M-NNN/AC-N` carrying the trailer is authoritative" — rebase- and amend-stable.
- **`acs-tdd-tests-missing` is opt-in.** New `aiwf.yaml.tdd.require_test_metrics` (bool, default `false`). Without the opt-in, the kernel emits no finding for a missing trailer — kernel correctness is decoupled from rituals-plugin install state. With the opt-in, missing trailers on `tdd_phase: done` ACs in `tdd: required` milestones produce a warning. The Tests tab's policy badge renders the active mode visibly.
- **JSON completeness on `aiwf show` is the precondition.** The HTML render is templating over the JSON envelope; the JSON must carry full frontmatter, body sections parsed into named blocks, scoped findings, forward references, reverse references (`referenced_by`), and full trailer parsing on history. The reverse-ref index lands as an I2 step (it benefits `aiwf check` audits and `aiwf show` independently of the render); the body parser lands as I3 step 1.
- **What's deliberately out of scope.** No `aiwf serve`, no interactivity, no JS framework, no test-runner integration (Tests tab renders what was *committed* during the TDD cycle, not CI runs), no Mermaid.js / mmdc dependency, no diagrams or image embedding, no GitHub Issues / Linear sync, no authentication or multi-tenant features. The render is one renderer alongside `aiwf status --format=md` and `aiwf show --format=json`, not a separate product.

### Layered location-of-truth

| Layer | Location | Why |
|---|---|---|
| Engine binary (`aiwf`) | Machine-installed via `go install` | Standard tool distribution; same as `git`, `ripgrep`, `jq` |
| Per-project policy (`aiwf.yaml`) | In the consumer repo, git-tracked | Team-shared, CI-readable, travels with clone |
| Per-project planning state (`work/`, `docs/adr/`) | In the consumer repo, git-tracked | Co-evolves with code, bisectable, no API friction |
| Per-developer config | `~/.config/aiwf/` | Personal preferences and tool-path overrides |
| Materialized skill adapters (`.claude/skills/aiwf-*`) | In the consumer repo, gitignored | Composed from the binary on `aiwf init`/`update`; stable across `git checkout`. The `aiwf-` prefix is the namespace boundary; non-`aiwf-*` skill directories are untouched. |
| Marker-managed git hooks (`.git/hooks/pre-push`, `.git/hooks/pre-commit`) | In the consumer repo, untracked | Composed from the binary on `aiwf init`/`update`; identified by an `# aiwf:<hook>` marker on the first content line so user-written hooks are left alone. |

The materialization invariant is load-bearing: artifacts are regenerated only on explicit `aiwf init` / `aiwf update`, never implicitly on `git checkout` or every verb invocation. This is what keeps the AI's behavior stable when switching branches. The on-disk files are a cache, not state: `aiwf update` wipes every `.claude/skills/aiwf-*/` directory and rewrites them from the binary's embedded skills, and refreshes every marker-managed hook from its embedded template; `aiwf doctor` reports drift via byte-compare against the embedded versions. No state file, no manifest, no version stamp.

`aiwf update` is the **upgrade verb**: it refreshes every marker-managed framework artifact the consumer is opted into — embedded skills, embedded git hooks, and any future templated artifact the framework ships. `aiwf init` is first-time setup that runs the same refresh pipeline at the end. Re-running either verb converges to the same state for a given binary version + `aiwf.yaml`. (Earlier in the PoC, `aiwf update` refreshed only skills; the broadening landed in `update-broaden-plan.md`.)

Skills are embedded in the `aiwf` binary via Go's `embed.FS` and copied out on `init` / `update`. This deliberately couples skill content to the binary version: skills are adapters that call binary-provided commands, so version-skew between them would silently break things. Distributing skills via a separate channel — e.g., as a Claude Code plugin — is a viable future *packaging* path for easier installation, but as an architecture choice it would re-introduce the version-skew problem that embedding avoids.

### Release and upgrade

Releases are git tags on the kernel repo. The Go module proxy (`proxy.golang.org`) reads tags directly: `go install github.com/23min/ai-workflow-v2/cmd/aiwf@v0.1.0` resolves the tag, `@latest` resolves to the highest semver tag, and `GET https://proxy.golang.org/<module>/@latest` returns the latest version as JSON without cloning. The running binary's own version comes from `runtime/debug.ReadBuildInfo()` — `v0.x.y` for tagged installs, `(devel)` for working-tree builds, a pseudo-version for `@main` installs. No release infrastructure beyond `git tag && git push --tags`.

`aiwf upgrade` is the one-command upgrade flow: it shells out to `go install <module>@<version>` (default `@latest`, override with `--version`), then re-execs the freshly-installed binary to run `aiwf update` in the consumer repo. The upgrade verb composes the two pieces (`go install` + the existing `aiwf update`) so a consumer never needs to remember the two-step ritual. `--check` reports the comparison without installing; `--yes` skips the confirmation prompt. The verb is the only place in the kernel that hardcodes the module path.

Version skew shows up in three rows of `aiwf doctor`, all advisory:

1. **Binary version** — always shown, from `ReadBuildInfo()`. Distinguishes tagged / devel / pseudo.
2. **`aiwf.yaml` pin coherence** — when `aiwf_version:` is set, compares to the running binary. Mismatch is a notice, not a hard fail; the pin records intent, not enforcement. Hardening it (refuse to run when binary < pin) is a separate, deliberate decision filed for later.
3. **Latest published** — opt-in via `aiwf doctor --check-latest`. Hits the module proxy with a 3s timeout; honors `GOPROXY=off`; network errors print "unavailable" without failing doctor. Off by default so `aiwf doctor` stays fast and offline.

The split between (2) and (3) is load-bearing: (2) is local-only and always available; (3) requires the network and is opt-in. Together they answer the two questions a user actually has — *am I matched to what this repo expects?* and *am I matched to the world?* — without making either a hard gate.

The full plan lives in [`upgrade-flow-plan.md`](../plans/upgrade-flow-plan.md).

### `aiwf.yaml` config

A short YAML file at the consumer repo root. Read by `aiwf` on every invocation; written by `aiwf init`. The file's presence is also how `aiwf` discovers the repo root: it walks up from the current working directory until it finds one.

| Field | Type | Required | Notes |
|---|---|---|---|
| `aiwf_version` | string | yes | Engine version the repo expects (e.g., `0.1.0`). `aiwf doctor` warns on mismatch. |
| `hosts` | []string | no | Hosts to materialize skills for. PoC default and only supported value: `[claude-code]`. |
| `contracts` | mapping | no | Contract bindings: a `validators` mapping (name → command + args), an `entries` list (each with `id`, `validator`, `schema`, `fixtures`), and `strict_validators` (bool, default false). Owned and round-tripped programmatically by `aiwf contract bind/unbind/recipe …`. See [`contracts-plan.md`](../plans/contracts-plan.md) §5. |
| `status_md` | mapping | no | `auto_update` (bool, default true) — install a marker-managed pre-commit hook that regenerates `STATUS.md` (a committed `aiwf status --format=md` snapshot) on every commit. Set to `false` to opt out; `aiwf init`/`update` will then leave the hook uninstalled and remove a previously-installed marker-managed one. The committed `STATUS.md` itself is the user's content once tracked — flipping the flag does not delete it. See [`update-broaden-plan.md`](../plans/update-broaden-plan.md). |
| `html` | mapping | no | `out_dir` (string, default `site`) — render output directory relative to repo root; `commit_output` (bool, default `false`) — when `false`, `aiwf init`/`update` add `out_dir` to the framework-managed gitignore block; when `true`, both verbs remove it from the block. The gitignore is a derived artifact controlled by this field. See [`governance-html-plan.md`](../plans/governance-html-plan.md) §2. |
| `tdd` | mapping | no | `require_test_metrics` (bool, default `false`) — opt in to the `acs-tdd-tests-missing` warning. When `true` and a milestone is `tdd: required`, ACs in `tdd_phase: done` whose first commit returned by `aiwf history` lacks an `aiwf-tests:` trailer produce a warning. When `false`, the trailer is purely informational and absence is not a finding. See [`governance-html-plan.md`](../plans/governance-html-plan.md) §4. |
| `doctor` | mapping | no | `recommended_plugins` (list of `<name>@<marketplace>` strings, default empty) — Claude Code plugin identifiers the consumer expects to be installed for this repo's project scope. `aiwf doctor` reads `~/.claude/plugins/installed_plugins.json` and emits one `recommended-plugin-not-installed` warning per declared entry that has no matching project-scope install. Each entry shape is validated at load time: `<name>@<marketplace>`, both sides non-empty, no whitespace. Empty list (or absent block) means the check makes zero observations — the kernel makes no assumption about which plugins a consumer "should" have. See M-0070 (E-0018). |

Example (the typical file with no contracts):

```yaml
aiwf_version: 0.1.0
```

That's the entire file in normal use. `hosts` is omitted to take the default. The actor identity is derived at runtime from `git config user.email` (per `provenance-model.md`); `--actor` overrides per invocation. `contracts:` is added by `aiwf contract recipe install` and `aiwf contract bind` — no hand-editing required. No project-name field, no per-project skill paths, no other policy knobs in the PoC; the kind FSM, id formats, and status sets are hardcoded in the engine — see *Six entity kinds*.

### One git commit per mutating verb

Every mutating verb produces exactly one git commit, or no change at all. The current set: `add`, `promote`, `cancel`, `rename`, `move`, `reallocate`, `import`, `update`, `render --write`, `init`, `contract bind`, `contract unbind`, `contract recipe install`, `contract recipe remove`. Verbs are *validate-then-write*: the verb computes the projected new tree in memory (an overlay on top of the loaded tree), runs `aiwf check` against the projection, and only when the projection is clean writes files (and `git mv`s) and creates the commit. On findings the working tree is never touched.

This gives per-mutation atomicity for free in the happy path, and the `Apply` orchestrator wraps the file-touch + commit sequence in a deferred rollback so a partial failure (write error, commit refused) leaves the working tree exactly as it was. There is no separate journal file, no two-phase commit ceremony, no event-log-then-confirm protocol. The git commit *is* the atomic boundary.

To prevent two `aiwf` mutations on the same repo from racing on id allocation, every mutating verb acquires an exclusive lock on `<root>/.git/aiwf.lock` (POSIX `flock`) before reading the tree. Read-only verbs (`check`, `history`, `status`, `render` without `--write`, `doctor`, `whoami`) do not lock — they remain free to run concurrently with mutations. See G4 in [`gaps.md`](../archive/gaps-pre-migration.md) for the rationale.

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
