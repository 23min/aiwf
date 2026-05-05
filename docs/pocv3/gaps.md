# PoC gaps and rough edges

A running list of known gaps, defects, and rough edges in the `aiwf` PoC. Each item has a severity, a concrete location in the source, why it matters, and a proposed fix. The matrix at the end tracks status.

This document is the canonical place to record "we know this is wrong / weak / under-documented" so it doesn't get lost between sessions. When you fix an item, tick it in the matrix and either delete the entry or replace the body with a one-line note pointing at the commit/PR.

The list was produced from a deliberate critique pass on `poc/aiwf-v3` after I1 closed. It is not exhaustive — additions welcome.

---

## Critical / High

### G1. Contract paths can escape the repo (via `..` or symlinks) — **resolved**

Resolved in commit `4ec5d84` (fix(aiwf): G1 — reject contract paths that escape the repo root). New packages `internal/pathutil` and `internal/contractconfig` are the single point of truth for path containment; both `contractcheck` and `contractverify` route through them. `..` traversal, absolute paths outside the repo, out-of-repo symlinks, and symlink loops all produce a `contract-config` / `path-escape` finding, and `contractverify` refuses to invoke a validator on any escaped entry. 100% line coverage on the new code, including a load-bearing test that asserts the validator marker file is never written for an escaped entry.

---

### G2. `Apply` is not atomic on partial failure — **resolved**

Resolved in commit `f77740c` (fix(aiwf): G2 — atomic rollback on Apply failure). Apply wraps its mutations in a deferred rollback that restores the worktree and index to HEAD when any step fails (write error, commit failure, panic). Brand-new files are removed entirely so the next invocation sees a clean tree. New `gitops.Restore` helper. Tests cover write-after-mv failure, git mv failure, brand-new file cleanup, commit failure (no identity), panic recovery, and dedupe of touched paths. apply.go coverage at 94.3% — two defensive branches (compound rollback-also-failed wrap and post-write `git add` failure) marked `//coverage:ignore` per `CLAUDE.md`'s allowance, with the load-bearing rollback path itself at 100%.

---

### G3. Pre-push hook fails opaquely when validators are missing — **resolved**

Resolved in commit `23f4231` (fix(aiwf): G3 — validator-unavailable is a warning, opt-in to strict). New `contractverify.CodeValidatorUnavailable` separate from `CodeEnvironment`. Default rendering: `contract-config` finding with subcode `validator-unavailable`, severity `warning`, exit 0. Opt in to strict mode via `aiwf.yaml: contracts.strict_validators: true` to upgrade to error. `aiwf doctor` now lists each configured validator with available/missing markers and explains the consequence (warning vs. blocking depending on strict_validators). aiwfyaml round-trips the new field. Tests cover the warning path, strict path, the YAML round-trip, and the doctor reporting in both modes.

---

### G4. No concurrent-invocation guard — **resolved**

Resolved in commit `620ecca` (fix(aiwf): G4 — exclusive repo lock for mutating verbs). New `internal/repolock` package wraps POSIX `flock(2)` on `<root>/.git/aiwf.lock` (with a `<root>/.aiwf.lock` fallback for non-git dirs). Every mutating verb acquires the lock before reading the tree; read-only verbs (check, history, status, render without --write, doctor) stay lock-free. Lock acquisition has a 2s timeout; on timeout the second invocation returns `exitUsage` with a clear "another aiwf process is running" message. Stale lockfiles from crashed processes are released by the kernel automatically. Tests cover the load-bearing concurrent-add scenario (one wins / one busy), check-doesn't-lock parity, and the repolock package itself at 90.6% (two defensive branches marked `//coverage:ignore`).

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

Resolved in commit `97283c0` (refactor(aiwf): G11 — thread context.Context through every mutating verb). Every mutating verb (Add, Promote, Cancel, Rename, Move, Reallocate, Import, ContractBind, ContractUnbind, RecipeInstall, RecipeRemove) now takes ctx as its first argument. CLI dispatchers in `cmd/aiwf` already had ctx in scope; tests use `context.Background()` or the runner's `r.ctx`. Today the verb bodies are pure-projection (the IO is in Apply, gitops, tree.Load) so this is a discipline/future-proofing fix, but it aligns with `CLAUDE.md` and gives a clean cancellation handle when verbs grow IO-touching helpers.

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

Resolved in commit `9486046` (fix(aiwf): G16 — add id-path-consistent check to catch silent path/id drift). Took the proposed approach: a new `idPathConsistent` check iterates `tree.Entities`, derives the expected id from each path via `entity.IDFromPath`, and emits an error finding on disagreement. Stubs are skipped (constructed from path-derived id by construction). Defensive: if `IDFromPath` returns false for an entity PathKind accepted (impossible by construction), the entity is skipped rather than panicked on. Hint table entry points the user at `aiwf reallocate` for renumbering (rewrites both sides + updates references atomically), `aiwf rename` for slug-only drift, or hand-correction when the user knows which side is right. Pinned by a new fixture file at `internal/check/testdata/messy/work/epics/E-01-orig/M-099-path-id-mismatch.md` (path encodes M-099, frontmatter says M-100) — `TestFixture_Messy` now asserts the new code appears alongside the existing ten. Coverage: 100% on `idPathConsistent`. Completes the path-vs-frontmatter story G14's stub mechanism implicitly relied on.

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

<a id="g46"></a>
### G46. `aiwf upgrade` fails opaquely when the install package path changes between releases — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G46 — structured remediation when go install reports the package-path-change failure). `runGoInstall` now tees stderr to a captured buffer (no UX change — the user still sees the live stream, the buffer just lets us introspect after the fact). New `pathChangedFromStderr` matches the Go toolchain's `module .* found .*, but does not contain package <subpath>` signature, captures the missing subpath, and `printPackagePathChangedHint` surfaces a kernel-friendly remediation: the install path may have changed, here's the CHANGELOG link, here's the manual `go install <new-path>@<target>` to recover, follow with `aiwf update` to refresh artifacts. False-positive guard: unrelated `go install` failures (network, invalid version, permission) do not trigger the hint. Tests pin both the table-driven detector cases and the runtime path through a stderr-emitting shim.

The fix can't help v0.3.x consumers retroactively — their binary's upgrade verb is frozen. But every release from v0.5.0 forward will produce a structured remediation if a future release relocates the cmd package again.

