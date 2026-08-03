---
id: M-0288
title: Sweep shipped surfaces to canonical placeholders and enforce at error severity
status: done
parent: E-0078
depends_on:
    - M-0287
tdd: required
acs:
    - id: AC-1
      title: No embedded surface carries a narrow id or placeholder
      status: met
      tdd_phase: done
    - id: AC-2
      title: Shipped entity templates seed no id shape that body-prose-id rejects
      status: met
      tdd_phase: done
    - id: AC-3
      title: The documenting passages describe rejected shapes rather than exhibit them
      status: met
      tdd_phase: done
    - id: AC-4
      title: The rule runs at error severity and this repo passes it
      status: met
      tdd_phase: done
---

## Goal

Convert every narrow id and narrow placeholder in the shipped surfaces to
canonical placeholder form, remove the two real-entity citations, repair the
entity templates, and close the milestone by turning the guard to error so the
debris cannot return.

## Context

The preceding milestone gives this one a mechanical worklist: run the rule, sweep
what it names, re-run. The population is 203 sites across 28 files — 51 narrow
numerics and 152 narrow placeholders — against 213 already-canonical placeholder
forms that show what the target looks like.

The sweep target is the canonical letter-N placeholder, uniformly. Canonicalizing
the digits would be the wrong fix twice over: it misses the 152 placeholders
entirely, and a fabricated canonical-width id is a *real* entity in most consumer
trees, which is the collision the placeholder convention exists to prevent.

## Acceptance criteria

### AC-1 — No embedded surface carries a narrow id or placeholder

Every file under the three embedded trees is free of narrow ids and
below-canonical-width placeholders. No file is exempted (D-0052).

This subsumes the two real-entity citations — a real milestone id in an
`aiwf-edit-body` fenced comment and a real epic id in an `aiwf-acknowledge`
`--reason` example. Both entities are `done` and archived, which is the rot the
skills policy predicts; the preceding milestone's rule catches them as real ids
rather than as width defects, and neither is fixed by widening.

Evidence: the rule from the preceding milestone, run over the real tree, reports
zero findings.

### AC-2 — Shipped entity templates seed no id shape that body-prose-id rejects

The shipped milestone, decision, and ADR templates carry narrow placeholders
today, and `body-prose-id` names exactly that shape as a leak it rejects in entity
bodies. The frontmatter occurrences are inert, since `aiwf add` stamps the real id over
them. The heading and prose-guidance lines are not: the rituals direct an author
to replace an entity's body with the template wholesale, which overwrites the
real heading `aiwf add` wrote, and author-facing guidance text survives into a
committed body — where the shipped check then fires on a consumer's own entity.

Evidence: `ScanBodyProseID` run over each shipped template's body reports zero
findings — the shipped check applied to the shipped template, so the two surfaces
cannot contradict each other again.

### AC-3 — The documenting passages describe rejected shapes rather than exhibit them

Six passages teach a rule by naming an id shape the kernel rejects: the
`aiwf-check` skill's findings table, both planning rituals' anti-pattern bullets,
the always-on guidance fragment's `body-prose-id` rule, the `aiwf-reallocate`
skill's no-suffix sentence, and the `aiwf-history` skill's prefix-match sentence.
Each is rewritten to name the rejected shape — a letter suffix, a spelled-out
word suffix, an all-caps letter placeholder, a numeric form narrower than
canonical width — instead of spelling an instance of it. That is what removes the
last thing an exemption would have been for.

These need rewriting rather than sweeping: a mechanical replacement would turn an
instruction about a bad shape into an instruction about a good one, which reads as
nonsense.

Evidence: the six files produce zero findings, paired with a structural assertion
— scoped to the passage's own section rather than the file — that each still
teaches its rule. Neither half suffices alone: a rewrite can clear the tokens by
deleting the instruction, and an instruction can survive alongside the token it
forbids.

### AC-4 — The rule runs at error severity and this repo passes it

The severity flip is the last act of the milestone: it lands only once the sweep
is complete, so no intermediate state blocks a push on debris that has not been
cleared yet.

Evidence: an assertion on the emitted finding's severity, plus this repo's own
`aiwf check` at zero errors.

## Constraints

- **Placeholder form is the canonical letter-N shape, never a canonical-width
  fictional id.**
- **The severity flip lands last.** No commit in this milestone leaves the tree
  with an incomplete sweep and an error-severity rule.
- **The documenting passages are rewritten, never swept mechanically.** A
  find-and-replace over a passage whose subject is a bad shape inverts its
  meaning.
- **Worked-example transcripts lose distinct ids and keep their titles.** The
  fictional scenario in the status and list skills renders every milestone as the
  same placeholder; the titles carry the distinctions. A guard with no transcript
  carve-out is worth more here than the vividness, because the sweep is re-run by
  a rule rather than by eye.

## Design notes

