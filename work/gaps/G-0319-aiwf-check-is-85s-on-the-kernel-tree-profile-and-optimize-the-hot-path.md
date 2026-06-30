---
id: G-0319
title: aiwf check is ~85s on the kernel tree — profile and optimize the hot path
status: open
discovered_in: M-0196
---
## What's missing

`aiwf check` on the kernel's own planning tree takes **~85s** (measured bare,
binary build excluded). That single number is the root of the wrap+push
slowness flagged after the M-0196 wrap:

- It is the dominant cost of the **pre-push hook** (`aiwf check` runs after the
  lint gate; ~85s on every push).
- It is what makes `TestM080_AC6_NoUnexpectedTreeFileWarning` ~82s (G-0320) —
  that test shells the full check.

85s to validate a markdown-and-frontmatter tree is disproportionate; the tree
is not large enough to justify it. There is almost certainly an algorithmic or
subprocess-fan-out hot path.

## Why it matters

`aiwf check` is the load-bearing chokepoint — pre-commit (shape-only), pre-push
(full), and CI all run it, and the test suite shells it. Every second of check
latency is paid many times per wrap. Speeding it up is the highest-leverage
single fix for the whole local-validation experience; it is the *root* that
G-0320 (test fixture) and G-0318 (lint redundancy) only work around at the
edges.

## Proposed investigation

Profile before optimizing. Concretely:

1. `go test`/pprof or a `-cpuprofile` on a `check.Run` harness over the live
   tree; or simply count subprocess spawns (`strace -f -e trace=execve` /
   wrapping `exec.CommandContext`) during one check.
2. **Lead hypothesis — per-entity `git log` subprocess fan-out.** Several rules
   shell git per invocation: `internal/check/acks.go` (`git log` at ~line 62 and
   125), `internal/check/area_mistag.go` (`git log` at ~line 159), and the
   history/trailer-reading rules generally. If any of these run **once per
   entity** rather than once per check, the cost is O(entities) process spawns —
   the classic markdown-tree-validator hot path. Confirm whether the git reads
   can be hoisted to a single `git log` pass shared across rules (one walk of
   the relevant history, parsed once, indexed by entity).
3. Secondary candidates: repeated full-tree filesystem walks (one per rule
   instead of a shared walk), the `body-prose-id` / `skill-body-id` prose scans,
   and cross-reference resolution that may be O(n²) over the entity set.

## Proposed fix shape (pending profile)

If per-entity git fan-out is confirmed: replace per-entity `git log` calls with
a **single bulk history read** parsed once into an index the rules consume —
mirroring how `aiwf history` already reads the log in one pass. Keep the
finding semantics identical; only the data-acquisition path changes. This is a
kernel-internal refactor (its own milestone), behavior-preserving, gated by the
existing check tests + a before/after wall-time measurement.

## Discovered in

M-0196 — measured `aiwf check` at ~85s while profiling the wrap+push slowness;
it is the shared root of G-0320 (the 82s policy-suite test) and the pre-push
hook cost.
