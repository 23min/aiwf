---
id: G-0106
title: TestBinary_ArchiveKernelMigration_LeavesCheckClean assertion depends on transient kernel-tree housekeeping state
status: open
---
# Problem

`TestBinary_ArchiveKernelMigration_LeavesCheckClean` in `cmd/aiwf/archive_kernel_migration_test.go` has a load-bearing dependency on a *transient* state of the live kernel planning tree: at line 67-70 it asserts (via `t.Fatalf`) that the pre-sweep `aiwf check` output contains `archive-sweep-pending` or `terminal-entity-not-archived`. That holds only when the kernel tree happens to carry at least one terminal-status entity not yet swept into `archive/`.

The test passed at commit `83339cf` and started failing at `cd87ced` — the archive sweep that cleaned the last unswept terminal (M-0090 + E-0027) out of the kernel tree. Routine housekeeping broke the test.

# Reproduction

```
go test -count=1 -run TestBinary_ArchiveKernelMigration_LeavesCheckClean ./cmd/aiwf/...
```

Fails on `main` (HEAD `7ccae87` at filing). Output:

```
archive_kernel_migration_test.go:69: pre-sweep aiwf check did not surface terminal-entity-not-archived or archive-sweep-pending — fixture not in expected pre-sweep state
output:
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit skipped

1 findings (0 errors, 1 warnings)
```

# Root cause: test design, not test data

The test copies the live kernel tree (`work/`, `docs/adr/`, `aiwf.yaml`) into a tempdir at test-run time (line 49-51) — the "fixture" is whatever shape the kernel tree carries when CI fires. M-0085 AC-7's original intent was to prove the *historical migration* (per ADR-0004 §Migration) leaves the tree clean. At authoring time the kernel tree was guaranteed to have unswept terminals (the migration's seed state); the assertion held trivially.

Post-migration, that guarantee evaporated. The test continued to pass only because the kernel tree habitually carried at least one terminal-but-active entity (M-0090 was the most recent example, sat un-swept for several days post-wrap). The first time the tree is fully housekept, the assertion's premise vanishes and the test fails with a misleading "fixture not in expected pre-sweep state" message — even though the tree being clean is precisely the desired steady state.

This is a latent landmine: every future `aiwf archive --apply` on a kernel tree with no further unswept terminals re-arms the failure. Cycle:

1. Milestone closes → tree gains unswept terminal → test passes.
2. Someone runs archive sweep → tree becomes clean → test fails.
3. Next milestone closes → tree un-cleans → test passes again.

That oscillation makes the test useless as a regression chokepoint — it tracks kernel-tree housekeeping schedule, not archive verb correctness.

# Why not a fixture-drift framing

My initial summary called this "fixture drift from kernel emission." That was wrong. The kernel renderer's output format hasn't changed; what changed is the live kernel tree's state. The fix is to break the test's coupling to that transient state, not to update an emission expectation.

# Fix options

| Option | Shape | Strength | Cost |
|---|---|---|---|
| **1. Synthesize Case A in the tempdir** | After the kernel-tree copy + initial commit, mutate one entity's frontmatter to a terminal status (e.g., flip a sample gap to `wontfix`) and commit. Tempdir then guaranteed-carries an unswept terminal regardless of live tree state. | Pins the assertion independently of kernel housekeeping. Exercises the substantive verb path every run. | ~20-30 lines of helper code in one file. |
| **2. Accept both Case A (rich) and Case B (trivial)** | Drop the pre-sweep `t.Fatalf`; let the test proceed regardless. Relax the "exactly one commit" assertion to handle the no-op case. | Cheap; defensible. | Weaker chokepoint — exercises whichever case happens to apply at CI time. |
| **3. `t.Skipf` when pre-condition absent** | Skip the test entirely when the kernel tree is clean. | One-line change. | The test becomes a no-op in the common steady state — verifies nothing in practice. |

Recommendation: **Option 1.** Removes the landmine, pins the test's claim, exercises the substantive path every run.

# Implementation sketch (Option 1)

After line 57 (`mustExec(t, repo, "git", "commit", ..., "seed kernel-tree copy ...")`), add a helper that guarantees a terminal-active entity exists in the tempdir:

```go
ensureTerminalActiveEntity(t, repo)
```

`ensureTerminalActiveEntity` scans `repo/work/gaps/` for any `*.md` file whose frontmatter has `status: open`, rewrites it to `status: wontfix`, `git add`s + commits with a `seed: synthesize a terminal-active gap for the test premise` subject. If no `open`-status gap exists (very unlikely on the kernel tree, but the helper degrades to a fatal error with a clear message), the test fails fast with a comprehensible reason. The selection is arbitrary — the test cares that *some* entity is terminal-but-active, not which one.

Alternative: invoke the binary itself (`aiwf add gap` + `aiwf cancel`) to create one. Cleaner provenance, but couples the test setup to two more verb invocations. Direct file-rewrite is simpler.

# Verify post-fix

- Test passes on a kernel tree where everything has already been archive-swept (the failure mode this gap reports).
- Test passes on a kernel tree carrying its own unswept terminals (the pre-fix passing case — no regression).
- `archive --apply` produces exactly one commit in both cases (the synthesized terminal is what gets swept; the test's count assertion stays valid).

# Why not urgent (but worth doing soon)

The test's `t.Fatalf` makes the binary-test suite red on the current kernel state. CI is failing on `main` now. The pre-commit hook only runs `internal/policies/...` so consumer commits aren't blocked, but the CI red status itself is a real signal that should be fixed promptly — both for general hygiene and because future contributors will start tuning out test signals if the suite is chronically red.

# Suggested resolution

`wf-patch` ritual: branch `fix/archive-migration-test-decouple`, implement Option 1, verify both pre- and post-archive-sweep states pass, commit, merge to main. ~30 minutes including verification.
