---
id: E-0040
title: Materialize per-turn aiwf guidance into consumer CLAUDE.md (closes G-0243)
status: proposed
---
# E-0040 — Materialize per-turn aiwf guidance into consumer CLAUDE.md (closes G-0243)

## Goal

Give aiwf a consent-gated write channel into the one consumer surface that is
re-injected on every turn and survives `/compact` — the consumer's `CLAUDE.md` —
so that the advisory rules aiwf cannot mechanically enforce actually bind in
consumer trees, not just in this repo.

## Context

aiwf materializes everything it ships into a consumer repo through exactly two
channels: `.claude/skills/aiwf-*` (verb skills, `aiwfx-*` rituals, `wf-*`
engineering skills, role agents, entity templates) and `.git/hooks/<hook>`. Both
are aiwf-owned by convention, off-trunk, and byte-refreshed by `aiwf init` /
`aiwf update`. The consumer's `CLAUDE.md` at the repo root is user-owned, and
aiwf has no write channel into it.

That matters because `CLAUDE.md` is the only consumer surface re-assembled every
turn and re-read fresh after `/compact`. The kernel's strongest advisory rules —
per-action gate discipline, "never suggest the user pause", "collision →
`aiwf reallocate`, not `git mv`" — have no mechanical chokepoint; their only
enforcement is the agent reading them, and that reading drifts across compaction
unless re-anchored every turn. Today those rules live only in skill bodies
(loaded when the skill is invoked), never in the per-turn surface. The strongest
LLM-binding channel is the one channel aiwf cannot reach.

G-0242 lifted the gate-discipline rule into *this* repo's `CLAUDE.md` and the
embedded skill preambles; it deferred the consumer-tree propagation to G-0243,
which this epic closes. The shipping mechanism builds on ADR-0014 / E-0038
(rituals embedded in the binary, materialized by `init`/`update`) and on the
per-invocation-consent precedent set by E-0039 / ADR-0015 (aiwf does not edit a
settings file without explicit consent).

### Design verified during planning

The chosen shape is an `@import`: `CLAUDE.md` gains one marker-wrapped
`@.claude/aiwf-guidance.md` line, and the guidance content lives in a file aiwf
already owns under `.claude/`. Claude Code resolves `@import` at context-assembly
time, inlines it per turn, re-reads it after `/compact`, and resolves relative
paths against the importing file. Verified empirically on Claude Code 2.1.177: an
in-repo `@.claude/...` import loads under a headless `claude -p` run with no
approval dialog — the undocumented "external imports" approval gate fires only
for out-of-repo paths. That is why the imported file must stay in-repo under
`.claude/`, never `~/.claude/` or an absolute path.

## Scope

### In scope

- A guidance file materialized at `.claude/aiwf-guidance.md` (gitignored,
  aiwf-owned, byte-refreshed like the skills), plus a single marker-wrapped
  `@.claude/aiwf-guidance.md` import line in the consumer's `CLAUDE.md` — the
  only edit to a user-owned, committed file.
- Consent-gated wiring on `aiwf init` and `aiwf update`: the import line is wired
  by default (including non-TTY), with `--no-wire-claudemd` to decline;
  `CLAUDE.md` is created if absent; the edit is announced with a printed notice.
  `aiwf update` is idempotent on the line and nudges rather than silently
  re-adding a line the operator removed. Reuse E-0039's wire-settings consent
  machinery where it composes; do not fork a parallel one.
- A new ADR establishing that aiwf does not edit user-owned consumer files
  without risk-calibrated consent — citing ADR-0015 as the settings.json instance
  and setting `CLAUDE.md` to default-on, justified by the lower risk profile.
- The guidance fragment content, authored in the embedded-rituals snapshot,
  version-pinned with an `aiwf-version` comment: per-action gate discipline;
  never suggest the user pause; collision → `aiwf reallocate` not `git mv`;
  AC promotion requires mechanical evidence; Q&A one-decision-at-a-time;
  finish-in-context / don't paper over; plus the id-shape rule as a one-line,
  chokepoint-backed nicety.
- The documented **inclusion principle** governing what may ever enter the
  fragment: a rule qualifies only if it has no mechanical chokepoint, its
  violation is invisible-until-named, and it governs the agent operating aiwf or
  its interaction with the human — with a hard boundary that the fragment says
  nothing about the consumer's own code.
- An advisory `aiwf doctor` finding (`claudemd-guidance-unwired`) that fires when
  the guidance file exists but `CLAUDE.md` does not import it, naming the exact
  remediation command (per G-0199).

### Out of scope

- Ratifying the still-`proposed` ADR-0015. It is a separate loose end; the new
  ADR cites it regardless of its status.
- A suppress knob for the `claudemd-guidance-unwired` finding. Advisory is the
  right pressure level for now; add a knob only if the nag proves annoying.
- Any guidance about the consumer's own codebase (language, test, or build
  conventions). That is the consumer's own `CLAUDE.md`'s job; the fragment is
  strictly about operating aiwf and interacting with the human.
- Writing to any consumer-root file other than `CLAUDE.md`.

## Constraints

- **The imported file stays in-repo under `.claude/`** — never `~/.claude/` or an
  absolute path. That is the only zone free of Claude Code's import-approval
  dialog (verified on 2.1.177); an out-of-repo import would silently fail to load
  in headless sessions.
