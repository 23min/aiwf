# aiwf — a small experimental framework for AI-assisted project tracking

> Pre-alpha PoC. This branch (`poc/aiwf-v3`) carries the implementation; `main` carries the design research that motivates it.

## The problem

When a human and an AI assistant work on a software project together over many sessions, the planning state ends up scattered: a goal in one chat, a half-decision in another, a task list in a Notion page, an ADR in a wiki. The next session — or the next person, or the next AI — cannot reliably see what was planned, what was decided, what was started, what was paused, what was abandoned, or *why*. Renames and re-numbers break references. "Done" is whatever the last conversation claimed.

Most existing tools optimise for one of those concerns and ignore the others. Issue trackers manage tasks but not decisions; ADR repos record decisions but not progress; AI assistants summarise but don't enforce. None of them treat the project's planning state as something the working repo itself can carry, version, and validate.

## How aiwf addresses it

`aiwf` keeps the planning state inside the consumer repo as plain markdown files with YAML frontmatter — six entity kinds (epic, milestone, ADR, gap, decision, contract), each with a closed status set and a stable id that survives rename, re-number, and merge collisions. The framework is a single Go binary that:

- **Allocates ids and creates entities** with the right shape (`aiwf add`).
- **Enforces legal status transitions** per kind (`aiwf promote`, `aiwf cancel`).
- **Preserves identity across renames and re-numbers** (`aiwf rename`, `aiwf reallocate`), rewriting references in other entities and producing exactly one git commit per mutation.
- **Validates the whole tree** (`aiwf check`) — uniqueness, references, status validity, cycles, frontmatter shape — and reports findings as `path:line: severity code: message — hint: <action>`.
- **Hooks itself into `git push`** (`aiwf init`) so an inconsistent tree never reaches the remote.
- **Reads the lifecycle from `git log`** (`aiwf history <id>`) via structured commit trailers; no separate event log.

Markdown files are the source of truth; `git log` is the audit trail; `aiwf check` is the validator. No server, no API key, no separate database. The framework is deliberately minimal: it does not try to be a project-management tool, and the AI host (Claude Code at the moment) sees the planning state through materialized skills, not a custom protocol.

For the lifecycle diagrams and the per-kind state machines, see [`docs/overview.md`](docs/overview.md). For worked walk-throughs of typical sessions and example AI prompts, see [`docs/workflows.md`](docs/workflows.md). For the design closure that produced this shape, see [`docs/poc-design-decisions.md`](docs/poc-design-decisions.md). For the working session plan, see [`docs/poc-plan.md`](docs/poc-plan.md).

---

## Status

| Session | Status | What shipped |
|---|---|---|
| 1 — Foundations and `aiwf check` | ✅ done | Frontmatter parser, tree loader, nine validators, JSON + text output, exit codes, fixture-driven integration test. |
| 2 — Mutating verbs and trailers | ✅ done | `aiwf add` (all six kinds), `promote`, `cancel`, `rename`, `reallocate`. Validate-then-write contract; structured commit trailers; `aiwf-prior-entity:` on reallocate. |
| 3 — Skills, history, hooks | ✅ done | `aiwf init`, `update`, `history`, `doctor`, materialized Claude Code skills, pre-push hook. |
| 4 — Polish for real use | ✅ done | `aiwf render roadmap`, `aiwf doctor --self-check`, polished error output (`file:line` + hints), workflows walk-through. |

The framework is usable end-to-end today: clone, `go install`, `aiwf init` in a target repo, then drive entities with the verbs below. The pre-push hook wired in by `aiwf init` is the chokepoint that makes the framework's guarantees real.

---

## Install

The fastest path is to let the Go toolchain fetch and build directly from the repo — no clone, no rebuild of any container, just one command:

```bash
go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@poc/aiwf-v3
```

Re-run the same command anytime to refresh to the latest commit on the branch. Pin a commit SHA instead of `poc/aiwf-v3` for a reproducible install (e.g. in CI):

```bash
go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@<sha>
```

The binary lands in `$GOBIN` (defaults to `$GOPATH/bin`, typically `~/go/bin`). Make sure that directory is on `$PATH`.

After re-installing, run `aiwf init` once to refresh the pre-push hook, which bakes in the binary's absolute path at install time.

### Prerequisites

- **Go 1.22+** in the environment running the install. Verify with `go version`.
- **`$HOME/go/bin` (or `$GOBIN`) on `$PATH`.** Verify with `command -v aiwf` after install.
- **Network access to GitHub.** If you're offline or behind a proxy without GitHub access, fall back to the clone path below.

