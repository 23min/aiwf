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

### G6. Design docs are stale relative to I1 (contracts)

**Location:** `docs/poc-design-decisions.md`, `docs/poc-plan.md`.

**Symptom:** Both documents predate the detailed contract design. They describe the six entity kinds and the verb surface but don't mention the contract status set, validator config, recipes, or bind/unbind. A reader following the docs will believe contracts are simpler than they are.

**Why it matters:** New contributors and future-you read these docs to orient. Drift between the design intent and the shipped code is exactly the kind of decay the framework is supposed to prevent.

**Proposed fix:** Short integration pass — add a `### Contracts (added in I1)` section to `poc-design-decisions.md` cross-referencing `poc-contracts-plan.md`, and update the plan checklist in `poc-plan.md` to mark the actual I1 deliverables. ~30 minutes of work.

---

### G7. Skill namespace is a convention, not a guard

**Location:** `tools/internal/initrepo/initrepo.go`, `tools/internal/skills/`.

**Symptom:** The materialization rule is `.claude/skills/aiwf-*`. Anything not matching that prefix is treated as user-authored and left alone. There is no defensive check that a third-party plugin (or a future aiwf companion) using the same prefix won't silently clobber materialized skills.

**Why it matters:** Low risk today, but `aiwf init` / `aiwf update` is supposed to be safe to re-run. A future name collision would silently overwrite or leave stale files.

**Proposed fix:** Maintain a manifest of files aiwf owns (e.g. `.claude/skills/aiwf-*/MANIFEST` or a top-level `.claude/skills/.aiwf-owned`). On update, refuse to overwrite any file not listed in the manifest, and only delete files listed there. Same shape as how npm-style tooling tracks owned files.

---

### G8. Slugify silently drops non-ASCII

**Location:** `tools/internal/entity/serialize.go:80` (`Slugify`).

**Symptom:** `Slugify("Café")` returns `"caf"`. The behavior is documented in the doc comment, but a user who tries to rename `"Café"` → `"cafe"` to "fix" the slug will get a confusing "new slug matches current slug" error with no hint that the dropped `é` is the cause.

**Why it matters:** Papercut, but a sharp one for any non-English title.

**Proposed fix:** Either (a) emit a one-line user-facing notice when input contains non-ASCII characters that were dropped, or (b) tighten Slugify to use Unicode-aware lowercasing and a basic transliteration (`é → e`). (a) is the YAGNI move; (b) is the right move if any real consumer hits this.

---

### G9. `aiwf doctor --self-check` is not run in CI

**Location:** `tools/cmd/aiwf/selfcheck.go`, `.github/workflows/`.

**Symptom:** The self-check exercises every verb end-to-end against a temp repo and is the closest thing to an integration test the project has. CI runs `go test` and `golangci-lint` but never invokes the binary.

**Why it matters:** Regressions in the commit-trailer format, the hook installer, or skill materialization wouldn't be caught until a user upgrades and tries `aiwf init` on a real repo. Unit tests don't cover binary-as-a-whole behavior.

**Proposed fix:** Add a `make selfcheck` target that builds the binary and runs `aiwf doctor --self-check`. Wire it into `.github/workflows/ci.yml` after the test job. Cheap, fast, high-leverage.

---

### G10. macOS case-insensitive filesystem assumption

**Location:** `tools/internal/verb/reallocate.go` (id-collision detection), `tools/internal/verb/common.go` (`pathInside`).

**Symptom:** Path comparisons use exact string matching on forward-slash-normalized paths. On the default macOS APFS volume (case-insensitive), `E-01-foo` and `E-01-Foo` are the same directory to git and the filesystem but distinct strings to aiwf. The id-collision check would not catch a rename collision that the FS already collapsed.

**Why it matters:** macOS is the primary development platform for the PoC. A rename via `git mv` to a case-only variant could produce an inconsistency that the framework doesn't detect.

