---
id: M-0095
title: Enforce human-only actor on aiwf promote E-NN active
status: done
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: non-human actor promoting epic to active is refused with override-hint error
      status: met
      tdd_phase: done
    - id: AC-2
      title: human actor promoting epic to active succeeds without override
      status: met
      tdd_phase: done
    - id: AC-3
      title: rule scoped to proposed-to-active edge; other epic transitions unaffected
      status: met
      tdd_phase: done
    - id: AC-4
      title: rule scoped to epic kind; other kinds unaffected by sovereign-act rule
      status: met
      tdd_phase: done
---

# M-0095 — Enforce human-only actor on `aiwf promote E-NN active`

## Goal

Make `aiwf promote E-NN active` a sovereign act: refuse non-human actors by default, mirroring the existing `--force` rule. The standard `--force --reason <text>` override remains available for any legitimate non-human invocation. This operationalizes G-0063's preflight row 5 ("`aiwf promote <epic> active` actor is `human/...` — refusal") and brings the principal × agent × scope provenance model to the epic-activation moment.

## Context

Today `aiwf promote E-NN active` accepts any actor without enforcement, which collapses the sovereign delegation moment into a routine FSM flip. The kernel's `--force` machinery already implements the "human-only with `--reason` override" pattern; this milestone reuses that machinery for the specific `epic → active` transition rather than inventing a parallel mechanism.

This is a standalone kernel rule. It does not depend on M-0094 (drafted-milestone finding); the two land in parallel and the skill in M-0096 consumes both.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- The verb's per-kind transition handling (in `internal/verb/promote*.go`) gains a sovereign-act check on the `epic / proposed → active` edge. Refusal returns a typed error referencing the rule and the `--force --reason` override.
- Test coverage spans: refusal path (non-human actor), explicit-`--force` path (succeeds with proper trailers), and the unaffected paths (human actor; non-epic kinds; other transitions on epic).
- A pre-implementation audit of automation paths in this repo (CI workflows, hooks, scripts) confirms no legitimate non-human caller exists that would break — if any surface; it either gains `--force --reason` or its invocation moves to a human actor.

## Dependencies

- None. Standalone kernel rule; lands in parallel with M-0094.

## References

- E-0028 epic spec.
- G-0063 — preflight checks table, row 5.
- CLAUDE.md *Provenance is principal × agent × scope* — the model this rule operationalizes.
- `internal/verb/promote*.go` — verb path for the transition; existing `--force` handling.

### AC-1 — non-human actor promoting epic to active is refused with override-hint error

Invoking `aiwf promote <epic-id> active` with an actor that does not start with `human/` returns a Go error from `verb.Promote`. The error message references the new sovereign-act rule (so a reader understands *why* the verb refused) and points at the `--force --reason "..."` override path (so a sovereign actor knows the unblock path). Test drives `verb.Promote` directly with a non-human actor against an in-memory fixture; asserts non-nil error and substring presence of the override hint.

### AC-2 — human actor promoting epic to active succeeds without override

Invoking `aiwf promote <epic-id> active` with a `human/...` actor succeeds in default mode (no `--force`, no `--reason` required). The commit's standard trailers land (`aiwf-verb: promote`, `aiwf-entity: <epic-id>`, `aiwf-actor: human/...`). Test drives `verb.Promote` with a human actor end-to-end against a `proposed` epic; asserts no error, status flipped to `active`, trailer set well-formed.

### AC-3 — rule scoped to proposed-to-active edge; other epic transitions unaffected

The rule fires only on the `epic / proposed → active` edge. Other epic transitions executed by a non-human actor — `active → done`, `proposed → cancelled`, `active → cancelled`, `done → active` (with `--force`), etc. — are not refused by *this* rule (other rules may still apply via separate mechanisms). Table-driven test covering each non-`proposed-to-active` transition with a non-human actor; asserts the rule's error message does not appear.

### AC-4 — rule scoped to epic kind; other kinds unaffected by sovereign-act rule

The rule is scoped to `entity.KindEpic`. Non-human actors invoking promote on other kinds — milestone (`draft → in_progress`), contract (`proposed → active`), gap (`open → addressed`), ADR (`proposed → accepted`), decision — are not blocked by this rule. Table-driven test covering each non-epic kind reaching its respective active/accepted/in_progress/addressed state with a non-human actor; asserts no rule-fired error (other unrelated errors are tolerated; we assert the absence of the sovereign-act message).

## Work log

<!-- Phase timeline lives in `aiwf history M-0095/AC-<N>`; the entries here capture
     one-line outcomes + the implementing commit's SHA (filled at wrap when the
     implementation lands as a single commit). -->

### AC-1 — non-human actor promoting epic to active is refused with override-hint error

Rule implemented as the `requireHumanActorForEpicActivation(kind, newStatus, actor)` helper in `internal/verb/promote_sovereign_epic_active.go`; wired into `verb.Promote`'s `!force` block alongside the existing `requireResolverForResolutionClass` check. Error message names the act as "sovereign", references the `human/` requirement, and points at the `--force --reason "..."` override — substring assertions cover each on the test side. · commit <wrap> · tests 1/1.

