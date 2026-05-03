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

### G20. `aiwf add ac` accepts prose titles, renders one giant `### AC-N — <title>` heading — **resolved**

`aiwf add ac M-NNN --title "..."` writes the title both into the YAML frontmatter `acs[].title` field AND into a body heading `### AC-N — <title>`. When the title is a short label that's fine, but real-world ACs that the user passes verbatim from a planning conversation often arrive with markdown bold, multiple sentences, or paragraph-length prose — the result is one h3 heading containing 200+ characters of bold-rendered text in the milestone view, not a heading + prose body. Reproducer:

```
aiwf add ac M-NNN --title "**Full embedment inventory.** A machine-reviewable table in the milestone tracking doc enumerates every rule encoded in: (a) ModelValidator.cs, (b) ModelParser.cs, …"
```

Resolved in commit `<TBD>` (feat(aiwf): G20 — refuse prose-y AC titles, add acs-title-prose warning). Took the strict refusal + standing-check pair:

- `entity.IsProseyTitle(s string) bool` — pure detector. Triggers: length > 80 chars, newlines, markdown formatting (`**`, `__`, backticks), link brackets (`](`), or multiple sentences (sentence-ending punctuation followed by space + capital).
- `verb.AddAC` refuses prose-y titles up front with a usage-shaped error pointing the user at the workflow: pass a short label for `--title`, hand-edit the body section under the scaffolded heading to add detail prose, examples, references.
- New `acs-title-prose` (warning) finding in `check/acs.go`; runs on every `aiwf check` pass to catch titles that landed via hand-edits or pre-G20 tooling. Severity is warning, not error — the title is still usable as a label, the user just gets nudged to refactor.

Tests pin the load-bearing cases: the actual G20 reproducer string, single-sentence labels (no false positive), exact 80-char boundary (false), 81-char (true), markdown forms, multi-sentence detection. Verb-level tests confirm the refusal happens before any disk change (zero ACs added) and that the happy short-label path still works.

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

### G22. Provenance model extension surface — **open**

The I2.5 provenance model ([`design/provenance-model.md`](design/provenance-model.md)) deliberately keeps the verb surface narrow. Six known extensions are filed here for future evaluation, all YAGNI for the PoC:

1. **Explicit revoke verb (`aiwf revoke <auth-sha> --reason "..."`).** End an active scope before its scope-entity reaches a terminal status. The trailer slot is reserved (`aiwf-revoked-by:`) but the verb is not implemented in I2.5. Scopes today auto-end only on terminal scope-entity status; a human cannot un-authorize an in-flight scope without forcing the entity to a terminal status.
2. **Time-bound scopes (`--until <date>` or `--for <duration>`).** Auto-end on a wall-clock deadline. Adds a clock dependency to the kernel; not present today.
3. **Verb-set restrictions (`--verbs add,promote`).** Constrain which verbs an agent can invoke under a scope. Real safety win in adversarial settings; significant added complexity.
4. **Pattern scopes (`--pattern "M-007/*"`).** Scope by id pattern instead of (or in addition to) reference-graph reachability. More flexible; harder to verify; the "did the agent act outside scope?" question gets fuzzier.
5. **Sub-agent delegation.** Whether an `aiwf-verb: authorize` commit may itself be inside a scope (an agent authorizing another agent). The mutually-exclusive pair `(aiwf-verb: authorize, aiwf-on-behalf-of:)` is *not* enforced in I2.5; G22 owns the policy decision when real friction shows up.
6. **Bulk-import per-entity actor attribution.** `aiwf import` today writes one collapsed `aiwf-actor:` trailer for the whole import. When the source data carries per-row author info, the importer should write per-entity `aiwf-actor:` pairs instead. Solves the migration case where authorship is recoverable only via `git blame` on the v1 source.

Severity: Low. Each item is a clear extension path; the I2.5 model leaves room for all of them without architectural retrofits.

---

### G23. Delegated `--force` via `aiwf authorize --allow-force` — **open**

Per the I2.5 provenance model: `--force` is human-only. An LLM operating in a scope cannot `--force` even when the human has authorized that scope. The path is for the LLM to prompt the human, who then invokes `aiwf <verb> --force --reason "..."` directly.

