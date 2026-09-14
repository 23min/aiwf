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

The file grew inside ordinary work. G-0676 names the four surfaces that let it: an edit to `CLAUDE.md` rides in any commit that also touches code, a doc-shaped AC may be evidenced by a sentence pinned in the file, the discoverability policies count the file as a channel, and the principle text names it as one. If those stay open, every cut below regrows at the rate that gap records, so the fence lands before the first cut.

## Scope

Staged, in this order. Each stage is a milestone; the ceiling steps down after each stage that removes text.

- **The fence.** A diff-scoped policy that fails a commit modifying `CLAUDE.md` alongside any other path, or carrying no `aiwf-entity` trailer that resolves; the ceiling policy, landed at the current size; `CLAUDE.md` removed from the discoverability channel list; the discoverability principle no longer naming the file. Closes G-0676.
- **The baseline.** The rubric, then the baseline observation of the judgment rules over a fixed task set.
- **Re-home by directory.** A thin root `CLAUDE.md`; nested `CLAUDE.md` files under `internal/`, `cmd/`, `docs/`, and `work/`, each loaded only when a session reads files there; every test that pins a moved passage re-aimed at the passage's new file; the `@`-imported Go module moved to `internal/CLAUDE.md`; the Python and TypeScript imports deleted.
- **Delete copies.** Text that already loads from another home: the rules the fragment carries, the generic Go conventions section (its source is the imported `200-go.md`), and the entity-id asides, whose reasoning moves to the entity that owns it where it is not already there.
- **Pointer cut.** An audit of every finding and policy message a `CLAUDE.md` section documents, messages that do not state the fix corrected, and the chokepointed sections compressed to one-line pointers. Closes G-0436. The ceiling reaches its target here.
- **The after observation**, against the same rubric.
- **The fragment.** The shipped fragment rewritten as one imperative plus one line of why per rule, `CLAUDE.md` and the fragment holding each anchor in exactly one place, the operating-anchors policy updated in the same commit. This stage runs only if the after observation shows no lost effect; otherwise it is cancelled and the epic wraps without it.

## Out of scope

- Entity templates (G-0530 owns the milestone template).
- Ritual and verb `SKILL.md` bodies; they load on demand.
- The operator's dotfiles repo, including whether it ships to other people.
- Any change to what a check enforces; only messages change.
- Guidance delivery failing unobserved (G-0523).
- A fragment-only ceiling; added only if the fragment drifts after the union cap lands.
- Any ceiling on a consumer's `CLAUDE.md`. The policy runs only in this repository's suite and never ships.

## Constraints

- The fence lands before the first cut, and nothing shipped changes before the after observation is judged.
- Every `CLAUDE.md` edit under this epic is its own commit carrying an entity trailer, the rule the fence enforces.
- The ceiling is one number on the union, not one per file: 3,500 words at target, whitespace-split, over root `CLAUDE.md` and every file its `@` imports resolve to. It lands at the current size and only ever steps down.
- No rule is dropped. A rule is relocated or compressed; a judgment rule keeps one line of why.
- A test that pins a moved passage is re-aimed at the passage's new home, or retired with the reason recorded in the milestone; none is deleted silently.
- No AC is evidenced by a sentence pinned in a `CLAUDE.md`, root or nested (D-0091); the diff-scoped scan that enforces it lands at the fence milestone.
- A chokepointed section becomes a pointer only after its finding message states the fix.
- Language conventions that are not aiwf-specific live in the operator's dotfiles; aiwf ships none (D-0089).
- The observation rubric is written before the baseline run and not changed after.
- Every measured figure recorded under this epic carries the command that produced it (G-0668).
- `CLAUDE.md` follows its own rule: the conclusion, not the drafting history.

## Success criteria

- [ ] A commit that modifies `CLAUDE.md` alongside another file, or without a resolving entity trailer, fails the profile-driven gate.
- [ ] The always-on set is at or under the ceiling and a policy test fails when it regrows.
- [ ] Root `CLAUDE.md` `@`-imports only the shipped fragment.
- [ ] Every test that pinned a root passage pins it at its new home or is retired with a recorded reason.
- [ ] `CLAUDE.md` carries no generic language convention and no rule the fragment also carries.
- [ ] Every chokepoint pointer in `CLAUDE.md` resolves to an existing policy id or finding code.
- [ ] The before and after observations are recorded, each with command, expectation, observation, and environment.
- [ ] G-0676 and G-0436 are closed.
- [ ] If the fragment stage runs: each anchor the operating-anchors policy pins appears in exactly one of `CLAUDE.md` and the fragment, and no fragment rule exceeds the per-rule word cap its milestone sets.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which finding messages already state the fix | no | the pointer-cut milestone produces the table |
| Whether `aiwf update` leaves a nested `CLAUDE.md` untouched | no | verified at the re-home milestone; it maintains only the root import marker today |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A cross-cutting rule re-homed under one directory is invisible to sessions that never read there | med | the after observation; the rule moves back to root |
| A gotcha that is not a rule leaves the always-on set and is lost | med | each one lands in the skill for its task before its section is cut |
| Additions queued in G-0235 and G-0370 regrow the set | med | the fence and the ceiling are the rules they fit or displace |
| The anchors policy pins phrases the fragment rewrite changes | low | policy and fragment change in one commit |

## Milestones

- `M-NNNN` — The fence: commit-seam gate, ceiling at current size, discoverability channel and principle text · depends on: —
- `M-NNNN` — The rubric and the baseline observation · depends on: —
- `M-NNNN` — Re-home by directory; re-aim the pins; move the Go import; delete the dead imports · depends on: the first two
- `M-NNNN` — Delete copies: fragment duplicates, the Go section, the id asides · depends on: the third
- `M-NNNN` — Audit finding messages; cut chokepointed sections to pointers; ceiling to target · depends on: the fourth
- `M-NNNN` — The after observation · depends on: the fifth
- `M-NNNN` — Rewrite the fragment; update the anchors policy · depends on: the sixth, and on its result

## References

- G-0676 — how `CLAUDE.md` grows inside ordinary work; the fence closes it
- ADR-0018 — the per-turn guidance fragment and its `CLAUDE.md` import
- D-0070 — what a test may pin in shipped prose; bounds the fragment's own tests
- D-0091 — no AC is evidenced by a sentence pinned in `CLAUDE.md`; enforcement diff-scoped, existing pins drained by the shrink
- D-0089 — aiwf ships no language-specific guidance; language conventions come from the operator's dotfiles
- `internal/policies/skill_edit_provenance_backstop.go` — the shape the commit-seam gate takes
- `internal/policies/m0211_guidance_operating_anchors.go` — the anchors the fragment rewrite must keep
- G-0436 — stale paths in `CLAUDE.md`, closed by the pointer cut
- G-0235, G-0370 — queued additions to `CLAUDE.md` and the fragment
- G-0668 — measured figures carry their command
