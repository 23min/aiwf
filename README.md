# aiwf — a small experimental framework for AI-assisted project tracking

> Pre-alpha PoC. This branch (`poc/aiwf-v3`) carries the implementation; `main` carries the design research that motivates it.

`aiwf` is a single Go binary that helps humans and AI assistants keep track of what's planned, decided, and done in a software project, by validating a small set of mechanical guarantees about a markdown-and-frontmatter project tree.

The framework is deliberately minimal. It does *not* maintain a separate event log or graph projection; it does *not* try to be a project management tool; it does *not* require a server, an API key, or a specific IDE. Markdown files in the consumer repo are the source of truth; `git log` is the audit trail; `aiwf check` is the validator.

For the design closure that produced this shape, see [`docs/poc-design-decisions.md`](docs/poc-design-decisions.md). For the working session plan, see [`docs/poc-plan.md`](docs/poc-plan.md).

---

## Status

| Session | Status | What shipped |
|---|---|---|
| 1 — Foundations and `aiwf check` | ✅ done | Frontmatter parser, tree loader, nine validators, JSON + text output, exit codes, fixture-driven integration test. |
| 2 — Mutating verbs and trailers | ✅ done | `aiwf add` (all six kinds), `promote`, `cancel`, `rename`, `reallocate`. Validate-then-write contract; structured commit trailers; `aiwf-prior-entity:` on reallocate. |
| 3 — Skills, history, hooks | ⏳ pending | `aiwf init`, `update`, `history`, `doctor`, materialized Claude Code skills, pre-push hook. |
| 4 — Polish for real use | ⏳ pending | `aiwf render roadmap`, error-message polish, walk-through doc. |

What this means in practice: the kernel works (validation + mutating verbs), but the bootstrap (`aiwf init`), the AI-host integration (skills + hook), and the history view aren't shipped yet. You can already use the binary against a hand-scaffolded `work/` tree, you just have to do the scaffolding manually.

---

## Install

```bash
git clone https://github.com/23min/ai-workflow-v2 && cd ai-workflow-v2
git checkout poc/aiwf-v3
go install ./tools/cmd/aiwf
```

`aiwf` lands in `$GOBIN` (typically `~/go/bin`). Add to PATH if not already.

Distribution via brew/apt/scoop/winget will come if and when the PoC graduates.

---

## Quick start (today)

In a consumer repository (or a fresh `mkdir + git init`):

```bash
mkdir -p work/epics work/gaps work/decisions work/contracts docs/adr
git init -q && git config user.email you@example.com

aiwf add epic --title "Discovery and ramp-up"        # → E-01
aiwf add milestone --epic E-01 --title "Map the system"   # → M-001
aiwf promote M-001 in_progress
aiwf rename M-001 system-survey
aiwf add adr --title "Adopt OpenAPI 3.1"             # → ADR-0001
aiwf check                                           # validates the tree
```

Each verb produces a single git commit with structured trailers. Inspect with:

```bash
git log --pretty="%h %s%n  %(trailers:unfold=true)"
```

Once Session 3 lands, `aiwf init` will replace the `mkdir` line and add the pre-push hook automatically.

---

## Verbs

### Shipped

| Verb | Purpose |
|---|---|
| `aiwf add <kind>` | Allocate id and create the entity. Kinds: `epic`, `milestone`, `adr`, `gap`, `decision`, `contract`. |
| `aiwf promote <id> <status>` | Transition status; rejected if the transition is illegal for the kind's FSM. |
| `aiwf cancel <id>` | Set status to the kind's terminal-cancel value (`cancelled`/`wontfix`/`rejected`/`retired`). |
| `aiwf rename <id> <new-slug>` | `git mv` to a new slug; the id is preserved. |
| `aiwf reallocate <id\|path>` | Renumber an entity (recovery from a merge collision); rewrites references in other entities. |
| `aiwf check` | Validate the tree and report findings. |

### Coming in Sessions 3–4

