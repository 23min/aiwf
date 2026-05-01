# PoC gaps and rough edges

A running list of known gaps, defects, and rough edges in the `aiwf` PoC. Each item has a severity, a concrete location in the source, why it matters, and a proposed fix. The matrix at the end tracks status.

This document is the canonical place to record "we know this is wrong / weak / under-documented" so it doesn't get lost between sessions. When you fix an item, tick it in the matrix and either delete the entry or replace the body with a one-line note pointing at the commit/PR.

The list was produced from a deliberate critique pass on `poc/aiwf-v3` after I1 closed. It is not exhaustive — additions welcome.

---

## Critical / High

### G1. Contract paths can escape the repo (via `..` or symlinks) — **resolved**

Resolved in commit `4ec5d84` (fix(aiwf): G1 — reject contract paths that escape the repo root). New packages `tools/internal/pathutil` and `tools/internal/contractconfig` are the single point of truth for path containment; both `contractcheck` and `contractverify` route through them. `..` traversal, absolute paths outside the repo, out-of-repo symlinks, and symlink loops all produce a `contract-config` / `path-escape` finding, and `contractverify` refuses to invoke a validator on any escaped entry. 100% line coverage on the new code, including a load-bearing test that asserts the validator marker file is never written for an escaped entry.

---

### G2. `Apply` is not atomic on partial failure — **resolved**

Resolved in commit `f77740c` (fix(aiwf): G2 — atomic rollback on Apply failure). Apply wraps its mutations in a deferred rollback that restores the worktree and index to HEAD when any step fails (write error, commit failure, panic). Brand-new files are removed entirely so the next invocation sees a clean tree. New `gitops.Restore` helper. Tests cover write-after-mv failure, git mv failure, brand-new file cleanup, commit failure (no identity), panic recovery, and dedupe of touched paths. apply.go coverage at 94.3% — two defensive branches (compound rollback-also-failed wrap and post-write `git add` failure) marked `//coverage:ignore` per `tools/CLAUDE.md`'s allowance, with the load-bearing rollback path itself at 100%.

---

### G3. Pre-push hook fails opaquely when validators are missing — **resolved**

Resolved in commit `23f4231` (fix(aiwf): G3 — validator-unavailable is a warning, opt-in to strict). New `contractverify.CodeValidatorUnavailable` separate from `CodeEnvironment`. Default rendering: `contract-config` finding with subcode `validator-unavailable`, severity `warning`, exit 0. Opt in to strict mode via `aiwf.yaml: contracts.strict_validators: true` to upgrade to error. `aiwf doctor` now lists each configured validator with available/missing markers and explains the consequence (warning vs. blocking depending on strict_validators). aiwfyaml round-trips the new field. Tests cover the warning path, strict path, the YAML round-trip, and the doctor reporting in both modes.

---

### G4. No concurrent-invocation guard — **resolved**

Resolved in commit `620ecca` (fix(aiwf): G4 — exclusive repo lock for mutating verbs). New `tools/internal/repolock` package wraps POSIX `flock(2)` on `<root>/.git/aiwf.lock` (with a `<root>/.aiwf.lock` fallback for non-git dirs). Every mutating verb acquires the lock before reading the tree; read-only verbs (check, history, status, render without --write, doctor) stay lock-free. Lock acquisition has a 2s timeout; on timeout the second invocation returns `exitUsage` with a clear "another aiwf process is running" message. Stale lockfiles from crashed processes are released by the kernel automatically. Tests cover the load-bearing concurrent-add scenario (one wins / one busy), check-doesn't-lock parity, and the repolock package itself at 90.6% (two defensive branches marked `//coverage:ignore`).

---

## Medium

### G5. Reallocate's prose references are warnings, not errors — **resolved**

Resolved in commit `0e247fe` (fix(aiwf): G5 — reallocate rewrites prose references mechanically). Prose mentions of the old id in any entity body — including the target's own body — are now rewritten in the same commit as the frontmatter rewrite. Word-boundary regex prevents false matches against longer ids (M-001 → M-003 leaves M-0010 untouched). The `reallocate-body-reference` warning code is removed; no half-step "fix it yourself" findings remain. Tests cover the load-bearing rewrite-across-entities scenario, the M-0010-must-not-match edge case, multiple-entities-rewritten-in-one-commit, and the target's own self-reference.

---

### G6. Design docs are stale relative to I1 (contracts) — **resolved**

