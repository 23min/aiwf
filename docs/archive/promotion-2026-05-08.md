# PoC promotion — `poc/aiwf-v3` → `main` (2026-05-08)

> **Archived record.** This is the working plan that drove the trunk-promotion procedure on 2026-05-08, kept for archaeology — the decisions made along the way, the running order that worked, and the conflict-resolution rules that produced the merge commit. The promotion itself is in git history (merge commit `e0a7fe5` and the surrounding sequence on `main`); this document is the *why* behind the *what*.

## Goal

Promote `poc/aiwf-v3` (the working framework, 768 commits ahead of `main`) to be the new trunk, while preserving:

- PoC's full git history as provenance.
- Main's research arc, working-paper v2, explorations, and `.github/` conventions.
- Main's role as the place where future research and `pocvN` drafts can happen.

The "intentional isolation" that PoC's CLAUDE.md describes was a working tool, not the end state. This plan re-unites the two trunks behind a single merge commit that reads as "PoC graduates."

---

## Decisions already made

1. **`docs/pocv3/` keeps its name** for now. May refactor docs later.
2. **Old design docs archive, not delete.** `architecture.md`, `build-plan.md`, original `working-paper.md` move to `docs/archive/`. Post-merge a new working paper (aiwf's thesis) gets written from scratch.
3. **`docs/continued-research` merges to `main` *first*** as a prerequisite — its content lands on main before the promotion procedure begins.
4. **`.scratch/` surveys graduate to `docs/explorations/surveys/`.** What's listed here gets tracked; anything left in `.scratch/` stays untracked working-tree-only.
5. **Strategy is a real `git merge --no-ff`** of prep-PoC into prep-main. PoC's history is provenance; we don't squash. Visually large fan-in accepted.

---

## Running order

### Step 1 — Merge `docs/continued-research` → `main`

Normal PR / merge. Brings the latest research arc (07–13, surveys list, refined working-paper v2, `docs/explorations/05-policy-model-design.md`) onto main before the promotion begins. After this step, `main` ≈ `docs/continued-research`.

### Step 2 — Graduate surveys to `docs/explorations/surveys/`

Small PR onto `main`. Move (and `git add`) tracked copies of:

- `.scratch/flowtime/` → `docs/explorations/surveys/flowtime/`
- `.scratch/liminara/` → `docs/explorations/surveys/liminara/`
- (From the PoC worktree at `/Users/peterbru/Projects/ai-workflow-v2`) `.scratch/governance-html-concept/` → `docs/explorations/surveys/governance-html-concept/`

Manual copy is required for the third one — `.scratch/` is per-worktree and gitignored, so neither merge nor branch op can move it for us. Copy from the PoC worktree's filesystem path to this worktree before committing.

Triage pass before committing: skim each survey, drop pure-scratch debris if any, keep what's useful as research-context.

Add a tiny `docs/explorations/surveys/README.md` explaining what these are (extracted/mined notes that fed the research arc).

### Step 3 — Prep PoC *(DEFERRED — user is actively working in `/Users/peterbru/Projects/ai-workflow-v2`)*

Resume after the user signals PoC is at a clean stopping point.

When that happens, on a `prep/promote-poc` branch off `poc/aiwf-v3`:

1. Rewrite the opening of `CLAUDE.md`: drop "aiwf PoC branch" / "intentionally isolated from main" framing. Describe the framework as the trunk it's about to be. Keep engineering principles, dogfooding rules, Go conventions section as-is.
2. Reframe `README.md` to describe a working framework, not a PoC under construction.
3. Scan `ROADMAP.md`, `CHANGELOG.md`, `STATUS.md` for "PoC" language that should now read as historical or be dropped.
4. Quick grep across `docs/pocv3/` for prose that asserts "the PoC will…" / "post-PoC we will…" — adjust where misleading. Don't refactor `docs/pocv3/` paths (decision 1).

This phase is one PR onto `poc/aiwf-v3`, kept narrow so the actual merge in Step 5 is mechanically simple.

### Step 4 — Prep `main`

On a `prep/main-pre-merge` branch off `main` (i.e. after Step 1 has landed):

1. Create `docs/archive/` with a `README.md` explaining "pre-PoC design documents kept for archaeology."
2. `git mv docs/architecture.md docs/archive/architecture.md`
3. `git mv docs/build-plan.md docs/archive/build-plan.md`
4. `git mv docs/working-paper.md docs/archive/working-paper-v1.md` (the v2 we keep as `docs/working-paper.md` — confirm which is which before moving).
5. `git rm -r tools/cmd/aiwf/ tools/CLAUDE.md` (124-line stub + Go conventions doc — both superseded by PoC's `cmd/` + `internal/` layout and PoC's CLAUDE.md Go-conventions section).
6. If `tools/` is now empty, remove it.
7. Add CHANGELOG entry under `[Unreleased]` noting the archival + stub removal.

This phase is one PR onto `main`. After it lands, `main` is structurally lean and ready to receive PoC.

### Step 5 — The merge *(blocked until Step 3 + Step 4 are both done)*

```
git checkout main
git merge --no-ff prep/promote-poc
```

Conflict-resolution rule:

- **PoC tree wins** for all code, `cmd/`, `internal/`, `e2e/`, `scripts/`, `Makefile`, `aiwf.yaml`, `go.mod`, `go.sum`, `CLAUDE.md`, `README.md`, `ROADMAP.md`, `STATUS.md`, `CHANGELOG.md`, `docs/pocv3/`, `docs/adr/`, `site/`, `work/`, root `aiwf` binary.
- **main tree wins** for `docs/research/`, `docs/explorations/` (including the new `surveys/`), `docs/archive/`, `docs/working-paper.md` (v2), `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`, `.github/workflows/pr-conventions.yml`, `CONTRIBUTING.md`.

If the prep phases were thorough, very few files should appear with conflicting edits in both trees — most "conflicts" should be one-side-only delete-vs-add, resolved by the rule above.

**CHANGELOG.md** is one of the "PoC wins" files per the rule above. At merge time, take PoC's wholesale (`git checkout --theirs CHANGELOG.md` if a conflict surfaces, then `git add CHANGELOG.md`). Main's `[Unreleased]` entries are temporarily dropped — but the merge commit's first parent (`HEAD^1`, i.e. pre-merge `main`) still has them in git history. The reconciliation happens as a focused follow-up commit in **Step 6.C** post-merge, where main's pre-merge `[Unreleased]` is recovered via `git show HEAD^1:CHANGELOG.md`, blended with PoC's `[Unreleased]`, and obsolete entries are dropped.

After resolving, before committing the merge:

- `go test -race ./...`
- `golangci-lint run`
- `bash tests/test-install.sh` (if PoC carries this — verify path)
- Skim the resulting `docs/` tree to confirm research arc, explorations, archive, and pocv3 all coexist as expected.

**No new tag at the merge.** PoC's existing version tags (`v0.6.0` and earlier) carry through naturally — they stay on their original commits, which are now reachable from `main` after the merge. The next release on `main` continues PoC's sequence (`v0.7.0` whenever it ships), per the user's instruction to keep using the PoC tag scheme without bumping at promotion time.

Just commit the merge with a clear subject (e.g. `chore: merge poc/aiwf-v3 into main (PoC graduates to trunk)`) and move on.

### Step 6 — Post-merge housekeeping

#### A. Immediate validation

1. `go test -race ./...`
2. `golangci-lint run`
3. `aiwf doctor --self-check` — drives every verb against a temp repo.
4. `aiwf check` against the merged tree — no new findings.
5. Skim `docs/` to confirm `research/`, `explorations/`, `archive/`, and `pocv3/` coexist as expected.

#### B. Doc-lint discovery pass

Run `wf-doc-lint` against the merged tree. Reports broken code references, removed-feature docs, orphan files, and doc TODOs. Expected hits:

- References to `docs/architecture.md` / `docs/build-plan.md` (now `docs/archive/...`).
- References to `tools/cmd/aiwf/` or `tools/CLAUDE.md` (removed).
- Cross-anchor breaks beyond the `#beyond-the-poc` one already caught.
- Possible `docs/pocv3/` orphans.

Findings are reports only — feed them into sections C / D as appropriate.

#### C. CHANGELOG.md reconciliation

The merge takes PoC's CHANGELOG body wholesale (per the conflict rule), so main's `[Unreleased]` entries get dropped at merge time and need to be re-added here.

1. **Combine `[Unreleased]` sections.** PoC's `[Unreleased]` holds done-but-untagged engine work (E-14 Cobra; possibly E-17 entity-body, E-18 operator-side dogfooding) — keep those; they ship under the eventual `v0.7.0` whenever cut. Add main's docs work as new `### Added` / `### Changed` entries: research arc, explorations 01-05, surveys, archive, policy-model.
2. **Drop entries that no longer describe reality.** Main's `[Unreleased]` *"Stage 2 PR 1: Go infrastructure scaffold"* described the 124-line stub now removed; remove the entry. Same for the "added a banner to architecture.md / build-plan.md" entry — those files are now archived; the move-to-archive entry from Step 4 is the right successor.
3. **Add a transition marker.** Top of `[Unreleased]`, brief: `### Changed - Repo structure: poc/aiwf-v3 promoted to main on YYYY-MM-DD; engine, planning state, and design research now live in a single trunk.` — gives future readers one landmark to find when the merge happened.
4. **Verify the preamble** says "Releases ship as git tags on `main`" (already corrected in Step 3, but confirm in the merged file).

#### D. Tree hygiene

1. **Verify `.github/workflows/` is a union** of PoC's CI workflows + main's `pr-conventions.yml`. (Flag at merge time; fix here if missed.)
2. Re-run `aiwf render roadmap` to refresh `ROADMAP.md`.
3. Let the pre-commit hook regenerate `STATUS.md` on the next commit, or run `aiwf status --format=md` manually.

#### E. Branch / worktree cleanup

1. **Keep `poc/aiwf-v3`** as a frozen reference (don't delete; it's the historical anchor). Optional: rename to `archive/poc-aiwf-v3` to signal status.
2. **Delete `prep/promote-poc`** — history is now in main.
3. **Delete `docs/continued-research`** — history is now in main.
4. **Triage stale branches:** `chore/research-arc-revision`, `epic/E-17-…`, `milestone/M-066/067/068`, `poc/aiwf-rename-skills`, `work/main-more-research` — likely stale, confirm before deleting.
5. **Decide on the second worktree** at `/Users/peterbru/Projects/ai-workflow-v2` (currently on `poc/aiwf-v3`). Options: switch to main, delete the worktree, or keep as a frozen PoC viewer.

#### F. Push to origin

1. `git push origin main`. Should be a clean fast-forward.
2. Verify `aiwf doctor --check-latest` shows the expected version row.
3. Confirm `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest` still resolves correctly (same module path).

#### G. Follow-up entities (optional)

Open whatever tracking surface fits — kernel epic, gap, or note. Candidates:

1. New working paper (aiwf's thesis), per decision 2.
2. `docs/pocv3/` rename / restructure if you decide to do it.
3. PoC-language sweep in `docs/pocv3/` body prose (deliberately not scrubbed in Step 3).
4. Whether `CONTRIBUTING.md` (came from main) still matches PoC's trunk-development workflow.

#### H. Plan-doc cleanup

1. **Delete `PROMOTION-PLAN.md`** — the temporary working doc, marked for deletion at the top of the file.

---

## Risks / open items

- **Force-push to `main`.** Should not be needed if the merge in Step 5 is a clean `--no-ff` merge of a branch whose base is `main`'s current tip. Verify before pushing; never `--force` to `main` without explicit go-ahead.
- **`tools/CLAUDE.md` deletion** assumes PoC's CLAUDE.md fully covers Go conventions. Diff the two before Step 4 to confirm nothing important is dropped.
- **PoC's `work/` directory** is the dogfooded entity store (per PoC CLAUDE.md). Confirm it should land on `main` as part of the promotion (probably yes — that's the dogfooding) rather than being moved/archived.
- **`docs/continued-research` may keep accumulating commits** while Step 3 is deferred. If so, re-run Step 1 (or rebase prep-main) before Step 4.
- **`origin/poc/aiwf-rename-skills`** branch exists too — confirm it's stale or fold into the plan if relevant.

---

## Status

- [x] Step 1 — merge `docs/continued-research` → `main` *(fast-forward to `67c8079`, local only)*
- [x] Step 2 — graduate surveys to `docs/explorations/surveys/` *(commit `b692251`, flowtime + liminara, governance-html-concept left in PoC `.scratch/` per user)*
- [x] Step 3 — prep PoC *(commit `987273d` on `prep/promote-poc`; CLAUDE.md / README.md / CHANGELOG.md re-framed for trunk; cross-anchor fix in `docs/pocv3/overview.md`. Side-quest: `c9b1ced` on `main` committed the untracked `docs/explorations/05-policy-model-design.md`.)*
- [x] Step 4 — prep main *(commit `d20ded2`; architecture.md + build-plan.md → docs/archive/, tools/ stubs removed, working-paper.md kept in place per user)*
- [x] Step 5 — the merge *(commit `e0a7fe5` — `chore: merge poc/aiwf-v3 into main (PoC graduates to trunk)`. Tests + lint clean pre-commit. Pre-merge main tip recoverable as `HEAD^1` = `c9b1ced` for Step 6.C.)*
- [~] Step 6 — post-merge housekeeping
  - [x] A — validation (tests, lint, `aiwf check`, `aiwf doctor`; non-blocking warnings noted)
  - [x] B — doc-lint discovery + actionable fixes *(commit `3b50108`)*
  - [x] C — CHANGELOG reconciliation *(commit `7c8f987`)*
  - [x] D — tree hygiene (`.github/workflows/` union confirmed; ROADMAP.md is fresh; STATUS.md auto-regen by hook)
  - [x] E — branch cleanup (deleted `prep/promote-poc`, `docs/continued-research`; rest kept per user)
  - [x] F — push to origin *(`febf253..7c8f987 main -> main`)*
  - [x] G — follow-up gaps opened: `G-074` (pocv3 body prose sweep), `G-075` (pocv3 rename decision), `G-076` (CONTRIBUTING.md reconciliation), `G-077` (post-promotion working paper)
  - [x] H — archive this plan as `docs/archive/promotion-2026-05-08.md` (instead of deleting; preserves decisions + procedural reasoning for future readers)
