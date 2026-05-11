---
id: M-0089
title: Per-code text-render summary with --verbose fallback
status: in_progress
parent: E-0026
tdd: required
acs:
    - id: AC-1
      title: Default text output is per-code-summarized for warnings
      status: met
      tdd_phase: done
    - id: AC-2
      title: Errors still print per-instance in default text
      status: met
      tdd_phase: done
    - id: AC-3
      title: --verbose flag restores full per-instance output
      status: met
      tdd_phase: done
    - id: AC-4
      title: JSON envelope is unchanged
      status: met
      tdd_phase: done
    - id: AC-5
      title: aiwf check --help documents --verbose
      status: met
      tdd_phase: done
    - id: AC-6
      title: cmd/aiwf/completion_drift_test.go passes
      status: met
      tdd_phase: done
    - id: AC-7
      title: Kernel-tree integration test — default output is short
      status: met
      tdd_phase: done
    - id: AC-8
      title: Summary lines name the code structurally, not by substring grep
      status: open
      tdd_phase: done
---

# M-0089 — Per-code text-render summary with `--verbose` fallback

## Goal

Replace the flat per-finding text render of `aiwf check` with a per-code summary by default; add a `--verbose` flag that restores the current full-detail behavior. Errors continue to print per-instance even in default mode (each error is per-instance-actionable). JSON envelope output is byte-identical to the pre-milestone baseline.

## Context