Resolved in commit `221b9ff` (docs(poc): G6 — sync design decisions and plan with the I1 contract surface). `design-decisions.md` (then named `poc-design-decisions.md`) gains a "Contracts (added in I1)" subsection cross-referencing `contracts-plan.md`, the chokepoint section now mentions contract verification joining the same envelope, the `aiwf.yaml` table includes the `contracts:` row, the verb list reflects the current 14-verb surface (with G2's rollback and G4's lock noted), and the "deliberately not in the PoC" table drops the now-false "schema-aware contract validation" row. `poc-plan.md` gains an "Iteration I1 — Contracts" section listing all eight sub-iterations as done, the obsolete `contract-artifact-exists` and `add contract --format/--artifact-source` lines are annotated as superseded.

---

### G7. Skill namespace is a convention, not a guard — **resolved**

Resolved in commit `971fa88` (fix(aiwf): G7 — track skill ownership via on-disk manifest). Materialize now reads `.claude/skills/.aiwf-owned`, wipes only directories listed in the prior manifest that are no longer in the current embed, writes the embedded skills, and updates the manifest. Foreign directories — including any future `aiwf-rituals-*` plugin — are left alone, even when they share the prefix. The manifest path is added to `MaterializedPaths` so the existing `aiwf init` gitignore step covers it. Tests cover the load-bearing "third-party prefix-sharing dir survives update" scenario plus the regression that real cleanup still works when the prior manifest claims ownership. Manual smoke verified: `aiwf-rituals-tdd/` content survives `aiwf update` byte-for-byte.

---

### G8. Slugify silently drops non-ASCII — **resolved**