`aiwf upgrade` invokes `go install <pkg>@<target>` where `<pkg>` is the install path the running binary was built from — hard-coded in `internal/version` (the `pkg` constant in `Latest()` and consumed by the upgrade verb's shell-out). When a release relocates the cmd package within the module — exactly what `v0.4.0` did, moving `cmd/aiwf` from `tools/cmd/aiwf` to `cmd/aiwf` as part of the Go-conventional reorg — the upgrade verb on the *prior* binary (v0.3.x) tries `go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@latest`, the module proxy resolves the module fine, but the subpath no longer exists in the new tag. `go install` exits 1 with `module ... found (v0.4.0), but does not contain package .../tools/cmd/aiwf`. `aiwf upgrade` surfaces the raw exit-1 to the user with no remediation hint.

**Concrete reproducer (real, today):** a consumer running `aiwf v0.3.0` runs `aiwf upgrade` after `v0.4.0` ships. The error message names the missing subpath; nothing in the output tells the consumer that the install path moved or that the recovery is one manual `go install` against the new path.

**Why this matters now:**

1. *We just shipped the break.* `v0.4.0` is the trigger. Any consumer upgrading hits it once.
2. *The fix can't be retroactive.* The v0.3.x binary is already shipped; its `aiwf upgrade` logic is frozen. Whatever we do here improves *future* path-change resilience, not the v0.3.x → v0.4.0 transition.
3. *Path changes are rare but not theoretical.* If we ever rename the binary directory again (e.g., split `aiwf` from a future `aiwf-server`, or move under an aiwf/aiwf org), the same failure mode recurs. The v0.4.0+ binary should handle the next break gracefully.

**Proposed fix (for v0.4.x or later):**

`aiwf upgrade` learns to detect "module found but subpath missing" specifically and either:

- *Print a structured remediation* — "the install path may have changed in `<target>`; check the CHANGELOG at https://github.com/23min/ai-workflow-v2/blob/main/CHANGELOG.md and re-install manually with `go install <module>/<new-subpath>@<target>`." Doesn't try to be clever; tells the user what to do.
- *Try a small set of known-alternate paths.* If `go install <module>/tools/cmd/aiwf` fails with the specific error, retry with `<module>/cmd/aiwf`. Hardcoded fallback list — three entries max, documented in source. Cleaner UX but couples the binary to past path layouts.

Lean: option 1 (structured remediation). YAGNI on the fallback list — we hope to never rename again, and if we do, the next break we know about can ship its own one-time message in the next release notes. The structured remediation generalizes; the fallback list bakes in path archaeology.

**Detection shape:** parse `go install` stderr for `module .* found .*, but does not contain package`. That's the exact phrasing the Go toolchain uses for this case (see `cmd/go/internal/modload/import.go`); pinning the regex to that line is reliable.

**Severity:** Medium. One-time stumble per consumer per path-change release. Doesn't corrupt state, just confuses the user. Filed as a follow-up to `v0.4.0`'s release pain, not as a `v0.4.0` blocker.

---

<a id="g47"></a>
### G47. `aiwf_version` pin is required, set-once, and never auto-maintained — chronic doctor noise — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G47 — retire the aiwf_version pin field). The field is no longer required by `internal/config/config.go` (validation drops the requirement); `aiwf init` no longer writes it (`Config{}` is the default and an empty marshal becomes a comment-header so later hand-edited yaml blocks parse correctly); `aiwf update` strips it via `StripLegacyAiwfVersion` (mirror of the legacy-actor-strip pattern); doctor's `pin:` row goes away and the `config:` row drops the `(aiwf_version=…)` text. Two new helpers + an opt-in deprecation note on doctor for any pre-G47 yaml the consumer hasn't yet updated. Tests: legacy yamls load fine, the strip is idempotent, fresh init writes neither `actor:` nor `aiwf_version:`, and the doctor advisory fires for legacy yamls but doesn't increment the problem count.

`aiwf init` writes `aiwf_version: <binary-version>` to `aiwf.yaml`; the field is currently *required* by the loader (`internal/config/config.go:215`: `aiwf_version is required`). Nothing maintains the field after init. `aiwf doctor` compares the pinned value against the running binary and reports a "pin skew" row whenever they disagree. After any binary upgrade, the row becomes a chronic nag — the consumer didn't pin intentionally; the value was just whatever was current at first init.

**Concrete reproducer:** consumer init'd at v0.1.1 a year ago, ran `aiwf upgrade` to v0.4.0 today, runs `aiwf doctor`:

```
config:    ok (aiwf_version=v0.1.1)
pin:       pinned v0.1.1, binary newer (v0.4.0) — update pin or roll back binary
```

The user's reasonable reaction is "I never pinned anything, why am I being asked to update a pin?"

**Why the field exists at all** (kernel arc context):

The pin has two implicit purposes that are in tension:

1. *Audit signal* — "this consumer last ran against version X." Wants auto-bump on every update. Cheap to maintain.
2. *Intentional pin* — "this consumer wants to stay on version X." Wants manual-only updates. Doctor's skew row is the load-bearing UX.

The current shape tries to do both: the field is set automatically on init but never bumped, so an unintentional pin from year-old init looks indistinguishable from a deliberate "stay on v0.1.1" choice. Doctor can't tell which it is and nags either way.

**Resolution: remove the field entirely** (YAGNI). The pin's information is available via cheaper channels:

- *"What version am I on?"* → `aiwf version` (the binary self-reports; no yaml lookup needed).
- *"Is there a newer release?"* → `aiwf doctor --check-latest` (queries the module proxy; opt-in).
- *"What was this consumer's last init/upgrade against?"* → reachable via `git log` on `aiwf.yaml` if you really need it.

The field's only kernel-side consumer is doctor's pin row, which becomes vestigial once the field goes.

This is the same shape as the `actor` field removal in I2.5: a field that *was* stored in `aiwf.yaml`, became runtime-derivable from authoritative sources elsewhere (`git config user.email`), and was retired via an auto-strip on `aiwf update`. The legacy-actor-strip step is the migration template here.

**Resolution path for v0.5.0 (proposed):**

1. **Loader becomes tolerant.** `internal/config/config.go` drops the `aiwf_version is required` validation. Existing yamls with the field load fine; new yamls without it load fine; no error.
2. **`aiwf init` stops writing the field.** New initializations produce a yaml without `aiwf_version:`.
3. **`aiwf update` strips the field on refresh.** Same pattern as the legacy actor strip — ledger reports `preserved aiwf.yaml (legacy aiwf_version strip)` when the field is removed.
4. **`aiwf doctor` drops the pin row.** The `binary:` row stays (always shown); `pin:` and `latest:` rows merge into a single optional advisory: `latest:` (opt-in via `--check-latest`).
5. **Update CLAUDE.md / README** to remove pin-related references.

The discoverability lint will flag the missing `aiwf_version` reference if any embedded skill or doc still names it; the discoverability haystack just stops including the field name.

**Severity:** Medium. Doctor noise is a UX nag, not a correctness issue, but it surfaces every doctor run and trains consumers to ignore the row — which is the opposite of what doctor exists to do. Resolution is a small, well-scoped follow-up that mirrors a previously-shipped pattern (legacy actor strip).

---

<a id="g45"></a>
### G45. aiwf-managed git hooks don't compose with consumer-written hooks — **resolved**

Resolved in commit `49e7764` (feat(aiwf): G45 — hook chaining via `.local` siblings + auto-migration). The marker-managed `pre-push` and `pre-commit` hooks now invoke a `<hook-name>.local` sibling (if present and executable) before running aiwf's own work. `aiwf init` / `aiwf update` auto-migrate a pre-existing non-marker hook to `<hook-name>.local`, preserving its content byte-for-byte and its executable bit, then install aiwf's chain-aware hook. New `ActionMigrated` step result. `HookConflict` now signals only the rare `.local`-already-exists collision (refuse to clobber a deliberate `.local`). `aiwf doctor` reports the chain shape per hook: absent, present + executable (`chains to ...`), or present + non-executable (error). Tests cover migration, the load-bearing collision case, the chain runtime semantics (`.local` exits 0 / non-zero / non-executable), and doctor's three states.

`aiwf init` / `aiwf update` install marker-managed hooks at `.git/hooks/pre-push` and `.git/hooks/pre-commit`. When a consumer already has a non-marker hook in place, init refuses to overwrite (correct, by design — see [`internal/initrepo/initrepo.go`](../../internal/initrepo/initrepo.go) `ensurePreHook` / `ensurePreCommitHook`). The user is left with three choices: remove their hook, manually compose it with `aiwf check`, or run `aiwf init --skip-hook` and lose the chokepoint. None of these match the kernel's "framework should add to the consumer's flow, not demand the consumer dismantle their own" stance.

This is the load-bearing collision once the kernel itself dogfoods aiwf (G38): the kernel's existing pre-commit hook (`scripts/git-hooks/pre-commit`, run via `core.hooksPath`) collides with aiwf's marker-managed hook. The same collision happens for any consumer using husky / lefthook / pre-commit.com that has hand-written hooks under `.git/hooks/`.

**Resolution:** Hook chaining. The marker-managed hook learns to invoke a `<hook-name>.local` sibling before running aiwf's own work. Specifically:

1. **Naming.** `.git/hooks/pre-commit.local` and `.git/hooks/pre-push.local`. Git itself ignores the `.local` suffix — only aiwf's hook ever invokes it. No risk of git running the user's hook behind aiwf's back.
2. **Chain order: user-first.** The aiwf hook runs `<hook-name>.local` if present and executable, then (only on exit 0) runs aiwf's own work (`aiwf check --shape-only` for pre-commit, `aiwf check` for pre-push, plus the optional STATUS.md regen for pre-commit). User-first matches the convention in chaining tools (pre-commit.com, etc.) and means the user's iteration loop isn't gated on aiwf's check.
3. **Auto-migration on `aiwf init`.** When init detects an existing non-marker hook, it `mv`s it to `<hook-name>.local`, preserves its executable bit, then installs aiwf's chain-aware hook. The user wakes up with a working composition. Init prints a clear ledger line naming what moved where. The `--skip-hook` flag still bypasses the entire dance for users who manage hooks via husky/lefthook.
4. **Collision guard.** If `<hook-name>.local` already exists when init wants to migrate (consumer has both a non-marker hook *and* a prior `.local`), refuse with a clear error rather than overwrite — the user has clearly engaged with chain plumbing on purpose. This is the one case where init still requires manual resolution.
5. **Non-executable `.local`: fail loud.** If the chain script finds `.local` exists but `! -x`, it fails the commit/push with a clear remediation message (`chmod +x`). A non-executable hook script is almost always a configuration mistake; silent skip would let the user think they have hook coverage when they don't.
6. **`aiwf doctor` reports chain shape.** New rows: `pre-commit hook: ok (aiwf-managed; chains to .git/hooks/pre-commit.local)` when the sibling exists and is executable; `(aiwf-managed; no .local sibling)` when absent; `error (... is not executable — chmod +x to enable)` when present but non-executable. The error case increments doctor's problem count.
7. **`aiwf update` is the redeployment vector.** Existing consumers pick up the chain plumbing automatically when they next run `aiwf update`; their `<hook-name>.local` (if any) is left untouched.

Severity: **Medium**. Not blocking the PoC, but blocks G38 (dogfooding the kernel against itself) cleanly, and blocks any consumer with a pre-existing hook from a friction-free `aiwf init`. Filed as the chokepoint that has to land before the dogfood migration.

---

### G21. Kernel surface is partially undocumented for AI assistants — **resolved (finding-code axis); other axes verified clean**

Resolved across commits `5a7df46` (docs(aiwf): document case-paths and load-error in aiwf-check skill) and `351e694` (feat(aiwf): extend discoverability policy from provenance-* to all codes).

A six-axis audit (verbs, flags, finding codes, trailer keys, body-section names, YAML fields) against the four CLAUDE.md-named documentation channels (`aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, CLAUDE.md / CLAUDE.md, and any markdown under `docs/pocv3/`) found:

- **Verbs (22), flags (40+), body sections (18), YAML fields (20+):** every item documented in at least one channel; zero gaps.
- **Trailer keys (15):** zero gaps. `aiwf-prior-parent` is mentioned in `design-lessons.md` (reachable via `CLAUDE.md`); the rest are in printHelp or `provenance-model.md`.
- **Finding codes (18 active across `check/` and `contractcheck/`):** two genuinely undocumented — `case-paths` and `load-error`. Both inline string literals (not named constants), invisible to the prior `provenance-*`-scoped policy. Added to the `aiwf-check` skill's errors table.

The `PolicyFindingCodesAreDiscoverable` policy in `internal/policies/discoverability.go` was extended in two ways so this gap can't reopen unnoticed:

1. *Code enumeration* expanded from "named `provenance-*` constants" to "every kebab-case finding code anywhere," via a new `loadCheckCodeLiterals` AST walk over `Finding{Code: "..."}` literals across `check/` and `contractcheck/`.
2. *Channel set* expanded from "`aiwf-check` skill + `main.go`" to the full CLAUDE.md set: every embedded skill, `main.go`, both `CLAUDE.md` files, and every markdown under `docs/pocv3/`.

`go test ./internal/policies/...` is the CI-enforced safety net: any new finding code added without a documentation mention fails `TestPolicy_FindingCodesAreDiscoverable` before merge.

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

Resolved in commit `40c3d2d` (build(repo): G25 — adopt core.hooksPath for the tracked pre-commit hook). The policy gate that enforces G21's discoverability rule (and every other policy under `internal/policies/`) lived in `.git/hooks/pre-commit` — installed per clone via `make hooks` (install-by-copy of `scripts/git-hooks/pre-commit`). The model has two failure modes:

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

There is no clean recovery verb. `aiwf cancel <id> --force --reason "..."` looks like the natural backfill, but `Cancel` at `internal/verb/promote.go:107-109` still errors `"already at target"` even under `--force` (the function-doc comment at lines 91-92 makes this explicit: the guard is intentional because there is no diff to write). The only currently-available repair is an empty hand-crafted commit with the right trailers — i.e., the same kind of manual commit that produced the problem.

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

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). New `cmd/aiwf/binary_integration_test.go` builds the cmd binary to a tempfile and subprocesses it; two test cases pin (a) ldflags-stamped Version reaches the verb output (`make install` path) and (b) without ldflags, `aiwf version` and `aiwf doctor`'s `binary:` row report the same value (the seam G27 was filed against). Companion fix in `cmd/aiwf/main.go`: `resolvedVersion`'s no-ldflags fallback now returns `version.Current().Version` directly, byte-coherent with the doctor row. Reverse-validated: restoring the v0.1.0 bug shape (`fmt.Println(Version)` printing the unstamped global) fails the fallback test with the exact "literal sentinel" + "seam mismatch" messages.

The policy text in `CLAUDE.md`'s Testing section ("Test the seam, not just the layer") is the durable rule that should prevent the next instance.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf version` returning `"dev"` despite a working `version.Current()` helper. Root cause: tests covered the new helper in isolation but no test exercised the verb that was supposed to use it. The verb's body kept printing an unrelated package-global (`Version`, the ldflags-stamped value defaulting to `"dev"`); the helper was wired into `aiwf doctor`'s `binary:` row but not into the `version` verb. Two parallel sources of truth for "what version am I" coexisted; the test surface covered only the new one.

The bug was caught by a manual smoke test against the v0.1.0 binary post-publish — exactly the wrong place for it to surface. The pattern generalizes: any time a new helper is added that an existing verb *should* adopt, a verb-level test must assert the verb's output reflects the helper's contract. Without that, a future refactor that introduces a new helper alongside an unrelated existing path repeats the bug.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Test the seam, not just the layer") in the same commit that files this gap. Implementation work to retrofit existing verbs:

1. Add a binary-level integration test (`cmd/aiwf/binary_integration_test.go` or similar): `go build -o $TMP/aiwf ./cmd/aiwf` then run `aiwf version`, `aiwf doctor` as subprocesses, assert their output. This catches the v0.1.0 bug class for every verb whose output depends on `runtime/debug.ReadBuildInfo`, `os.Args[0]`, `os.Executable()`, or `-ldflags`-stamped globals.
2. Audit each existing verb that consumes a shared helper (`version.Current`, `version.Latest`, `entity.SchemaForKind`, etc.) and confirm there is at least one verb-level test asserting the helper is the actual source of truth.

A future `aiwf check`-style policy could detect "exported helper imported by `cmd/aiwf` but no test in `cmd/aiwf` references it" — overkill for the PoC, but the policy framework already exists (G21/G26) and a fourth policy in that family would be cheap.

Severity: Medium. The class of bug is high-impact (shipped correctness regression), and the policy is the durable defense; the implementation work is small.

</details>

---

<a id="g28"></a>
### G28. `version.Latest()` test was implementation-driven, not contract-driven — stale `/@latest` cache went unnoticed — **resolved**

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestLatest_RealProxy` ("version is non-empty") replaced by `TestLatest_RealProxy_ContractTest` which fetches `/@v/list` directly via raw `net/http` (not through `version.Latest`), computes the expected highest semver via a test-side reference implementation, then asserts `version.Latest()` returns that exact value. The reference implementation is deliberately not imported from the version package so a future regression can't be hidden by a matching regression in the helper. New `TestLatest_PrereleaseExcludedFromHighestSelection` pins the pre-release-skipping invariant offline via httptest.

The policy text in `CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf doctor --check-latest` displaying a stale pseudo-version instead of `v0.1.0`. Root cause: `version.Latest()` queried the proxy's `/@latest` endpoint and unit tests served whatever JSON the implementation expected. The real proxy behavior — that `/@latest` and `/@v/list` are cached independently, and `/@latest` can serve a pre-tag pseudo-version answer for hours after the first tag lands — was not modeled. The Go toolchain's own resolver uses `/@v/list` first for exactly this reason; we re-discovered the lesson by shipping the wrong endpoint and noticing in v0.1.0 verification.

The existing real-proxy integration test (`TestLatest_RealProxy`) queries `gopkg.in/yaml.v3` and only asserts the version is non-empty. It would have passed with either implementation choice (`/@latest` returning yaml.v3's tag happens to work because nobody queried that module's `/@latest` before tags existed). The test was *cooperative* — it tested the parsing round-trip, not the resolution semantics.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") in the same commit that files this gap. The Latest() resolution itself was fixed in v0.1.1 (commit `32672cd`); G28's residual work is the *test* that pins the contract:

1. Tighten `TestLatest_RealProxy` to derive the expected version through an **independent** code path. Concretely: the test fetches `https://proxy.golang.org/<known-tagged-module>/@v/list` directly (without going through `version.Latest`), parses the response, computes the highest semver triple, and asserts `version.Latest()` returns that exact value. Today's "version is non-empty" assertion is replaced by "version matches the independently-derived expected value."
2. Add a multi-tag fixture test using the existing httptest seam to pin the highest-of-N selection logic without network: serve a `/@v/list` body with three or four tags including a pre-release, assert the highest non-pre-release wins.

Severity: Medium. Same class as G27 — the implementation has been fixed; the policy + the contract test are what stop the next instance.

</details>

---

<a id="g29"></a>
### G29. Pseudo-version regex was example-driven, not spec-driven — initial test set missed two of three forms plus `+dirty` — **resolved**

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestParse`, `TestProxyBase`, and `pseudoVersionRE`'s doc comment now cite the upstream specs (`go.dev/ref/mod#pseudo-versions`, `semver.org`, `go.dev/ref/mod#environment-variables`); `TestParse` cases now cover all three pseudo-version forms explicitly (was: form 1 + form 3 only) plus the `+dirty` stamping case for both base shapes. The citations make spec-drift detectable: a future Go-toolchain change to pseudo-version grammar will be flagged by anyone reading the spec, rather than missed because tests were example-driven.

The policy text in `CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The first pass of `version.isTagged` had a `pseudoVersionRE` that only matched the basic `v0.0.0-DATE-SHA` shape. The Go module spec defines three pseudo-version forms (basic, post-tag `vX.Y.(Z+1)-0.DATE-SHA`, pre-release-base `vX.Y.Z-pre.0.DATE-SHA`) and Go's VCS stamping adds the `+dirty` suffix on working-tree builds with uncommitted changes. The regex caught only the first form; the other three were missed.

The bug was caught mid-implementation by a smoke test (the working-tree build of aiwf reported `"v0.0.0-...-...+dirty (tagged)"`), not by the unit-test pass that immediately preceded it. Root cause: test cases were sourced from "the example I had in mind" rather than from the spec. A spec-sourced enumeration would have listed all four shapes from `https://go.dev/ref/mod#pseudo-versions` plus the VCS-stamping behavior on first writing.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") in the same commit that files this gap. The implementation already covers the cases (regex updated mid-step-2 to `[-.]\d{14}-[0-9a-f]{12}$` and `+dirty` checked separately); G29's residual work is small:

1. Add a `// per https://go.dev/ref/mod#pseudo-versions` comment above the test data in `version_test.go` so the spec-sourcing is visible to future readers.
2. Audit other test sets that enumerate upstream-defined input spaces (frontmatter shapes against YAML 1.2; commit-trailer shapes against `git interpret-trailers`; semver against the semver.org grammar) for analogous unsourced enumerations, and either add the citation or document the omission.

Severity: Low. Bug already resolved; the policy + the citation are the durable defense. The audit pass is one read-through, not a refactor.

</details>

---

<a id="g30"></a>
### G30. `git log --grep` false-positives leak prose-mention commits into Recent activity / `aiwf history` — **resolved**

`aiwf status` (Recent activity table) and `aiwf history <id>` both pre-filter `git log` with `--grep "^aiwf-verb: "` (or the anchored entity variant). The grep matches any line in the commit message that starts with the literal string — including **wrapped prose paragraphs** in hand-authored commit bodies that quote trailer keys. Real example from this repo: commit `18a00e6` ("docs(aiwf): I2.5 + I3 planning sweep") has the wrapped prose

```
…fold the audit-trail manual-commit gap (no
aiwf-verb: trailers) into I2.5 as steps 5b…
```

The second line happens to start with `aiwf-verb:` because of the line-wrap. The grep matches; the record lands in the candidate set; the parsed-trailer columns (`%(trailers:key=aiwf-verb,…)`) correctly find no structured trailer (Git's trailer parser has stricter rules than the naïve grep). Result: a row in the output table with the expected date and subject but **empty Actor and Verb columns** — visually noise, semantically wrong (the framework's "trailered commit" set was contaminated with prose mentions).

Caught while auditing this repo's `STATUS.md` after v0.2.1 shipped: the kernel repo's "Recent activity" had 5 false-positive rows, every one a docs commit whose body referenced trailer keys in prose.

**Resolution path:**

1. *Post-filter on parsed trailers, not just grep.* Both `readRecentActivity` (`status_cmd.go`) and `readHistory` (`admin_cmd.go`) already extract trailer columns via `%(trailers:key=…,valueonly=true,…)`; the fix is to discard records where the trailer column is empty (Git's trailer parser found no actual trailer for the key the caller cares about). The grep stays as an I/O-narrowing pre-filter; correctness is gated on the parsed columns. Two-line change per caller.
2. *Pin the regression.* Add a fixture commit in tests whose body wraps a sentence such that a line starts with `aiwf-verb:`, assert it does **not** appear in `aiwf status` recent activity or in `aiwf history`.
3. *Audit other `--grep` callers* (`provenance.go`, `provenance_check.go`, `scopes.go`, `show_scopes.go`, `admin_cmd.go` `loadAuthorizedScopes`). Each of those parses the trailer columns inside its loop and acts only on parsed-trailer presence, so a prose-line false-positive produces an empty trailer set that no rule branches on — structurally safe. Confirmed by inspection; documented in the fix commit so future readers don't re-litigate.

Severity: **Medium**. Doesn't affect correctness in any of the standing rules (provenance findings, scope FSM) — those iterate parsed trailers and ignore empty records — but corrupts the user-facing read views (`aiwf status` Recent activity, `aiwf history`) that are the daily surface.

---

<a id="g31"></a>
### G31. Squash-merge from the GitHub UI defeats the trailer-survival contract — **resolved**

When a PR is squash-merged via GitHub's UI (the default merge strategy for many repos), the resulting commit on the integration branch carries a synthesized message — typically the PR title plus the body, sometimes a list of squashed commit subjects. **Trailers from individual commits are not preserved**. A feature branch with five well-formed `aiwf <verb>` commits squash-merged via the UI lands one trailer-less commit on `main`; every entity transition from those five commits is invisible to `aiwf history <id>` against the merged tree (the only commits that carried `aiwf-verb:` trailers no longer reach HEAD via first-parent).

This breaks the framework's central correctness story ("git log is the audit log") on the most common GitHub merge strategy. Surfaced during the G24 follow-up audit (after issue #5 + G30 closed): merge surfaces had not been re-walked end-to-end since I2.5, and squash-merge had no detection.

**Resolution path:**

A subcode under the existing `provenance-untrailered-entity-commit` finding flags the case explicitly so the operator gets a more specific hint:

- *Detection.* `RunUntrailedAudit` matches the commit's subject against the GitHub squash-merge regex `\s\(#\d+\)$` (PR title followed by ` (#NNN)`). When a flagged untrailered commit fits the pattern, the finding is emitted with subcode `squash-merge`. The default `(provenance-untrailered-entity-commit)` finding still applies; the subcode just specializes the hint.
- *Hint.* The hint table entry for the subcode names the merge-strategy gotcha and the recovery path: switch the repo to rebase-merge or `--no-ff` merge for branches that touch entity files, or run `aiwf <verb> <id> --audit-only --reason "..."` per entity touched.
- *Skill text.* `aiwf-check` SKILL.md gains a row for the subcode pointing at the same recovery path.
- *Pinned by* `TestRunUntrailedAudit_SquashMergeSubcode`: a fixture commit with subject `… (#42)` touching an entity file emits the finding with subcode `squash-merge`; subjects without that suffix produce the bare code.

What this fix does NOT do: recover trailers from a squash-merged commit's source SHAs (would require walking GitHub's `refs/pull/<N>/head` references — out of scope for the kernel). The detection surfaces the gap; the audit-only recovery path is the operator's lever to backfill what the squash-merge dropped.

**Limitations:**

- Detection is opportunistic: it fires only while the squash commit is in the audit's `@{u}..HEAD` (or `--since`) range. After the operator pushes/pulls, the squash commit becomes `@{u}` and is no longer scanned. The companion README entry under "Known limitations" frames squash-merge as something the operator should re-audit on the integration-branch *before* pushing further.
- The regex matches the GitHub default. Custom squash-commit-message templates that drop the `(#NNN)` suffix won't trigger the subcode (the bare warning still fires; only the hint specializes).

Severity: **High**. Real audit-trail hole on the dominant merge strategy; the framework's central promise depended on a pattern most consumers don't follow by default. Fixed by detection + hint + skill update, plus a known-limitation note in the README.

---

<a id="g32"></a>
### G32. Merge commits silently bypass the untrailered-entity audit — **resolved**

`readUntrailedCommits` ran `git log --name-only`, which by default shows **no file list for merge commits** (true merge commits show diff content only with `-m` or `--cc`). A merge commit that absorbs entity-file changes from a feature branch produces an empty `Paths` slice, and `RunUntrailedAudit` skips it.

Concrete: trunk has G-001 at `open`. A feature branch makes a manual commit changing G-001 to `wontfix`. Operator merges feature → trunk via `git merge --no-ff feature`. The merge commit's message lacks aiwf trailers; its `--name-only` is empty. The audit pass on trunk says nothing — even though the audit-trail hole is real (no `aiwf-verb:` trailer ever recorded G-001's transition).

This was salvageable on the *feature* branch (the original untrailered commit was flagged before merge), but only if the operator ran `aiwf check` between the manual commit and the merge. Feature → merge → push without an interim check left the warning silent on the merge commit itself. A merge commit that itself made changes (conflict resolution touching entity files) was also silent.

**Resolution path:**

`readUntrailedCommits` now invokes `git log -m --first-parent`. Combined effects:

- *`--first-parent`*: walks first-parent ancestry of the integration branch only. Feature-branch commits are NOT shown (correctly — they're the feature branch's own warning scope). Merge commits ARE shown.
- *`-m`*: causes merge commits to show diffs against their first parent — i.e., the changes the merge introduced into the integration branch. Entity-file paths flow into the audit pass.

Together: a merge that brings in feature-branch entity-file changes surfaces those file paths. Per-(commit, entity) findings (post-G30) fire on each touched entity. Audit-only on the integration branch clears them via the same per-entity suppression path.

**Limitations:**

- Octopus merges (3+ parents) are rare and produce one record per non-first-parent diff under `-m`; the existing per-entity dedupe inside the loop handles the common case (an octopus that brings the same entity from multiple branches collapses to one finding per entity at the loop level).
- A merge commit that introduces NO new entity-file changes (the integration branch already had everything) produces an empty path list and stays silent — correct behavior.

Pinned by `TestRunUntrailedAudit_MergeCommitSurface` (in `cmd/aiwf/show_scopes_unit_test.go` next to the existing `--since` tests): a fixture with a merge commit whose second parent introduced an entity-file change is flagged.

Severity: **Medium**. Doesn't compromise correctness — `aiwf check` still fires on the original commit on the feature branch — but loses signal at the integration-branch boundary, which is exactly when the operator's last chance to repair lives.

---

<a id="g33"></a>
### G33. `aiwf doctor --self-check` doesn't exercise the audit-only recovery path — **resolved**

The G24 recovery story has three load-bearing pieces (manual commit detection, `--audit-only` empty-diff repair, lock-contention diagnostics in `Apply`). Self-check covered init / add / promote / cancel / render / etc., but did not drive the recovery loop end-to-end. A regression in the suppression rule (issue #5's all-or-nothing was such a regression) wouldn't be caught by CI's self-check stage; it'd ship until a user noticed.

**Resolution path:**

New self-check step (after the existing `cancel` step) that:

1. Synthesizes a manual untrailered commit that touches an entity file.
2. Runs `aiwf check`; asserts a `provenance-untrailered-entity-commit` finding with the expected entity-id is present.
3. Runs `aiwf cancel <id> --audit-only --reason "self-check"`.
4. Runs `aiwf check`; asserts the previously-emitted finding for that entity is gone.

The step also exercises the per-entity suppression that issue #5 fixed — a regression in that path would fail the assertion at step 4.

Severity: **Medium**. CI safety-net pattern, same shape as G9's "self-check covers every verb" rule. Small fix; pinned the recovery path that until now was only covered by unit tests in `internal/check/provenance_test.go`.

---

<a id="g34"></a>
### G34. Mutating verbs sweep pre-staged unrelated changes into their commit — **resolved**

Resolved in commit `(this commit)` (fix(aiwf): G34 — isolate verb commits from user's pre-staged work via stash). `verb.Apply` (and `aiwf render roadmap`, the only other mutating call site outside Apply) now check `gitops.StagedPaths` before running. Two halves:

1. **Conflict guard.** When a path the verb is about to write is *also* pre-staged by the user, refuse before any disk mutation — the user's staged content and the verb's computed content can't both land. The error names every conflicting path and points at `git restore --staged` / `git stash`.
2. **Stash isolation.** When the user has unrelated staged work, push it onto the stash (`git stash push --staged`), run the verb's normal `git commit -m <msg>` flow (so the pre-commit STATUS.md hook composes correctly — its `git add STATUS.md` lands in the verb's commit as designed), then pop the stash so the user's WIP is back in the index for their next commit. Pop also runs on rollback paths so a partial failure doesn't strand the stash.

Initial attempt scoped the commit via `git commit -- <verbPaths>` (--only semantics) but that compose poorly with hooks that auto-`git add` extra files: git captures the hook's addition in HEAD but resets the post-commit index to only the explicitly-named paths, leaving a phantom staged-deletion behind. The stash approach gives the verb a clean index to commit against, hooks behave normally, and the user's stage round-trips intact.

New `gitops.StashStaged` / `gitops.StashPop` / `gitops.StagedPaths` primitives (StagedPaths uses `-z` to handle paths with spaces/newlines safely). Tests: `TestApply_PreservesUnrelatedStagedChanges`, `TestApply_RefusesConflictingPreStagedPath`, `TestApply_AllowEmptyPreservesUnrelatedStaged`, `TestApply_AllowEmptyOnCleanIndex` cover the verb seam; `TestStashStaged_PushPopRoundTrip` and `TestStagedPaths` pin the gitops primitives. Manual smoke confirms the user's reproducer (`git add unrelated.go && aiwf add gap …`) now lands a single-path gap commit (plus hook-regenerated STATUS.md) with `unrelated.go` still staged for the user.

---

<a id="g35"></a>
### G35. HTML site only generates pages for epic and milestone — gap/ADR/decision/contract links 404 — **resolved**

Resolved in commit `(this commit)` (fix(aiwf): G35/G36 — render gap/ADR/decision/contract pages with HTML markdown bodies). New shared `entity.tmpl` covers the four kinds without specialized rendering; `htmlrender.Render` iterates over `KindADR`/`KindGap`/`KindDecision`/`KindContract` after the existing epic/milestone loops, calling a new `renderEntity` that pulls per-page data from a new `PageDataResolver.EntityData(id)` method. Default resolver returns frontmatter + sidebar (no body — that's a cmd-side concern); cmd-side `renderResolver.EntityData` reads the body from disk, parses sections via the new `entity.ParseBodySectionsOrdered`, and surfaces forward+reverse linked entities and recent history. Tests: per-kind structural assertions on `G-001.html`, `ADR-0001.html`, `D-001.html`, `C-001.html` (kicker carries the kind label, `<h1>` carries the title, sidebar link back to index present). Smoke: rendered a synthesized kernel-style tree and walked every kind's page in a browser. Pairs with G36 — fixing both in the same commit means the new gap/ADR/decision/contract pages don't ship with the same rendered-as-raw-text defect.

---

<a id="g36"></a>
### G36. Entity body markdown rendered as escaped raw text in HTML — **resolved**

Resolved in commit `(this commit)` (fix(aiwf): G35/G36 — render gap/ADR/decision/contract pages with HTML markdown bodies). New `markdownToHTML` helper in `internal/htmlrender/markdown.go` runs each body section through `goldmark` and returns `template.HTML` so the rendered HTML isn't double-escaped. Goldmark configured with Tables/Strikethrough/Linkify/TaskList extensions but raw-HTML pass-through OFF — bodies are committed to git but the static-site step refuses to upgrade that trust into "browser-executable" (XSS guard pinned by `TestMarkdownToHTML_RawHTMLEscaped`). `epic.tmpl`, `milestone.tmpl`, and the new `entity.tmpl` route every body section through the helper. New dep `github.com/yuin/goldmark v1.8.2` — pure-Go CommonMark renderer, no CGO, single transitive tree (justified per `CLAUDE.md` Dependencies). Tests: `TestMarkdownToHTML_RenderingShapes` covers paragraphs, fenced code, inline code, ordered/unordered lists, links, subheadings, emphasis; `TestRender_BodyMarkdownRendersAsHTML` is the verb-seam test that drives a real page render and asserts `<ul>/<li>`, `<code>aiwf check</code>`, link `href`, `<pre>` all appear and that no raw-markdown source leaks through. Smoke: rendered a fixture body with lists, links, fenced code blocks — output is correct HTML.

---

<a id="g37"></a>
### G37. Cross-branch id collisions split the audit trail; allocator is local-tree only — **resolved**

`entity.AllocateID` (`internal/entity/allocate.go:43`) walks the caller's working tree and picks `max+1`. The doc comment at lines 34-37 names this as deliberate ("cross-branch coordination is by design out of scope; collisions are caught by the ids-unique check and resolved with `aiwf reallocate`"). The design *predicted* the collision class but the resolution path was sized for a single-entity oops, not for "two parallel sessions both did real work under the same id."

**Concrete reproducer (flowtime-vnext):** main allocated `G-035 = "Promote InvariantAnalyzer warnings to CI gate"` (commit `01960ab`). Branch `milestone/M-066-edge-flow-authority-decision` independently allocated `G-035 = "Pre-aiwf v1 framework docs survived migration..."` (commit `95e4b18`). Both branches did real work under that id. Merge attempt surfaced the collision via a side-effect path (STATUS.md regen, not the entity files themselves — they have different slugs and would have merged silently into a tree with two G-035s, only caught by pre-push `aiwf check`).

**Why this is severe:**

1. *Detection is too late to be cheap.* By the time the merge happens, both branches have committed real work under the same id and both have been discussed with humans / AI under that name. Retraction cost scales with how long the branches diverged before noticing.
2. *Reallocate splits the audit trail.* Whichever branch loses, its pre-rename commits forever reference an id that, post-reallocate, means something else. `git log --grep "aiwf-entity: G-035"` returns commits from both branches under one id that now means two entities. The framework's "git log is the audit log" promise has an unsignalled hole in the multi-branch case.
3. *The path-conflict surface is a symptom, not the bug.* The two G-035 files have different slugs — git doesn't conflict on the entity files themselves. Whatever did conflict (in this case STATUS.md regen) just happened to be where the deeper id-collision became visible. Without it, the merge would silently produce a tree with two G-035s.

**Resolution:** Specified in [`design/id-allocation.md`](design/id-allocation.md) and shipped in two layers:

1. **Layer (a) — trunk-aware allocator + cross-tree `ids-unique`** (commit `271f514`). The allocator reads the working tree and the configured trunk ref (default `refs/remotes/origin/main`, overridable via `aiwf.yaml: allocate.trunk`). On a missing ref with no remotes the read is silently skipped (sandbox repos); on a missing ref *with* remotes the verb fails with a clear message — no silent fallback. `ids-unique` reads the trunk ref too, so a cross-tree collision surfaces as a normal pre-push finding (subcode `trunk-collision`). No `--against` flag, no merge simulation.
2. **Layer (b) — `prior_ids` frontmatter + reallocate trunk-ancestry tiebreaker + history chain walk** (commits `b9d73d8`, `c5a98c1`, plus integration scenario `a6e8067`). `aiwf reallocate` appends the old id to a `prior_ids: []` frontmatter list on the renumbered entity. When two entities collide on an id, the verb resolves the renumber target via `git merge-base --is-ancestor` against the trunk ref — the side already in trunk keeps the id; if ancestry can't decide, the verb refuses with a clear diagnostic and asks for a path. `aiwf history` resolves any id (current or prior) through `tree.Tree.ResolveByCurrentOrPriorID`, expands the queried id through the entity's `PriorIDs` chain, and runs one `git log` grep over `aiwf-entity:` and `aiwf-prior-entity:` for the union — pre-rename, rename, and post-rename commits arrive as one chronological timeline. The doc was reconciled to match the shipped reality (both surfaces ship; trailer is the git-log-readable source, frontmatter is the tree-readable source) in commit `685f288`.

The design deliberately omits origin-pinning, a counter-branch push-CAS allocator, surrogate identities, and an all-refs walk — each was considered and judged more code than this gap requires. The migration verb (`aiwf migrate-lineage`) for backfilling `prior_ids` from `aiwf-prior-entity:` trailers in pre-G37 reallocate history stays unbuilt-by-design: no consumer currently has the kind of legacy reallocate history that would benefit, and the verb earns its own follow-up if one surfaces.

Severity: **High**. Audit-trail integrity is one of the framework's central correctness stories ("git log is the audit log"); the multi-branch case had an unsignalled hole. The reproducer was real and recent, not theoretical.

---

<a id="g38"></a>
### G38. The kernel repo does not dogfood aiwf — feasibility and fit need investigation — **open**

The framework targets "humans + AI assistants tracking planning state via aiwf entities under git." The kernel repo itself does not. There is no `aiwf.yaml`, no `work/` directory, no `.git/hooks/pre-push` running `aiwf check`, and no `aiwf-verb: / aiwf-entity:` trailers on any commit on this branch. The PoC's own roadmap, gaps, and design notes live as plain markdown under `docs/pocv3/` rather than as aiwf entities.

That choice was implicit, not deliberate. It worked because the test suite covers the kernel's correctness directly, the policy lint pass enforces commit-message and structural rules, and the work plan in `poc-plan.md` is small enough to track by hand. But it leaves three real costs unmeasured:

1. *We don't ergonomics-test our own product.* Every aiwf consumer hits the verb surface, the skill content, and the operator-facing error messages dozens of times per day. We hit them only through tests and ad-hoc fixtures. Bugs in flow ("the gap title produces an awful slug," "the help text doesn't mention the new flag," "the pre-push error message is opaque") don't surface until someone else uses the framework.
2. *We don't catch bugs that only fire on real consumer state.* G37's trunk-aware path is a recent example — the PoC's own commits do not exercise it, so a regression that only fires on populated repos with multi-clone history would slip past every check we currently have.
3. *We force ourselves to maintain a parallel system.* `gaps.md`, `poc-plan.md`, and the design docs together do what aiwf does for entities (track status, references, history, lifecycle) — by hand, in markdown. Some redundancy is fine; some of it is friction we'd rather not pay.

**The investigation this gap kicks off, before any wiring lands:**

1. *Is dogfooding feasible at all on this branch?* The PoC branch deliberately is not planned to merge to main, commits flow directly without PR ceremony, and the work plan is in markdown. Could the existing markdown roadmap migrate into entities (one epic per session, milestones per session step, gaps as gaps, design decisions as decisions, ADRs as ADRs)? Or do some artifacts not fit any of the six kinds, in which case the "convert to entities" story falls apart?

2. *What conflicts with the kernel's own development workflow?*
   - The pre-push hook would run `aiwf check` on every push. The PoC does pushes that are deliberately rough (WIP commits, design-in-flight, broken-test moments documented later in commit messages). Would `aiwf check` block the iteration loop more than it helps?
   - Mutating verbs require an `aiwf-actor` trailer. Does that compose cleanly with the existing manual commit flow, or would every commit need to go through a verb? (We don't add entities every commit — most commits are pure code.)
   - The pre-commit hook here already runs the policy lint. Would adding `aiwf check` to pre-push be additive or duplicative?
   - `STATUS.md` is currently regenerated by the pre-commit hook when `aiwf` is on PATH. If `work/` becomes populated, every commit would also touch STATUS.md — does that conflict with our existing commit hygiene (e.g., G34's pre-staged-unrelated-changes guard)?

3. *What about commits that pre-date the migration?* The kernel branch has hundreds of commits without any `aiwf-*` trailers. The framework's untrailered-entity audit (per G24) would either need to be run with `--since` pointing at the migration commit, or the migration would need to deliberately exempt prior history. Decide which.

4. *What about the fact that the design docs reference each other in prose?* `design-decisions.md`, `id-allocation.md`, `provenance-model.md` cross-link each other as relative markdown paths. If some of those become aiwf entities (decisions, ADRs), the prose-reference style needs to coexist with the framework's reference fields. Either (a) the docs stay as docs and point at entity ids by reference, or (b) the docs become entities and the design narrative is decomposed into bodies + frontmatter — which may lose the long-form prose that makes them useful to read sequentially.

5. *Cost-benefit: is there enough churn for dogfooding to pay?* Sessions 1–4 of the PoC are largely complete. The remaining churn lives in gaps and design refinements. If we dogfood now, we mostly migrate stable artifacts and accept ongoing-maintenance cost without much further benefit. If we don't, we may never bootstrap a real consumer-shaped feedback loop on our own product before the PoC closes.

**Possible outcomes the investigation should produce, in order of escalation:**

- *Don't dogfood.* Document the decision and the reasons (kernel branch is short-lived, conversion cost outweighs benefits, parallel system is fine). Closes the gap as "deferred with rationale."
- *Dogfood thinly.* Run `aiwf init`, install the pre-push hook, leave `work/` empty. The kernel doesn't track its own work as entities, but the verb surface and the hook fire on every push, exercising the framework against a "real consumer that just doesn't have any entities yet." Catches some classes of regression (`aiwf check` itself misbehaving) without forcing a content migration.
- *Dogfood partially.* Convert one slice — say, the open gaps — into `G-NNN` entities under `work/gaps/`. Keep the markdown `gaps.md` as the human-readable index, possibly auto-generated from the entity tree. Mid-cost, mid-value.
- *Dogfood fully.* Convert all in-flight roadmap, decisions, and gaps to entities. Reorganize PoC sessions as epics + milestones. Highest cost; highest signal. Probably only worth it if the PoC is going to live longer than originally planned.

The gap is *open* until the investigation produces a written decision (in this gap entry, in `design-decisions.md`, or as an ADR — itself a decision the investigation has to make about where the decision belongs).

Severity: **Medium**. Not blocking PoC completion, but a real source of "we don't know what we don't know" about our own framework's ergonomics and edge cases.

---

### G42. Pre-commit hook coupled enforcement and convenience — `status_md.auto_update: false` removed the tree-discipline gate too — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G42 — decouple pre-commit hook responsibilities). G41 wired the tree-discipline gate into the pre-commit hook, but the hook installer was still gated by `aiwf.yaml: status_md.auto_update` — a flag whose original purpose was to opt out of *STATUS.md regeneration*, not enforcement. Flipping the flag removed the entire hook, which now meant losing the gate too. Pre-push still caught stray files, but the in-loop early-warning that motivated G41 disappeared.

The fix decouples the two responsibilities at the script level:

- The pre-commit hook now installs unconditionally when aiwf is adopted in the repo (the `SkipHooks` opt-out at init time remains the single escape hatch for "I want no aiwf hooks at all").
- `preCommitHookScript(execPath, regenStatus)` takes a bool for the regen step. When false, the script body contains only the tree-discipline gate; when true, it includes the gate followed by the existing tolerant STATUS.md regen.
- The `ensurePreCommitHook` action set is now {`Created`, `Updated`, `Skipped` (alien hook)}; `Removed` no longer occurs through this path. `aiwf doctor`'s pre-commit reporting is updated accordingly: missing-hook is always drift, present-with-mismatching-regen is drift, and the new "ok, gate-only" line marks the desired-and-actual-agree state under `auto_update: false`.
- `extractPreCommitExecPath` now handles the `if ! 'path' …` negation form introduced for the gate; without this, the doctor would have reported a malformed hook for the gate-only mode.

Tests cover both modes end-to-end:

- `TestEnsurePreCommitHook_RegenOff_FreshInstall` and `_RefreshDropsRegen` pin the new install/refresh contracts.
- `TestEnsurePreCommitHook_RegenOff_AlienHookPreserved` proves the always-install change does not weaken alien-hook preservation.
- `TestRefreshArtifacts_FlipFlagDropsRegenKeepsGate` and `TestRun_UpdateDropsRegenKeepsGateOnOptOut` exercise the canonical opt-out flow at the package and verb levels.
- `TestPreCommitHookScript_RegenStatus_Decoupling` pins the script-template invariant: gate always present, regen only when `regenStatus=true`.
- The doctor self-check repo's update round-trip is rewritten ("keeps gate, drops regen" + "reinstates regen") so a regression that re-couples the responsibilities surfaces in the self-check, not in the field.

Severity: **High**. The coupling silently negated G41's enforcement guarantee for any consumer who had touched the unrelated STATUS.md flag. Caught in review immediately after G41 shipped.

---

### G41. Tree-discipline ran only at pre-push — LLM-loop signal lands too late — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G41 — pre-commit gate + `aiwf check --shape-only`). G40 shipped the tree-discipline rule wired into the full `aiwf check` pipeline at pre-push only. That guarantees the bad state never *pushes*, but it does not give the LLM an in-loop signal — by the time pre-push fires, the stray commit has already landed locally, possibly been amended onto, or been bypassed via `git push --no-verify`. The user pushed back on two points:

1. **Agent-agnosticism.** A marker-managed CLAUDE.md fragment (the original early-warning proposal) ties aiwf to Claude Code; Cursor uses `.cursor/rules`, AGENTS.md is emerging, etc. Git hooks fire for any client that uses git — which is all of them. The hook is the agent-agnostic surface.
2. **Pre-commit beats pre-push for this rule.** Stray-file detection is fast and exact; there is no legitimate "WIP" state where a stray exists. Moving the check earlier costs nothing in correctness and gains the in-loop feedback signal. The kernel's existing "marker-managed framework artifacts" principle already covers git hooks, so no new surface is created.

The fix has three pieces, all in this commit:

1. **`aiwf check --shape-only` flag.** Runs only the tree-discipline rule (no trunk read, no provenance walk, no contract validation), reads `aiwf.yaml: tree.{allow_paths,strict}` the same way the full check does. Cheap enough to fire on every commit. Exit codes match the standard contract: 0 ok, 1 findings (only when tree.strict promotes the warning to error), 3 internal.
2. **Pre-commit hook gains the gate.** The aiwf-managed pre-commit hook now invokes `aiwf check --shape-only` *before* the existing STATUS.md regen step. The shape check is non-tolerant — non-zero exit blocks the commit (only fires when strict). The status step remains tolerant per the existing design.
3. **Skill + design doc updates.** `aiwf-check` SKILL documents the `--shape-only` flag and the pre-commit/pre-push split as a two-row table; `tree-discipline.md` records the chokepoint design rationale and explicitly rejects the marker-CLAUDE.md alternative with the agent-agnosticism reasoning.

Followup decoupling closed in G42 — see below. Severity: **High**.

---

### G40. `work/` is mechanically unprotected — `aiwf check` silently ignores stray files — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G40 — tree-discipline check + tree.allow_paths config + skill rule). The tree loader at `internal/tree/tree.go` walks `work/*` subdirectories and registers everything `entity.PathKind` recognizes; until this fix, files at *any* other path were silently skipped — no finding, no warning, no log line. An LLM-written `work/scratch.md`, an accidental `work/epics/E-01-foo/notes.md`, or a leftover `work/old-stuff/` directory was invisible to `aiwf check` and therefore invisible to the pre-push hook. The chokepoint that was supposed to make tree-shape guarantees real had a blind spot the size of the whole `work/` tree.

The fix has three pieces, all in this commit:

1. **Mechanical layer** — `tree.Tree` gains a `Strays []string` field populated during the walk for `work/*` subtrees (docs/adr/ stays permissive). New `check.TreeDiscipline(t, allow, strict) []Finding` filters strays through (a) auto-exempt for files inside any recognized contract directory, (b) user allow-list via `aiwf.yaml: tree.allow_paths` glob list, and emits `unexpected-tree-file` findings for the rest. Severity is **warning** by default; **error** when `aiwf.yaml: tree.strict: true` (the pre-push hook then blocks the push). Wired into `runCheck` after the standard rule chain so render/status callers don't get tree-discipline noise on every read.
2. **AI-discoverable layer** — `aiwf-add` SKILL gains a "Tree discipline" subsection naming the rule and the failure mode; `aiwf-check` SKILL gains the new finding code in its warnings table. **No new skill** — folded into existing skills to avoid skill sprawl per the user's directive.
3. **Doctrine layer** — new `docs/pocv3/design/tree-discipline.md` records the canonical path shapes, the "verbs own tree shape; body prose can be edited directly" rule, the `aiwf.yaml: tree.*` configuration surface, and the explicit decision *not* to add an `aiwf edit` verb (YAGNI; revisit if untrailered-body-edit audit warnings become noisy).

What's allowed without a verb: **body-prose edits to existing entity files**, with the resulting untrailered commit reconciled by `aiwf adopt` (the existing G24 surface). What's not allowed: any *new* file under `work/` outside the six recognized path shapes (epic/milestone/gap/decision/contract/ADR), and any change to an existing entity file's frontmatter without going through the appropriate verb. The mechanical check is the guarantee per the kernel principle "framework correctness must not depend on the LLM's behavior"; the skill is the convenient version, not the load-bearing one.

The check enforces this in the consumer repo. The kernel's own `CLAUDE.md` is *not* aiwf's responsibility — aiwf ships the embedded skill (gitignored, refreshed on `aiwf update`) and the check; the consumer's hand-written `CLAUDE.md` is theirs alone. This matches the existing kernel principle "marker-managed framework artifacts" — aiwf does not write to consumer-authored files.

The discoverability test policies (G21 / G26 lineage) caught both the new finding code and the new yaml field at first run, exactly as designed — the implementation could not land without the doctrine + skill updates. That's the policy framework working: if you add a kernel surface and forget to make it discoverable, the lint refuses to pass.

Severity: **High**. The blind spot was load-bearing (the pre-push hook is the chokepoint, and the chokepoint was leaking). No reported wedge, but the user discovered it via a real LLM-mistake under `work/` in a consumer repo before any kernel guarantee fired.

---

### G39. `aiwf upgrade` mis-parses `go env` output when GOBIN is unset — **resolved**

Resolved in commit `(this commit)` (fix(aiwf): G39 — upgrade flow's go env parser fails when GOBIN is unset). The post-install lookup in `goBinDir` now queries `go env GOBIN` and `go env GOPATH` in two separate calls instead of one combined call. The combined call returns one line per name, and an unset GOBIN renders as a leading blank line — `strings.TrimSpace` was eating that blank, leaving a 1-element slice that tripped the `len(lines) < 2` guard. Anyone with stock Go install (no GOBIN exported, GOPATH at default) hit a non-zero exit immediately after `go install` succeeded, with the operator-facing message `"unexpected `go env` output: \"\n/home/.../go\n\""` and a generic "run aiwf update manually" hint. The fix removes the multi-line parser entirely; each call returns at most one value, so there is no shape to mis-parse.

Companion UX upgrade: when locating the new binary still fails for any reason, `runUpgrade` now prints a concrete fallback path derived from `$GOBIN`, `$GOPATH/bin`, or `$HOME/go/bin` so the user can recover with one command (`<path> update --root <root>`) instead of guessing where `go install` writes.

Test coverage added in the same commit:

- `TestGoBinDir_Matrix` — table test driving `goBinDir` through the shim across the four GOBIN/GOPATH shape combinations (gobin set, gobin empty + gopath set, both set, both empty). The "gobin empty, gopath set" row is the case this gap was filed for.
- `TestRunUpgrade_FullFlow_GOBINUnset` — verb-level seam test mirroring the pre-existing `TestRunUpgrade_FullFlow_NoReexec` but with `AIWF_TEST_GOBIN=""`, asserting the resolution falls through to GOPATH/bin and that `env GOPATH` is queried after `env GOBIN` returns empty.
- `TestInstallLocationHint` — covers the env-var precedence of the new fallback hint helper.

The pre-existing test seam parameterized the shim's `env` arm with hard-coded non-empty paths, so the empty-GOBIN shape was never exercised. This is a recurrence of G27's pattern (helper covered in isolation, integration shape not covered) — `CLAUDE.md`'s "Test the seam, not just the layer" rule predates this gap and explicitly calls out the pattern. The lesson is that "drive the helper through the shim" is necessary but not sufficient: the shim's input space must enumerate the upstream tool's real output shapes (per G29's spec-sourced-inputs rule applied to runtime tools, not just data formats).

Severity: **High**. Operator-facing regression on the most common Go install setup; would have blocked any tagged-release upgrader on a fresh devcontainer or stock workstation. Caught by the user during a real upgrade attempt against `v0.2.3`.

---

### G43. Go toolchain and lint surface trail current best-practice — LLM-generated Go drifts toward stale idioms — **resolved**

Resolved in commit `(this commit)` (feat(aiwf): G43 — refresh Go floor and lint surface). All five menu items landed in one stacked commit:

1. **Go floor 1.22 → 1.24.** `go.mod` and the five `go-version: "1.22"` pins in `.github/workflows/go.yml` (the gap miscounted four; the workflow has vet, lint, test, build, selfcheck) bumped to `1.24`. `CLAUDE.md` §Dependencies floor line updated. Local toolchain is 1.26.1; CI's `1.24` resolves to the latest 1.24.x patch via `actions/setup-go@v5`. Validated with `go vet` / `go build` / `go test -race ./...` / `golangci-lint run` — all clean. Unlocks `for range n`, `iter.Seq`, `testing/synctest` (1.24+) for any new code.
2. **`govulncheck` CI job.** New `vuln` job in `.github/workflows/go.yml` runs `govulncheck ./...` against the dep tree. Blocking per the kernel's "validation is the chokepoint" principle. Local run against go1.26.1 flags 5 stdlib vulns + 1 unreached transitive; CI's go-version: "1.24" resolves to a patched 1.24.x and the picture there is the load-bearing one. If the first CI run fires, the right reaction is bumping CI's go-version (decoupling it from go.mod's floor), not silencing the linter.
3. **`thelper`, `forbidigo`, `errorlint` linters.** Added to `.golangci.yml`. `forbidigo` forbids `^panic\(` and `^os\.Exit\(` in library code with two sanctioned exemptions: `cmd/aiwf/main.go` (the documented `os.Exit` chokepoint) and `internal/verb/apply.go` (G2's controlled re-panic after rollback). The gap proposal also listed `fmt.Println\b` — dropped on review: `fmt.Println` to stdout is the documented way to write tool output (`CLAUDE.md` §CLI conventions), and the stderr-as-log smell is too contextual to forbid mechanically. **`errorlint` caught one real bug**: `internal/verb/apply.go:73` wrapped the rollback-also-failed error with `%v` instead of `%w`, so an `errors.As` walk would not surface the rollback error to a caller diagnosing a compound failure. Fixed by switching to `%w` (multiple-`%w` was added in Go 1.20 and is fine on the new 1.24 floor).
4. **`CLAUDE.md` checklist rewrite.** The old "Pre-commit checklist" section was a manual list of seven items presented as if they were locally enforced. Three of them (`go vet`, `golangci-lint run`, `go test -race`) are blocked by CI, not by any pre-commit/pre-push hook in this repo (the hooks run `aiwf check` against the planning tree, not Go-side validation). Replaced with a "What's enforced and where" table that names the chokepoint for each rule and honestly marks the four genuinely-advisory items (context.Context as first arg, no package-level mutable state, dep justification, deliberate Go floor bumps) as code-review-only. Eliminates the "is this advisory or blocking?" ambiguity. Cross-references G2 / G9 / G24 / G31 / G32 / G41 / G43 as relevant.
5. **`flake-hunt` workflow.** New `.github/workflows/flake-hunt.yml`, `workflow_dispatch`-only, runs `go test -race -count=10 -timeout 30m ./...`. ~30 min runtime, zero routine-CI overhead, intended cadence "before tagging a release." The `verb`, `cmd/aiwf`, and `contractverify` test packages all do real git/filesystem work and are the most likely homes for ordering-dependent races; this is the chokepoint for catching them before a tag rather than after.

Severity: **Medium**. None of the items was a live bug at the time the gap was filed, but item 3 surfaced one (the `%v`-on-wrapped-error in apply.go's compound-failure path) that would have silently hidden the rollback error from any caller using `errors.As`. The five items together close the LLM-drift axis: the Go floor moves from 1.22 to 1.24, the lint floor catches three new classes of regression, the doc honestly names what enforces what, and the pre-release flake gate is in place. Not blocking PoC completion but worth closing now before any consumer adopts the framework and inherits whatever Go-floor we shipped at v1.0.

---

### G44. Test surface is example-driven only — no fuzz, property, or mutation coverage of high-value parsers and FSMs — **closed**

**Item 3 — on-demand mutation testing — closed in commit `(this commit)`** (`feat(aiwf): G44 item 3 — on-demand mutation testing via gremlins`). New `.github/workflows/mutate-hunt.yml` adds a `workflow_dispatch`-only job (no cron — mutation testing is too expensive for routine CI) that installs `github.com/go-gremlins/gremlins` and runs it against a user-chosen Go package pattern. The default scope is `./internal/...`, but contributors can target a single package via the `pkg_pattern` input.

Local validation revealed two non-obvious tuning needs documented in the workflow's comments:

- **`--workers 1`** — the default CPU-count parallelism causes the entity package's test runs to time out reliably on this repo (concurrent workers contend on the test-binary build cache). Single-worker is slower in wall-time but produces stable results.
- **`--timeout-coefficient 15`** — gremlins's default of 3 is too tight for the kernel's test suite (especially packages that do filesystem or git work).

Local runs against the kernel's packages established the baseline mutation efficacy:

| Package | Killed | Lived | Not covered | Efficacy |
|---|---|---|---|---|
| `internal/pathutil` | 6 | 0 | 0 | 100% |
| `internal/version` | 33 | 3 | 5 | 91.7% (3 lived are all noise: 2 equivalent mutants in `tripleGreater` where `a[i] > b[i]` and `a[i] >= b[i]` are semantically identical after the `!=` guard, plus 1 unreachable branch in `parseTriple` where the caller pre-validates input) |
| `internal/gitops` | 64 | 6 | 5 | 91.4% |
| `internal/entity` (workers=1) | 58 | 9 | 44 | ~86.5% |

The kernel's test suite is mutation-resistant on the load-bearing paths. Most surviving mutants on inspection are equivalent-mutant noise or unreachable branches. Real surviving mutants would surface as concrete file:line entries in the workflow report and warrant either a new test or a refactor that eliminates the mutation site.

Reading the report (documented in the workflow file): KILLED is good, LIVED is signal-or-noise (review by hand), NOT COVERED is a coverage gap. Equivalent mutants and unreachable-branch mutants are documented false positives — don't chase them; the right resolution is either a refactor that removes the equivalent-mutant pair, or accepting the signal as bounded noise.

**Per the gap's original menu, all three items are now closed.**

**Item 2 — exhaustive property tests for the FSMs (+ drift-prevention follow-up) — closed in commits `fb589c9` (tests) and `(this commit)` (policy).**

Initial commit (`feat(aiwf): G44 item 2 — exhaustive FSM property tests`, `fb589c9`): new `internal/entity/transition_property_test.go` with 11 property tests across all 8 FSMs (6 entity kinds, AC status, TDD phase). Properties: state-set agreement between schemas table and FSM, every declared status is an FSM source, at least one terminal per kind, no self-transitions, all states reachable from initial, `ValidateTransition` total over the closed-set cross-product, `IsLegalACTransition` / `IsLegalTDDPhaseTransition` total, `CancelTarget` always returns a terminal status. Deviation from the gap's `pgregory.net/rapid` proposal: FSMs are tiny enough for exhaustive enumeration to dominate random walks; no new dep added.

Follow-up commit (`feat(aiwf): G44 item 2b — FSM-invariants policy for drift prevention`, this commit): the initial tests had two structural holes a code review surfaced. (1) The iteration source was the test target — they iterated `transitions` (the unexported FSM map), so a new entity Kind added without an entry in `transitions` was *invisible* to the loop and failed silently. (2) The kernel commitment "FSM is one-directional — no demote" lived in prose only; a contributor adding a transition that closed a cycle (e.g., `cancelled → active` to resurrect a cancelled epic) would not trip any test, since the state set is unchanged.

The follow-up encodes both checks as a new policy: `internal/policies/fsm_invariants.go`. `PolicyFSMInvariants` iterates `entity.AllKinds()` (the canonical Kind enum) and asserts: (a) every kind has non-empty `AllowedStatuses`; (b) every kind has at least one non-terminal status (catches "Kind in AllKinds without FSM wiring"); (c) every transition target is in the kind's declared closed set; (d) `CancelTarget(kind)` returns a status that is in the closed set and is terminal; (e) the kind's FSM is acyclic (DFS three-color back-edge detection). Same checks run on the AC-status and TDD-phase composite FSMs via the public `IsLegalACTransition` / `IsLegalTDDPhaseTransition` predicates.

Why a policy and not a co-located test: encoding the checks in `internal/policies/` makes them discoverable as kernel invariants alongside the other 25+ repo-shape rules, rather than buried in a parser-specific test file. The policy uses entity's exported API only (`AllKinds`, `AllowedStatuses`, `AllowedTransitions`, `CancelTarget`, `IsAllowedStatus`), preserving a clean dependency direction (`policies → entity`).

Verified by temp-injection: a deliberate `cancelled → active` cycle in `KindEpic` produced exactly two violations (CancelTarget non-terminal + cycle detected); a deliberate unwired Kind constant added to `AllKinds()` produced exactly one violation (unwired Kind). Both reverted.

Limit deliberately accepted: the policy detects FSM cycles and unwired kinds but does **not** detect "an arbitrary new transition added between existing states." Catching that would require a snapshot/golden-file test (gap item proposed but not implemented — the snapshot mechanism degrades silently if reviewers don't actually review golden-file diffs). For a PoC, the dynamic invariants are enough; the snapshot belongs to a follow-up gap if a real instance of transition-set drift ever ships.

**Item 1 — fuzz tests for high-value parsers — closed in commit `b3e1b2f`** (`feat(aiwf): G44 item 1 — fuzz tests for parsers + CI workflow`). Five `Fuzz*` functions across four files target the load-bearing parsers: `entity.Slugify` / `entity.Split` (`internal/entity/serialize_fuzz_test.go`), `gitops.parseTrailers` (`internal/gitops/trailers_fuzz_test.go`), `version.Parse` covering pseudo-version + `+dirty` per G29 (`internal/version/version_fuzz_test.go`), `pathutil.Inside` covering G1 path-escape (`internal/pathutil/pathutil_fuzz_test.go`). New CI workflow `.github/workflows/fuzz.yml` runs each target for 2 minutes via a 5-job matrix on `workflow_dispatch` and a weekly Sunday cron; corpus directories upload as artifacts on failure.

Fuzzing surfaced one finding during local validation: `parseTrailers` accepts mid-line `\r` into the key, but only on input that real `git log` output never produces. Resolution: relaxed the fuzz invariant from `\r\n` to `\n` (the actual splitter token), kept the corpus seed (`testdata/fuzz/FuzzParseTrailers/acfcce373c0758bf`) so the boundary case stays a regression test, documented the decision in the test file. The production code is unchanged — the fuzz invariant was over-strict relative to the parser's documented input contract. This is the value of fuzz testing: it forced an explicit decision about the parser's contract that the example-driven tests had left implicit.

Items 2 (state-machine property tests for the six entity-kind FSMs using `pgregory.net/rapid`) and 3 (on-demand mutation testing) remain open.

---

#### Original gap text (preserved for items 2 & 3 context)

`CLAUDE.md` and the existing test discipline cover **example-driven test correctness** thoroughly: seam tests (G27), contract tests for cached upstreams (G28), spec-sourced inputs (G29), structural-not-substring assertions, human-verified renders, branch coverage with `//coverage:ignore` rationale, and the no-papering-over-failures rule. Coverage targets are explicit (90% PoC floor, aim for 100% on `internal/...`). CI uploads a `coverage.out` artifact every test run.

What the existing surface does *not* cover is **input-space and assertion-strength coverage**:

- **No fuzz tests.** Zero `func Fuzz*` / `*testing.F` in the codebase. `testing/F` is stdlib and on the new Go 1.24 floor (G43); the cost of adoption is a target list, not a dependency.
- **No property-based tests.** No `testing/quick`, no `pgregory.net/rapid`, no state-machine property generators. The FSM transition functions for the six entity kinds (per kernel commitment 1) are exactly the shape property-based state-machine testing was built for and are currently exercised only by hand-written transition tables in unit tests.
- **No mutation testing.** No `go-mutesting`, no `gremlins`. The existing "structural assertions, not substring matches" rule catches one slice of weak-assertion failure modes; the rest (e.g., a test that passes even after `>` → `<`, or `errors.Is(err, X)` → `err == X`) is unguarded.

**Concrete bugs the kernel has already shipped that one of these techniques would have caught:**

- **G29** (pseudo-version regex example-driven; missed two of three spec forms + the `+dirty` suffix). A `FuzzPseudoVersion` test seeded with one example per spec form, asserting "if the canonical Go toolchain regex matches, ours matches; if it doesn't, ours doesn't" would have surfaced the gap on first run, not mid-implementation. The existing "spec-sourced inputs" rule covers this *if a contributor remembers to enumerate*; fuzzing makes the enumeration mechanical.
- **G8** (Slugify silently drops non-ASCII). A `FuzzSlugify` accepting arbitrary Unicode would have failed on day one against the invariant "if input contains a non-ASCII rune, the dropped-runes set is non-empty." The eventual fix added that invariant explicitly via `SlugifyDetailed`; fuzzing would have driven it before the production hit.
- **G1** (contract path escape via `..` or symlinks). `pathutil.IsContained` is the canonical fuzz target — random path strings + an independent reference implementation (`filepath.Abs` + symlink resolution + prefix check) running side-by-side. The existing test set is example-driven; a fuzz pass would systematically explore the path-grammar surface that the original v0.1 implementation got wrong.
- **FSM-related bugs latent in the closed status sets.** Every entity kind has a hand-coded transition function. A property-based state-machine test (rapid is the canonical Go library here) would assert: from any reachable state, only declared transitions succeed; no sequence of legal transitions reaches a non-declared state; cancellation is terminal. Today these properties are enforced by the type system + a set of unit-test cases the contributor remembered to write — strong but not exhaustive.

**Why this isn't a sub-gap of something else:**

- Not G43 (Go toolchain and lint surface) — that gap closed the *static-analysis* axis. This is the *runtime test-input* axis.
- Not G27 (seam tests) — that rule fixes coverage at the integration boundary; it does not address input-space exhaustion within a unit.
- Not G28 / G29 (contract-test / spec-sourced-input rules) — those discipline how a contributor *writes* example tests; they do not generate inputs the contributor did not think of.

**Proposed fix, in order of payoff. Treat as a menu, not a sequence.**

1. **Add ~5 `Fuzz*` functions against high-value parsers, plus a CI job.** Targets, all with seed corpora from existing test cases:
   - `FuzzSlugify` — invariants: ASCII-only output; non-empty input ⇒ non-empty output OR non-empty `dropped` set; idempotent (`Slugify(Slugify(x)) == Slugify(x)`).
   - `FuzzParseFrontmatter` — invariant: never panics; on success, round-trips back to byte-equivalent YAML; on failure, error is one of the declared finding codes.
   - `FuzzParseTrailers` — invariants: never panics; output trailer set is a subset of declared keys; no key/value contains a newline.
   - `FuzzPseudoVersionRegex` — invariant: agrees with the canonical Go toolchain's pseudo-version detection on a seed corpus drawn from `go list -m -versions` output of a real module.
   - `FuzzPathContained` — invariant: agrees with an independent reference (`filepath.Abs` + `EvalSymlinks` + `HasPrefix`) on every random path; never returns "contained" for a path that escapes via `..` or symlink loop.
   Wire as a new `fuzz` job in `.github/workflows/go.yml` triggered by `workflow_dispatch` and a weekly cron, budget 2 minutes per target. Findings get filed as gaps; fuzz seeds for any reproducer get checked into `testdata/fuzz/`.

2. **Add a state-machine property test for each entity kind's status FSM using `pgregory.net/rapid`.** One generator per kind. Properties:
   - From the initial state, every reachable state is in the declared closed set.
   - Every legal transition produces a state in the declared closed set.
   - Terminal states (`cancelled`, `wontfix`, `rejected`, `retired`, `done` where `done` is terminal for that kind) admit no further transitions.
   - The transition function is total: every (state, action) pair either succeeds or fails with a typed error from the declared error set; never panics, never produces an undeclared state.
   `rapid` is the only new dep; it is widely used and small. The state-machine API (`rapid.StateMachine`) is exactly the shape we need — one struct per kind, generators auto-derived from the closed-set constants per kernel commitment 8.

3. **Defer mutation testing to on-demand.** A `mutate-hunt` workflow modeled on G43 item 5's `flake-hunt` — `workflow_dispatch`-only, run before tagging a release, results reviewed by hand. Tool choice: `github.com/zimmski/go-mutesting` or `github.com/go-gremlins/gremlins`; gremlins is the more actively maintained today (2026-05). Mutation testing has higher false-positive noise (mutants in defensive code, error-message strings, dead branches) and is best as a periodic audit, not a routine gate. Items (1) and (2) close most of what mutation testing would catch in this codebase; (3) is the long-tail backstop.

**Possible outcomes the work should produce, in the matrix entry:**

- *All three.* Fuzz + property + on-demand mutation. Highest coverage; fuzz and property each land as one commit, mutation as a separate workflow file.
- *Items 1 and 2 only.* Defers mutation testing entirely until either (a) a real bug ships that mutation-only would have caught, or (b) the test surface stabilizes after the PoC closes. Probably the right call for the PoC phase.
- *Item 1 only.* Fuzz-first; defer property-based until rapid's value vs. cost is clearer against this codebase. Lowest cost of the three. Justifiable if FSM mutation rate is low going forward (the closed-set kinds are stable per kernel commitment 1).

**Why now:**

The PoC's parser surface has stabilized but is not yet frozen — adding fuzz tests now seeds the corpus while the inputs are still small and the bug-density is highest. Once the framework gains real consumers, fuzz findings on shipped parsers become external-facing bugs; finding them now keeps them as internal-facing fixes. Property tests for the FSMs are even more time-sensitive: kernel commitment 1 freezes the six kinds and their status sets; if the FSM is going to be canonically pinned in this PoC, the property tests are the load-bearing assertion that "the closed set is closed under every reachable transition." That's the kind of property the kernel relies on but does not currently enforce.

Severity: **Medium**. None of the items is a live bug today, and the example-driven test discipline catches most regressions. But the kernel has shipped four bugs (G1, G8, G29, plus the `apply.go` `%v`-on-`%w` caught by G43's errorlint addition) that one of these techniques would have caught earlier, and the FSM correctness is currently rests on hand-written transition tables that would not survive a kind-set extension without re-auditing every test by hand. Not blocking PoC completion; worth the work before any consumer adopts the framework.

Discovered through a follow-up question on G43: "does the doc say anything about coverage vs property-based vs fuzz vs mutation testing?" The answer was: coverage yes, the other three no. This gap files the gap.

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
| G30 | `git log --grep` false-positives leak prose-mention commits into Recent activity / `aiwf history` | Medium | [x] `7141f2a` |
| G31 | Squash-merge from the GitHub UI defeats the trailer-survival contract | High | [x] (this commit) |
| G32 | Merge commits silently bypass the untrailered-entity audit | Medium | [x] (this commit) |
| G33 | `aiwf doctor --self-check` doesn't exercise the audit-only recovery path | Medium | [x] (this commit) |
| G34 | Mutating verbs sweep pre-staged unrelated changes into their commit | High | [x] `890ab01` |
| G35 | HTML site only generates epic/milestone pages — gap/ADR/decision/contract links 404 | High | [x] (this commit) |
| G36 | Entity body markdown rendered as escaped raw text in HTML | High | [x] (this commit) |
| G37 | Cross-branch id collisions split the audit trail; allocator is local-tree only | High | [x] `271f514` (a) + `b9d73d8` (b lineage) + `c5a98c1` (b tiebreaker) + `a6e8067` + `685f288` |
| G38 | The kernel repo does not dogfood aiwf — feasibility and fit need investigation | Medium | [ ] open |
| G39 | `aiwf upgrade` mis-parses `go env` output when GOBIN is unset | High | [x] `9a06c74` |
| G40 | `work/` mechanically unprotected — `aiwf check` silently ignores stray files | High | [x] `bdd43c2` |
| G41 | Tree-discipline ran only at pre-push — LLM-loop signal lands too late | High | [x] `fb2e1e4` |
| G42 | Pre-commit hook coupled enforcement and convenience — `status_md.auto_update: false` removed the gate too | High | [x] (this commit) |
| G43 | Go toolchain and lint surface trail current best-practice — LLM-generated Go drifts toward stale idioms | Medium | [x] (this commit) |
| G44 | Test surface is example-driven only — no fuzz, property, or mutation coverage of high-value parsers and FSMs | Medium | [x] items 1 (`b3e1b2f`), 2 (`fb589c9` + drift policy `49e72f5`), 3 (this commit) |
| G45 | aiwf-managed git hooks don't compose with consumer-written hooks — chokepoint for G38 dogfooding | Medium | [x] `49e7764` |
| G46 | `aiwf upgrade` fails opaquely when the install package path changes between releases — surfaced by v0.4.0 reorg | Medium | [x] (this commit) |
| G47 | `aiwf_version` pin is required, set-once, and never auto-maintained — chronic doctor noise | Medium | [x] (this commit) |

When an item is closed, mark it `[x]` and append a short note (commit SHA or PR link) to the row's title. When deferred deliberately, mark `[x] (deferred)` and add a one-line rationale either in the row or in the body of the entry.
