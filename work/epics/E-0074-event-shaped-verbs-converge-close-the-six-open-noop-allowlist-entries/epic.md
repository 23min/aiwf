---
id: E-0074
title: 'Event-shaped verbs converge: close the six OPEN NoOp allowlist entries'
status: proposed
---
## Goal

Finish the same-state convergence M-0281 started, so the convention the kernel
advertises holds everywhere it claims to.

Addresses G-0458, G-0459, G-0460 and G-0461.

## Context

M-0281 converged twelve operator-facing verbs and shipped
`internal/policies/verb_result_noop_invariant.go` to keep the convention from
rotting — but it left six allowlist entries marked OPEN, each an explicit IOU
rather than a by-design exemption. The policy therefore claims a discipline it
does not yet enforce everywhere, which is the same half-rolled-out condition
M-0281 was itself filed against.

Definition of done is **zero OPEN entries in that allowlist** — but nothing
asserts that today: the allowlist's `Reason` strings are read by no test, so "no
OPEN entries" is currently a grep over a comment. Making the bar real is part of
this epic's scope, not a precondition it inherits. And an entry rewritten with a
by-design reason rather than a NoOp assertion changes no behavior and adds no
test, so the rewrite branch needs its own evidence bar to satisfy the
mechanical-evidence rule.

One ordering edge runs out of this epic and into another. `PromoteACPhase` —
G-0458's target — writes frontmatter, so it falls inside E-0075's route list and
under that epic's first decision, which settles where a frontmatter precondition
sits relative to the same-state comparison. ADR-0038 settles it as two seams: a
commit-side guard at the top of `verb.Apply`, and a claim-side guard in each
verb's prelude that runs ahead of its same-state comparison.

## Scope

The six OPEN entries. Only one ordering edge is internal — G-0460 before the
`Authorize` entry; the other four are independent and can land in any order or
in parallel.

**Make the bar mechanical, first.** Until a test reads the allowlist's `Reason`
strings, "no OPEN entries" is unenforced and each subsequent closure is
self-reported. The policy scans *exported* entry points, so an unexported
composite branch is invisible to it and needs its own test.

**G-0459 — four of the six entries.** Five event-shaped verbs append a fresh
record on an identical re-run: `acknowledge mistag`, `authorize --to`,
`promote --audit-only`, `promote <id>/AC-N --phase --audit-only`, and
`cancel --audit-only`. `acknowledge illegal` already received a HEAD-walk
duplicate guard in M-0281; these did not. `acknowledge mistag` is the closest
analogue — `check.WalkAcknowledgedMistags` already walks HEAD for exactly the
relevant commits, so the detection capability exists and is unused. The
`--audit-only` trio needs a guard keyed on an existing audit *record*, because
their precondition is that the entity already sits at the target state.

One open question runs through them: what the duplicate guard keys on.
`acknowledge illegal`'s existing guard keys on the SHA alone, so a re-run with a
*corrected* `--reason` is silently discarded (measured). For the `--audit-only`
trio, whose entire payload is `--reason`, "mirror that guard" is therefore a
decision rather than a mechanical port.

**G-0461 — a composite `--for-entity` ack that suppresses nothing.**
`aiwf acknowledge illegal <sha> --for-entity <id>` emits `aiwf-entity` at the
id's full composite width and `check.WalkAcknowledgedSHAEntities` stores that
value canonicalized but otherwise verbatim, while the rule the flag exists to
suppress looks acks up through a path that rolls the touched id up to its parent
first. The composite-width ack never matches, so the operator sees a successful
acknowledgment and the finding keeps firing.

This is here for a shared cause rather than a shared verb name: the key on which
an ack is stored and the key on which it is read disagree, and G-0459's new
`acknowledge mistag` guard has to pick a key on exactly that axis. Settling one
without the other invites a second inconsistency in the same map. G-0461 is
pre-existing and wider than the convergence work that surfaced it — M-0281 fixed
only the verb's own duplicate guard, which had inherited the same rolled-up
lookup; the suppression path is untouched by that fix.

**G-0460 gates the `Authorize` entry only.** A repeat
`aiwf authorize <id> --to <agent>` exits 0, appends a second commit, and leaves
two simultaneously-active scopes with no check finding. The allowlist entry says
convergence may be the wrong fix, because a second grant can be a legitimate new
event.

What has to be decided is narrower than the entry implies. Multiple parallel
active scopes are already defined behavior: `docs/design/provenance-model.md`
states that a human may hold several at once and that the kernel resolves a match
to the most-recently-opened scope deterministically, and `verb.Allow` implements
exactly that. So resolution is not ambiguous and there is no missing invariant to
establish. The open question is whether an *exactly-duplicate* re-grant is a
distinct event or a same-state input — which is answerable directly, and does not
gate the other five entries. G-0460 also asks for a check rule so an
already-divergent tree is reported rather than silently carried; that is in
scope.

**Last, G-0458** — `promote <id>/AC-N --phase <same-phase>` refuses via the
TDD-phase FSM. This is a decision, not a defect: the phase ladder is audit-bearing
evidence and the verb carries a `--tests` payload, so convergence needs a
deliberate metrics carve-out rather than a mechanical repeat. Resolve by
converting with that carve-out, or by rewriting the allowlist entry with a
by-design reason. It goes last because it cannot be decided by implementing it.

## Out of scope

- **Write-scope preconditions** — whether a verb should run at all against a
  frontmatter-dirty entity (G-0463, G-0466), tracked as E-0075. Same layer,
  different axis, kept separate so a broad precondition change does not ride along
  with mechanical dedup work. The one coupling that is not separable runs the
  other way and is stated in *Context*: ADR-0038 settles where that precondition
  sits.
