---
id: G-0226
title: aiwf acknowledge-illegal hard-requires SHA reachable from HEAD
status: addressed
discovered_in: M-0161
addressed_by_commit:
    - 7e3d3627
    - e73dc9bb
---
## Problem

`aiwf acknowledge-illegal <sha>` enforces a hard precondition that the target SHA must be reachable from HEAD via `git merge-base --is-ancestor <sha> HEAD`. The verb refuses with exit 2 ("SHA … is not reachable from HEAD") when the precondition fails.

This precondition is correct for the verb's original use case: silencing findings on commits the operator can see in `aiwf history`. But M-0161/AC-5 introduced a new finding class — `isolation-escape-orphaned-ai-commit` — that fires on commits which are by construction **unreachable from HEAD**: an AI-actor commit force-pushed off a ritual branch's tip, preserved in the local object store only via the reflog. The orphan IS what the rule names; the rule's own definition guarantees `merge-base --is-ancestor` will fail.

AC-5 body line 349 assumed the existing ack mechanism composes cleanly:

> "The existing `aiwf acknowledge-illegal <sha>` verb silences the warning per its existing mechanism (writes an empty commit with `aiwf-force-for: <sha>` + human actor + reason); no new override path needed."

This is false. The verb refuses every force-push orphan unconditionally.

## Observed in

M-0161/AC-5 cycle, 2026-06-04. The "Cell 5: force-push orphans AI commit + aiwf acknowledge-illegal → silent" scenario in [`internal/cli/integration/isolation_escape_force_push_scenarios_test.go`](../../internal/cli/integration/isolation_escape_force_push_scenarios_test.go) failed during the AC-5 RED→GREEN cycle with:

```
aiwf acknowledge-illegal: SHA "<orphan>" is not reachable from HEAD (git merge-base exit 1)
```

The scenario was deferred from the AC-5 deliverable with an in-test carve-out comment; see [D-0020](../decisions/D-0020-m-0161-ac-5-cell-5-orphan-acknowledgment-deferred-to-verb-extension.md) for the architectural deferral.

The real-world consequence: the kernel repo itself has a M-0120-era orphan on `epic/E-0033-pin-legal-kernel-verb-workflows-mechanically` (commit `af1051d1`, "feat(policies): add ADR-0011 structural test (M-0120/AC-2)") that fires the new warning on every `aiwf check` and cannot be silenced via the existing verb.

## Why parked

Three resolution paths exist; the choice belongs to a future cycle that explicitly scopes the verb-side surface:

1. **Extend `aiwf acknowledge-illegal` with `--allow-unreachable`.** A new flag (sovereign-gated: requires `--reason` non-empty + human actor, same as today's verb) that bypasses the reachability check. The verb's per-SHA closed-set scoping stays put. Simplest path; preserves the verb's single-surface promise.

2. **Introduce a separate verb `aiwf acknowledge-orphan <sha>`.** Lifts the reachability constraint but keeps it for the existing verb (so a typo'd SHA on `acknowledge-illegal` still refuses noisily). Adds a second surface; clearer semantics; more code.

3. **Rewrite AC-5's composition claim.** Treat orphans as unsilenceable by design — the warning fires forever until git GC actually removes the object. This is operator-discipline only and accepts the af1051d1-class residual debt indefinitely.

The decision belongs to a verb-design cycle, not the AC-5 wrap. Until it's made, force-push-orphan acknowledgment is unavailable.

## Mechanical assertion shape (when implemented)

Whichever path resolves this, the implementing milestone must add:

- An E2E scenario under `internal/cli/integration/isolation_escape_force_push_scenarios_test.go` re-introducing the cell-5 test (force-push orphan + ack → silent) with the new verb shape.
- Symmetric test that the new flag/verb refuses without `--reason` and without human actor (sovereign-gate preserved).
- Update the AC-5 body matrix row at [`work/epics/E-0030-…/M-0161-…md` line 362](../epics/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) to remove the deferral note and assert the composition again.
- Update the in-test carve-out comment at [`internal/cli/integration/isolation_escape_force_push_scenarios_test.go`](../../internal/cli/integration/isolation_escape_force_push_scenarios_test.go) lines 114-135 — remove the deferral block, re-add the scenario.

## Out of scope

- **Pre-existing reachability-class refusals.** The verb refuses other reachability cases (e.g., a typo'd SHA, a SHA from a different repo) — those refusals are correct. The new surface should narrowly cover the orphan case, not loosen the general reachability check.
- **Auto-detection of orphan vs typo.** The verb is told the SHA; it should not try to distinguish "this is an orphan you meant" from "this is a typo you didn't". The operator names the orphan; the `--reason` (or a dedicated verb) is the consent surface.
- **Orphan history reconstruction.** The verb writes a metadata-only acknowledgment commit; it does NOT bring the orphan back into HEAD's reachability set. If the operator wants the commit restored, that's a different verb (`git cherry-pick` / `git update-ref`), not within acknowledgment scope.

## References

- [D-0020](../decisions/D-0020-m-0161-ac-5-cell-5-orphan-acknowledgment-deferred-to-verb-extension.md) — the deferral decision recorded at AC-5 wrap
- [`internal/verb/acknowledgeillegal.go`](../../internal/verb/acknowledgeillegal.go) — the verb whose reachability check is the chokepoint
- M-0161/AC-5 (G-0205) — the cycle that surfaced this dependency
- AC-5 body line 349 — the false-composition claim this gap retires
- `af1051d1` — the real-world M-0120-era orphan demonstrating the operator-visible cost