- `PolicySkillEditStructuralTestBackstop` requires every modified
  `embedded-rituals/**/SKILL.md` to be referenced by some `internal/policies`
  test. Nine ritual skill files need edits in this sweep and all nine already
  carry a referencing test, so the backstop is satisfied without new test-writing
  work. The verb-skill tree is outside the backstop's scope, and role-agent cards
  and templates are not `SKILL.md` at all.

## Out of scope

- The repo-facing doc corpus.
- Test fixtures, whose narrow ids are largely the narrow-read-tolerance suite.
- Code comments.

## Dependencies

- M-0287 — supplies both the worklist and the assertion this milestone is
  measured against.

## References

- G-0481 — per-tier counts, the two real-entity citations, the template vector.

## Work log

### AC-3 — Documenting passages

Six shipped passages teach a rule by naming an id shape the kernel rejects; each
now describes the shape instead of spelling an instance. The criterion that
selects them — a mechanical replacement would invert the instruction — reaches
past the `aiwf-check` findings table and the two planning rituals to the always-on
guidance fragment, `aiwf-reallocate`'s no-suffix sentence, and `aiwf-history`'s
prefix-match sentence.

The two ritual bullets forbid the canonical letter-N placeholder outright, which
reads oddly beside a shipped-surface rule that requires that exact shape. It is
correct as written: both bullets address an author writing an entity body, where
`body-prose-id` does reject it. The rewrite describes the shape rather than
spelling one, and leaves that polarity intact.

The metavariable and distinct-placeholder sites in these files needed no
exemption either. A CLI synopsis metavariable canonicalizes without loss, and the
two-different-milestones convention resolves to the `<id>[,<id>]` list form the
same tables already use for other multi-id flags. That is the measured answer to
the class G-0514 records, over the sites that actually exist.

Five mutations probed both halves of the evidence — deleting an instruction,
reintroducing a rejected shape, renaming a section heading, dropping two
remediation phrases — and all were killed. A sixth confirmed the section scoping
is load-bearing: a rule relocated into a neighbouring section passes without it.

On this tree: 115 findings → 88, six files clean.

· commit a0fe2d90c · `make check-fast` exit 0

### AC-2 — Entity templates

