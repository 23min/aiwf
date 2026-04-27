# PoC plan — four sessions

This is the working document for the `poc/aiwf-v3` branch. Four focused sessions, each with a deliverable that runs end-to-end before moving on. Mark items as you go; commit per logical step.

For the design context that justifies this shape, see [`poc-design-decisions.md`](poc-design-decisions.md). For the engineering principles, see the root [`CLAUDE.md`](../CLAUDE.md) and [`tools/CLAUDE.md`](../tools/CLAUDE.md).

---

## Session 1 — Foundations and `aiwf check`

**Goal:** an executable that loads the tree, validates it, reports findings. No mutating verbs yet.

- [ ] Go module skeleton in place under `tools/cmd/aiwf/` and `tools/internal/`.
- [ ] Frontmatter parser (use `gopkg.in/yaml.v3`).
- [ ] Tree loader: walks `work/epics/**`, `work/gaps/**`, `work/decisions/**`, `work/contracts/**`, `docs/adr/**`. Parses every entity into a typed in-memory model.
- [ ] Six kind types defined as Go structs with their hardcoded status enums.
- [ ] `aiwf check` with these checks (each as a small function):
  - [ ] `ids-unique` — no duplicate ids (severity: error). Detected via path prefix collision.
  - [ ] `refs-resolve` — `parent`, `depends_on`, `supersedes`, `superseded_by`, `discovered_in`, `addressed_by`, `relates_to` all resolve (severity: error).
  - [ ] `status-valid` — every status is in the allowed set for the kind (severity: error).
  - [ ] `frontmatter-shape` — required fields present, types correct (severity: error).
  - [ ] `no-cycles` — no cycle in `depends_on` or `parent` (severity: error).
  - [ ] `contract-artifact-exists` — for every contract, the `artifact:` path resolves (severity: error).
  - [ ] `titles-nonempty` — title is set and non-empty (severity: warning).
  - [ ] `adr-supersession-mutual` — if `A.superseded_by = B`, then `B.supersedes ⊇ {A}` (severity: warning).
  - [ ] `gap-resolved-has-resolver` — addressed gap has non-empty `addressed_by` (severity: warning).
- [ ] JSON output (`--format=json`) and human-readable text (default).
- [ ] Exit codes: `0` clean, `1` findings, `2` usage error, `3` internal.
- [ ] Synthetic-tree fixtures under `testdata/`, one per finding type.

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
- [ ] `aiwf reallocate <id>` — pick next free id, `git mv`, walk every entity's frontmatter and rewrite reference fields, surface body-prose references as findings, commit.
- [ ] Every mutating verb runs `aiwf check` post-mutation; abort and roll back working-tree changes if errors are found.
- [ ] Every commit-producing verb writes structured trailers: `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`.
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
- [ ] `aiwf init`:
  - [ ] writes `aiwf.yaml` (~10 lines) at consumer repo root,
  - [ ] scaffolds `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/`,
  - [ ] materializes skills to `.claude/skills/wf-*/SKILL.md`,
  - [ ] adds materialized-skill paths to `.gitignore`,
  - [ ] writes a short `CLAUDE.md` template if none exists,
  - [ ] installs `.git/hooks/pre-push` that runs `aiwf check`.
- [ ] `aiwf update` — re-materialize skills (no commit; updates gitignored files).
- [ ] `aiwf history <id>` — read `git log` filtered for `aiwf-entity: <id>` trailers; pretty-print.
- [ ] `aiwf doctor` — check binary version vs. `aiwf.yaml`'s `aiwf_version`, check skill freshness, check id-collision health.
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