**Proposed fix:** Detect filesystem case-sensitivity at startup (write a temp file, attempt to stat its uppercased name) and apply case-insensitive comparison on case-insensitive filesystems. Alternatively, refuse case-only renames in the rename verb. Document explicitly in either case.

---

## Low / nits

### G11. `context.Context` not threaded through mutation verbs

**Location:** `tools/cmd/aiwf/main.go` and the verb-level functions in `tools/internal/verb/`.

**Symptom:** `context.Background()` is created in CLI entry points but most mutation verbs (`Add`, `Promote`, `Reallocate`) don't accept a context. Ctrl-C mid-operation can leave partial state and isn't propagated cleanly.

**Why it matters:** Violates the `tools/CLAUDE.md` rule that `context.Context` is the first arg of every IO-touching function. Also blocks future cancellation features (timeouts, graceful shutdown in editor integrations).

**Proposed fix:** Mechanical — add `ctx context.Context` as the first argument to every verb function and thread it through. Combined with G2's atomicity work, gives clean Ctrl-C behavior.

---

### G12. Pre-push hook hard-codes binary path at install time

**Location:** `tools/internal/initrepo/initrepo.go`.

**Symptom:** The hook script written by `aiwf init` embeds the absolute path to the `aiwf` binary at the moment of install. If the user later moves or upgrades the binary to a different location, the hook silently breaks (or runs the wrong version).

**Why it matters:** A stale hook violates the framework's authoritative-enforcement promise. A user could believe pre-push is gating and not notice it isn't.

**Proposed fix:** Either (a) write the hook to look up `aiwf` on PATH at run time (simpler, but loses determinism), or (b) keep the absolute path but have `aiwf doctor` verify the hook target still exists and matches the current binary's path. (b) is the better fit for the framework's "verifier, not validator" stance.

---

### G13. No Windows guard

**Location:** `tools/internal/initrepo/initrepo.go` (writes `#!/bin/sh` hook), `tools/internal/contractverify/` (shells out to validators).

**Symptom:** The code is Unix-only in practice but has no `//go:build !windows` guards or a runtime check. A Windows user can `go install` the binary; first failure surfaces deep in the call stack.

**Why it matters:** Friendlier failure mode and clearer scope.

**Proposed fix:** Either guard the Unix-specific bits with build tags and provide a stub that errors with a clear message, or check `runtime.GOOS == "windows"` at startup and refuse with a one-line "Windows is not supported in the PoC; see docs/poc-design-decisions.md". Either way, document in README.

---

## Status matrix

| ID  | Title                                                       | Severity | Status |
|-----|-------------------------------------------------------------|----------|--------|
| G1  | Contract paths can escape the repo (via `..` or symlinks)   | High     | [x] `4ec5d84` |
| G2  | `Apply` is not atomic on partial failure                    | High     | [x] `f77740c` |
| G3  | Pre-push hook fails opaquely when validators are missing    | High     | [x] `23f4231` |
| G4  | No concurrent-invocation guard                              | High     | [x] `620ecca` |
| G5  | Reallocate's prose references are warnings, not errors      | Medium   | [x] `0e247fe` |
| G6  | Design docs are stale relative to I1 (contracts)            | Medium   | [ ]    |
| G7  | Skill namespace is a convention, not a guard                | Medium   | [ ]    |
| G8  | Slugify silently drops non-ASCII                            | Medium   | [ ]    |
| G9  | `aiwf doctor --self-check` is not run in CI                 | Medium   | [ ]    |
| G10 | macOS case-insensitive filesystem assumption                | Medium   | [ ]    |
| G11 | `context.Context` not threaded through mutation verbs       | Low      | [ ]    |
| G12 | Pre-push hook hard-codes binary path at install time        | Low      | [ ]    |
| G13 | No Windows guard                                            | Low      | [ ]    |

When an item is closed, mark it `[x]` and append a short note (commit SHA or PR link) to the row's title. When deferred deliberately, mark `[x] (deferred)` and add a one-line rationale either in the row or in the body of the entry.
