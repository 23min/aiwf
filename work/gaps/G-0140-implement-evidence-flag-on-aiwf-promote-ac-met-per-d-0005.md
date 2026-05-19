---
id: G-0140
title: Implement --evidence flag on aiwf promote AC met per D-0005
status: open
discovered_in: M-0123
---
## What's missing

Per **D-0005** (committed in M-0123 phase 1), `aiwf promote <id>/AC-N met`
under a milestone with `tdd: required` should require a `--evidence
<test-symbol>` flag binding the AC's "met" claim to a concrete test that
exercises the AC's assertion. The verb today accepts `aiwf promote
M-NNNN/AC-N met` with no evidence binding.

The spec's `acRules()` in `internal/workflows/spec/rules.go` encodes:

- a **legal** cell (preconditioned on `self.evidence` non-empty),
- an **illegal** companion cell (precondition `self.evidence == ""`,
  ExpectedErrorCode `ac-evidence-missing`).

`ac-evidence-missing` is listed in `deferredImplErrorCodes` (M-0123/AC-5)
with this gap as the tracking reason.

## Why it matters

CLAUDE.md's "AC promotion requires mechanical evidence" rule is currently
operator-discipline only. Promoting an AC to met without a test symbol is
the failure mode that "framework correctness must not depend on the LLM's
behavior" forbids. The kernel chokepoint is the verb; the verb needs the
flag.

## Proposed fix shape

- Add `--evidence <symbol>` flag to `aiwf promote` (visible only on the
  `<id>/AC-N met` shape; reject for other promote targets).
- Write evidence into the AC's `evidence:` frontmatter field (new field,
  schema additive).
- Verb refuses with `ac-evidence-missing` when:
  * milestone is `tdd: required`,
  * promote target is `met`,
  * `--evidence` is empty.
- Test surface: integration test against the binary with a fixture
  milestone that has a passing test symbol; verb succeeds. Same fixture
  without `--evidence`; verb refuses with the structured code.
- Once landed, remove `ac-evidence-missing` from `deferredImplErrorCodes`.

## Open questions

- Does `--evidence` validate the symbol exists in the test tree at
  promote-time, or only record it (and validate via a check rule)?
  Validation at promote-time is stronger but couples the verb to the
  language toolchain. Recording + check separation matches the kernel's
  "verb does one thing, check polices" pattern. Lean: record + check.
