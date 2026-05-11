---
id: M-0081
title: Canonical 4-digit IDs in parser, renderer, and allocator
status: done
parent: E-0023
tdd: required
acs:
    - id: AC-1
      title: Allocator emits canonical 4-digit ids for every kind
      status: met
      tdd_phase: done
    - id: AC-2
      title: Parser tolerates both widths at every audited call site
      status: met
      tdd_phase: done
    - id: AC-3
      title: Every display surface emits canonical ids regardless of filename
      status: met
      tdd_phase: done
    - id: AC-4
      title: Pre-existing narrow-width trailers match canonical-id queries
      status: met
      tdd_phase: done
    - id: AC-5
      title: Test-fixture sweep canonicalizes hardcoded narrow ids in test code
      status: met
      tdd_phase: done
    - id: AC-6
      title: aiwf check on this repo's pre-rename tree is green
      status: met
      tdd_phase: done
---
## Goal

Make the kernel's id width policy uniform: parser accepts both narrow and canonical widths on input; renderer canonicalizes every display surface to 4-digit form; allocator emits 4-digit form for every kind. Hardcoded narrow-width ids in the test suite update to canonical form. After M-A ships, every consumer's existing tree (still on narrow-width filenames) continues to validate; new allocations are canonical; display is uniform.

The change is pure-additive at the parser layer — acceptance widens, no existing valid input becomes invalid. Old trees, branches, skills with hardcoded narrow widths keep working indefinitely.

## Context

ADR-0008 names this milestone as the load-bearing kernel change before the migration verb (M-B) and the drift check (M-C) can ship. The migration verb depends on parser tolerance to read existing trees; the drift check depends on the allocator emitting canonical ids so that "mixed-state tree" is a meaningful signal.

Today's policy is encoded in `internal/verb/import.go::canonicalPadFor`, which returns 2 for epic, 4 for ADR, and 3 for everything else. This milestone unifies it: every kind returns 4. The renderer and the allocator both consume `canonicalPadFor`; consolidating the policy into a single source of truth is implicit in the change.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **Pure-additive parser change.** The parser's accept set widens; no existing valid input becomes invalid. AC-2 asserts both-widths-equivalence; AC-6 asserts this repo's existing tree continues to validate.
- **No git history rewrite.** Old commit trailers (`aiwf-entity: E-22`) keep matching via parser tolerance; the kernel never re-emits narrow-width ids in new commits. AC-4 covers this.
- **Single source of truth for pad width.** The allocator and renderer both consume `canonicalPadFor`; if M-A introduces a duplicate (a new pad-width helper alongside the existing one), the next milestone's TDD pass catches the drift. AC-1 covers this.
- **Composite ids included.** `M-NN/AC-N` and `M-NNNN/AC-N` parse equivalently; the existing composite-id parser is one of the audited sites. AC-2 covers composite cases.
- **TDD: required.** Each AC drives a red→green→refactor cycle. AC-5 (test-fixture sweep) is content edit (no new logic) but its assertion is mechanical: a structural grep that returns matches only inside an allowlist.

## Design notes

### Pre-implementation: id-parsing call site audit

The TDD pass starts with a grep-derived enumeration of every id-parsing call site in the kernel. Initial list (committed to this section as the work begins; updated if new sites surface during implementation):

- `internal/entity/` — `ParseID`, `Split` (composite-id parser for `M-NN/AC-N`), frontmatter forward-ref validators for `parent`, `depends_on`, `linked_acs`, `linked_entities`, `discovered_in`, `linked_adr`, `superseded_by`, `waived_by`.
- `internal/gitops/` — `ParseTrailers` (extracts ids from `aiwf-entity:` trailer values).
- `internal/check/` — `refsResolve` and any rule that parses ids from frontmatter or body content.
- `internal/render/` — anchor/link generators for HTML output; id-bearing fields in any rendered envelope.
- `internal/verb/` — id-handling in verbs that take ids as positional args (`promote`, `cancel`, `show`, `history`, `reallocate`, `authorize`, `edit-body`, `retitle`, `rename`).
- `cmd/aiwf/` — Cobra completion functions emitting/accepting entity ids.

