# PoC plan — four sessions

This is the working document for the `poc/aiwf-v3` branch. Four focused sessions, each with a deliverable that runs end-to-end before moving on. Mark items as you go; commit per logical step.

For the design context that justifies this shape, see [`poc-design-decisions.md`](poc-design-decisions.md). For the engineering principles, see the root [`CLAUDE.md`](../CLAUDE.md) and [`tools/CLAUDE.md`](../tools/CLAUDE.md).

---

## Session 1 — Foundations and `aiwf check`

**Goal:** an executable that loads the tree, validates it, reports findings. No mutating verbs yet.

- [x] Go module skeleton in place under `tools/cmd/aiwf/` and `tools/internal/`.
- [x] Frontmatter parser (use `gopkg.in/yaml.v3`).
- [x] Tree loader: walks `work/epics/**`, `work/gaps/**`, `work/decisions/**`, `work/contracts/**`, `docs/adr/**`. Parses every entity into a typed in-memory model.
- [x] Six kind types defined as Go structs with their hardcoded status enums.
- [x] `aiwf check` with these checks (each as a small function):
  - [x] `ids-unique` — no duplicate ids (severity: error). Detected via path prefix collision.
  - [x] `refs-resolve` — every reference field resolves to an existing entity of the kind permitted by the frontmatter schema (severity: error). Findings distinguish *unresolved* (no such id) from *wrong-kind* (id exists but is the wrong kind).
  - [x] `status-valid` — every status is in the allowed set for the kind (severity: error).
  - [x] `frontmatter-shape` — required fields present, types correct (severity: error).
  - [x] `no-cycles` — no cycle in `depends_on` (milestone DAG) or in the `supersedes`/`superseded_by` chain (ADR DAG) (severity: error).
  - [x] `contract-artifact-exists` — for every contract, `artifact:` is a relative path with no `..` segments that resolves to an existing file *inside* the contract directory (severity: error).
  - [x] `titles-nonempty` — title is set and non-empty (severity: warning).
  - [x] `adr-supersession-mutual` — if `A.superseded_by = B`, then `B.supersedes ⊇ {A}` (severity: warning).
  - [x] `gap-resolved-has-resolver` — addressed gap has non-empty `addressed_by` (severity: warning).
- [x] JSON output (`--format=json`) and human-readable text (default).
- [x] Exit codes: `0` clean, `1` findings, `2` usage error, `3` internal.
- [x] Synthetic-tree fixtures under `testdata/`, one per finding type.

**Deliverable:** `aiwf check` runs against a hand-crafted `work/` directory and reports findings correctly.

**Shipped (commit `162bf54`):** entity package (six kinds, status enums, id regexes, frontmatter parser), tree loader, nine validators, JSON + text renderers, exit codes, fixture-driven integration test (`testdata/clean` and `testdata/messy`).

---

## Session 2 — Mutating verbs and commit trailers

**Goal:** the verbs that produce git commits with structured trailers.

- [x] `aiwf add epic --title "..."` — allocate `E-NN`, write `work/epics/E-NN-<slug>/epic.md`, commit.
- [x] `aiwf add milestone --epic E-NN --title "..."` — allocate `M-NNN`, write file under epic, commit.
- [x] `aiwf add adr --title "..."` — allocate `ADR-NNNN`, write file, commit.
- [x] `aiwf add gap --title "..." [--discovered-in M-NNN]` — allocate `G-NNN`, commit.
- [x] `aiwf add decision --title "..." [--relates-to E-NN,M-NNN]` — allocate `D-NNN`, commit.
- [x] `aiwf add contract --title "..." --format <fmt> --artifact-source <path>` — allocate `C-NNN`, create directory + `contract.md`, copy artifact into `schema/`, commit.
- [x] `aiwf promote <id> <status>` — read entity, validate transition (one Go function per kind), edit frontmatter, commit.
- [x] `aiwf cancel <id>` — promote to the kind's terminal-cancel status (`cancelled`/`wontfix`/`rejected`/`retired`).
- [x] `aiwf rename <id> <new-slug>` — `git mv` + commit. The id is preserved; title is unchanged (edit frontmatter manually if you want it tracked).
- [x] `aiwf reallocate <id|path>` — pick next free id, `git mv`, walk every entity's frontmatter and rewrite reference fields, surface body-prose references as findings, commit. Accepts a path (instead of an id) when the id is ambiguous — required after a merge collision where two files share the same id.
- [x] Every mutating verb computes the projected new tree in memory, runs `aiwf check` against the projection, and either (a) writes files and creates the single commit when clean, or (b) returns findings without touching the working tree. No rollback path: nothing is written until the projection is known good.
- [x] Every commit-producing verb writes structured trailers: `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`. `reallocate` additionally writes `aiwf-prior-entity: <old-id>` so both the old and new id's histories are queryable.
- [x] Round-trip tests for each verb against a fresh git repo fixture.

**Deliverable:** end-to-end planning workflow works. `aiwf init && aiwf add epic && aiwf add milestone && aiwf promote ... && aiwf rename ...` produces a sensible git history.

**Shipped (commits `9230fa4`, `deaf72f`):** five mutating verbs (add for all six kinds, promote, cancel, rename, reallocate); `entity` extended with serialize/slug/templates/transitions/allocator; new `gitops` and `verb` packages; `Apply` orchestrator; PlannedFiles overlay so `contract-artifact-exists` validates the projected world; round-trip tests for every verb. Follow-up commit added edge-case coverage (reallocate by path/contract; cancel-already-terminal; same-slug rename; CLI dispatcher tests; actor regex fix) and the `projectionFindings` diff so pre-existing tree errors don't block unrelated verbs.

---

## Session 3 — Skills, history, hooks

