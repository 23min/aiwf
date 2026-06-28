# Epic wrap — E-0047

**Date:** 2026-06-28
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0047-harden-and-ship-the-aiwf-aware-claude-code-statusline
**Merge commit:** 560e3910 (merge of epic tip 3a3752cb into main e85da03d)

> Retroactive record. This wrap.md was not created during the wrap-epic merge (an
> omission); it is reconstructed here from the epic branch's git history. See
> "Wrap hygiene" below for the related bookkeeping gap it surfaced.

## Milestones delivered

- M-0191 — Behavioral test harness for the statusline + stale-CI-after-push fix (merged 2bccadd2)
- M-0192 — Statusline shows in-flight epics (branch-contextual epic HUD) (merged 1abf9580)
- M-0193 — Statusline health indicator from a cached `aiwf check --fast` findings signal (merged cd391c0d)

## Milestones cancelled

- M-0194 — Ship the statusline via aiwf init/update with portability fixes — **cancelled** (4013c36c).
  Re-scope: the consumer install path (G-0183) was already delivered and tested by M-0155
  (`go:embed` + `--statusline` with `--scope project|user`, scaffold-if-absent, both scopes),
  and the portability fixes (macOS `tac` fallback, IFS-default sync parse) had already landed in
  M-0153. No build work remained, so the capstone milestone was unnecessary.

## Summary

Hardens the aiwf-aware Claude Code statusline (`.claude/statusline.sh`) into a behaviorally-tested,
branch-correct, health-aware artifact. M-0191 replaces the regex-over-source assertions (which never
ran the script — the `||`-binding bug nearly shipped because of exactly that) with a Go harness that
*runs* `statusline.sh` end-to-end against a hermetic temp git repo + transcript fixture + stubbed
`gh`, strips ANSI, and asserts the rendered segment shapes; its first behavioral target is the
stale-CI-after-push fix — the CI segment compares the run's `headSha` against local HEAD and renders
a gray pending glyph on mismatch, with HEAD folded into the cache key so a push auto-invalidates.
M-0192 adds the branch-contextual epic HUD. M-0193 adds the health glyph (⚠), driven by a new
content-only `aiwf check --fast` mode read from a cheap cached signal — render-safe by construction,
never a live full check. The planned "ship" milestone (M-0194) was cancelled once the install path
and portability work were found already complete (M-0155 / M-0153).

## ADRs ratified

- none. M-0194 would have used ADR-0015 (settings-edit consent) for the `--statusline` wiring, but
  that ADR predates this epic and M-0194 was cancelled.

## Decisions captured

- none as standalone D-NNNN entries — milestone-local decisions live in each milestone spec.

## Gaps closed

- G-0188 — Statusline shows no in-flight epics on non-ritual branches (addressed by M-0192; the
  in-flight-list behavior was subsequently superseded by G-0304's session-entity narrowing).
- G-0290 — aiwf statusline shows a warning indicator when check reports findings (addressed by M-0193).
- G-0183 — aiwf has no install path for its aiwf-aware statusline (found already delivered by M-0155;
  confirmed at the M-0194 cancel; archived/addressed).
- G-0187 — Statusline rendering has no end-to-end behavioral test (**delivered** by M-0191, commit
  64ae762c — see Wrap hygiene).
- G-0189 — CI statusline shows stale result after push (**delivered** by M-0191, commit 64ae762c —
  see Wrap hygiene).

## Wrap hygiene

Two pieces of wrap bookkeeping were skipped at the original wrap and are recorded here:

1. **This wrap.md was never written.** Reconstructed retroactively (this file).
2. **G-0187 and G-0189 were delivered but never promoted.** Their fix landed in M-0191's `64ae762c`
   ("render stale CI as pending; add behavioral harness (G-0189, G-0187)"), but the
   `aiwf promote … addressed` step was missed, so both still carry `status: open`. Recommended
   follow-up: `aiwf promote G-0187 addressed --by-commit 64ae762c` and the same for G-0189, then an
   archive sweep.

## Follow-ups carried forward

- G-0302 — `aiwf check --fast` omits in-memory contract-config validation (open; discovered during M-0193).
- Post-epic statusline continuation on main: G-0303 (aggregate the CI glyph across all workflows for
  HEAD — addressed), G-0304 (narrow the HUD to the session entity + adopt the `patch/G-NNNN-<slug>`
  branch convention + repo-name fix — addressed), and G-0305 (reconcile the M-0193 health surface with
  the per-producer `health.aiwf.json` model — open).

## Handoff

The statusline is behaviorally tested, CI-accurate (no longer stale after a push), and health-aware,
with a consumer install path that already existed (M-0155). Ready for the next epic. Deliberately
out of scope and tracked separately: G-0302 (fast-check contract validation) and G-0305 (health-file
reconciliation). One bookkeeping action remains recommended before this epic is fully clean — the
G-0187 / G-0189 addressed-promotes noted under Wrap hygiene.
