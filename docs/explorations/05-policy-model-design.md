# Policy model

This is the design proposal for **policy** as an aiwf primitive: a normative claim about state or process, written down with authority, intended to bind future actors — humans and AI agents alike. Policy is an **opt-in module**: a consumer who has not added `policy` to their `aiwf.yaml` `modules:` list sees no `work/policies/` scaffolding, no policy verbs in `aiwf --help`, no policy skill materialized into the AI host. When opted in, policy becomes a seventh entity kind whose subject is the *rules under which work happens* rather than the work itself.

The proposal is deliberately KISS-cut: six kernel commitments, the rest reserved namespace, frontmatter forward-compatible, earned through observed friction rather than speculation.

If a proposed change conflicts with anything below, treat it as a kernel-level decision and surface it explicitly.

---

## What a policy is

A policy is a normative claim with three load-bearing properties:

1. **Unilateral.** A team's commitment about how work proceeds, not a bilateral promise to a counterparty (that is what contracts are for).
2. **Mechanically evaluable by a shared engine.** Unlike ADRs and decisions — which are judgment-shaped and rely on human reading — a policy carries a pointer to an evaluator that runs deterministically over project state. One engine evaluates all policies; the policy authoring surface is shaped to feed that engine.
3. **Lifecycled independently of the work it constrains.** A policy outlives any single milestone or ADR; it is the rule, not the case.

The audience separation against existing kinds is genuine: ADRs are the architectural record, contracts are interface promises, decisions are scope commitments, gaps are open questions. A policy is none of these, and cramming it into one would dilute the kind that absorbed it. The deeper reason a policy is its own kind is the engine commitment: only policies declare an evaluator the framework runs uniformly over the corpus.

## Why a primitive

The framework already has policies in everything but name. Engineering rules are scattered across `CLAUDE.md`, prescriptive sentences inside skills, MUST/SHOULD claims smuggled into ADR bodies, ad-hoc lint rules, and the engine's own `internal/invariants/` package (which audits the kernel itself; narrower than what consumers need, hence the distinct name). They are all the same shape — a normative claim with provenance, a lifecycle, and an enforcement story — and they all suffer the same failure modes when they live as scattered prose:

- **No queryable surface.** An LLM about to compose an action has no way to ask "what claims apply to this action?" without reading every doc the project ships.
- **No shared engine.** Each lint rule, each smuggled MUST, each contract validator has its own runner. There is no single place to ask "what failed and why."
- **No lifecycle.** When a rule is wrong, the act of replacing it is a pull-request edit, not a supersession event with a recorded rationale.
- **No provenance trail.** When a rule is invoked, ratified, or waived, there is no per-event record of who invoked it and why.
- **No scale story.** A repo with twenty rules can keep them in `CLAUDE.md`. A repo with two hundred cannot — the LLM context budget is the binding constraint, and prose-bulk does not survive it.

The primitive's job is to make claims **first-class**: addressable by id, lifecycled with status transitions, tagged with applicability metadata so retrieval is cheap, evaluated by a shared engine at known trigger points, summarized in a fixed-format digest the LLM consults for situational awareness, and provenanced through the same trailer system every other aiwf entity uses.

---

## Opt-in via `aiwf.yaml`

The framework's modules split cleanly into two groups:

- **Baseline modules**, always active: `epic`, `milestone`, `adr`, `gap`, `decision`. The framework's reason for existing; opting out doesn't make sense.
- **Opt-in modules**, off by default: `contract`, `policy`. Real cost (additional vocabulary, file-tree footprint, learning curve); paid only by repos that need them.

A consumer turns a module on by listing it under `modules:` in `aiwf.yaml`:

```yaml
# aiwf.yaml
modules:
  - policy        # opt in to the policy module
  # contract is omitted; consumer doesn't use bilateral-interface machinery yet
```

If `modules:` is missing or empty, only baseline modules are active. The framework reads this file at startup; opt-in verbs from inactive modules don't appear in `aiwf --help`, return "module not enabled" if invoked directly, and verb-time hooks in baseline modules see the inactive state and proceed without consultation, at zero cost.

`aiwf init` re-run is idempotent: editing `aiwf.yaml` to add `policy`, then re-running `aiwf init`, scaffolds `work/policies/`, creates the empty `.aiwf/policy-index.json` and `.aiwf/policy-digest.md`, materializes the `aiwf-policy` skill into the consumer's AI host (e.g., `.claude/skills/wf-policy/SKILL.md`), and registers the verb-time and CI hooks. Removing `policy` from `modules:` and re-running `aiwf init` deactivates: the materialized skill is removed, hooks are deregistered, files under `work/policies/` are left in place (so opting back in restores the corpus).

The same shape applies to `contract`: opt in via `aiwf.yaml`, opt out by removing it. Either module can be enabled without the other; both can be enabled together; neither needs to be.

---

## Bundles

A **bundle** is a named collection of policies grouped together for findability and distribution. Bundles are how the framework ships pre-curated policy sets and how consumers organize their own corpus.

The model is deliberately minimal:

- **Bundle = parent directory name.** A policy at `work/policies/aiwf-rituals/P-014-closing.commit.md` is in the `aiwf-rituals` bundle. The path is the truth for the canonical home; no manifest, no separate registry.
- **Every policy lives in some bundle directory.** Policies that don't fit a named bundle live under `work/policies/local/`; `local` is the conventional name for the consumer's unsorted/project-specific policies but it is just a directory like any other (no implicit-vs-explicit handling, no special cases). A loose policy file directly under `work/policies/` (no bundle parent) is a tree-discipline error.
- **Multi-bundle membership via `labels:`.** A policy may carry an optional `labels:` array in frontmatter listing every bundle it belongs to. When present, the first entry MUST equal the parent directory name (the canonical bundle is always written first); subsequent entries declare additional bundles. A policy in `dotnet-stack/` with `labels: [dotnet-stack, security-baseline]` is queryable from either bundle. Single-bundle policies omit `labels:` entirely.
- **No version pinning.** Re-syncing a bundle is `aiwf policy sweep --source bundle:<name>`; the LLM dedups against the local set, triage decides what lands. Upstream changes never auto-apply.

### Layers (documentation convention)

Bundles fall into four conceptual layers:

| Layer | Audience | Source |
|---|---|---|
| `kernel` | Framework-shipped, broadly applicable | aiwf engine bundles (`meta-policies`, `aiwf-rituals`, `aiwf-provenance`) |
| `workflow` | Framework-shipped, opt-in workflow patterns | aiwf engine bundles |
| `stack` | Toolchain-specific (`.NET`, Rust, TypeScript-with-Svelte, …) | consumer-authored or community-pulled — **the framework does not ship stack bundles** |
| `domain` | Project-specific, applies only in this repo | consumer-authored (`engine-correctness`, `numeric-safety`, …) |

The layer is documentation, not data. There is no `--layer` filter in v0; bundles are listed by name and counted by member.

**The framework does not ship stack bundles.** Policies that target a specific toolchain are stack-flavored: the consumer authors them under `work/policies/<name>/` or pulls them from a community source via `aiwf policy sweep`. The framework's role for stack-shaped concerns is *discovery*, not authoring — see §Stack discovery.

### Stack discovery

`aiwf init` scans the repo root for known manifest files (`*.csproj`/`*.sln` → `dotnet`, `Cargo.toml` → `rust`, `go.mod` → `go`, `package.json` → `node` or `typescript` if `tsconfig.json` is also present, `pyproject.toml`/`requirements.txt` → `python`, `Gemfile` → `ruby`, …) and writes the detected set to `aiwf.yaml`:

```yaml
# aiwf.yaml
stack: [dotnet, typescript]   # auto-populated by aiwf init; consumer-editable
```

The field is multi-valued because polyglot repos are common. Re-running `aiwf init` re-detects and rewrites the array; consumer hand-edits between runs are overwritten (the field is data, not config — if a consumer needs to assert a stack the detector misses, they can also hand-edit and skip future re-detection by removing the auto-populated marker). v0 commits to detection and recording only; **no v0 verb consumes the field**. The data is collected so future use as a mining hint (suggest stack-flavored community bundles), an applicability axis ("applies in repos with `stack: [dotnet]`"), or an LLM context signal can be designed from observation rather than speculation. See §What v0 does not commit to → Stack-discovery consumption.