This is the right default. But occasional friction is plausible: a long-running autonomous scope where every kernel-refusal-that-needs-overriding becomes a synchronous prompt to the human. The escape hatch would be a flag on `aiwf authorize` — `--allow-force` — which extends the agent's authorization to include forced acts within the scope. Even then, the trailer would still write `aiwf-principal: human/...` (the human authorized force-permitted scope), preserving the "sovereign acts trace to a named human" rule.

YAGNI for the PoC. The honest minimum-viable path forward is to ship I2.5 without it, watch where the friction lands, and revisit. If `--allow-force` ships, it's a flag-and-finding addition (`provenance-force-disallowed-in-scope` for misuse), not an architectural change.

Severity: Low. Specific named extension worth its own audit row so it doesn't get folded into G22 and lost.

---

### G21. Kernel surface is partially undocumented for AI assistants — **resolved (finding-code axis); other axes verified clean**

Resolved across commits `5a7df46` (docs(aiwf): document case-paths and load-error in aiwf-check skill) and `351e694` (feat(aiwf): extend discoverability policy from provenance-* to all codes).

A six-axis audit (verbs, flags, finding codes, trailer keys, body-section names, YAML fields) against the four CLAUDE.md-named documentation channels (`aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, CLAUDE.md / tools/CLAUDE.md, and any markdown under `docs/pocv3/`) found:

- **Verbs (22), flags (40+), body sections (18), YAML fields (20+):** every item documented in at least one channel; zero gaps.
- **Trailer keys (15):** zero gaps. `aiwf-prior-parent` is mentioned in `design-lessons.md` (reachable via `tools/CLAUDE.md`); the rest are in printHelp or `provenance-model.md`.
- **Finding codes (18 active across `check/` and `contractcheck/`):** two genuinely undocumented — `case-paths` and `load-error`. Both inline string literals (not named constants), invisible to the prior `provenance-*`-scoped policy. Added to the `aiwf-check` skill's errors table.

The `PolicyFindingCodesAreDiscoverable` policy in `tools/internal/policies/discoverability.go` was extended in two ways so this gap can't reopen unnoticed:

1. *Code enumeration* expanded from "named `provenance-*` constants" to "every kebab-case finding code anywhere," via a new `loadCheckCodeLiterals` AST walk over `Finding{Code: "..."}` literals across `check/` and `contractcheck/`.
2. *Channel set* expanded from "`aiwf-check` skill + `main.go`" to the full CLAUDE.md set: every embedded skill, `main.go`, both `CLAUDE.md` files, and every markdown under `docs/pocv3/`.

`go test ./tools/internal/policies/...` is the CI-enforced safety net: any new finding code added without a documentation mention fails `TestPolicy_FindingCodesAreDiscoverable` before merge.

The audit's other-axes verification was a one-shot pass; if a future iteration adds a new axis (e.g., a JSON envelope field schema), the audit-and-policy pattern from this gap is the template.

---

<a id="g24"></a>
### G26. `findings_have_tests` policy mirrors G21's old shape — only sees named-constant codes — **resolved**

Resolved in commit `f37dc07` (feat(aiwf): G26 — extend findings_have_tests to inline-literal codes). After G21 broadened `PolicyFindingCodesAreDiscoverable` to enumerate every kebab-case finding code (named constants + inline `Code: "..."` literals across `check/` and `contractcheck/`), `PolicyFindingCodesHaveTests` was left on the old narrow enumeration: it only verified test references for named-constant codes (i.e., the `provenance-*` family). Inline-literal codes — most of the pre-I2.5 surface, including `acs-tdd-audit`, `acs-shape`, `case-paths`, `load-error`, etc. — could be production-emitted without any test asserting the exact code string. A typo in the emission site would slip through every existing test.

The fix shares `loadCheckCodeLiterals` with the discoverability policy so the two now operate on the same code population. For inline-literal codes the only acceptable test reference is the quoted string value (no constant name to fall back on). Codes also declared as constants are deduped against the named pass.

The broadened policy immediately surfaced one real violation: `acs-tdd-audit` was emitted in `check/acs.go` and exercised by three tests in `acs_test.go` that asserted severity and entity-id but never the code string. Two of those tests were tightened to also assert `Code == "acs-tdd-audit"` — a typo at the emission site would now fail them.

Severity: Low. The policy gap was structural (test-coverage rule didn't match the docs-coverage rule's scope); the one real violation it found was a single under-tested code, not a correctness regression. But the symmetry is the point: G21 and G26 together now mean every kernel finding code is *both* documented in at least one channel *and* asserted by string in at least one test — the kind of pair where letting one half drift while the other tightens is exactly how subtle holes open up.

---

<a id="g24"></a>
### G25. Pre-commit policy hook is per-clone, install-by-copy — drifts silently — **resolved**

Resolved in commit `40c3d2d` (build(repo): G25 — adopt core.hooksPath for the tracked pre-commit hook). The policy gate that enforces G21's discoverability rule (and every other policy under `tools/internal/policies/`) lived in `.git/hooks/pre-commit` — installed per clone via `make hooks` (install-by-copy of `scripts/git-hooks/pre-commit`). The model has two failure modes:

1. **Drift.** The installed copy can fall behind the tracked source between `make hooks` runs. Concrete reproducer at gap-filing time: this very repo's tracked `scripts/git-hooks/pre-commit` (May 1) only regenerated `STATUS.md`; the installed `.git/hooks/pre-commit` had drifted ahead with the policies test gate. Nothing detected this — the only signal would have been a contributor running `make hooks` and noticing the file change in the diff.
2. **First-clone footgun.** A new contributor who clones and starts committing without running `make hooks` skips the policy gate entirely. CI catches it eventually, but every PR that lands in that window is one the contributor could have caught locally.

The fix is structural, not just procedural: switch from install-by-copy to `git config core.hooksPath scripts/git-hooks`. Git then executes the tracked file directly — no `.git/hooks/<name>` copy exists, no drift can occur, and `git pull` updates everyone's hook in sync with the policy it enforces. The `make hooks` target is renamed to `make install-hooks`; the README's new "Contributing to aiwf" section instructs new contributors to run it once after cloning. The hook itself stays tolerant (missing `go` is silently skipped) so doc-only commits from a non-Go environment aren't blocked.

Severity: Medium. Doesn't break correctness when the hook is current, but the safety net the policies package was designed to be is only real for contributors who have the up-to-date hook installed — and the install-by-copy model gave no signal when that wasn't true.

This gap is repo-internal: it applies to the kernel-development repo (`ai-workflow-v2`), not to consumer repos using `aiwf init`. Consumer-side hooks are managed by the kernel binary and refreshed by `aiwf update`; that path has its own drift-detection story under G12 (pre-push) and is out of scope here.

---

<a id="g24"></a>
### G24. Manual commits bypass `aiwf-verb:` trailers; no first-class repair path — **resolved**

Resolved across I2.5 steps 5b, 5c, and 7b: `aiwf cancel <id> --audit-only --reason "..."` and `aiwf promote <id> <status> --audit-only --reason "..."` (commit `bc4183e`) record properly-trailered empty-diff commits on entities already at the named state; `Apply` classifies `index.lock` failures and surfaces the holder PID via `lsof` with no silent retries (commit `6cc0648`); a `provenance-untrailered-entity-commit` warning fires on every push for commits ahead of `@{u}` that touch entity files without `aiwf-verb:` (commit `0e44ad6`); the warning clears once the audit-only commit lands (commit `be2ea27`). Cross-cutting integration test in `9c1b010`. The "git log is the audit log" promise now has both a surface-the-gap signal and a first-class recovery verb.

When a mutating verb (`aiwf cancel`, `aiwf promote`, …) fails partway through and the operator finishes the work with a plain `git commit`, the resulting commit lands without the structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`). The entity reaches its correct state — `aiwf check` is clean — but `aiwf history <id>` and `aiwf status` (both filter `git log --grep "^aiwf-verb: "`) report no event for the change. The audit trail goes silent for events that did happen.