### Alternate: clone-and-install

If you want a local checkout to read or modify the source:

```bash
git clone https://github.com/23min/ai-workflow-v2 && cd ai-workflow-v2
git checkout poc/aiwf-v3
go install ./tools/cmd/aiwf
```

Distribution via brew/apt/scoop/winget will come if and when the PoC graduates.

---

## Quick start

In a consumer repository (or a fresh `mkdir + git init`):

```bash
git init -q && git config user.email you@example.com

aiwf init                                                 # writes aiwf.yaml, scaffolds dirs, installs pre-push hook, materializes skills
aiwf add epic --title "Discovery and ramp-up"             # → E-01
aiwf add milestone --epic E-01 --title "Map the system"   # → M-001
aiwf promote M-001 in_progress
aiwf rename M-001 system-survey
aiwf add adr --title "Adopt OpenAPI 3.1"                  # → ADR-0001
aiwf check                                                # validates the tree
aiwf history E-01                                         # show this entity's lifecycle from git log
aiwf render roadmap                                       # markdown table of epics + milestones
```

Each mutating verb produces a single git commit with structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`). The pre-push hook installed by `aiwf init` runs `aiwf check` on every push so an inconsistent tree never reaches the remote.

To verify your install works end-to-end, run `aiwf doctor --self-check` — it spins up a throwaway repo, drives every verb, and reports pass/fail per step.

### Sample of `aiwf check` output

When validation finds something, output is one finding per line in linter form: `path:line: severity code: message — hint: <action>`.

```text
work/epics/E-01-foo/M-001-bad.md:5: error refs-resolve/unresolved: milestone field "parent" references unknown id "E-99" — hint: check the spelling, or remove the reference if the target was deleted
work/epics/E-01-foo/epic.md:4: error status-valid: status "bogus" is not allowed for kind epic (allowed: proposed, active, done, cancelled) — hint: use one of the allowed statuses listed above
work/epics/E-01-foo/epic.md:3: warning titles-nonempty: title is empty or whitespace-only — hint: set a non-empty `title:` in the frontmatter

3 findings (2 errors, 1 warnings)
```

Pipe through `--format=json` (with optional `--pretty`) when feeding CI. Exit codes: `0` clean, `1` errors found, `2` usage error, `3` internal error.

---

## Verbs

| Verb | Purpose |
|---|---|
| `aiwf init` | Write `aiwf.yaml`, scaffold planning dirs, install pre-push hook, materialize Claude Code skills. Idempotent. |
| `aiwf update` | Re-materialize the embedded skills (after a binary upgrade). |
| `aiwf add <kind>` | Allocate id and create the entity. Kinds: `epic`, `milestone`, `adr`, `gap`, `decision`, `contract`. |
| `aiwf promote <id> <status>` | Transition status; rejected if the transition is illegal for the kind's FSM. |
| `aiwf cancel <id>` | Set status to the kind's terminal-cancel value (`cancelled`/`wontfix`/`rejected`/`retired`). |
| `aiwf rename <id> <new-slug>` | `git mv` to a new slug; the id is preserved. |
| `aiwf reallocate <id\|path>` | Renumber an entity (recovery from a merge collision); rewrites references in other entities. |
| `aiwf check` | Validate the tree and report findings. |
| `aiwf history <id>` | Render `git log` filtered for the entity's structured trailers; dual-matches reallocate's old/new id. |
| `aiwf doctor` | Self-diagnostics: binary version, skill drift, id-collision health. `--self-check` drives every verb against a throwaway repo. |
| `aiwf render roadmap` | Markdown table of epics + milestones. `--write` updates `ROADMAP.md` and commits. |

---

## Common flags

| Flag | Verbs | Default |
|---|---|---|
| `--root <path>` | every verb | walk up from cwd looking for `aiwf.yaml`, else cwd |
| `--actor <role>/<id>` | mutating verbs | derived from `git config user.email` localpart (e.g., `human/peter`) |
| `--format <fmt>` | `check`, `history` | `text` (alternative: `json`) |
| `--pretty` | with `--format=json` | indented JSON |
| `--write` | `render roadmap` | print to stdout (no commit); with `--write` updates `ROADMAP.md` and commits |
| `--self-check` | `doctor` | run normal diagnostics; with `--self-check` drives every verb against a temp repo |

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

## Repo layout

`aiwf init` creates this layout in the consumer repo:

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
└── .claude/skills/aiwf-*/                 # gitignored; materialized by aiwf init
```

