---
id: G-0489
title: Ratchet confirmed review findings into checks and define a review stop rule
status: addressed
priority: high
addressed_by_commit:
    - e001c3593
---
## What's missing

The review loop has no rule that turns a confirmed finding into a durable check,
and no definition of when reviewing is done. Every disposition point says "fix
it" and stops there.

- `wf-review-code` §8 (:82-94) classifies findings **blocking / track-for-later /
  non-issue** — an urgency axis. A blocking finding "must be fixed before the
  change merges." Durable capture is specified only for the deferred class.
- `aiwfx-wrap-milestone` step 2 (:39) — "fix every blocking finding as a
  corrective commit on the milestone branch." There is no path from a finding to
  a check; at best a prose line lands in `## Reviewer notes`.
- `wf-patch` mandates a `CHANGELOG.md` entry with no skip (:68, :197) while a
  regression test is only a judgment-gated escalation to `wf-tdd-cycle` (:180).
  A bug fix can land documented but unpinned — prose mandatory, oracle optional.

One skill already states the law correctly. `wf-property-test` step 5 (:56):
"Commit discovered counterexamples as regression seeds ... so the failure is
pinned and can't silently return", restated as a constraint at :96. It is stated
once, for one finding class, and generalized nowhere.

Two further absences compound it:

- **No stop rule.** Every review skill emits one verdict and terminates. None
  says when a review loop is over, so "converged" is undefined and the loop feels
  endless by construction. `aiwfx-wrap-milestone` (:39) re-dispatches a reviewer
  *scoped to the changed surface*, which is right for confirming a fix but means
  no round ever re-samples the whole artifact.
- **The objective/judgment split is unnamed.** It is implicit in
  blocking/track-for-later and never stated, so a taste preference phrased with a
  concrete suggestion (which `wf-review-code`'s anti-pattern list at :116-122
  explicitly asks for) lands as blocking and earns a fix.

## Why it matters

An objective defect measured against a fixed oracle converges under looping: each
round strictly reduces the set. A judgment disagreement does not — each
independent reviewer samples a different subset of an effectively unbounded
space, so more reviewers means more surface area, not less.

Fixing a finding without encoding it leaves the oracle the same size as before,
so the next fresh reviewer re-samples the same judgment space from scratch. That
is the "every round turns up new issues" behaviour: findings are disposed of, the
oracle never grows, and the loop cannot terminate because nothing defines its
end.

The ratchet converts a finding from judgment to objective permanently. Once
encoded it cannot resurface as a discovery, so the space that can still surprise
a reviewer shrinks every round. That is the mechanism by which a spec converges
while remaining incomplete.

## What is already mechanical

The law is not new machinery. It is instantiated three times in this repo, on
three different surfaces, and named zero times:

- `internal/policies/branch_coverage_audit.go` — diff-scoped, working-tree
  against base: every statement on a changed line must be tested or carry
  `//coverage:ignore`. For Go logic this **is** the ratchet.
- `internal/policies/skill_edit_structural_test_backstop.go` — a `SKILL.md` edit
  must land alongside a referencing structural test. The ratchet applied to
  prose, and the precedent that a non-code change can be mechanically required to
  arrive with a check.
- The `acs-tdd-tests-missing` rule (`internal/cli/check/tests_metrics.go:57`) —
  an AC at `tdd_phase: done` under a `tdd: required` milestone must carry an
  `aiwf-tests:` trailer. Opt-in and off here; its adoption problem is separate.

So this gap is a **naming and connective-tissue change over machinery that
already exists**, plus one prose law covering the disposition step no gate can
see: whether a finding got a check at all, or was quietly downgraded.

The distinction matters for consumers. All three arms live in
`internal/policies/` and ship to nobody. A consumer repo running aiwf gets the
guidance and the skills and none of that machinery — so downstream the prose
carries the whole weight, which is the argument for getting its wording exact.

## Options