Observed concretely: in a working session, three gap closures (G-021, G-030, G-031 in a separate consumer tree) were committed manually after `aiwf cancel` failed on `.git/index.lock` contention each time. The frontmatter reflects `wontfix`, but `aiwf history` returns "no history" for all three.

There is no clean recovery verb. `aiwf cancel <id> --force --reason "..."` looks like the natural backfill, but `Cancel` at `tools/internal/verb/promote.go:107-109` still errors `"already at target"` even under `--force` (the function-doc comment at lines 91-92 makes this explicit: the guard is intentional because there is no diff to write). The only currently-available repair is an empty hand-crafted commit with the right trailers — i.e., the same kind of manual commit that produced the problem.

**Probable cause of the lock contention.** `aiwf cancel` takes its own lock at `.git/aiwf.lock` (separate from git's `.git/index.lock`), so the two don't collide directly. Inside the verb, `verb.Apply` runs `git mv` → `git add` → `git commit` as subprocesses; the pre-commit hook then runs `aiwf status --format=md` (read-only `git log`) plus `git add STATUS.md`. None of that should contend with itself. The likely culprit is an external process — VS Code's git extension, a file-watcher, or a stale `.git/index.lock` from a prior crash — holding `index.lock` just long enough for the in-flight `git commit` to fail. Capturing the actual `index.lock` error (stderr from a failed `aiwf cancel`) and `lsof .git/index.lock` is the diagnostic next step; the lock-contention root-cause is its own thread, not in scope here.

**Failure modes and consequences.**

1. *Audit-trail gap.* `aiwf history` / `aiwf status` cannot see the change. Downstream readers conclude "no recent activity" when there was; decisions made on those outputs are reading from incomplete data.
2. *Provenance gap.* "Who, when, why" is recoverable only by re-reading the manual commit's prose, which doesn't follow the trailer schema and isn't queryable.
3. *No first-class repair.* The framework provides no verb to backfill an audit-only event. The recovery path that exists is to make the same kind of manual commit that created the problem.
4. *Silent invariant violation.* `aiwf check` passes because frontmatter is consistent. The framework's core promise — "git log is the audit log" (kernel decisions §3 / §4) — is broken without raising any alarm.
5. *Recurrence risk.* If the contention is environmental (concurrent IDE, watcher), it will recur; the framework treats every commit failure as fatal and does not retry, log, or surface the offending process.

**Resolution path.** Folded into I2.5 (`provenance-model-plan.md` steps 5b, 5c, 7b). Three-part fix:

1. *Audit-only recovery mode* — `aiwf cancel <id> --audit-only --reason "..."` and `aiwf promote <id> <status> --audit-only --reason "..."`. Records a properly-trailered, empty-diff commit on an entity already at its target state. Plan step 5b.
2. *Diagnostic instrumentation in `Apply`* — classify lock-contention failures, surface the holder PID via `lsof`, point the operator at the audit-only recovery path. No silent retries. Plan step 5c.
3. *Pre-push trailer audit* — new `provenance-untrailered-entity-commit` warning in `aiwf check` for commits ahead of `@{u}` that touch entity files without `aiwf-verb:`. Plan step 7b.

Severity: **High**. The framework's central correctness story (git log is the audit log) had an unsignalled hole; the I2.5 fix surfaces the gap (warning) and provides the recovery verb (`--audit-only`).

---

<a id="g27"></a>
### G27. Test-the-seam policy missing — verb-level integration tests skipped the cmd → helper integration — **resolved**

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). New `tools/cmd/aiwf/binary_integration_test.go` builds the cmd binary to a tempfile and subprocesses it; two test cases pin (a) ldflags-stamped Version reaches the verb output (`make install` path) and (b) without ldflags, `aiwf version` and `aiwf doctor`'s `binary:` row report the same value (the seam G27 was filed against). Companion fix in `cmd/aiwf/main.go`: `resolvedVersion`'s no-ldflags fallback now returns `version.Current().Version` directly, byte-coherent with the doctor row. Reverse-validated: restoring the v0.1.0 bug shape (`fmt.Println(Version)` printing the unstamped global) fails the fallback test with the exact "literal sentinel" + "seam mismatch" messages.

