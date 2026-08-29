---
id: M-0325
title: Add an operator-driven end to aiwf authorize
status: draft
parent: E-0090
depends_on:
    - M-0323
tdd: required
acs:
    - id: AC-1
      title: The end mode ends the targeted scope without changing its entity's status
      status: open
    - id: AC-2
      title: Re-ending converges to a NoOp; naming no resolvable scope is refused
      status: open
    - id: AC-3
      title: The end mode is reachable from --help, tab-completion, and the authorize skill
      status: open
---
## Goal

Give a human a way to end an authorization scope deliberately, without changing the status of the entity it was opened on.

## Context

`AuthorizeMode` is a closed three-value set — open, pause, resume. A scope's only exit today is the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect of the status change. So withdrawing a delegation requires closing the work, and nothing in the log distinguishes a human ending a delegation from an entity happening to reach a terminal status.

G-0022 reserved an `aiwf-revoked-by:` trailer slot for a revoke verb that was never built. This milestone builds that surface. No existing invocation changes behaviour: the automatic end fires exactly where it already fires, and the one change to it is the predicate deciding which scopes it covers.

ADR-0047 settles the semantics. An end names its scope by authorize-commit SHA and defaults to the entity's sole candidate; ending covers scopes in `active` or `paused` state; and nothing undoes an end.

## Acceptance criteria

### AC-1 — The end mode ends the targeted scope without changing its entity's status

After the verb runs, the targeted scope's replayed state is `ended` and the entity's status is what it was before. The two assertions together are the point: an end that only worked by moving the entity to a terminal status would be the behavior this milestone exists to replace.

"Targeted" is what ADR-0047 defines: the scope named by `--scope <auth-sha>`, or the entity's sole non-ended scope when `--scope` is absent. More than one candidate and no `--scope` is a refusal listing them, which AC-2 covers.

### AC-2 — Re-ending converges to a NoOp; naming no resolvable scope is refused

Two behaviors, one rule ordering. Re-running the verb against a scope already `ended` returns exit 0, writes no commit, and reports the state as already holding. Naming a scope that does not resolve — a bad sha, an entity with no scope ever opened — is refused, not converged.

That ordering is the kernel's R1-before-R2: an argument must name something real before the verb asks whether the request is already satisfied. Converging on an unresolvable target would assert success for state that cannot exist.

### AC-3 — The end mode is reachable from --help, tab-completion, and the authorize skill

The flag appears in `aiwf authorize --help`, it is tab-completable, and the `aiwf-authorize` skill documents it. The completion half is already policed by the drift test that fails CI on a flag added without completion wiring.

## Constraints

- The automatic scope-end fires where it already fires; only the predicate choosing which scopes it covers changes. No existing invocation behaves differently.
- One mutation, one commit, or none — a converging re-run writes nothing and carries no `commit_sha`.
- The mode is a peer of `--to` / `--pause` / `--resume`, which are mutually exclusive; the new one joins that exclusion rather than combining with them. `--scope` modifies the end mode and is not itself a mode.
- What undoes an end is answered in ADR-0047, not invented here.

## Design notes

Scope state is a projection over git trailers, not stored state — `internal/scope` replays commits forward from the authorize commit. So ending a scope means writing a trailer that the replay reads, in the same shape the automatic end already writes, rather than mutating a record. `ReplayScopes` therefore needs no change: it already resolves `aiwf-scope-ends: <auth-sha>` by SHA.

An operator end stays distinguishable from an automatic one in history without a new trailer, because it rides an `aiwf-verb: authorize` commit rather than a `promote` or `cancel`.

The paused-scope fix is one predicate in `loadActiveScopeAuthSHAsForEntity` (`internal/cli/cliutil/provenance.go`), which today collects only scopes in `active` state.

The CHANGELOG entry belongs to this milestone: the surface is new and consumer-visible.

## Out of scope

- Removing or re-timing the automatic end. Only its predicate changes.
- Same-state convergence for a duplicate `authorize --to` re-grant (G-0460). It shares the targeting question and is a separate defect.
- Time-bound scopes, verb-set restrictions, and the rest of G-0022's extension list.

## Dependencies

- M-0323 — produced ADR-0047, which settles which scope the mode targets, what "ending" covers, and what undoes an end.
