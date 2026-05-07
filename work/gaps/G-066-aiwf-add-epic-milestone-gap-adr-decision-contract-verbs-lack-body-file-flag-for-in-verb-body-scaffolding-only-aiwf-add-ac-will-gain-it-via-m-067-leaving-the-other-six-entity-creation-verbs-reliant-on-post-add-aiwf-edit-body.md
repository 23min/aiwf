---
id: G-066
title: aiwf add epic/milestone/gap/adr/decision/contract verbs lack --body-file flag for in-verb body scaffolding; only aiwf add ac will gain it via M-067, leaving the other six entity-creation verbs reliant on post-add aiwf edit-body
status: open
---

## What's missing

E-17's M-067 adds `aiwf add ac --body-file <path>` so AC bodies can be authored in the same atomic commit as the AC creation. The other six entity-creation verbs do not get the same treatment:

- `aiwf add epic --title "..."`
- `aiwf add milestone --title "..." --epic E-NN`
- `aiwf add gap --title "..."`
- `aiwf add adr --title "..."`
- `aiwf add decision --title "..."`
- `aiwf add contract --title "..."`

After E-17 ships, an operator authoring an epic still has to:

1. Run `aiwf add epic --title "..."` (commit 1: empty-body entity).
2. Edit the file's body in the working tree (uncommitted hand-edit).
3. Run `aiwf edit-body E-NN` (commit 2: body content).

Two commits and an interim "empty body" state where E-NN's body section fails E-17's `entity-body-empty` rule. The same shape applies to milestone, gap, adr, decision, contract.

In contrast, AC authoring with `--body-file` is a single atomic commit and never produces an interim empty-body state.

### Why E-17 didn't generalize the flag

The rescope decision (per G-063 sub-decision #4) generalized the *check rule* (`acs-body-empty` → `entity-body-empty`) to all kinds, but kept the *flag work* AC-only. Reasoning recorded in E-17's body:

> *"AC has the highest authoring volume; the asymmetry is acceptable in the short term. The check rule fires for those kinds; operators currently rely on `aiwf edit-body` to fill the body post-add."*

That reasoning closes the door on M-067's scope but explicitly leaves a follow-up surface — this gap.

### Suggested shape

A new milestone or epic that adds `--body-file <path>` (and potentially `--body-file -` for stdin, mirroring M-067's design) to each of the six remaining `aiwf add` verbs. Open design questions:

- **Single milestone or six?** The flag pattern is identical across kinds; one milestone implementing all six is plausibly the right unit. If kind-specific quirks emerge during M-067 implementation, split.
- **Section-prefilled scaffolds?** When `--body-file` is omitted today, `aiwf add` scaffolds the body with empty section headings (e.g., `## What's missing`, `## Why it matters` for gaps). When `--body-file` is provided, should the file's content replace the entire body, or be inserted under existing scaffolded headings? Lean: full replacement (operator owns the body), but document this clearly.
- **Skill update.** `aiwf-add` would gain documentation of the new flag for each kind, mirroring M-068's update pattern.

### Suggested ordering

This gap depends on M-067 landing first — the AC implementation establishes the flag's behavior contract (path validation, frontmatter rejection in body files, single-vs-multi semantics). Generalizing six verbs to follow that contract is mechanical once the contract is pinned.

## Why it matters

- **Asymmetric authoring ergonomics.** Operators creating ACs get a one-step authoring flow; operators creating any other kind get two steps and an interim empty-body commit. The friction is highest exactly where authoring is highest-leverage (epics, milestones — the units that *contain* ACs and define their context).
- **Interim empty-body commits trip E-17's check rule.** During the two-step flow (add → edit-body), `aiwf check` will report `entity-body-empty` between commit 1 and commit 2. CI runs that catch the working tree mid-flow will fail or warn. This is recoverable but noisy; `--body-file` makes it a non-issue.
- **AI assistants will hit this constantly.** When an AI assistant in conversation creates an epic, the natural pattern is "create + write the body" as a single intent. Forcing two commits (and the editor-roundtrip in between) is awkward and slows down planning sessions.

### Predecessor / sibling references

- Triggering instance: M-066 / E-17 rescope (2026-05-07), recorded in those entities' "Rescope note (per G-063)" sections.
- Sibling: G-058 (open) — AC body chokepoint; closed by E-17 (now generalized).
- Sibling: G-063 (open) — start-epic ritual; sub-decision #4 forced E-17's rescope and surfaced this gap.
- Predecessor design: M-067 (E-17, draft) — establishes the `--body-file` contract for ACs. This gap's fix follows that contract.

