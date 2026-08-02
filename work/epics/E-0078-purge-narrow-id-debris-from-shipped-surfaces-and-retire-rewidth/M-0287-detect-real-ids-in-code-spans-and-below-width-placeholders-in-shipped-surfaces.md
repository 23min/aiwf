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

Measuring the three files showed they carry roughly twice as much ordinary debris
as teaching citation — including a narrow placeholder in one of their shipped
`description:` frontmatter lines — so an exemption keyed by path would
have laundered real defects out of the sweep's worklist permanently. The teaching
citations that remain are avoidable: the rule's contract already accepts a
shape-description in place of an exhibited shape, which leaves nothing to exempt.
Rewriting those passages belongs to the sweep milestone, with the rest of the
content it edits.

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
- Every finding is a warning until the sweep flips severity. The rule draws no
  distinction between its two shapes or between prose and code placement: a rule
  that held error severity for the one shape with no outstanding sites would buy
  a one-milestone guarantee and pay a per-token severity function, a byte-range
  helper, and a second parse of every file for it.
- The two shapes are therefore distinguishable only by the defect the message
  names. A test that asserts classification asserts on the message; one that
  checks only that something fired cannot tell them apart.
- The width detector subsumes the partial one that lived in
  `TestSkillBodyID_PlaceholdersAreCanonical`, which is deleted rather than
  rewired: rewiring it to read the rule's output would have duplicated the
  per-body real-tree assertion, which already scans that corpus. The production
  rule owns the property over a strictly larger one — every `*.md` whole-file,
  frontmatter included, with code constructs in scope.
- The `:65` comment claiming external policing is itself the defect class E-0076
  is built around: a rule stated in an authoritative surface with no detector
  behind it reads as enforced, so the next reader stops looking. Whichever shape
  the check takes, that comment stops making a claim nothing backs.

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

`classifySkillToken` sorts each id-shaped token into real id, canonical
placeholder, or placeholder defect, and anything but the canonical letter-N form
fires. The boundary that decides the classifier: a narrow NUMERIC id is a real id
at a legacy width, not a malformed placeholder, because read tolerance keeps it
resolving. On this tree: 115 findings — 81 placeholder, 34 real-id — all
warnings, 0 errors.

The partial width test is deleted rather than rewired. Rewiring it to read the
rule's output would have produced a duplicate of the per-body real-tree
assertion, which already scans that corpus; the production rule now owns the
property over every `*.md` whole-file with code constructs in scope.

The severity staging collapsed to a uniform warning in the same milestone, which
removed a per-token severity function, a byte-range helper, and the second parse
of every file that fed it — a net deletion. That collapse also removed what had
been distinguishing the rule's two shapes incidentally, so the classification
assertions moved onto the message, where the distinction actually lives.

`internal/policies/narrow_id_sweep_test.go` gains an allowlist entry: the narrow
numeric literals in the width test are the input space proving the classifier
does not confuse a legacy width with a malformed shape.

· commits ff6d186b0, fd97f0ee4 · `make check-fast` exit 0

### AC-3 — body-prose-id's view of a body does not move

A differential test asserts the two masks directly rather than inferring them
from downstream findings: they agree on prose, on raw HTML, and on all four
non-prose link carriers, and diverge only on code constructs. A second test pins
that both are same-length projections preserving newline positions, so a mask
that stripped rather than blanked would be caught even though the differential
alone would not notice.

No production change — the behaviour already held. The AC is a characterization
pin, so its red is that the test catches the defect it guards: four drift modes
probed by mutation (code spans, fenced blocks, indented blocks, newline
preservation), all four caught, scoped to this test so the kill is its own.

The AC's other named fixture — a real id inside a code span in an entity body
staying silent — already exists in the `body-prose-id` suite, so it is not
duplicated here. The "existing fixtures pass unchanged" half is met by that
suite's eleven-case CommonMark table continuing to pass.

· commit e656ed5d4 · `make check-fast` exit 0

## Decisions made during implementation

- D-0051 — extend `skill-body-id` as one un-subcoded rule rather than a sibling
  or a subcode split.
- D-0052 — dissolve the shipped-surface keep-list rather than mechanizing it;
  cancels AC-4 and moves the passage rewrite to the sweep milestone.
