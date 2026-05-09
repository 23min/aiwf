---
id: M-082
title: Implement aiwf rewidth verb and apply to this repo's tree
status: in_progress
parent: E-23
depends_on:
    - M-081
tdd: required
acs:
    - id: AC-1
      title: aiwf rewidth verb structure with dry-run default
      status: open
      tdd_phase: done
    - id: AC-2
      title: Active-tree file rename to canonical width
      status: open
      tdd_phase: red
    - id: AC-3
      title: In-body reference rewrite for narrow ids
      status: open
      tdd_phase: red
    - id: AC-4
      title: Idempotent and deterministic on canonical tree
      status: open
      tdd_phase: red
    - id: AC-5
      title: Apply aiwf rewidth to this repo at wrap
      status: open
      tdd_phase: red
---
## Goal

Implement `aiwf rewidth`, the migration verb that takes a consumer's narrow-width tree to canonical 4-digit form. Distributed with the kernel binary; idempotent; one commit per `--apply`. Active-tree only; archives untouched. Apply the verb to this repo's tree as M-B's wrap deliverable, producing a single rename + reference-rewrite commit.

After M-B ships, this repo runs at canonical width. Downstream consumers gain a tested distribution of the verb to migrate their own trees on demand. New consumers post-graduation continue to be born canonical (per M-A's allocator).

## Context

ADR-0008 specifies the verb shape: top-level Cobra command, dry-run default, `--apply` commits one transaction, active-tree only, idempotent. M-A made the parser tolerate narrow widths (so the verb can read a narrow tree) and made the allocator emit canonical (so post-migration files are uniform). With those in place, M-B implements the verb's logic and proves it against real data — this repo's own tree.

The verb's reference-rewrite engine handles three concrete patterns: id-form mentions in prose, composite ids (`M-NN/AC-N`), and markdown links targeting active-tree paths. Code fences and archive paths are excluded — the rule is "rewrite active-tree references in active-tree files, leave everything else alone."

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **TDD: required.** Each AC drives a red→green→refactor cycle. AC-3's reference-rewrite engine in particular needs careful test coverage of edge cases (code fences, trailing-digit guards, archive-path exclusion).
- **Pure forward motion.** The verb takes narrow → canonical. No "narrow it back" path. Reversal is `git revert` on the migration commit.
- **Single commit per `--apply` invocation.** Per kernel principle #7. Multi-entity sweeps are a special case in the trailer-keys policy (same shape as `aiwf archive`); trailer is `aiwf-verb: rewidth` with no `aiwf-entity:` trailer.
- **Active tree only.** `<kind>/archive/` files are skipped for renaming; archive paths are skipped for rewriting. ADR-0004's forget-by-default principle for archives is preserved.
- **Idempotent.** Running on an already-canonical or empty tree is a no-op; no commit produced.
- **Skill coverage allowlisted, not per-verb skill.** ADR-0006 case "no skill when --help suffices" applies — the verb is one-shot and self-documenting.
- **Cobra completion drift test passes.** New verb threads through `cmd/aiwf/completion_drift_test.go`.

## Design notes

### Walk order and determinism

The verb walks kinds in a fixed sequence (`epic, milestone, gap, decision, contract, adr` — composite-parent kinds last) and entities within a kind in alphabetical order by current filename. This determinism makes idempotence testable: a second invocation on the same tree visits files in the same order and produces no operations.

### Reference-rewrite scope

Three patterns rewritten:

- **Id-form mentions in prose.** Regex matches `\b[EMGDCF]-[0-9]{1,3}\b` (narrow forms only) and rewrites to canonical 4-digit. Trailing-digit guard via word boundaries: `E-22` matches but `E-220` does not; `E-2200` doesn't match either. Composite-id mentions (`M-NN/AC-N`) are detected separately to avoid double-rewriting.
- **Composite ids.** Regex `\bM-[0-9]{1,3}/AC-[0-9]+\b` rewrites the milestone portion to canonical; AC portion preserved.
- **Markdown links to active-tree paths.** Regex matches `\(work/<kind>/[EMGDCF]-[0-9]{1,3}-<slug>(?:\.md)?\)` and rewrites the embedded id to canonical. Links to `<kind>/archive/...` paths excluded.

**Code fences excluded.** A markdown parser identifies fenced code blocks; content inside fences is not rewritten. Inline backtick spans are also excluded (id mentions inside `` `E-22` `` stay as-is — they typically denote literal id text in documentation).

**Archive paths excluded.** Markdown links targeting `work/<kind>/archive/...` are not rewritten regardless of the embedded id width.

### Apply to this repo's tree

The wrap PR for M-B contains both the verb's source code + tests AND the result of running `aiwf rewidth --apply` against this repo's tree. The diff includes:

- File renames in `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/` from narrow to canonical width.
- Body content rewrites in active-tree markdown files (id mentions, composite ids, markdown links).

Manual diff review is a named checkpoint in the wrap commit body. `aiwf check` and lychee CI green are gates.

## Surfaces touched

- `cmd/aiwf/rewidth_cmd.go` (new) — Cobra command definition.
- `internal/verb/rewidth.go` (new) — verb implementation: walk, rename, rewrite.
- `internal/verb/rewidth/` (new package, optional) — reference-rewrite engine if it grows substantial.
- `cmd/aiwf/completion_drift_test.go` — entry for new verb (auto-discovered or explicit).
- `internal/policies/skill_coverage.go` — allowlist entry for `rewidth` with rationale.
- `internal/policies/<test>` — drift-prevention test if appropriate.
- This repo's `work/`, `docs/adr/` — file renames + body rewrites at wrap time.

## Out of scope

- The drift-check rule `entity-id-narrow-width` — that's M-C.
- ADR-0003 amendment — that's M-C.
- CLAUDE.md commitment #2 update — that's M-C.
- Embedded skill content refresh — that's M-C.
- Rituals plugin coordination — that's M-C.
- Doc-tree narrow-id sweep beyond what `aiwf rewidth` handles automatically — M-C handles `docs/`, `README.md`, `CHANGELOG.md` updates if they're outside the active-tree scope of `aiwf rewidth`.
- Reverse path (`canonical → narrow`). No use case; not implemented.
- Width 5 or 6 future-proofing — YAGNI per ADR-0008.

### AC-1 — aiwf rewidth verb structure with dry-run default

`aiwf rewidth` is a top-level Cobra command. Default invocation is dry-run: walks the active tree, prints planned file renames and reference rewrites (per kind, with counts), exits with code 0 and produces no git commit.

`aiwf rewidth --apply` performs the changes in a single commit per kernel principle #7, with trailer `aiwf-verb: rewidth` and a body listing per-kind rename counts and reference-rewrite counts. No `aiwf-entity:` trailer (multi-entity sweeps are a special case in the trailer-keys policy, same shape as `aiwf archive`).

The verb threads through `cmd/aiwf/completion_drift_test.go`; skill coverage is allowlisted in `internal/policies/skill_coverage.go` with rationale "one-shot migration ritual; --help is sufficient discovery surface" (per ADR-0006's "no skill when --help suffices" case).

Verified by: a synthetic narrow-width fixture tree; `aiwf rewidth` (no flag) shows planned moves with exit code 0 and no git commit produced; `aiwf rewidth --apply` produces exactly one commit with the required trailer and message body shape; the new verb is enumerated by `aiwf --help`'s available-commands list and tab-completes in the test harness.

### AC-2 — Active-tree file rename to canonical width

The verb walks each kind's active directory (`work/epics/<epic-dir>/`, `work/gaps/`, `work/decisions/`, `work/contracts/<contract-dir>/`, `docs/adr/`); for each entity file at narrow width, performs `git mv` to canonical-width filename. Archive entries (`<kind>/archive/`) are skipped entirely — both for renaming during this AC and for any subsequent mixed-state computation by M-C's drift check.

Milestone files inside an epic directory ride with their parent directory's renaming when the parent epic itself renames; if the epic is already canonical and only milestones are narrow (mixed-within-epic), each milestone file renames in place.

Verified by: synthetic fixture trees covering each kind; assertion that post-`--apply`, `find <active-tree> -name '<prefix>-[0-9][0-9][0-9]?-*'` returns empty (no narrow-width filenames in active tree) and `find <archive>` matches pre-state byte-for-byte (archive untouched). Composite cases tested: epic-dir-narrow + milestones-narrow (both rename); epic-dir-canonical + milestones-narrow (milestones rename in place); epic-dir-narrow + milestones-canonical (epic dir renames, milestone names unchanged inside).

### AC-3 — In-body reference rewrite for narrow ids

The verb rewrites references in body content of active-tree files. Three patterns:

- **Id-form mentions in prose**: `E-22` → `E-0022`. Detection scoped to canonical id forms; trailing-digit guards via word boundaries prevent mistaken substitution (e.g., `E-220` should not become `E-0220`; `E-2200` should not match either).
- **Composite ids**: `M-22/AC-1` → `M-0022/AC-1`. AC suffix preserved.
- **Markdown links to active-tree paths**: `[text](work/epics/E-22-foo)` → `[text](work/epics/E-0022-foo)`. Links to archive paths (`work/<kind>/archive/...`) are not rewritten regardless of embedded id width.

**Code fences are excluded** from rewriting — content inside ``` ``` ``` triple-backtick blocks is preserved. Inline backtick spans are also excluded (`` `E-22` `` stays as-is — typically denotes literal id text in documentation).

**Unrelated content** is not modified: text mentioning `E22` (no dash), `e-22` (lowercase prefix), `EM-22` (wrong prefix), or `E-22` inside a URL fragment all stay unchanged.

Verified by: fixture markdown files containing each pattern in expected and unexpected positions; assertion that post-rewrite, the targeted patterns are canonical and the unrelated content is byte-identical to pre-rewrite. Code-fence preservation specifically tested with a fixture containing a fenced block with `E-22` inside — content unchanged after rewrite. Archive-path exclusion tested with a fixture link targeting `work/gaps/archive/G-001-foo.md` — link text unchanged.

### AC-4 — Idempotent and deterministic on canonical tree

Running `aiwf rewidth --apply` on an already-canonical tree is a no-op: no `git mv` operations performed; no body content modified; no git commit created; exit code 0 with message "no changes needed" (or equivalent). Subsequent invocations after a successful `--apply` are also no-ops.

Determinism pinned by stable walk order: the verb visits kinds in a fixed sequence and entities within a kind in alphabetical order by current filename. Test exercises:

- **Narrow tree → first --apply** produces a commit; second invocation produces no commit and prints "no changes needed."
- **Empty tree** (no entities of any kind) — `--apply` is a no-op; exit 0; no commit.
- **Already-canonical tree** — `--apply` is a no-op from the first invocation; exit 0; no commit.

**Mixed-state trees** (some canonical + some narrow) are handled by `--apply` migrating the narrow files only; the canonical files are preserved byte-for-byte. Test fixture includes mixed-state input; assertion that post-run all active-tree files are canonical and the originally-canonical files are byte-identical to pre-run.

The dry-run mode is similarly idempotent: running `aiwf rewidth` (no flag) on a canonical tree prints "no changes needed" with exit 0 and no side effects.

### AC-5 — Apply aiwf rewidth to this repo at wrap

`aiwf rewidth --apply` runs against this repo's tree as part of M-B's wrap PR. The PR contains:

- The verb's source code + tests + completion wiring + skill-coverage allowlist entry.
- The resulting file-rename and reference-rewrite diff over `work/` and active `docs/adr/` (single commit, trailer `aiwf-verb: rewidth`).

Verified at wrap by structural assertions over the resulting tree:

- `find work/ docs/adr -path '*/archive/*' -prune -o -type f -name '[EMGDCF]-[0-9][0-9][0-9]?-*' -print` returns empty (no narrow-width filenames in active tree).
- `grep -rEn 'work/[a-z]+/[EMGDCF]-[0-9]{1,3}-' work/ docs/` (excluding archive paths) returns empty (no markdown links to narrow-width active paths).
- `grep -rEn '\bM-[0-9]{1,3}/AC-' work/ docs/` (excluding archive paths) returns empty (no narrow-width composite ids in active prose).
- `find work/*/archive/ -type f` shows narrow-width forms preserved (archive untouched).
- `aiwf check` post-rename: green.
- Lychee CI: green (no broken path-form links).

Manual diff review checkpoint named in M-B's wrap commit body before the rename PR is approved by a human. The diff is large (~100 file renames + N reference rewrites) and warrants explicit human review even though every assertion above is mechanical.

