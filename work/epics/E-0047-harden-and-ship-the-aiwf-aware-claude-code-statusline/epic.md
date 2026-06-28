---
id: E-0047
title: Harden and ship the aiwf-aware Claude Code statusline
status: active
---
## Deliverable

The aiwf-aware Claude Code statusline (`.claude/statusline.sh`) becomes behaviorally tested, correct on every branch, health-aware, and shippable to consumers. Closes the statusline cluster: G-0187 (no behavioral test), G-0189 (stale CI after push), G-0188 (no epics on non-ritual branches), G-0290 (no findings/health indicator), G-0183 (no consumer install path + portability defects).

## Why now / parallel-safety

The statusline lives entirely in the safe zone untouched by the in-flight E-0044 (areas-hardening): `.claude/statusline.sh`, `internal/policies/statusline_*`, and statusline-adjacent render code. So this epic runs in parallel with E-0044 without file contention. Several cluster members are real bugs surfaced in the last ~10 days (G-0189, G-0188, G-0290).

## Milestones (sequence M1 → M4)

1. **M1 — Behavioral test harness + stale-CI fix** (G-0187, G-0189). A test that *runs* the script against fixtures is the foundation every later milestone asserts against; its first target is the stale-CI bug.
2. **M2 — In-flight epics on every branch** (G-0188).
3. **M3 — Health/findings indicator** (G-0290). Reads a cheap *cached* check-state, never a live `check`.
4. **M4 — Ship + portability** (G-0183). `aiwf init`/`update` materializes the statusline; fixes `tac`/literal-tab portability; `--statusline` wires settings via ADR-0015 consent.

## Through-line

M3 establishes a **shared tree-health signal** (findings-state) surfaced on the statusline. The same concept later extends to `aiwf doctor` (G-0289 check-health summary) and the default `aiwf status` divergence flag (G-0277) — a future "health across status surfaces" thread, out of scope here but enabled by M3.
