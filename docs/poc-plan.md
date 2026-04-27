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

---

## Session 2 — Mutating verbs and commit trailers

**Goal:** the verbs that produce git commits with structured trailers.

- [ ] `aiwf add epic --title "..."` — allocate `E-NN`, write `work/epics/E-NN-<slug>/epic.md`, commit.
- [ ] `aiwf add milestone --epic E-NN --title "..."` — allocate `M-NNN`, write file under epic, commit.
- [ ] `aiwf add adr --title "..."` — allocate `ADR-NNNN`, write file, commit.
- [ ] `aiwf add gap --title "..." [--discovered-in M-NNN]` — allocate `G-NNN`, commit.
- [ ] `aiwf add decision --title "..." [--relates-to E-NN,M-NNN]` — allocate `D-NNN`, commit.
- [ ] `aiwf add contract --title "..." --format <fmt> --artifact <path>` — allocate `C-NNN`, create directory + `contract.md`, optionally move artifact into `schema/`, commit.
- [ ] `aiwf promote <id> <status>` — read entity, validate transition (one Go function per kind), edit frontmatter, commit.
- [ ] `aiwf cancel <id>` — promote to the kind's terminal-cancel status (`cancelled`/`wontfix`/`rejected`/`retired`).
- [ ] `aiwf rename <id> <new-slug>` — `git mv` + frontmatter title update + commit. The id is preserved.
- [ ] `aiwf reallocate <id|path>` — pick next free id, `git mv`, walk every entity's frontmatter and rewrite reference fields, surface body-prose references as findings, commit. Accepts a path (instead of an id) when the id is ambiguous — required after a merge collision where two files share the same id.
- [ ] Every mutating verb computes the projected new tree in memory, runs `aiwf check` against the projection, and either (a) writes files and creates the single commit when clean, or (b) returns findings without touching the working tree. No rollback path: nothing is written until the projection is known good.
- [ ] Every commit-producing verb writes structured trailers: `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`. `reallocate` additionally writes `aiwf-prior-entity: <old-id>` so both the old and new id's histories are queryable.
- [ ] Round-trip tests for each verb against a fresh git repo fixture.

**Deliverable:** end-to-end planning workflow works. `aiwf init && aiwf add epic && aiwf add milestone && aiwf promote ... && aiwf rename ...` produces a sensible git history.

---

## Session 3 — Skills, history, hooks

**Goal:** the AI can use it; `git log` becomes queryable.

- [ ] Skill markdown files written and embedded via `embed.FS`. Skills shipped:
  - [ ] `wf-add` — how to create each kind with proper frontmatter.
  - [ ] `wf-promote` — how to advance status legally per kind.
  - [ ] `wf-rename` — how to rename without breaking references.
  - [ ] `wf-reallocate` — how to resolve id collisions.
  - [ ] `wf-history` — how to ask "what happened here?".
  - [ ] `wf-check` — what `aiwf check` reports and how to fix common findings.
- [ ] `aiwf init` (idempotent; safe to re-run; produces no git commit — the user commits when ready):
  - [ ] writes `aiwf.yaml` (~10 lines) at the consumer repo root if missing; preserves an existing file unchanged. The `actor` field defaults to `human/<local-part-of-git-config-user.email>` (e.g., `human/peter` for `peter@example.com`); if neither `user.email` nor `user.name` is set, errors with an instruction to set git config or pass `--actor`. The actor value (whether derived or explicit) is validated against `^\S+/\S+$` before write; the same regex validates `aiwf.yaml`'s `actor:` field on every verb invocation and any `--actor` flag override.
  - [ ] scaffolds `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/` if missing; never modifies existing directories or their contents.
  - [ ] materializes skills to `.claude/skills/wf-*/SKILL.md` (wipe-and-rewrite per the cache contract; non-`wf-*` skill directories are untouched).
  - [ ] appends materialized-skill paths to `.gitignore` if not already present; does not rewrite the file.
  - [ ] writes a short `CLAUDE.md` template only if the file is missing.
  - [ ] installs `.git/hooks/pre-push` that runs `aiwf check`. The hook carries an `# aiwf:pre-push` marker comment. If a hook exists with the marker → overwrite (idempotent). If a hook exists without the marker → refuse with a useful error explaining how to integrate `aiwf check` into the existing hook manually, or use a hook manager (husky/lefthook) that composes hooks.
  - [ ] pre-existing entity files in `work/` and `docs/adr/` are not modified or validated by `init`; they show up as findings on the next `aiwf check` and serve as the migration to-do list when adopting `aiwf` against an existing repo.
- [ ] `aiwf update` — remove every `.claude/skills/wf-*/` directory and re-materialize from the binary's embedded skills (no commit; updates gitignored files). Directories not matching `wf-*` are untouched (user-authored skills are namespace-isolated).
- [ ] `aiwf history <id>` — read `git log` filtered for `aiwf-entity: <id>` *or* `aiwf-prior-entity: <id>` trailers (so reallocate events are visible from both the old and new id). Default output is one line per event: `DATE  ACTOR  VERB  DETAIL  COMMIT`, where `DETAIL` is the commit subject line shaped by the verb at commit time (`"title"` for add, `old → new` for promote, `slug → <new>` for rename, `→ cancelled` for cancel, `<old-id> → <new-id>` for reallocate). `--format=json` mirrors `aiwf check`'s machine-readable contract. Trailer-matched events only — `aiwf history` does not show side-effect file edits (use `git log -- <path>` for byte-level history).
- [ ] `aiwf doctor` — check binary version vs. `aiwf.yaml`'s `aiwf_version`, byte-compare each materialized skill against its embedded version and report drift, check id-collision health.
- [ ] Tests: `aiwf init` in a fresh git repo produces the expected layout; `aiwf history` returns the expected events for a multi-step fixture.

**Deliverable:** in a fresh consumer repo, `aiwf init` sets things up; the AI host (Claude Code) sees the skills; the pre-push hook catches errors before push.

---

## Session 4 — Polish for real use

**Goal:** ready for use on a real project.

- [ ] `aiwf render roadmap` — print a markdown table of epics + milestones; with `--write` updates `ROADMAP.md` and commits.
- [ ] `aiwf doctor --self-check` — runs all the verbs against a temp directory.
- [ ] Error-message polish — every finding is one line, names file:line, suggests a fix.
- [ ] README polish — clear install instructions, quick-start that works.
- [ ] A short usage walk-through in `docs/` showing a typical first session.

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
