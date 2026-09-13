---
id: D-0089
title: aiwf ships no language-specific guidance
status: proposed
relates_to:
    - E-0092
---
> **Date:** 2026-09-13 · **Decided by:** Peter Bruinsma (human), while planning E-0092

## Question

Where do language conventions (a formatter and lint set, test idioms, error wrapping) live for a downstream project that uses aiwf: in aiwf's shipped surfaces, or elsewhere? Non-obvious because the shipped guidance fragment is the broadest-reach always-on surface a consumer has, and this repo's own `CLAUDE.md` restates every topic of the operator's Go module, which reads as if aiwf owned them.

## Decision

aiwf ships no language-specific guidance. Conventions that are not aiwf-specific come from the operator's own tooling; here, the dotfiles repo whose modules the root `CLAUDE.md` imports. This repo's `CLAUDE.md` carries only what is specific to developing aiwf, and the shipped fragment carries only how to operate aiwf.

## Reasoning

- A language convention is a per-human preference about how code is written, not a property of the planning tree aiwf validates. Shipping it makes every consumer inherit one person's preferences.
- aiwf is language-agnostic and public. Its always-on fragment reaches every consumer, Go or not, so a Go section there is dead weight for the rest and grows the load that E-0092's ceiling exists to cap.
- A shipped on-demand skill per language loads only when the model judges it relevant, and would stand as a second source beside the module the operator already imports.
- A materialized per-stack fragment gated by `aiwf.yaml` needs a materializer, a config key, and doctor and policy wiring, for a need no consumer has raised.
- Both lost to a delivery that already exists and is already imported.

## Consequences

- The generic Go conventions section leaves `CLAUDE.md` under E-0092; the imported module remains the one source.
- A third party using aiwf brings their own language conventions.
- A request to add language content to the shipped fragment, for any language, is declined by this record.