- **Prelude error-envelope uniformity** (G-0456). Shares the word "uniformity" and
  nothing else.
- **Rewriting duplicate records already in history.** These guards prevent new
  duplicates; existing commits stay as the record of what happened. This does *not*
  exclude detection: the check rule G-0460 asks for reports an already-divergent
  tree, and that is in scope.

## Constraints

- Zero OPEN allowlist entries is the bar, and the bar itself has to become
  mechanical — a `Reason` string read by no test enforces nothing.
- An entry resolved by rewriting its reason rather than by converging changes no
  behavior and adds no test. It still needs evidence under the
  AC-mechanical-evidence rule, so the rewrite branch carries its own bar.
- `internal/policies/verb_result_noop_invariant.go` scans *exported* entry
  points. An unexported composite branch is invisible to it and needs its own
  test rather than relying on the policy.
- A duplicate guard must not silently discard a corrected payload. The measured
  behavior of `acknowledge illegal` — a re-run with a corrected `--reason`
  dropped, because the guard keys on the SHA alone — is the failure mode any new
  guard has to answer for rather than inherit.
- `PromoteACPhase` writes frontmatter, so ADR-0038's claim-side guard applies to
  it: the precondition runs in the verb's prelude, ahead of the same-state
  comparison a convergence guard would add.

## Success criteria

<!-- Observable at epic close. Phrased as references to the lists above rather
     than reproduced counts. -->

- [ ] No entry in `internal/policies/verb_result_noop_invariant.go`'s allowlist
      is marked OPEN, and a test fails if one is added or left that way — the bar
      is an assertion rather than a grep over a comment.
- [ ] Every verb named in *In scope* either converges on an identical re-run
      (exit 0, no second commit) or carries an allowlist entry stating a
      by-design reason, and the allowlist test distinguishes the two.
- [ ] A re-run of an event-shaped verb whose payload *differs* from the recorded
      one is not silently discarded — the operator either gets the corrected
      record or is told why not.
- [ ] `aiwf check` reports a tree that already carries duplicate
      simultaneously-active scopes, rather than carrying it silently.
- [ ] `aiwf acknowledge illegal <sha> --for-entity <id>` with a composite id
      suppresses the finding it names, or refuses — it no longer records an
      audit commit that suppresses nothing.
- [ ] G-0458, G-0459, G-0460 and G-0461 are promoted to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| What the duplicate guard keys on for the `--audit-only` trio, whose entire payload is `--reason` | yes | decided at milestone-planning; the answer also determines whether `acknowledge illegal`'s existing SHA-only guard changes |
| Which key ack ingest and ack lookup agree on — roll up at ingest, roll up at emit, or look up both | yes | G-0461 leans roll-up-at-ingest; settled together with the guard key above, since both write the same map |
| Whether an exactly-duplicate `authorize --to` re-grant is a distinct event or a same-state input | yes, for the `Authorize` entry only | G-0460; does not gate the other five entries |
| Whether `promote <id>/AC-N --phase <same-phase>` converges with a metrics carve-out or keeps a by-design refusal | yes, for that entry only | G-0458; cannot be decided by implementing it, so it is decided before it is scheduled |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A duplicate guard keyed so as to ignore `--reason` silently discards a corrected reason, reproducing in three more verbs the defect measured in `acknowledge illegal` | high | the key is an explicit open question resolved before implementation, not a port of the existing guard |
| A by-design rewrite of an allowlist entry changes no behavior, so it lands with no test and leaves the bar unenforced | medium | the mechanical bar lands first, before any entry is rewritten |
| A convergence guard on `PromoteACPhase` interacts with E-0075's frontmatter precondition, and the two are designed against different assumptions | medium | ADR-0038 fixes both seams, so the convergence guard is designed against a known shape rather than in parallel with it; G-0458 stays scheduled last |
| Rolling ack keys up at ingest changes the verb's own duplicate-guard read path, which looks the composite spelling up first and falls back to the parent | medium | verified: with the composite key gone the guard still finds the parent-scoped cover — that fallback exists precisely because the two sides disagree about width — but the binding it reports back changes from the AC to its parent, and that value is what the NoOp message names. The cost is an operator-visible message, not a missed guard, and the ingest option is chosen with it in view |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- Make the definition of done mechanical — a test over the allowlist's entries so
  "no OPEN entries" is an assertion. Nothing depends on it, and everything after
  it is verified by it.
- Settle the ack key, then land the duplicate guard for `acknowledge mistag` and
  close the composite-`--for-entity` suppression mismatch. These share the map,
  so they land together.
- Duplicate guards for the `--audit-only` trio, keyed per the decision above.
- The `authorize --to` re-grant per G-0460's decision, plus the check rule for a
  tree that already carries duplicate active scopes.
- `promote <id>/AC-N --phase` — convert with a metrics carve-out, or rewrite the
  allowlist entry with a by-design reason. Last, because it cannot be decided by
  implementing it.

## References

- G-0458 — `promote --phase <same-phase>` refuses rather than converging
- G-0459 — event-shaped verbs append a fresh record on an identical re-run
- G-0460 — a repeat `authorize --to` leaves two simultaneously-active scopes
- G-0461 — composite `--for-entity` acks never suppress the rule they target
- ADR-0036 — same-status FSM transitions converge to NoOp, not refusal
- E-0075 — write scope; its first decision landed as ADR-0038
- ADR-0038 — the frontmatter precondition's two seams, the claim-side one ahead
  of each verb's same-state comparison
- `internal/policies/verb_result_noop_invariant.go` — the allowlist this epic empties
- `docs/design/provenance-model.md` — parallel active scopes and deterministic resolution
- CLAUDE.md §"Same-state convergence — resolve, then converge"
