---
id: D-0098
title: Finish interrupted guidance installation on the next update
status: proposed
relates_to:
    - M-0346
    - E-0094
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Question

How should project guidance recover when an installation stops after replacing some files but before completing the installed set?

## Decision

Finish the installation on the next `aiwf update`, using the current selection and upstream content. Retain recovery receipts and keep incomplete installation visible until recovery succeeds.

## Reasoning

Automatic rollback requires a separate lifecycle for backups and restoration. Finishing on the next explicit update keeps recovery within the existing maintenance operation and avoids that complexity. Recovery receipts distinguish an interrupted generated file from unrelated or edited content, so retrying does not grant permission to overwrite arbitrary files.

## Consequences

Do not claim atomic replacement of the whole guidance set. Host routing must warn about a pending installation and avoid using the incomplete pack set. Retry must retain enough ownership evidence to recover across repeated interruptions, including when upstream content or the selected packs change between attempts. M-0346 implements this behavior for E-0094.
