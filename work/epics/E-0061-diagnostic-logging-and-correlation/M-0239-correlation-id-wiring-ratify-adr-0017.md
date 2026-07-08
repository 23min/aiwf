---
id: M-0239
title: Correlation id wiring; ratify ADR-0017
status: done
parent: E-0061
depends_on:
    - M-0237
    - M-0238
tdd: required
acs:
    - id: AC-1
      title: An envelope's correlation_id matches the run_id in that invocation's log lines
      status: met
      tdd_phase: done
    - id: AC-2
      title: Mutating verbs report per-verb-appropriate metadata in their envelope
      status: met
      tdd_phase: done
    - id: AC-3
      title: An operator can pass --trace to see per-phase timings via the logger
      status: met
      tdd_phase: done
    - id: AC-4
      title: A renamed Envelope field is caught by a structural policy test
      status: met
      tdd_phase: done
    - id: AC-5
      title: ADR-0017 reads accepted with CLAUDE.md matching shipped behavior
      status: met
      tdd_phase: done
---

## Goal

Close the loop between an invocation's JSON envelope and its diagnostic-log
lines with one shared correlation id, so RCA on any finding is a grep, not a
manual timestamp-matching exercise — then ratify ADR-0017 now that the
codebase actually matches it.

## Context

`render.Envelope.Metadata.correlation_id` is declared today but dead: no
caller populates it. M-0237 shipped the logger (with its own per-invocation
`run_id`); M-0238 migrated the known diagnostic call sites onto it. This
milestone is the capstone: it ties those two pieces together with one id,
and — because ratifying an ADR means the implementation now matches the
decision in full, not just in part — this is also where ADR-0017 moves
`proposed → accepted`.

## Acceptance criteria

### AC-1 — An envelope's correlation_id matches the run_id in that invocation's log lines

The Cobra root mints one id per invocation (a UUID) and threads it into
`render.Envelope.Metadata.correlation_id`. The same id is passed into
`logger.WithVerb(...)` as `run_id`. One grep on either value finds the other.

### AC-2 — Mutating verbs report per-verb-appropriate metadata in their envelope

Today only read-only verbs (e.g. `aiwf check`) populate `metadata`. Mutating
verbs gain their own per-verb-appropriate fields: `promote` reports
`entity_id`/`from`/`to`/`commit_sha`; `archive` reports
`swept_count`/`commit_sha`; and so on per verb. The shape is per-verb; the
discipline (every mutating verb reports *something*) is uniform.

### AC-3 — An operator can pass --trace to see per-phase timings via the logger

`--trace` is a logger consumer, not an envelope consumer — it depends on
M-0237's logger existing, which it now does. Emits per-phase timings at
`debug` level through the same bound logger, not a separate mechanism.

### AC-4 — A renamed Envelope field is caught by a structural policy test

`internal/policies/envelope_structural_assertion.go` pins the envelope's
required-key set against the `Envelope` struct's field tags, so a future
field rename that would silently break a downstream JSON consumer fails CI
instead.

### AC-5 — ADR-0017 reads accepted with CLAUDE.md matching shipped behavior

`aiwf promote ADR-0017 accepted` once AC-1 through AC-4 (and M-0237, M-0238)
are done. CLAUDE.md's Go conventions §CLI conventions logging paragraph is
rewritten to reflect the shipped opt-in/XDG-file/`forbidigo` behavior,
replacing the stale "log/slog to stderr default INFO" prescription, with a
cross-link to ADR-0017.

## Constraints

- `correlation_id` is an opaque per-invocation identifier — never compared
  or branched on for anything but exact-match correlation.
- Ratifying the ADR (AC-5) is the last thing that happens in this
  milestone, not the first — it certifies a state that must already be true.

## Design notes

- ADR-0017 and G-0232 are the locked design; this milestone is their
  implementation, not a re-scoping of either.

## Surfaces touched

- `cmd/aiwf` (Cobra root: correlation id minting)
- `internal/render/render.go` (`Envelope.Metadata.correlation_id`)
- `internal/verb/*` (mutating-verb metadata)
- `internal/policies/envelope_structural_assertion.go` (new)
- `docs/adr/ADR-0017-...md`, `CLAUDE.md`

