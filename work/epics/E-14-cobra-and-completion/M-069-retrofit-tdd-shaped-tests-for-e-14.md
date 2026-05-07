---
id: M-069
title: Retrofit TDD-shaped tests for E-14
status: draft
parent: E-14
tdd: required
acs:
    - id: AC-1
      title: Envelope conforms to documented schema for every --format=json verb
      status: open
      tdd_phase: done
    - id: AC-2
      title: Single-commit-per-verb invariant asserted per mutating verb
      status: open
      tdd_phase: done
    - id: AC-3
      title: Trailer-key shape asserted per mutating verb
      status: open
      tdd_phase: red
    - id: AC-4
      title: Pre-push hook byte-golden plus template-equals-installed cross-check
      status: open
      tdd_phase: red
    - id: AC-5
      title: init then doctor --self-check seam in a fresh tempdir repo
      status: open
      tdd_phase: red
    - id: AC-6
      title: Native-Cobra drift test fails CI on passthrough-adapter regression
      status: open
      tdd_phase: red
    - id: AC-7
      title: Help-quality drift asserts Example present and no migration prose
      status: open
      tdd_phase: red
---
## Goal

Retrofit the load-bearing tests E-14 (Cobra and completion) shipped without. The audit closed via G-055 found that several E-14 milestones were promoted `done` while the AC text referenced behavior that no test exercises. This milestone is `tdd: required`; each AC walks `red` → `green` → `done` so the gap is closed by mechanical evidence, not narrative.

The seven ACs each pin one gap class identified in the audit. Production code is, in the audit's premise, already correct — what's missing is the test that would have failed before E-14 shipped and would fail again on regression. Where the production path turns out *not* to satisfy the test, that's a real second-order finding and gets fixed inline.

## Acceptance criteria

### AC-1 — Envelope conforms to documented schema for every --format=json verb

`internal/render/render.go` documents the JSON envelope contract: every `--format=json` invocation emits a single object with `tool` (always `"aiwf"`), `version` (non-empty string), `status` (one of `"ok"` / `"findings"` / `"error"`), `findings` (array, never null/missing — empty when none), optional `result` (verb-specific payload), and optional `metadata`. The contract is the load-bearing thing CI scripts and downstream tools key off — `findings` is grepped the same way across verbs, `result` is switched on by verb name.

The existing per-verb tests check the envelope loosely. `TestRun_ShowJSONEnvelope` asserts `tool == "aiwf"` and `status == "ok"` for one show invocation; nothing pins the contract across the *full* set of verbs that emit it. A regression where, say, `aiwf status --format=json` started omitting the `findings` array, or `aiwf contract verify` returned a status string outside the closed set, would not be caught by any current test — `findings` is the field downstream consumers grep, so its disappearance is a silent breaking change.

This AC adds a structural conformance test that exercises every verb supporting `--format=json`, parses the envelope, and asserts the schema:

- top-level keys are exactly the documented set (`tool`, `version`, `status`, `findings`, optionally `result`, optionally `metadata`); unknown keys fail the test.
- `tool` equals `"aiwf"` exactly.
- `version` is a non-empty string.
- `status` is one of the three closed-set values.
- `findings` is a JSON array (decodes into `[]any`, may be empty, must not be `null` or missing).
- `result` and `metadata`, when present, are JSON objects.

The matching uses `go-cmp.Diff` against a canonical schema-shaped value with `cmpopts.IgnoreFields` (and a comparer for the `findings` array contents) so the test pins **structure**, not specific run-varying values like the build-info version string or metadata timing fields. The verb table is the source of truth: a new `--format=json` verb that lands without an entry is the regression we want this test to catch on the next CI run.

The test drives the verbs through `run([]string{...})` (the same dispatcher production uses) and captures stdout, so it sits at the seam between the verb's flag-binding and `render.JSON` — a verb that constructed its envelope manually, or skipped `render.JSON` and emitted ad-hoc JSON, would fail the conformance assertion even if its godoc still claimed the contract.

The verbs covered: `check`, `show`, `history`, `status`, `schema`, `template`, `contract verify`, and `render --format=html` (which emits the envelope on stdout while writing HTML to disk).

### AC-2 — Single-commit-per-verb invariant asserted per mutating verb

CLAUDE.md design decision §7: "Every mutating verb produces exactly one git commit. That gives per-mutation atomicity for free. A failed mutation aborts before the commit." This is one of the load-bearing properties any change must preserve — together with stable ids, layered location-of-truth, and `aiwf check` as the chokepoint — and the audit closed via G-051 ("Planning sessions emit one commit per entity, not per logical mutation") was the user-visible symptom of an earlier era when this invariant was not enforced.

`TestBinary_MutatingVerbs_Subprocess` already runs every migrated mutating verb as a subprocess sequence and asserts that each invocation exits cleanly. It does *not* assert the commit count delta per verb. A regression where `aiwf promote` started emitting two commits (one for the entity update, one for a side-effect projection) — or where `aiwf cancel` emitted zero commits and stamped its mutation as part of the *next* verb's commit — would still pass that test. The kernel's atomicity guarantee, the property `aiwf history` projects against, and the per-mutation rollback story all depend on this delta being exactly 1.

This AC adds a sequence test that drives each mutating verb through the in-process dispatcher (`run([]string{...})`), records `git rev-list --count HEAD` before and after the verb, and asserts the delta is exactly 1. The sequence mirrors a typical consumer-repo lifecycle and is exhaustive over the user-facing mutating verb surface: `add` (each kind), `promote` (entity status, AC status, AC tdd_phase), `rename`, `edit-body`, `move`, `cancel`, `authorize` (open / pause / resume), `import` (default bundled-commit mode — multi-entity manifest must still be one commit), `reallocate`, and the `contract` family (`recipe install`, `bind`, `unbind`, `recipe remove`).

The "default-mode `import`" subcase is the audit's namesake: a manifest with N entities must produce exactly one commit, not N. The test seeds a multi-entity manifest specifically to pin that.

The assertion is `delta == 1` (strict equality), not `delta >= 1`. Strict equality catches both ends of the regression: a verb that silently produces a *second* commit (e.g. an audit-trail commit), and a verb that emits *zero* commits (the "deferred to next verb" smell). The verb table is, again, the source of truth — adding a new mutating verb without a row here is the regression this test surfaces.

### AC-3 — Trailer-key shape asserted per mutating verb

### AC-4 — Pre-push hook byte-golden plus template-equals-installed cross-check

### AC-5 — init then doctor --self-check seam in a fresh tempdir repo

### AC-6 — Native-Cobra drift test fails CI on passthrough-adapter regression

### AC-7 — Help-quality drift asserts Example present and no migration prose