### Per-repo divergence is expected

Two consumer repos that both pull `dotnet-stack` will diverge. Triage decides what lands in each repo, and local additions accumulate over time. That divergence is the point — bundles are *starting points*, not contracts the consumer is locked to.

### Discoverability

`aiwf policy bundles list` lists every bundle with a count of member policies, where a bundle is any parent directory under `work/policies/` plus any distinct value appearing in a `labels:` array. A separate verb so the listing surface is uniform regardless of how a consumer organized their tree.

### Why opt-in over always-on

- **Progressive disclosure.** A consumer with twenty rules in `CLAUDE.md` does not need a policy module. A consumer with two hundred does. The 7th kind should only be paid by repos that need it.
- **Smaller blast radius for v0.** The design hardens against real use; opt-in lets the framework ship the kind without imposing it on every existing consumer.
- **Clean boundary.** Modules are already the framework's unit of self-containment. Policy fits the existing pattern; nothing about activation is policy-specific.

### Repo layout when opted in

```
<repo-root>/
├── aiwf.yaml                    # human-authored: modules, runners, top-level config
├── .aiwf/                       # framework-owned: regenerated artifacts, drift-checked
│   ├── policy-index.json
│   ├── policy-digest.md
│   └── .gitignore               # ships with init; gitignores caches but not committed projection state
└── work/                        # human-authored markdown entities
    ├── epics/
    ├── milestones/
    └── policies/                # only present when policy module is enabled
        ├── aiwf-rituals/        # bundle directory (see §Bundles)
        │   └── P-014-closing.commit.md
        ├── meta-policies/
        │   └── P-027-trailers.required.md
        ├── local/               # unbundled policies live here (just a directory like any other)
        │   └── P-200-engine-correctness.md
        └── draft/
            ├── aiwf-rituals/    # mining drafts staged by bundle; triage promotes
            └── local/           # unbundled mining drafts
```

Three audiences cleanly isolated: human-authored config (`aiwf.yaml`), framework-owned generated artifacts (`.aiwf/`), human-authored entities (`work/`). The `.aiwf/` directory is framework-owned regardless of which modules are enabled (it's the engine's working space); enabled modules contribute artifacts to it.

---

## Kernel commitments

Six commitments. Every other behavior is consequence or convention.

1. **Policy is an entity kind with frontmatter shape locked at write time.** Six required fields, two optional (`labels`, `enforcement`). The shape is forward-compatible with the deferred features in §What v0 does not commit to (additive fields land when they earn their weight).
2. **The status set is `proposed | in-effect | retired`.** Three states, one terminal. Lifecycle expansion (`waived`, `superseded`) is reserved namespace.
3. **An index manifest (`.aiwf/policy-index.json`) is regenerated by every policy-mutating verb in the same commit.** Retrieval queries read the index, never the policy tree. The index is the property that makes feedforward retrieval cheap at thousand-policy scale.
4. **Every policy declares a `surface` (`digest | on-demand`), independent of any enforcement choice.** The digest carries policies with `surface: digest`. Surface and enforcement are orthogonal axes; digest size is bounded by editorial choice.
5. **Every policy with non-null `enforcement` is evaluated by one shared engine (`aiwf verify --kind policy`, with `aiwf policy verify` as a day-one alias).** The same engine, same index, same runner dispatch, same finding format runs at three scopes — verb-time (post-mutation entity), pre-commit (staged diff), CI (full tree or PR diff) — with auto-measured runner placement letting verb-time and pre-commit run only the runners that fit their budget. Three triggers, one mechanism. This is the property that distinguishes policy from ADR and decision; the engine is what justifies the kind.
6. **Mining is a first-class verb (`aiwf policy sweep`) that produces drafts, not active policies.** Drafts go to `policies/draft/` and require `aiwf policy triage` to promote. The consumer always reviews.

Together these solve scale (1, 3, 4), feedforward retrieval (1, 3, 4 with skill-side discipline §10), deterministic enforcement (5), and bootstrap (6).

---

## The kind shape

### Recognized path

```
work/policies/<bundle>/P-NNN-<slug>.md            ← bundled policy
work/policies/local/P-NNN-<slug>.md               ← unbundled policy (local/ is just a directory)
work/policies/draft/<bundle>/P-NNN-<slug>.md      ← mining output, awaiting triage
work/policies/draft/local/P-NNN-<slug>.md         ← unbundled mining output
```

The bundle is the parent directory name. Files directly under `work/policies/` (no bundle parent) are tree-discipline errors. See §Bundles.

`<slug>` matches `^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$` — a dotted-name with at least two segments, lower-case, alphanumeric. Examples: `ac.required`, `closing.commit`, `trailers.required`, `nolint.rationale`. The slug is the policy's stable human-readable handle: it appears in finding codes, in citations (`per ac.required`), and in the digest's second column. Slugs are corpus-unique (two policies in different bundles cannot share a slug). The slug is author-chosen at `aiwf policy add` time and validated against the format; once written, it is immutable in v0 (rename = retire-and-create; a `aiwf policy rename-slug` verb is reserved namespace).

The `draft/` subdirectory is the staging area for mined candidates; the loader recognizes these as `policy-draft` and treats them as a separate kind for tree-discipline purposes. Drafts mirror the bundle layout — every draft lives under `draft/<bundle>/`, with `draft/local/` for drafts that don't target a named bundle.

### Frontmatter

```yaml
---
id: P-001
status: in-effect           # proposed | in-effect | retired
surface: digest             # digest | on-demand
severity: error             # error | warn
summary: "Every milestone in `done` MUST have all acceptance criteria met."
labels: [aiwf-rituals, kernel-musts]   # optional; multi-bundle membership; first MUST equal parent dir; omit when single-bundle
applicability:
  kinds: [milestone]
  on_verbs: [promote]
  path: null                # optional glob; null = applies regardless of path
enforcement:
  ref: "cue vet ./policies/ac.cue -"   # advisory-only policies omit the enforcement block entirely
---
```

**Required fields.**

Six fields are required on every policy:

| Field | Type | Rule |
|---|---|---|
| `id` | string | Allocator-issued, format `P-NNN`, monotonic per allocator scan; LLM never invents. |
| `status` | enum | `proposed \| in-effect \| retired`. v0 closed-set; reserved values listed in §13. |
| `surface` | enum | `digest \| on-demand`. Determines whether this policy appears in `.aiwf/policy-digest.md`. Independent of enforcement. Default: `on-demand`. |
| `severity` | enum | `error \| warn`. Drives finding output and verb-time blocking when enforcement runs. (`info` is reserved; v0 ships only the two values that have load-bearing distinct meaning.) |
| `summary` | string | Single-sentence normative claim, RFC 2119 keyword preferred (`MUST`, `SHOULD`, `MAY`, `MUST NOT`). LLM-readable; loaded into context at retrieval. |
| `applicability` | object | Sub-fields below; required-together. |

**Optional fields.**

| Field | Type | Rule |
|---|---|---|
| `labels` | array of strings | Bundle membership beyond the parent directory. When set, the first entry MUST equal the parent directory name (the canonical bundle is always written first); subsequent entries are additional bundles. Omitted when the policy belongs to one bundle. Indexed alongside the directory bundle, so a policy in `dotnet-stack/` with `labels: [dotnet-stack, security-baseline]` is queryable from either bundle. |

**`applicability` sub-fields.**

| Sub-field | Type | Rule |
|---|---|---|
| `kinds` | array of strings | Closed-set: any subset of currently-enabled entity kinds. Empty array means "applies to no specific kind"; combined with `path`, expresses path-shaped policies that don't bind to entities. |
| `on_verbs` | array of strings | Closed-set: any subset of the verb registry. Null or empty array means "all verbs." |
| `path` | string \| null | Optional glob. When set, the policy applies to paths matching the glob; when null, applies regardless of path. |

