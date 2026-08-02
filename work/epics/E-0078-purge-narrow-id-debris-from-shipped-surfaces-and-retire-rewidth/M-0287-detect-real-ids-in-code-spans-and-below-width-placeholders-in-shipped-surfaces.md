---
id: M-0287
title: Detect real ids in code spans and below-width placeholders in shipped surfaces
status: in_progress
parent: E-0078
tdd: required
acs:
    - id: AC-1
      title: A real entity id inside a code span or fenced block produces a finding
      status: met
      tdd_phase: done
    - id: AC-2
      title: A below-canonical-width letter-N placeholder fails; canonical width passes
      status: met
      tdd_phase: done
    - id: AC-3
      title: body-prose-id's view of a body does not move
      status: met
      tdd_phase: done
    - id: AC-4
      title: The three keep-list teaching files produce no finding
      status: cancelled
---

## Goal

Close the two holes that let 203 narrow-id sites accumulate under a rule already
built to reject them, so the sweep that follows works from a mechanical worklist
rather than from a grep.

## Context

`skill-body-id` is already the right rule for this corpus. It scans the three
embedded trees whole-file including frontmatter, covers entity templates and
role-agent cards, runs pre-push at error severity, is inert in a consumer repo,
and already holds the polarity these surfaces need — a real digit-bearing id is
the defect and a canonical letter-N placeholder is correct.

Every one of the 203 sites survives it for two specific reasons. It masks code
constructs, so an id inside a command example or a fenced transcript is exempt by
construction — which alone accounts for every real-id citation in the tree, none
of which sits in prose. And it declines to check placeholder width, under a
comment asserting that placeholder normalization is policed elsewhere. What
polices it is one test over a strictly smaller corpus: `SKILL.md` files only,
post-frontmatter only, and through the same mask — so it sees none of the
code-construct cases, none of the role-agent cards or entity templates, and none
of the `description:` frontmatter. The comment's claim is narrower than it reads.

This milestone lands detection only. No shipped surface changes here; the rule's
own output becomes the worklist the next milestone sweeps.

## Acceptance criteria

### AC-1 — A real entity id inside a code span or fenced block produces a finding

`ScanSkillBodyID` currently masks with the shared prose mask, blanking inline
code spans and fenced blocks. After this milestone a real, digit-bearing entity
id in either position produces a `skill-body-id` finding at the correct line.

Two live instances exist in the tree today and make good fixtures: a real
milestone id inside a `#` comment within a fenced block, and a real epic id
inside an inline code span in a `--reason` example.

Evidence: fixtures asserting a finding, at the right line, for a real id inside
an inline code span, inside a fenced block, and inside an HTML comment — plus the
existing plain-prose case still firing unchanged.

### AC-2 — A below-canonical-width letter-N placeholder fails; canonical width passes

A letter-N placeholder narrower than canonical width produces a finding; one at
canonical width does not. Holds for both the bare and the composite id form.

Evidence: a table-driven fixture over every kind prefix, crossing narrow against
canonical and bare against composite, asserting fire and no-fire respectively.

### AC-3 — body-prose-id's view of a body does not move

`body-prose-id` needs code constructs exempt — an entity body discussing id
syntax wraps the shape in backticks precisely so it does not fire. The shipped
surfaces need the opposite. The two masks are therefore one walker under two
settings, and the invariant is behavioural: `body-prose-id`'s findings are
identical before and after.

Evidence: `body-prose-id`'s existing fixtures pass unchanged, plus an explicit
assertion that a real id inside a code span in an *entity body* still produces no
finding.

### AC-4 — The three keep-list teaching files produce no finding

Cancelled: the keep-list this criterion exists to prove is not built (D-0052).
Measuring the three files showed an exemption keyed by path would have laundered
real debris out of the sweep's own worklist, and that the teaching citations it
would protect are avoidable. The rewrite that replaces it belongs to the sweep
milestone.


## Constraints

- **`body-prose-id`'s view of a body does not move.** It needs code constructs
  exempt, since a backticked id-shape is how an entity body legitimately
  discusses id syntax. The two masks are one walker under two settings rather
  than two copies, so the invariant is held mechanically: a differential test
  fixes the single axis they may differ on, and every way the shared walker
  could drift `body-prose-id` fails it.
- **This milestone lands at warning severity.** At error it would fail
  `aiwf check` on this repo and block every push before the sweep that clears
  the tree exists. The flip to error is the last act of the following milestone,
  and severity is uniform until then — the rule draws no distinction between its
  two shapes or between prose and code placement.
- **No exemption mechanism.** The rule carries no keep-list; passages that
  document a rejected shape describe it rather than exhibit it (D-0052).

## Design notes

- The width check extends `skill-body-id` as one un-subcoded finding code rather
  than shipping as a sibling rule or splitting by subcode (D-0051). Both shapes
  share a remediation — write the canonical letter-N placeholder — so one hint
  states the fix for both, and the taxonomy gains nothing a consumer could act on
  in a repo where this rule is structurally inert.

## Out of scope

- Any content edit to a shipped surface. Detection only.
- The repo-facing doc corpus, which needs a genuinely width-shaped rule over a
  different corpus where real ids are legitimate.
- The `import` mint hole, which ships as a standalone patch outside this epic.

## Dependencies

- None. First milestone in the epic.

## References

- G-0481 — the audit: per-tier counts, both guard holes, the keep-list rationale.
- E-0076 — the same missing-detector pattern across three unrelated instances.

