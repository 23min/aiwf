# aiwf — a small experimental framework for AI-assisted project tracking

> Pre-alpha PoC. This branch (`poc/aiwf-v3`) carries the implementation; `main` carries the design research that motivates it.

`aiwf` is a single Go binary that helps humans and AI assistants keep track of what's planned, decided, and done in a software project, by validating a small set of mechanical guarantees about a markdown-and-frontmatter project tree.

The framework is deliberately minimal. It does *not* maintain a separate event log or graph projection; it does *not* try to be a project management tool; it does *not* require a server, an API key, or a specific IDE. Markdown files in the consumer repo are the source of truth; `git log` is the audit trail; `aiwf check` is the validator.

For the design thinking that produced this shape, see the research arc on `main` (start with `docs/research/KERNEL.md` and `docs/research/06-poc-build-plan.md`).

---

## Install

```bash
go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@<branch-tip-or-tag>
```

The PoC binary is not yet shippable. Distribution via brew/apt/scoop/winget will come if and when the PoC graduates. For now, install from a branch tip or a tagged commit on this branch.

---

## Quick start

In a consumer repository:

```bash
aiwf init                                       # scaffold aiwf.yaml, work/, docs/adr/, install pre-push hook, materialize skills
aiwf add epic --title "Discovery and ramp-up"   # E-01
aiwf promote E-01 active
aiwf add milestone --epic E-01 --title "Map the existing system"   # M-001
aiwf add adr --title "Use the existing CI pipeline"                 # ADR-0001
```

Skills materialize to `.claude/skills/wf-*` (gitignored). Open Claude Code; the assistant discovers the skills automatically.

---

## Verbs (PoC)

| Verb | Purpose |
|---|---|
| `aiwf init` | Scaffold `aiwf.yaml`, planning directories, install pre-push hook, materialize skills |
| `aiwf update` | Re-materialize skills after a binary upgrade |
| `aiwf add <kind>` | Allocate id and create the entity (kinds: `epic`, `milestone`, `adr`, `gap`, `decision`, `contract`) |
| `aiwf promote <id> <status>` | Transition status; rejected if the transition is illegal for the kind |
| `aiwf cancel <id>` | Set status to the kind's terminal-cancel value |
| `aiwf rename <id> <new-slug>` | `git mv` plus title update; the id is preserved |
| `aiwf reallocate <id>` | Resolve an id collision; pick next free id and update references |
| `aiwf check` | Validate the tree, report findings; runs as the pre-push hook |
| `aiwf history <id>` | Render `git log` filtered for an entity, formatted |
| `aiwf render roadmap` | Print a markdown table of all epics + milestones |
| `aiwf doctor` | Self-diagnostics: binary version, skill drift, collision health |

---

## Conventions

The framework imposes a small set of conventions in the consumer repo:

```
<consumer-repo>/
├── aiwf.yaml                              # tiny config (~10 lines)
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
├── .claude/skills/wf-*/                   # gitignored; materialized by aiwf init
└── ROADMAP.md                             # rendered on demand by aiwf render roadmap
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

## Status

Pre-alpha. The PoC is being built across four focused sessions described in [`docs/poc-plan.md`](docs/poc-plan.md). Not recommended for production use.

The PoC branch is not planned to merge back to `main`. Future versions of the framework are free to take a different shape; the on-disk format is simple enough that a v2 reader could import a v1 tree mechanically.

---

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