| Verb | Purpose |
|---|---|
| `aiwf init` | Scaffold `aiwf.yaml`, planning dirs, install pre-push hook, materialize skills. |
| `aiwf update` | Re-materialize skills after a binary upgrade. |
| `aiwf history <id>` | Render `git log` filtered for an entity, dual-id matching. |
| `aiwf doctor` | Self-diagnostics: binary version, skill drift, id-collision health. |
| `aiwf render roadmap` | Print a markdown table of all epics + milestones. |

---

## Common flags

| Flag | Verbs | Default |
|---|---|---|
| `--root <path>` | every verb | walk up from cwd looking for `aiwf.yaml`, else cwd |
| `--actor <role>/<id>` | mutating verbs | derived from `git config user.email` localpart (e.g., `human/peter`) |
| `--format <fmt>` | `check` | `text` (alternatives: `json`) |
| `--pretty` | `check`, with `--format=json` | indented JSON |

Verb-specific flags for `add`:

| Flag | Kind |
|---|---|
| `--title "..."` | required for every kind |
| `--epic <id>` | milestone (required) |
| `--discovered-in <id>` | gap (optional) |
| `--relates-to <id,id,...>` | decision (optional) |
| `--format <fmt>` | contract (required) |
| `--artifact-source <path>` | contract (required; copies into `schema/`) |

---

## Conventions

The framework imposes a small set of conventions in the consumer repo:

```
<consumer-repo>/
├── aiwf.yaml                              # tiny config (~10 lines, in Session 3)
├── work/
│   ├── epics/
│   │   └── E-NN-<slug>/
│   │       ├── epic.md
│   │       └── M-NNN-<slug>.md            # milestones live inside their epic
│   ├── gaps/
│   │   └── G-NNN-<slug>.md
│   ├── decisions/
│   │   └── D-NNN-<slug>.md
│   └── contracts/
│       └── C-NNN-<slug>/
│           ├── contract.md
│           └── schema/                    # OpenAPI, JSON Schema, .proto, etc.
├── docs/
│   └── adr/
│       └── ADR-NNNN-<slug>.md
└── .claude/skills/wf-*/                   # gitignored; materialized by aiwf init (Session 3)
```

Six entity kinds, each with a closed status set:

| Kind | Statuses | ID format |
|---|---|---|
| Epic | `proposed`, `active`, `done`, `cancelled` | `E-NN` |
| Milestone | `draft`, `in_progress`, `done`, `cancelled` | `M-NNN` |
| ADR | `proposed`, `accepted`, `superseded`, `rejected` | `ADR-NNNN` |
| Gap | `open`, `addressed`, `wontfix` | `G-NNN` |
| Decision | `proposed`, `accepted`, `superseded`, `rejected` | `D-NNN` |
| Contract | `draft`, `published`, `deprecated`, `retired` | `C-NNN` |

---

## Validators (`aiwf check`)

Nine validators run on every `aiwf check` invocation, and (after Session 3) on every `git push` via the pre-push hook:

| Code | Severity | What it checks |
|---|---|---|
| `ids-unique` | error | No two entities share an id. |
| `frontmatter-shape` | error | Required fields present; id format matches kind; per-kind required fields (e.g., milestone `parent`). |
| `status-valid` | error | Status is in the kind's allowed set. |
| `refs-resolve` | error | Every reference field resolves to an existing entity *of the right kind*. |
| `no-cycles` | error | No cycles in `depends_on` (milestones) or the `supersedes`/`superseded_by` chain (ADRs). |
| `contract-artifact-exists` | error | Artifact path is relative, no `..` segments, file exists inside the contract dir. |
| `titles-nonempty` | warning | Every entity has a non-empty title. |
| `adr-supersession-mutual` | warning | If A.superseded_by = B, then B.supersedes ⊇ {A}. |
| `gap-resolved-has-resolver` | warning | A gap with status `addressed` has a non-empty `addressed_by`. |

Exit codes: `0` no errors (warnings allowed), `1` errors found, `2` usage error, `3` internal error.

---

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