## Work log

### AC-1 — Real ids in code constructs

`ScanSkillBodyID` scans through `proseAndCodeMask`, a second entry point onto
`proseMask`'s walker parameterized by whether code constructs are content, so
`body-prose-id`'s narrower view is unchanged by construction rather than by
convention. On this tree at AC-1: 34 real-id findings, all warnings — the start of the
worklist the sweep milestone consumes, blocking no push.

Measured before building: all 50 real-id citations in the shipped trees sit
inside code constructs and none in plain prose, so the mask alone accounted for
the rule's complete silence.

Two contract changes ride along. The code-construct exemption was pinned as a
deliberate carve-out in three places; those now assert the reverse, with the
doc-link destination kept as the one surviving carve-out. And the two real-tree
assertions are inverted for the sweep window: they assert the worklist is
non-empty, so the sweep's arrival breaks them and forces their restoration.

· commit 08c6489f9 · check package green, `make check-fast` exit 0

### AC-2 — Non-canonical placeholders

`skillTokenMessage` classifies each id-shaped token and returns the finding text
for it, or empty when the token is the canonical letter-N form — the sibling
rule's empty-means-clean idiom. The boundary that decides it: a narrow NUMERIC id
is a real id at a legacy width, not a malformed placeholder, because read
tolerance keeps it resolving. On this tree: 115 findings — 81 placeholder, 34 real-id — all
warnings, 0 errors.

`internal/policies/narrow_id_sweep_test.go` gains an allowlist entry: the narrow
numeric literals in the width test are the input space proving the classifier
does not confuse a legacy width with a malformed shape.

· commits ff6d186b0, fd97f0ee4 · `make check-fast` exit 0

### AC-3 — body-prose-id's view of a body does not move

A differential test asserts the two masks directly rather than inferring them
from downstream findings: they agree on prose, on raw HTML, and on all four
non-prose link carriers, and diverge only on code constructs.

No production change — the behaviour already held. The AC is a characterization
pin, so its red is that the test catches the defect it guards: four drift modes
probed by mutation (code spans, fenced blocks, indented blocks, newline
preservation), all four caught.

The AC names a second fixture — a *real* id inside a code span in an entity body
staying silent — which would assert nothing: `body-prose-id` is silent on any
id that resolves, mask or no mask. The fixtures that do exercise the mask are the
malformed and unresolved shapes, and those already exist in that suite. The
"existing fixtures pass unchanged" half is met by its eleven-case CommonMark
table continuing to pass.

· commit e656ed5d4 · `make check-fast` exit 0

### Review round

Three sliced reviewers found twelve blocking issues; a fourth reviewed the fixes
for bloat. The fixes corrected the shipped severity documentation, tightened the
composite placeholder arm to milestones, named both defect classes in the hint
and the skill table, and replaced two skipped real-tree assertions with
inversions that break when the sweep lands. The bloat pass then folded a
three-value enum into the message function, deleted a test pinning an
implementation detail no behaviour depends on, and removed prose stating the
same four facts across three entities.

· commits 94d28133a, plus the trim commit carrying this entry

## Decisions made during implementation

- D-0051 — extend `skill-body-id` as one un-subcoded rule rather than a sibling
  or a subcode split.
- D-0052 — dissolve the shipped-surface keep-list rather than mechanizing it;
  cancels AC-4 and moves the passage rewrite to the sweep milestone.

## Validation

`make ci` exit 0 — 72 packages green, diff-scoped coverage gate clean, firing-fixture
meta-gate clean, `aiwf doctor --self-check` 29 steps. `aiwf check` on this tree: 0
errors, 116 warnings (115 `skill-body-id` + one `provenance-untrailered-scope-undefined`,
which is the branch having no upstream, not a property of the work).

Branch coverage: no changed statement is uncovered. The five uncovered blocks in the
two changed files all predate this milestone and sit outside every changed hunk; two
carry `//coverage:ignore` for TOCTOU.

## Deferrals

- G-0514 — `skill-body-id` classifies CLI metavariables (`M-id`, `E-id`, `C-id`),
  distinct-placeholder conventions (`M-PPPP`/`M-QQQQ`), and non-id acronyms
  (`ADR-NEW`, `ADR-OPSPEC`) as placeholder defects, and hands each a remediation
  that would corrupt a correct command synopsis. The classification is defensible;
  the instruction is not. Deferred deliberately: the sweep milestone enumerates the
  population, and its per-token judgments are the evidence for which of the three
  resolution shapes is right.

## Reviewer notes

Three fresh-context reviewers over the full change-set, sliced by concern
(production code / test changes / entity prose). All returned request-changes;
every blocking finding was fixed on the branch before wrap. A fourth pass
reviewed the fixes for bloat and cut 94 lines the first three had not
questioned.

Declined, with reasons, so a later reviewer meets a decision rather than a blank:

- **Restoring the deleted placeholder test.** Reintroduces the duplicate width
  implementation D-0051 exists to remove. The inversion fixes the defect that
  actually bit — a removal trigger living only in prose.
- **A whole-tree gate during the detection window.** Deliberately absent. Detection
  ships a milestone ahead of cleanup; that is the epic's warning-first constraint,
  and the exposure is recorded in D-0051's Consequences.
- **Narrowing `idTokenPattern` to kill the false-positive class.** The grammar is
  shared with `body-prose-id`, so narrowing it is not a local change. Tracked as
  G-0514 and decided against the sweep's real worklist.
