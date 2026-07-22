---
id: D-0047
title: Contract-first AC timing and red-first ordering enforcement
status: accepted
relates_to:
    - G-0252
    - G-0440
---
# D-0047 — Contract-first AC timing and red-first ordering enforcement

> **Date:** 2026-07-22 · **Decided by:** human/peter

## Question

G-0252 sketched three red-first-ordering mechanisms — running the test suite
at `--phase green`, an `aiwf-red-commit` SHA trailer, an AC-scope cap — each
either language-coupled and expensive, or a self-reported claim vulnerable to
the same "existence not relevance" gap D-0038 named for the sibling
`--evidence` mechanism. Separately, milestones were observed sitting on main
with unpopulated AC entities: `aiwfx-plan-milestones` merges to main without
ever calling `aiwf add ac`, which happens instead inside
`aiwfx-start-milestone`'s preflight — one FSM stage later than G-0216/D-0039's
already-shipped completeness guard fires (G-0440). Both surfaced from one
conversation about making TDD-cycle discipline hold structurally rather than
by LLM vigilance. Does each get a mechanism, and what should each be?

## Decision

1. **Red-first ordering** (G-0252) is enforced by a working-tree diff-shape
   check on the AC's TDD-phase promotes, not test execution or a
   self-reported SHA trailer. The gate attaches to the **live `"" -> red`
   promote** — the event that means "a failing test has been written and
   shown to fail." `aiwf promote M-NNN/AC-N --phase red` refuses if the
   working tree's diff against HEAD touches any non-test path; `--phase
   green` refuses unless a non-test path is dirty now (a stateless check on
   the current diff — the verb keeps no red-time snapshot, and none is
   needed: ordering is enforced by the *pair* of gates, test-only-dirty at
   red and impl-dirty at green, not by a "grown since red" comparison the
   verb cannot compute). Test-path classification is a glob predicate: the
   `areamatch` doublestar matcher is reusable, but the test-path glob set
   itself is a **new config surface** — the areas `paths:` config maps
   source to workstreams, not test-vs-source, so it is not the classifier.
   This gate requires a real `"" -> red` event to exist, which today it does
   not: `aiwf add ac` seeds `tdd: required` ACs directly at `red`
   (born-at-red), spending the transition before any test is written.
   Correcting that seeding so ACs are born at the pre-cycle `""` state is a
   **prerequisite**, tracked in G-0441. The check proves file-touch
   ordering; it does not prove the test failed at red-time — that judgment
   stays with `wf-tdd-cycle`'s own "confirm they fail for the right reason"
   step, `wf-vacuity`, and `wf-review-code`, matching the boundary D-0038
   already drew between mechanizable structural claims and semantic judgment.
2. **AC-entity creation and content-filling move from
   `aiwfx-start-milestone`'s preflight into `aiwfx-plan-milestones`**
   (G-0440), before its merge-to-main step: `aiwf add ac` and each `### AC-N`
   body land during planning, not deferred to the worktree. A new
   warning-severity check-time finding (extending `internal/check/acs.go`
   alongside `milestoneDoneIncompleteACs`) surfaces a `draft` milestone with
   zero ACs or empty AC bodies on `aiwf check`/`aiwf status` — warn, don't
   block, per D-0039's own block-at-transition/warn-at-rest split, since
   `draft` is a legitimate mid-planning state that shouldn't be punished the
   way a real FSM transition is.

## Reasoning

**Why diff-shape over test execution.** Running the suite at `--phase
green` and checking fail-then-pass is the strongest possible signal, but
D-0038 already rejected the same toolchain coupling (`go test -list`) for
`--evidence` on cost and stack-agnosticism grounds — aiwf ships guidance for
five languages, and baking one's test-discovery/execution model into a
kernel check gives the other four nothing. The same objection applies here
without qualification.

**Why diff-shape over a SHA trailer.** A self-reported `aiwf-red-commit
<SHA>` trailer proves a commit exists and is reachable before green — it
does not prove *that commit's diff* contains the failing test, only that
some earlier commit does. That is D-0038's "existence, not relevance"
critique one layer down: the trailer can point at an empty scaffold or an
unrelated commit and still pass. A diff-shape check derived from the actual
commits' file-touch pattern doesn't have this hole — it isn't a claim about
a commit, it's a structural fact about which paths changed and when.

