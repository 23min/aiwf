---
id: M-0292
title: Give provenance-force-non-human a ratification path
status: done
parent: E-0079
tdd: required
acs:
    - id: AC-1
      title: Acknowledging a forced non-human commit clears its finding
      status: met
      tdd_phase: done
    - id: AC-2
      title: The acknowledged commit is unchanged and its reason is readable
      status: met
      tdd_phase: done
---

## Goal

Let a human clear the finding a forced non-human act leaves behind — through a
verb, with the reason recorded — instead of rewriting history.

## Context

The rule walks git history and fires at error severity, so the commit blocks the
push. `aiwf acknowledge illegal` does not clear it: the rule is absent from the
consumer roster the acknowledgement mechanism reads. What remains is re-authoring
the commit or amending a human actor into it — history rewrites, in a repo whose
tooling is policed against them.

This is needed whether or not M-0291 lands. That milestone closes the verb route,
but the rule fires on commits no verb produced, so the history route keeps
producing findings that nothing can clear.

## Acceptance criteria

### AC-1 — Acknowledging a forced non-human commit clears its finding

After the acknowledgement, the finding for that commit is gone on the next check
run.

An unacknowledged sibling in the same tree still reports. Without that half, a
passing test cannot distinguish an acknowledgement keyed on the commit from the
rule having been switched off — which is the failure mode worth guarding, since
the epic forbids clearing the finding by weakening it.

The roster joined must be the one that drives enforcement, not the one that
enumerates the rules for readers. Two rosters exist and the chokepoint checks
them for different properties, so joining the wrong one passes the policy while
changing nothing an operator can observe.

Evidence: a real-repo test acknowledging one forced commit with a second left
unacknowledged, asserting exactly one finding survives.

### AC-2 — The acknowledged commit is unchanged and its reason is readable

Ratification records; it does not rewrite. The acknowledged commit is
byte-identical afterwards — same tree, same parents, same trailers — and the
reason is readable from the acknowledging commit itself, matching how the
existing acknowledge verbs already behave.

Evidence: HEAD's reachable commit list compared before and after — it must be
the old list with exactly one entry appended — plus the reason read out of the
acknowledging commit's `aiwf-reason:` trailer.

Comparing the acknowledged commit's own bytes would assert nothing. A commit
object is content-addressed and immutable, so re-reading it after an amend or a
rebase returns the bytes it always had; the rewritten copy is a different object
and the original merely becomes unreachable. The reachable list is what a
rewrite changes. Reading the reason through the trailer rather than as a
substring of the commit matters for the same reason: the verb writes it to both
the message body and the trailer, so a substring match passes with the durable
carrier gone.

## Constraints

- The ratification path is human-only and carries a written reason.
- `provenance-force-non-human` stays at error severity. This milestone adds a way
  to clear the finding, never a way to ignore it.

## Design notes

- **An acknowledgement is keyed on the commit, not on the rule.** It therefore
  covers every rule consuming the roster for that SHA. Accepted as-is: an
  acknowledgement is a judgment about a commit, which is what the existing
  consumers already assume. Revisit only if someone needs to accept one rule
  while another still blocks — recorded as a non-blocking open question on the
  epic.

## Surfaces touched

- `internal/check/provenance.go` — the guard, ahead of all three rule groups.
- `internal/check/acks.go` — the consumer roster and its doc.
- `internal/check/hint.go` — the derived ratification sentence and the
  ratifiability predicate.
- `internal/cli/check/provenance.go`, `internal/cli/check/check.go` — the gather
  wiring.
- `internal/policies/acks_helper_lift.go` — both policy rosters.
- `internal/cli/acknowledge/illegal.go`, `internal/verb/acknowledgeillegal.go` —
  the verb's help and doc.
- `internal/skills/embedded/aiwf-check/SKILL.md`,
  `internal/skills/embedded/aiwf-acknowledge/SKILL.md` — the shipped surfaces.
- `docs/design/provenance-model.md` — §Ratification.
- `CHANGELOG.md`.

## Out of scope

- The verb-route wiring (M-0291).

## Dependencies

- None. Independent of M-0291 and can land in either order.

## Work log

### AC-1 — Acknowledging a forced non-human commit clears its finding

`RunProvenance` joined the `ackedSHAs` consumer roster and now skips an
acknowledged commit ahead of all three of its rule groups · commit `c6046dcf6`,
corrected to per-commit scope in `77d6f93d8` · tests 6/6 green at the AC.