## Out of scope

- The correctness stress harness that will *consume* this correlation id at
  scale — the second epic named in
  `docs/initiatives/robustness-correctness-stress-testing.md`.

## Dependencies

- M-0237 — the logger and its `run_id` must exist.
- M-0238 — the ADR isn't ratifiable until the migration + chokepoint half of
  it is also true.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- G-0232 — envelope enrichment: correlation_id wiring + mutating-verb metadata

---

## Work log

### AC-1 — An envelope's correlation_id matches the run_id in that invocation's log lines

`Execute` mints one id per invocation (`logger.NewRunID()`) and threads it
through `NewRootCmd` into every mutating verb's `NewCmd(correlationID
string)`, landing on a new `cliutil.OutputFormat.CorrelationID` field.
`outputformat.go`'s three envelope emitters (error, findings, success) all
inject `metadata.correlation_id` when set, via a single `metadata()` helper
— so the id is universal across every mutating verb's JSON envelope, not
just the ones that also log. `cancel` and `move` (the two verbs already
calling `logger.WithVerb`) now reuse the threaded id as `run_id` instead of
minting their own, with a same-value fallback for a direct-`Run` caller
that bypasses `NewCmd`/`Execute` (only reachable from a test, never from the
CLI surface, since `Execute` always mints a real id).

Two nested-subcommand constructors (`add`'s `newACCmd`, `worktree`'s
`newAddCmd`) needed the id threaded as an explicit parameter rather than
picking it up from an enclosing closure; a first mechanical pass missed
`worktree`'s, caught immediately by a compile error (`undefined:
correlationID`) rather than shipping silently.

Tested end-to-end (envelope `metadata.correlation_id` present, and for
cancel/move specifically equal to the log line's `run_id`) across 8 of the
15 correlation-id-bearing verb constructors, spanning every structurally
distinct code shape this plumbing touches: `DecorateAndFinish`-mediated
(`promote`, `cancel`, `move`, `rename`, `add`), `FinishVerb`-direct
(`authorize`, `acknowledge illegal`), and nested-subcommand threading
(`add ac`, `acknowledge illegal`). The remaining 7 (`retitle`, `setarea`,
`renamearea`, `reallocate`, `editbody`, `acknowledge mistag`, `worktree
add`) are structurally identical to a verified verb and were confirmed by
direct inspection to have no `out` reassignment between the
`CorrelationID` write and the emit call — the one failure mode compilation
doesn't already rule out.

`wf-vacuity` mutation probes: `metadata()` forced to always return `nil`
(caught — 3 tests failed), `cancel`'s reuse dropped in favor of always
minting fresh (caught), `move`'s reuse dropped the same way — **not
caught** by the test suite as it existed at that point, since no test
cross-checked `move`'s `run_id` against its envelope's `correlation_id`.
Fixed by adding `TestCorrelationID_MoveMatchesLogRunID`; re-ran the
mutation, now caught. All mutations reverted via captured pre-mutation
content, never `git stash`/`checkout`/`restore`. Commit `5eb2d1ef`.

### AC-2 — Mutating verbs report per-verb-appropriate metadata in their envelope

`verb.Result` gains an optional `Metadata map[string]any` field. `FinishVerb`
merges it with AC-1's `correlation_id` and (on a successful apply)
`commit_sha` into the envelope, via `OutputFormat.Metadata` — re-exported from
AC-1's private `metadata()` helper once a real second caller (a verb building
its own envelope outside `emitSuccess`) existed.

Populated per-verb metadata across all 15 correlation-id-bearing verb
constructors: `promote`/`cancel` (both plain-entity and composite-AC-id
shapes, the latter via the shared `finalizeACPlan` chokepoint so
`promoteAC`/`PromoteACPhase`/`cancelAC` all gained it in one edit), `rename`/
`retitle` (plain + composite), `setarea`, `renamearea`, `reallocate`,
`editbody` (explicit + bless paths), `authorize` (open + pause/resume),
`acknowledge illegal`/`mistag`, `add` (kind + `add ac`), `move`, and
`worktree add`. `archive`, `rewidth`, and `importcmd` had no `--format=json`
support at all before this AC — added from scratch (own `render.Envelope`
construction, since none of the three route through `FinishVerb`), each
reporting its own shape (`swept_count`, `renamed_count`,
`imported_count`/`entity_ids` respectively) plus `commit_sha`.

The branch-coverage audit surfaced two real gaps beyond the AC's own scope:
`move` already had AC-1's `correlation_id` wired but no per-verb metadata yet
(a straightforward miss, fixed here); and `worktree add` builds its own
`render.Envelope` directly rather than calling `emitSuccess`, so its
`OutputFormat.CorrelationID` field was being set correctly (AC-1) but never
actually read anywhere — the envelope's `metadata.correlation_id` was silently
absent until this pass exported `Metadata` and wired it in. Separately, the
audit surfaced that `contract bind`/`unbind`/`recipe install`/`recipe remove`
and `milestone depends-on` were missed entirely during AC-1: their `NewCmd`
constructors call `cliutil.AddFormatFlags` (confirmed via a full-repo grep,
not the narrower survey the AC-1 pass used) but had no `correlationID`
threading at all until this commit. All five now carry both `correlation_id`
and their own metadata (`entity_id`/`validator`, `depends_on`).

`wf-vacuity` mutation probes: `FinishVerb`'s success path forced to pass `nil`
metadata (caught), `withCommitSHA` mutated to drop the sha assignment
(caught), `promote`'s `from`/`to` values swapped (caught), `importedEntityIDs`
mutated to always return `nil` (caught). Manual branch-coverage walk found
`failArchive`/`failRewidth`/`failImport`'s JSON-mode error paths untested (one,
`failImport`, was 0% covered — no existing test reached any of its error
conditions in either format) and `failImport`'s text-mode path also untested;
fixed by adding dedicated JSON- and text-mode error-envelope tests for all
three. The same walk found `withCommitSHA`'s "both empty" early-return branch
unreachable at all four call sites (sha is always non-empty by the time any
of them run) — simplified away across all four copies rather than annotated,
relying on `encoding/json`'s `omitempty` treating a zero-length map identically
to nil (verified directly before relying on it).

Every verb in the AC-2 rollout has a dedicated test asserting its own
metadata field values specifically (`entity_id`/`from`/`to`, `old_slug`/
`new_slug`, `old_title`/`new_title`, `area`, `old_area`/`new_area`, `old_id`/
`new_id`, `agent`/`action`, `sha`, `kind`, `ac_ids`), not just `correlation_id`
— the class of gap `move` and `worktree add` both fell into above. Commit
`29f0c8ff`.

**Post-`met` correction, from the wrap-time independent review** (see
`## Reviewer notes`): two real gaps in this AC's own scope. All five
`--audit-only` plan producers (`internal/verb/auditonly.go`) set no
`Metadata` at all — `aiwf promote/cancel <id> --audit-only --format=json`
silently dropped `entity_id`/`to` that the normal path reports for the
same verb. Composite-AC `cancel` (`finalizeACPlan`, shared by
`promoteAC`/`PromoteACPhase`/`cancelAC`) reused the trailer-suppression
empty `to` for `metadata.to`, reporting `""` instead of `"cancelled"` —
inconsistent with top-level `Cancel`'s own metadata line one function
away; `finalizeACPlan` now takes a separate `metaTo` param decoupled
from the `aiwf-to` trailer value. Fixed test-first (RED confirmed
against the unfixed code, GREEN after), each fix mutation-probed
(mutant caught, files restored byte-identical). The `finalizeACPlan`
signature edit also resurfaced a genuine, previously-unflagged coverage
gap in its own projection-findings early return and in the wholly-
untested `cancelACAuditOnly` composite branch — both closed with new
tests. Commit `4abb73f4`.

