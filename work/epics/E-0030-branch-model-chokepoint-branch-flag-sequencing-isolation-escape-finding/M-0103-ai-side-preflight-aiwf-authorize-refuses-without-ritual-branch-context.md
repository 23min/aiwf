---
id: M-0103
title: 'AI-side preflight: aiwf authorize refuses without ritual branch context'
status: draft
parent: E-0030
depends_on:
    - M-0102
tdd: required
---

## Goal

Make `aiwf authorize <id> --to ai/<agent>` refuse the dispatch when no ritual branch context is in play — either `--branch <name>` is passed naming an existing ritual-shape branch, or the current checkout is already on a recognized ritual-shape branch. Refusal produces an actionable error pointing at the ritual surface to use.

## Context

M-0102 added the `--branch` flag and the trailer; this milestone wires the chokepoint behavior that makes [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s AI-isolation rule enforceable at the verb level. Together with M-0106's post-hoc kernel finding, this is defense in depth — the preflight blocks the bad dispatch at the source; the kernel finding catches drift that slips through.

Human-actor `aiwf authorize` invocations are unaffected — the preflight only fires when `--to ai/<id>` is in play. Author sovereignty is preserved per ADR-0010.

## Out of scope

- Rituals reorder (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Branch *creation* — the preflight only checks existence; cutting the branch is the ritual's job.
- Any changes to the trailer key or flag itself (already shipped in M-0102).

## Dependencies

- **M-0102** — provides the `--branch` flag and trailer key this milestone enforces.

## Open questions for AC drafting

- **Branch-context detection:** Pattern-match `git branch --show-current` against ritual shapes (`epic/E-*`, `milestone/M-*`, `fix/*`, `patch/*`, `doc/*`, `chore/*`)? Require explicit `--branch`? Or both — accept either signal, fail if neither? Likely "either"; documented in the milestone spec.
- **Error message shape:** What's the actionable hint? E.g., *"This invocation opens an autonomous scope on ai/claude but is not on a ritual branch. Run `aiwfx-start-epic` or `aiwfx-start-milestone` first, or pass `--branch <name>` naming an existing branch."*
- **Sovereign override path:** Should `--force --reason` bypass the preflight? Likely yes (consistent with kernel pattern), but explicit.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0103` time. -->