### AC-2 — The acknowledged commit is unchanged and its reason is readable

Characterization pins over the existing verb; no implementation change was
needed, which is what the AC predicted ("matching how the existing acknowledge
verbs already behave") · commit `c6046dcf6` · tests 2/2 green.

### Review corrections

`77d6f93d8`, `4f08b9513`, `ae8338aff` — the three corrective commits from review
rounds 1–3. Their subjects and bodies carry what each round found.

## Decisions made during implementation

- **The exemption is per-commit, not per-code.** The milestone shipped a
  narrower first cut that cleared only the two sovereign-trailer codes; measured
  against the real binary, an operator was still blocked, because a forced act by
  an authorized agent also raises `provenance-trailer-incoherent` whose message
  restates the rule just ratified. The argument and the worked case live in
  `docs/design/provenance-model.md` §Ratification, which is the single home; the
  epic's open-questions row already recorded this scoping and needed no change.
  No `D-NNNN` was filed — a decision entity would be a fourth copy of an argument
  the Normative doc already carries.
- **`provenance-audit-only-non-human` was folded in** alongside the force rule at
  the operator's direction. It is the same rule shape with the same dead end, and
  the milestone body's scope names only the force rule.
- **Discoverability was added as scope, not deferred.** Review found the
  capability shipped with every surface still naming remedies impossible for a
  landed commit — one of them advising `git commit --amend`, the rewrite this
  epic exists to avoid. The remedy is now derived for the whole ratifiable family
  from one constant in `HintFor`. No AC covers this; it is recorded here rather
  than as a gap because the work is done and the checks landed with it.

## Validation

- `make check-fast` — clean (unit tests + full `golangci-lint`).
- `AIWF_COVERAGE_BASE=epic/E-0079-… make coverage-gate` — clean.
- `aiwf check` — 0 errors, 1 warning
  (`provenance-untrailered-scope-undefined`; the branch has no upstream).
- `go test ./internal/check/ ./internal/policies/ ./internal/cli/integration/
  ./internal/verb/` — all `ok`.
- Behaviour measured against a binary built from `HEAD` in disposable repos: both
  the bare and the fully-decorated forced shapes exit 0 after an acknowledgment,
  using ack commits written by an earlier binary — the record did not change,
  only the reading of it.

## Deferrals

None. Every finding from the three review rounds was fixed in-branch; the checks
that pin them landed in the same commits.

## Reviewer notes

Three rounds, six independent reviewers, fourteen confirmed defects. Two are
worth a future reader's attention because no gate here would have caught them:

- **Round 1 — the milestone's Goal was unmet while every AC test passed.**
  Clearing one finding code left a second rule blocking the same push. The AC
  test could not see it because it filtered to one code. The tests now assert at
  two scopes: per code (the named rule clears, its unacknowledged sibling still
  fires) and per commit (no error-severity finding names the acknowledged commit
  afterwards). The second is the one that would have caught it.
- **Round 3 — a test written to close a vacuity gap was itself circular.** It
  enumerated the hint table filtered by the predicate under test, then asserted
  the string that predicate causes to be appended. Denylisting two codes the
  rules genuinely emit stripped their remedy with the suite green. It now derives
  the emitted set by driving `RunProvenance` over the fixtures and asserts both
  directions.

Judgment findings declined, so a later reviewer meets a decision rather than a
blank:

- **Rule-scoped acknowledgment (`--rule <code>`)** was proposed as the principled
  long-term shape. Declined here: it needs a new flag, a new trailer and a
  per-rule roster, and the epic defers it explicitly. The per-commit scoping is
  what the epic recorded.
- **A `D-NNNN` for the scoping decision** was requested twice. Declined — see
  *Decisions made during implementation* above.
- **`aiwf acknowledge illegal` verifies only that the SHA resolves**, never that
  it carries a sovereign trailer or fires any rule, so a SHA can be acknowledged
  pre-emptively. Pre-existing; this milestone widens what such an ack would
  silence. Left alone: adding a "must currently fire" precondition would make the
  verb's answer depend on check state at write time, which is a different design
  than the one shipped.
- **The blanket ack's reason is not readable through `aiwf history`.** A
  blanket acknowledgment carries no `aiwf-entity:` trailer, so it renders in
  neither `aiwf history <id>` nor `aiwf history <sha>`. The design doc and the
  verb's help now say so plainly rather than claiming otherwise. Making `aiwf
  history` accept a SHA is a real improvement and belongs to that verb, not here.