E-0024 made `aiwf check` produce 177 lines on the kernel tree in its pre-sweep state, of which 176 are near-identical advisories sharing the same hint. The text renderer has always concatenated findings flat; the friction is observable now because the aggregate-paired finding shape (`archive-sweep-pending` summarizing `terminal-entity-not-archived` leaves) is the new normal until the historical migration runs. This milestone collapses default text output to per-code summaries while preserving every other surface unchanged.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0089 --title "..."` at start time per aiwfx-plan-milestones anti-pattern guidance. Intended landing zone below. -->

Intended landing zone:

- **AC-1: Default text output is per-code-summarized for warnings.** Each warning-severity code appears once as `<code> (warning) × N — <representative message>`. Test via fixture tree containing ≥2 instances of a warning code; assert the summary line appears once and the per-leaf lines do not.
- **AC-2: Errors still print per-instance in default text.** A fixture containing 3 instances of an error-severity finding produces 3 lines (path + code + message), not one summary. Test via fixture tree with a mix of error + warning findings.
- **AC-3: `--verbose` flag restores full per-instance output.** Running `aiwf check --verbose` against any fixture produces output byte-identical to the current (pre-M-0089) behavior — same lines, same ordering, same hint formatting. The pre-M-0089 output is the golden reference.
- **AC-4: JSON envelope is unchanged.** `aiwf check --format=json` (with or without `--verbose`) produces an envelope whose `findings` array contains every finding individually. Byte-identical to a saved pre-M-0089 baseline.
- **AC-5: `aiwf check --help` documents `--verbose`.** Help text names the flag with one-line description; the Example block shows both default and `--verbose` invocation.
- **AC-6: `cmd/aiwf/completion_drift_test.go` passes.** The new flag is wired through Cobra's flag set with appropriate completion hooks (boolean flag, no completion func needed beyond the default).
- **AC-7: Kernel-tree integration test — default output is short.** Running the binary against this repo's actual planning tree (in the pre-sweep state with the 176 advisories) produces ≤10 lines in default mode. The exact count depends on the number of finding *codes* present, not instances. Binary-level test per CLAUDE.md *Test the seam, not just the layer*.
- **AC-8: Summary lines name the code structurally, not by substring grep.** The per-code summary is generated from the `Finding.Code` field; tests assert on structured parse of the rendered output (split by line, parse the leading code token), not flat substring match per CLAUDE.md *Substring assertions are not structural assertions*.

## Constraints

- **JSON envelope stays byte-identical.** The render-layer change is text-format-only. Any change observable in JSON is a defect.
- **Errors print per-instance in default text.** Summary form applies to warnings only — each error is per-instance-actionable.
- **`--verbose` reproduces the current behavior byte-for-byte.** Anyone who wants the old shape names the flag; the kernel does not silently drop the old shape.
- **Order of summary lines:** sort by count descending (highest-volume offender first), break ties alphabetically by code. Pinned here so test golden files don't drift.
- **Sample message per code:** the first finding's `Message` field, verbatim. No truncation, no rewording, no template substitution beyond what already happens at finding-emission time.
- **No new flag named `--quiet`** in this milestone. Deferred to a downstream gap if/when a CI consumer asks.
- **No change to any check rule.** No new findings, no scope changes, no severity changes.

## Design notes

- Render layer lives in `internal/render/text.go` (or wherever `render.Text` lives — verify the actual path). The summary is computed by grouping findings on `Code`, counting, picking the first message, and emitting one line per group with a `(severity)` annotation that comes from the first finding in the group (all members share severity by code-naming convention).
- The verb body in `cmd/aiwf/check.go` (or wherever the check command lives) passes a `verbose bool` into the renderer. The flag default is false.
- Errors-still-full is implemented by partitioning findings: errors flow through the per-instance path (existing), warnings flow through the summary path (new). No two-pass complexity.
- Sample-message selection: take the **first** finding by render-iteration order. Tests assert on this choice so it's pinned.

## Surfaces touched

- `cmd/aiwf/check.go` (verb wiring, `--verbose` flag)
- `cmd/aiwf/check_cmd_test.go` (tests — name varies by current convention)
- `internal/render/text.go` (the text renderer for findings)
- `internal/render/text_test.go`
- `cmd/aiwf/completion_drift_test.go` (no opt-out needed; flag is bool, auto-completes)

## Out of scope

- `--quiet` mode, severity-filter flags, or per-code suppression. Future gaps if demand surfaces.
- JSON envelope changes.
- Any check-rule changes.
- Re-rendering `aiwf status` (already one-screen by design).

## Dependencies

- None. This is a render-layer change; no upstream milestone needed.
- ADR-0006 (skills policy) — no embedded skill needed for this change. The verb already exists; the flag is discoverable via `aiwf check --help`. Add an entry to `skillCoverageAllowlist` rationale only if `--verbose` introduces a closed-set value that the skill policy would otherwise flag.

## References

- G-0098 — gap this milestone closes (via E-0026).
- E-0024 (done) — surfaced the friction; the 176-advisory pre-sweep state is the worked example.
- `internal/check/check.go::Run` — returns the `[]Finding` the renderer summarizes.
- `internal/render/text.go` — current renderer (verify path).
- CLAUDE.md "CLI conventions" §JSON envelope — the unchanged machine-readable contract.
- CLAUDE.md "Render output must be human-verified before the iteration closes" — applies; the milestone is not done until the binary's default output has been read against the kernel tree and confirmed to read cleanly.
- CLAUDE.md "Substring assertions are not structural assertions" — AC-8 cites this rule directly.

---

## Work log

- Branched milestone/M-0089-… off main at `5523e99`.
- Promoted M-0089 draft → in_progress.
- Allocated 8 ACs verbatim from the spec's "Intended landing zone".
- Captured pre-change goldens by building the binary at `5523e99` (`/tmp/aiwf-baseline-m0089`) and running it against a tempdir copy of `internal/check/testdata/messy` for text + JSON + JSON-pretty. Goldens stored at `cmd/aiwf/testdata/m0089/`.
- TDD pass for AC-1: red test (`TestTextSummary_WarningsCollapsedByCode`) → added `render.TextSummary` partitioning errors per-instance and grouping warnings by Code with count-desc / alphabetic tie-break → green.
- Refactored `render.Text` to share `renderPerInstance` so the verbose path is identical-by-construction to the pre-M-0089 behavior (AC-3).
- Wired `--verbose` boolean flag into `aiwf check`. Boolean flags are auto-skipped by `completion_drift_test.go` per its existing rule, so AC-6 was green without extra wiring.
- Added binary integration tests under `cmd/aiwf/check_summary_binary_test.go`: AC-3 byte-identical against `verbose-text.golden`, AC-4 structural against `json.golden` / `json-pretty.golden` (modulo `metadata.root`), AC-7 kernel-tree ≤10 lines, AC-5 `--help` structurally documents `--verbose`.
- Filled in 100% branch coverage in `internal/render/` via failing-writer tests (`TestTextSummary_WriteErrorBubblesUp`, `TestText_WriteErrorBubblesUp`, `TestRenderPerInstance_WriteErrorPerShape`).
- Human-verified default-mode output against the kernel tree — 6 lines, reads cleanly. Captured under *Validation* below.

## Decisions made during implementation

- **AC-4 strict byte-identity is structurally over-stated.** The pre-M-0089 baseline at `/tmp/m0089-fixture` carries that absolute path in `metadata.root`; any test run resolves the consumer repo to a fresh tempdir and emits a different `root` value. Strict byte-equality across machines is impossible. The chosen reading: byte-identical to baseline modulo `metadata.root`, with the field's presence + non-emptiness still asserted. The compare is structural via `cmp.Diff` on parsed envelopes, which proves every per-finding field is unchanged. The renderer is the stdlib's `encoding/json` so within-binary byte-shape (key order, whitespace, escaping) is deterministic; the JSON contract is intact.
- **`TextSummary` lives in `internal/render/render.go`, not a sibling file.** The spec named `internal/render/text.go` as the renderer's home; the actual renderer is `internal/render/render.go` (no separate `text.go`). The new function lives next to `Text` so callers see one entry point per output shape. Refactored existing `Text` to delegate per-instance lines to a shared `renderPerInstance` helper so the two paths cannot drift (AC-3 byte-identity is enforced by code structure, not just by golden file).
- **Shape-only check path unchanged.** `runCheckShapeOnly` (the pre-commit hook's fast path) still calls `render.Text` per-instance — it has no `--verbose` flag because it only emits tree-discipline findings and a summary would obscure the per-file paths the hook is meant to flag.
- **JSON path remains `render.JSON` regardless of `--verbose`.** AC-4: JSON is the machine-readable contract; `--verbose` is text-only. The text-branch switch on `verbose` happens in `runCheckCmd`'s `case "text"` arm; the `case "json"` arm passes the full `findings` slice straight through.
- **Boolean flag, no completion wiring needed.** `--verbose` is a bool, which `completion_drift_test.go` auto-skips. Cobra renders it without a value argument and tab-completion does not enumerate values, so no `RegisterFlagCompletionFunc` or opt-out entry was needed.

## Validation

Kernel-tree default-mode output (rendered by `aiwf check` against this repo at branch `milestone/M-0089-…`):

```
terminal-entity-not-archived (warning) × 176 — entity ADR-0002 has terminal status "rejected" but file is still in the active tree; awaiting `aiwf archive --apply` sweep
entity-body-empty (warning) × 10 — M-0089/AC-1 body under `### AC-1` is empty
archive-sweep-pending (warning) × 1 — 176 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit skipped

