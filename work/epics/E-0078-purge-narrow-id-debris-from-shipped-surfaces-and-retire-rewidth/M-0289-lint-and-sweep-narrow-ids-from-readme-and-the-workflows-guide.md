---
id: M-0289
title: Lint and sweep narrow ids from README and the workflows guide
status: in_progress
parent: E-0078
tdd: required
acs:
    - id: AC-1
      title: A narrow id in README or the workflows guide fails a gate
      status: met
      tdd_phase: done
    - id: AC-2
      title: Neither README nor the workflows guide carries a narrow id
      status: met
      tdd_phase: done
    - id: AC-3
      title: The deferred doc-residue gap exists naming its three paths and reason
      status: met
      tdd_phase: done
    - id: AC-4
      title: An id written with a slug contradicting the real entity fails a gate
      status: met
      tdd_phase: done
    - id: AC-5
      title: Shipped guidance and skill docs state which id rule applies where
      status: open
      tdd_phase: green
---

## Goal

Stop the two repo-facing docs an assistant reads to learn the workflow from
modelling narrow width as current, behind a width-shaped lint that keeps them
that way — and record the residue this milestone deliberately does not sweep.

## Context

The shipped-surface work is a real-id problem where width is incidental. This is
the opposite: `README.md` and `docs/workflows.md` are repo-facing, real ids in
them are entirely legitimate, and the defect is purely that they are written at
a width no allocator has emitted since the migration. So this needs a genuinely
width-shaped rule over a different corpus with the opposite stance on real ids —
a sibling of the shipped-surface guard, not a mode of it.

Two properties of the corpus shape the rule. The sites are tutorial fiction —
invented ids in a walkthrough — so the fix is the placeholder form rather than
a widened number, and the rule cannot be gated on whether a token resolves. And
they concentrate in command examples rather than sentences, so a rule that
exempts code spans and fenced blocks sees almost none of them.

The two files are in scope because they teach the workflow. The rest of the
active doc tree is not, for a reason worth recording rather than leaving implicit:
its narrow ids are mostly citations of entities that were genuinely real at narrow
width, so the correct fix there is widening to the real canonical id, not
placeholdering. That is a different edit at a lower payoff, and folding it in
would bloat this lint's allowlist.

## Acceptance criteria

### AC-1 — A narrow id in README or the workflows guide fails a gate

A below-canonical-width id shape introduced into a scanned doc produces a finding
naming the file and line. The claim is about width alone — `E-01` is wrong
because no allocator emits two digits, not because of what it does or does not
name. Real canonical-width ids do not fire; unlike the shipped-surface rule,
this corpus is where real ids belong.

Code spans and fenced blocks are in scope, so backticks are not an opt-out. The
debris lives in command examples, and a reader copies a command example as
readily as a sentence.

Severity is a warning by default and an error where `aiwf.yaml` raises it. A
repo that migrated its entities still carries narrow ids through its docs —
`rewidth` never touched prose — so an error-by-default rule would block pushes
in every such repo on upgrade, over files its operator never edited, with
neither a fixer nor a suppression mechanism available. This repo raises it to
error; a consumer opts in once their own sweep is done.

Evidence: a fixture asserting fire on a narrow id and no-fire on the canonical
form of the same id, for each kind prefix, at both severities.

### AC-2 — Neither README nor the workflows guide carries a narrow id

Both files are swept. Where the narrow id was a teaching example, it becomes the
canonical placeholder form; where it named a real entity, it becomes that
entity's canonical id.

Evidence: the rule from AC-1, run over the real files, reports zero findings.

### AC-3 — The deferred doc-residue gap exists naming its three paths and reason

The residue this milestone declines is captured as its own gap rather than left
as an informal intention — naming the three paths and the widen-rather-than-
placeholder reason, so the next reader does not re-derive the scoping decision or
mistake the omission for an oversight.

Evidence: a structural assertion that the gap resolves through the loader and its
body names all three paths.

### AC-4 — An id written with a slug contradicting the real entity fails a gate

A doc that writes an id together with a slug — the shape a file path and
`aiwf add` output both take — produces a finding when that slug is not the
slug the entity actually carries.

This closes the half of the fiction problem that has an exact test. The width
rule cannot see a fictional id at canonical width, and the sweep found one
being used for an invented ADR while that id names a real and entirely
unrelated entity, with a filename spelled out that contradicts the real one.
Nothing would have caught its return.

