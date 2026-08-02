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
      status: open
      tdd_phase: red
    - id: AC-3
      title: proseMask is unchanged and body-prose-id still exempts code constructs
      status: open
    - id: AC-4
      title: The three keep-list teaching files produce no finding
      status: open
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

### AC-3 — proseMask is unchanged and body-prose-id still exempts code constructs

`body-prose-id` shares `proseMask` and legitimately needs code constructs exempt
— an entity body discussing id syntax wraps the shape in backticks precisely so
it does not fire. The new scanning behavior is therefore a distinct mask, not an
edit to the shared one.

Evidence: `body-prose-id`'s existing fixtures pass unchanged, plus an explicit
assertion that a real id inside a code span in an *entity body* still produces no
finding.

### AC-4 — The three keep-list teaching files produce no finding

Three shipped files cite narrow ids as the subject of a rule rather than as
examples of current shape: the `aiwf-check` skill's grammar table, and the two
planning rituals that sanction narrow numerics as conversational shorthand for
not-yet-allocated milestones. All three must stay silent under both new
behaviors.

Evidence: an assertion against the real files — not synthetic copies — that each
produces zero findings, keyed by path so a rename surfaces as a failure rather
than as silent loss of coverage.

## Constraints

- **`proseMask` is not edited.** `body-prose-id` shares it and needs its current
  behavior. The new scan uses a distinct mask.
- **This milestone lands at warning severity.** At error it would fail
  `aiwf check` on this repo with 203 findings and block every push before the
  sweep that clears them exists. The flip to error is the last act of the
  following milestone.
- **The keep-list is by path**, and each entry carries a one-line rationale in
  the same shape the repo's other allowlists use.

## Design notes

- The width check extends `skill-body-id` as one un-subcoded finding code rather
  than shipping as a sibling rule or splitting by subcode (D-0051). Both shapes
  share a remediation — write the canonical letter-N placeholder — so one hint
  states the fix for both, and the taxonomy gains nothing a consumer could act on
  in a repo where this rule is structurally inert. Severity follows the detected
  class rather than the rule: a real id in prose keeps error severity, since it
  has no outstanding sites and so blocks no push, while the newly-reachable
  classes land at warning.
- The severity split is scaffolding with a defined lifetime. Once the sweep
  completes and severity flips, the prose/code distinction stops meaning
  anything, so the machinery implementing it is deleted alongside the flip rather
  than left standing.
- The width detector subsumes the partial one in
  `TestSkillBodyID_PlaceholdersAreCanonical`, which reads `SKILL.md`
  post-frontmatter bodies through the shared mask. That test keeps its real-tree
  assertion but reads the rule's output instead of re-deriving the property, so
  the two cannot drift.
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
convention. Severity follows the detected class. On this tree: 34 warnings, 0
errors — the worklist the sweep milestone consumes, blocking no push.

Measured before building: all 50 real-id citations in the shipped trees sit
inside code constructs and none in plain prose, so the mask alone accounted for
the rule's complete silence.

Two contract changes ride along. The code-construct exemption was pinned as a
deliberate carve-out in three places; those now assert the reverse, with the
doc-link destination kept as the one surviving carve-out. And the two real-tree
assertions filter to error severity for the sweep window — a real, temporary
reduction that ends when the flip lands, at which point they become whole-tree
zero gates again with no edit.

· commit 08c6489f9 · check package green, `make check-fast` exit 0

## Decisions made during implementation

- D-0051 — extend `skill-body-id` as one un-subcoded rule rather than a sibling
  or a subcode split.