Resolved in commit `668031c` (fix(aiwf): G8 — surface a warning when a non-ASCII title's slug drops chars). New `entity.SlugifyDetailed` returns both the slug and the list of dropped runes; `Slugify` is now a thin wrapper. `verb.Add` and `verb.Rename` surface a `slug-dropped-chars` warning naming the dropped characters and the resulting slug — the verb still succeeds (the YAGNI option per the proposed fix). A user who titled an entity `"Café au Lait"` gets `caf-au-lait` plus a clear one-line notice instead of a silent-then-confusing follow-up rename.

---

### G9. `aiwf doctor --self-check` is not run in CI — **resolved**

Resolved in commit `07f8a84` (ci(aiwf): G9 — run aiwf doctor --self-check in CI). New `selfcheck` job in `.github/workflows/go.yml` builds the binary and runs `aiwf doctor --self-check` end-to-end. New `make selfcheck` target for local parity, folded into `make ci`. The push trigger paths gain `Makefile` so a Makefile-only change still runs CI. End-to-end regressions (broken trailers, hook installer drift, missing skills, init-against-fresh-repo failures) are now caught at the CI layer rather than waiting for a user to discover them on upgrade.

---

### G10. macOS case-insensitive filesystem assumption — **resolved**

Resolved in commit `8950874` (fix(aiwf): G10 — surface case-equivalent paths and FS case-sensitivity). New `check.casePaths` validator flags any pair of entity paths that differ only in case (severity error), so a Linux-committed `E-01-foo` + `E-01-Foo` collision is caught at validation time before silently collapsing on macOS reviewer machines. `aiwf doctor` gains a "filesystem: case-sensitive | case-insensitive" line probed via temp-file + uppercased-stat. README's new "Known limitations" section documents the case-sensitivity contract alongside concurrent-invocation, validator-availability, and Unix-only scope.

---

## Low / nits

### G11. `context.Context` not threaded through mutation verbs — **resolved**

Resolved in commit `97283c0` (refactor(aiwf): G11 — thread context.Context through every mutating verb). Every mutating verb (Add, Promote, Cancel, Rename, Move, Reallocate, Import, ContractBind, ContractUnbind, RecipeInstall, RecipeRemove) now takes ctx as its first argument. CLI dispatchers in `tools/cmd/aiwf` already had ctx in scope; tests use `context.Background()` or the runner's `r.ctx`. Today the verb bodies are pure-projection (the IO is in Apply, gitops, tree.Load) so this is a discipline/future-proofing fix, but it aligns with `tools/CLAUDE.md` and gives a clean cancellation handle when verbs grow IO-touching helpers.

---

### G12. Pre-push hook hard-codes binary path at install time — **resolved**

Resolved in commit `8ed5051` (fix(aiwf): G12 — aiwf doctor detects pre-push hook drift). Took option (b) from the proposed fix: hook content stays absolute-path (preserves the existing rationale that hooks shouldn't depend on the user's interactive PATH at push time), and `aiwf doctor` now reads `.git/hooks/pre-push` and reports drift. Five distinct states surface in the output (`ok`, `missing`, `stale path`, `not aiwf-managed`, `malformed`) and stale/missing/malformed increment the problem count so doctor exits non-zero. Re-running `aiwf init` is the documented remediation. Tests cover ok / stale / missing.

---

### G13. No Windows guard — **resolved**

Resolved in commit `dda370d` (fix(aiwf): G13 — refuse Windows up front with one clear message). Took both halves of the proposed fix: (a) `cmd/aiwf` gained `assertSupportedOS` called at the top of `main`, exiting 2 with a clear message on `runtime.GOOS == "windows"`; (b) `repolock` got a Windows stub (`repolock_windows.go`) so the package cross-compiles on Windows — without it, `syscall.Flock undefined` was exactly the deep-stack confusion the gap was filed against. Verified `GOOS=windows go build` produces a clean PE32+ binary that fires the assertSupportedOS message on first run. README's Known Limitations section (added in G10) already documents the Unix-only stance.

---

### G14. Parse failure cascades into refs-resolve findings — **resolved**

Resolved in commit `e2a39ee` (fix(aiwf): G14 — register stub for unparseable entity to suppress refs-resolve cascade). Took the proposed approach: on parse (or read) failure the loader derives the entity's id from its path via the new `entity.IDFromPath` and registers a stub in `tree.Tree.Stubs`; `refsResolve` indexes Stubs alongside Entities so referrers resolve cleanly; `idsUnique` consults Stubs too so duplicate-id collisions involving stubs are still flagged. End-to-end `TestFixture_ProliminalCascadeEndToEnd` reproduces the wild proliminal.net case (E-01 + 12 referrers) and confirms the 13→1 reduction. Verb-level `TestAdd_GapDiscoveredInStubbedEntity` confirms `Tree.Stubs` propagates through `projectAdd`'s shallow copy into the projection check, so verbs adding a referrer to a stubbed entity are not blocked. Coverage on changed code: 100% on `idsUnique`, `refsResolve`, `registerStub`; 89.5% on `IDFromPath`. Upstream skill fix in `ai-workflow-rituals` `d9a726c` removed the wrap-epic instruction that originally triggered this in the wild.

---

### G19. `aiwf init` writes per-skill `.gitignore` entries; new skills aren't covered — **resolved**

Resolved in commit `92f5d51` (fix(aiwf): G19 — emit wildcard skill .gitignore entry, future-proof against new skills). Took the proposed approach: `skills.MaterializedPaths` renamed to `skills.GitignorePatterns`, returning a two-element constant slice (`.claude/skills/aiwf-*/` plus `.claude/skills/.aiwf-owned`). The trailing slash restricts the wildcard to directories. Adding a new aiwf-* skill to the embedded set no longer requires consumers to re-run `aiwf init` to refresh their `.gitignore`. Existing consumers with the per-skill list pick up the two new lines on next `aiwf init`; old entries are harmless (the wildcard subsumes them) and cleanup is the consumer's choice. New `TestInit_GitignoreFutureProof` asserts the property the rename was made for: re-init with the wildcard already present does not duplicate it. Smoke-tested end-to-end against the actual binary.

---

### G18. Contract-config validation is hook-only on `contract bind` and `add contract --validator …` — **resolved**

Resolved in commit `202a14a` (fix(aiwf): G18 — run contractcheck on contract bind / add+bind projection). Took the proposed approach: `ContractBind` and `Add`'s atomic-bind path now run `contractcheck.Run` on the projected `aiwf.yaml.contracts` config and surface any error-level findings whose `EntityID` matches the bound id, before mutating the doc. Catches missing-schema, missing-fixtures, and path-escape (G1) at verb time instead of push time. `contractverify.Run` (the actual validator execution) remains hook-only as a defensible carve-out — documented in `architecture.md` §3. Three new tests cover the verb-side enforcement; existing tests updated to pass a `bindRepo(t)` tmpdir with the referenced schema/fixtures present.

---

### G17. No published per-kind body template for skill authors — **resolved**

Resolved in commit `f4a0fae` (fix(aiwf): G17 — add 'aiwf template' verb, completes the per-kind contract surface). Took the proposed approach: a read-only `aiwf template [kind]` verb mirrors `aiwf schema`. With no kind, emits every kind separated by `KIND: <kind>` headers. With a kind, emits just that template raw and unprefixed, so `aiwf template epic > new_epic_body.md` works as a one-liner. Standard `--format=text|json [--pretty]` envelope. JSON shape: `{result: {templates: [{kind, body}]}}`. Reads from `entity.BodyTemplate` (already exported); no internal data move required. Together with `aiwf schema`, this completes the published per-kind contract that AI scaffolders need to author files outside the `aiwf add` path. Coverage: 85.3% on `runTemplate`, 80% on `writeTemplateText`.

Resolved in commit `9486046` (fix(aiwf): G16 — add id-path-consistent check to catch silent path/id drift). Took the proposed approach: a new `idPathConsistent` check iterates `tree.Entities`, derives the expected id from each path via `entity.IDFromPath`, and emits an error finding on disagreement. Stubs are skipped (constructed from path-derived id by construction). Defensive: if `IDFromPath` returns false for an entity PathKind accepted (impossible by construction), the entity is skipped rather than panicked on. Hint table entry points the user at `aiwf reallocate` for renumbering (rewrites both sides + updates references atomically), `aiwf rename` for slug-only drift, or hand-correction when the user knows which side is right. Pinned by a new fixture file at `tools/internal/check/testdata/messy/work/epics/E-01-orig/M-099-path-id-mismatch.md` (path encodes M-099, frontmatter says M-100) — `TestFixture_Messy` now asserts the new code appears alongside the existing ten. Coverage: 100% on `idPathConsistent`. Completes the path-vs-frontmatter story G14's stub mechanism implicitly relied on.

---

### G15. No published per-kind schema for skill authors — **resolved**

Resolved in commit `0ba0e61` (fix(aiwf): G15 — add 'aiwf schema' verb, single source of truth for entity schemas). Took the proposed approach: a new read-only `aiwf schema [kind]` verb prints the per-kind frontmatter contract — id format, allowed statuses, required and optional fields, and reference fields with cardinality and allowed target kinds — in text or JSON envelope. The verb reads from `entity.SchemaForKind`, which is now the single source of truth that also drives `entity.AllowedStatuses`, `entity.IDFormat`, and (pinned by `TestSchemaMatchesCollectRefs`) the allowed-kinds table consulted by `check.refsResolve`. Skill authors and AI-driven scaffolding tooling can now consume the schema programmatically (`aiwf schema --format=json --pretty`) instead of guessing at field names. Coverage: 100% on `SchemaForKind` / `AllSchemas`; 84.8% on the verb's main and 71.9% on its text renderer (the missing branches are defensive io.Writer error returns).

---

## Status matrix

| ID  | Title                                                       | Severity | Status |
|-----|-------------------------------------------------------------|----------|--------|
| G1  | Contract paths can escape the repo (via `..` or symlinks)   | High     | [x] `4ec5d84` |
| G2  | `Apply` is not atomic on partial failure                    | High     | [x] `f77740c` |
| G3  | Pre-push hook fails opaquely when validators are missing    | High     | [x] `23f4231` |
| G4  | No concurrent-invocation guard                              | High     | [x] `620ecca` |
| G5  | Reallocate's prose references are warnings, not errors      | Medium   | [x] `0e247fe` |
| G6  | Design docs are stale relative to I1 (contracts)            | Medium   | [x] `221b9ff` |
| G7  | Skill namespace is a convention, not a guard                | Medium   | [x] `971fa88` |
| G8  | Slugify silently drops non-ASCII                            | Medium   | [x] `668031c` |
| G9  | `aiwf doctor --self-check` is not run in CI                 | Medium   | [x] `07f8a84` |
| G10 | macOS case-insensitive filesystem assumption                | Medium   | [x] `8950874` |
| G11 | `context.Context` not threaded through mutation verbs       | Low      | [x] `97283c0` |
| G12 | Pre-push hook hard-codes binary path at install time        | Low      | [x] `8ed5051` |
| G13 | No Windows guard                                            | Low      | [x] `dda370d` |
| G14 | Parse failure cascades into refs-resolve findings           | Medium   | [x] `e2a39ee` |
| G15 | No published per-kind schema for skill authors              | Medium   | [x] `0ba0e61` |
| G16 | Path-encoded id and frontmatter id can disagree silently    | Medium   | [x] `9486046` |
| G17 | No published per-kind body template for skill authors       | Medium   | [x] `f4a0fae` |
| G18 | Contract-config validation is hook-only on `contract bind`  | Medium   | [x] `202a14a` |
| G19 | `aiwf init` writes per-skill `.gitignore`; new skills uncovered | Medium | [x] `92f5d51` |

When an item is closed, mark it `[x]` and append a short note (commit SHA or PR link) to the row's title. When deferred deliberately, mark `[x] (deferred)` and add a one-line rationale either in the row or in the body of the entry.
