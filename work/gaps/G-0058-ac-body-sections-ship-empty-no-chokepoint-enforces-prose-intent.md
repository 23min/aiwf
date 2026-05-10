---
id: G-0058
title: AC body sections ship empty; no chokepoint enforces prose intent
status: open
discovered_in: E-0016
---

## What's missing

The design specifies that each AC's body section carries prose detail — description, examples, edge cases, references — anchored under a `### AC-N — <title>` heading:

- [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](../../docs/pocv3/plans/acs-and-tdd-plan.md): *"`### AC-N — <title>` heading per AC, with prose detail (description, examples, edge cases, references)."*
- [`docs/pocv3/design/design-decisions.md:139`](../../docs/pocv3/design/design-decisions.md): *"The body carries a matching `### AC-N — <title>` heading per AC for prose."*
- [`docs/pocv3/plans/acs-and-tdd-plan.md:267`](../../docs/pocv3/plans/acs-and-tdd-plan.md) (about `aiwfx-start-milestone`): *"scaffold AC body sections in the milestone doc."*

The kernel rule that exists — `acs-body-coherence` — only checks that each frontmatter AC has a *matching heading* in the body and vice versa (`acs-and-tdd-plan.md:197`). It does **not** check that the section under the heading is non-empty. `aiwf add ac` scaffolds the empty heading and never prompts the operator (or LLM) to fill it. The `aiwf-add` skill says nothing about a follow-up body-prose pass. As a result, the entire historical tree (M-0049 through M-0061, every milestone in this repo) ships with bare AC headings — title is the entire spec.

## Why it matters

This is structurally the same gap as [G-0055](G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md): design intent says X, no chokepoint enforces X, the skill produces the wrong shape, an LLM following the skill faithfully reproduces the defect. The kernel principle violated is "framework correctness must not depend on the LLM's behavior."

Concrete consequences when AC bodies are empty:

- **The title becomes the entire spec.** A 60-character label has to carry "what passing looks like, edge cases, references to the relevant code path." It can't. Reviewers, future maintainers, and AI agents picking up the work later have to *infer* the AC's intent from the title alone, often re-deriving decisions the original author already made and didn't write down.
- **TDD becomes shallow.** Under `tdd: required`, an AC walks `red → green → refactor → done`. Without a body that names the test shape and the failure modes, "red" is a vibe, not a spec. The phase trailers record motion through the FSM but the actual *testing intent* is undocumented.
- **Cross-milestone reuse is impossible.** When a later milestone wants to reference "the same kind of subprocess integration test as M-0062 AC-7" there's nothing to link to — the AC body is empty, so the test pattern is invisible to anyone who didn't read the original commit diff.
- **The I3 governance render reads what's written.** The Build / Tests tabs surface the AC's prose context to anyone looking at the static site. Empty bodies render as empty — the site faithfully reproduces the defect.

Empirical scope: every milestone in this repo has empty AC bodies. The gap is repo-wide and reproduces every time `aiwf add ac` runs.

## Possible remedies

Three layers, increasing in invasiveness:

1. **`aiwf check` finding `acs-body-empty`** *(load-bearing)* — warning when an AC's body section under `### AC-N — <title>` contains no non-heading content; error under `aiwf.yaml: tdd.strict: true`. Definition of "empty": between the AC's heading and the next `###` (or EOF), there is no markdown content other than whitespace. Catches every existing milestone immediately and prevents new ones from shipping bare. This is the chokepoint that backs the design intent.

2. **`aiwf add ac` accepts `--body` / `--body-file` per AC** — extend the existing pattern (whole-entity `--body-file` is already in place, per the `aiwf-add` skill's documented flags) so the verb can scaffold the body content in the same atomic commit that creates the AC. Multi-AC form takes one `--body-file` per `--title` (positional pairing) or accepts a directory of files named `AC-N.md`. Keeps the chokepoint enforcement local to the verb the author is already invoking.

3. **`aiwf-add` skill update** — name "fill in the body section before declaring the AC done" as the next required step after `aiwf add ac`. Document the expected shape (one paragraph: pass criteria, edge cases, code references). Pure documentation, deferrable, but cheap and high-signal.

Layer 1 alone closes the gap mechanically; layers 2 and 3 reduce friction. The promote-time guard that G-0055 considered (a milestone can't transition `draft -> in_progress` without all AC bodies populated) is overkill once the check finding is in place — same reasoning as G-0055's deferred promote-time guard.

## Out of scope

- Changes to the title-length validator (`≤80 chars` is fine; the body is where the detail goes).
- A schema for body content. The body is markdown prose, and the kernel principle "prose is not parsed" applies (`acs-and-tdd-plan.md:197`). The check rule asserts presence, not structure.
- Retroactive backfill of historical milestones' AC bodies. The grandfather rule from G-0055 applies: existing milestones surface as warnings, but `acs-tdd-audit` is not retroactively engaged. Authors can choose to backfill prose, but it is not blocking.

## Related

- [G-0055](G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) — same class of defect: design intent silently absent because no chokepoint exists. The remedy pattern (verb chokepoint + check finding + skill update) transfers directly.
- [E-0016](../epics/E-16-tdd-policy-declaration-chokepoint-closes-g-055/epic.md) — the epic closing G-0055; the AC body fleshing pass that follows this gap (one commit per milestone) is dogfooding the design intent for E-0016's own ACs.