The comparison is a string equality against a path the loader already holds,
so there is no heuristic and no false-positive surface: either the slug
matches the entity's own or it does not. It is deliberately narrower than
"every id in a doc must resolve" — that is a far larger behavioral change,
carrying the cross-branch and archived-id questions the entity-body rule
already had to settle, and it is tracked separately rather than folded in
here.

The residue is a bare canonical-width id used as fiction where no slug is
written. No mechanical signature distinguishes it from a citation, and any
prose-proximity heuristic would fire on legitimate text; the placeholder
convention plus review is what covers it.

Evidence: a fixture pairing a real entity with a doc citing its id under a
different slug, asserting the mismatch fires and the entity's true slug
stays silent.

### AC-5 — Shipped guidance and skill docs state which id rule applies where

The surfaces a consumer reads say which rule governs which files, and say it
in one place rather than leaving it to be assembled from separate finding-code
entries.

Three of the four id-shape rules are live in a consumer repo: `body-prose-id`
over their entity files, and both doc rules over their `README.md`. Two of them
disagree about the same token deliberately — a canonical placeholder is the
defect in an entity body and the correct form in a doc — and backticks exempt
in one and not the other. Met without explanation, that reads as the tool
contradicting itself.

The always-on guidance predates the doc rules and states the entity-body rule
over "committed prose", which now spans both corpora. Read literally it tells a
consumer to strip exactly the placeholders the doc rule asks for. Scoping that
sentence is a correction to what this milestone shipped, not new work.

Rationale stays out of the shipped surfaces: a consumer needs the instruction,
not the argument for it. Why code spans are exempt in one corpus and scanned in
the other, and why entity templates are read by two rules at once, is design
reasoning and belongs in this repo's docs.

Evidence: structural assertions that the guidance names both corpora and both
rule families, and that the check skill carries a section comparing them; plus
an entry in the curated guidance-anchor set, so the scoping cannot silently
drift back out.

## Constraints

- **This lint's corpus and polarity are both distinct from the shipped-surface
  guard's.** Real ids are correct here and defective there; only width is at
  issue. Sharing an implementation is fine, conflating the rules is not.
- **The lint is scoped to the two named files**, not the active doc tree, so its
  allowlist stays short enough to read. `README.md` is the shipped default,
  since every repo has one and it is the doc most likely to cite entities;
  `docs/workflows.md` joins the corpus through this repo's own config.
- **A rule that ships retroactively constrains its severity, not its
  existence.** Blocking a push over prose the operator never edited, in a repo
  with no fixer for it, is the harm the warning default prevents.
- **Stale-width ids inside entity bodies are a different defect** — there the
  prose does name a real entity, so the fix is a widened number and the rule is
  reference-shaped. Out of scope here; AC-3's residue gap is not the place for
  it either.

## Design notes

- The epic leaves open whether this fires from `aiwf check` or from
  `internal/policies`. Decided here: `aiwf check`, as a rule that genuinely
  ships. The corpus is not repo-only — every consumer tree has a `README.md` —
  which rules out the policy tier's usual justification and rules out a marker
  asking whether the rule is running inside aiwf's own repo. It also makes the
  rule worth shipping on its merits: a stale-width id in a consumer's own docs
  is a defect by aiwf's own grammar, so this is a feature rather than
  dogfooding scaffolding.
- What shipping costs is the severity default, not the rule. The blocking
  behavior is what a consumer opts into; `tree.strict` is the precedent for a
  config raising a rule from advisory to blocking.
- Width-shaped rather than reference-shaped. A rule gated on whether the token
  resolves would fire in a densely-allocated tree and stay silent in a young
  one, for identical prose — the id space's population is not a property
  anyone can reason about. Width holds either way.

## Out of scope

- `docs/design/**`, `docs/overview.md`, `docs/architecture.md` — the residue AC-3
  files as its own gap.
- Everything frozen by convention: the doc archive, research and explorations
  trees, the changelog, and the migration ADR itself.

## Dependencies

- None. Independent of the shipped-surface milestones and of the retirement.

## References

- G-0481 — the tier split and the reason the residue is deferred rather than
  swept.

## Work log

### AC-1 — the width rule

`doc-id-width` in `internal/check`, width-shaped and code-in-scope, advisory
by default with `docs.id_width.strict` escalating it at the same seam
`ApplyTDDStrict` uses · commit 568228c74 · tests 11/11

### AC-2 — the sweep

112 narrow sites to canonical placeholders across both docs, plus five
fictional uses of a real ADR's id, a self-contradicting sample finding, and
the quickstart transcript's alignment; `strict` enabled once clean · commit
ece70f010 · tests 2/2