The policy text in `tools/CLAUDE.md`'s Testing section ("Test the seam, not just the layer") is the durable rule that should prevent the next instance.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf version` returning `"dev"` despite a working `version.Current()` helper. Root cause: tests covered the new helper in isolation but no test exercised the verb that was supposed to use it. The verb's body kept printing an unrelated package-global (`Version`, the ldflags-stamped value defaulting to `"dev"`); the helper was wired into `aiwf doctor`'s `binary:` row but not into the `version` verb. Two parallel sources of truth for "what version am I" coexisted; the test surface covered only the new one.

The bug was caught by a manual smoke test against the v0.1.0 binary post-publish — exactly the wrong place for it to surface. The pattern generalizes: any time a new helper is added that an existing verb *should* adopt, a verb-level test must assert the verb's output reflects the helper's contract. Without that, a future refactor that introduces a new helper alongside an unrelated existing path repeats the bug.

**Resolution path:** Policy added to `tools/CLAUDE.md`'s Testing section ("Test the seam, not just the layer") in the same commit that files this gap. Implementation work to retrofit existing verbs:

1. Add a binary-level integration test (`tools/cmd/aiwf/binary_integration_test.go` or similar): `go build -o $TMP/aiwf ./tools/cmd/aiwf` then run `aiwf version`, `aiwf doctor` as subprocesses, assert their output. This catches the v0.1.0 bug class for every verb whose output depends on `runtime/debug.ReadBuildInfo`, `os.Args[0]`, `os.Executable()`, or `-ldflags`-stamped globals.
2. Audit each existing verb that consumes a shared helper (`version.Current`, `version.Latest`, `entity.SchemaForKind`, etc.) and confirm there is at least one verb-level test asserting the helper is the actual source of truth.

