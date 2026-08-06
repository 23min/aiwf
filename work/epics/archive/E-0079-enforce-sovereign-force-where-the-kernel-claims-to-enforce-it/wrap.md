# Epic wrap — E-0079

**Date:** 2026-08-06
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0079-enforce-sovereign-force-where-the-kernel-claims-to-enforce-it
**Merge commit:** pending — the merge to `main` is held; see *Handoff*

## Milestones delivered

- M-0291 — Wire trailer coherence at the apply seam (merged ff049a390)
- M-0292 — Give provenance-force-non-human a ratification path (merged e613abe4a)
- M-0293 — Correct the surfaces that claim force enforcement (merged 4489d8c9a)
- M-0294 — Derive the coherence rule lists from one declaration (merged cf28c1128)

## Summary

The kernel documented `--force` as human-only and enforced it for one verb of
four. A forced act by an agent committed, then raised an error-severity finding
at pre-push that no verb could clear — leaving history rewriting as the only
exit, in a repo whose tooling is policed against them. This epic made the
guarantee true where it is claimed, and gave the finding a way out.

The guard now sits at `verb.Apply`, the single seam downstream of both
trailer-assembly shapes, and refuses before anything is written. `aiwf
acknowledge illegal` clears the finding on a landed commit with a human's
written reason, leaving the acknowledged commit byte-identical. Every surface
that described the guarantee now states what the kernel does, and the coherence
rule set is described by one declaration the domain axis, the seam's enforced
subset, and the reachability check all derive from.

Scope shifted once, on evidence. The epic recorded a re-aiming of the
sovereign-dispatcher policy as decided; building it measured that a
dispatcher-layer guard cannot work, because whether `--force` names a sovereign
act depends on the verb and on tree state. The policy was retired instead
(D-0061), and the epic's scope was corrected to match. The final milestone was
likewise rescoped, from a Pin-and-bijection cell registry to a derived
declaration (D-0062).

## ADRs ratified

- ADR-0040 — prevent sovereign acts at the verb route, ratify at the history route

ADR-0029 was corrected rather than ratified: its Decision and Consequences named
a chokepoint that did not exist, and now name the guards `verb.Apply` actually
runs.

## Decisions captured

- D-0060 — the seam enforces the force-predicated rules, decided by satisfiability
- D-0061 — retire the sovereign-dispatcher policy; the force rule lives at the apply seam
- D-0062 — derive the coherence rule lists instead of a Pin-and-bijection cell registry

## Follow-ups carried forward

Discovered by this epic and deliberately left open:

- G-0544 — wire the contract verbs through the provenance decoration layer
- G-0545 — fold the coherence-guard seam policy into the commit-construction policy
- G-0546 — verb trailer sets are completed by the CLI layer after the verb returns
- G-0550 — force with no actor at all is refused by neither rule set
- G-0551 — nothing checks that the verb-side and check-side rule sets agree
- G-0552 — nothing bounds the Go build cache in the devcontainer

Referenced but not this epic's to close:

- G-0333 — the Tier-1 / Tier-2 override boundary. Only its finding-hint audit was
  folded in here; the documentation work stays with the gap.
- G-0023 — delegated `--force`. Out of scope by construction: it changes the
  provenance model, where this epic made the current model true.

## Doc findings

Scoped to the epic's change-set (`docs/adr/ADR-0029`, `docs/adr/ADR-0040`,
`docs/design/design-decisions.md`, `docs/design/legal-workflows-audit.md`,
`docs/design/provenance-model.md`).

Clean. Every intra-repo markdown link resolves; no `TODO`/`FIXME` markers; every
Go symbol cited in the changed docs exists in the source tree.

## Handoff

**The merge to `main` is not performed.** `main` is checked out by another
session and advanced 121 commits during this epic's final day. Mainline has been
reconciled *into* this branch (commit a66980f2a) and the full `make ci` gate
passes on the reconciled tree, so the branch is merge-ready — but the merge
itself, the epic's promote to `done`, and the push are left to whoever holds
`main`.

That reconcile carried one conflict worth knowing about. Main dropped the
`Typical fix` column from the shipped findings tables and added a policy pinning
the two-cell shape; this epic added a ratification paragraph to the same
section. The resolution takes main's structure and keeps the paragraph, which is
where this epic's assertions actually read — both policies hold.

Ready for the next epic: the coherence rule set now has one declaration, so a
rule added to it enters the domain, the seam's subset, and the reachability
check together. What is deliberately left open is the seam's *membership*
criterion — the design review measured that satisfiability, as D-0060 states it,
is a necessary condition rather than a sufficient one, so it under-determines
which rules belong at the seam. Nothing shipped depends on the answer, and
G-0544 is the follow-up that would retire the exclusion it was written to
justify.