For the full kind/status/transition reference and the per-kind state-machine diagrams, see [`docs/overview.md`](docs/overview.md).

---

## Coexistence with your `.claude/`

`aiwf` is designed to live alongside your own Claude Code setup — your own skills, agents, slash commands, output styles, and any other tooling you've configured. It uses a strict `aiwf-*` namespace and never touches anything outside it.

**What aiwf writes:**

- `.claude/skills/aiwf-*/SKILL.md` — six skill files materialized from the binary. Wiped and rewritten by `aiwf init` / `aiwf update`.
- `.gitignore` — appends *only* the six `aiwf-*` skill paths. Your other `.claude/` content is yours to commit or gitignore as you choose; `aiwf` does not gitignore the directory wholesale.
- `aiwf.yaml`, `CLAUDE.md`, `.git/hooks/pre-push` — written only if absent. The pre-push hook carries an `# aiwf:pre-push` marker; if a hook without the marker already exists, `aiwf init` skips the hook step (leaving the existing one untouched), prints the per-step ledger so you can see exactly what landed, and finishes with a remediation block — either add `aiwf check || exit 1` inside your existing hook, or compose hooks with husky/lefthook. The exit code in that case is 1 so CI notices.

**What aiwf does *not* touch:**

- Skills outside the `aiwf-*` namespace. Your own user-authored skills sit next to aiwf's and are never overwritten by `aiwf update`. Optional companion plugins (such as the planned [ai-workflow-rituals](https://github.com/23min/ai-workflow-rituals) marketplace, which uses the `aiwfx-*` and `wf-*` namespaces) are installed by Claude Code into separate plugin directories — they don't share `.claude/skills/` with aiwf core.
- Anything under `.claude/agents/`, `.claude/commands/`, `.claude/output-styles/`, your `.claude/settings.json`, or any other path under `.claude/` outside `skills/aiwf-*/`.
- An existing `CLAUDE.md`, `.gitignore`, or `aiwf.yaml`.
- Anything outside the consumer repo. There are no writes to `~/.claude/`, no changes to your MCP server config, and no API settings touched.

**Why the `aiwf-*` skills are gitignored.** The materialized skills are a derivable cache: `aiwf init` and `aiwf update` regenerate them byte-for-byte from the binary's embedded copies. Gitignoring the cache rather than committing it means teammates on different `aiwf` versions don't fight merge conflicts, an old `git checkout` doesn't drag stale skill text along with it, and the source of truth stays in the binary. `aiwf doctor` byte-compares the on-disk copies against the embedded ones and surfaces drift; `aiwf update` is the one-button restore.

---

## Validators (`aiwf check`)

Nine validators run on every `aiwf check` invocation, and on every `git push` via the pre-push hook installed by `aiwf init`:

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

Each finding renders as `path:line: severity code[/subcode]: message — hint: <action>`. The `hint` is also exposed as `hint` in the `--format=json` envelope so downstream tools can surface it.

Exit codes: `0` no errors (warnings allowed), `1` errors found, `2` usage error, `3` internal error.

---

## Beyond the PoC

The PoC is deliberately self-contained: markdown files in the consumer repo, no server, no external sync. That is the right shape for proving the kernel works.

The longer-term aspiration is a modular architecture where a *backend adapter* can connect the local entity model to an external PM system — GitHub Issues, Linear, Jira, Azure DevOps, etc. — so a team that lives partly in `aiwf` and partly in their existing tracker can have the two stay in step. The pieces the PoC has already committed to make this plausible without re-architecting:

- **Closed-set entities with stable ids.** A milestone with id `M-007` is the obvious target for a sync adapter to map onto a Linear issue or a GitHub issue number.
- **Structured commit trailers** (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) on every mutation. An adapter can read `git log` after each push and replay the lifecycle into the external system without scraping the markdown.
- **Validate-then-write semantics.** The chokepoint is already factored: a backend adapter can hook in at the same boundary that the pre-push hook uses today, so an outbound sync only happens against a tree that already validates.

That said: no adapter has been implemented, the adapter interface is not yet designed, and the choice of which backend to support first will be driven by the first real consumer who needs it — not by speculation. This section is direction, not commitment.

If you would find a particular backend valuable, opening an issue with the use case is the right move; that is what would prioritise it.

---

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