A future `aiwf check`-style policy could detect "exported helper imported by `cmd/aiwf` but no test in `cmd/aiwf` references it" — overkill for the PoC, but the policy framework already exists (G21/G26) and a fourth policy in that family would be cheap.

Severity: Medium. The class of bug is high-impact (shipped correctness regression), and the policy is the durable defense; the implementation work is small.

</details>

---

<a id="g28"></a>
### G28. `version.Latest()` test was implementation-driven, not contract-driven — stale `/@latest` cache went unnoticed — **resolved**

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestLatest_RealProxy` ("version is non-empty") replaced by `TestLatest_RealProxy_ContractTest` which fetches `/@v/list` directly via raw `net/http` (not through `version.Latest`), computes the expected highest semver via a test-side reference implementation, then asserts `version.Latest()` returns that exact value. The reference implementation is deliberately not imported from the version package so a future regression can't be hidden by a matching regression in the helper. New `TestLatest_PrereleaseExcludedFromHighestSelection` pins the pre-release-skipping invariant offline via httptest.

The policy text in `tools/CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf doctor --check-latest` displaying a stale pseudo-version instead of `v0.1.0`. Root cause: `version.Latest()` queried the proxy's `/@latest` endpoint and unit tests served whatever JSON the implementation expected. The real proxy behavior — that `/@latest` and `/@v/list` are cached independently, and `/@latest` can serve a pre-tag pseudo-version answer for hours after the first tag lands — was not modeled. The Go toolchain's own resolver uses `/@v/list` first for exactly this reason; we re-discovered the lesson by shipping the wrong endpoint and noticing in v0.1.0 verification.

The existing real-proxy integration test (`TestLatest_RealProxy`) queries `gopkg.in/yaml.v3` and only asserts the version is non-empty. It would have passed with either implementation choice (`/@latest` returning yaml.v3's tag happens to work because nobody queried that module's `/@latest` before tags existed). The test was *cooperative* — it tested the parsing round-trip, not the resolution semantics.

**Resolution path:** Policy added to `tools/CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") in the same commit that files this gap. The Latest() resolution itself was fixed in v0.1.1 (commit `32672cd`); G28's residual work is the *test* that pins the contract:

1. Tighten `TestLatest_RealProxy` to derive the expected version through an **independent** code path. Concretely: the test fetches `https://proxy.golang.org/<known-tagged-module>/@v/list` directly (without going through `version.Latest`), parses the response, computes the highest semver triple, and asserts `version.Latest()` returns that exact value. Today's "version is non-empty" assertion is replaced by "version matches the independently-derived expected value."
2. Add a multi-tag fixture test using the existing httptest seam to pin the highest-of-N selection logic without network: serve a `/@v/list` body with three or four tags including a pre-release, assert the highest non-pre-release wins.