### AC-3 — An operator can pass --trace to see per-phase timings via the logger

`OutputFormat` gains a `Trace bool` field; `AddFormatFlags` registers `--trace`
directly, so every mutating verb gets it automatically with no per-verb
wiring (unlike `CorrelationID`, which needed a constructor param since it's
invocation-scoped rather than flag-scoped). `FinishVerb` times its
`verb.Apply` call — the shared chokepoint every mutating verb already funnels
through, and the only part of a verb's execution that touches git/filesystem
— and emits one `phase.apply` event at `debug` level with `elapsed_ms` when
`Trace` is set. `cliutil.ResolveTraceLogger` is a `--trace`-specific variant
of `ResolveLogger` that forces the invocation's logger on at debug level
regardless of `AIWF_LOG`, so the flag genuinely needs no separate env
configuration (its whole reason for existing).

Two real design gaps surfaced during implementation itself, not just at the
testing stage. First: `logger.ResolveConfig` short-circuits to a completely
empty `Config{}` (format and destination never even resolved) whenever no
level is supplied from any source — a first version that patched
`Enabled`/`Level` onto whatever came back therefore silently discarded
`AIWF_LOG_FILE`/`AIWF_LOG_FORMAT` and produced no output at all under
`--trace` with no `AIWF_LOG` set (the exact scenario `--trace` exists for).
Fixed with a `forcedGetenv` wrapper that supplies a synthetic `"debug"` for
`AIWF_LOG` only when the real environment and `aiwf.yaml` both have no level
set, so `ResolveConfig` always takes its normal fully-resolving path and
every other key still reads the real environment/yaml unchanged. Second: the
error-fallback for an invalid `AIWF_LOG_FORMAT` value also discarded the
destination, meaning an unrelated env typo would silently defeat `--trace`
too — fixed to preserve `AIWF_LOG_FILE`/`aiwf.yaml`'s destination even when
format resolution itself fails.