- **No silent mutation of a user-owned file.** Default-on wiring is licensed by
  "running `aiwf init` is the consent to adopt aiwf" plus a printed notice — not
  silence. The default-on stance is a deliberate departure from ADR-0015's
  settings.json default (opt-in / refuse-in-non-TTY), ratified in the new ADR and
  justified by the lower risk profile: one reversible, append-only line that
  clobbers nothing and cannot force a broken artifact onto a teammate.
- **Correctness must not depend on the LLM.** Each AC pins a mechanical assertion
  per CLAUDE.md's AC-promotion rule — including the doc-shaped ACs (ADR content,
  fragment content, inclusion-principle prose), which assert on named sections or
  on the embedded bytes via a path constant (the G-0182 pattern).
- **The fragment is small and high-signal.** Per-turn re-anchoring only works if
  the content is short enough to actually re-read; the fragment stays terse
  (target well under ~50 lines).
- **Authoring location is the embedded snapshot** (ADR-0014 / ADR-0016), not a
  retired upstream; AC tests assert content against the embedded bytes.

## Success criteria

<!-- Observable at epic close, not tests. -->

- [ ] A consumer who runs `aiwf init` (or `aiwf update`) ends up with a
      `CLAUDE.md` that imports the aiwf guidance, and an LLM in that repo sees the
      guidance every turn — including in a headless `claude -p` session and after
      a `/compact` — with no extra configuration step.
- [ ] A consumer who declines (`--no-wire-claudemd`) is left untouched, and a
      consumer whose tree is unwired is surfaced by `aiwf doctor` with the exact
      command to fix it.
- [ ] No path mutates `CLAUDE.md` without consent: the edit is marker-scoped,
      announced, reversible, and leaves all content outside the markers verbatim.
- [ ] The materialized guidance content reflects the full-set fragment listed
      under *Scope → In scope* and is version-pinned to the binary.
- [ ] The consent decision is ratified in the ADR listed in the *ADRs produced*
      table below, and the inclusion principle is documented in a durable
      location.
- [ ] G-0243 promotes to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Gitignore the guidance file (like the skills) or commit it? | no | Lean gitignored — consistent with the skills model; the committed import line is the only tracked artifact. Settle at the wiring milestone. |
| Does the AC-mechanical-evidence rule belong in the fragment once D-0005 / G-0140 land the `--evidence` flag? | no | Ship it now (advisory today); revisit when G-0140 closes and the rule gains a chokepoint (inclusion-criterion #1). |
| Where does the inclusion principle live — the ADR, CLAUDE.md, or a design doc? | no | Decide at the content milestone; leaning the ADR (it is decision-shaped) with a CLAUDE.md pointer. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A future Claude Code release gates in-repo `@.claude/` imports behind the approval dialog, silently dropping the guidance in headless sessions. | med | Content degrades gracefully (a missing import = no guidance, not breakage); the `doctor` finding surfaces the unwired state; behavior is pinned to a verified version and revisited on upgrade. |
| Default-on editing of a user-owned file surprises an operator. | low | Loud printed notice, marker-scoped and reversible, `--no-wire-claudemd` opt-out, and content outside the markers left verbatim. |
| The fragment accretes rules over time until it is too long to re-anchor. | med | The documented inclusion principle is the chokepoint on growth; additions must clear all three criteria and the hard boundary. |

## Milestones

<!-- Candidates; ids are allocated by aiwfx-plan-milestones, which fills this list
     with real milestone links and dependency edges. Execution order top to bottom. -->

- Embedded guidance artifact (full-set fragment, version-pinned) + the documented
  inclusion principle; AC tests assert content against the embedded bytes.
- Consent-gated `CLAUDE.md` wiring on `aiwf init` / `aiwf update` (the
  `--no-wire-claudemd` flag, marker insert/refresh, create-if-absent, default-on
  including non-TTY, idempotency, materialize the file + import line, printed
  notice); the new consent ADR ratified here.
- `aiwf doctor` `claudemd-guidance-unwired` advisory finding + fixtures.

## ADRs produced

- New consent ADR — aiwf does not edit user-owned consumer files without
  risk-calibrated consent; cites ADR-0015 (the settings.json instance) and sets
  `CLAUDE.md` to default-on. Id allocated when the ADR is authored.

## References

- [G-0243](../../gaps/G-0243-aiwf-cannot-reach-consumer-claude-md-for-kernel-advisory-guidance.md) — the gap this epic closes (carries the full design direction).
- [G-0242](../../gaps/archive/G-0242-per-action-gate-discipline-absent-from-claude-md-rule-does-not-survive-compact.md) — lifted gate discipline into this repo's CLAUDE.md + skill preambles; deferred consumer-tree propagation to G-0243.
- ADR-0015 / E-0039 — the per-invocation-consent precedent for user-owned-file edits (settings.json), whose wiring machinery this epic reuses.
- ADR-0014 / ADR-0016 / E-0038 — the embed-and-materialize mechanism and the embedded snapshot as single source of truth.
- G-0184 / G-0199 / G-0182 — the body-prose-id chokepoint (the id-shape fragment rule), the exact-remediation-command rule (the doctor finding), and the embedded-path-constant test pattern.
- D-0005 / G-0140 — the AC-mechanical-evidence `--evidence` flag that will later give one fragment rule its chokepoint.
- `internal/cli/cliutil/statusline.go`, `internal/skills/settings.go` — the consent-flow code to compose with.
