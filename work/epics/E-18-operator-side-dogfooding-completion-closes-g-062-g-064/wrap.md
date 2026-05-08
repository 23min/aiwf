# Epic wrap — E-18

**Date:** 2026-05-08
**Closed by:** human/peter
**Integration target:** poc/aiwf-v3 (the kernel branch — the PoC does not merge to main; see [CLAUDE.md](../../../CLAUDE.md) "Working on the PoC")
**Epic branch:** *(none — milestones branched directly off `poc/aiwf-v3` and merged back per the kernel-branch convention)*
**Merge commits on `poc/aiwf-v3`:** `71eeacd` (M-070), `8706d7b` (M-071)

## Milestones delivered

- M-070 — aiwf doctor warning for missing recommended plugins (merged `71eeacd`)
- M-071 — Install ritual plugins in kernel repo + document operator setup path (merged `8706d7b`)

## Summary

E-18 closes the operator-side gap that G-038 left partial: the kernel repo's design assumed `aiwf-extensions` and `wf-rituals` were present, but neither plugin was installed for this project's scope, and there was no kernel mechanism to detect that. M-070 added the detection — a config-driven `doctor.recommended_plugins` check that warns once per declared-but-missing plugin against `~/.claude/plugins/installed_plugins.json`. M-071 adopted the mechanism here: declared both plugins in `aiwf.yaml`, project-scope-installed them, and documented the operator setup path in CLAUDE.md (including the project-scope-vs-user-scope nuance discovered live during the install). The two milestones validated each other end-to-end: M-070's check fired exactly when M-071's pre-install state predicted, and went silent exactly when the install completed.

## ADRs ratified

- *(none)* — no architectural decision emerged that wasn't already articulated in the epic spec itself. The kernel-vs-Claude-Code separation ("the kernel signals; the operator installs") was set in the epic's *Out of scope* section before any work started; lifting it into an ADR would not add explanatory weight beyond what's already in [`epic.md`](epic.md).

## Decisions captured

- *(none as D-NNN entities)* — the implementation-time decisions (deferred JSON envelope, deleted only the doctor-side `ritualsPluginInstalled` block, helper takes `cfg` rather than re-loading, CLAUDE.md documents the interactive `/plugin` menu rather than the CLI shorthand) all live in the milestone specs' `## Decisions made during implementation` sections. None reach beyond their own milestone's scope.

## Follow-ups carried forward

Both deferrals are filed as gap entities so they survive the epic's closure on `aiwf status`. The narrative context (why each was deferred, what the fix shape looks like) lives in the milestone wrap-side specs; the gaps are the kernel's tracked surface.

- **G-069** — `aiwf init`'s `printRitualsSuggestion` hardcodes the CLI install form, which defaults to *user* scope and short-circuits with "already installed globally." After E-18, that nudge silently steers fresh operators away from the project-scope outcome the framework actually wants. Migrating the nudge to either reference the interactive `/plugin` menu or read `aiwf.yaml: doctor.recommended_plugins` is a separate decision worth its own ticket. Discovered during M-071/AC-2; first noted in [M-070's *Deferrals* section](M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md#deferrals).

- **G-070** — `aiwf doctor` has no `--format=json` envelope; M-070's AC-3 spec text references a `finding.code` / `finding.data` envelope structure that the doctor verb has no surface for today. Forward-looking; the fix waits on a JSON-consuming caller appearing. Recorded under [M-070's *Deferrals* section](M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md#deferrals).

## Gaps closed

- G-062 — `aiwf doctor` does not surface missing recommended plugins → addressed by M-070.
- G-064 — Kernel repo dogfooding closed partial (G-038) without installing the ritual plugins → addressed by M-071.

## Doc findings

`wf-doc-lint` was run scoped to each milestone's change-set at wrap time:
- M-070: 0 findings (1 narrative doc touched: `docs/pocv3/design/design-decisions.md`).
- M-071: 0 findings (1 narrative doc touched: `CLAUDE.md`).

No drift, no broken references, no orphan files, no documentation TODOs introduced by E-18.

## Handoff

What is ready for the next epic:
- The kernel now has a generic mechanism (`doctor.recommended_plugins`) that any consumer can use to declare expected plugins and get a warning when state doesn't match. Other consumers can adopt the same pattern; the docs in [`design-decisions.md`](../../../docs/pocv3/design/design-decisions.md) describe the field for them.
- This repo's operator-setup story is reproducible: a fresh clone + the install procedure in CLAUDE.md's "Operator setup" section produces a working Claude Code workspace with the rituals available. The dogfood loop closes.

What is deliberately left open:
- `aiwf init`'s plugin-install nudge (see *Follow-ups carried forward*). Not blocking; needs its own scoping decision.
- A `--format=json` surface for `aiwf doctor` (also in *Follow-ups*). Wait for a forcing function.
- Auto-install of recommended plugins from inside `aiwf` (e.g. `aiwf doctor --install-recommended`). Explicitly out of scope per E-18; a separate decision before any future epic considers it.
