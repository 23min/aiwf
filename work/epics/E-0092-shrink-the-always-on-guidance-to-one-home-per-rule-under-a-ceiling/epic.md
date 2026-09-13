---
id: E-0092
title: Shrink the always-on guidance to one home per rule under a ceiling
status: proposed
---

## Goal

Cut the instruction load a session reads before its task to a fixed ceiling, enforced by a policy test, with every rule kept in exactly one home and no operating rule lost.

## Context

The always-on set is root `CLAUDE.md` plus every file it `@`-imports: the shipped guidance fragment (ADR-0018, E-0040) and three language modules from the operator's dotfiles. Measured 2026-09-13 in the devcontainer, at the tree of this epic's allocation commit:

```
wc -w CLAUDE.md .claude/aiwf-guidance.md \
  ~/.agents/guidance/200-go.md ~/.agents/guidance/201-python.md ~/.agents/guidance/203-typescript.md
```

| file | words |
|---|---|
| `CLAUDE.md` | 9,234 |
| `.claude/aiwf-guidance.md` | 2,238 |
| `200-go.md` | 285 |
| `201-python.md` + `203-typescript.md` | 322 |

Length is not where the effect is. A rule with a chokepoint takes its effect from the check; the prose only spares one failed-check round trip, which is worth a pointer once the finding message states the fix. Most of `CLAUDE.md` documents such rules. Its two largest sections are design reasoning needed only when designing a verb or a stress scenario, and the ADRs and design docs already hold most of it. The operating rules the fragment ships are restated in `CLAUDE.md`, against the repo's own rule that consumer-operating guidance lives in the fragment and is not forked. Every topic in `200-go.md` is restated in the Go conventions section. The Python and TypeScript modules load into a Go repo. The premise the observation milestone tests is that per-rule compliance falls as the rule count rises, so the rules that matter compete with thousands of words that do not, and that the register of the load is the register every output imitates.

A subdirectory `CLAUDE.md` is loaded only when a session reads files under it, and `@` imports resolve relative to the importing file (Claude Code memory documentation). That gives repo-development conventions a home outside the always-on set with no new mechanism.

## Scope

- A policy test that measures the always-on set against a ceiling and fails when it is exceeded. It lands first at the current size, so nothing regrows during the epic, and is lowered to the target last.
- An audit of every finding and policy message that a `CLAUDE.md` section documents; messages that do not state the fix are corrected.
- Chokepointed sections of `CLAUDE.md` compressed to one-line pointers.
- Design reasoning relocated to ADR-0036, ADR-0038, `docs/design/design-decisions.md`, and `docs/design/oracles.md`; aiwf-development Go specifics relocated to `internal/CLAUDE.md`, with a one-line `cmd/CLAUDE.md` that imports it.
- Deletions: the generic Go conventions section (its source remains the `@`-imported `200-go.md`), the Python and TypeScript imports, and the entity-id asides in `CLAUDE.md`, whose reasoning moves to the entity that owns it where it is not already there.
- `CLAUDE.md` stripped of every rule the fragment carries; the fragment rewritten as one imperative plus one line of why per rule; the operating-anchors policy updated in the same commit.
- A recorded before/after observation of the judgment rules over a fixed task set, with the rubric written before the baseline run.

## Out of scope

- Entity templates (G-0530 owns the milestone template).
- Ritual and verb `SKILL.md` bodies; they load on demand.
- The operator's dotfiles repo, including whether it ships to other people.
- Any change to what a check enforces; only messages change.
- Guidance delivery failing unobserved (G-0523).
- A fragment-only ceiling; added only if the fragment drifts after the union cap lands.

## Constraints

- The ceiling is one number on the union, not one per file: 3,500 words, whitespace-split, over root `CLAUDE.md` and every file its `@` imports resolve to.
- No rule is dropped. A rule is relocated or compressed; a judgment rule keeps one line of why.
- A chokepointed section becomes a pointer only after its finding message states the fix.
- Language conventions that are not aiwf-specific live in the operator's dotfiles; aiwf ships none.
- The operating-anchors policy stays green through the fragment rewrite.
- The observation rubric is written before the baseline run and not changed after.
- Every measured figure recorded under this epic carries the command that produced it (G-0668).
- `CLAUDE.md` follows its own rule: the conclusion, not the drafting history.

## Success criteria

- [ ] The always-on set is at or under the ceiling and a policy test fails when it regrows.
- [ ] Each anchor the operating-anchors policy pins appears in exactly one of `CLAUDE.md` and the fragment.
- [ ] Every chokepoint pointer in `CLAUDE.md` resolves to an existing policy id or finding code.
- [ ] `CLAUDE.md` carries no generic language convention and no rule the fragment also carries.
- [ ] The before and after observations are recorded, each with command, expectation, observation, and environment.
- [ ] G-0436 is closed by the pointer cut.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which finding messages already state the fix | no | the message-audit milestone produces the table |
| Whether `aiwf update` leaves a nested `internal/CLAUDE.md` untouched | no | verified at the relocation milestone; it maintains only the root import marker today |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A gotcha that is not a rule leaves the always-on set and is lost | med | each one lands in the skill for its task before its section is cut |
| Additions queued in G-0235 and G-0370 regrow the set | med | the ceiling is the rule they fit or displace |
| The anchors policy pins phrases the rewrite changes | low | policy and fragment change in one commit |

## Milestones

- `M-NNNN` — Land the ceiling policy at the current size; write the rubric; record the baseline observation · depends on: —
- `M-NNNN` — Audit finding messages, fix the ones that do not self-explain, cut chokepointed sections to pointers · depends on: the first
- `M-NNNN` — Relocate design reasoning and aiwf-development conventions; delete the language sections and imports · depends on: the second
- `M-NNNN` — Strip `CLAUDE.md` of fragment rules and id asides; rewrite the fragment; update the anchors policy · depends on: the third
- `M-NNNN` — Lower the ceiling to the target; record the after observation · depends on: the fourth

## References

- ADR-0018 — the per-turn guidance fragment and its `CLAUDE.md` import
- D-0070 — what a test may pin in shipped prose; bounds the fragment's own tests
- D-0089 — aiwf ships no language-specific guidance; language conventions come from the operator's dotfiles
- `internal/policies/m0211_guidance_operating_anchors.go` — the anchors the fragment rewrite must keep
- G-0436 — stale paths in `CLAUDE.md`, closed by the pointer cut
- G-0235, G-0370 — queued additions to `CLAUDE.md` and the fragment
- G-0668 — measured figures carry their command
