---
title: Normative-docs drift audit — where the prose stopped matching the kernel
status: captured
date: 2026-08-06
---

# Normative-docs drift audit — where the prose stopped matching the kernel

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf
entity kind ([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so this file lives under `docs/initiatives/` as an umbrella capture,
following the precedent of [`id-lifecycle.md`](id-lifecycle.md) and
[`quality-signal-and-cadence.md`](quality-signal-and-cadence.md).

Unlike most initiatives, this one is an **inventory rather than a proposal**.
It is a dated snapshot of measured divergence between the Normative-tier docs
and the kernel as built. It ages by construction: every entry is either fixed
(and deleted) or still true. The `date:` above is what makes it honest — read
it as an observation from that day, not as current truth about the tree.

Tracked by [G-0560](../../work/gaps/G-0560-the-normative-doc-tree-has-drifted-from-the-kernel-it-documents.md).
The code-side root cause under §F is tracked separately by
[G-0559](../../work/gaps/archive/G-0559-schema-idformat-literals-still-carry-pre-adr-0008-narrow-widths.md).

---

## Scope and method

The audit covers the Normative tier as CLAUDE.md defines it: `docs/architecture.md`,
`docs/overview.md`, `docs/workflows.md`, `docs/skill-author-guide.md`,
`docs/design/`, and `docs/migration/`. `docs/adr/` was checked for supersession
hygiene rather than read end to end.

Every claim below was checked against a binary built from the working tree
(`go build -o … ./cmd/aiwf`), not against the `aiwf` on PATH — plus live probes
in a disposable repo for the behavioral claims, and source reads for the
structural ones. Claims are cited `file:line`, with the measured counter-evidence
alongside.

## Why nothing catches this

`wf-doc-lint` is grep-based: it finds broken links, removed-feature docs, orphan
files, and stale CLI invocations. Every finding here is a **semantic** claim —
prose that parses fine, links fine, and is false. The `doc-id-*` rules only see
what `aiwf.yaml`'s `docs.paths` names (`README.md` and `docs/workflows.md`), so
the rest of the tree is unscanned even for the mechanical id-width class.

The drift clusters by age, not by topic. Files touched on or after 2026-07-29
are current; the 2026-07-21/22 cohort carries the bulk of it.

---

## A. Prose that states the opposite of current behavior

The reader who follows these is actively misled.

**`workflows.md:88` — epic roll-up is enforced now.** The doc says an epic can go
`done` "regardless of whether every milestone underneath is itself `done`. The
framework deliberately does not enforce roll-up", closing with "that warning is
the only nudge." Measured:

```
aiwf promote: cannot promote E-0001 to done: 1 non-terminal child milestone(s)
[M-0001] (epic-promote-non-terminal-children); cancel or done each before
promoting the epic to a terminal status
```

`aiwf cancel` refuses symmetrically, and `epicTerminalNonTerminalChildren`
(`internal/check/check.go:170`, G-0393) is a standing rule. A deliberate
non-guarantee became a hard guarantee; the doc still sells the non-guarantee as
a design position.

**`tree-discipline.md:57–64`, `:94` — the CLAUDE.md carve-out was reversed.** A
whole section titled "Why no marker-managed CLAUDE.md fragment" argues aiwf
must not write to the consumer's `CLAUDE.md`, ending "Consumer's `CLAUDE.md` —
*not* aiwf's responsibility." ADR-0018 decided the opposite. `aiwf init` in a
fresh repo writes a `CLAUDE.md`, and `init`/`update` maintain the marker-wrapped
`@.claude/aiwf-guidance.md` import (default-on, `guidance.wire_claudemd` opts
out). `design-decisions.md:201` documents the current behavior — so two
Normative-tier docs now contradict each other on a kernel commitment.

**`skill-author-guide.md:154` — the stated rationale for Rule 4 is false.**
"`aiwf check` does not enforce them at runtime", of the trailers.
`provenance-untrailered-entity-commit` exists and fires
(`internal/check/provenance.go:47`).

**`render roadmap --write` does not commit** — asserted in three places:
`skill-author-guide.md:28` (cheat-sheet column reads **yes**),
`design-decisions.md:251` (listed among the one-commit-per-verb set), and
`workflows.md:225` ("if you want it committed"). Measured: HEAD does not move
and `ROADMAP.md` is left untracked. `--help` agrees — "updates the file on disk
only (no commit — the caller commits it)."

**`architecture.md:90` — moves are not `git mv`.** ADR-0022 replaced the
mechanism; `internal/verb/apply.go:20` reads "runs every OpMove via a pure
filesystem rename". `design-decisions.md:100,102` repeats the stale claim.

**`design-decisions.md:100` — `rename` no longer touches the title.** "does
`git mv` plus a title update." `rename` is slug-only; `retitle` owns the title
(ADR-0037).

## B. Worked examples that fail as written

**Bare `aiwf add adr|gap|decision` is refused.** G-0326's born-complete-kind
gate:

```
aiwf add: ADR-0001: empty load-bearing body section(s) `## Context`,
`## Decision`, `## Consequences`; … pass --body "..." or --body-file <path>
with real prose, or --force --reason "..." to create anyway
```

Affected: `workflows.md:142,150,159`; `skill-author-guide.md:96` — which is step
4 of the guide's single worked example, the thing a scaffolder copies first.
Epic and milestone are exempt from the gate.

**Every `aiwf add milestone` example omits the required `--tdd`.**

```
aiwf add: --tdd <required|advisory|none> is required for kind=milestone
```

Affected: `workflows.md:22,55,126,129,131,169,216`.

**`design-decisions.md:213` documents `aiwf upgrade --yes`.** No such flag —
`--version`, `--check`, `--root` only.

**`provenance-model.md:233–234` names two finding codes the kernel never
emits**: `provenance-no-active-scope-to-pause` and
`provenance-no-paused-scope-to-resume`. Neither appears anywhere in source; the
real refusal is uncoded prose with no `code` field in the JSON envelope. Finding
codes are a published-surface item per `architecture.md:156`, which makes this a
phantom contract rather than a typo.

## C. Stale enumerations

**Narrow-width id formats contradict ADR-0008** (`CanonicalPad = 4`,
`internal/entity/allocate.go:19`): `overview.md:15–20`,
`design-decisions.md:47–52`, `tree-discipline.md:22–27` all publish the
`E-NN` / `M-NNN` / `G-NNN` / `D-NNN` / `C-NNN` shapes. See §F — the root cause
is code, and fixing the docs alone re-seeds the drift.

Raw narrow-id counts across the tier on the audit date, as a sizing signal only:
provenance-model 30, overview 12, id-allocation 12, design-decisions 9,
import-format 5, design-lessons 3, architecture 3.

**`design-decisions.md:251` — the one-commit-per-verb list omits eleven verbs**:
`retitle`, `edit-body`, `set-area`, `set-priority`, `rename-area`, `archive`,
`authorize`, `acknowledge`, `milestone depends-on`, `milestone tdd`,
`worktree add`. The read-only list at `:255` omits `list`, `show`, `schema`,
`template`.

**`skill-author-guide.md:13–29` — the verb cheat-sheet** omits the same verbs
plus `aiwf list` and `aiwf show`, the two read verbs a skill author reaches for
first.

**`skill-author-guide.md:150` teaches avoidance of a valid field.** It names
`priority:` among fields that are not in the schema. `aiwf schema gap` reports
`optional fields: discovered_in, addressed_by, addressed_by_commit, priority`.

**`architecture.md:5,182` — "the seven kernel commitments".**
`design-decisions.md` carries thirteen subsections under "What the PoC commits
to"; CLAUDE.md enumerates ten. Seven matches neither.

**`design-lessons.md:69` — "the eight embedded skills".** There are nineteen
under `internal/skills/embedded/`.

**`design-decisions.md` omits three of CLAUDE.md's commitments outright**: the
canonical 4-digit width (#2), NoOp same-state convergence (#7, ADR-0036), and
the uniform archive convention (#10, ADR-0004). CLAUDE.md points at this file as
where the commitments are distilled, so the omission is load-bearing.

**`tree-discipline.md:18–27` — "The loader recognizes exactly these shapes"**,
followed by six paths with no `archive/` variants.
`internal/entity/entity.go:711–728` recognizes a one-level `archive/` segment
for every kind (ADR-0004).

## D. Cross-references that resolve to nothing

Link-checkers pass on all of these; the *target* is what has moved.

**`design-lessons.md:88` cites a principle by name into two files that do not
contain it** — "the existing **'immutability of done'** principle (in
`architecture.md` and the root `CLAUDE.md`)". The phrase survives only in
`docs/archive/architecture.md:446`, the archived pre-migration doc.

**`design-lessons.md:90,92` — the verb-design checklist cites verbs aiwf does
not have**: `hotfix`, `complete`, `set-status`. They are vocabulary from the
prior system. CLAUDE.md's own "Designing a new verb" section — which `:114`
claims was derived from this document — uses correct aiwf examples, so the
source is stale relative to its own derivative.

**`provenance-model.md:74` points at a section that does not exist** — "(No
`ended` — see "Scope termination.")".

**`id-allocation.md` contradicts itself.** `:150` lists "A walk over every ref in
the repo" under *What this is not*, while the E-0052 update block at `:36–47`
directly above states the allocator now unions every local `refs/heads/*` and
every remote-tracking `refs/remotes/*` (ADR-0025). Separately, `:49–51` quotes a
doc comment "at `internal/entity/allocate.go:34`" that has since been rewritten
and moved — line 34 is now `IDPrefix`.

**`provenance-model.md:71` is internally inconsistent on `bot/`.** It says
principal is "Required when `aiwf-actor:` starts with `ai/`"; `:52` and `:86`
correctly say human-vs-everything-else, matching `--help`. A `bot/` actor reads
as exempt.

**`legal-workflows-audit.md:408–409` (R-AUDIT-0201/0202) describe a mechanism
that no longer exists** — a cross-repo fixture pattern at
`internal/policies/testdata/<skill-name>/SKILL.md` plus a drift check against
`~/.claude/plugins/cache/ai-workflow-rituals/`. That directory is absent and
ADR-0014/0016 retired the channel; the current backstop is
`internal/policies/skill_edit_provenance_backstop.go`, which asks a skill edit
to name its owning entity rather than to be referenced by a test. `ADR-0010:98`
carries the same stale reference.

**Rituals-plugin framing survives ADR-0014/0016** at `architecture.md:54`
("lives in the rituals plugin (`ai-workflow-rituals` repo)" — archived upstream,
now hand-edited in-tree) and `design-decisions.md:145,157,161,186`. ADR-0007
handles the same transition well, with an explicit "See also ADR-0014" header
saying what changed and what survived; the narrative docs should copy that
pattern rather than be rewritten silently.

## E. Framing and hygiene

**`design-decisions.md:3` and `:285` are two migrations out of date** — "this
branch deliberately does not include those documents" and "The PoC is
deliberately discardable. The branch is not planned to merge back to `main`."
This is `main`, and this is the shipped kernel.

**`legal-workflows-audit.md:3` self-describes as `Status: in_progress`**, keyed
to M-0121. M-0121, M-0122 and M-0123 are all `done` and archived — Pass C landed.

**Drafting-history narration**, against CLAUDE.md §"state the conclusion, not the
drafting history": `design-decisions.md:155` ("Earlier drafts named an
`acs-transition` check rule; it was never shipped"), `:205` ("Earlier in the PoC,
`aiwf update` refreshed only skills"), `design-lessons.md:88` ("An earlier draft
included a fourth principle"), `:112–114` (a struck-through progress log),
`tree-discipline.md:59` ("Earlier design rounds considered…"),
`id-allocation.md:21,36` ("Later widened — see Update below").

**`design-lessons.md` numbers its three principles §1, §2, §6** with no §3–§5 and
no note explaining the gap. `architecture.md:170–174` cites the same numbering,
so the two are consistent — but a first reader hits an unexplained hole.

**`architecture.md:187–188`** ends with two consecutive bullets both linking
`../CLAUDE.md`, the second labelled "Go-specific rules" — a target that moved.

## F. Adjacent — not doc bugs, but they gate the doc fix

**The id-format drift originates in code.** `internal/entity/entity.go:564+`
hardcodes `IDFormat: "E-NN"` per kind, and the published schema surface prints
it:

```
$ aiwf schema
  id format:        E-NN
  id format:        M-NNN
```

The docs faithfully copy `aiwf schema`. Correcting §C's tables without
correcting the emitter leaves the published surface contradicting ADR-0008 and
regrows the same drift on the next doc pass. Sequencing matters here: code
first, docs follow from a correct source. Tracked as
[G-0559](../../work/gaps/archive/G-0559-schema-idformat-literals-still-carry-pre-adr-0008-narrow-widths.md),
which therefore gates §C.

**ADR-0003 is `accepted` and unimplemented.** "Add finding (F-NNN) as a seventh
entity kind" — there is no `KindFinding`, no `work/findings/`, and every
Normative doc plus CLAUDE.md commitment #1 asserts six hardcoded kinds. ADR-0001
handles the identical situation correctly by staying `proposed`, with D-0037
recording the deferral. ADR-0003 is an accepted architectural decision the
kernel contradicts, with nothing in the Normative tier marking it as pending.
This is a planning-state disposition, not a prose fix.

## What was checked and found clean

`docs/design/growth.md`, `performance.md` and `oracles.md` — all touched
2026-08-02..04 — hold up. `growth.md` is the model worth copying: every figure
carries the date it was measured, names the script that reproduces it
(`scripts/growth-report.py`), and keeps an iteration log, so the numbers age
into history rather than into falsehood.

`migration/from-prior-systems.md` verified clean, including the
`ROADMAP.md ## Candidates` claim (`internal/cli/render/render.go:198`).
`tree-discipline.md`'s `filepath.Match` glob semantics are accurate. ADR
supersession hygiene is good throughout.

---

## Provenance

Audited 2026-08-06 in conversation, against the working tree at that day's
`main`. Method as described under *Scope and method*; no fixes were applied in
the same pass, so every citation reflects the tree as read.
