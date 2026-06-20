# Changelog

All notable changes to `aiwf` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project follows [Semantic Versioning](https://semver.org/).

Releases ship as git tags on `main`. The Go module proxy resolves
them when a consumer runs `aiwf upgrade` or
`go install <pkg>@latest`.

When cutting a release, see [`CLAUDE.md`](CLAUDE.md) § *Go conventions §
Release process*. The tag-push CI check at
[`.github/workflows/changelog-check.yml`](.github/workflows/changelog-check.yml)
fails any pushed `v*` tag that does not have a matching `## [X.Y.Z]`
section in this file.

## [Unreleased]

### Changed — E-0042: test-quality debt burned down across policies and the kernel corpus (internal)

No user-visible behavior change. The firing-fixture-presence meta-gate's `grandfatherDark` ledger went from 43 entries to its irreducible 1 (`fsm-invariants`) — every policy now carries mechanical evidence it can fire (M-0166). The dormant forbidigo lint enforcement was revived with an execution firing harness, and a redundant policy now covered by gocritic was removed (M-0167, M-0170). Two complementary probes ran over the kernel: a gremlins mutation sweep (M-0168) and a directed wf-vacuity assertion-shape audit (M-0169), killing 11 mutation survivors and strengthening 3 vacuous test assertions; per-package mutation-efficacy baselines (entity 85.5% / gitops 91.9% / verb 86.2% / check 88.5%) are recorded in `docs/pocv3/`.

## [0.15.1] — 2026-06-18

### Fixed — diff-scoped coverage gate marker placement (no functional changes)

The `//coverage:ignore` annotation on `readBuildInfoVersion`'s unreachable degenerate-build `return` was moved inline onto the statement so the diff-scoped coverage gate (G-0067) recognizes it within the statement's coverage block; the v0.15.0 push otherwise failed CI on that line. No functional or behavioral change versus v0.15.0 — the annotated line is the unreachable `runtime/debug.ReadBuildInfo` fallback, and the moved marker is a comment.

## [0.15.0] — 2026-06-18

### Added — three new engineering ritual skills (wf-rethink, wf-property-test, wf-vacuity)

`aiwf init`/`update` now materialize three more `wf-*` engineering rituals. **`wf-rethink`** (G-0256) re-evaluates one unit's design by reconstructing it from intent against a list of pinned obligations, then keeps or rewrites — biased to keep, with any rewrite gated behind explicit human approval. **`wf-property-test`** (G-0257) turns a unit's crisp invariant (conservation, round-trip, idempotence, monotonicity) into a generative or metamorphic property test instead of a single example — and unlike the design-quality ritual the artifact is a real CI gate. **`wf-vacuity`** (G-0258) adversarially audits whether a unit's existing tests can actually fail: a mutation probe (break the implementation, confirm a test goes red) plus a tautology/over-narrowing probe that flags assertions passing for the wrong reason.

### Added — G-0067: diff-scoped coverage gate for changed Go code

`wf-tdd-cycle`'s branch-coverage HARD RULE is now mechanical for aiwf's own Go code: every statement on a line changed since the base ref must be exercised by a test or annotated `//coverage:ignore <reason>`, else CI fails the job naming the `file:line`. The engine is `internal/policies/branch_coverage_audit.go` (`PolicyBranchCoverageAudit`), run in the CI coverage-gate step and locally via `make coverage-gate`. Honest limit: Go's `-cover` is statement coverage, so this is diff-scoped *statement* coverage with the `//coverage:ignore` escape — true per-arm branch correlation is tracked as a follow-up (G-0253).

### Fixed — G-0235 candidate B: single-source binary version

A `make install` / ldflags-stamped binary reported two different version strings — `aiwf version` printed the stamp while the JSON envelope and `aiwf doctor`'s binary row read buildinfo (`(devel)` for a working-tree build). The ldflags stamp moved from `internal/cli.Version` into `internal/version.Stamp` so every surface resolves through `version.Current()` and one binary reports one string. A new `version-single-source` policy test blocks a parallel version global from reappearing outside `internal/version`.

### Fixed — G-0255: `promote --superseded-by` writes the reciprocal supersedes link

`aiwf promote <old> superseded --superseded-by <new>` wrote `superseded_by` on the superseded ADR but never the reciprocal `supersedes` on the superseding ADR, and no other verb could. The two-sided `adr-supersession-mutual` check therefore fired permanently with no CLI path to clear it; the verb now writes both sides in the one commit.

### Fixed — G-0198: branch-id prefix coherence in `ParseEntityFromBranch`

The ritual-branch parser matched any `E-`/`M-`/`G-` id under any prefix, so a typo'd `epic/M-0001-foo` parsed to `M-0001` and silently miscorrelated in `aiwf status --worktrees`. The pattern now scopes each prefix to its own id family.

### Fixed — G-0247: `aiwf add ac` no longer duplicates AC body headings

On a milestone scaffolded from the ritual template (which ships placeholder `### AC-1`/`### AC-2` headings), `aiwf add ac` appended a second `### AC-N`; the `acs-body-coherence` check collapsed body headings to a set and passed the duplicate clean. The verb is now idempotent on headings and the check flags duplicates.

### Fixed — G-0250 / G-0251: git test-fixture hardening

Test fixtures that shell out to git inherited the ambient environment (a parent git hook's `GIT_DIR`/`GIT_INDEX_FILE` retargeted them at the parent repo) and allowed background auto-gc to race fixture commits and `t.TempDir` cleanup — both surfacing as the recurring "invalid object / Error building trees" / "directory not empty" flake. New `testsupport.HardenGitTestEnv()` scrubs the locator vars and disables auto-gc; `PolicyGitTestEnvHardened` enforces the call in every exec-bearing `internal/*` test package's `TestMain`.

### Fixed — `aiwf init` gitignore block label

The aiwf-managed `.gitignore` block is now labeled "marker-managed framework artifacts" — accurately covering the materialized agents, templates, and guidance fragment, not just skills.

## [0.14.0] — 2026-06-16

### Added — E-0040: per-turn LLM guidance auto-wired into the consumer's CLAUDE.md

`aiwf init`/`update` now materialize a version-pinned guidance fragment (`.claude/aiwf-guidance.md`) and automatically maintain a marker-wrapped `@.claude/aiwf-guidance.md` import in the consumer's root `CLAUDE.md` — so the advisory rules aiwf can't mechanically enforce (per-action gate discipline, never-suggest-pause, collision→`reallocate`, AC-evidence, …) load on every turn and survive `/compact`. The wiring is self-healing (re-added on the next `update` if removed), line-anchored (touches only its own marker block; surrounding content preserved verbatim), and default-on with an `aiwf.yaml` opt-out (`guidance.wire_claudemd: false`) — no CLI flag, mirroring how skills/hooks are materialized. An advisory `aiwf doctor` finding (`claudemd-guidance-unwired`) surfaces an unwired tree and names `aiwf update` as the fix. Ratified in ADR-0018 (risk-calibrated consent for user-owned file edits); closes G-0243. Delivers every milestone listed in `work/epics/E-0040-*/wrap.md`.

### Fixed — pre-push lint gate: per-working-tree golangci-lint cache

The G-0179 pre-push lint hook (and `make lint` / `make ci`) now scope `GOLANGCI_LINT_CACHE` to the checkout's own git dir (`.git/`, or `.git/worktrees/<name>/` for worktrees — removed automatically with the worktree) instead of golangci-lint's shared user-level cache. The shared cache stores raw issues keyed by package content hash but carrying the absolute paths of whichever checkout linted the package; a cache hit from a since-deleted worktree replays paths the nolint/filter processors can no longer read, so they fail open and leak suppressed findings — observed false-blocking a `main` push with `gosec` ghosts from a removed worktree. An operator-set `GOLANGCI_LINT_CACHE` is still respected. Contract pinned in `internal/policies/prepush_lint_hook_test.go`.

### Changed — G-0221: disk-level atomic writes via a central temp+fsync+rename helper

Every kernel write to a persistent file now routes through `pathutil.AtomicWriteFile` (sibling temp file → `fsync` → `os.Rename`), so an OS crash or hard kill mid-write can no longer leave a half-written entity file, `aiwf.yaml`, settings file, git hook, or rendered HTML page behind. Rerouted sites: the verb mutation writer (`verb.Apply`, including its rollback restore), the three `aiwf.yaml` rewrites in `internal/config`, the statusline settings/scaffold/gitignore writes, the skill/agent/template materialization and ownership manifests, `aiwfyaml.Doc.Write` (previously temp+rename without fsync), the htmlrender per-page emit (renders to memory first — a render killed mid-run no longer truncates pages), and the `aiwf init` artifact writes (`.gitignore`, CLAUDE.md template, all four git hooks). The new `atomic-write-chokepoint` policy test blocks future raw `os.WriteFile` / `os.Create` / write-mode `os.OpenFile` calls in production code; the two allowlisted exceptions are `aiwf doctor --self-check` (sandbox-confined writes) and the repo lockfile (fd carries flock only, never content). As a side effect of the buffer-first render, a template error during `aiwf render` no longer destroys the existing page it was about to replace.

## [0.13.0] — 2026-06-12

### Added — G-0179: pre-push golangci-lint boundary gate; rituals name the full local CI gate

Long-lived unpushed branches accumulated `golangci-lint` debt invisibly: CI only lints on push, and per-milestone validation as practiced ran `go vet` + `go test`, not the full lint set — the discovery evidence was 9 CI-blocking findings spanning three milestone wraps on E-0038's epic branch. Two layers close it. **Mechanical (kernel repo):** `scripts/git-hooks/pre-push` runs the same linter CI runs, locally, before any push whose range touches Go surfaces (`*.go`, `go.mod`, `go.sum`, `.golangci.yml`); planning-tree pushes short-circuit in ~25 ms, warm-cache code pushes pay ~2–3 s. Installed as the G45 chain target `.git/hooks/pre-push.local` via `make install-hooks`; a missing linter warns rather than blocks (CI remains the backstop); `git push --no-verify` is the deliberate escape hatch; the lint runs against the checked-out working tree (documented approximation — CI, which lints the pushed sha, remains authoritative). Decision logic is pinned branch-by-branch in `internal/policies/prepush_lint_hook_test.go` via the `AIWF_PREPUSH_LINT_CMD` override. **Ritual (consumer-facing):** `wf-patch`'s verify step, `aiwfx-wrap-milestone`'s verify-completion step, and a new `aiwfx-wrap-epic` precondition now name the project's full local CI gate (e.g. `make ci`), not a subset — the prior "any project-specific lint" phrasing was vague enough that `go vet` read as satisfying it. Per-wrap enforcement on a still-unpushed branch remains ritual-level; the hook guarantees the machine boundary.

### Changed — wf-patch gates consolidate to commit / wrap / push, with a declared-sequence wrap gate

A full `wf-patch` under per-action gating cost ~7 approvals (commit, branch push, merge, mainline push, promote, promote push, cleanup) — disproportionate for the low-stakes surface now that mechanical gates carry the safety load (full local CI gate at the verify step, the G-0179 pre-push lint hook, `aiwf check` pre-push). The skill restructures to three gates: **commit gate** (now preceded by an explicit self-review step in the `wf-review-code` checklist shape, so the gate presents a reviewed diff with green-gate evidence), **wrap gate** (one approval covering the patch's *enumerated* terminal sequence — local merge to mainline, tracker closure, cleanup — binding to exactly the verbatim list, with partial approval honored and any deviation re-gating), and **push gate** (push to origin is never part of the wrap sequence; it always stands alone). PR-flow projects keep the prior shape — the declared sequence assumes local merge. The kernel repo's CLAUDE.md gains the matching sanctioned exception, scoped to wf-patch only; milestone and epic wraps keep per-action gates.

### Fixed — G-0245: embedded ritual agents no longer instruct the retired work/tracking-doc convention

Thirteen stale lines across five embedded ritual artifacts (`builder` and `reviewer` agents, `aiwfx-record-decision`, `aiwfx-wrap-milestone` — including the `description:` line Claude Code surfaces in skill discovery — and `aiwfx-wrap-epic`) still instructed the v1 separate tracking-doc convention the snapshot itself had retired, so a builder agent following its own instructions recreated the retired `work/tracking/` directory and nothing mechanical objected. All thirteen now point at the in-spec replacements (frontmatter `acs[]`, the milestone spec's `## Work log` and `## Decisions made during implementation` sections). A new `internal/policies/` chokepoint (`embedded-rituals-no-retired-tracking-doc`) fails CI on any `work/tracking/` path reference, or any "tracking doc" phrasing outside explicit v1-historical context, anywhere in the embedded snapshot — the retired convention cannot be silently reintroduced by a future ritual edit.

### Fixed — G-0240 + G-0241: `body-prose-id` false-positive classes (CommonMark-aware mask, trunk resolution tier)

The rule's code-span stripper is now CommonMark-aware (multi-backtick spans, indented code blocks, link destinations, multi-line spans) per G-0240, so prose discussing id syntax inside those constructs no longer self-trips. Id resolution gains a trunk tier per G-0241, so trunk-only ids referenced in body prose no longer surface as `unresolved` on feature branches with stale trunk awareness.

### Fixed — G-0237: `aiwf acknowledge-illegal --for-entity` accepts composite AC ids

The flag previously rejected `M-NNNN/AC-N` composite ids. Companion hardening: archive-path resolution for `--for-entity` is pinned by test (G-0238), and the acks-helper-lift policy now also polices `WalkAcknowledgedSHAEntities` (G-0239).

### Changed — G-0244 follow-up: `aiwfx-release` CI-green check is path-filter-aware

The skill's Pre-release checks step now explains how to verify CI green when HEAD is a markdown-only commit excluded by workflow path filters: walk back to the most recent Go-affecting commit and verify that commit's run, rather than concluding from an empty run list.

## [0.12.0] — 2026-06-11

### Fixed — G-0244: aiwfx-release gains a CI-green precondition; vuln + lint findings cleared on main

Closes the gap where the `aiwfx-release` ritual's Constraints section asserted *"releases ride on green commits"* but the procedural Pre-release checks (step 1) never verified CI green on the target commit — only local test/build green. Discovery case: an earlier v0.12.0 tag was cut against pre-existing red CI on main (the `vuln` and `lint` jobs in `go.yml` had been failing across the last 10+ commits back to E-0030's wrap) because the operator (Claude in this session) confused locally-green with CI-green. The tag was yanked from origin and re-cut after this patch landed.

The fix carries three coordinated changes. **`aiwfx-release` skill** — Pre-release checks step 1 gains an explicit "CI is green on the target commit" sub-bullet pointing operators at `gh run list --workflow=go.yml --branch=main --limit 1`; the Constraints line strengthens to *"Green is CI-green, not just locally-green"* with the discovery-case citation. **`.github/workflows/go.yml`** — six `go-version: "1.25.10"` pins bumped to `"1.25.11"` to pick up the stdlib patches for `GO-2026-5039` (`net/textproto` error escaping) and `GO-2026-5037` (`crypto/x509` candidate hostname parsing); both were reachable from `gitops.BlobReader.Read` and `status.RenderWorktreeViews`. **Lint findings** — six pre-existing gocritic / gofumpt / govet findings cleared across five files (`authorize_scenarios_test.go` ×2 `appendAssign`, `branch_scenarios_helpers_test.go` `rangeValCopy`, `m0162_ac4_pin_call_shape_test.go` gofumpt, `trailer_order_matches_constants.go` gofumpt comment spacing, `m0162_ac2_build_tag_test.go` `shadow err`).

The closure evidence is the next release ritual exercising the new step — if the operator forgets the `gh run list` check, the local-vs-CI distinction is now explicit in the Constraints line they're reading at the Tag gate.

### Changed — G-0242: gate-discipline rule lifted into CLAUDE.md and the mutating-walker ritual skills

Closes the gap where the rule "each mutating action is its own approval gate" lived only in advisory-skill bodies (encoded as per-step 🛑 markers) and the agent's general-purpose system prompt — surfaces that either load only when the skill is invoked or remain general-purpose. Neither survived `/compact` cleanly: post-compaction summaries preserve *that* approvals were given but not the *granularity* at which they fired, and the agent reads the summary as cadence rather than history. Discovery case: the G-0163 wf-patch close-out bundled five distinct mutating actions (commit + push + merge + promote + archive) into a single approval prompt because that was the inherited shape from a prior session's summary.

The fix lifts the rule into surfaces that re-anchor it. **`CLAUDE.md`** gains a third bullet under "Working with the user" naming the rule as a standing invariant: per-action gating, no bundling, no inferring cadence from post-compaction summaries — re-injected every turn. **`wf-patch`** gains a `## Gate discipline` preamble at the top stating the invariant, and step 7 ("commit, push" — bundled) splits into a separate Commit gate (step 6, unchanged), a new Push gate (step 8), and a new "delete remote branch" gate (step 11) so the audit-trail steps each gate independently. **`aiwfx-release`** gains the same preamble and splits the old section 5 ("Tag as vX.Y.Z and push?" — bundled tag-creation + commit-push + tag-push) into a Tag gate (creation only, locally reversible) and a separate Push gate (the irreversible boundary). Steps 7–9 renumber accordingly; the Constraints lines update to name each gate.

Layer 3 (consumer CLAUDE.md materialization — aiwf today does not touch the consumer's CLAUDE.md, only `.claude/skills/aiwf-*` and `.git/hooks/`) tracked separately under a follow-up gap.

### Fixed — G-0163: `aiwf cancel` no longer routes ADR/Decision `accepted` through the FSM-illegal `rejected` target

`entity.CancelTarget(KindADR, StatusAccepted)` and `(KindDecision, StatusAccepted)` previously returned `StatusRejected` unconditionally. The verb's `Cancel` path applied the transition without an FSM legality check, but `accepted → rejected` is not in the FSM's allowed-transitions set — the only outgoing edge from `accepted` is `superseded` (via `promote --superseded-by`). The verb succeeded; the FSM-illegal `rejected` state landed in trunk on every `aiwf cancel ADR-NNNN` against an accepted ADR or Decision. `CancelTarget` is now state-aware for ADR and Decision (mirroring M-0131's Contract pattern): only `proposed` returns `rejected`; `accepted` returns `""` and the verb surfaces `<id> (kind "adr", status "accepted") has no cancel target`. The fsm-invariants drift mode 3 policy and `TestCancelTarget_AllKinds` property test were relaxed from "non-terminal must have a cancel target" to "if non-empty, must be legal+terminal" — the FSM may have no cancel-target from a particular non-terminal state, and the policy must reflect what the FSM actually permits. M-0125/AC-2's `adr-accepted-cancel` cell graduates from `ac2KnownImplGaps` to live end-to-end coverage (`errorSubstringsFor("fsm-transition-illegal")` matches the new "no cancel target" phrasing). R-RULE-021 in the legal-workflows audit catalog updated to describe the ADR/Decision state-aware branch alongside Contract's.

### Changed — G-0184 reviewer follow-throughs: line resolution, edge-case coverage, defensive tests, two filed gaps

Addresses items surfaced across the three G-0184 reviewer passes that were tracked but not fixed at the time. **(1) Line resolution** — `check.ScanBodyProseID` now computes the 1-based line number within the body from each match's byte offset (was hardcoded 1 via the `Field: "body"` resolution-fallback path); `bodyProseID` adjusts to file-relative by adding the body-start line. `resolveLines` no longer overwrites pre-set `Line` values, so the new resolution stays. **(2) Edge-case coverage** — `TestBodyProseID_EdgeCases` table (9 cases) pins the ASCII-only contract (Greek `M-α`, Cyrillic `M-АБВ` silent), HTML-tag prose, prefix-suffix concatenation (`M-0001prefix` fires malformed-shape), narrow-numeric conversational leak (`M-1` fires), and start/end-of-body tokens. `TestBodyProseID_ResolvesLineFromTokenOffset` pins the line-resolution contract. **(3) Defensive verb-time tests** — `TestImport_RefusesMalformedIDInManifestBody` and `TestReallocate_RefusesMalformedIDInProseRewrite` close the test-symmetry gap the second reviewer flagged (rewidth bug had escaped because its seam looked identical to add/edit-body and the symmetry argument was fragile). Both assert `EntityID` is the canonical id, not the file path. **(4) Messy-fixture coverage** — `G-001` body now exercises two body-prose-id subcodes (`M-foo` malformed-shape + `G-9999` unresolved); `TestFixture_Messy` expected-codes list extended; `TestBinary_Check*` baselines regenerated. **(5) `widenEntityID` bare-id contract** — docstring + 18-case test pin the helper's contract (below-grammar narrow widens, composite ids pass through, unknown shapes pass through). **(6) Const-string surgery cleanup** — `longEnrichedBodyAC2` in `trunk_rename_g0167_test.go` rewritten with `[BT...BT]` placeholders rendered via a small `bt()` helper; the raw-string stays clean. **(7) G-0237 retitled** to drop literal backticks from the title for cleaner render in `aiwf list` output.

Filed as follow-up gaps (architectural, not one-patch fixes): **G-0240** (`body-prose-id` stripper is not CommonMark-aware — multi-backtick spans, indented blocks, link URLs, multi-line spans) and **G-0241** (`BodyProseIDIndex` skips `TrunkIDs` — trunk-only ids in body prose surface as `unresolved` on feature branches with stale trunk awareness).

### Added — G-0184 follow-up: `body-prose-id` runs at verb time across body-supplying verbs

Strengthens the G-0184 chokepoint from "pre-push hook only" to "verb-time refusal." Previously a body containing a malformed or unallocated id-shaped token (`M-a`, `M-NNNN`, `M-9999`) supplied via `aiwf add --body-file`, `aiwf edit-body` (both bless and `--body-file` modes), `aiwf import`'s manifest `body:` field, `aiwf reallocate`'s prose rewrites, or `aiwf rewidth`'s body rewrites would land silently at write time; only the next `aiwf check` (in the pre-push hook or standalone) surfaced the bad content. The verb-time scan now catches this before any file is written: each affected verb calls `check.ScanBodyProseID` against the planned-write bytes and refuses to produce a `Plan` (returning `body-prose-id` findings instead) if any errors fire. The previously-named `skipDuringProjection` filter stays in place but is no longer load-bearing — verb-time scanning is the gate, the filter just suppresses noisy false-positives against stale on-disk bytes that the verb's planned writes will replace.

The refactor introduces `check.ScanBodyProseID(body, entityID, path, idx)` plus `check.BodyProseIDIndex(t)` as the shared scanner+index pair; the existing tree-walking `bodyProseID(t)` rule (run by `aiwf check`) now calls through the same helpers. The on-disk audit path is unchanged in behavior, so a body-prose-id finding that slips past a verb (via direct `git commit`, `--no-verify`, or a build that predates this change) still surfaces at pre-push.

### Added — G-0184: `body-prose-id` check rule pins id-shape chokepoint at the committed-body layer

Closes the gap where an LLM (or human) could write a malformed id-shaped token (`M-a`, `M-alpha`, `M-NNNN`) or an unallocated canonical-shape token (`M-9999`) into entity body prose and have it land in trunk silently — `refs-resolve` only covers structured frontmatter references. The new `internal/check/body_prose_id.go` rule scans every active entity's body prose for tokens shaped like aiwf ids and classifies each into `malformed-shape` (letter suffix, uppercase placeholder, or narrow-numeric like `M-1`), `unresolved` (well-formed canonical id that resolves to no entity), `unresolved-milestone` (composite id with missing parent), or `unresolved-ac` (composite id with present parent but missing AC). Inline code spans (`` `...` ``) and fenced code blocks (```` ``` ```` and `~~~`) are exempt so prose discussing id syntax does not self-trip. Archive entities are skipped, mirroring `refs-resolve` scoping. The advisory layer adds a CLAUDE.md standing rule against fabricated id-shapes plus the same guidance in `aiwfx-plan-epic` / `aiwfx-plan-milestones`: in conversation, narrow numeric labels (`M-1`, `M-2`) are acceptable conversational shorthand; in committed prose, use the allocator-assigned canonical id or backticks. The active-tree backfill wraps 30 pre-existing meta-syntax mentions (`M-NNNN`, `E-NN`, `D-NNN`, `ADR-shaped`, `C-option`, etc.) in backticks across active gaps, decisions, and ADR bodies. The verb-projection `projectionFindings` filter skips `body-prose-id` so `aiwf reallocate`'s prose-rewrite step isn't blocked by intermediate-state findings the verb itself fixes.

### Changed — G-0195: canonical trailer-key set derived from `trailerOrder`; const ↔ order drift policed

`internal/cli/integration/trailer_shape_test.go::canonicalTrailerKeys` was a hand-maintained mirror of `trailerOrder` in `internal/gitops/trailers.go`. The whole point of the mirror was to detect "new trailer landed without a `Trailer*` constant" — but the membership set itself drifted silently (G-0231 backfilled the specific gaps it had accumulated for `TrailerForceFor` / `TrailerBranchSHA`). This release closes the drift class structurally on both sides: `gitops.CanonicalTrailerKeys()` is the new accessor returning the membership view derived from `trailerOrder` at package init, and the integration test now reads through it. A new `internal/policies/trailer_order_matches_constants.go` AST policy asserts set-equality between the `Trailer*` const block and identifiers inside `trailerOrder` so the next-layer-up drift (a new constant added to the block but not appended to the slice) fails CI with the offending identifier named.

### Fixed — G-0236: `aiwf acknowledge-illegal` now accepts orphan SHAs

The verb's M-0136/AC-4 reachability check (`git merge-base --is-ancestor <sha> HEAD`) refused acks against `isolation-escape-orphaned-ai-commit` findings because the rule's offending SHAs are by construction unreachable from HEAD — they're force-pushed-away tips surfaced via the reflog walker. The asymmetry mirrors G-0214 (which closed the same shape for `forced-untrailered`): the rule consumed the ack-set map, but the verb refused to mint the ack commit. The fix adds a fallback path: when reachability fails, the verb checks `git rev-parse --verify <sha>^{commit}` and accepts if the SHA exists in the local object DB. Typo guard preserved — a SHA that resolves to no commit at all fails both checks. Per-SHA closed-set scoping unchanged. The `acknowledge-illegal --help` and skill body list every rule the ack-set now covers (FSM-history, isolation-escape, orphaned-ai-commit, promote-on-wrong-branch, id-rename-untrailered, trailer-verb-unknown).

### Added — G-0218: commit-msg hook + post-hook severity tightening for fabricated `aiwf-verb:` trailers

Closes the composition-time gap that let an operator (human or LLM) type any string into the `aiwf-verb:` slot of a hand-rolled commit message and have it land silently in trunk history. **Patch 1** materializes a `# aiwf:commit-msg` git hook via `aiwf init` / `aiwf update` (mirroring the existing pre-commit / pre-push hook pattern); the hook shells `aiwf check --commit-msg "$1"` and refuses values outside the closed registered-verbs set ∪ ritualVerbs allowlist. Body prose mentioning `aiwf-verb: X` is unaffected (the hook canonicalizes the trailer block via `git interpret-trailers --parse` first). **Patch 2** tightens the post-hoc `trailer-verb-unknown` rule from advisory warning to **error** for commits whose ancestry includes the hook-install SHA — landing a fabricated trailer past the hook required `--no-verify` or git plumbing, so the rule treats it as a policy violation rather than a historical inheritance. Pre-hook history (and any clone where the hook-install SHA is unreachable — shallow clone, fork divergence) stays at warning so `addressed_by_commit` references aren't retroactively broken. Sovereign-human override (`aiwf acknowledge-illegal <sha>`) continues to silence regardless of severity tier. Closes G-0218.

### Added — E-0030: Branch model chokepoint — `--branch` flag, sequencing, isolation-escape finding

Ratifies the ritualized branch model (ADR-0010): work happens on `epic/E-*` and `milestone/M-*` branches; author iteration lives on `main`. Delivers the mechanical enforcement on top of the existing verb surface — no new top-level verbs were introduced. `aiwf authorize` (pre-existing) gains a `--branch <ritual>` flag and emits `aiwf-branch` + `aiwf-branch-sha` trailers that couple a scope to its branch. `aiwf acknowledge-illegal` (pre-existing; from M-0136/E-0033) gains two additional silencing targets — `fsm-history-consistent/forced-untrailered` and `isolation-escape` — via the shared `aiwf-force-for: <sha>` trailer mechanism. Six new check-time findings catch AI-actor escapes across the matrix: `isolation-escape` (M-0106), `isolation-escape-oracle-failure` (M-0161/AC-3 — fail-shut on correctness, fail-open on coverage per D-0019), `isolation-escape-shallow-clone` (M-0161/AC-4), `isolation-escape-orphaned-ai-commit` (M-0161/AC-5), `id-rename-untrailered` (M-0160/AC-4), `promote-on-wrong-branch` (M-0161/AC-8). Two new verb-time typed errors carry the preflight-refusal surface: `branch-context-required` (M-0103) and `rung-pair-illegal` (M-0161/AC-2, subsuming the earlier `branch-not-found` per D-0018). The layer-4 branch-choreography spec catalog (M-0158, M-0162) names 129 cells whose 1:1 correspondence to tests is mechanically enforced at CI time by a bijection meta-test split across static AST scan (invariants 1/2/3) and runtime `branchtest.Pins()` post-hook (invariant 4) per D-0024. Skill-discipline fixes for `aiwfx-start-epic` (M-0104) and `aiwfx-start-milestone` (M-0105) close the ordering gaps that surfaced during epic execution. Closes G-0099, G-0210; addresses every milestone listed in `work/epics/E-0030-*/wrap.md`. See ADRs ratified there (-0010, -0011, -0012, -0013) for the durable architecture.

## [0.11.0] — 2026-05-31

### Added — E-0039: optional install path for the aiwf-aware statusline

The aiwf-aware Claude Code statusline (entity badges, token ball, CI state, sync indicators) ships as an optional install via `aiwf init/update --statusline [--scope project|user]`. The script is embedded in the binary (`go:embed`), scaffolded once (never clobbered by `aiwf update`), and activated via consent-gated settings wiring: interactive `[y/N]` prompt on a TTY, or explicit `--wire-settings` flag in non-TTY / CI contexts (ADR-0015). Project scope writes to `settings.local.json`; user scope writes to `~/.claude/settings.json`; a pre-existing `statusLine` key is never overwritten. `aiwf doctor` reports dep availability (`jq`, `gh` with platform-branched install hints), installed-but-not-wired state, embedded-vs-on-disk drift, and a container `--scope user` nudge — all advisory. The shipped script is portable (macOS `tail -r` + GNU `tac` fallback, default-IFS ahead/behind parse) and hardened against Git index-lock contention (`GIT_OPTIONAL_LOCKS=0`). Closes G-0183; see `work/epics/E-0039-*/wrap.md`.

### Fixed

- **`aiwf render` roadmap reconciles the on-disk `ROADMAP.md` filename**
  across case-sensitive and case-insensitive filesystems — a repo
  tracking lowercase `roadmap.md` gets that file updated, not a divergent
  second `ROADMAP.md`; new `roadmap-case-collision` check finding for the
  two-variants-present state. Closes G-0185.

- **`aiwf promote --by-commit`** now validates that each commit SHA
  resolves to a real commit, rejecting unresolvable SHAs (mirrors `--by`
  entity-id validation); `--force` bypasses for sovereign overrides.
  Closes G-0186.

- **Statusline HUD shows in-flight epics on every branch** with canonical
  status glyphs (`→` active, `○` proposed) and colors. On ritual branches,
  the current epic is accentuated (bold `▸`) with its milestone inline.
  Also adds effort level segment, branch truncation at 30 chars, dirty
  marker icon (`✎`), and CI glyphs (`✓`/`✗`/`→`) replacing text labels.
  Closes G-0188.

## [0.10.0] — 2026-05-30

### Changed — E-0038: rituals ship embedded; Claude marketplace retired

The companion rituals — planning/lifecycle skills (`aiwfx-*`), engineering skills (`wf-*`), the role agents (planner/builder/reviewer/deployer), and entity templates — are now **embedded in the `aiwf` binary** from a pinned upstream snapshot and materialized into `.claude/` by `aiwf init` / `aiwf update`, alongside the verb skills. There is no marketplace install and no `/plugin` step; the ritual version always equals the binary version. `aiwf doctor` now reports a `rituals:` line verifying the materialized artifacts (pointing at `aiwf update` if any are missing) and a `marketplace-rituals-overlap` de-dupe guard that instructs operators to disable a still-enabled `ai-workflow-rituals` plugin — without editing `settings.json`. The `doctor.recommended_plugins` config key is retired (old yamls still load; the key is ignored). The materializer is parameterized by agent target, so non-Claude targets (Codex `.agents/skills/`, etc.) become new writers behind the seam. Operator-setup docs (CLAUDE.md, README) are rewritten to the one-command flow. Closes G-0177; see `work/epics/E-0038-*/wrap.md`.

## [0.9.0] — 2026-05-28

### Added — `aiwf doctor` cross-platform plugin-index path-corruption advisory (closes G-0174)

When running in a Linux container, `aiwf doctor` scans the Claude Code plugin index (`known_marketplaces.json`, `installed_plugins.json`) for path values rooted at a foreign-OS prefix (e.g. macOS `/Users/...` inside a Linux container, or the inverse) and emits an advisory `plugin-paths:` line naming [anthropics/claude-code#31388](https://github.com/anthropics/claude-code/issues/31388) and pointing at the shadow-mount remediation. Complements the existing `plugin-mount:` line (which only checks that the in-container target exists and is populated, not that the paths *inside* the index are OS-correct). Advisory only — never increments doctor's problem count, since cached skills often still load while marketplace refresh fails.

### Added — E-0037: scope-reach as an executable legality precondition in the spec

`scope-reach` (D-0006's three-edge scope reachability) is now an executable, legality-classed predicate in the legal-workflow spec — the verb-time out-of-scope refusal, previously enforced only in hand-written Go (M-0141), now lives inside the spec's bidirectional drift net. **No runtime reachability behavior changed**; this completes the formal-model certification M-0141 deferred (closes G-0171).

- **M-0144 — ADR-0013: global-precondition representation + out-of-scope legality classification.** Decides how a cross-cutting precondition is represented in the spec and sizes the cellcoverage extension. (Amended at M-0147 to the separate-`GlobalRules()`-accessor mechanism after the originally-ratified `Global` flag proved to fan skip-exceptions across the per-cell meta-tests.)
- **M-0145 — `scope-reach` evaluable in `EvaluatePredicate`.** The spec predicate evaluates reachability by delegating to `tree.ReachesScope` (no re-derivation of D-0006), with `EvalContext` carrying the actor scope-entity + target. Provably agrees with the runtime gate.
- **M-0146 — authorized-scope cellcoverage fixtures.** `CellFixture.AuthorizeScope` stands up a real `aiwf authorize` scope so the driver path can exercise a scope-gated cell (in-scope succeeds; out-of-scope refused).
- **M-0147 — global `scope-reach` rule + legality reclassification.** A `spec.GlobalRules()` rule carries the precondition; `provenance-authorization-out-of-scope` is reclassified to a `codes.Code{ClassLegality}` descriptor (D-0011 pattern), so the AC-5 fourth arm covers it. Closes G-0171.

### Fixed

- **`aiwf status --worktrees` no longer reports merged worktrees as in-flight (G-0172).** A worktree whose branch is fully merged into trunk and whose driver entity is terminal on trunk now renders under the existing "SAFE TO REMOVE" path instead of as phantom in-flight work. `BuildWorktreeViews` resolves driver terminality against the main-checkout's tree (trunk) when the branch carries no ahead-of-trunk commits; the merged-branch gate and trunk-terminal check are layered so a freshly-forked worktree (also zero-ahead, but active on trunk) is never mis-flagged.

### Changed — E-0036: Reconcile impl to the legal-workflow spec

- **M-0138 — Typed `Coded` error pattern for legality-pertinent verb refusals.** Introduced `entity.Coded` (`interface { error; Code() string }`) + `entity.Code` (`errors.As`-based extraction) and the `codes.Code{ID, Class}` descriptor, piloted on `fsm-transition-illegal` (`FSMTransitionError`). Verb-time legality errors now carry a first-class structured code, mirroring `check.Finding{Code}`. ADR-0012 ratifies the pattern; D-0011 records the descriptor-class decision. Closes G-0142.
- **M-0139 — Refuse cancel of parents with non-terminal children/ACs, via coded errors.** `aiwf cancel` now refuses an epic with non-terminal child milestones (`epic-cancel-non-terminal-children`) or a milestone with `open` ACs (`milestone-cancel-non-terminal-acs`), each a `codes.ClassLegality` typed error (D-0003/D-0004). Closes G-0139.
- **M-0140 — Classify legality finding codes; close the AC-5 bidirectional arm.** The legality code set is now enumerable from the `codes.Code` descriptors themselves (no parallel allowlist); the AC-5 spec↔impl drift policy gained its fourth arm — every `ClassLegality` impl code must be named by an illegal-outcome spec rule. Closes G-0145.
- **M-0141 — Enforce three-edge scope reachability at verb-time.** Scope reachability (verb-time `verb.Allow` and check-time `provenance-authorization-out-of-scope`) narrowed from the full reference graph to D-0006's exact three edges (parent-forward, composite-id containment, `discovered_in`-reverse); governance edges (`depends_on`, `addressed_by`, `relates_to`, `supersedes`, `superseded_by`, `linked_adrs`) no longer cross a scope boundary — closing a scope-leak where an authorized agent could reach cross-epic entities. The out-of-scope refusal now carries the structured `provenance-authorization-out-of-scope` code (`errors.As`-able, surfaced as `error.code` under `--format=json`), distinct from no-active-scope. D-0014 records the reconcile + the split of the formal-model arm to G-0171. Closes G-0143.
- **M-0143 — Surface `Coded` verb refusals in the `--format=json` envelope.** Every mutating verb now accepts `--format`/`--pretty` (uniform with the read verbs). A verb-time legality refusal emits an additive `error: {code, message}` object under `status: "error"` — the structured code resolved via `entity.Code` (`errors.As`), not message-parsing. **Behavior change:** a `Coded` (legality) verb refusal now exits `1` (was `2`), in both text and JSON modes, unifying the exit code with the check-time exit for the same violation class; non-`Coded` verb errors still exit `2`. The envelope `error` slot is additive — existing `--format=json` consumers are unaffected. D-0013 records the decision (A2/representation/exit-code); `import`, `rewidth`, and the read-display commands route around the shared chokepoint and are tracked for JSON wiring in G-0169.
- **M-0142 — Rename finding code `gap-resolved-has-resolver` → `gap-addressed-has-resolver`.** Matches the gap FSM's `addressed` terminal; the old name referenced a `resolved` state the FSM no longer has. **Breaking change to the `aiwf check --format=json` `findings[].code` surface:** any downstream tool that pins the literal `gap-resolved-has-resolver` must refresh to `gap-addressed-has-resolver`. The break is narrow — no `aiwf.yaml` knob and no committed rendered artifact (`STATUS.md`, `ROADMAP.md`) references finding codes, so only a hand-written JSON-parsing script is affected, and the rename is upgrade-gated per consumer. D-0012 records the decision and the per-repo confirmation step; closes G-0144.

### Added — E-0033: Pin legal kernel-verb workflows mechanically

The kernel now commits to a spec table for legal and illegal verb workflows.
`internal/workflows/spec/rules.go` enumerates every (Kind, FromState, Verb)
cell with `Outcome ∈ {Legal, Illegal}`, an `ExpectedErrorCode` for illegals,
and a `RejectionLayer` (verb-time vs check-time) axis. Positive and negative
drivers under `internal/policies/` exercise every cell end-to-end against
the real `aiwf` binary; AC-4 meta-tests on both sides enforce coverage
parity with the spec, including an AST-based assertion-strength tooth on
the negative driver that catches silent-skip regressions on impl-gap cells.
ADR-0011 documents the methodology. Separately, M-0130 added the
`fsm-history-consistent` check rule (per-entity git history walk that emits
findings for FSM-illegal status transitions) and M-0136 added
`aiwf acknowledge-illegal` so kernel-repo legacy violations get explicit
ack commits instead of history rewrites; M-0131 fixed Contract's
`CancelTarget` mapping (`deprecated → retired`, not `accepted → retired`);
M-0137 closed the batched git-ops silent-swallow path that M-0130 surfaced.

- **M-0120 — Ratify legal-workflow spec methodology in ADR.** ADR-0011 ratifies the three-pass methodology (audit → first-principles → reconcile) and commits the kernel to Go-as-canonical-form for the spec table. Structural test pins the ADR's seven decision sections.
- **M-0121 — Pass A audit: catalog legal-workflow rules from existing surfaces.** Markdown catalog of every rule observed in the impl (FSM transitions, verb guards, check-rule findings) without first-principles reasoning. 191 rules catalogued under `docs/pocv3/design/legal-workflows-audit.md`.
- **M-0122 — Pass B first-principles: derive legal-workflow rules from entity model.** Independent first-principles derivation, blind to Pass A's output, surfacing 84 rules from the entity model. The blind-derivation discipline produced the cross-check that Pass C reconciles against.
- **M-0123 — Pass C reconcile to canonical Go spec table + drift policy.** `internal/workflows/spec/rules.go` lands as the closed-set spec table. Drift policies under `internal/policies/m0123_ac*` enforce: every FSM transition / verb / finding code is referenced; every illegal cell has either an impl-side `Code: "..."` literal or a `deferredImplErrorCodes` entry with a tracking gap. Six decisions captured: D-0002 through D-0007.
- **M-0124 — Positive cell coverage: legal workflows succeed with expected post-state.** Per-cell driver that enumerates every legal cell and drives the real `aiwf` binary against a per-cell fixture, asserting verb success + post-state matches the cell's contract. AC-4 meta-test (`TestM0124_AC4_*`) enforces full enumeration + name uniqueness + target-derivation invariants.
- **M-0125 — Negative cell coverage: illegal workflows rejected with named errors.** Per-cell drivers for both verb-time (≥27 cells) and check-time (≥2 cells) illegal cells. Two-way staleness teeth track impl-gap divergences (kernel under-rejects → assertion fires when kernel learns to reject; kernel over-rejects → assertion fires if kernel softens). Five meta-tests including an AST-based no-`t.Skip` tooth on the driver files. Authorize-kind allowlist guard (D-0007 verb-time refusal) landed as part of the milestone; G-0166 documents the spec/impl axis mismatch on the two check-time cells.
- **M-0130 — Implement `fsm-history-consistent` check rule for FSM tree-invariant.** New check rule under `internal/check/fsm_history_consistent.go` walks per-entity git history in DAG order (per-parent comparison, not linearization adjacency) and emits findings for status transitions that violate the per-kind FSM. Lives in the CLI layer (not `check.Run`) so the per-entity git walk doesn't stall the pre-commit hook's shape-only policy path.
- **M-0131 — State-aware `CancelTarget` for Contract: cancel deprecated targets retired.** `entity.CancelTarget(KindContract, "deprecated")` now returns `"retired"`. Previously returned `"retired"` only for `accepted`; the deprecated→retired transition is the operationally correct path (D-0002 codifies why operational kinds admit abrupt-stop).
- **M-0136 — `aiwf acknowledge-illegal`: retroactive force trailer for historical violations.** New verb that emits `aiwf-force-for: <sha>` trailers exempting specific historical commits from the `fsm-history-consistent` check rule. Kernel repo's 4 legacy squash-merge violations are now ack'd explicitly; future operators can ack their own legacy state without history rewrites or `aiwf.yaml` pollution.
- **M-0137 — `fsm-history-consistent`: batched git ops + silent-swallow fix.** Per-entity history walk now uses a single batched git invocation (one `git log` call per entity instead of one per entity-commit-pair) so the check rule is tractable on real kernel-sized trees; the silent-swallow path where a missing entity raised an error that the rule dropped instead of surfacing is now an explicit finding.

Five M-0123-era deferred-impl gaps remain open as a deliberate carry-forward: G-0139 (cancel refusal on non-terminal children/ACs per D-0003/D-0004), G-0140 (`--evidence` flag per D-0005), G-0141 Phase 2 (structured-code emission for verb errors per D-0007 follow-up — Phase 1 verb-time refusal landed in M-0125), G-0142 (structured `fsm-transition-illegal` error), G-0143 (scope-tree three-edge reachability per D-0006). Plus M-0125-session discoveries: G-0144 (rename `gap-resolved-has-resolver` semantics), G-0145 (legality-pertinent finding-code classifier), G-0160 (per-edge FSM coverage drift), G-0161 (antirules negative coverage), G-0166 (RejectionLayerCheckTime spec/impl axis mismatch), G-0167 (ids-unique trunk-collision false positive on retitle + body enrichment — fix in flight on `fix/trunk-collision-rename-threshold`), G-0168 (kernel missing mutation verbs for set-at-create frontmatter fields).

## [0.8.1] — 2026-05-21

### Added — E-0035: Devcontainer-based dev loop (dogfooded on this repo)

The devcontainer is now the default dev surface. `make ci` runs green from VS Code's "Reopen in Container" without macOS-specific setup; `CLAUDE.md` leads with the container-primary test-running path and demotes the macOS host wrapper (`scripts/sign-and-run.sh` + `-parallel 8` cap + G-0127/G-0128/G-0133 diagnostic discipline) to a clearly-labeled fallback. The kernel surfaces that broke under multi-context use (PATH-relative hook resolution, worktree-aware `aiwf update`, `aiwf doctor` reading `enabledPlugins` from `.claude/settings.json` instead of path-strict `installed_plugins.json`) now hold across worktrees, devcontainers, and re-clones. `aiwf doctor` gains two informational lines — `env:` (container vs host) and `plugin-mount:` (shadow-mount health) — so operators land on a quick "where am I + is the workaround healthy" signal. Cross-repo dogfooding (Liminara, FlowTime) deferred to G-0146 once the dogfooding loop proved itself on this repo.

- **M-0132 — Land `.devcontainer` skeleton.** Go-base devcontainer + project-scope plugin install + shadow-mount workaround for [claude-code#31388](https://github.com/anthropics/claude-code/issues/31388). `.devcontainer/initialize.sh` sets up the host-side `~/.claude-linux/plugins` symlink before the container starts; `devcontainer.json`'s mount entry backs the in-container `~/.claude/plugins` onto it so the Linux plugin index lives in a parallel store from the macOS host index. `make ci` runs green inside the container with no signing wrapper.
- **M-0133 — Multi-context kernel surfaces: portable hooks + doctor check.** Three coordinated fixes. AC-1 (G-0135): all three aiwf-installed hooks (`pre-push`, `pre-commit`, `post-commit`) resolve `aiwf` via PATH at hook-fire time (`command -v aiwf`) instead of a baked install-time path; doctor detects both the new and the pre-G-0135 shapes. AC-2 (G-0136): `aiwf update` from a worktree writes hooks to the shared `git rev-parse --git-common-dir/hooks` location, not the worktree-local `.git/hooks` (which doesn't exist in worktrees). AC-3 (G-0138): `aiwf doctor`'s recommended-plugin check reads `enabledPlugins` from the project-committed `<rootDir>/.claude/settings.json` (path-independent), not the machine-local `~/.claude/plugins/installed_plugins.json` (path-strict; produced false positives across worktrees and devcontainers).
- **M-0134 — `CLAUDE.md` test-running doctrine refresh.** The `## Go conventions → ### Testing` area now leads with `#### Running tests in the devcontainer (primary)` and demotes the macOS wrapper discipline to `#### Running tests on macOS host (fallback)`. The stale `"Structural fix (Linux devcontainer) is parked."` sentence is gone (it shipped). New policy `PolicyM0134ClaudeMdTestRunningSections` walks the markdown heading hierarchy and pins the structure mechanically (per CLAUDE.md *Substring assertions are not structural assertions*).
- **M-0135 — `aiwf doctor` containerized-env awareness.** New `InContainer()` probe in `internal/cli/doctor/env.go` checks `/.dockerenv` + `AIWF_DEVCONTAINER` and emits an `env:` line on doctor output. When in container, a `shadowMountStatus(home)` probe inspects `<home>/.claude/plugins/` and emits a `plugin-mount:` line reporting `ok (N plugin entries cached)` / `empty` / `missing` (100+ cap on the count). Both lines are read-only — never increment doctor's problem count.

One follow-up gap filed during the epic: G-0146 (cross-repo dogfooding hardening — Liminara, FlowTime — deferred until the in-repo loop proved its shape; survives the epic for the next forcing function).

### Changed — E-0025: Test-suite parallelism and fixture-sharing pass (closes G-0097)

`go test ./internal/... -count=1` now runs in 24.5s on a 20-core dev host, down from 53.6s baseline (~2.2× speedup at default parallelism). `cmd/aiwf/` tests see a ~47% wall-time reduction (174s → 87s on a clean run). The parallel-by-default convention is now documented under a new `### Test discipline` section in `CLAUDE.md` (*Go conventions*) and pinned mechanically: every `internal/*` test-bearing package fails CI unless it carries a `setup_test.go` with `func TestMain(m *testing.M)`, and every race-mode `go test` invocation across the Makefile + GitHub workflows fails CI unless it carries `-parallel 8`. The race-cap chokepoint already shipped in M-0091; M-0093 surfaced it in the *What's enforced and where* table and added the setup_test.go-presence chokepoint alongside.

- **M-0091 — TestMain + t.Parallel across `internal/*` test packages.** 24 packages converted (1 commit per package + a leading `-parallel 8` cap commit landing the cap in `Makefile`, `.github/workflows/go.yml`, and `.github/workflows/flake-hunt.yml`). `internal/policies/race_parallel_cap.go` pins the cap structurally; `internal/policies/shared_tree_test.go::sharedRepoTree` exposes a `sync.Once`-shared live-repo `*Tree` (read-only by convention; 5 consumers wired). Six helpers stripped of `t.Setenv` blocks (now redundant under TestMain); one serial test by design (`TestApply_RollsBackOnCommitFailure` deliberately clears GIT identity to provoke a commit failure). 10/10 clean under `go test -race -parallel 8 -count=1 ./internal/...`.
- **M-0092 — TestMain + t.Parallel + no-ldflags dedup in `cmd/aiwf/`.** 337 of 447 cmd-side Test* functions adopt `t.Parallel()`; 110 documented serial across four categories (integration_g37_test.go subprocess fan-out; captureStdout/Stderr/Run-caller stdout-mutation; t.Setenv callers; os.Chdir callers) in `cmd/aiwf/setup_test.go`'s skip-list comment. 6 no-ldflags `buildBinary` calls in `binary_integration_test.go` swapped to `aiwfBinary(t)` (already sync.Once-shared via `integration_test.go`). AC-4 (strict 10/10 `-race -parallel 8` reliability) deferred to G-0125: macOS dense subprocess fan-out produces 20–30% flake rate from `os/exec` deadlocks in `gitops.StagedPaths`; multiple tests participate, no single fixable culprit. First Linux CI run on the wrapped state was green.
- **M-0093 — Document test-discipline convention and lock its chokepoint.** New `### Test discipline` section under `## Go conventions` in CLAUDE.md covers the five load-bearing rules (setup_test.go per package, t.Parallel first-line, serial skip-list, sync.Once for shared fixtures, -parallel 8 cap). `internal/policies/test_setup_presence.go` is the AST-level chokepoint: walks every `internal/*` test-bearing directory and fails CI if `setup_test.go` is missing or its TestMain declaration has the wrong signature. `internal/policies/claude_md_test_discipline.go` is the belt-and-suspenders structural assertion that the CLAUDE.md section exists under the right parent heading. G-0097 closed in this milestone via `aiwf promote G-0097 addressed --by E-0025`.

Two follow-ups carried forward: G-0125 (the macOS subprocess fan-out reliability question; four remediation paths sketched in its body) and G-0104 (whether to ship the test-parallelism discipline to consumers via `wf-rituals` or stay BYO; decision becomes interesting once a second consumer hits the same wall).

### Changed — E-0029: Glanceable governance HTML render — layout, sidebar, chips (closes G-0114)

The rendered governance site (`aiwf render --format=html`) becomes usable for current-state synthesis at a glance. Body fills the viewport (no more `max-width: 78rem` cap) with 1rem uniform padding; sidebar widens to 285px and sits flush-left; prose blocks inside `main` cap at 50rem for readability while tables, code, and AC cards stretch with the panel (M-0098). Per-kind index pages (`gaps.html`, `decisions.html`, `adrs.html`, `contracts.html`) collapse from active/all-pair to a single file per kind with a `:target`-driven `[Active] [All]` chip strip at the top; archived rows are hidden by default and revealed via `#all`; `*-all.html` cousins no longer emit (M-0099). The sidebar gains a `Gaps (N)` entry (non-archived count) in its top section and its own `[Active] [All]` chip strip with the distinct `#sidebar-all` fragment so the sidebar archive filter and kind-index page filter toggle independently — archived epics hide by default, closing the "all 29 epics drown the in-flight ones" half of the glanceability issue (M-0100). Tab clicks (M-0098/AC-5) and chip clicks (M-0100/AC-4) no longer scroll the page on hash change — `scroll-margin-top: 100vh` clamps the page at top. Playwright e2e suite repaired across three independent rot layers (repo reorg paths, kernel `aiwf init` hook-installation change, ID width migration) and the G-0055 `--tdd` chokepoint — 55 passing tests covering layout, chip filters, sidebar, and no-scroll-on-click (M-0107, renumbered from M-0102 at wrap due to concurrent allocation on main). M-0101 (in-page status hierarchy in `gaps.html`) deferred: mechanism choice (server-side sort vs CSS `order:` vs grouped sections vs hybrid) needs more design thought; cancelled-milestone body preserved for a future iteration.

Two follow-up gaps filed during the epic: G-0115 (`aiwf render roadmap --write` rewrites entity refs in epic prose to broken paths — blocks the roadmap-regen step at wrap until fixed) and G-0116 (the rituals-plugin `aiwfx-start-epic` skill orders worktree creation before promote/authorize, producing the wrong commit topology for trunk-based projects). Both survive the epic for future attention.

### Added — `aiwf status --worktrees` worktree-aware view (closes G-0122)

`aiwf status --worktrees` lists every git worktree against the repo (including the main checkout) alongside the in-flight epics and milestones for each. Surfaces the "what am I working on in which worktree?" question for operators juggling multiple feature branches in sibling worktrees. Read-only.

### Added — `aiwf-show` embedded skill (closes G-0087)

The `aiwf show <id>` verb gains a same-named embedded skill so AI assistants reach for it on prompts like "show me M-NN" or "what's the body of G-NN?". Materializes via `aiwf init` / `aiwf update`. Closes the deferred follow-up flagged in E-18's M-074 allowlist rationale.

### Added — `make diag-aiwf` operator target for worktree binary discipline (closes G-0147)

New `Makefile` target builds `./bin/aiwf-diag` from the current worktree source and prints its absolute path so operators diagnose kernel behavior against the in-flight code, not against the stale `/go/bin/aiwf` they happened to install earlier. CLAUDE.md *Worktree binary discipline* documents the convention. Operator discipline only — nothing mechanically blocks a stale-PATH `aiwf` call in a worktree.

### Changed — `aiwf list` / `aiwf status` use bold labels + uniform status glyphs (partial G-0080)

Headers and section labels render bold when stdout is a TTY and `NO_COLOR` is unset; pipes and redirected output stay escape-free. The status palette is uniform across both verbs: `✓` for done/accepted/addressed, `→` for active/in_progress, `○` for draft/open/proposed, `✗` for cancelled/wontfix/rejected/retired/superseded. `aiwf list` gains a per-row glyph prefix on the `STATUS` column; `aiwf status`'s milestone marker extends to cover the previously-unmarked draft and cancelled states. Glyphs are content (always emitted); only the ANSI bold attribute is TTY-gated.

### Changed — `aiwf doctor` label-column alignment + `plugin-mount` rename (closes G-0130)

Doctor's per-line label column aligns visually. The container-only `plugin-index-mount:` line renames to `plugin-mount:` to match what the underlying mount actually is (a full plugin store, not just the index).

## [0.8.0] — 2026-05-11

### Added — E-0028: Start-epic ritual `aiwfx-start-epic` (closes G-0063 start-side)

Activating an aiwf epic is now a deliberate sovereign ritual. The rituals plugin's new `aiwfx-start-epic` skill orchestrates a 10-step workflow at activation time: preflight reads of the epic spec, drafted-milestone check (new kernel finding), `aiwf check` cleanliness, tests/build advisory pass, worktree-placement Q&A (three options: no-worktree on `main`, `.claude/worktrees/<branch>/`, or sibling `../aiwf-<branch>/`), branch-shape Q&A (placeholder pending G-0059), delegation prompt (in-loop vs. `aiwf authorize E-NN --to ai/<id>`), sovereign promotion via `aiwf promote E-NN active`, optional `aiwf authorize`, and hand-off to `aiwfx-start-milestone`. The promote verb refuses non-`human/` actors at runtime (mirroring the existing `--force` actor coherence rule); the standard `--force --reason "..."` override remains available for genuine sovereign-act-shaped exceptions. CI/script chokepoint added under `internal/policies/` to catch static `aiwf promote E-... active` invocations missing `--force`. Wrap-side concerns (`scope-end-before-done`, human-only on `done`, `aiwfx-wrap-epic` update) deliberately deferred to follow-up gap G-0111. Rituals-repo fixture lands at `ai-workflow-rituals/87fc790`.

- **M-0094 — `aiwf check` finding `epic-active-no-drafted-milestones`.** New warning fires when an `active` epic has zero `draft`-status milestones — the kernel signal feeding the skill's drafted-milestone preflight. Reading-A strict-literal semantics chosen over reading B/C in the planning conversation; rule scoped to `active` epics only. Documented in `internal/skills/embedded/aiwf-check/SKILL.md` per the AI-discoverability policy.
- **M-0095 — Sovereign-act enforcement on `aiwf promote E-NN active`.** New `requireHumanActorForEpicActivation` helper runs inside the existing `!force` block alongside `requireResolverForResolutionClass`. Refusal error explicitly names "sovereign", references the `human/` requirement, and points at the `--force --reason "..."` override path so a non-human actor reading the message understands both *why* and *how*. Scoped to `epic / proposed → active` only; other kinds and other epic transitions unaffected.
- **M-0096 — `aiwfx-start-epic` skill fixture + structural AC tests + drift-check.** Fixture authored at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` per CLAUDE.md *Cross-repo plugin testing*; five structural AC tests pin frontmatter shape, the 10-step Workflow section, the three worktree-placement options, the G-0059 deferral note in the branch prompt, and the sovereign-promotion step's content (verb + `human/` + `--force --reason`). Rituals-repo copy committed at `87fc790`; drift-check test in this repo compares cache against fixture when present and skips cleanly when absent.
- **M-0097 — CI/script audit chokepoint and AC-5 drift comparator.** Late-added milestone closing two verification seams surfaced during M-0096's confidence audit. `auditUnforcedEpicActivate` helper + `TestPolicy_NoNonForcedEpicActivateInCIScripts` seam test convert M-0095's "we checked CI/scripts" paper trail into a permanent CI chokepoint; `compareSkillBytes` helper + `TestCompareSkillBytes_BranchCoverage` exercises M-0096/AC-5's drift comparator's two arms synthetically (independent of marketplace-cache state). Manual mutation review in place of `mutate-hunt` because `gremlins --diff <ref>` excludes new files in worktree configs (filed as G-0110).

### Changed — E-0027: Trailered merge commits from `aiwfx-wrap-epic` (closes G-0100)

The rituals plugin's `aiwfx-wrap-epic` skill now prescribes a *trailered* merge commit for the integration-target merge: `git merge --no-ff --no-commit <epic-branch>` followed by `git commit --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"`. Without `--no-commit`, git produces an untrailered merge commit and the kernel's existing `provenance-untrailered-entity-commit` finding fires once per entity file the merge touched (historical instances on E-0024 and E-0026 wrap commits remain as accepted artefacts; no history rewrite). The change is fixture-first per CLAUDE.md *Cross-repo plugin testing* — authoring at `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md`, structural drift-check tests in `internal/policies/aiwfx_wrap_epic_test.go`, copy-to-rituals-repo at wrap. No kernel rule changes; the chokepoint stays strict.

- **M-0090 — `aiwfx-wrap-epic` emits trailered merge commits; fixture + drift-check tests.** Fixture body rewrites step 5 of the wrap-epic workflow (and tightens step 8's wrap-artefact commit to carry the same trailers). Six AC tests pin: frontmatter shape, trailered-sequence substring in the merge-step section, structural section-scoping per CLAUDE.md *Substring assertions are not structural assertions*, cache-vs-fixture parity against the active install resolved from `installed_plugins.json`, post-wrap rituals-repo SHA recording, and kernel-rule unchanged. Rituals-repo copy committed at `3faae39`. This epic's own merge commit is the dogfood — the first trailered-merge wrap under the new ritual.

### Changed — E-0026: `aiwf check` per-code summary by default (closes G-0098)

Default text output of `aiwf check` collapses warnings to one line per finding-code: `<code> (warning) × N — <representative message>`. Errors continue to print per-instance — each error is per-instance-actionable. A new `--verbose` flag restores the full pre-epic per-instance shape byte-for-byte. The JSON envelope is unchanged modulo `metadata.root` (which is environmental); machines still receive every finding via `--format=json` regardless of `--verbose`. On the kernel tree the post-E-0023 / post-E-0024 advisory state (~176 near-identical `terminal-entity-not-archived` lines + the paired `archive-sweep-pending` aggregate) shrinks from a ~180-line scroll to a 5-line scannable summary. Sort order is count-desc with alphabetic tie-break (pinned so golden files don't drift). No check rules, severities, or finding codes changed.

- **M-0089 — Per-code text-render summary with `--verbose` fallback.** New `render.TextSummary` partitions findings: errors flow through the existing per-instance path, warnings group by `Code` into per-code buckets. `Text` was refactored to share a `renderPerInstance` helper so the verbose path stays byte-identical to the pre-epic behaviour by construction, not just by golden file. Sample message per code is the first finding's `Message` verbatim. Binary integration tests at `cmd/aiwf/check_summary_binary_test.go` (kernel-tree ≤10-line bound, byte-identity against captured baselines for verbose text, structural-equal modulo `metadata.root` for JSON, `--help` documentation of `--verbose`). Discovered the friction post-E-0024 when the advisory paired-finding shape became the new normal; this milestone collapses the noise at the render layer alone.

### Added — E-0024: Uniform archive convention for terminal-status entities (ADR-0004)

Every entity kind now stores terminal-status files under a per-parent `archive/` subdirectory — `work/gaps/archive/`, `work/decisions/archive/`, `work/contracts/archive/`, `work/epics/archive/`, `docs/adr/archive/`. The active-directory listing reflects what is currently in-flight without filter ceremony. Movement is decoupled from FSM promotion: `aiwf promote` and `aiwf cancel` flip status only; the new `aiwf archive` verb sweeps qualifying entities into their archive subdirs as a single commit per invocation. The loader resolves ids across active and archive, so cross-references stay live indefinitely. Drift is policed via the new `archive-sweep-pending` advisory finding, with an opt-in `archive.sweep_threshold` knob in `aiwf.yaml` that flips the finding to blocking past the named count. Recorded as item 10 of CLAUDE.md *What aiwf commits to*.

### Fixed — G-0102: Title length cap at write-time

`aiwf add`, `aiwf retitle`, `aiwf import`, and `aiwf rename` now hard-reject titles exceeding `entities.title_max_length` (default 80 chars — the Conventional Commits subject-line convention so entity-touching commit subjects stay scannable in `git log`). The slug shares the same budget so filesystem and frontmatter stay in sync. Non-positive configured values fall back to the kernel default. Existing entities with pre-cap titles are grandfathered (write-time policy doesn't retroact); operators retitle them manually when convenient.

### Changed — G-0108: `aiwf retitle` syncs the on-disk slug

`aiwf retitle` now atomically renames the entity file's on-disk slug to match the new title in the same commit. Previously the slug stayed on the old form until a separate `aiwf rename` was run; the two-verb cleanup pass collapses to one verb per entity going forward.

### Fixed — G-0109: `aiwf check` trunk-collision recognizes branch renames

The `ids-unique/trunk-collision` rule now recognizes git renames since the merge-base with trunk and treats them as the same entity moved, not duplicate id allocations. Previously a slug-renamed entity on trunk plus the old-slug version on a feature branch fired as a hard collision; now the rename is detected and the check passes. Critical companion fix to the G-0102 / G-0108 work (renames on trunk while a branch is in flight would otherwise block the branch's wrap merge).

## [0.7.0] — 2026-05-10

### Changed (breaking) — Module path rename `github.com/23min/ai-workflow-v2` → `github.com/23min/aiwf` (closes G-0094)

The kernel's Go module path now matches its binary name. The GitHub repo is renamed from `23min/ai-workflow-v2` to `23min/aiwf`; the old repo is archived under the same owner and remains accessible as a historical record. Pre-rename tags (`v0.1.x` and earlier) continue to resolve at the old path until the Go module proxy expires them; post-rename releases ship at the new path only.

**Existing installs require manual reinstall.** `aiwf upgrade` from a binary built before this rename will keep querying the old proxy path and silently keep returning pre-rename tags. To migrate:

```bash
go install github.com/23min/aiwf/cmd/aiwf@latest
```

Clones of the repo: `git remote set-url origin git@github.com:23min/aiwf.git` (or HTTPS equivalent). GitHub's auto-redirect handles the URL change for already-cloned trees, but the new origin is the durable one.

### Changed — E-0023: Uniform 4-digit kernel ID width (closes G-0093)

Three milestones (M-0081, M-0082, M-0083) collapse the per-kind id-width policy (E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN) to a single canonical 4-digit form across every kernel id kind. Parsers tolerate narrower legacy widths on input so existing trees, branches, and commit trailers continue to validate without history rewrite; renderers and allocators always emit canonical width. Files **ADR-0008** (policy precedent for the entire epic; promoted from `proposed` to `accepted` at wrap). Closes **G-0093**.

- **M-0081 — Canonical 4-digit IDs in parser, renderer, and allocator.** Replaces the per-kind `canonicalPad` map with `const entity.CanonicalPad = 4`; consolidates the duplicate `canonicalPadFor` from `internal/verb/import.go` into the entity package. New `entity.Canonicalize(id)` left-pads any recognizable id to canonical width; `entity.IDGrepAlternation(id)` emits a regex matching both widths for `git log --grep` callers. Lookup-seam canonicalization threaded through `Tree.ByID` and friends; display surfaces (`aiwf list / status / show / history / render`) emit canonical ids regardless of on-disk filename. AC-5 mechanical chokepoint at `internal/policies/narrow_id_sweep_test.go` pins the test-fixture-sweep discipline.
- **M-0082 — `aiwf rewidth` verb + apply to this repo's tree.** Net-new top-level Cobra verb canonicalizes a consumer's narrow-width tree to 4-digit form: file renames per kind (active tree only; archives preserved per ADR-0004 forget-by-default) plus body-content reference rewrites (bare ids, composite ids, markdown links — code fences, inline backticks, URL fragments, archive paths excluded). Idempotent on canonical or empty trees; one commit per `--apply` with multi-entity-sweep trailer (`aiwf-verb: rewidth`, no `aiwf-entity:`). Applied to this repo's tree at wrap (200 file renames + 212 body rewrites in commit `f937288`).
- **M-0083 — Drift check, normative-doc amendments, skill content refresh.** New `entity-id-narrow-width` warning rule fires only on mixed-state active trees (some canonical alongside some narrow), staying silent on uniform-narrow, uniform-canonical, and empty trees — the on-demand-migration framing. ADR-0003 §"Id and storage" amended `F-NNN → F-NNNN` with cross-reference to ADR-0008; CLAUDE.md "What aiwf commits to" §2 collapses to a single uniform rule. Doc-tree narrow-id sweep over `docs/pocv3/design/` and `docs/pocv3/plans/` (~70 references) with allowlist for foreign-project / illustrative / historical mentions. Cross-repo skill refresh: kernel-embedded fixture at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` matches the rituals plugin commit `808ad70bb368c7d687a207cc7b749e0b11529323`.

A follow-up `wf-patch` (`fix/rewidth-preflight-checks`) hardens M-0082's verb with a default-on preflight: `aiwf rewidth --apply` runs `aiwf check` first and refuses to proceed on error-severity findings (`--skip-checks` opts out); missing expected kind directories emit advisory warnings; all-missing bails with `exitUsage`. Discovered during epic dogfooding when the first `--apply` attempt revealed the absence of any built-in readiness check beyond dry-run.

### Changed — Trunk promotion (poc/aiwf-v3 → main)

The `poc/aiwf-v3` branch graduates to `main` on 2026-05-08. After this point, the engine, planning state, and design research all live in a single trunk. Future releases tag on `main` (next: v0.7.0); the `poc/aiwf-v3` branch is preserved as a frozen historical reference for the PoC's full git history.

The merge brings the design research previously isolated on `main` into the same trunk as the implementation:

- **Research arc** at `docs/research/` — `KERNEL.md`, `0-introduction.md`, and thirteen numbered docs (`00`–`13`) tracing the project from the original event-sourced ambition through to policies-as-primitive.
- **Working paper** at `docs/working-paper.md` — thesis-style synthesis; one-document entry point for visitors.
- **Explorations** at `docs/explorations/` — `01`–`05` on policies as a primitive, plus mining-corpus surveys at `surveys/{flowtime,liminara}/` and landscape surveys at `docs/research/surveys/`.
- **Pre-PoC design archive** at `docs/archive/` — original `architecture.md` and `build-plan.md` preserved for archaeology.
- **Repo conventions** carried from main: `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`, `.github/workflows/pr-conventions.yml`. The trunk-based development model in `CLAUDE.md` is the primary workflow; PR ceremony applies to external contributions. `CONTRIBUTING.md` awaits a reconciliation pass.

### Added — E-21: Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md

Three milestones (M-078, M-079, M-080) graduate the open-work synthesis pattern from a one-off `critical-path.md` snapshot into the reproducible `aiwfx-whiteboard` skill. The skill ships in the `ai-workflow-rituals` plugin, materialises into the marketplace cache via `aiwf init` / `aiwf update`, and answers natural-language direction questions (*"what should I work on next?"*, *"draw the whiteboard"*, *"where should we focus?"*) by synthesising tree state into a tiered landscape, recommended sequence, first-decision fork, and Q&A-gated pending decisions. Files **ADR-0007** (placement, tiering, name rationale; status `proposed`). Adds two CLAUDE.md doctrines mid-flight: *AC promotion requires mechanical evidence* (born from M-078's "wrapped without tests" episode) and *Cross-repo plugin testing* (fixture-first authoring at `internal/policies/testdata/<skill>/SKILL.md`; deploy to rituals repo at wrap; drift-check test against marketplace cache). Retires `work/epics/critical-path.md` and the standing `unexpected-tree-file` warning it produced.

- **M-078 — Planning-conversation skills design ADR.** ADR-0007 captures placement (rituals plugin, not kernel), tiering (pure-skill first; kernel verb only when usage demands it), and name rationale (`aiwfx-whiteboard`; the deferred kernel-verb backing is `aiwf whiteboard`, not `aiwf landscape`).
- **M-079 — `aiwfx-whiteboard` skill: classification rubric, output template, Q&A gate.** Skill body covers input shape (which read verbs to call and in what order), tier classification rubric, output template, Q&A flow, and anti-patterns. Output ordering: action-shaped blocks (sequence / fork / pending) lead the rendered output; tiered landscape moves to last as supporting reference.
- **M-080 — Whiteboard skill fixture validation; retire `critical-path.md`; close E-21.** Fixture at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` is the canonical authoring location; `TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent` re-applies the kernel skill-coverage invariants to the plugin skill (kernel policy walks `internal/skills/embedded/` only — see G-088), and `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` guards against fixture/cache drift.

Three follow-up gaps carry loose ends forward: **G-088** (kernel skill-coverage policy doesn't police plugin skills), **G-089** (whiteboard gitignored cache anti-pattern revision — *addressed* mid-epic), **G-090** (drift-check test has three branches not unit-tested; refactor lookup to take cache root as parameter for hermetic testing).

### Added — E-22: Planning toolchain fixes (closes G-071, G-072, G-065)

Three milestones (M-075, M-076, M-077) close the Tier-1 planning-toolchain frictions whose cumulative tax on every multi-milestone planning session justified bundling them. Closes **G-071**, **G-072**, and **G-065**.

- **M-075 — Lifecycle-gate the `entity-body-empty` rule (closes G-071).** The check now skips pre-implementation drafts and post-terminal artifacts: per-kind status gating exempts entities still being scoped (e.g. `draft` epics) and entities past their useful life (e.g. `done` milestones, `cancelled` gaps). Drops the kernel repo's warning baseline from 46 to ~1. Adds the shared `entity.IsTerminal(kind, status)` helper used by E-20/M-072 as well.
- **M-076 — Writer surface for milestone `depends_on` (closes G-072).** Closes the kernel asymmetry where `depends_on` had six read sites and zero writers. New `--depends-on` flag on `aiwf add milestone` lands the field at create time; new dedicated verb `aiwf milestone depends-on M-NNN --on M-AAA --on M-BBB` (with `--clear` to drop the field) edits it post-creation. Both produce trailered atomic commits with referent validation. Verb shape is forward-compatible with G-073's eventual cross-kind generalisation. **G-079** filed for `aiwfx-plan-milestones` plugin-skill documentation update upstream.
- **M-077 — `aiwf retitle` verb for entities and ACs (closes G-065).** New verb `aiwf retitle <id> --to "<new title>"` corrects frontmatter `title:` for top-level kinds and composite-id ACs in a single trailered commit, so scope refactors no longer leave `title:` permanently misleading. Ships with a dedicated `aiwf-retitle` embedded skill plus a redirect from `aiwf-rename` (slug rename remains the rename verb's job). README.md updated to add the two new verbs and refresh the materialized-skills count to twelve.

### Added — E-20: Add list verb (closes G-061)

Three milestones (M-072, M-073, M-074) ship `aiwf list` as the AI's hot-path read primitive over the planning tree, route AI discovery to it via a split-skill design that demotes `aiwf status` to its real role (human-curated narrative), and lock the discoverability surface against drift via a kernel policy. Closes **G-061** and **G-085**. Files **ADR-0006** (skills policy: per-verb default, topical multi-verb when concept-shaped, no skill when `--help` suffices, discoverability priority justifies splitting). Adds CLAUDE.md *Skills policy* section.

- **M-072 — `aiwf list` verb + status filter-helper refactor + contract-skill drift fix.** New verb `aiwf list` with V1 flags `--kind`, `--status`, `--parent`, `--archived`, `--format=text|json`, `--pretty`. Default behavior: filter out terminal-status entities (forward-compatible with ADR-0004's archive convention). Default sort: id ascending. No-args invocation prints per-kind counts. `--format=json` emits the standard envelope with `result` as an array of `{id, kind, status, title, parent, path}` summaries. Closed-set completion wired for `--kind` and `--status` (kind-aware). Shared filter helper `tree.FilterByKindStatuses` extracted from `cmd/aiwf/status_cmd.go` so list and the *Open gaps* / *In-flight* / *Open-decisions* slices of `aiwf status` cannot drift. New `entity.IsTerminal(kind, status)` helper. Seven `aiwf list contracts` references in `docs/pocv3/plans/contracts-plan.md` and the `aiwf-contract` skill swept to `aiwf list --kind contract`; `TestNoReintroducedDeadVerbForms_ContractsAndSkill` is the future-drift guard.
- **M-073 — `aiwf-list` skill + `aiwf-status` skill tightening.** New embedded skill `internal/skills/embedded/aiwf-list/SKILL.md` with description densely populated by list-shaped natural-language phrasings; body covers filter recipes, output shape, JSON envelope, and when-to-use-list-vs-status criteria. The `aiwf-status` skill body opens with a bold-paragraph redirect to `aiwf list` for tree queries; description tightened to narrative-snapshot phrasings only. Both skills materialize via `aiwf init` and `aiwf update`.
- **M-074 — Skill-coverage policy + judgment ADR + CLAUDE.md skills section.** New kernel policy `internal/policies/skill_coverage.go` asserts every embedded skill has non-empty `name:`/`description:` frontmatter, skill `name:` matches its directory and the `aiwf-<topic>` convention, every top-level Cobra command is documented by some embedded skill or appears in the allowlist with one-line rationale, and every backticked `aiwf <verb>` mention inside a skill body resolves to a real registered verb. Subsumes G-061's `skill-references-unknown-verb` follow-up suggestion. ADR-0006 captures the judgment rule; CLAUDE.md *Skills policy* section points at the ADR (judgment) and the policy (mechanical companion); the *What's enforced and where* table gains a row pinning the policy to its CI test chokepoint. **G-087** filed as the deferred `aiwf-show` skill follow-up referenced by the allowlist's `show` entry rationale.

### Added — E-18: Operator-side dogfooding completion (closes G-062, G-064)

Two milestones (M-070, M-071) close the operator-side gap that G-038 left partial: the kernel repo's design assumed `aiwf-extensions` and `wf-rituals` were present, but neither plugin was installed for this project's scope, and there was no kernel mechanism to detect that. Closes **G-062** and **G-064**.

- **M-070 — `aiwf doctor` warning for missing recommended plugins.** Config-driven `doctor.recommended_plugins` check warns once per declared-but-missing plugin against `~/.claude/plugins/installed_plugins.json`. Generic mechanism — any consumer can declare expected plugins in `aiwf.yaml` and get the warning when state doesn't match.
- **M-071 — Install ritual plugins in kernel repo + document operator setup.** This repo declares both plugins in `aiwf.yaml`, project-scope-installs them, and CLAUDE.md gains an *Operator setup* section documenting the install path — including the project-scope-vs-user-scope nuance discovered during the install (the CLI form `claude /plugin install <name>@<marketplace>` defaults to user scope; only the interactive `/plugin` menu offers a project-scope choice).

Two deferrals filed as gaps: **G-069** (`aiwf init`'s `printRitualsSuggestion` hardcodes the user-scope CLI form) and **G-070** (`aiwf doctor` has no `--format=json` envelope).

### Added — E-17: Entity body prose chokepoint (closes G-058)

Three milestones (M-066, M-067, M-068) make non-empty body prose a kernel-enforced property across every entity kind, not just ACs. Closes **G-058**. Captures **D-001** (asymmetric semantics: top-level sections count sub-headings as content; AC bodies require non-heading prose).

- **M-066 — `aiwf check` finding `entity-body-empty`.** New rule lights up empty load-bearing body sections per kind: warning by default, error under `aiwf.yaml: tdd.strict: true`. Per-kind body-section dispatch with the asymmetric semantics call codified in D-001 — top-level sections count sub-headings as content, AC bodies require leaf prose.
- **M-067 — `aiwf add ac --body-file` for in-verb body scaffolding.** `aiwf add ac M-NNN` now accepts `--body-file <path>` (positional pairing for batched ACs, stdin shorthand for single ACs); leading `---` is refused across both forms. AC body content lands in the same atomic create commit as the heading, completing the body-file surface across every `aiwf add` shape.
- **M-068 — `aiwf-add` skill names "fill in the body" as required next step.** The embedded skill cross-references M-066 and M-067 so operators reading the skill alone get the full non-empty-body picture — rule, verb, workflow — without grepping source.

Two follow-up gaps survive the wrap: **G-067** (wf-tdd-cycle is LLM-honor-system advisory under load) and **G-068** (discoverability policy misses dynamic finding subcodes).

### Added — E-14: Cobra and completion

Eight milestones (M-049 through M-055 plus M-061) replace the stdlib-`flag` dispatch with [`github.com/spf13/cobra`](https://github.com/spf13/cobra), ship `aiwf completion bash|zsh|fish|powershell`, and wire shell tab-completion for every value-taking flag and id positional. New CLAUDE.md principle: **CLI surfaces must be auto-completion-friendly** — mechanically enforced by a drift test that fails CI when a flag lands without completion wiring or an opt-out entry.

- **M-049 — Bootstrap Cobra dispatch + migrate `version`.** `cmd/aiwf/main.go` becomes a Cobra root command tree; `version` is the first natively-Cobra subcommand. Other verbs continue through a `newPassthroughCmd` adapter until their own milestone migrates them. Exit codes 0/1/2/3 are preserved across Cobra's `Execute` boundary via a typed `*exitError` shuttle.
- **M-050 — Read-only verbs migrated.** `check`, `history`, `doctor`, `schema`, `template`, `render` are now native Cobra commands. `render` keeps its dual surface (`render roadmap` subcommand vs `render --format=html` parent flag) so the public CLI shape doesn't break consumer scripts.
- **M-051 — Mutating verbs migrated.** `add` (with `add ac` as a Cobra subcommand sharing PersistentFlags), `promote`, `cancel`, `rename`, `edit-body`, `move`, `reallocate`, `import`. Single-commit-per-verb invariant preserved; trailer keys (`aiwf-verb` / `aiwf-entity` / `aiwf-actor` plus the I2.5 provenance set) byte-identical; repo-lock contract unchanged.
- **M-052 — Setup verbs migrated.** `init`, `update`, `upgrade`. Marker-based artifact regeneration (skills under `.claude/skills/aiwf-*`, hooks under `.git/hooks/`) goes through the same `initrepo.Init` and `initrepo.RefreshArtifacts` entry points as before, so installed hooks stay byte-identical.
- **M-053 — `aiwf completion <bash|zsh|fish|powershell>` + static completion.** Cobra's auto-generated completion verb emits sourceable shell scripts; the README install one-liner is `source <(aiwf completion zsh)`. Static `--format=text|json` completion across read-only verbs; kind enumeration on `aiwf add`, `aiwf schema`, `aiwf template`; per-kind status enumeration on `aiwf promote <id> <TAB>` derived from the id's prefix without loading the tree; closed-set `--phase` (red|green|refactor|done) and `--on-collision` (fail|skip|update); `add ac M-NNN` filtered to milestone ids.
- **M-054 — Dynamic id completion + drift-prevention test.** `--epic`, `--discovered-in`, `--relates-to`, `--linked-adr`, `--by`, `--superseded-by` enumerate live entity ids; positionals on `history`, `promote`, `cancel`, `rename`, `edit-body`, `move`, `reallocate` likewise. Failures (no `aiwf.yaml`, malformed tree) collapse to an empty list rather than spamming the shell. The drift test in `cmd/aiwf/completion_drift_test.go` walks the Cobra tree and asserts every value-taking flag has either a completion function bound or an entry in the curated opt-out list with a one-line rationale.
- **M-055 — Documentation pass.** Every migrated verb's `--help` now carries a Cobra `Examples:` block with concrete copy-pastable invocations. CLAUDE.md § Go conventions § CLI conventions names Cobra as the standard CLI library and points to the `runXCmd` helper / drift-test chokepoint pattern.
- **Follow-up cleanup commit — `show`, `status`, `whoami`, `authorize` migrated** (the four verbs the epic Scope didn't enumerate). Each gains native Cobra dispatch + completion wiring; the drift test's opt-out list shrinks accordingly.
- **M-061 — Contract family migration + help-recursion regression test.** `aiwf contract` becomes a native Cobra command tree: `verify`, `bind <C-id>`, `unbind <C-id>`, `recipes`, plus the `recipe` sub-tree (`show`, `install`, `remove`). `--validator` completes from declared validators in `aiwf.yaml`; `recipe show|install` from the embedded recipe set; `recipe remove` from declared validators; `bind`/`unbind` positionals from contract entity ids. Subprocess integration test exercises every subcommand against the migrated binary. Bundled into this milestone: a regression test pinning the SetHelpFunc inheritance fix (a `c.Help()` re-entry that recursed to stack overflow on `aiwf <subverb> --help` until M-053 — the test caught a still-live instance of the same bug on `render roadmap --help` during authoring), and this very `[Unreleased]` retrofill.

User-visible behavior delta: `--help` output for every verb now uses Cobra's standard rendering (Long description + Examples + Usage + Flags blocks), replacing the hand-rolled per-verb usage strings. Exit codes, JSON envelope shape, single-commit-per-verb, and trailer-key behavior are byte-identical to pre-migration.

Auto-completion install (one line in shell rc):

```bash
# zsh
source <(aiwf completion zsh)

# bash (requires bash-completion v2)
source <(aiwf completion bash)
```

After sourcing, `aiwf <TAB>` lists the verb catalog; `aiwf promote E-01 <TAB>` lists the kind's allowed statuses; `aiwf check --format=<TAB>` lists `text|json`; `aiwf add milestone --epic <TAB>` enumerates live epic ids from the planning tree.

## [0.6.0] — 2026-05-06

### Added — E-15: Reduce planning-verb commit cardinality

Five milestones cumulatively cut a planning session's commit count by ~75% and remove the last skill/check policy contradictions around body editing. Closes **G-051**, **G-052**, **G-053**, and **G-054**.

- **M-056 — `--body-file` flag on `aiwf add` for all six kinds.** Pass `aiwf add <kind> --title "..." --body-file <path>` (or `--body-file -` for stdin) to land body prose in the same atomic commit as the new frontmatter, replacing the per-kind default template. Eliminates the create-then-hand-edit pattern that today triggers `provenance-untrailered-entity-commit` warnings on the body-edit commit. The file must contain body content only — leading `---` (frontmatter delimiter) is refused so the create commit can't accidentally produce a double-frontmatter file.
- **M-057 — Batched `--title` on `aiwf add ac`.** `aiwf add ac M-NNN --title "..." --title "..." --title "..."` now creates N acceptance criteria in one atomic commit instead of one commit per AC. The commit emits one `aiwf-entity:` trailer per created composite id (allocation order), so `aiwf history M-NNN/AC-X` finds the batch commit for any AC in the batch. Whole-batch validation: if any title is empty or prosey, the entire batch aborts before disk work. `--tests` is rejected when N>1 (a single test-metrics value can't apply unambiguously to multiple ACs). Single-`--title` invocation continues to work unchanged — same subject shape, same single trailer.
- **M-058 — `aiwf edit-body` verb + skill reconciliation.** New verb `aiwf edit-body <id> --body-file <path>` (or `--body-file -` for stdin) replaces the markdown body of an existing entity in a single atomic trailered commit. Frontmatter is left untouched — that stays the domain of `promote` / `rename` / `cancel` / `reallocate`. Refuses leading `---` content and composite ids (AC body sub-section editing deferred). The aiwf-add skill no longer carves out plain-git body edits; every entity-file mutation now goes through a verb route, and `provenance-untrailered-entity-commit` only fires on accidental hand-edits as designed.
- **M-059 — Resolver-pointer flags on `aiwf promote`.** New `--by`, `--by-commit`, and `--superseded-by` flags write the matching frontmatter field atomically with the status change, so `aiwf promote G-NNN addressed --by M-007` (or `--by-commit <sha>`) and `aiwf promote ADR-NNNN superseded --superseded-by ADR-MMMM` no longer require a follow-up hand-edit to satisfy the `gap-resolved-has-resolver` and `adr-supersession-mutual` checks. Flags reject mismatched kind/target-status combinations and are mutex with `--audit-only` (which is empty-diff by definition).
- **M-060 — Bless-current-edits mode for `aiwf edit-body`.** Running `aiwf edit-body <id>` with no `--body-file` flag now commits whatever the user has edited in the working copy of the entity file, with the standard `edit-body` trailer set. Refuses cleanly when there is no diff, when the diff includes frontmatter changes (pointer at `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate`), or when the file has no HEAD version (pointer at `aiwf add`). The existing `--body-file <path>` (and stdin) mode from M-058 stays exactly the same — same trailers, same atomicity, same rules; only difference is what happens when the flag is absent. AC body sub-section edits work for free via bless mode on the parent milestone — no composite-id resolver needed.

## [0.5.2] — 2026-05-06

### Fixed
- **G50 — Pre-commit hook tolerant of gitignored `STATUS.md`.** The hook regenerates `STATUS.md` and stages it via `git add`; when the consumer has gitignored the file (legitimate — it's regenerated every commit), `git add` exited non-zero and `set -e` aborted the entire hook, in violation of the hook header's "tolerant by design — never blocks commits" promise. The aborted commit also commonly orphaned `.git/index.lock`, masquerading as an aiwf bug on subsequent verbs. Fix: append `2>/dev/null || true` to the `git add` invocation. Consumer migration: run `aiwf update` after upgrading. (`572bc96`)
- **G48 — `aiwf init`, `aiwf update`, and `aiwf doctor` honor `core.hooksPath`.** A consumer who has set git's `core.hooksPath` (a tracked-hooks pattern via husky/lefthook or a home-grown convention) previously got hooks installed at `.git/hooks/` regardless — git's hook lookup missed them and the validation chokepoint silently disappeared. New `gitops.HooksDir` helper resolves the effective hooks directory once; init, update, and doctor read through it. Reports (`StepResult.What`, doctor lines, migration messages) reflect the actual install path. Consumer migration: run `aiwf update` after upgrading. (`6432f0f`)

## [0.5.1] — 2026-05-06

### Added
- **G49 — `addressed_by_commit:` field on the gap kind.** Optional multi-string list of commit SHAs that resolved a gap. The `gap-resolved-has-resolver` rule now passes when *either* `addressed_by:` (entity refs) or `addressed_by_commit:` (commit SHAs) is non-empty. Models the real semantic: a gap can be closed by a specific commit, not just a milestone. (`de39e01`)

### Internal
- **G38 — Kernel now dogfoods aiwf against itself.** The kernel repo became an aiwf consumer: 47 legacy gaps imported as `G-NNN` entities, 13 epics + 48 milestones imported from the historical PoC plan, `gaps.md` and `poc-plan.md` archived under `docs/pocv3/archive/`. The hook reshape (drop `core.hooksPath`, slim the kernel's policy-lint hook to `.git/hooks/pre-commit.local` via the G45 chain) treats the kernel like any consumer with pre-existing hooks. End-to-end exercise of `aiwf init`, `aiwf import`, `aiwf add gap`, `aiwf promote`, `aiwf promote --audit-only`, and the pre-commit/pre-push hook chains against real consumer-shaped state. New gaps surfaced and filed via the framework: G48 (`aiwf init` should honor `core.hooksPath`), G49 (above, fixed in this release).

## [0.5.0] — 2026-05-05

### Added
- **G46 — Structured remediation on `go install` package-path failures.** When a release relocates the cmd package within the module (as v0.4.0 did), `aiwf upgrade` now detects the Go toolchain's "module found but does not contain package" stderr and prints a hint pointing at the CHANGELOG plus the manual `go install <new-path>@<target>` recovery command. The v0.3.x → v0.4.0 transition pain that surfaced this gap is the canonical example; v0.5.0+ binaries handle the next path-change gracefully. (`93a3e2b`)

### Changed (breaking)
- **G47 — `aiwf_version` field retired from `aiwf.yaml`.** The field was a set-once pin that produced chronic doctor noise without serving its intended purpose. Removed via the same pattern as the I2.5 legacy-actor-strip: loader becomes tolerant (no longer required), `aiwf init` no longer writes it, `aiwf update` strips it on every refresh, doctor's `pin:` row goes away. Pre-G47 yamls load fine and surface a one-line deprecation note until `aiwf update` cleans them. The running binary's version is now the authoritative answer (`aiwf version`); newer-release detection lives at `aiwf doctor --check-latest`. (`25bf5ea`)

## [0.4.0] — 2026-05-05

### Changed (breaking)
- **Repo reorg to Go-standard layout.** `tools/cmd/aiwf` → `cmd/aiwf`, `tools/internal/...` → `internal/...`, `tools/e2e` → `e2e`. Module path unchanged at `github.com/23min/ai-workflow-v2` *(at the time of this release; subsequently renamed — see Unreleased)*. **Install path changed:** `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest` (previously `.../tools/cmd/aiwf@latest`). Existing tags resolve via the old path; new releases via the new path. `tools/CLAUDE.md` merged into root `CLAUDE.md` as a new "Go conventions" section so discovery semantics work post-reorg. (`a137132`)

### Added
- **G45 — Hook chaining via `.local` siblings.** aiwf-managed `pre-push` and `pre-commit` hooks now invoke `<hook-name>.local` (if present and executable) before running aiwf's own check. `aiwf init` auto-migrates a pre-existing non-marker hook to its `.local` sibling, preserving content byte-for-byte and exec bit. Non-executable `.local` fails loud (both at hook runtime and via `aiwf doctor`). New `ActionMigrated` step result; `HookConflict` now signals only the rare `.local`-already-exists collision. `aiwf doctor` reports chain shape per hook. Unblocks consumers with pre-existing hooks (husky, lefthook, hand-written) from `aiwf init` friction. (`49e7764`)

## [0.3.0] — 2026-05-05

### Added
- **G37** — Cross-branch id collisions are detectable and resolvable. Trunk-aware allocator + `prior_ids` lineage; `aiwf history` walks renamed entities through their full chain; `aiwf check` flags cross-tree collisions; `aiwf reallocate` handles the trunk-ancestry tiebreaker. (`271f514`, `b9d73d8`, `c5a98c1`, `a6e8067`, `685f288`)
- Three new policies + 14 backfilled subcode docs to broaden discoverability-lint coverage. (`2b094e3`)
- **G38** — Dogfooding gap filed: investigation of running aiwf against its own kernel repo. Open. (`dd25c06`)
- **G40** — `aiwf check` now reports `unexpected-tree-file` for stray files under `work/`. Tree-shape changes go through verbs; body-prose edits to existing entity files remain free-form. Configurable via `aiwf.yaml: tree.allow_paths` (glob exemptions) and `tree.strict: true` (promote warning to error). Files inside contract directories are auto-exempt. New design doc `docs/pocv3/design/tree-discipline.md`; rule folded into `aiwf-add` and `aiwf-check` skills (no new skill). (`bdd43c2`)
- **G41** — Tree-discipline now runs at pre-commit *and* pre-push. New `aiwf check --shape-only` flag runs only the tree-discipline rule (no trunk read, no provenance walk, no contract validation), wired into the aiwf-managed pre-commit hook. The LLM gets an in-loop signal when a stray file lands, regardless of which AI client it's using — git hooks are agent-agnostic. (`fb2e1e4`)
- **G42** — Pre-commit hook responsibilities decoupled. The tree-discipline gate now installs unconditionally when aiwf is adopted in the repo; `aiwf.yaml: status_md.auto_update` controls only whether the script body includes the STATUS.md regen step. Opting out of the regen no longer removes the gate. `aiwf doctor`'s pre-commit reporting updated with new "ok, gate-only" healthy state.

### Changed
- `aiwf upgrade` prints a concrete recovery path (`$GOBIN`, `$GOPATH/bin`, or `$HOME/go/bin`) when post-install binary lookup fails, instead of a generic "run aiwf update manually" message. (`9a06c74`)

### Fixed
- **G39** — `aiwf upgrade` mis-parsed `go env` output when GOBIN was unset (the default Go install setup), failing immediately after `go install` succeeded. The post-install lookup now queries GOBIN and GOPATH in separate calls so there is no multi-line shape to mis-parse. (`9a06c74`)

## [0.2.3] — 2026-05-04

### Fixed
- **G35 / G36** — HTML render now generates pages for every entity kind (gap, ADR, decision, contract — previously 404'd) and renders entity-body markdown as HTML instead of escaped raw text. (`d1bf1e1`)

## [0.2.2] — 2026-05-04

### Fixed
- **G30** — `aiwf status` and `aiwf history` no longer pick up prose-mention false positives from `git log --grep`. (`7141f2a`)
- **G31 / G32 / G33** — Squash-merge defeating the trailer-survival contract; merge commits bypassing the untrailered-entity audit; `aiwf doctor --self-check` not exercising the manual-commit-recovery path. (`ad1175c`)
- **G34** — Mutating verbs no longer sweep pre-staged unrelated changes into their commit (uses a stash for isolation). (`890ab01`)

## [0.2.1] — 2026-05-03

### Fixed
- `aiwf check` provenance UX: per-entity findings, defined audit scope, `--since` flag (#5). (`c5b6ab7`)

## [0.2.0] — 2026-05-03

### Added
- **HTML render (Iteration I3)** — `aiwf render --format=html` produces a static site with full per-kind templates (epic, milestone, gap, ADR, decision, contract), a Linear-leaning palette, dark mode, content-hashed CSS for cache-busting, sidebar with logo and wordmark, left-side navigation panel, and a render report under `aiwf doctor`. Playwright browser tests cover the end-to-end render. (`6730c1a`, `e3977ad`, `cce0c21`, `9b88108`, `d9183d8`, `606bfab`)
- **`aiwf-tests` commit trailer** — opt-in TDD enforcement for milestones; new `acs-tdd-tests-missing` warning. (`d7fd072`, `77ccfb1`)
- JSON-completeness on `aiwf show`. (`7fd6524`)
- Testing rules in `tools/CLAUDE.md`: substring-vs-structural assertions, human verification, untested-branches policy, no-workaround rule. (`a30c509`)

### Changed
- `aiwf init` / `aiwf update` reconcile `.gitignore` for the html `out_dir`. (`056139d`)
- Release-prep polish: legacy actor stripped on update; `aiwf render --help`; aiwf-render skill; menu reordering; README clarification. (`406ac48`)
- Integrated `aiwf status` as a rendered HTML page. (`44ea40b`)

### Fixed
- **G27 / G28 / G29** — Test-the-seam, contract-driven, and spec-sourced testing gaps closed; binary-level integration tests retrofitted across version-related verbs. (`f810a86`)
- I3 audit follow-up: three render bugs + structural test additions. (`f71949f`)

## [0.1.1] — 2026-05-03

### Fixed
- `aiwf version` verb now reflects `version.Latest()` proxy resolution — the v0.1.0 seam regression that motivated G27/G28/G29. (`32672cd`)

## [0.1.0] — 2026-05-03

Initial PoC release. Six entity kinds; stable ids that survive rename, cancel, and collision; `aiwf check` as the validation chokepoint; marker-managed framework artifacts; structured commit trailers; principal × agent × scope provenance.

### Added
- **`aiwf upgrade` verb** — one-command flow with skew detection, install via `go install`, and re-exec into `aiwf update`. Includes `--check`, `--version=`, and friendly messages on missing `go` or proxy-disabled. Wired into `aiwf doctor --self-check`. (`3e2d7ff`, `d1c4b1c`, `6136754`, `efa59c2`)
- **`version` package** — `Current()`, `Compare()`, `Latest()` against the Go module proxy. (`62928e5`, `05dd773`)
- **G1 – G26** resolved across iterations I0–I2. See [`docs/pocv3/archive/gaps-pre-migration.md`](docs/pocv3/archive/gaps-pre-migration.md) for the full matrix.
