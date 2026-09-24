---
id: E-0095
title: Adopt applicable guidance automatically on upgrade and update
status: proposed
---
## Goal

An `aiwf upgrade` or `aiwf update` leaves a repository with the engineering guidance that applies to it installed, routed into its selected hosts, and reported, with no prompt to answer and nothing to discover afterwards. Packs for languages the repository gains later arrive on the next update the same way.

## Context

E-0094 delivered external guidance packs as tracked `.guidance/` files, adopted only through explicit selection: an interactive prompt on a terminal, or a stderr suggestion otherwise. `aiwf upgrade` re-executes `update`, so upgrades run from scripts, pipes or agent sessions adopt nothing, and the only durable sign is an `aiwf doctor` line. A consumer upgrade ended exactly that way.

ADR-0054 replaces explicit selection with add-only automatic adoption, removes the guidance prompt, and requires an explicit guidance report on every run. Detection, retrieval, validation, installation, host routing and legacy handover already exist in `internal/projectguidance` and `internal/initrepo`; this epic changes who selects and what the operator is told.

## Scope

- Adoption: every enabled `init` / `update` run adds each applicable, unignored catalogue pack to `guidance.packs` and installs it with host routing, on a terminal or not. Existing selections and ignores are honoured; nothing is removed automatically.
- Removal of the select / not now / ignore guidance prompt and of the flag help and command help that describe it.
- A guidance report closing every `init`, `update` and `upgrade` run: packs adopted this run, packs already installed and the installed source commit, ignored packs, and any blocker (fetch failure, incompatible legacy delivery, handwritten legacy import, locally edited owned file) with the action that clears it. The report distinguishes an unset selection, an empty one and disabled maintenance, in the vocabulary `aiwf doctor` uses.
- Help text, embedded skills and the host setup guide describing adoption, opt-out through `guidance.ignored`, and the report.
- An end-to-end check that `aiwf upgrade` without a terminal installs applicable guidance in a consumer repository that had none.

## Out of scope

- A `--no-prompt` flag for `aiwf update`'s hook-consent prompt. Removing the guidance prompt leaves hook consent as the only prompt `update` can raise; that remaining gap is tracked separately.
- Automatic removal of packs whose detection no longer matches. Removal stays a recorded `guidance.ignored` choice.
- Changes to the external catalogue's packs or detection patterns, which the catalogue repository owns.
- Changes to ai-dotfiles synchronization or its session hook; ADR-0052 assigns those to the personal-bootstrap maintainer.

## Constraints

- `update` and `init` never commit or push; adoption is a working-tree change the maintainer reviews.
- A guidance failure never fails unrelated update work, and leaves installed guidance and its recorded revision unchanged, as E-0094 established.
- `guidance.enabled: false` suppresses detection, adoption and refresh.
- `aiwf.yaml` edits preserve unrelated fields and comments.
- Every report line and help string is reachable through `--help` or an embedded skill.
- Leave the shared guidance fragment, the installed file layout, the owned set and the routing block format unchanged; E-0092 rewrites the fragment and fences commits around that layout.

## Success criteria

- [ ] Upgrading a consumer repository with no `guidance` configuration, without a terminal, leaves it with `.guidance/`, an updated `guidance.packs`, host routing, and a report naming what was adopted.
- [ ] Adding files that match a further pack, then running `aiwf update`, installs that pack and reports it.
- [ ] A pack listed in `guidance.ignored` is never adopted, and moving an adopted pack there removes its unmodified output on the next update.
- [ ] When guidance cannot be installed, the update report states why and what clears it, and `aiwf doctor` agrees.
- [ ] No `init` or `update` code path prompts for guidance.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| When handover or retrieval blocks installation, is the adoption still recorded in `guidance.packs` (desired but not installed) or deferred until installation succeeds? | no | Settled in M-0351: adoption is recorded only with a successful installation. |
| Does `upgrade` print the report itself, or rely on the re-executed `update` printing it? | no | Settled in M-0352: the re-executed `update` prints it; M-0352/AC-5 pins that an upgrade's output contains it. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Consumers are surprised by `.guidance/`, `aiwf.yaml` and CLAUDE.md changes after an upgrade | med | Changes stay uncommitted and the report names each one with the opt-out |
| A stray matching file adopts an unwanted pack | low | One `guidance.ignored` entry, reported in the output |
| Repositories on legacy delivery hit a handover blocker on every update | med | The report names the blocker and the fix; unrelated update work proceeds |

## Milestones

- `M-0351` — Adopt applicable guidance packs automatically in init and update · depends on: —
- `M-0352` — Report guidance outcomes at the end of init, update and upgrade · depends on: `M-0351`
- `M-0353` — Verify an unattended upgrade installs applicable guidance · depends on: `M-0351`, `M-0352`

## References

- ADR-0054 — automatic add-only adoption, no prompt, explicit report.
- D-0089 — superseded by that ADR.
- ADR-0052 — ownership and legacy handover, unchanged.
- E-0094 — the delivery this epic builds on.
