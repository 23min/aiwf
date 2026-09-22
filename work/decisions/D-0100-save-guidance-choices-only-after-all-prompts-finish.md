---
id: D-0100
title: Save guidance choices only after all prompts finish
status: proposed
relates_to:
    - M-0347
    - E-0094
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Question

Should answers to guidance suggestions survive interruption partway through an interactive selection session?

## Decision

Save choices once, after all guidance prompts finish. Enter means not now. Interruption discards every answer from that prompt session and leaves installed guidance unchanged.

## Reasoning

Saving each answer immediately can leave a partly completed selection in configuration before installation. A single configuration write keeps persistence simple and gives interruption a clear meaning, at the cost of repeating earlier answers on the next run.

## Consequences

A completed session records desired policy before validation and installation. If installation fails, retain the desired selection, preserve existing installed guidance, and report the failure so a later update can retry. This decision concerns guidance choices; it does not roll back unrelated init or update work. M-0347 implements the interaction.
