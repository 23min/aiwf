---
id: M-0089
title: Per-code text-render summary with --verbose fallback
status: in_progress
parent: E-0026
tdd: required
acs:
    - id: AC-1
      title: Default text output is per-code-summarized for warnings
      status: open
      tdd_phase: red
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

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Default text output is per-code-summarized for warnings

