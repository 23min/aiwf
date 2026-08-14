---
id: G-0584
title: Ritual skill tests assert section-scoped literals that cannot fail
status: open
discovered_in: M-0308
---
## What's missing

The `wf-*` ritual skills are backstopped by policy tests that scope to a named
markdown section and then match a literal read out of that section. Verified in
full for one file: `TestWfStructuralSweep_HasFourNamedLenses` asserts the lenses
section contains `"Dead paths"`, `"Textual clones"`, `"Convergent duplication"`
and `"Data flow"`, and `TestWfStructuralSweep_DataFlowLensNamesProducedButUnconsumed`
asserts `### Lens 4` contains `"consume"` and `"unconsumed"`.

The scoping is structural and does its job — it proves the literal sits in the
right place rather than anywhere in the file, which is what CLAUDE.md
§"Substring assertions are not structural assertions" asks for. The matching is
not. Every expected value was chosen by reading the prose, so the assertion
passes because someone typed the word. Gut a lens to a heading with an empty
body and both the heading count and the substring checks still pass. Rename the
concept anywhere else in the repo and nothing goes red.

Measured 2026-08-14 across the ritual policy tests, counting call sites rather
than judging each one:

| file | `strings.Contains` | section-scoped |
|---|---|---|
| `wf_ritual_honesty_test.go` | 16 | 0 |
| `wf_patch_changelog_test.go` | 10 | 4 |
| `wf_structural_sweep_test.go` | 10 | 5 |
| `wf_codebase_health_economy_test.go` | 7 | 4 |
| `wf_patch_reconcile_test.go` | 7 | 4 |
| `wf_rethink_wfpatch_xref_test.go` | 2 | 3 |

`wf_ritual_honesty_test.go` is the sharpest: sixteen substring assertions with
no section scoping at all, so each proves only that a string exists somewhere in
the file. The counts are measured; whether every individual call site is vacuous
is verified only for `wf_structural_sweep_test.go`, and the rest are candidates
rather than findings until read.

## Why it matters

These tests are the backstop the repo's own policy demands for every `SKILL.md`
edit, and the rituals they guard are the procedures an assistant follows to
plan, implement and wrap work. A backstop that cannot fail reports health it
does not have, and it is the artifact a future author will copy — the shape
spreads by being the nearest example.

Distinct from G-0540, which names the same class in three specific policies
produced by the gate-scope repairs. These are a different instance set in
different files, and G-0540's resolution does not reach them.

Part of why the shape spread is that no independent source was available to
derive an expected value from: there is no verb-name enumeration in the tree,
and `worktree` is a CLI command under `internal/cli/worktree/` rather than an
entry in `internal/verb/`. The cobra command tree, walked from the root, is one
source that does exist — a ritual naming a command can be asserted to name one
that resolves, so a rename on either side goes red. M-0308 builds that
derivation for its own criteria and is the first instance of it.