`wf-vacuity` mutation probes: `FinishVerb`'s `if out.Trace` guard negated
(caught by both trace tests), the debug-level clamp in
`ResolveTraceLogger` removed (caught by
`TestResolveTraceLogger_ClampsAMoreRestrictiveAIWFLOG` specifically — the
right test, not just some test). One line — the final `w.(io.Closer)`
fallback in `ResolveTraceLogger` — is `//coverage:ignore`d with a proven
reachability argument rather than left silently uncovered: `OpenDestination`
only returns a nil, non-`Closer` writer when `cfg.Enabled` is false (its own
early return), but `ResolveTraceLogger` forces `Enabled = true`
unconditionally before that call, so the fallback that IS reachable in
`ResolveLogger` (via its own genuinely-disabled path) provably isn't here.
Commit `2ad6528f`.

### AC-4 — A renamed Envelope field is caught by a structural policy test

`internal/policies/envelope_structural_assertion.go` AST-parses
`internal/render/render.go`'s `Envelope` struct directly and pins its json
tag set against a checked-in required-key list (`tool`, `version`, `status`,
`findings`, `result`, `error`, `metadata`), modeled on
`config_fields_discoverable.go`'s reflect-over-parsed-tag idiom. This is a
structural check, distinct from `internal/cli/integration/envelope_schema_test.go`'s
existing runtime check (drives every `--format=json` verb and diffs the
marshaled output): the runtime test catches a shape drift a verb introduces
at its own call site, while this one catches the type declaration itself
drifting from its documented contract — reachable even before any verb's
JSON output is ever inspected.

Nine tests cover every reachable branch: a renamed field, a dropped field, an
added (unpinned) field, the current correct shape, the missing-type-entirely
case, a non-struct `Envelope` (a type alias), a syntactically broken source
file (parse error), an unreadable file (permission-denied, exercised with a
real `os.Chmod`), and untagged/`json:"-"`-tagged fields (both correctly
ignored). One branch — the `os.IsNotExist` short-circuit — is
`//coverage:ignore`d: every fixture in this policy's own test file writes the
pinned file before calling the policy, so no test path can reach it, and no
synthetic fixture that omits the file exists to reach it either.