188 findings (0 errors, 188 warnings)
```

6 lines total — well under the AC-7 ≤10 bound. Ordering is count-desc, alphabetic tie-break (`archive-sweep-pending` before `provenance-untrailered-scope-undefined` on the 1-count tier). Each line names the code, severity, instance count, and a representative message. Reads cleanly.

The pre-M-0089 binary at the same SHA emits 179 lines for the same tree (177 of which were near-identical `terminal-entity-not-archived` warnings sharing the same hint). This milestone closes that friction by collapsing the warning leaves into a per-code summary.

Verbose-mode output reproduces the pre-M-0089 shape byte-for-byte against the `messy` fixture; pinned by `TestBinary_CheckVerbose_ByteIdenticalToBaseline`.

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Default text output is per-code-summarized for warnings

`render.TextSummary` partitions findings: errors flow through `renderPerInstance` unchanged, warnings group by `Code` into per-code buckets emitted as `<code> (warning) × N — <sample>`. The sample message is the first finding's `Message` verbatim per the spec's *Constraints*.

Test: `TestTextSummary_WarningsCollapsedByCode` in `internal/render/render_test.go` constructs a 5-finding slice with two distinct warning codes (one ×3, one ×2), asserts exactly two summary lines, parses the leading code token structurally (no flat substring match), and verifies the per-leaf form is *absent* from the output. Binary-level: `TestBinary_CheckDefault_SummarizesWarnings` against the `messy` fixture asserts the full 5-warning-code summary set.

### AC-2 — Errors still print per-instance in default text

The error path in `TextSummary` accumulates errors into a slice that goes through `renderPerInstance` — the same formatting `Text` uses in verbose mode. Errors are never summarized.

Test: `TestTextSummary_ErrorsPrintPerInstance` constructs 3 errors (same code) + 2 warnings (same code), asserts 3 per-instance error lines extracted by the documented `<path>:<line>: error <code>: <msg>` shape and 0 error-severity summary lines. Binary-level: `TestBinary_CheckDefault_SummarizesWarnings` runs the same shape end-to-end through `aiwf check`.

### AC-3 — `--verbose` flag restores full per-instance output

`Text` was refactored to share `renderPerInstance` with `TextSummary`; the verbose branch in `runCheckCmd` selects `Text` instead of `TextSummary`. Per-instance rendering is identical by construction, not just by golden file.

Test: `TestBinary_CheckVerbose_ByteIdenticalToBaseline` builds the post-change binary, runs `aiwf check --verbose --root <tempdir-copy-of-messy>`, and asserts byte-identical stdout to `cmd/aiwf/testdata/m0089/verbose-text.golden`. The golden was captured from the pre-change binary at SHA `5523e99` against the same fixture, so a single-byte drift in the verbose path's output fails the test.

### AC-4 — JSON envelope is unchanged

The text-branch switch on `verbose` lives only inside `case "text":`; the `case "json":` arm is unchanged and the `render.JSON` function is untouched. The full `findings` slice flows into the envelope regardless of `--verbose`.

Test: `TestBinary_CheckJSON_ByteIdenticalToBaseline` runs `aiwf check --format=json` and `aiwf check --format=json --verbose` against the `messy` fixture, parses both stdout and `cmd/aiwf/testdata/m0089/json.golden`, and compares structurally via `cmp.Diff` — modulo `metadata.root` which is the absolute path of the resolved consumer repo (legitimately environmental). `TestBinary_CheckJSONPretty_ByteIdenticalToBaseline` does the same for the pretty branch and additionally pins that 2-space indentation is present. See *Decisions made during implementation* for the structural-modulo-root reasoning.

### AC-5 — `aiwf check --help` documents `--verbose`

The Cobra `Flags()` block on the check command names `--verbose` with description "print one line per warning instance instead of the per-code summary; errors are always per-instance regardless". The Examples block was updated to show three invocations: default (per-code summary), `--verbose` (full per-instance), and `--format=json --pretty` (CI script consumption).

Test: `TestBinary_CheckHelp_DocumentsVerbose` runs `aiwf check --help` (Cobra writes to stderr; captured via `runBinaryCombined`), parses the flags block to locate the `--verbose` row and asserts a non-empty description, then walks the Examples block to confirm both a bare-default invocation and a `--verbose` invocation are present. Flat substring grep on the word "verbose" would not have distinguished a stale description from a fresh one.

### AC-6 — `cmd/aiwf/completion_drift_test.go` passes

`--verbose` is a boolean flag. `TestPolicy_FlagsHaveCompletion` auto-skips boolean flags (see the test's `if f.Value.Type() == "bool" { return }` at the top of the visitor). No `RegisterFlagCompletionFunc` or opt-out entry was needed.

Test: `TestPolicy_FlagsHaveCompletion` runs unchanged and passes.

### AC-7 — Kernel-tree integration test — default output is short

The kernel's planning tree currently has ~3-4 distinct warning codes producing ~180 warnings; the summary collapses to ~5 lines + footer.

Test: `TestBinary_CheckDefault_KernelTreeShortOutput` builds the binary, runs `aiwf check --root <repo-root>` (the kernel's own tree, located via `repoRootForTest`), and asserts ≤10 lines total. The exact line count depends on how many distinct warning codes the kernel tree happens to carry at test time, so the bound is the assertion, not a fixed value. Per CLAUDE.md *Render output must be human-verified before the iteration closes*, the *Validation* section above also pins the rendered output the human read.

### AC-8 — Summary lines name the code structurally, not by substring grep

Both the render-layer tests (`extractSummaryLines` helper in `render_test.go`) and the binary-level tests (`parseSummaryLines` helper in `check_summary_binary_test.go`) use the anchored regex `^(\S+) \((warning|error)\) × (\d+) — (.+)$` to extract the code token from each summary line. The leading `\S+` plus start-of-line anchor means a per-instance line (which begins with `<path>:<line>:`) cannot accidentally satisfy the pattern. The error case in `TestTextSummary_ErrorsPrintPerInstance` confirms this — the same parser scans 3 error lines and finds 0 summaries.