The `kinds` enum is dynamic and **soft-validated**. A consumer who has not enabled the `contract` module sees a warning at `aiwf policy add` time when authoring a policy with `applicability.kinds: [contract]`, but the policy is accepted and persists. At evaluation time, references to disabled kinds are inert (no matches) and emit an `inactive-kind-reference` finding so the consumer can decide whether to re-enable the module, retire the policy, or rewrite its applicability. Disabling a module never strands authored entities.

**`enforcement` (optional).** Absent block = advisory-only; the policy is read by humans and LLMs but not mechanically evaluated. To make a policy mechanically evaluable, declare an enforcement block with a single field:

| Sub-field | Type | Rule |
|---|---|---|
| `ref` | string | Shell command. Receives the relevant entity state on stdin as JSON; emits findings on stdout as JSON Lines (one per violation, conforming to the framework's finding schema); exits 0 for pass / non-zero for "see stdout." Tools referenced in the command must be declared in `aiwf.yaml`'s `runners:` section (§9). |

v0 has only one shape of runner — a shell command — so there is no `kind:` field. If a future runner shape (in-process Go plugin, wasm module, …) earns its weight, a `kind:` discriminator can be added additively (§What v0 does not commit to). Runner placement across the three scopes (verb-time, pre-commit, CI) is **measured automatically** rather than declared per-policy; see §The engine, Performance.

**Reserved fields (forward-compatible).**

v0 has no reserved frontmatter fields beyond the optional set above. Policy-level sovereignty protection is handled by the governance gate at the verb layer (see §Provenance integration), not by a per-policy frontmatter flag. A stricter per-policy gate (`human_only: true`, rejecting even conversational/tool-mode LLM acts) is reserved namespace; if a real case earns it, the field can be added additively.

The body of the policy file is free prose: rationale, examples, history, edge cases. Loaded by `aiwf policy show <id-or-slug>`; not pre-loaded into LLM context. Bodies may use RFC 2119 keywords or EARS-style structured-English requirements when the rule benefits from that fidelity; mechanical evaluation never reads the body — that is the runner's job, via `enforcement.ref`.

### Status FSM

```
[*] → proposed → in-effect → retired → [*]
        ↓                       ↑
        └────── (rejected) ─────┘   (cancel from proposed = retired)
```

Three transitions:

| Transition | Verb | Notes |
|---|---|---|
| `(create)` → `proposed` | `aiwf policy add` | Default for hand-authored policies and triage targets. |
| `proposed` → `in-effect` | `aiwf policy triage` (accept) or `aiwf policy promote P-NNN` | Activation. Validates frontmatter completeness; if an `enforcement` block is present, validates the command resolves and the tools it uses are declared in `runners:`. |
| `proposed \| in-effect` → `retired` | `aiwf policy retire P-NNN --reason "..."` | Terminal. `--reason` required; recorded as `aiwf-reason:` trailer. |

Reverse transitions are disallowed; a retired policy stays retired (immutability of done). To resurrect a retired policy, write a new one with the same body and a new id; cite the old one in the body as historical context.

---

## The engine

The engine is what makes "policy" structurally distinct from "ADR" or "decision." A single binary, invoked via the canonical `aiwf verify` (which defaults to every enabled validator kind; in v0 only the policy kind exists), evaluates every in-effect policy with an enforcement block against project state and emits findings in the framework's existing finding format. `aiwf verify --kind policy` is the explicit-kind form; `aiwf policy verify` is the day-one alias. When contracts grow their own validator wiring, the top-level `aiwf verify` runs both kinds without a verb-rename — that is the property §Verb surface forward-pins.

### One mechanism, three scopes

The engine is one thing, run at three different scopes:

| Scope | Input state | Runner subset |
|---|---|---|
| **Verb-time** | post-mutation state of the entity the verb is acting on | runners measured under the verb-time budget |
| **Pre-commit** (opt-in hook) | the staged diff; applicability filters by changed paths | runners measured under the pre-commit budget |
| **CI** | the full tree or PR diff | every applicable runner |

Same index, same runner dispatch, same finding format across all three. The differences are which slice of project state the engine sees and which subset of runners fits the budget for that scope.

Verb-time is the strongest enforcement: the verb refuses to apply when an `error`-severity finding fires. Pre-commit catches what slipped through (hand edits, ad-hoc paths). CI is the safety net.

Runner placement across scopes is determined by measurement, not declaration; see §Performance.

### Runner contract

A runner is a shell command. The contract is:

- **Input.** The relevant entity state, JSON-encoded, on stdin. The framework derives the state from the policy's `applicability.kinds` (or, for path-only policies, from the file content at the matched paths).
- **Output.** Findings on stdout as JSON Lines, one finding per line, matching the framework's finding schema (code, severity, message, location). Counterexample-style messages encouraged but not enforced.
- **Exit code.** 0 if the runner had nothing to say (everything's fine); non-zero if findings were emitted on stdout. The exit code's only role is to distinguish "ran cleanly with no findings" from "ran and found violations" — the engine reads stdout for the actual findings.
- **Environment.** Runners run with the working directory at the repo root and `PATH` constructed from tools declared in `runners:` (§9) plus the OS baseline. The framework does not sandbox the runner beyond that; runners are trusted code authored or vetted by the consumer.

A runner can be written in any language: a Go binary, a Rust binary, a .NET binary, a Python script, a shell pipeline. The framework doesn't care about the runner's implementation; only the input-output contract matters. This is what makes the engine language-agnostic: "Go shop," "Rust shop," ".NET shop" each ships its own runners; the engine's surface is the same.

Common cases like `cue vet schema.cue -` or `gofumpt --check ./...` are written as inline shell commands; the tools they reference are declared once in `runners:`. There is no first-class `kind: cue` or `kind: gofumpt` — those would privilege specific tools without buying real ergonomics.

### Findings

The engine emits findings in the framework's existing format: a finding code (`policy-violation:<slug>`), a severity (from the policy), a message (from the runner's stdout), and a location (file path + line range when the runner reports it). Findings flow into the same surface as every other check in the framework — `aiwf check`, the verb-time refusal, the CI report.

### Performance

The pre-commit budget is the binding constraint. Three properties protect it:

- **Index-driven filtering.** The applicability filter narrows the policy set before any runner is invoked. The reduction factor depends on corpus shape (how many policies, how narrowly each is scoped) and commit shape (how many kinds and paths the diff touches); v0 makes no quantitative claim about the typical match count and instead measures it. The runner-timing data and fire counter (§Fire counter) provide the observability needed to know whether the index is doing real work.
- **Parallel dispatch.** Independent runners run in parallel up to a worker-count cap (default: number of CPUs; configurable).
- **Auto-measured runner placement.** Every runner invocation is timed; an exponentially-weighted moving average is cached in `.aiwf/runner-timing.json` keyed by policy id. A runner whose measured time fits the verb-time budget runs at every scope; one that fits only the pre-commit budget runs at pre-commit and CI; one that exceeds both runs only at CI. The classification updates as observations accumulate.

A new (un-cached) runner runs at every scope on first invocation; the first measurement seeds the cache. Demotion across scopes is silent in the steady state; `aiwf policy verify --time-report` surfaces the per-runner classification and the budgets it's measured against, so editorial decisions can react to observed behavior. There is no declared `speed` field — the runner's measured behavior is the truth.

### Fire counter (observability for editorial pressure)

A per-policy fire counter is maintained in `.aiwf/policy-fire-counts.json`: each time a runner emits one or more findings for a policy, the count for that policy id increments. The counter is the observability primitive for editorial pressure on `surface: digest` allocation — concretely, the data that lets a human notice "this `on-demand` policy has fired blocking findings forty times; the agent should know about it in advance."

v0 ships **collection only**: the counter is written by the engine; no verb reads it automatically; no finding is emitted on threshold-cross; triage does not surface hot policies. `aiwf policy hot --threshold N` is the manual audit verb — reads the counter and lists policies with `surface: on-demand` whose fire count exceeds `N`. Whether to add automated surfacing (a finding from `verify`, a category in `triage`) is deferred (§What v0 does not commit to → Automated surfacing of the fire counter); the counter exists so that decision can be made from data, not hypothesis.

---

## The verb surface

```
aiwf policy add <slug> [--bundle <name>] [--label <name>]... [--surface ...] [--severity ...] [--applies-to ...]
aiwf policy show <id-or-slug>
aiwf policy list [--bundle <name>] [--surface ...] [--status ...] [--applies-to-kind ...]
aiwf policy bundles list
aiwf policy applicable --kind <k> [--verb <v>] [--path <p>]
aiwf policy sweep --source <path|skill|bundle> [--source ...]
aiwf policy triage
aiwf policy promote <id-or-slug>
aiwf policy retire <id-or-slug> --reason "..."
aiwf policy verify [--scope verb|pre-commit|ci] [--time-report]
aiwf policy hot [--threshold N]
aiwf policy doctor
```

The top-level `aiwf verify` (no `--kind`) is the canonical evaluator entry: it runs every enabled validator kind in parallel and tags findings by kind. In v0 there is only one kind to run (policy), so `aiwf verify` and `aiwf verify --kind policy` produce the same output; `aiwf policy verify` is the day-one alias. The shape is deliberately forward-pinned: when contracts grow validator wiring, `aiwf verify --kind contract` joins the dispatch loop without a verb-rename. `aiwf doctor` (kernel-level self-check) and `aiwf policy doctor` (toolchain manifest) remain separate verbs in v0; their unification is reserved namespace until a second module-level doctor concern earns the umbrella.

Module activation/deactivation is not a verb — it's a `aiwf.yaml` edit followed by `aiwf init` (§3). No `aiwf policy enable`.

### `aiwf policy add`

Allocates the next `P-NNN`, scaffolds the file under `work/policies/<bundle>/P-NNN-<slug>.md`, populates frontmatter from flags. When `--bundle` is omitted, the bundle defaults to `local` and the file lands at `work/policies/local/P-NNN-<slug>.md`. The bundle directory is created if it doesn't yet exist. Validates the slug against the dotted-name format. Soft-validates `applicability.kinds` against currently-enabled entity kinds (warning, not rejection — see §Applicability). When `--label` is passed (repeatable), populates `labels:` with the parent directory name first followed by the additional labels. Status `proposed` unless `--accept` is given (which goes straight to `in-effect`; allowed only for human actors, recorded with `aiwf-on-behalf-of:` if the runner is an LLM under authorize-scope).

### `aiwf policy show`

Reads the policy file by id or slug, emits frontmatter + body. Used by the LLM to fetch full text after retrieval has identified a relevant policy. Optional `--format=json` for structured consumption.

### `aiwf policy list`

Filtered listing. Default emits id, slug, bundle (every label, primary first), summary, surface, status, applicability summary. Filters compose: `--bundle dotnet-stack --surface digest --status in-effect --applies-to-kind milestone`. `--bundle X` matches any policy whose primary directory or `labels:` contains X.

### `aiwf policy bundles list`

Lists every bundle directory under `work/policies/` with a count of member policies (in-effect, proposed, retired). The `local/` directory, if present, is one bundle in the listing like any other.

### `aiwf policy applicable`

The feedforward retrieval verb. Takes `--kind`, optional `--verb`, optional `--path`; returns matching `in-effect` policies in the index, sorted by severity then id. Output is a small JSON array — id, slug, summary, severity, surface — designed to be small enough that the LLM can load every match cheaply and decide which slugs warrant a full-body fetch.

```
aiwf policy applicable --kind milestone --verb promote
[
  {"id":"P-001","slug":"ac.required","severity":"error","surface":"digest",
   "summary":"Every milestone in `done` MUST have all acceptance criteria met."},
  {"id":"P-014","slug":"closing.commit","severity":"warn","surface":"digest",
   "summary":"Promotion to `done` SHOULD record a closing-commit reference."}
]
```

The LLM-side discipline that consumes this verb is encoded in the `aiwf-policy` skill (§10).

### `aiwf policy sweep`

The mining verb. Walks the sources, produces draft policies under `work/policies/draft/<bundle>/` (with `draft/local/` for drafts that don't target a named bundle). Sources:

- `--source <path>` — a file or directory in the consumer repo. Common targets: `CLAUDE.md`, `docs/`, an inherited rituals directory. Drafts default to the `local` bundle.
- `--source skill:<name>` — a named skill from the framework's skill registry, including the framework's own materialized skills. Drafts default to the `local` bundle.
- `--source bundle:<name>` — a framework-shipped or org-shared bundle (a directory of pre-structured policy files). Drafts land in `draft/<name>/`. The LLM dedups against the existing local set; pre-structured policies skip the extraction pass entirely (they're already in v0 frontmatter shape).

(Mining from arbitrary git URLs is reserved namespace; v0 keeps mining repo-local.)

For path and skill sources, the verb invokes an LLM-side extraction pass. The extraction prompt is fixed (the framework ships it); the LLM reads the source, identifies normative claims, outputs structured candidates in the v0 frontmatter shape with applicability fields populated as best-guess and no enforcement block by default. Drafts carry provenance trailers recording where each claim came from.

For bundle sources, the verb copies pre-authored policy files verbatim, then asks the LLM to flag duplicates and variants against the existing local set. Triage decides what lands.

Sweep is incremental: a re-run over the same sources de-duplicates against the existing draft + active set on summary similarity. The de-duplication is best-effort; triage handles residual duplicates.

### `aiwf policy triage`

Walks every `proposed` policy file (in `work/policies/` and `work/policies/draft/`) and presents them as a structured Q&A list — same UX shape as `git rebase -i`. Per entry: accept, reject, edit, defer, retire. Bulk operations on mined batches ("accept all from `--source skill:wf-tdd-cycle`") are first-class.

Triage runs as one commit per session, with the count of decisions in the commit message. Per-policy operations (`aiwf policy promote P-007`) are allowed but discouraged; the skill text directs the LLM toward `aiwf policy triage` as the load-bearing flow. Edit-during-triage is where authors typically add `enforcement` (point a command), since the LLM-extracted draft starts without an enforcement block.

### `aiwf policy promote` / `aiwf policy retire`

`promote` moves `proposed` → `in-effect`. `retire` moves `proposed | in-effect` → `retired`, with `--reason` required and recorded as a trailer. Index and digest update in the same commit.

### `aiwf policy verify`

The engine. Filters the index to in-effect policies with non-null `enforcement`, dispatches runners, collects findings. `--scope` defaults to `ci` (the full surface); `--scope verb` and `--scope pre-commit` apply the auto-measured budget filter. `--time-report` emits per-runner timing and the resulting scope classification.

### `aiwf policy hot`

Reads `.aiwf/policy-fire-counts.json` and lists policies with `surface: on-demand` whose accumulated fire count exceeds `--threshold` (default: 5). Output includes id, slug, fire count, and current surface — designed to be skimmed during a manual editorial review for "should this be promoted to `surface: digest`?" The verb does not mutate the counter or the policies; it is a read-only audit surface. v0 ships no automated equivalent (no finding from `verify`, no triage prompt) — see §What v0 does not commit to → Automated surfacing of the fire counter.

### `aiwf policy doctor`

Validates the toolchain manifest in `aiwf.yaml`'s `runners:` section against the actual environment. For each declared tool, checks the command resolves and (where declared) the version constraint is met. Emits findings on missing tools or version mismatches. Run once at `aiwf policy verify` startup with a short cache; runnable independently to debug setup before authoring runners.

---

## Toolchain — the `runners:` section

Every tool a runner invokes — beyond the OS baseline (`git`, `bash`, `sed`, common POSIX utilities, which are assumed) — is declared in `aiwf.yaml`'s `runners:` section. The framework does not bundle, ship, or vendor any toolchain; tools are entirely the consumer's responsibility. This is what keeps aiwf small and language-agnostic: a Go shop's `runners:` lists Go tooling, a Rust shop's lists Rust tooling, a polyglot shop's lists everything its policies use.

```yaml
# aiwf.yaml
modules:
  - policy

runners:
  cue:
    cmd: cue
    min_version: "0.7"
    version_check: "cue version"
  gofumpt:
    cmd: gofumpt
  staticcheck:
    cmd: staticcheck
    min_version: "2024.1"
  myorg-checks:
    cmd: ./bin/myorg-checks      # repo-local consumer-built binary
  cargo-deny:
    cmd: cargo-deny
    min_version: "0.14"
```

| Field | Required | Meaning |
|---|---|---|
| `cmd` | yes | Command to invoke. Resolved against `PATH` unless it's a path (starts with `./` or `/`). |
| `min_version` | no | Minimum version required. Validated by `aiwf policy doctor`. |
| `version_check` | no | Command that emits the tool's version on stdout (default: `<cmd> --version`). |

A policy's `enforcement.ref` references these tools by name as part of its shell command:

```yaml
enforcement:
  ref: "cue vet ./policies/ac.cue -"
```

The engine resolves `cue` through the `runners:` map at evaluation time. If `cue` is not declared, `aiwf policy verify` fails with a clear error (`policy P-NNN references undeclared tool 'cue'; add it to runners: in aiwf.yaml`). Per-policy `requires:` declarations are intentionally absent — the manifest is the single audit surface for "what does this repo's policy machinery depend on?"

OS-baseline tools are not declared. A runner that uses `bash`, `sed`, `awk`, or `git` does so directly; if these are missing the runner fails at the OS level with a normal command-not-found, which is rare and out of aiwf's scope.

Future option (not v0): `runners:` may grow compatibility with `mise` / `asdf` / `nix` so a consumer running `mise install` from the repo gets the right toolchain. Until that earns its weight, `aiwf policy doctor` is the validation surface.

---

## Applicability

Applicability is the metadata that makes feedforward retrieval and engine dispatch cheap. The query has three axes — kind, verb, path — and a policy is applicable when *all three of its axes match the query*.

| Policy axis | Policy value | Query value | Match rule |
|---|---|---|---|
| `kinds` | array | single kind | query value must be in the array, OR array is empty (apply-everywhere) |
| `on_verbs` | array \| null | single verb \| null | query value must be in the array, OR array is null/empty (apply-to-all-verbs) |
| `path` | glob \| null | single path \| null | query value must match the glob, OR policy path is null (apply-anywhere) |

Examples:

| Policy | Frontmatter `applicability` | Match `--kind milestone --verb promote --path work/epics/E-3/M-7.md`? |
|---|---|---|
| Every milestone-promote claim | `kinds: [milestone], on_verbs: [promote]` | yes |
| Every milestone claim regardless of verb | `kinds: [milestone], on_verbs: null` | yes |
| All entity-touching mutations under `work/` | `kinds: [], on_verbs: null, path: "work/**"` | yes |
| Apply only to ADR work | `kinds: [adr], on_verbs: null` | no (kind mismatch) |

Matching is exact on closed-set strings plus glob match on path; no regex engine, no fuzzy matching. This keeps the index query O(N) over the index entries with N small per filter.

### What applicability does not capture (yet)

- **Severity escalation across a chain of related policies.** A `warn` and an `error` that both apply to the same query are returned in severity order; the kernel makes no further decision.
- **Conflict between two policies that both apply.** Two `error` policies that contradict each other are returned together; the LLM and user are responsible for noticing. Conflict detection at acceptance time is reserved (§13).
- **Conditional applicability.** "Applies only when the milestone has more than five ACs" is not expressible. The closest workaround is splitting into two policies with different `path` globs.

---

## The digest

The digest is the file the LLM reads at session start. It holds **only policies with `surface: digest`**, regardless of whether those policies are mechanically enforced or advisory. Formatted for tokenizer-friendly compression and uniform skim.

The digest is a courtesy. It is not the enforcement surface; the engine is. Its purpose is to let the LLM avoid mistakes the engine would catch anyway, so the user does not have to discover violations through the verifier when the LLM could have known in advance.

### Path

```
.aiwf/policy-digest.md
```

The digest is regenerated on every policy-mutating verb in the same commit. Hand-edits surface as `policy-digest-drift` findings.

### Format

```
# .aiwf/policy-digest.md  (generated by aiwf; do not edit)
# Legend: ! = MUST, ~ = SHOULD, ? = MAY, ⊘ = MUST NOT
# Cite policies in your reasoning as (per <slug>). Use `aiwf policy show <slug>` for full text.

[aiwf-rituals]
P-001 | ac.required    | ! all acceptance criteria met before `done` | kinds: milestone, verbs: promote | error
P-014 | closing.commit | ~ record closing-commit on done             | kinds: milestone, verbs: promote | warn

[meta-policies]
P-027 | trailers.required | ! every entity-touching commit carries trailers | kinds: *, verbs: * | error

[dotnet-stack]
P-053 | nolint.rationale | ⊘ //nolint without one-line rationale | path: **/*.go | warn
```

Properties:

- **Pipe-delimited, fixed-shape.** Five columns per entry: id, slug, sigil-bound summary, applicability, severity. Tokenizers handle this efficiently.
- **RFC 2119 sigils.** `!` `~` `?` `⊘` for MUST, SHOULD, MAY, MUST NOT.
- **Grouped by bundle.** Group headers (`[aiwf-rituals]`, `[dotnet-stack]`) are the bundle directory names. Authors do not write group headers separately; placement in a bundle directory drives the grouping.
- **No prose body.** The summary is what's in the digest; the body is fetched on demand via `aiwf policy show`.

### Size budget

At a thousand-policy corpus with editorial discipline holding the `surface: digest` ratio at ~10–15%, the digest is ~100–150 entries. At ~25 tokens/entry plus headers, that lands at ~3–4k tokens — well under any session budget.

The protective mechanism is **editorial discipline**. The friction of authoring a policy entity at all (frontmatter, slug, summary, applicability) is the first filter; periodic review of the digest-bound set is the second. `aiwf policy list --surface digest` is the audit surface. There is no hard cap; the policy body is where authors should record why a digest-bound rule earns the slot, but the framework does not require a structured rationale field.

The kernel rule: **the digest format is a kernel commitment, not a consumer choice.** Consumers may add their own headers; they may not change the per-entry shape.

### Surface and enforcement are independent axes

| `surface` | `enforcement` | Example |
|---|---|---|
| `digest` | absent | LLM-discipline rule, no checker (`"cite policies in your reasoning as (per <slug>)"`) |
| `digest` | runner present | **The dominant case.** Agent should know in advance AND engine blocks (`"every entity-touching commit MUST carry trailers"`) |
| `on-demand` | absent | Engineering convention the agent looks up when relevant |
| `on-demand` | runner present | Style rule — agent fetches if asked; engine catches what slips through |

Judgment-shaped guidance — escalation playbook, precedence rules between conflicting principles, "when in doubt, ask the user" — lives in `CLAUDE.md` prose, not in the policy store. Judgment that *does* warrant an entity is a policy with `surface: on-demand` and no enforcement block.

---

## The skill — feedforward courtesy for the LLM

A skill — `aiwf-policy` — ships in the policy module and is materialized into the consumer's AI host (e.g., `.claude/skills/wf-policy/SKILL.md`) when the policy module is enabled. It is the load-bearing piece that makes feedforward useful.

The skill instructs the LLM:

1. **At session start, read `.aiwf/policy-digest.md`.** The digest is the small, always-loaded surface.
2. **Before any mutating action, query `aiwf policy applicable --kind <k> --verb <v>`.** Match returns are usually 0–10 policies; load them all into reasoning.
3. **For any returned policy whose summary signals load-bearing relevance to the action, call `aiwf policy show <slug>` to fetch the full body.**
4. **Cite policies in commit messages and reasoning as `(per <slug>)`.** Citations are textual; v0 does not verify them.
5. **When the user describes a rule that has no matching policy, propose `aiwf policy add` (or `aiwf policy sweep --source conversation` for batches).** Don't smuggle a new rule into prose; route it through the entity surface.
6. **When a governance verb returns `policy-governance-requires-human`, surface to the user with the exact command they should run.** This happens when the LLM is operating under autonomous authorize-scope (no per-action human consent) and tries to mutate an active policy. Working verbs — `sweep`, `verify`, `show`, listings — are unaffected; governance verbs (`add`, `triage`, `promote`, `retire`) need the human to either run the verb directly or restate the intent in conversation so the act qualifies as conversational/tool-mode. Treat the refusal as routine, not as an error to debug.

The framework does not police the skill's adherence. A non-compliant agent produces a worse experience but not an incorrect one — the engine's verb-time refusal still blocks `error`-severity violations. The skill saves the user from rework; the engine guarantees correctness.

**Non-AI consumers bypass the skill entirely.** A CI script invoking `aiwf milestone promote` does not read the digest, does not call `aiwf policy applicable`, does not cite slugs in commit messages. The engine still runs verb-time enforcement and refuses on `error` findings exactly as it does for an AI-driven invocation. The digest and the skill are courtesy infrastructure for AI consumers; they're inert for everyone else, and that's fine.

---

## Mining

Mining is the verb that lets a project bootstrap a corpus from material it already has, and stay current as in-repo sources evolve.

### Sources (v0)

| Source | Shape | Provenance recorded | Mining mode |
|---|---|---|---|
| `--source <path>` | File or directory in the consumer repo | path, file commit hash | LLM extraction |
| `--source skill:<name>` | Named skill in the skill registry | skill name, version, the framework binary's commit hash | LLM extraction |
| `--source bundle:<name>` | A bundle (framework-shipped or org-shared directory of pre-structured policy files) | bundle name, source rev | Verbatim copy + LLM dedup |

For path and skill sources, the verb invokes an LLM-side extraction pass. The extraction prompt is fixed (the framework ships it); the LLM reads the source, identifies normative claims, outputs structured candidates in the v0 frontmatter shape with applicability fields populated as best-guess and no enforcement block by default. The triage step is where enforcement intent is added.

For bundle sources, no extraction is needed — the bundle ships pre-authored policy files in v0 frontmatter shape. The LLM's role is dedup and variant-surfacing against the existing local set.

### Provenance trailers

| Trailer | Meaning |
|---|---|
| `aiwf-policy-source:` | source identifier (path or `skill:<name>`) |
| `aiwf-policy-source-rev:` | upstream commit hash for skill-sourced claims |
| `aiwf-policy-source-line:` | line range in the source file where the claim was extracted |
| `aiwf-policy-mined-at:` | RFC 3339 timestamp |

The principal × agent × scope provenance model already accommodates mining: the principal is the human who runs sweep; the agent is the framework binary plus the LLM that extracted; the scope is the bulk-mine operation.

### Triage flow

Mined drafts land at `work/policies/draft/<bundle>/P-NNN-<slug>.md` with status `proposed`. They do not appear in `aiwf policy applicable` queries. Triage produces one commit per session containing every accept/reject/edit/retire decision. Bulk operations on a `--source`-tagged batch are first-class.

### Updates

Re-running `aiwf policy sweep` over the same sources re-mines and de-duplicates against the existing active + draft set:

- **Pure addition** — new claim surfaces as a new draft.
- **No-op** — existing draft or active policy already covers the claim; silently skipped.
- **Possible revision** — existing active policy whose summary differs from the source; surfaces as a draft with `aiwf-policy-revises:` trailer pointing at the existing id. Triage decides.

The kernel rule: **mining is sourcing; ratification is local; updates require consent.**

---

## Tree discipline

| Path | Recognized as | Tree-discipline behavior |
|---|---|---|
| `work/policies/<bundle>/P-NNN-<slug>.md` | `policy` (bundle = parent directory; `local/` is a bundle like any other) | normal entity; verb-mediated |
| `work/policies/draft/<bundle>/P-NNN-<slug>.md` | `policy-draft` | mining-output; verb-mediated; promoted to `policy` via triage |
| `work/policies/P-NNN-<slug>.md` (loose, no bundle parent) | stray | `unexpected-tree-file` finding |
| empty bundle directory under `work/policies/` | bundle scaffold | allowed; no findings |
| anything else under `work/policies/` (loose non-policy files) | stray | `unexpected-tree-file` finding |
| `.aiwf/policy-index.json` | framework-owned projection | regenerated by every policy-mutating verb; hand-edits → `policy-index-drift` finding |
| `.aiwf/policy-digest.md` | framework-owned projection | same |
| `.aiwf/policy-fire-counts.json` | framework-owned counter | incremented by `aiwf verify` when a runner emits findings; read by `aiwf policy hot`; hand-edits silently overwritten on next verify run, no drift finding (the file is data, not a projection) |
| `.aiwf/runner-timing.json` | framework-owned cache | written by `aiwf verify`; same data-not-projection treatment |
| `.aiwf/` (anything else) | framework-owned scratch | engine writes; consumer reads if at all; no manual editing |

Body-prose edits in `work/policies/*.md` are allowed without the verb (consistent with the existing tree-discipline carve-out); frontmatter edits are not. The verb authoritatively owns frontmatter; hand-edits to status, surface, severity, summary, labels, applicability, or enforcement bypass the index regen and the digest update, and surface as `policy-frontmatter-drift` findings on the next `aiwf check`.

---

## Provenance integration

The existing provenance model covers all policy verbs without amendment.

### `aiwf-policy-source:` family

Reserved by mining. The source family records origin (path or skill) plus revision and line range. Provenance is single-repo in v0; no cross-project federation.

### Force semantics

`aiwf policy add --force` and `aiwf policy retire --force` are reserved namespace; v0 does not implement `--force`. The principle (force is human-only, recorded with reason) is established and will apply unchanged when `--force` lands.

### Authorize-scope coverage

A human authorizing an LLM to "draft policies for the security area" is expressible in the existing authorize-scope model: the scope entity is an area (e.g., a milestone or epic), the LLM commits with `aiwf-on-behalf-of:` and `aiwf-authorized-by:` for each draft. No new authorize-scope semantics are required.

### Governance gate

Policy verbs split into two classes by what they produce:

- **Working verbs** — `sweep` (creates inert drafts in `work/policies/draft/`), `verify`, `show`, `list`, `bundles list`, `applicable`, `hot`, `doctor`. No actor-posture gate. An LLM under autonomous authorize-scope runs these freely.
- **Governance verbs** — `add`, `triage`, `promote`, `retire`. Mutate active policies or create new ones outside the inert draft area. Require a human in the loop *per action*.

The gate is on actor posture, not on the policy:

| Posture | Trailers | Governance verbs allowed? |
|---|---|---|
| Sovereign | `aiwf-actor: human/<name>` | yes |
| Conversational / tool | `aiwf-actor: ai/<name>`, `aiwf-on-behalf-of: human/<name>` | yes — human is the principal-in-the-loop per action |
| Autonomous authorize-scope | `aiwf-actor: ai/<name>`, `aiwf-authorized-by: human/<name>`, active scope | **no** — rejected with `policy-governance-requires-human` |

Default-deny for autonomous-scope is the correct safety default for the most sensitive resource: an LLM under a wide authorize-scope cannot disable, retire, or weaken a policy that constrains the work it is doing. The split by verb class — working vs governance — lets autonomous LLMs do *work* on policies (mining, drafting, verifying) while the human retains sole authority over what becomes a binding rule.

When an autonomous LLM hits the gate, the skill (§The skill) directs it to surface the action to the user with the exact command to run. There is no `--force` override on governance verbs in v0; sovereign acts already pass the gate.

---

## Relationship to contracts

Contracts and policies remain distinct entity kinds. They share one mechanism (the engine's evaluation surface) and several conventions (finding format, trigger points, auto-measured budget classification, opt-in via `aiwf.yaml`); they differ on audience, lifecycle, and authoring.

### Distinction

| Property | Contract | Policy |
|---|---|---|
| Audience | Bilateral — producer and consumer of an interface | Unilateral — one team's commitment |
| What the validator answers | "Do the two parties agree on this shape?" | "Is this team's state in compliance with this rule?" |
| Lifecycle pressures | Version pinning, breaking-change protocol, both-parties-consent | Ratification, supersession, retirement |
| Typical evaluator | Schema validator, fixture-based check | Constraint runner, lint, tree invariant |

Cramming a contract into the policy kind would force a unilateral lifecycle onto a bilateral concept. Cramming a policy into the contract kind would force a producer-consumer audience onto a one-team rule. Both kinds earn their separation.

### Shared mechanism

The framework's evaluator surface runs validators from both kinds with one finding format, one set of trigger points (verb-time, pre-commit, CI), and one auto-measured budget convention (verb-time-eligible, pre-commit-eligible, CI-only). v0 ships the top-level `aiwf verify` as the canonical entry point, defaulting to every enabled validator kind; in v0 that is just policy, so `aiwf verify`, `aiwf verify --kind policy`, and `aiwf policy verify` (alias) produce the same output. When contracts grow their own validator wiring, `aiwf verify --kind contract` joins the dispatch loop and an unscoped `aiwf verify` runs both kinds without a verb-rename. The verb shape is forward-pinned in v0; only the contents of "enabled validator kinds" grows over time.

### Independent opt-in

Either module can be enabled without the other:

- **Contract only**: bilateral interface validators, no unilateral rules entity.
- **Policy only**: unilateral rules with engine evaluation, no bilateral interface machinery.
- **Both**: each kind manages its own lifecycle; the engine evaluates both.
- **Neither**: baseline modules only; no validators of either kind.

### Interaction patterns

The two kinds are mostly independent by design. Two real interactions:

- **A policy can have a contract as its subject.** Example: a policy `every module in framework/modules/ MUST have a contract.cue` whose runner checks contract-file presence. The contract entity is data the policy reads about; the policy doesn't change the contract.
- **A contract change can violate a policy.** Example: a consumer adds a contract whose path doesn't match a `contracts.location` policy. At verb-time when the contract is added, the policy fires through the engine.

In both directions, the interaction is mediated by the shared engine reading shared state. No special wiring, no entity-to-entity references in v0.

---

## What goes in the digest

The digest is bounded by editorial choice, not corpus size. Each policy carries `surface: digest` or `surface: on-demand`; only the former appears in `.aiwf/policy-digest.md`. The author declares it on creation; triage can override; `aiwf policy list --surface digest` is the discoverability surface.

The decision rule: **does the agent need to know this rule *before* composing an action that might touch it?**

| `surface` | When to choose it |
|---|---|
| `digest` | The agent's correctness depends on knowing in advance. Choosing wrong is expensive — wrong-shape code, wrong field naming, wrong migration posture, wrong commit shape. The cost of "do it wrong, fix on finding" far exceeds the cost of "know in advance." |
| `on-demand` | The agent does not need the rule pre-loaded. Either the engine catches violations cheaply, the rule applies only in narrow situations the agent can detect and look up via `aiwf policy applicable`, or the rule is judgment-shaped. |

### Bounded digest at thousand-policy scale

With editorial discipline, the `surface: digest` count stays roughly flat as the corpus grows. A repo with 1500 policies can have the same ~150-entry digest as a repo with 800 policies; what scales is the on-demand corpus, retrieved through `aiwf policy applicable`.

A future engine-side hint can flag editorial mismatch: an `on-demand` policy whose enforcement has fired blocking findings repeatedly is a candidate for promotion to `surface: digest`. v0 does not produce this hint; editorial discipline is human-driven, with `aiwf policy list --surface digest` as the visible audit surface.

---

## What v0 does not commit to (reserved namespace)

Several items are deliberately deferred. Each names what would have to land for it to earn its place, and what's reserved in the v0 surface so the addition is non-breaking.

### Waivers (`aiwf policy waive`)

**Why deferred.** The base case (write, ratify, evaluate, retire) needs to be in use first. Waivers are state machinery on top of evaluation; landing them too early over-fits the design.

**What's reserved.** The `waived` status value; the `aiwf-policy-waived:` and `aiwf-policy-waiver-until:` trailer keys; the `aiwf policy waive` verb name.

**Earn it with.** A real exception case the consumer wants tracked through a verb rather than a status flip, plus a time-bound expiration story.

### Supersession chain

**Why deferred.** v0's `retired` terminal status absorbs the supersession case crudely: replace policy P-7 by writing a new one and retiring the old. The supersedes-chain adds a navigable lineage but does not change the steady state.

**What's reserved.** The `superseded` status value; the `aiwf policy supersede` verb name. v0's frontmatter has no slot for a supersession chain; if the chain lands later, a frontmatter field can be added additively.

**Earn it with.** A consumer with a corpus large enough that lineage navigation is a real usability concern.

### Conflict detection at triage

**Why deferred.** Glob-overlap detection is non-trivial in the general case; conflicts surface naturally during human triage of related digest-bound policies. v0's compromise: conflicts surface as findings at engine-evaluate time when both policies fire on the same input.

**What's reserved.** No frontmatter or trailer changes required; the verb-time check can be added later without breaking anything.

**Earn it with.** A reproducible case where two policies conflict in production and the evaluate-time finding is too late.

### Citation checking `(per <slug>)`

**Why deferred.** Citation-by-slug in commit messages is a behavior the LLM adopts under skill direction; verifying that citations resolve is a cleanup pass that earns its place once the citing behavior is observable.

**What's reserved.** The `(per <slug>)` notation in skill prose; the verb `aiwf policy verify-citations` is reserved.

**Earn it with.** The skill being followed in practice; a consumer asking the engine to police hallucinated citations.

### Additional runner shapes (`wasm`, in-process plugins)

**Why deferred.** A shell command is language-agnostic and covers every shape of runner a consumer might build. New runner shapes have to clear a real bar — something a shell command cannot express, worth the engine surface area.

**What's reserved.** The `enforcement` block has no `kind:` discriminator in v0 (only `ref:`); when a second runner shape earns its weight, `kind:` is added additively, defaulting to `command` for existing policies.

**Earn it with.** A concrete case where a shell-command runner produces an unacceptable evaluator (e.g., per-policy invocation overhead dominates real-world runs).

### Automated surfacing of the fire counter

**Why deferred.** v0 ships the counter (`.aiwf/policy-fire-counts.json`) and a manual audit verb (`aiwf policy hot`) so the data is collected from day one. Whether to also surface that data automatically — a finding from `aiwf verify` when an `on-demand` policy crosses a threshold, a category in `aiwf policy triage`, a separate periodic prompt — depends on observed editorial behavior. Shipping automation before observation risks either nagging or missing the right signal.

**What's reserved.** No frontmatter or trailer changes; the surfacing is purely an engine-side feature, additive at any time.

**Earn it with.** Counter data from real corpora showing repeated `on-demand` policy fires that the consumer would have wanted flagged proactively.

### Conditional applicability

**Why deferred.** "Applies only when the entity has more than five children" / "applies only when status is `done`" is not expressible in v0's three applicability axes (kinds, on_verbs, path). The escape valves are `path` glob splitting and runner-side filtering: the runner sees full entity state and can return zero findings when its conditions don't match. v0's claim is that these escape valves are sufficient until friction shows otherwise.

**What's reserved.** No frontmatter slot; if a richer applicability surface lands, an `applicability.when:` field can be added additively. Prior-art pointer: BDD's Given/When/Then has decades of solved problems for conditional preconditions, including the gotchas (over-loose `When` clauses, scenario explosion); start there rather than designing from scratch.

**Earn it with.** A friction-journal entry where the path-glob and runner-side workarounds produce duplication or unreadable rules.

### Doctor unification

**Why deferred.** v0 keeps `aiwf doctor` (kernel self-check) and `aiwf policy doctor` (toolchain manifest validation) as separate verbs. Designing the umbrella before there's a second module-level doctor concern (e.g., when contracts grow their own setup-validation needs) means designing from one example.

**What's reserved.** The verb shape `aiwf doctor --kind <module>` is the obvious extension; verb-name compatibility is preserved by the existing `aiwf doctor` continuing to mean "kernel self-check."

**Earn it with.** A second module-level doctor concern that makes the unified surface concrete.

### Toolchain manager integration (`mise` / `asdf` / `nix`)

**Why deferred.** v0 keeps `runners:` self-contained: the consumer declares tools and version constraints; `aiwf policy doctor` validates the local environment matches; the consumer installs the tools via whatever mechanism they prefer. Each toolchain manager has its own version-pinning convention, activation model, and error semantics; aiwf would be reinventing what these tools already do well.

**What's reserved.** A future `manager:` field on `runners:` entries can be added additively if integration earns its weight.

**Earn it with.** A consumer with enough runners across enough ecosystems that setup churn is a measurable, reproducible problem — and a clear case that aiwf shelling out to `mise install` (or equivalent) is meaningfully better than a clear `aiwf policy doctor` error message pointing the consumer to their own install command.

### Stack-discovery consumption

**Why deferred.** v0 ships stack *detection*: `aiwf init` writes `stack: [...]` to `aiwf.yaml` (§Stack discovery). What v0 does not ship is any verb that *consumes* the field — no mining hint that suggests stack-relevant community bundles, no applicability axis that filters policies by detected stack, no LLM-context injection. Each candidate consumer needs its own design pass; shipping all three speculatively is the wrong shape.

**What's reserved.** Three concrete consumer shapes are anticipated and additive when they earn their weight: (1) `aiwf policy sweep --by-stack` selecting community-bundle subsets that match the consumer's stacks; (2) an `applicability.stacks: [...]` axis on policies; (3) the `aiwf-policy` skill loading the `stack:` field as session-start context.

**Earn it with.** A concrete first consumer — typically a community bundle large enough that stack-filtered selection is real ergonomics, or a policy whose applicability genuinely depends on the toolchain.

### Stricter per-policy human-only gate (`human_only: true`)

**Why deferred.** v0's verb-layer governance gate (§Provenance integration) already blocks autonomous-scope LLMs from mutating active policies. Conversational/tool-mode LLM acts pass — the human is the principal-in-the-loop per action. The stricter gate ("even conversational LLM cannot mutate this policy; only sovereign human action qualifies") is a meaningful additional restriction, but v0 has not yet identified a concrete policy that needs it.

**What's reserved.** The `human_only: true` frontmatter field; the rejection finding code (`policy-requires-sovereign-actor` or similar). No frontmatter slot in v0; the field can be added additively when it earns its weight.

**Earn it with.** A specific v0 policy where conversational/tool-mode LLM mutation is genuinely too loose — typically a sovereignty rule whose retirement should require deliberate human action, not a chat turn.

### Slug rename (`aiwf policy rename-slug`)

**Why deferred.** Slugs are immutable in v0; renaming is retire-and-create (write a new policy with the same body and a new slug; retire the old). Renames are rare in practice, and the retire-and-create path keeps the kernel small.

**What's reserved.** The verb name `aiwf policy rename-slug`. No alias-table machinery in v0; if a rename verb lands later, it can ship without aliases (citations in old commits become stale-citation noise) or with an alias mechanism if observed friction warrants it.

**Earn it with.** Repeated retire-and-create churn from a real consumer; clear pattern of slug-naming convention drift.

### `info` severity

**Why deferred.** v0 ships `error | warn` only. The corpus practice the FlowTime survey observed was binary (blocking vs soft-signal); a third value didn't add load-bearing distinct meaning.

**What's reserved.** The `info` enum value; it can be added additively without breaking existing data.

**Earn it with.** A consumer with a real soft-signal sub-class that `warn` doesn't already cover.

---

## Pointers

- `design-decisions.md` — the kernel commitments. The seven-kind set is itself a kernel commitment; the policy module's opt-in nature means the seventh kind is reserved-but-inactive in any consumer that has not added `policy` to `aiwf.yaml`'s `modules:` list.
- `design-lessons.md` — the three principles. Identity-is-not-location applies (policy ids are stable, slugs are stable); atomicity-is-a-unit applies (every policy verb is one commit); don't-fight-the-substrate applies (the engine-internal `internal/invariants/` package is named for what it does — audit kernel invariants — distinct from the user-facing policy module).
- `provenance-model.md` — the trailer system and authorize-scope semantics. Mining and the engine both reuse the existing model without amendment.
- `tree-discipline.md` — the recognized-paths model. `work/policies/`, `work/policies/draft/`, and `.aiwf/` extend the recognized-paths table.
- `id-allocation.md` — the trunk-aware allocator. Policy ids are allocated by the same mechanism; the allocator's regex set extends with `P-NNN`.

---

## What lands in the build plan

Sequenced for the build that implements this proposal:

1. **`aiwf.yaml` `modules:` machinery.** Reading the list at startup, gating verb visibility, gating hooks. Generalizes to contract too. Default: empty list (baseline modules only). `aiwf init` also runs the stack detector and writes the auto-populated `stack: [...]` field (manifest-file presence: `*.csproj`, `Cargo.toml`, `go.mod`, `package.json`/`tsconfig.json`, `pyproject.toml`, `Gemfile`, …); no v0 verb consumes the field — it's an observability primitive for future use (§Stack discovery).
2. **`.aiwf/` directory.** Framework-owned scratch and projection space; the layout, the .gitignore, the drift-finding integration. Establishes the home for any future generated artifact.
3. **Schema and entity registration.** Adds `policy` to the entity registry, the path table, the schema/template surfaces. Slug format validator. Soft-validated dynamic `applicability.kinds` enum (warning, not rejection) plus `inactive-kind-reference` finding at evaluation time. Bundle directory recognition. `labels:` field with first-entry-equals-parent-directory validator.
4. **The verb surface (authoring + retrieval).** `add` (with `--bundle`, repeatable `--label`), `show`, `list`, `bundles list`, `applicable`, `triage`, `promote`, `retire`, `hot`. The index regenerator (recording every bundle from path AND `labels:`), the digest renderer (grouping by bundle, primary first). The governance gate (default-deny for autonomous-scope LLMs on `add`/`triage`/`promote`/`retire`, sovereign and conversational/tool-mode pass) is wired here, reusing the existing actor-classification helpers.
5. **The toolchain section.** `aiwf.yaml` `runners:` parser, `aiwf policy doctor` validator.
6. **The engine.** Top-level `aiwf verify` (defaults to all enabled validator kinds); v0 ships only the policy kind, so `aiwf verify`, `aiwf verify --kind policy`, and `aiwf policy verify` (alias) produce the same output. The `command` runner. Verb-time hook integration in baseline-module verbs. Pre-commit hook script. CI wiring. Auto-measured runner placement stored in `.aiwf/runner-timing.json`; per-policy fire counter stored in `.aiwf/policy-fire-counts.json` (incremented when a runner emits findings); `--time-report` surfaces the per-runner classification. Same engine, three scopes.
7. **The skill.** `aiwf-policy` SKILL.md; the retrieval discipline, the citation convention, the triage prompt, and the governance-refusal handling (when the LLM hits `policy-governance-requires-human` from autonomous scope, surface the verb to the user with the exact command to run).
8. **Mining.** `aiwf policy sweep` with the path, skill, and bundle source kinds; the LLM-side extraction prompt for path/skill; the verbatim-copy + dedup path for bundle.
9. **Bundles shipped.** Framework-shipped bundles materialize as directories of pre-structured policy files alongside the policy module — kernel and workflow layers only. Stack-flavored bundles are explicitly out of scope: consumers author their own under `work/policies/<name>/` (or pull from community sources via `aiwf policy sweep`).
10. **Tree-discipline integration.** Recognize `work/policies/<bundle>/` and `work/policies/draft/<bundle>/` (including `local/` as a normal bundle directory), reject loose policy files at `work/policies/` root, and recognize the framework-owned `.aiwf/` artifacts. Wire the drift findings.
11. **Documentation.** Update `overview.md` (seven kinds, opt-in modules, `.aiwf/`, bundles), `architecture.md` (path table, verb table, skill list, engine), `skill-author-guide.md`, root `README.md`.

A separate build plan under `plans/policy-model-plan.md` will detail the build sequence with iteration boundaries and acceptance criteria.
