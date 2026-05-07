---
id: M-064
title: aiwf update migration for existing aiwf.yaml with loud output
status: draft
parent: E-16
tdd: required
acs:
    - id: AC-1
      title: 'aiwf update inserts tdd.default: required when missing'
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf update leaves an existing tdd.default value unchanged
      status: open
      tdd_phase: red
    - id: AC-3
      title: Comments and key order in aiwf.yaml preserved across update
      status: open
      tdd_phase: red
    - id: AC-4
      title: Re-running aiwf update is a no-op (idempotent)
      status: open
      tdd_phase: red
    - id: AC-5
      title: 'Text output includes aiwf.yaml: section listing changes'
      status: open
      tdd_phase: red
    - id: AC-6
      title: JSON envelope mirrors changes in result.changes[]
      status: open
      tdd_phase: red
    - id: AC-7
      title: No-change runs surface tdd.default presence (not silent)
      status: open
      tdd_phase: red
    - id: AC-8
      title: Subprocess integration test covers insert, skip, idempotent
      status: open
      tdd_phase: red
---

## Goal

Existing consumer repos absorb `tdd.default: required` automatically when they next run `aiwf update`, without overwriting any value the human set deliberately. The verb's output makes the change visible in both human-readable text and the `--format=json` envelope, so the operator (or CI) sees the policy shift land at exactly the moment it takes effect — not buried in release notes, not delayed until the next `aiwf add milestone` surprises them with a refusal.

`aiwf upgrade` already calls `aiwf update` as its post-install step, so wiring this through `aiwf update` covers both invocation paths with one implementation.

## Approach

`aiwf update` reads the consumer repo's `aiwf.yaml`, detects whether `tdd.default` is present (any value), and inserts `tdd.default: required` at top level with a comment block when missing. Insertion preserves surrounding comments and key order — use a YAML library that keeps positional context (e.g. `yaml.v3` Node API) rather than the round-trip approach which strips comments. Idempotent: a second run is a no-op (and the no-op is also surfaced loudly so the operator gets confirmation, not silence).

Loud-output shape per the [G-055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) spec — text mode prints a clearly-separated `aiwf.yaml:` section listing each change (key added, value, note); `--format=json` envelope mirrors this in `result.changes[]` with fields `path`, `key`, `value`, `note`. Tests cover missing-key insertion, present-key skip (any value), comment + key-order preservation, no-op idempotency on rerun, and the JSON envelope shape.

## Acceptance criteria

### AC-1 — aiwf update inserts tdd.default: required when missing

`aiwf update` against a consumer repo whose `aiwf.yaml` has no `tdd.default` field rewrites the file to include `tdd.default: required` at top level, with the same comment block as M-063's init seeding. The insertion is the *only* change to the file — no other keys touched, no whitespace stripped outside the inserted block. The rewrite happens in-place via a temp-file-then-rename so a crash mid-write cannot leave a half-written `aiwf.yaml`. Tested with a fixture `aiwf.yaml` that has several other keys; the post-update file is byte-compared against a golden that differs only by the inserted block.

### AC-2 — aiwf update leaves an existing tdd.default value unchanged

`aiwf update` against an `aiwf.yaml` that already has `tdd.default` (any value — `required`, `advisory`, `none`, or even an invalid value the consumer hand-set) leaves the field untouched. The principle is that `aiwf update` migrates *missing* fields to safe defaults; it does not enforce policy by overwriting human choices. An invalid value is M-063's parse-time concern (and `aiwf check` will surface it); `update` does not attempt to "fix" it. Tested against three fixtures: `tdd.default: none`, `tdd.default: advisory`, and `tdd.default: bogus` — all three pass through unchanged.

### AC-3 — Comments and key order in aiwf.yaml preserved across update

The update path uses a YAML library that preserves positional context (the `yaml.v3` Node API or equivalent) — round-tripping the file through a struct-based decode/encode would strip comments and reorder keys, which is unacceptable. Verified by a fixture with leading file comment, comment between top-level keys, trailing whitespace, and a non-alphabetical key order; the post-update file's only diff is the inserted `tdd:` block at the chosen insertion point. Comment placement around the inserted block follows the same convention as the init template (M-063 AC-4).

### AC-4 — Re-running aiwf update is a no-op (idempotent)

After a first `aiwf update` that inserted `tdd.default: required`, a second `aiwf update` against the same repo produces no file changes (the file mtime may update; the byte content does not). The verb's exit code is 0; the output indicates "already present" per AC-7. Tested by running update twice in the same tempdir and asserting the post-second-run file bytes match the post-first-run file bytes.

### AC-5 — Text output includes aiwf.yaml: section listing changes

When `aiwf update` modifies `aiwf.yaml`, its human-readable text output includes a clearly-separated section, e.g.:

```
aiwf.yaml:
  + tdd.default: required   (new in vX.Y.Z)
      New milestones now require an explicit --tdd value. The project default
      applies when --tdd is omitted. Override repo-wide by editing this field;
      opt out per-milestone with `aiwf add milestone --tdd none ...`.
```

Format pieces: section header (`aiwf.yaml:`); change-line prefix (`+` for added, future `~` for changed); key path; resolved value; parenthetical version note; indented multi-line note explaining the override paths. The exact format is asserted byte-exact by the integration test (AC-8) so the surface stays stable for human readers.

### AC-6 — JSON envelope mirrors changes in result.changes[]

`aiwf update --format=json` emits the standard envelope (`tool`, `version`, `status`, `findings`, `result`, `metadata`) where `result.changes` is an array of structured change records, one per modified file × modified key. Each entry has `path` (the relative path of the modified file), `key` (the dotted YAML path, e.g. `tdd.default`), `value` (the resolved value as a string), `op` (`add` for new keys; future `update` for changed values), and `note` (the same multi-line text from AC-5). The shape is asserted by parsing the envelope back into a typed struct in the integration test — substring matching on the JSON would not catch a wrong-section regression (per CLAUDE.md "substring assertions are not structural assertions").

### AC-7 — No-change runs surface tdd.default presence (not silent)

When `aiwf update` runs against an `aiwf.yaml` that already has `tdd.default`, the output includes a one-line acknowledgement, e.g. `aiwf.yaml: tdd.default already set (required)`. The line is a deliberate counter to the "silent success" failure mode — an operator re-running update after a manual edit gets confirmation that the value they set is the value the verb sees. In `--format=json`, the `result.changes[]` array is empty but `result.config` (or equivalent) carries the resolved `tdd.default` value so a CI script can audit the active policy without parsing prose.

### AC-8 — Subprocess integration test covers insert, skip, idempotent

A binary-level test (`go build -o $TMP/aiwf ./cmd/aiwf` then `exec.Command(...)` per CLAUDE.md "test the seam") in `cmd/aiwf/binary_integration_test.go` exercises three cases against fresh tempdirs:

1. **Insert** — start with `aiwf.yaml` lacking `tdd.default`; run `aiwf update`; assert exit 0, file diff is exactly the inserted block, text output contains the section from AC-5, JSON envelope (in a separate run) contains the `result.changes[]` entry from AC-6.
2. **Skip** — start with `aiwf.yaml` containing `tdd.default: none`; run `aiwf update`; assert file unchanged, output contains the AC-7 confirmation line.
3. **Idempotent** — run `aiwf update` twice in case 1's tempdir; assert second-run file bytes match first-run bytes; second-run output contains the AC-7 confirmation.

The fixtures live under `cmd/aiwf/testdata/update_tdd_default/` (or the existing test-data layout). No mocks; the test invokes the real binary against real files.