Each call site is exercised by a table-driven test in M-A's TDD pass with both narrow and canonical inputs, asserting equivalent resolution.

### Canonical pad width unification

`canonicalPadFor(kind)` currently lives in `internal/verb/import.go` (with a duplicate formatting site in `internal/entity/AllocateID`). This milestone consolidates to a single source — preferably `internal/entity/` since pad width is an entity-level property, not a verb-specific one. The verb-side function becomes a thin re-export or is deleted; verbs consume the entity-side helper directly.

If consolidation is more disruptive than the milestone's scope wants, both sites update consistently to return 4 for every kind, with a short follow-up gap to consolidate later. The duplication is currently silent but it's the kind of footgun that bites M-C's drift check — easier to fix here.

### Renderer canonicalization scope

The renderer reads on-disk filenames (which may be narrow) but rewrites display output to canonical width. Two cases:

- **Id appears in structural output** (`aiwf show`'s frontmatter render, JSON envelope, HTML anchor): pass the id through a `canonicalize(id)` helper that left-pads to canonical width. Helper lives in `internal/entity/`.
- **Id appears in body prose** (markdown content, e.g., `aiwf show`'s rendered body): the body content is what the user wrote on disk; the renderer doesn't rewrite prose. Body-content canonicalization happens in M-B's `aiwf rewidth` verb, not at render time.

The distinction matters: M-A canonicalizes *structural* surfaces only. Body content stays as authored until M-B rewrites it.

## Surfaces touched

- `internal/entity/` — `AllocateID`, `ParseID`, composite-id parser (`Split`), frontmatter validators, new `canonicalize(id)` helper.
- `internal/gitops/` — `ParseTrailers` (trailer-value tolerance).
- `internal/check/` — `refsResolve` and any rule with id-parsing.
- `internal/render/` — display-surface canonicalization.
- `internal/verb/` — verbs that take id arguments (most consume entity helpers; few or no direct changes expected).
- `cmd/aiwf/` — Cobra completion functions.
- `internal/**/*_test.go`, `internal/**/testdata/`, `cmd/aiwf/*_test.go` — fixture sweep to canonical width.
- `internal/policies/` — possibly a new policy test asserting the both-widths-equivalent invariant if the drift-prevention shape fits cleanly.

## Out of scope

- The migration verb `aiwf rewidth` — that's M-B.
- The drift-check rule `entity-id-narrow-width` — that's M-C.
- File renames in this repo's `work/` and `docs/adr/` trees — those happen when `aiwf rewidth --apply` runs in M-B's wrap.
- ADR-0003 amendment (F-NNN → F-NNNN) — that's M-C.
- CLAUDE.md commitment #2 update — that's M-C.
- Embedded skill content refresh — that's M-C.
- Rituals plugin coordination — that's M-C.
- Doc-tree narrow-id sweep (`docs/`, `README.md`, `CHANGELOG.md`) — those refresh in M-C's doc updates.
- Consolidating `canonicalPadFor` if it requires substantial restructuring beyond a thin re-export. If consolidation is non-trivial, both sites update consistently with a follow-up gap.

### AC-1 — Allocator emits canonical 4-digit ids for every kind

`internal/entity/AllocateID` (and its `canonicalPadFor` helper or successor) returns canonical 4-digit ids for every entity kind. New ids allocated via `aiwf add <kind>` produce filenames at canonical width and frontmatter `id:` values at canonical width.

Verified by table-driven test: for each kind in `entity.Kind`, a fresh repo state with the previous high-water mark at sequence N is set up; `aiwf add <kind> --title "..."` is invoked; the resulting filename is asserted to be `<prefix>-<N+1 padded to 4 digits>-...md` and the frontmatter `id:` is asserted to match.

The `canonicalPadFor` source(s) of truth — currently `internal/verb/import.go` and any duplicate in `internal/entity/` — return 4 for every kind. If consolidation reduces them to one source (preferred), the test exercises that single helper; otherwise the test asserts every source returns 4 consistently.

### AC-2 — Parser tolerates both widths at every audited call site

The pre-implementation audit (in this milestone's *Design notes — Pre-implementation: id-parsing call site audit*) enumerates every id-parsing call site. Each site is exercised by a table-driven test with both narrow (`E-22`) and canonical (`E-0022`) inputs and asserts equivalent resolution: same entity returned from `entity.ParseID`, same trailer match from `gitops.ParseTrailers`, same ref resolution from `refsResolve`, same composite parse from `M-22/AC-1` and `M-0022/AC-1`, same completion-helper output, same JSON envelope formatting.

The audit list is updated if new sites surface during implementation; the test enumeration must cover every documented site before this AC promotes to met. New sites discovered post-AC-met are tracked as gaps.

This is a pure-additive change at the parser layer: the accept set widens, no existing valid input becomes invalid. AC-6 is the load-bearing assertion that the existing tree continues to validate.

### AC-3 — Every display surface emits canonical ids regardless of filename

A fixture tree containing narrow-width files (e.g., `E-22-foo.md` on disk) is loaded; running each display surface's command produces output containing the canonical form (`E-0022`) and never the narrow form (`E-22`) in id-bearing positions.

Surfaces verified:
- `aiwf list` — id column emits canonical.
- `aiwf status` — id mentions canonical.
- `aiwf show <id>` — frontmatter render shows canonical id; body prose unchanged (per Design notes' renderer canonicalization scope).
- `aiwf history <id>` — accepts narrow input; emits canonical output.
- `aiwf render --format=html` — anchors and id-bearing structural elements canonical (verified by structural HTML assertion per CLAUDE.md "substring assertions are not structural assertions").
- JSON envelopes from `--format=json` invocations of each of the above — id fields canonical.

Body-content prose is *not* rewritten by the renderer — that's M-B's `aiwf rewidth` job. The renderer only canonicalizes structural surfaces (frontmatter renders, envelope fields, HTML anchors).

### AC-4 — Pre-existing narrow-width trailers match canonical-id queries

A synthetic git repo with a commit containing trailer `aiwf-entity: E-22` is queried via `aiwf history E-22` and `aiwf history E-0022`; both return the same commit. The query path goes through `gitops.ParseTrailers` and any width-canonicalization helper introduced in AC-2.

Table-driven test enumerates each entity kind with narrow- and canonical-form trailer values; each resolves equivalently. This AC is the load-bearing backward-compatibility assertion at the trailer layer; it complements AC-6's tree-load assertion.

The kernel never writes narrow-width trailers in new commits — `aiwf-entity:` values from the verbs are emitted at canonical form per AC-1's allocator change. Old narrow trailers in pre-existing commits remain unchanged (no history rewrite); only the read path canonicalizes.

### AC-5 — Test-fixture sweep canonicalizes hardcoded narrow ids in test code

Every hardcoded narrow-width id literal in `internal/**/*_test.go`, `internal/**/testdata/`, and `cmd/aiwf/*_test.go` is updated to canonical 4-digit form.

Verified by structural assertion: `grep -rE '"[EMGDCF]-[0-9]{1,3}-' internal/ cmd/aiwf/` returns matches only inside an explicitly-named both-widths-equivalence allowlist (AC-2's parser-tolerance tests must use narrow inputs by design — those exemptions are listed in a small allowlist file or in-test comment block, named, and committed alongside the sweep).

The allowlist's purpose is documented in a comment near each entry: *"intentional narrow-width input for AC-2 parser-tolerance test."* The test suite's behavior post-sweep is: the production paths and assertions use canonical-width inputs and outputs throughout; only the explicitly-allowlisted cases use narrow inputs to verify parser tolerance.

Doc-tree narrow-id sweep (`docs/`, `README.md`, `CHANGELOG.md`) is out of scope here — it rides with M-C's doc updates.

### AC-6 — aiwf check on this repo's pre-rename tree is green

After M-A ships and before M-B's `aiwf rewidth --apply` runs, executing `aiwf check` on this repo's existing tree — still on narrow-width filenames in `work/` and `docs/adr/` — produces zero new findings related to id widths. Parser tolerance covers the existing state: narrow filenames load correctly, refs resolve through parser tolerance, trailer history works.

Comparison: `aiwf check` finding count and codes on `main` are identical before and after M-A's PR lands, modulo any unrelated work landing on the branch. If M-A introduces any new finding code that fires on the existing tree, AC-6 is the regression guard — that's a sign the parser-tolerance change isn't actually pure-additive.

This AC is the load-bearing backward-compatibility assertion at the tree-load layer; it complements AC-4's trailer-layer assertion. Together they pin: existing trees continue to load, refs resolve, history queries return correct results, and `aiwf check` is silent about width concerns until the consumer voluntarily migrates via M-B's `aiwf rewidth`.

## Work log

Phase timeline lives in `aiwf history M-081/AC-N` for every AC; the entries below are the post-cycle outcome and the SHA of the kernel `met` commit. The production-code diff for all six ACs is bundled in this milestone's wrap commit (the kernel commits above are spec-frontmatter mutations only).

### AC-1 — Allocator emits canonical 4-digit ids for every kind

`internal/entity/allocate.go` replaced the per-kind `canonicalPad` map (E=2, M=3, ADR=4, others=3) with `const CanonicalPad = 4`; `AllocateID` formats with that constant. `internal/verb/import.go::canonicalPadFor` was deleted; `formatID` now reads `entity.CanonicalPad` directly — single source of truth per ADR-0008. Kernel met commit: `3d10790`. Tests: `TestAllocateID_CanonicalFourDigitForEveryKind`, `TestAllocateID_CanonicalAfterNarrowHighWater`, plus the existing `TestAllocateID_*` cases re-pinned to canonical-width expectations.

### AC-2 — Parser tolerates both widths at every audited call site

New `internal/entity/canonicalize.go` introduces `Canonicalize(id) string` (left-pads any recognizable id to `CanonicalPad`; composite ids recurse on the parent) and `IDGrepAlternation(id) string` (POSIX-extended regex matching both narrow and canonical-width renderings of an id, used by `git log --grep` callers reading pre-migration trailers). Lookup-seam canonicalization threaded through `internal/tree/tree.go::ByID/ByIDAll/ByPriorID/ResolveByCurrentOrPriorID/ReferencedBy/buildReverseRefs/compositeParentOrSame`, `internal/check/check.go::refsResolve/idPathConsistent/resolveCompositeRef`, `internal/check/provenance.go::RunUntrailedAudit/isEntityCoveredByLaterAudit`, `internal/contractcheck/contractcheck.go` id index, `internal/contractverify/contractverify.go::SkipIDs`, `cmd/aiwf/admin_cmd.go::readHistoryChain`, `cmd/aiwf/scopes.go` readers, and `cmd/aiwf/main.go::completeEntityIDs`. Kernel met commit: `78b5ca5`. Tests: `TestCanonicalize`, `TestIDGrepAlternation_MatchesBothWidths`, `TestIDGrepAlternation_EdgeCases`, `TestIsCompositeID_TolerantOfBothWidths`, `TestTree_ByID_AcceptsBothWidths`, `TestTree_ByPriorID_AcceptsBothWidths`.

### AC-3 — Every display surface emits canonical ids regardless of filename

`internal/htmlrender/default_resolver.go` (`IndexData`/`EpicData`/`MilestoneData`/`EntityData`/sidebar) and `internal/roadmap/roadmap.go::Render` canonicalize every emitted id. `cmd/aiwf/list_cmd.go::buildListRows`, `status_cmd.go::buildStatus`, `show_cmd.go::buildShowView/buildCompositeShowView/filterFindingsByID`, and `render_resolver.go` thread the same helper. Body prose stays as authored (deferred to M-0082's `aiwf rewidth`); filenames stay as on disk so anchor links keep resolving. Kernel met commit: `2450691`. Tests: `TestRender_HTML_CanonicalIDsFromNarrowTree` (structural assertions via `htmlElement`/`htmlSection` per CLAUDE.md), `TestList_JSON_CanonicalIDsFromNarrowTree`, `TestStatus_JSON_CanonicalIDsFromNarrowTree`, `TestShow_JSON_CanonicalIDsFromNarrowTree`.

### AC-4 — Pre-existing narrow-width trailers match canonical-id queries

`cmd/aiwf/admin_cmd.go::readHistoryChain` and the scope readers under `cmd/aiwf/scopes.go` / `show_scopes.go` / `provenance.go` use `IDGrepAlternation` so `git log --grep` finds both narrow and canonical-width trailer values; canonical-output canonicalization on the read path means a `aiwf history E-22` query returns identical events to `aiwf history E-0022`. Verb-side trailer emissions (`aiwf-entity:` values) canonicalize on every mutating verb (`promote`, `add`, `rename`, `move`, `editbody`, `auditonly`, `milestone_depends_on`, `authorize`, `import`, `ac`, `contractbind`, `contractrecipe`, `reallocate`). Kernel met commit: `c51a733`. Tests: `TestHistory_NarrowTrailerMatchesCanonicalQuery` (per-kind table), `TestHistory_NewVerbsEmitCanonicalTrailers`.

### AC-5 — Test-fixture sweep canonicalizes hardcoded narrow ids in test code

Test fixtures that asserted *expected outputs* hardcoded at narrow width swept to canonical 4-digit form (~110 modified files). Files whose narrow ids are *parser-tolerance inputs* (AC-2 / AC-4 tests, entity-grammar tests, gitops trailer-shape tests, allocator parser-tolerance tests, contractbind round-trip, skill body-prose markers, selfcheck) are listed in `internal/policies/narrow_id_sweep_test.go`'s allowlist with one-line rationale per entry. Mechanical chokepoint: `TestPolicy_NarrowIDLiteralsAllowlisted` — greps for `"[EMGDC]-[0-9]{1,3}"` literals and fails if any match falls outside the allowlist (windows skipped). Kernel met commit: `cb5ce36`.

### AC-6 — aiwf check on this repo's pre-rename tree is green

`internal/policies/this_repo_tree_clean_test.go::TestPolicy_ThisRepoTreeIsClean` loads this repo's tree via `tree.Load`, runs `check.Run`, and fails on any error-severity finding whose code is in the id-width-shaped set (`refs-resolve`, `ids-unique`, `id-path-consistent`, `frontmatter-shape`). Standalone `aiwf check` on the working tree returns 0 errors and 1 unrelated warning (`provenance-untrailered-scope-undefined` — no upstream configured), confirming the parser-tolerance change is genuinely pure-additive. Kernel met commit: `d69ed24`.

## Decisions made during implementation

- **`Canonicalize` is below-grammar-floor-tolerant.** An input like `E-1` (below the 2-digit floor of `E-\d{2,}`) passes through verbatim rather than being padded to `E-0001`. Rationale: the grammar's per-kind floor defines the input space; `Canonicalize` tolerates legacy widths but does not invent well-formed ids from non-conforming input. Documented in the function's docstring and in the `epic-below-floor-passthrough` test case.
- **ADR is exempt from narrow-tolerance work.** Its grammar (`ADR-\d{4,}`) was always at canonical width; the AC-5 sweep regex (`"[EMGDC]-[0-9]{1,3}"`) excludes it deliberately.
- **Body-prose ids are not canonicalized at the kernel layer.** The renderer canonicalizes structural surfaces (headings, anchors, kicker, sidebar links) but leaves authored body content alone; that rewrite rides with M-0082's `aiwf rewidth --apply`.
- **Filenames are not canonicalized.** `idToFileName` and `idToHTMLFile` preserve the on-disk shape so links keep pointing at the actual file; M-0082 is the file-rename surface.
- **Contract-binding yaml entries preserve on-disk width verbatim.** `aiwf contract unbind` keeps remaining entries at their authored width; lookup compares canonical-to-canonical so a narrow legacy entry still matches a canonical query. Allowlisted in the AC-5 sweep test.
- **`aiwf history` event detail/subject text is verbatim from git-log.** The chip text and structural id columns canonicalize, but pre-migration commit-subject prose carrying narrow trailers renders as narrow because that's the literal git content. Matches the spec's "no history rewrite" guarantee.

No ADRs filed mid-implementation. ADR-0008 was already the policy precedent for the entire epic.

## Validation

- `go build -o /tmp/aiwf ./cmd/aiwf` — clean.
- `go test -race ./...` — 25 packages, 0 failures.
- `golangci-lint run` — 0 issues.
- `aiwf doctor --self-check` — 30/30 steps.
- `aiwf check` on this repo's tree — 0 errors, 1 unrelated warning (`provenance-untrailered-scope-undefined`, no upstream configured).
- `aiwf show M-081` — all 6 ACs `met` with `phase: done`; no findings.
- Coverage on the new helpers: `Canonicalize` 95.5% (one defensive branch marked `//coverage:ignore`), `IDGrepAlternation` 100%, `AllocateID` 93.8%.

## Deferrals

None. The two known follow-ons remain on the originally-planned milestones:

- **Doc-tree narrow-id sweep** (`docs/`, `README.md`, `CHANGELOG.md`, ADR-0003 amendment, CLAUDE.md commitment #2 update, embedded skill content refresh, rituals-plugin coordination) — rides with **M-0083** per the epic plan; out of scope here by design.
- **Active-tree file rename to canonical width** (`work/E-22-…` → `work/E-0022-…` etc.) — rides with **M-0082**'s `aiwf rewidth --apply`. The on-disk filenames remain at narrow width post-M-0081; parser tolerance carries the load until M-0082 lands.

## Reviewer notes

- **Two parallel sources of truth collapsed.** `internal/verb/import.go::canonicalPadFor` had been quietly drifting from `internal/entity/allocate.go::canonicalPad`; AC-1 deleted the verb-side duplicate and routed every caller to `entity.CanonicalPad`. The "single source of truth for pad width" constraint in the spec is now mechanical.
- **HTML structural assertions use a hand-rolled balanced-tag walker** (`htmlElement`/`htmlSection` in `cmd/aiwf/canonicalize_render_test.go`) rather than `golang.org/x/net/html`. The walker scopes by tag+class before substring-matching, which honors CLAUDE.md's "substring assertions are not structural assertions" rule in spirit. A future improvement could swap to a real parser; not blocking.
- **ADR-0008 still references the now-removed `internal/verb/import.go::canonicalPadFor`** at lines 8, 38, 117. These read as accurate pre-migration history of the function this ADR's policy displaced. Convention is to leave ADR bodies as authored; the reviewer can decide whether to add a post-migration footnote pointing at `entity.CanonicalPad`. Doc-lint flagged it; the milestone deliberately did not amend.
- **Below-grammar-floor passthrough is a deliberate non-goal.** `Canonicalize("E-1")` returns `"E-1"`, not `"E-0001"`. If a future consumer needs left-padding of malformed ids, that's a new capability; this milestone's contract is "narrow legacy widths tolerated, malformed inputs passed through verbatim."
- **18 mechanical aiwf state-transition commits** sit ahead of the wrap commit (red→green→done→met for each of 6 ACs). They modify only the milestone spec's frontmatter and STATUS.md; production code is bundled in the wrap commit only.
