---
id: M-0228
title: Strip shipped-prose history/rationale; broaden the authoring principle
status: in_progress
parent: E-0056
depends_on:
    - M-0227
tdd: advisory
acs:
    - id: AC-1
      title: CLAUDE.md Skills-policy states the broadened authoring principle
      status: open
      tdd_phase: green
---
## Goal

Shipped consumer surfaces carry no development history, provenance narrative, or
rationale/argumentation — only imperative, consumer-scoped instruction. This
repo's `CLAUDE.md` states the broadened authoring principle, so a future edit
that reintroduces history or rationale is held to it at review.

## Approach

`M-0227` already stripped the statusline's provenance comments (it owned the full
statusline comment rewrite), so the residual work here is the history/rationale
prose that carries no machine-detectable shape:

- the "the v1 separate tracking doc is gone" asides in `aiwfx-start-milestone`,
  `aiwfx-wrap-milestone`, and `templates/milestone-spec.md` — remove the whole
  tracking-doc mention (`PolicyEmbeddedRitualsNoRetiredTrackingDoc` requires any
  surviving "tracking doc" reference to carry "v1" on the same line, so softening
  a bare one is not an option — it goes entirely);
- the "Why date and decided_by are in the body ..." argumentation blocks in
  `templates/adr.md` and `templates/decision.md` — reduce each to the imperative
  instruction, drop the "why".

Extend `CLAUDE.md` § "Skills policy" (the existing "Shipped skill bodies cite no
real entity id" paragraph) to name the full surface list — `SKILL.md` bodies and
`description:` frontmatter, entity templates, role-agent cards, the guidance
fragment, and the statusline's comments — and add the content class: no
development history, no provenance tags, no rationale or war-stories. (The
reference/dead-link discipline is owned separately by the doc-link milestone,
which encodes it in the `aiwfx-record-decision` skill.)

Depends on the broadened check (`M-0227`) landing first: it mechanically removed
every *id-bearing* provenance tag from the scanned surfaces, so what this
milestone rewrites is the residual *non-id* history/rationale prose — "the v1 ...
is gone", the "why" blocks — which carries no stable machine-detectable shape.

## Acceptance criteria — evidence split (load-bearing; do not blur)

**Only mechanizable claims become met-with-a-test ACs. The prose-quality cleanup
rides the `aiwfx-wrap-milestone` review backstop and is deliberately NOT an AC.**

Why the split is load-bearing, not a shortcut: the repo rule "AC promotion
requires mechanical evidence" binds even at `tdd: advisory` — "I read it and it
looks right" is not evidence, and a doc-shaped AC needs a structural assertion
scoped to a named section. But "reads as imperative, consumer-scoped, no
war-stories" is a judgment no structural assertion can pin. Once `M-0227` has
stripped the id-bearing tags, the remaining history/rationale prose has no
machine-detectable shape. Forcing it into an AC would demand either a vacuous
test or a rule violation. So it is delivered as the milestone's work and verified
at the wrap review — never promoted to `met` against a test.

Formalized at start-milestone into the single met-with-a-test AC below, plus the
review-backstop item that is deliberately not an AC:

1. **(met-with-a-test AC → AC-1)** `CLAUDE.md` § "Skills policy" states the
   broadened authoring principle — the full surface list plus the history /
   provenance / rationale content class — pinned by a structural assertion scoped
   to that named section (not a bare substring grep).

2. **(review backstop — NOT an AC)** The named surfaces read as imperative,
   consumer-scoped instruction with no development-history aside, rationale
   block, or war-story. Delivered work, verified at wrap review, not against a
   test.

A second met-with-a-test AC was to be added *only* if a non-brittle structural
assertion presented itself (e.g. a targeted absence guard over a specific cleaned
file). None did — a phrase-coupled absence guard over the cleaned prose is exactly
the brittle/vacuous test the split exists to avoid — so this milestone ships with
the single mechanizable AC above, and that is correct, not a coverage gap.

### AC-1 — CLAUDE.md Skills-policy states the broadened authoring principle

`CLAUDE.md` § "Skills policy" states the broadened authoring principle for shipped
surfaces: the full surface list — `SKILL.md` bodies *and* `description:`
frontmatter, entity templates, role-agent cards, the guidance fragment, and the
statusline's comments — plus the content class it now forbids: no development
history, no provenance narrative, no rationale or war-stories (alongside the
pre-existing no-real-id rule). Mechanical evidence:
`PolicyM0228SkillsPolicyBroadenedPrinciple` in `internal/policies/`, which walks
the `CLAUDE.md` heading hierarchy to the `(## Go conventions, ### Skills policy)`
span and asserts each broadened-surface and content-class marker (`statusline`,
`template`, `agent`, `guidance`, `history`, `rationale`) is present *within that
section* — a section-scoped structural assertion, not a bare whole-file grep. All
six markers are absent from the section before the rewrite, so the test is red
until the paragraph is broadened; three fixtures in
`firing_fixtures_multi_site_test.go` (missing file, missing section, missing
markers) keep the policy's construction sites lit under the firing-fixture
meta-gate and cover every branch.
