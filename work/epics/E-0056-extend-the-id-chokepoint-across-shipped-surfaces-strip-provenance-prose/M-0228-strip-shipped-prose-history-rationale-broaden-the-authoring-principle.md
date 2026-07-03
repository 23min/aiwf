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
      status: met
      tdd_phase: done
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

## Work log

- **AC-1** — met. `4e3c0f76` (red — `PolicyM0228SkillsPolicyBroadenedPrinciple` +
  its real-tree test + three `firing_fixtures_multi_site_test.go` rows; all six
  section markers fail on the pre-broadening §"Skills policy") → `973c8876` (green
  — `CLAUDE.md` §"Skills policy" broadened to the full shipped-surface list and the
  history/provenance/rationale content class). Phases red→green→done ran live;
  `db167628` promoted met. 100% statement coverage of the new policy.
- **Prose cleanup (review backstop — not an AC)** — `f55c05c8`. Stripped the "v1
  tracking-doc is gone" asides (`aiwfx-start-milestone` ×2, `aiwfx-wrap-milestone`
  ×1, `templates/milestone-spec.md` ×2 — whole mention removed so the
  `embedded-rituals-no-retired-tracking-doc` policy stays satisfied), reduced the
  "why date" argumentation blocks (`templates/adr.md`, `templates/decision.md`) to
  imperative, and trimmed the Nygard "Provenance" war-story in `templates/adr.md`
  to a one-line orientation. Verified at the independent wrap review, not a test.

## Decisions made during implementation

- The vestigial statusline bullet in the Approach was dropped at start-milestone:
  `M-0227` already owned the full statusline comment rewrite, so this milestone's
  statusline residual was empty. Recorded in the setup `edit-body`; no decision
  entity required.
- The Nygard "Provenance" blockquote in `templates/adr.md` — not named in the
  original Approach — was **trimmed** (not removed, not left) to a one-line
  orientation, an operator call during the cleanup: it is provenance war-story in a
  shipped surface, the class this milestone codifies against, and leaving it would
  contradict the principle just written into `CLAUDE.md`. A prose-scope call, not
  durable architecture — no `D-NNN` minted.

## Validation

- `go build ./...` — clean.
- `make check-fast` (golangci-lint + `go vet` + `go test`) — all packages green; 0
  lint issues.
- `make coverage-gate` — diff-scoped branch-coverage audit, firing-fixture-presence
  meta-gate, and skill-edit-structural-test backstop all green;
  `PolicyM0228SkillsPolicyBroadenedPrinciple` at 100.0% statement coverage.
- `aiwf check` (worktree binary, real tree) — 0 errors, 12 pre-existing warnings; 0
  `skill-body-id` / `body-prose-id`.
- AC-1 red→green proven independently: the reviewer materialized HEAD and reverted
  only the broadened paragraph — all six markers went missing.

## Deferrals

None. No AC was deferred or cancelled; no work was punted to a gap. Enforcing the
history/provenance/rationale content class only for the id-shaped subset (the rest
held at review) is the milestone's deliberate, reasoned design (see the
evidence-split section above), not a deferral.

## Reviewer notes

- Independent fresh-context review (code-quality + prose-quality lenses):
  **approve**, no blocking findings. Every claim verified by measurement — the AC-1
  test's non-vacuousness by reverting the paragraph and observing red; section
  scoping by confirming the six marker words appear many times elsewhere in
  `CLAUDE.md` yet the reverted section still reports them missing; coverage,
  id-leak, and tracking-doc-policy each re-run.
- **Marker choice is deliberate.** The test pins four surface markers (`statusline`,
  `template`, `agent`, `guidance`) plus two content-class markers (`history`,
  `rationale`), all absent pre-broadening. `description` and `provenance` were
  omitted as required markers on purpose — both already appear in the section
  pre-broadening, so requiring them would make the assertion vacuous. This pins
  against a literal narrowing back to "just skill bodies"; semantic drift (keeping a
  marker word while dropping the real broadening) is out of a structural
  assertion's reach and accepted per the repo's section-scoping rule.
- **A pre-existing test also guards this paragraph** (`m0195`, asserting `cite no
  real entity id` plus five required phrases). The broadening retains all of them,
  so it stays green — confirmed by the reviewer and the full suite.
- **DRY (considered, not a defect):** the section extraction is an inline
  heading-walk, not the shared `extractMarkdownSection` helper. That helper lives in
  a `_test.go` file (unavailable to a production policy `.go` file) and lacks the
  parent-H2 (`## Go conventions`) scoping this policy needs. The inline walk is
  required, not avoidable duplication.
- **Content class is review-enforced by design** for the non-id subset — no
  non-brittle structural assertion exists (the crux of the evidence split). Future
  shipped-surface edits carry no mechanical guard against reintroducing
  history/rationale; the wrap-ritual review is the standing backstop. Accepted
  trade-off, reasoned in the AC-split.
