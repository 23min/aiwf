---
id: G-0635
title: Wrap rituals author prose after the review, or with no review at all
status: open
---
## What's missing

Both wrap rituals author substantial prose that no independent reviewer reads.

`aiwfx-wrap-milestone` runs its deciding two-lens review at step 2 and writes the
wrap-side sections at step 4 — `## Work log`, `## Validation`, `## Deferrals`,
`## Reviewer notes`. Step 4's only re-review trigger is scoped to code: *"If it
touched source or tests, re-run step 1's gates and re-enter step 2's scoped
confirmation — a fix landing after the deciding review is still code no reviewer
has seen."* Prose the step itself produces has no equivalent trigger. A tell sits
in the same step, which describes `## Reviewer notes` as what *"the reviewer agent
reads first"* — imagined as input to some later reviewer rather than as something
the current one checks.

`aiwfx-wrap-epic` has no review step at all. It references no reviewer, no
fresh-context pass, and `wf-review-code` nowhere. It authors
`work/epics/E-NNNN-<slug>/wrap.md`, whose `## Changelog entry` is copied verbatim
into `CHANGELOG.md` at its own step 6 and becomes user-facing release notes.

Observed 2026-08-25 during E-0088's wrap. Three independent fresh-context reviews
of M-0316 each returned request-changes. Every confirmed defect across all three
sat in prose; none was in code. The first round's findings sat in material that
existed at step 2 and the review caught them. Later rounds' findings sat in the
step-4 sections — the Work log's attribution of which mutants its tests killed,
and the bodies of two gaps the wrap opened. Those were reviewed only because the
loop re-opened repeatedly, which is not the ritual's normal path. On a
single-pass wrap none of them is seen by anyone but the author.

The epic wrap's own artefact for E-0088 illustrates the second half: its Summary
and its changelog bullets reached `main` with no independent read, because the
ritual has no step at which one could happen.

## Why it matters

The wrap-side sections are where a milestone's evidence lives. The Work log
carries the measurement an AC was promoted on; `## Validation` carries the gate
results; `## Deferrals` names what survives the milestone. A claim that is wrong
there is wrong in the record the tree keeps, and the archived milestone is what a
future reader consults when asking what was actually established.

The epic case reaches further. `wrap.md`'s changelog entry is not internal
record — it is copied into `CHANGELOG.md` and read by anyone tracking what
changed. It is authored once, at wrap, by the same author who did the work, and
nothing between that keystroke and the release notes is an independent read.

Both rituals otherwise specify their steps precisely — exact commands, exact
trailers, exact ordering, with named failure modes for each. The prose steps are
the exception, and they are the steps whose output no gate can check.

## Resolution shape

The two rituals want different fixes, and neither is an extra review round.

**`aiwfx-wrap-milestone` — reorder rather than add.** Three of the four
wrap-side sections can be written before the review instead of after it. Their
inputs all exist by step 2: the Work log cites per-AC implementation commits,
which `aiwfx-start-milestone` lands during implementation; `## Validation` records
step 1's gate results; `## Deferrals` records decisions already taken. Writing
them before the deciding review puts them inside the surface that review already
covers, at no extra pass.

`## Reviewer notes` cannot move. It records the review's outcome, so it cannot be
read by the review it records. That leaves exactly one section unreviewed by
construction, which is worth stating in the ritual rather than leaving as an
accident — a reader who knows which section is unreviewed can weigh it; one who
assumes the whole spec was reviewed cannot.

**`aiwfx-wrap-epic` — a scoped read, not a two-lens review.** The full
milestone-style apparatus is disproportionate for an aggregation ritual whose
milestones were each already reviewed. What is missing is narrower: `wrap.md`
makes fresh claims, and one of them ships to users. A single scoped pass — someone
who did not write it checking the artefact's claims against the tree, before the
changelog copy at step 6 — covers the exposure without importing the machinery.

**Ruled out: changing `wf-review-code`.** Its §7 already requires that entity
bodies and docs state what holds now, and that every attribution be checked
against its source with the exact sentence quoted. It fired correctly on this
material whenever a reviewer was pointed at it. The defect is not what the review
checks or how it reports; it is that in the normal path no reviewer is aimed at
this prose at all. A change there would leave both rituals exactly as they are.

## Where to fix

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
  — steps 2 and 4, and step 4's code-scoped re-review trigger.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`
  — a scoped read of the wrap artefact, before the changelog copy at its step 6.
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-review-code/SKILL.md`
  — no change; recorded above so the reasoning is not re-derived.

Both ritual edits ship to consumers through `aiwf init` / `aiwf update`, and each
must ride a commit naming this gap so `skill-edit-provenance-backstop` passes.
