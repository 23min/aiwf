---
id: G-0208
title: aiwf-force amend override has no operator UX path
status: addressed
discovered_in: M-0158
addressed_by:
    - M-0159
---
M-0106/AC-8 pins that a violating commit amended with
`aiwf-force: <reason>` trailer + `aiwf-actor: human/...` trailer
suppresses the `isolation-escape` finding. The test
[`TestIsolationEscape_AC8_ForceAmendedCommitSilent`](../../internal/check/isolation_escape_test.go)
constructs such a commit in a fixture and asserts silent.

The test pins **behavior**. It does NOT pin that operators can
achieve this trailer state through any documented kernel verb or
ritual.

## Reality check

Today, to amend an AI commit with the right trailers an operator
must:

1. Identify the violating commit (`aiwf check` output → SHA).
2. Run `git commit --amend --no-edit --trailer 'aiwf-force: <reason>' --trailer 'aiwf-actor: human/<id>'`.
3. The actor change is structural — the original `aiwf-actor: ai/...`
   trailer must be removed AND `aiwf-actor: human/...` added.
   `git commit --trailer` doesn't support trailer-removal by default.
   The operator has to either: (a) drop into `git commit --amend -e`
   and hand-edit, or (b) use `git interpret-trailers --trim-empty`
   with `--unfold`, or (c) write a custom script.

This is:
- **Not discoverable.** No aiwf verb names this path; no skill
  documents the trailer sequence.
- **Error-prone.** Manual trailer editing is exactly the
  failure mode that produces malformed commits the kernel
  refuses elsewhere.
- **Inconsistent with the `aiwf-force` pattern elsewhere.** Other
  `aiwf-force` uses (`aiwf promote --force`, `aiwf authorize --force`)
  are verb flags that the kernel emits correctly. Only the M-0106
  amend path requires manual trailer editing.

## What's needed

Either:

1. **A new verb** like `aiwf override <sha> --reason "..."` that
   amends the named commit with the correct trailer set, atomically.
2. **A ritual** (`wf-override-isolation-escape` or similar) that
   walks the operator through the trailer-amend dance.
3. **Document the limitation** at minimum, with the exact
   `git commit --amend` invocation in M-0106's hint text or
   epic body.

Option 3 is the smallest fix; options 1+2 are more correct.

## Why parked

The M-0158 honest-scope audit surfaced this. AC-8 of M-0106 ships
"met" by virtue of the test fixture; the operator-facing UX path
does not exist. Address as part of the real-world hardening
milestone.