A real, if minor, gap surfaced only by deliberately re-checking rather than
trusting the initial "done" claim: every other policy in this package has an
entry in `policies_test.go`'s shared `TestPolicy_<Name>` + `runPolicy(...)`
live-repo smoke-test list — the canonical place a maintainer scanning this
file finds "what does this repo enforce." The first version omitted this
entry and carried a redundant hand-rolled live-repo check in its own test
file instead — functionally equivalent, but invisible to that canonical
list. Fixed by adding the missing `TestPolicy_EnvelopeStructuralAssertion`
entry and removing the now-duplicate test. Separately, running
`make coverage-gate` directly (rather than assuming its two component
gates would pass, since neither had actually been invoked before that
point) confirmed `TestPolicy_FiringFixturePresence` passes clean for this
policy's `Violation` construction sites, and that the gate's large
branch-coverage-audit failure list is entirely pre-existing G-0386 debt from
earlier milestones — grepped directly to confirm zero hits against either
new file or the `policies_test.go` edit.

`wf-vacuity` mutation probes: dropped `metadata` from the pinned required-key
list (caught — 4 tests failed, confirming the required-set comparison is
live, not vacuous), deleted the missing-key detection loop entirely (caught
by the one test specifically written to exercise it, not just some other
test coincidentally catching it). Commit `c247b625`.

### AC-5 — ADR-0017 reads accepted with CLAUDE.md matching shipped behavior

CLAUDE.md's "### CLI conventions" paragraph (under "## Go conventions")
described `log/slog` going unconditionally to stderr — stale since
M-0237/M-0238 shipped ADR-0017's opt-in, default-off, XDG-state-home-file
behavior. Rewritten to describe the shipped behavior (opt-in/default-off,
the three `AIWF_LOG*` env knobs, the XDG-state-home default destination,
`forbidigo` enforcement) with a cross-link to ADR-0017; the "tool output →
stdout" half — already correct — is unchanged, per ADR-0017's own
Consequences section naming exactly that split.

`internal/policies/claudemd_cli_conventions_logging.go` is the mechanical
evidence: it walks CLAUDE.md's heading hierarchy (`extractMarkdownSubsection`,
a purpose-built two-level `##`/`###` walker — `entity.ParseBodySections`,
the kernel's own markdown-section reader, only tracks `## ` boundaries and
folds every `###` subsection into one blob, wrong for isolating one `###`
subsection among many under a much larger `##` parent) to isolate exactly
the "CLI conventions" subsection's own prose, then asserts it no longer
carries the stale claim, describes logging as opt-in/default-off, and
cross-links ADR-0017 — a structural check, not a whole-file substring grep
(CLAUDE.md §"Substring assertions are not structural assertions"), so a
mention of "opt-in" or "ADR-0017" in some other section of this large file
doesn't satisfy it. Confirmed the check genuinely detects the *live*
staleness (not just synthetic fixtures) before rewriting the paragraph, and
genuinely detects the fix afterward.

Registered in `policies_test.go`'s shared live-repo smoke-test list from the
start this time, learning directly from AC-4's own gap.

`wf-vacuity` surfaced two real findings, not just missing tests. First: my
own initial "fires on stale claim" fixture used prose that didn't actually
contain either literal pattern the code checks for (no arrow character, no
literal word "to") — it was passing, but for the wrong reason (failing the
missing-ADR-0017-link check instead), a vacuous test that would have stayed
green even if the actual stale-claim detection code were deleted. Rewrote
all three "fires" tests to isolate exactly one violation condition each, so
each genuinely proves its own check fires for its own stated reason. Second:
a mutation dropping `extractMarkdownSubsection`'s parent-heading-scoping
check entirely (matching "### CLI conventions" under *any* `##` parent, not
just "Go conventions" specifically) survived every existing test — none of
the fixtures had the child heading nested under the wrong parent. Fixed by
adding a dedicated test proving the parent check matters, then re-confirmed
the mutation is caught. Commit `f2a27e0b`.

**Post-`met` correction, from wrap-time doc-lint**: the rewritten paragraph
itself claimed `forbidigo` bans bare `log.Print*` call sites, but neither
`.golangci.yml`'s `forbid` list nor the `logging-chokepoint` AST policy
enforce anything on `log.Print*` — only `fmt.Print*`/`Fprintln`/`Fprintf`
are covered. Reworded to name what's actually enforced; the structural
policy's three checks (stale-claim absence, opt-in wording, ADR-0017 link)
still pass against the corrected text. Commit `be43dbcf`.