**Goal:** the AI can use it; `git log` becomes queryable.

- [x] Skill markdown files written and embedded via `embed.FS`. Skills shipped:
  - [x] `wf-add` — how to create each kind with proper frontmatter.
  - [x] `wf-promote` — how to advance status legally per kind.
  - [x] `wf-rename` — how to rename without breaking references.
  - [x] `wf-reallocate` — how to resolve id collisions.
  - [x] `wf-history` — how to ask "what happened here?".
  - [x] `wf-check` — what `aiwf check` reports and how to fix common findings.
- [x] `aiwf init` (idempotent; safe to re-run; produces no git commit — the user commits when ready):
  - [x] writes `aiwf.yaml` (~10 lines) at the consumer repo root if missing; preserves an existing file unchanged. The `actor` field defaults to `human/<local-part-of-git-config-user.email>` (e.g., `human/peter` for `peter@example.com`); if neither `user.email` nor `user.name` is set, errors with an instruction to set git config or pass `--actor`. The actor value (whether derived or explicit) is validated against `^\S+/\S+$` before write; the same regex validates `aiwf.yaml`'s `actor:` field on every verb invocation and any `--actor` flag override.
  - [x] scaffolds `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/` if missing; never modifies existing directories or their contents.
  - [x] materializes skills to `.claude/skills/wf-*/SKILL.md` (wipe-and-rewrite per the cache contract; non-`wf-*` skill directories are untouched).
  - [x] appends materialized-skill paths to `.gitignore` if not already present; does not rewrite the file.
  - [x] writes a short `CLAUDE.md` template only if the file is missing.
  - [x] installs `.git/hooks/pre-push` that runs `aiwf check`. The hook carries an `# aiwf:pre-push` marker comment. If a hook exists with the marker → overwrite (idempotent). If a hook exists without the marker → refuse with a useful error explaining how to integrate `aiwf check` into the existing hook manually, or use a hook manager (husky/lefthook) that composes hooks.
  - [x] pre-existing entity files in `work/` and `docs/adr/` are not modified or validated by `init`; they show up as findings on the next `aiwf check` and serve as the migration to-do list when adopting `aiwf` against an existing repo.
- [x] `aiwf update` — remove every `.claude/skills/wf-*/` directory and re-materialize from the binary's embedded skills (no commit; updates gitignored files). Directories not matching `wf-*` are untouched (user-authored skills are namespace-isolated).
- [x] `aiwf history <id>` — read `git log` filtered for `aiwf-entity: <id>` *or* `aiwf-prior-entity: <id>` trailers (so reallocate events are visible from both the old and new id). Default output is one line per event: `DATE  ACTOR  VERB  DETAIL  COMMIT`, where `DETAIL` is the commit subject line shaped by the verb at commit time (`"title"` for add, `old → new` for promote, `slug → <new>` for rename, `→ cancelled` for cancel, `<old-id> → <new-id>` for reallocate). `--format=json` mirrors `aiwf check`'s machine-readable contract. Trailer-matched events only — `aiwf history` does not show side-effect file edits (use `git log -- <path>` for byte-level history).
- [x] `aiwf doctor` — check binary version vs. `aiwf.yaml`'s `aiwf_version`, byte-compare each materialized skill against its embedded version and report drift, check id-collision health.
- [x] Tests: `aiwf init` in a fresh git repo produces the expected layout; `aiwf history` returns the expected events for a multi-step fixture.

**Deliverable:** in a fresh consumer repo, `aiwf init` sets things up; the AI host (Claude Code) sees the skills; the pre-push hook catches errors before push.

**Shipped:** new `skills` package with embedded `wf-*/SKILL.md` files for the six verbs; new `config` package owning `aiwf.yaml` parse/validate/write (and `--actor` resolution now consults it); new `initrepo` package with idempotent setup (config, scaffolding, skill materialization, `.gitignore` append, `CLAUDE.md` template, marker-aware `pre-push` hook); four new CLI subcommands (`init`, `update`, `history`, `doctor`); `gitops.GitDir` helper for worktree-aware hook install; `aiwf history` consumes structured trailers via `git log --grep` with `\x1f`/`\x1e` field separators and queries both `aiwf-entity:` and `aiwf-prior-entity:` so reallocate events surface from either id; `aiwf doctor` byte-compares embedded vs. on-disk skills and runs `ids-unique` from `check`. Coverage: `initrepo`, `skills`, `config` unit-tested; CLI dispatcher tests cover init/update/history/doctor through the top-level `run`.

---

## Session 4 — Polish for real use

**Goal:** ready for use on a real project.

- [x] `aiwf render roadmap` — print a markdown table of epics + milestones; with `--write` updates `ROADMAP.md` and commits.
- [x] `aiwf doctor --self-check` — runs all the verbs against a temp directory.
- [x] Error-message polish — every finding is one line, names file:line, suggests a fix.
- [x] README polish — clear install instructions, quick-start that works.
- [x] A short usage walk-through in `docs/` showing a typical first session.

**Deliverable:** the framework is good enough to start using on a real project.

---

## Total

Roughly 3–4 days of focused work. After session 4 the framework is small, self-contained, and self-validating. Real use surfaces the next priority; nothing else is committed to in advance.

---

## Notes for the working sessions

- The PoC branch is not planned to merge back to `main`. Commit directly on the branch; no PR ceremony.
- Conventional Commits subject lines (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs(poc): ...`) keep the log readable.
- If session 3's deliverable is not reached within a reasonable timebox, abandon and patch the existing framework instead. The PoC's value is bounded; do not over-invest.
- When in doubt, the smaller change is the right change. KISS and YAGNI from the root [`CLAUDE.md`](../CLAUDE.md) are load-bearing here.
