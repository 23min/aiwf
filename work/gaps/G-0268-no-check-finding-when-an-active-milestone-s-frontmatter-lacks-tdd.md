---
id: G-0268
title: 'No check finding when an active milestone''s frontmatter lacks tdd:'
status: open
discovered_in: E-0016
---
## What's missing

`aiwf add milestone` hard-requires `--tdd <required|advisory|none>` at creation time (G-0055 layer 1, commit `09fd0ccf`), so every milestone created through the verb declares its TDD policy explicitly. But the verb is the *only* chokepoint. A milestone whose `tdd:` field is later stripped by a hand-edit, or one created by a path that bypasses the verb (`aiwf import`, a raw file write), silently reverts to the kernel's "absent ⇒ `tdd: none`" treatment with no surface reporting it. `aiwf check` has no `milestone-tdd-undeclared` rule, so the post-creation drift goes unseen.

## Why it matters

The hard refusal makes the *creation* path safe, but "framework correctness must not depend on LLM behavior" wants the guarantee to hold regardless of how the milestone got onto disk. The creation chokepoint structurally cannot catch a field stripped after the fact, or an entity that never went through `aiwf add milestone`. The check is the authoritative backstop the creation chokepoint can't be. The concrete uncaught holes: (a) a hand-edit removes `tdd:` from an in-flight milestone; (b) `aiwf import` ingests a milestone with no `tdd:`.

The code already anticipates this rule. `internal/config/config.go`, `internal/check/entity_body.go`, and `internal/check/entity_body_test.go` carry forward-references — "`milestone-tdd-undeclared` will join the same bumper when its rule lands" — pointing at the now-cancelled E-0016 / M-0065. This gap carries that surviving slice forward; those references get re-pointed here when the rule lands.

## Scope

- New `aiwf check` rule `milestone-tdd-undeclared`: emits a **warning** for any non-archived milestone whose frontmatter lacks `tdd:` (empty string and explicit `null` treated as absent). Archive-scoped via `entity.IsArchivedPath` (the M-0086 / ADR-0004 pattern already used by `acs-tdd-audit`), so the 61 grandfathered `done` milestones produce no findings.
- Severity escalates to **error** under `aiwf.yaml: tdd.strict: true` by joining the existing `check.ApplyTDDStrict` bumper (M-0066) — no new config knob, single source of truth for the project's TDD strictness posture.
- The finding includes the milestone id, file path, and a hint pointing at `--tdd <required|advisory|none>` for new milestones (hand-edit the frontmatter for grandfathered ones).
- Pinned test for the no-retroactive property: a grandfathered-shape fixture (every AC `met`, no `tdd_phase`) produces the `milestone-tdd-undeclared` warning only when non-archived, and never an `acs-tdd-audit` finding.
- `aiwf-check` skill gains a findings-table row (discoverability per `PolicyFindingCodesAreDiscoverable`, G-0021).

## Out of scope — decided 2026-06-21, superseding E-0016

These were E-0016's design (carried over from G-0055's deferred layers) and are deliberately dropped:

- **`aiwf.yaml: tdd.default` project-default fallback** and the resolver change that would let an omitted `--tdd` resolve to it. The hard refusal stays. Relaxing a working chokepoint for unproven ergonomics is YAGNI, and a project default of `required` relocates the silent-wrong-policy risk (a docs milestone silently gets `required` unless the operator remembers `--tdd none`) rather than removing it. Retires E-0016 M-0062 / M-0063 / M-0064.
- **`aiwf init` seeding / `aiwf update` migration** of `tdd.default` — moot without the field.
- **Promote-time guard** (G-0055 layer 3, `draft → in_progress` refused when `tdd:` absent) — unchanged; remains deferred, now largely redundant with this check rule.