A template is the one artifact both id rules scan, and they disagree by design:
a shipped surface wants the canonical letter-N placeholder, an entity body
rejects it. Width alone therefore cannot satisfy this criterion — the epic
template was already fully canonical, passed `skill-body-id`, and `body-prose-id`
rejected it anyway. The fix is placement-shaped: frontmatter keeps canonical
placeholders (outside `body-prose-id`'s scope), prose carries them inside
backticks (a code span is masked by one rule and accepted by the other), and
text inside HTML comments — where backticks do not parse — is rephrased to name
the kind instead of the id.

The H1 headings are not inert. The rituals direct an author to replace an
entity's body with the template wholesale, which overwrites the real heading
`aiwf add` wrote; a bracket placeholder now reads as unfilled rather than as a
plausible id.

The test scans against an empty resolution index, since a consumer's tree
resolves none of aiwf's ids and a template clean only against aiwf's own tree is
not clean where it ships. All four templates are held as a class. Six mutations — two
restored H1 tokens, two un-backticked prose placeholders, a re-narrowed
placeholder, and a real id smuggled into a body — were all killed.

On this tree: 88 findings → 77.

· commit 17bc2c8b0 · `make check-fast` exit 0

### AC-1 — The sweep

Three passes cleared the remaining 77: width normalization of every narrow
letter-N placeholder, real ids collapsed to one placeholder per kind with the
worked examples' titles carrying the distinctions, and the residual
metavariables resolved to canonical or to the `<id>[,<id>]` list form the same
tables already use for other multi-id flags. Two sites carried aiwf development
history rather than only an id and lost it: a milestone id annotating
`edit-body`'s explicit-content mode, and an era reference in an `acknowledge`
`--reason` example.

Five policy tests pinned the old spellings. Four pinned a narrow id as
incidental spelling inside a search string and were widened with their subject
unchanged. The fifth pinned that the `aiwf-add` typo example *uses two distinct
ids* — a guard against exactly this collapse, protecting a form this epic
removes: a shipped surface carries no real id, and two distinct placeholders are
not canonical either. The example now states the failure instead of staging it,
and the test pins the behavior it owes the reader — unvalidated flag, no
verb-time signal, named downstream finding — scoped to its section rather than
grepping the file.

Seven mutations across seven distinct shipped-surface classes were all killed:
skill body, narrow placeholder, metavariable, role-agent card, `description:`
frontmatter, statusline shell comment, entity template.

On this tree: 77 findings → 0.

· commit 3ca397918 · `make check-fast` exit 0

### AC-4 — Error severity

The flip is one line; the rest is what it makes true or false. The documented
row moved back to the errors table — which is also the only one of the two with
a `Typical fix` column, so the remediation it had been carrying invisibly
renders again. Two CLAUDE.md claims about the rule not blocking a push were
corrected. D-0051 is left as written: it records the staging plan as decided,
and the plan executed as described.

The probe found a defect in the severity pin itself. It called itself a contract
between rule and doc but named the expected table as a literal, so reverting the
rule to warning left it green — it caught the doc drifting and never the rule.
It now derives the expected table from a live scan. Re-verified from four
directions: rule-alone and doc-alone each fail, and both-moved-together passes,
which is what distinguishes a contract from a layout assertion.

· commit 705a3cc72 · `make check-fast` exit 0 · `aiwf check` 0 errors

### Review round

Three fresh-context reviewers over the full change-set, sliced by concern
(production code / tests / prose). All three returned request-changes; ten
findings, every one real and fixed on the branch before wrap.

Two were false statements introduced by this milestone, both in surfaces a
reader trusts rather than verifies. The `aiwf-check` findings table described
`body-prose-id/malformed-shape` as firing on a number narrower than canonical
width; the kind patterns admit two digits for an epic and three for the rest, so
a narrow numeric is strict-form and resolves, and never reaches that subcode.
`ScanSkillBodyID`'s contract comment still said findings were warnings and named
the severity flip as future work, 45 lines above the line that had already made
it an error.

Two were vacuities in this milestone's own AC-3 test, both invisible without
mutation. Its phrases each recurred elsewhere inside the same section — one of
them as a prefix of two neighbouring row keys — so deleting the passage under
test left the assertion green; the phrases are now pinned on strings occurring
exactly once, verified by count. And the section-narrowing anchor failed open,
because splitting on an absent separator returns the input unchanged, so
renaming the anchoring heading widened the scope to the whole file with no
signal. It now refuses a missing anchor the same way a missing heading is
refused.

The rest: two `--reason` examples kept a development war-story after losing the
id that made it concrete, a bracket-synopsis metavariable inside a `bash` fence
was neither shell-safe nor still demonstrating the comma-separated list it
existed for, a spaced metavariable in command position read as two positional
arguments, three fenced blocks lost their comment column, a test name described
a property deliberately removed from it, and a comment claimed some templates
were clean when all four carried findings.

· commit 3e9a45f1f · `make ci` exit 0

## Validation

`make ci` exit 0 — race suite, lint, diff-scoped coverage audit, firing-fixture
meta-gate, `aiwf doctor --self-check` 29 steps. `aiwf check` on this tree: 0
errors, 1 warning (`provenance-untrailered-scope-undefined` — the branch has no
upstream, which is a property of the branch rather than of the work).

`skill-body-id` findings over the shipped trees: 115 → 0, at error severity.

Branch coverage: the only production change is one line, on a statement already
covered; no new branches. `AIWF_COVERAGE_BASE` against the milestone base passes.

## Deferrals

- G-0516 — neither comment-history rule catches a comment that goes stale
  because of a change elsewhere in the same file: the pre-push scan is
  diff-scoped to changed lines, and the whole-tree policy matches a fixed
  past-tense phrase set that present-tense narration evades. Found because this
  milestone shipped exactly that shape and both gates were green.

## Reviewer notes

- **A prior milestone's pin was deliberately weakened.** The `aiwf-add` skill
  test asserted its worked typo example "uses two distinct ids" — a guard
  against precisely the collapse this milestone performs. The form it protected
  cannot survive the shipped-surface rule: a real id is banned outright, and two
  distinct placeholders are not canonical either. The example now states the
  failure rather than staging it, and the test pins the behaviour it owes a
  reader — unvalidated flag, no verb-time signal, named downstream finding —
  scoped to its section. Measured: relocating the passage still fails the test,
  so the scoping is stronger than the regex it replaced.
- **The two id rules disagree about templates, and that is not a defect.** A
  shipped surface requires the canonical letter-N placeholder; an entity body
  rejects it. A template is scanned by both. It satisfies them only by placement
  — canonical in frontmatter, backticked in prose, rephrased inside HTML
  comments where a code span does not parse.
- **The metavariable convention has one forced exception.** `<ADR-id>` trips
  `skill-body-id`, because `ADR` is a kind prefix; the hyphenated form is
  therefore unavailable for that one kind. Template headings use the generic
  `<id>` and command positions the specific hyphenated form.
- **Worked-example transcripts render several distinct entities as one
  placeholder**, which is output the tool cannot produce. Accepted per this
  milestone's constraint — the titles carry the distinctions — and nothing pins
  that they continue to.
- **AC-3's zero-findings half is redundant** with the whole-tree assertion: any
  finding in those six files fails both. It is AC-scoped evidence, not
  independent coverage.
- **G-0514's premise did not survive the worklist.** It anticipated that
  metavariables and deliberately-exhibited bad shapes would need an exemption.
  Measured against every site that exists, all of them rewrote cleanly and no
  surface needed exempting. What remains is narrower and consumer-facing: the
  finding message prescribes canonicalization where rephrasing is the right
  advice.