Ratifying `aiwf promote ADR-0017 accepted` is the milestone's own final
constraint ("the last thing that happens... it certifies a state that must
already be true") — a separate, deliberate gate after AC-1 through AC-4 and
this paragraph rewrite all landed, not folded into this commit.

## Decisions made during implementation

- (none) — ADR-0017's `proposed -> accepted` promotion is AC-5's own deliverable
  (certifying a state the milestone made true), not a new decision entity.

## Validation

Full gate, green, after the wrap-time corrective commits landed:

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race ./...` — all packages pass.
- `make lint` (`golangci-lint run`) — 0 issues.
- `aiwf check` — 0 errors; 2 pre-existing, unrelated warnings
  (`epic-active-no-drafted-milestones`, `provenance-untrailered-scope-undefined`).
- `make coverage-gate` — fails, but exclusively on the ~194-branch
  `internal/cli/*` debt already tracked as G-0386 (discovered in M-0238,
  deliberately deferred there); confirmed by diffing against the pre-fix
  tree that no `internal/verb/*` line was in that set until this wrap's own
  `finalizeACPlan` edit resurfaced one — closed with a new test (see AC-2's
  Work log) before this validation pass.

## Deferrals

- (none) — G-0386 (the coverage-gate's `internal/cli/*` debt) is a pre-existing
  gap from M-0238, not new work this milestone deferred.

## Reviewer notes

**Independent two-lens review, run before wrap:**

- **Code-quality (`wf-review-code`), sliced by AC** — four independent,
  fresh-context reviewers, one per AC (AC-1 correlation-id, AC-2 metadata,
  AC-3 `--trace`, AC-4+AC-5 policies/CLAUDE.md), each briefed adversarially
  and instructed to verify by measuring (building, running tests, exercising
  the real binary) rather than trusting this spec's own narrative.
  - AC-1: **approve**. Verified end-to-end against the real binary
    (`correlation_id` byte-identical to the log's `run_id`, distinct across
    invocations, absent from read-only verbs). Two non-blocking notes: the
    spec's own "(a UUID)" wording is cosmetic-inaccurate (the id is 16 hex
    chars, not RFC-4122); no drift guard yet couples "calls
    `AddFormatFlags`" with "sets `CorrelationID`" for a *future* new verb.
  - AC-2: **request-changes** — two real bugs, both fixed as corrective
    commits (see AC-2's Work log): audit-only paths reporting no metadata,
    and composite-AC cancel reporting `to: ""` instead of `"cancelled"`.
  - AC-3: **approve**. Verified the `forcedGetenv` scope, the clamp
    direction, the fallback-destination preservation, and the
    `//coverage:ignore` justification all hold under adversarial probing
    (six env-configuration scenarios against the real binary).
  - AC-4+AC-5: **approve**. Verified the structural policies genuinely fire
    on the regressions they target (traced the AST-walk and the
    parent-heading-scoping logic by hand) and confirmed ADR-0017's promote
    was a pure status flip. Independently caught the same `log.Print*`
    inaccuracy this wrap's own doc-lint pass found, and confirmed the fix.
- **Design-quality (`wf-rethink`)**: not run. This milestone introduced no
  new module/package boundary, core abstraction, or data model — every AC
  is wiring an already-existing primitive (the logger package, `OutputFormat`,
  `verb.Result`) through the CLI layer, or a new policy following the
  established `internal/policies/` AST-walk pattern (~30 existing
  precedents). Nothing here meets `wf-rethink`'s non-trivial-design trigger.

**Residual self-checks**: no `TODO`/`FIXME` left in the diff; one stray
`scratch_cov2.html` (a `go tool cover` artifact from a reviewer's coverage
run) found in the working tree and removed before the corrective commits.
CLAUDE.md's `### CLI conventions` section is the shipped-surface doc update
this milestone's public behavior required (AC-5's own deliverable); no other
README/inline-doc surface changed.
