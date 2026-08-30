# Epic wrap — E-0090

**Date:** 2026-08-30
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0090-make-closing-an-epic-a-sovereign-act-and-give-scopes-an-operator-driven-end

## Milestones delivered

- M-0323 — Settle what closing an epic is (merged `d9e5bf812`)
- M-0324 — Refuse a non-human actor on every terminal epic edge (merged `b707f6e69`)
- M-0325 — Add an operator-driven end to aiwf authorize (merged `93ef0e4b9`)

## Changelog entry

### Changed — E-0090: closing an epic is a human's act

Every edge into a terminal epic status — `active → done`, `active → cancelled`,
`proposed → cancelled` — now requires a `human/` actor, joining the activation edge
that was already gated. A non-human actor is refused at the verb, before anything is
written, and `aiwf cancel` consults the same closed set `aiwf promote` does, so both
spellings of a terminal edge are gated alike. Sovereignty here tracks irreversibility
rather than effort: cancelling a *proposed* epic discards a plan that never became
work, but `cancelled` is terminal whichever state it is reached from.

This changes an invocation that used to succeed. `aiwf promote <epic> done` run by an
agent exited 0 before; it is now refused, and the remedy is for a human to run it —
not `--force`, which a non-human actor cannot wield either. One historical commit in
this repo's own log qualified and was ratified.

Closes G-0646. See ADR-0047.

## Summary

The epic set out to make closing an epic mean something the kernel could check, and
to give a scope an exit that is not a side effect of closing work. Both landed.
ADR-0047 settled the two questions together because they answer the same one — who
may declare that authorized work is over — and the implementation followed it
without amendment.

Scope held, with one deliberate widening. The end mode was specified as additive,
and it is; but the paused-scope fix was folded in once M-0323 established that
ending covers every non-ended state, because shipping an operator end that covered
paused scopes while the automatic end stranded them would have left the two routes
disagreeing.

## ADRs ratified

- ADR-0047 — Gate every terminal epic edge; end a scope by naming it

## Decisions captured

- D-0081 — Ratification evidence is the kernel rule, not a policy-test duplicate

## Follow-ups carried forward

- G-0111 — the scope-end weld to the terminal promote survives by decision; the
  operator gesture it lacked now exists, and the claim was corrected at wrap rather
  than closed. The three kernel surfaces that work around the weld are what remains.
- G-0022 — item 1 of six (explicit scope end) delivered here; the other five
  extensions are untouched.
- G-0460 — repeat authorize leaves two active scopes on one entity with no finding.
  ADR-0047 names it as the signal that would reopen the targeting rule: if the kernel
  ever refuses a second live scope, `--scope` guards a state nothing can produce.
- G-0649 — the sovereign-act refusal carries no finding code, so it exits 2 rather
  than 1 and the legal-workflow spec cannot bind it to a cell.
- G-0651 — a backticked span in a flag's usage string renders as that flag's value
  placeholder in `--help`, across the registrations the gap enumerates.

## Handoff

The sovereignty surface is complete for epics: all four edges gated, one closed set,
one call site per verb, and the historical commit ratified. Extending it to
milestones is a separate decision ADR-0047 explicitly declines to make.

On the scope side, the operator gesture exists and the two end routes agree. What is
deliberately left open is the weld itself — the automatic end still rides the status
flip, and `aiwfx-wrap-epic`'s commit ordering plus both `internal/check/provenance.go`
carve-outs still exist because of it. Separating them is G-0111's remaining half and
wants its own epic.

## Doc findings

`wf-doc-lint`, scoped to the five docs this epic touched (ADR-0047, `design-decisions.md`,
`legal-workflows-audit.md`, `legal-workflows-first-principles.md`, `provenance-model.md`):
no TODO markers, every relative link resolves, no heading-level skips.

Six backticked invocations name verbs the CLI does not have — `aiwf install`, `prepush`,
`preview-merge`, `serve`, `reactivate`, `un-archive`. All six are the check's documented
exception: three sit in a deferred-extensions table, and the other three inside explicit
negations ("No `aiwf serve`", "No `aiwf reactivate` or `aiwf un-archive` verb exists").
None is on a line this epic touched.

## Wrap-artefact review

The artefact was read by a fresh reader before the closing gate, checking its claims
against the tree rather than against the epic spec. It found two blocking errors in the
`## Changelog entry` section, both corrected above before this file was committed.

The first is worth keeping: the entry claimed the paused-scope fix was "the one existing
invocation whose behaviour changes". ADR-0047 scopes that claim to the scope-end side and
warns in terms against stating it absolutely, because the gating change alters an existing
invocation too — `aiwf promote <epic> done` by an agent exited 0 before and is refused now.
Measured against binaries built either side of the epic, exactly as the ADR says. The
sentence had been correct in the milestone's own CHANGELOG entry, where it carried the
scope, and became false when lifted into an entry covering both halves of the epic.

The second was structural: `[Unreleased]` already carried entries for `--end` and for the
paused-scope fix, landed during M-0325, while the sovereignty gating — the epic's headline
breaking change — had no entry at all. Copying the draft in would have announced the scope
side twice and the gating side never. The entry now covers only what was unrecorded.
