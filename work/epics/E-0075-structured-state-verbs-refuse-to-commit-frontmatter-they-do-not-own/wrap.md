# Epic wrap — E-0075

**Date:** 2026-08-02
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0075-structured-state-verbs-refuse-to-commit-frontmatter-they-do-not-own
**Merge commit:** <SHA — filled at merge>

## Milestones delivered

- M-0282 — Settle the write-scope precondition's seam, scope, verdict and escape hatch (merged bf77f46be)
- M-0283 — Add the shared write-scope precondition and route the single-entity verbs (merged 8cf803b1a)
- M-0284 — Cover the nested-path vector and call each multi-entity sweep in or out (merged 1fd9e2817)
- M-0285 — Fail CI when a new frontmatter-writing route bypasses the precondition (merged 1a243853f)
- M-0286 — Close the archive sweep's referrer and destination gaps (merged 2969dd17b)

## Summary

A structured-state verb re-serialized the whole entity file around the field it
computed, so any hand-edit sitting in the working copy — frontmatter or body —
rode into that verb's commit wearing the verb's own provenance. The epic settles
that a verb commits only what it computed, and enforces it at two seams: a
commit-side guard covering every path a plan carries, including nested entities
no verb named, and a claim-side guard for the converging case, which returns
before a plan exists and so never reaches the first. Multi-entity sweeps decline
per candidate rather than per verb, so one mid-edit file costs its own candidate
and not the sweep.

Scope grew once, deliberately. The first two milestones scoped the guard to the
target entity; measuring the nested and sweep cases showed that scope was wrong
rather than merely incomplete, and M-0284 and M-0286 widened it. The last
milestone found the sweep's two predicates deciding from different views of the
tree — one reading the working copy, the other the record — and made them agree.

## ADRs ratified

- ADR-0038 — refuse verb writes over HEAD-divergent entity content

## Decisions captured

- none as separate entities; each milestone's `## Decisions made during
  implementation` section carries its own, and ADR-0038 carries the epic-level
  ones (seam placement, verdict shape, the escape hatch, per-candidate scoping)

## Follow-ups carried forward

- G-0475 — the FSM history walker's rename-plus-status blind spot, which this
  epic's precondition masks incidentally without fixing
- G-0480 — after-the-fact detection of laundering already in history; the epic
  governs the source tree, that rule governs the commit log
- G-0486 — directory moves dereference symlinks and force mode 100644
- G-0488 — the loader's area normalization rides into the next verb's commit
- G-0493 — `edit-body`'s two modes judge frontmatter divergence differently
- G-0494 — verb refusals reach machine callers as prose
- G-0498 — verb commits bypass git's content filters
- G-0500 — duplicate id reachable via `edit-body` over a moved entity file
- G-0501 — `init`/`update` replace a symlinked CLAUDE.md
- G-0502 — a gitlink under a moved directory is stranded
- G-0506 — `PromoteACPhase` computes its refusal from working-copy bytes
- G-0507 — claim-guard effectiveness is pinned only by a hand-maintained table
- G-0508 — three near-copies of the `internal/verb` AST scan
- G-0509 — this epic's user-visible refusal is absent from CHANGELOG; closed by
  the wrap entry itself
- G-0511 — `LsTreePaths` filters in Go instead of passing a pathspec to git
- G-0512 — a directory at a move's destination is invisible to archive's decline
- G-0513 — masked-terminal report misses a candidate unparseable on disk

## Doc findings

Scoped doc-lint over the epic's change-set: no broken links, no orphans, no
stale CLI invocations — every backticked verb in ADR-0038 resolves against the
current binary, and every intra-repo link resolves.

One finding, fixed here: `CLAUDE.md`'s same-state-convergence section described
this epic's subject as open ("every structured-state verb commits a hand-edited
field as its own"). It now states what the two seams guarantee and which policy
holds them.

## Handoff

The precondition is in place and mechanically defended: `PolicyClaimGuardPresence`
fails CI when a converging route reaches neither guard, and `noOpClaimScopes`
records what each verb's claim is scoped to. A new verb inherits the discipline
by construction rather than by review.

Deliberately left open: detection of laundering already in git history (G-0480),
which needs a history-walking check rule rather than a write-time guard, and the
three archive-sweep refinements the last milestone's review surfaced (G-0511,
G-0512, G-0513). None of them can lose data — the first is throughput, the second
fails the verb rather than corrupting it, the third under-reports.

The guard's *effectiveness* — which paths each verb actually passes it — is
still pinned by a hand-maintained table a new verb does not join (G-0507). That
is the weakest remaining link in the epic's own guarantee.
