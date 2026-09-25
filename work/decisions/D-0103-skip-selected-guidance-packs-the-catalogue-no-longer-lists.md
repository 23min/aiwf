---
id: D-0103
title: Skip selected guidance packs the catalogue no longer lists
status: proposed
relates_to:
    - M-0351
    - E-0095
    - ADR-0054
---
> **Date:** 2026-09-25 · **Decided by:** Peter Bruinsma

## Question

What should `aiwf init` and `aiwf update` do when `guidance.packs` names a pack the guidance catalogue no longer lists? Under automatic, add-only adoption aiwf writes those ids itself and never removes them, so a catalogue rename or removal would otherwise fail every later run's retrieval, installing and adopting nothing, over an id no one typed.

## Decision

- Skip the missing id at retrieval and still retrieve, install and adopt the rest of the selection.
- Keep the id in `guidance.packs`; do not remove it automatically.
- Retire the missing pack's unmodified installed files, as for any pack no longer installed; locally edited files still block retirement.
- Report the id on the guidance step with the fix: remove it from `guidance.packs` or move it to `guidance.ignored`.

## Reasoning

Failing the whole retrieval turns a routine catalogue change into a permanent block on all guidance, including the pack a rename just introduced. Dropping the id automatically would bend add-only adoption and silently discard a selection a maintainer may have made by hand. Keeping the vanished pack's files installed but frozen would need a new "owned but not in the catalogue" state in the installer and an index that no longer lists everything routed, while leaving stale guidance in place. Retiring through the existing path needs no new installer state, handles a rename in one run (old files retired, new id adopted when it applies), and the report keeps the loss visible.

## Consequences

The installed set is the selection minus the ids the catalogue lacks. A repeated id in the selection is still refused, as configuration validation already does. If the catalogue lists the pack again, the retained id installs it again on the next run.