**Why this is the right stopping point, not further toward correctness.**
The diff-shape check cannot tell whether the red-phase test actually fails,
only that no implementation path was touched yet. Closing that residual gap
requires either running code (rejected above) or a human/LLM judgment about
relevance — which is exactly what `wf-vacuity` (does the test actually catch
a bug) and `wf-review-code` (AC-coverage discipline) already do at wrap.
Building a second mechanism to duplicate that judgment mechanically would
repeat D-0038's mistake in the opposite direction: oversized cost for a
thinner guarantee than the reviewer discipline already provides.

**Why this is low-friction.** An honest cycle already has test files dirty
at red-time and nothing else — the check costs nothing extra on the common
path, unlike a new trailer or flag that has to be remembered. It is also
agent-agnostic: it inspects working-tree state, not authorship, so it
applies unchanged whether a human, the main assistant, or a future per-AC
subagent (the deferred E-0019 model) wrote the commits.

**Why the gate attaches to the `"" -> red` promote, and why born-at-red must
be fixed first.** The gate is only coherent when `red` is an honest achieved
state — "a failing test exists." G-0286 (addressed) already ratified that
meaning, and that an untouched AC's honest phase is *absent* (`""`), not
`red`; the check layer enforces it (`internal/check/acs.go` treats an absent
phase as legal until `met`). But `aiwf add ac` still seeds `tdd: required`
ACs at `red`, so the `"" -> red` transition — the natural "I wrote the
failing test" event — never fires on the honest path: the FSM refuses `red ->
red`, and `wf-tdd-cycle` tells the operator to skip the red promote. A gate
on a promote that never runs fires zero times. Fixing the seeding (G-0441)
restores the event and gives the gate a real, semantically-honest home — it
is a correctness fix the kernel already committed to via G-0286, not a
concession invented for this gate.

**Why AC-entity timing moves to planning, not just a start-milestone
reminder.** G-0216/D-0039 already established that "fill the contract before
coding" must be mechanical, not vigilance — a milestone's AC bodies being
empty right up until someone begins implementation is exactly the failure
mode that guard exists to close. The residual gap is that the guard's
trigger (`draft → in_progress`) fires *after* the milestone has already been
visible, empty, on main since `aiwfx-plan-milestones`' own merge. Moving the
actual `aiwf add ac` call earlier — into the ritual step that already fills
the milestone's other prose sections — closes the visibility gap at its
source instead of only at the point someone starts the work.

**Why warn, not block, at the planning stage.** `draft` is not a
terminal-feeling state the way `in_progress`/`done` are — a milestone can
legitimately sit in draft across several planning sessions while its shape
settles. D-0039 already chose block-at-transition (a real FSM move) over
block-at-rest (a milestone merely existing in a state) for exactly this
reason at the `done` end; the same logic applies at `draft`. A warning
surfaces the gap on every `aiwf check`/`aiwf status` without forcing a
planning session to finish in one sitting.

## Consequences

- G-0252 is refined, not closed — its "Candidate mechanisms" list is
  replaced by the diff-shape mechanism decided here.
- G-0440 captures the AC-entity-timing fix (ritual sequencing change to
  `aiwfx-plan-milestones` plus the new check-time warning) — filed
  separately since it is architecturally distinct from G-0252 (ritual +
  check-time finding, vs. a verb-time mechanism).
- Point 1 has a hard prerequisite: the born-at-red seeding fix (G-0441) must
  land first, so the `"" -> red` promote is a live event the gate can attach
  to. G-0441 also carries two born-at-red consequences to sweep — reversing
  `wf-tdd-cycle`'s "skip the red promote" guidance, and reconciling the
  `--tests`-at-`add` flag.
- Implementing point 1 touches `internal/verb/ac.go` (both `PromoteACPhase`
  and the `AddAC` seeding path) and adds a new test-path glob config surface
  (schema key, `aiwf.yaml` parsing, validation, completion) — the areas
  `paths:` config is not reusable as the classifier; only the `areamatch`
  glob predicate is.
- Implementing point 2 touches the `aiwfx-plan-milestones` ritual text and
  `internal/check/acs.go`.
- Neither mechanism is scheduled yet — this decision settles *what* to
  build, not *when*; sequencing is a future milestone-planning question.