1. **State the law once, cite it at each disposition point.** A new force in
   `wf-codebase-health` section D (which is already "Tests that pin behavior, not
   implementation", D1-D4), primed by one line in the embedded guidance's
   code-health block alongside the existing D1 line. Then five short edits that
   cite it rather than restate a divergent version: the `wf-review-code` verdict
   block, `aiwfx-wrap-milestone`'s step-2 loop and :39, `wf-patch`:180, and
   `aiwfx-plan-milestones`:134 (which surfaces a real amend-vs-defer fork and
   never hands off to `aiwfx-record-decision`). No new skills.
2. **Edit each skill independently** without a canonical statement. Cheaper per
   edit, but reproduces the condition this gap describes — five wordings that
   drift apart, which is why the rule reads five different ways today.
3. **Do nothing prose-side and rely on the existing mechanical arms.** Defensible
   inside this repo, where the coverage gate carries most of the load for Go
   logic; wrong for consumers, who get none of it.

Option 1, split across two changes: the spine (the force, the guidance line, the
`wf-review-code` verdict block), then the citation sites. The first is valuable
standing alone and establishes the vocabulary the second references.

## What the wording must get right

The rule is ignorable if any of these is loose, and one meaningless application
teaches a reader to skip it wholesale:

- **Scope it to findings about code or behavior**, not "every finding". A ratchet
  obligation on a prose nit is meaningless.
- **Name what counts as a pinning check** concretely — a Go test, a policy under
  `internal/policies/`, a kernel finding-rule, a fixture-validation script, or a
  lint rule; generalized for consumers as a test, a lint rule, or a check in the
  project's own gate. Unfalsifiable phrasing is a dead rule.
- **Do not overclaim discrimination.** "A check that fails without the fix" is the
  standard; `wf-vacuity` is the verifier and is advisory by design
  (`wf-tdd-cycle`:101). The wording must not imply mechanical proof it does not
  deliver.
- **The escape is explicit and reasoned** — a finding that cannot be pinned
  becomes a recorded decision or a gap, never a silent corrective. Same shape as
  `//coverage:ignore <reason>` and `//history:ok <reason>`.
- **Extend `wf-patch`'s no-logic carve-out (:80) to the regression-test
  mandate.** A regression test for a typo fix is absurd, and a rule that reads
  that way gets ignored in a way that generalizes.
- **Stop rule: intermediate fix-confirmations stay scoped; the pass that decides
  convergence is full-surface.** A terminator evaluated only against a narrowing
  re-scan certifies a slice, which is worse than no terminator because it reads
  as convergence.
- **A finding that reveals an uncovered requirement routes to `aiwf add gap`** —
  one durable sink, no ambiguity.

## Known residuals

Stated so the change does not promise enforcement it will not deliver:

- A fix that only **deletes** code leaves no changed statements, so the
  diff-coverage gate cannot see it.
- Whether a pinning check actually **discriminates** is `wf-vacuity`'s job and is
  LLM-judged by design. That tradeoff is deliberate and is not reopened here.
- A finding **reclassified** to non-issue rather than fixed is not detectable by
  any mechanism, mechanical or prose.
- The **stop rule stays prose**. The only mechanical form available is a shape
  check such as "promote to done requires non-empty `## Reviewer notes`", which
  is satisfiable with a junk line; and skipping the stop rule costs review
  efficiency, not correctness.

## Deliberately out of scope

**Requiring an AC to name the check that decides it**, in either a kernel rule or
a ritual preflight bar. It is the highest-leverage place a mechanical entry bar
could sit, and it is excluded for two reasons.

Chronologically it does not work: ACs are created at plan time, while the check
that decides one is typically a test written at the red phase weeks later. Naming
the test artifact is a forward reference to a symbol that does not exist and that
a rename silently breaks; naming the oracle's shape is available at plan time but
is prose, so it is only shape-checkable — the same weak, satisfiable-with-junk
class rejected for the stop rule above.

Independently of that, the AC cadence is load-bearing and tightly coupled —
`tdd_phase` ordering, the red-first diff-shape gate, and the `acs-tdd-audit`
projection interlock — and the risk of perturbing it exceeds the value of a shape
check. Revisit only with evidence that a missing oracle statement is the common
AC failure, which is currently believed and unmeasured.

## Related

- `wf-property-test` — the one skill that already states the ratchet, and the
  phrasing precedent to generalize.
- `wf-vacuity`, `wf-rethink` — the strong anti-churn forces this change does not
  touch. `wf-rethink`'s default-to-keep and obligation gate are the existing
  counter-pressure against review-driven churn.
- ADR-0006 — the skills policy governing where a rule of this kind is reachable.
