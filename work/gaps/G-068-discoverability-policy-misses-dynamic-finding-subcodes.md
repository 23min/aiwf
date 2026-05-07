---
id: G-068
title: Discoverability policy misses dynamic finding subcodes
status: open
discovered_in: M-066
---

## What's missing

`PolicyFindingCodesAreDiscoverable` (`internal/policies/discoverability.go`) substring-matches finding codes against the AI-discoverable channel set. The set of "must be documented" codes is collected in two ways:

1. Named string constants in `internal/check/` whose values look like finding codes (`loadCheckCodeConstants` + `looksLikeFindingCode`).
2. Inline literals in `Finding{}` composite literals — `Code: "..."` and the paired `Subcode: "..."` when both are string-literal expressions (`loadCheckCodeLiterals` via `stringFieldValue`).

Step 2's AST walker only sees **literal** `Subcode` values. When a rule emits `Subcode: someExpression()` — most directly `Subcode: string(e.Kind)`, as `internal/check/entity_body.go:133` does for the per-kind subcodes — the expression evaluates only at runtime; the AST walker sees an `*ast.CallExpr` (or `*ast.Ident`), not an `*ast.BasicLit`. `stringFieldValue` returns `""` for non-literals, the loop's `if sub != ""` guard skips it, and the composite `<code>/<subcode>` never enters the required-discoverable set.

Concretely for M-066: the rule emits seven distinct subcodes — `entity-body-empty/{epic,milestone,ac,gap,adr,decision,contract}`. Only `/ac` is a literal in source. The other six are derived via `string(e.Kind)` and bypass the policy entirely. Today, all seven appear in `internal/skills/embedded/aiwf-check/SKILL.md` because M-066/AC-1 spelled them out for operator clarity — not because the policy compelled it. A future kind addition (or a new rule that uses dynamic subcodes) would silently ship undocumented.

## Why it matters

The whole point of `PolicyFindingCodesAreDiscoverable` is the kernel principle "kernel functionality must be AI-discoverable" — a code that an AI assistant cannot reach via `--help`, embedded skills, `CLAUDE.md`, or `docs/pocv3/**` is, by definition, undocumented. The policy was added (G-021) to make that property mechanical rather than reviewer-dependent. A blind spot for dynamically-derived subcodes means a class of new emissions can ship past the chokepoint silently — exactly the failure mode the policy exists to catch.

Two complementary fixes worth considering:

- **Static path**: enumerate the closed-set values that production code feeds into the dynamic expression, and synthesize the composite codes from them. For `Subcode: string(e.Kind)` specifically, the per-kind values come from the closed set of `entity.Kind` constants, all six of which are already known to the policy package. The AST walker could detect the expression shape (`Subcode: string(<some-Ident>)` where `<some-Ident>` resolves to a `Kind`-typed field) and expand the required set to `code/<kind>` for every Kind. Targeted; works only for the `string(e.Kind)` shape; cheap.
- **Convention path**: ban dynamic Subcode expressions and require every emission to be a literal. Per-kind dispatch loops would have to enumerate explicitly (one `Finding{Subcode: "epic", ...}` branch per Kind) — boilerplate-heavy in the rule code but trivially discoverable. Reverses the current ergonomic trade-off.

Either fix needs a follow-up `PolicyForbidsDynamicSubcodes` (or equivalent) test to make the rule mechanical going forward.

Discovered during M-066/AC-6's RED-first sanity check: deleting the SKILL.md row produced exactly one violation (`entity-body-empty/ac`), surfacing the asymmetry. Documented as a follow-up rather than scoped into M-066 because the fix is a kernel-discipline concern in `internal/policies/`, not in the M-066 rule itself.
