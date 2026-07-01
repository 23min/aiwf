# Epic wrap — E-0053

**Date:** 2026-06-30
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0053-make-aiwf-check-fast
**Merge commit:** af947958

## Milestones delivered

- M-0215 — Profile aiwf check + the policies suite to a per-rule wall-time baseline (merged 2f3d70ad)
- M-0216 — Shared per-check git-history context: collapse per-entity subprocess fan-out (merged 0db5f508)
- M-0220 — Re-fixture heavy real-tree check integration tests to synthetic fixtures (merged 5ef295b1)

## Milestones cancelled (measured not worth it)

- M-0217 — Skip redundant pre-push golangci-lint via a last-green lint marker — dropped after two independent adversarial reviews flagged it as risky for a marginal win.
- M-0218 — Drive the internal/policies suite below its ~9s floor — dropped: the policies suite runs fully overlapped *behind* the integration package, so optimizing it changes no wall-clock anyone waits on. This finding redirected the effort into M-0220.
- M-0219 — Wire git commit-graph maintenance into init/update — dropped after a measurement spike: the projected ~9s win shrank to ~1.5s/6% post-M-0216, and git already writes the commit-graph by default (`gc.writeCommitGraph`). Closed G-0322 (wontfix).

## Summary

E-0053 set out to cut the wall-time of `aiwf check` (the pre-push/CI chokepoint) and the policies test suite. M-0215 profiled the bottleneck; M-0216 shipped the real `aiwf check` win — a shared per-check git-history context that collapses per-entity subprocess fan-out; M-0220 cut the test-suite critical path ~25% (`go test -parallel 8 ./...` 93s → 70s) by re-fixturing a redundant 35s real-tree check rendering test to a synthetic fixture and sharing the built binary across check tests. Three further milestones (M-0217/M-0218/M-0219) were measured and cancelled — the measurement-first discipline doing exactly its job: drop work whose projected win didn't survive contact with the numbers. Net: the two genuine wins shipped; three speculative ones were cleanly dropped; no guarantee weakened and no rule moved from pre-push to CI.

## ADRs ratified

- none — E-0053's decisions (M-0216's shared-context refactor, M-0220's fold-vs-synthetic-fixture choice) are internal / test-suite shaping, captured in the milestone specs, not durable architectural choices with rejected alternatives.

## Decisions captured

- none as `D-NNNN` — M-0220's fold-vs-synthetic decision is recorded in its "Decisions made during implementation" section; the three cancellations are recorded in their milestone/gap cancel rationales (visible via `aiwf history`).

## Follow-ups carried forward

Future check-perf levers (profiled in M-0215, unstarted):
- G-0323 — Incremental `aiwf check` via a validated trunk watermark (walk only new commits)
- G-0324 — Branch hygiene: prune merged ritual branches / oracle skips merged refs
- G-0325 — Parallelize `aiwf check` independent history walks + blob reads

From milestone work / review:
- G-0327 — Harden missing/non-zero blob in the fsm-history walk to a finding (M-0216 review)
- G-0328 — Golden-fixture byte-identity comparator for `aiwf check` (M-0216 review)
- G-0329 — Committing verbs run during a pending merge silently reset MERGE_HEAD (merge-safety)
- G-0326 — `aiwf add` permits empty load-bearing bodies on born-complete kinds (unrelated discovery)

Resolved during the epic (close-out pending):
- G-0330 — `internal/cli/integration` (~128s) is the test-suite critical path — **addressed by M-0220**; promote to `addressed` as a follow-up.

Unfiled:
- `internal/contractverify` `TestRun_EvolutionRegression` ETXTBSY flake (exec-after-write race under parallel load) — recommend filing a gap; it can flake CI on any push.

## Handoff

The remaining structural check-perf levers (G-0323 incremental watermark walk, G-0325 parallelize walks) are the next tier if `aiwf check` wall-time becomes a pressure point again — both profiled-but-unstarted. The test-suite floor (~60s integration, ~70s full `./...`) is now the genuine cost of end-to-end seam testing (real-binary subprocess + real git commits), not waste — documented in M-0220's Deferrals. No release follows this wrap (closure only).

## Doc findings

Scoped doc-lint over the epic's changed markdown (milestone specs, CLAUDE.md cadence note, derived ROADMAP, gap entities): no broken code/file references. The milestone specs were reviewer-validated at each wrap; ROADMAP is regenerated at step 4 of the wrap sequence.