### AC-2 — human actor promoting epic to active succeeds without override

End-to-end test (`r.must(verb.Promote(...))`) confirms the happy default path: human actor + `proposed → active` lands cleanly with no flags. The runner's `must` helper applies the plan, so trailer well-formedness rides through `verb.Apply` and is implicitly asserted by the absence of errors there. · commit <wrap> · tests 1/1.

### AC-3 — rule scoped to proposed-to-active edge; other epic transitions unaffected

Table-driven test across three non-`proposed → active` epic transitions (`proposed → cancelled`, `active → done`, `active → cancelled`), each driven by a non-human actor. Asserts the rule's tell-tale "sovereign" substring is absent from any returned error. Covers the `kind == epic && newStatus == active` guard's false arm. · commit <wrap> · tests 3/3.

### AC-4 — rule scoped to epic kind; other kinds unaffected by sovereign-act rule

Table-driven test across four non-epic kinds — milestone, contract, gap, ADR — each invoked by a non-human actor on their kind-appropriate transition. The contract case is the most adjacent (contract also has `proposed → active`), so it's a load-bearing assertion that the rule's kind guard is correct. · commit <wrap> · tests 4/4.

## Decisions made during implementation

- **No new flag.** The rule reuses the existing `actor` parameter on `verb.Promote` (already derived from `git config user.email` per the kernel's identity convention). No new CLI flag, no new `verb.PromoteOptions` field. The override path is the existing `--force --reason "..."` machinery, which already requires `human/` actors via provenance coherence — so non-human + `--force` continues to fail at the coherence chokepoint, and humans + `--force` continue to work. This means the rule's chokepoint composes cleanly with the existing sovereign-act surface (`--force`, `--audit-only`) rather than inventing a parallel mechanism.
- **Helper file, not inline.** The check could have inlined in `promote.go` as three lines, but a separate file (`promote_sovereign_epic_active.go`) keeps the rule's documentation (G-0063 reference, scope conditions, override semantics) co-located with the implementation and matches the existing pattern (`promote_resolver_enforcement_test.go`, `auditonly.go`, etc. — verb-package files scoped to one concern).

## Validation

- `go test -race -count=1 ./...` — 25 packages, 0 FAIL lines, exit 0.
- `golangci-lint run ./internal/verb/` — 0 issues.
- `go test -coverprofile=… ./internal/verb/` — `requireHumanActorForEpicActivation` at 100.0% statement coverage; branch-coverage audit walked the two early-return arms (kind/status guard, human/-prefix actor) and the fall-through error path, each matched to a named test.
- Pre-implementation automation audit — no `aiwf promote` invocations found in `.github/`, `Makefile`, or `scripts/`. No automation paths need migration to `--force` or a human actor.
- Doc-lint sweep against the change-set — no broken references, no removed-feature docs. The one workflow doc example (`docs/pocv3/workflows.md:64`) continues to work for the canonical human-actor case.
- `aiwf check` (kernel planning tree from the worktree) — clean modulo the expected post-M-0094 advisory warnings (`archive-sweep-pending`, `terminal-entity-not-archived` for M-0094 awaiting sweep, and the benign `provenance-untrailered-scope-undefined` no-upstream warning).

## Deferrals

- (none)

## Reviewer notes

- **Composition with existing sovereign machinery.** The rule reuses the existing `actor` parameter on `verb.Promote` rather than inventing a new flag, and reuses the existing `--force --reason` machinery as the override. Non-human + `--force` continues to fail at the provenance coherence chokepoint (`aiwf-force requires a human/ actor`), so the override path is human-only by construction — no parallel mechanism, no new combinatorial surface to test.
- **Scope is intentionally narrow.** The rule fires only on `kind == epic && newStatus == active`. Generalizing to `contract-active`, `ADR-accepted`, or other sovereign-shaped transitions is out per the epic's design (Open Questions, downstream-question #2 in G-0063). If real friction surfaces — e.g., an `ai/...` actor mistakenly ratifying an ADR — that's a separate sovereign-act rule with its own milestone, not a quiet generalization of this one.
- **Why before ValidateTransition matters for ordering.** The check sits inside the `!force` block alongside `requireResolverForResolutionClass` (G-0096 precedent), AFTER `ValidateTransition`. Order rationale: if the transition is illegal at the FSM level (e.g., `done → active` without `--force`), the FSM error is the more fundamental signal and should fire first. If the transition is legal (`proposed → active`), my rule fires next on actor. This means an `ai/...` actor attempting an illegal-and-non-human transition gets the FSM error, not the sovereign-act error — that's the right shape (they need to fix the target status first; actor comes second).
- **Trailer convention for the wrap commit** — same as M-0094 (`aiwf-verb: implement`, `aiwf-entity: M-0095`, `aiwf-actor: human/peter`).

