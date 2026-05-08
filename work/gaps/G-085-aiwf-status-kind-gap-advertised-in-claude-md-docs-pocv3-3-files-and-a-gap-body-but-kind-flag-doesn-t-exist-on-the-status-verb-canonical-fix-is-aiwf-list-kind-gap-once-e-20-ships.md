---
id: G-085
title: '`aiwf status --kind gap` advertised in CLAUDE.md, docs/pocv3 (3 files), and a gap body, but `--kind` flag doesn''t exist on the status verb; canonical fix is `aiwf list --kind gap` once E-20 ships'
status: open
discovered_in: E-20
---
## Problem

CLAUDE.md and three `docs/pocv3/` files instruct readers to run `aiwf status --kind gap` to inspect open gaps. The `aiwf status` verb does not accept `--kind`; running the advertised command yields:

    aiwf: unknown flag: --kind

The nudge dates from the G-038 dogfood-migration cutover (2026-05-05) that moved gaps from a single markdown file into `aiwf` entities under `work/gaps/`. At that point a generic filter verb did not exist; `aiwf status --kind gap` was forecast prose, not shipped surface. G-061 captures the same shape from the contracts side — a non-existent `aiwf list contracts` verb referenced as canonical in `contracts-plan.md` and the `aiwf-contract` skill.

This violates the kernel principle *"kernel functionality must be AI-discoverable"*: a contributor (human or AI) reading the published doc surface lands on an unknown-flag error. The completion-drift test (`cmd/aiwf/completion_drift_test.go`) doesn't catch this — the fault is in prose, not in source.

## Sites

- `CLAUDE.md:3` — first-paragraph onboarding nudge.
- `docs/pocv3/README.md:11` — engine-contributor onboarding.
- `docs/pocv3/architecture.md:184` — architecture write-up.
- `docs/pocv3/archive/gaps-pre-migration.md:3` — migration-archive header (frozen content but the instruction is current).
- `work/gaps/G-078-no-priority-field-on-entities-backlog-isn-t-filterable-or-sortable-by-importance.md:9` — gap body cites `aiwf status --kind gap` and gestures at "any future `aiwf list` verb (G-061)" as a planned replacement.

## Fix shape

E-20 (in flight) ships `aiwf list --kind <kind>` as the canonical filter verb. Once M-072 lands the verb, all five sites become a mechanical search-and-replace:

    aiwf status --kind gap   →   aiwf list --kind gap

Drop-in; no behavior change beyond the verb name. Two natural closure paths:

1. Fold into M-074's CLAUDE.md edit pass (it already touches CLAUDE.md for the *Skills policy* section) and extend its scope to the four other files.
2. File as a follow-up doc-sweep after E-20 closes, parallel to the existing `aiwf list contracts` drift fix wired into M-072's AC-8.

## References

- G-061 — parent shape (same drift class, contracts surface).
- E-20 — ships the verb that makes the fix possible.
