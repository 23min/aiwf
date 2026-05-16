# Epic wrap — E-0025

**Date:** 2026-05-16
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** none (trunk-based; each milestone merged to main directly)
**Merge commits:**

- M-0091 — `4965636`
- M-0092 — `f12508b`
- M-0093 — `c66d924`

## Milestones delivered

- M-0091 — Roll out TestMain + t.Parallel across internal/* test packages (merged `4965636`)
- M-0092 — Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/ (merged `f12508b`)
- M-0093 — Document test-discipline convention and lock its chokepoint (merged `c66d924`)

## Summary

E-0025 establishes parallel-by-default test execution across the module. Pre-conversion `go test ./internal/... -count=1` walked 53.6s on a 20-core dev host; post-conversion the same invocation runs in 24.5s (~2.2× speedup at default parallelism). The cmd/aiwf/ surface saw a ~47% wall-time reduction (174s → 87s on a clean run).

The convention is now mechanically pinned: every `internal/*` test-bearing package carries a `setup_test.go` with `TestMain(m *testing.M)` (asserted via `internal/policies/test_setup_presence.go`), and every race-mode `go test` invocation across the Makefile + GitHub workflows carries `-parallel 8` (asserted via `internal/policies/race_parallel_cap.go`). `CLAUDE.md` documents the five load-bearing rules under a new `### Test discipline` section in Go conventions; that section's presence is itself asserted structurally so future drift fires CI.

Scope shifts mid-flight: M-0092 AC-4 (the strict "10/10 -race -parallel 8 reliability" reading) deferred to G-0125 once the macOS dev host produced a 20–30% flake rate from dense subprocess fan-out (`gitops.StagedPaths` deadlocks under concurrent verb-dispatch). The first CI run on Linux post-merge **was green** (single iteration, 1m15s); a CI-side 10-run loop would settle G-0125 with high confidence.

## ADRs ratified

- none — the parallel-by-default convention lives in CLAUDE.md (Go conventions, *Test discipline*) and is mechanically enforced. No durable architectural decision was abstracted to ADR shape during the epic; the convention's load-bearing pieces are written into the playbook rule, the chokepoint code, and the milestone specs themselves.

## Decisions captured

- none new — mid-flight decisions are recorded in each milestone spec's *Decisions made during implementation* section:
  - M-0091 — subagent dispatch without `aiwf authorize` produces unrecoverable trailer drift; cited as evidence in E-0031's "Evidence in flight" section for M-0108 to encode in `legal-workflows.md`.
  - M-0092 — AC-4 deferred to G-0125 rather than papered over (CLAUDE.md *Don't paper over a test failure*).
  - M-0093 — belt-and-suspenders structural assertion for AC-1's CLAUDE.md section (`PolicyClaudeMdTestDisciplineSection`); G-0097 closed via `--by E-0025` inside M-0093 rather than at epic wrap.

## Follow-ups carried forward

- **G-0125** — cmd/aiwf -race -parallel 8 flakes under subprocess fan-out (macOS). Status `open`. First Linux CI run on the wrapped state was green; a CI-side 10-run loop would settle whether the macOS-specific flake class is platform-only or a deeper systemic issue. Four remediation paths sketched in the gap body: token-bucket around verb-level git invocations, per-package cap split, refactor specific patterns, accept macOS as degraded host.
- **G-0104** — Test-parallelism discipline: ship to consumers via wf-rituals or BYO? Status `open` (declared not-blocking in its body). Decision becomes interesting once a second consumer hits the same wall.

## Handoff

What's ready for the next epic:
- Test suite runs ~2× faster on internal/* + ~47% faster on cmd/aiwf/ at default parallelism.
- New test files in any `internal/*` package fail CI unless they include `setup_test.go` + `TestMain` — the convention propagates by chokepoint, not by reviewer vigilance.
- The race-cap chokepoint (`internal/policies/race_parallel_cap.go`) pins the `-parallel 8` value uniformly across Makefile + workflows; the policy fires per-file if any race-mode invocation drops the cap.
- `internal/policies/shared_tree_test.go::sharedRepoTree` exposes a `sync.Once`-shared live-repo `*Tree` for any future policy test that walks the planning tree (`// do not mutate`).

What's deliberately left open:
- **Reliability ceiling on macOS** — G-0125 names the remediation candidates but does not pick one. Recommended next action: enable a CI-side 10-run loop (`-race -parallel 8 -count=1` × 10) on a scheduled workflow; if green over 30+ days, close G-0125 as "macOS-host stress only, CI is the chokepoint." If CI flakes too, the remediation conversation has data.
- **`cmd/aiwf/` setup_test.go-presence chokepoint** — scoped out of M-0093 because the cmd-side audit shape (captureStdout/Stderr/Run-caller serialization, integration_g37 file-level serial) is finer-grained than a presence check can express. The discipline is reviewer-enforced via `cmd/aiwf/setup_test.go`'s comment block. File a gap if cmd-side drift becomes a real concern.
- **captureStdout/captureStderr/captureRun helper refactor** — M-0092 documented that ~70 tests stay serial because these helpers mutate package-level `os.Stdout` / `os.Stderr`. A future helper redesign (per-test pipes) would unlock those tests in one move. Out of M-0092's scope per the "no test-semantics change" constraint; candidate for a follow-up gap if CI signal motivates it.

## Doc findings

`wf-doc-lint` was not invoked separately — E-0025 touched `CLAUDE.md` and Go source under `internal/policies/` + `cmd/aiwf/`; no files under `docs/` were modified during the epic. The CLAUDE.md edits land under the existing structural anchors (`## Go conventions` parent heading + the new `### Test discipline` subsection asserted by `PolicyClaudeMdTestDisciplineSection`), so the link/anchor surface is self-consistent.