Severity: Medium. Same class as G27 — the implementation has been fixed; the policy + the contract test are what stop the next instance.

</details>

---

<a id="g29"></a>
### G29. Pseudo-version regex was example-driven, not spec-driven — initial test set missed two of three forms plus `+dirty` — **resolved**

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestParse`, `TestProxyBase`, and `pseudoVersionRE`'s doc comment now cite the upstream specs (`go.dev/ref/mod#pseudo-versions`, `semver.org`, `go.dev/ref/mod#environment-variables`); `TestParse` cases now cover all three pseudo-version forms explicitly (was: form 1 + form 3 only) plus the `+dirty` stamping case for both base shapes. The citations make spec-drift detectable: a future Go-toolchain change to pseudo-version grammar will be flagged by anyone reading the spec, rather than missed because tests were example-driven.

The policy text in `tools/CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The first pass of `version.isTagged` had a `pseudoVersionRE` that only matched the basic `v0.0.0-DATE-SHA` shape. The Go module spec defines three pseudo-version forms (basic, post-tag `vX.Y.(Z+1)-0.DATE-SHA`, pre-release-base `vX.Y.Z-pre.0.DATE-SHA`) and Go's VCS stamping adds the `+dirty` suffix on working-tree builds with uncommitted changes. The regex caught only the first form; the other three were missed.

The bug was caught mid-implementation by a smoke test (the working-tree build of aiwf reported `"v0.0.0-...-...+dirty (tagged)"`), not by the unit-test pass that immediately preceded it. Root cause: test cases were sourced from "the example I had in mind" rather than from the spec. A spec-sourced enumeration would have listed all four shapes from `https://go.dev/ref/mod#pseudo-versions` plus the VCS-stamping behavior on first writing.

**Resolution path:** Policy added to `tools/CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") in the same commit that files this gap. The implementation already covers the cases (regex updated mid-step-2 to `[-.]\d{14}-[0-9a-f]{12}$` and `+dirty` checked separately); G29's residual work is small:

1. Add a `// per https://go.dev/ref/mod#pseudo-versions` comment above the test data in `version_test.go` so the spec-sourcing is visible to future readers.
2. Audit other test sets that enumerate upstream-defined input spaces (frontmatter shapes against YAML 1.2; commit-trailer shapes against `git interpret-trailers`; semver against the semver.org grammar) for analogous unsourced enumerations, and either add the citation or document the omission.

Severity: Low. Bug already resolved; the policy + the citation are the durable defense. The audit pass is one read-through, not a refactor.

</details>

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
| G20 | `aiwf add ac` accepts prose titles, renders one giant heading | Medium   | [x] `e6de134` |
| G21 | Kernel surface is partially undocumented for AI assistants  | Medium   | [x] `5a7df46` + `351e694` |
| G22 | Provenance model extension surface (revoke, time, verb-set, pattern, sub-agent, bulk-import attribution) | Low | [ ] open |
| G23 | Delegated `--force` via `aiwf authorize --allow-force`     | Low      | [ ] open |
| G24 | Manual commits bypass `aiwf-verb:` trailers; no repair path | High     | [x] I2.5 steps 5b/5c/7b (`bc4183e`, `6cc0648`, `0e44ad6`, `be2ea27`) |
| G25 | Pre-commit policy hook is per-clone, install-by-copy — drifts silently | Medium | [x] `40c3d2d` |
| G26 | `findings_have_tests` policy only sees named-constant codes (G21 mirror) | Low | [x] `f37dc07` |
| G27 | Test-the-seam policy missing — verb-level integration tests skipped the cmd → helper integration | Medium | [x] `f810a86` |
| G28 | `version.Latest()` test was implementation-driven, not contract-driven — stale `/@latest` cache went unnoticed | Medium | [x] `f810a86` |
| G29 | Pseudo-version regex was example-driven, not spec-driven — initial test set missed two of three forms plus `+dirty` | Low | [x] `f810a86` |

When an item is closed, mark it `[x]` and append a short note (commit SHA or PR link) to the row's title. When deferred deliberately, mark `[x] (deferred)` and add a one-line rationale either in the row or in the body of the entry.
