---
id: G-0129
title: Typed finding-code constants mechanically enforced at comparison sites
status: addressed
discovered_in: M-0123
addressed_by_commit:
    - 56ad4b84
---
## Problem

Finding codes emitted by `internal/check/` are inconsistently typed. The `CodeProvenance*` family in [`internal/check/provenance.go:28-55`](../../../internal/check/provenance.go) (13 codes) carries typed string constants — `CodeProvenanceTrailerIncoherent = "provenance-trailer-incoherent"` etc. — so the compiler catches removal of a referenced code name. The remaining ~32 finding codes scattered across [`internal/check/check.go`](../../../internal/check/check.go), [`acs.go`](../../../internal/check/acs.go), [`archive_rules.go`](../../../internal/check/archive_rules.go), [`entity_body.go`](../../../internal/check/entity_body.go), [`epic_active_drafts.go`](../../../internal/check/epic_active_drafts.go), and [`entity_id_narrow_width.go`](../../../internal/check/entity_id_narrow_width.go) are bare string literals (`"acs-shape"`, `"refs-resolve"`, `"frontmatter-shape"`, `"status-valid"`, …) — no typed constant, no compiler closure when a code is renamed or retired.

This is the same shape G-0126 closed for status / kind / phase values, applied to the finding-code surface.

## Why it matters

Three concrete downsides today:

1. **No compiler closure on rename.** Renaming `"acs-shape"` requires grepping every emit site, every test, and every downstream consumer (docs, skills, CI assertions). The compiler doesn't help.
2. **Emit-vs-test drift is silent.** Test sites like [`internal/check/acs_test.go:279`](../../../internal/check/acs_test.go), `:300`, and [`archive_rules.go:156`](../../../internal/check/archive_rules.go) compare against bare-string codes. Renaming the emit site without updating the test (or vice versa) compiles cleanly and only surfaces at test time — when it surfaces at all.
3. **M-0123's spec→impl drift policy is asymmetric without it.** The workflows-spec table's `ExpectedErrorCode` column closes via the compiler for the ~13 typed codes and via a runtime test for the ~32 bare-string codes. Typed-ifying the remaining codes lets every cell inherit the same compile-time closure as the `Kind` / `FromState` columns.

Per the kernel rule *"framework correctness must not depend on LLM behavior,"* this is exactly the kind of mechanical hygiene that should be a compiler chokepoint, not reviewer attention.

## Code references

- [`internal/check/provenance.go:28-55`](../../../internal/check/provenance.go) — precedent: typed `Code*` constants for the provenance finding family.
- [`internal/check/check.go`](../../../internal/check/check.go), [`acs.go`](../../../internal/check/acs.go), [`archive_rules.go`](../../../internal/check/archive_rules.go), [`entity_body.go`](../../../internal/check/entity_body.go), [`epic_active_drafts.go`](../../../internal/check/epic_active_drafts.go), [`entity_id_narrow_width.go`](../../../internal/check/entity_id_narrow_width.go) — the ~32 bare-string emit sites.
- [`internal/policies/enum_literal_adoption.go`](../../../internal/policies/enum_literal_adoption.go) — G-0126's policy; comment at `:96-97` explicitly anticipates this expansion: *"the seed denylist per the M-0119 spec; expansion to `Kind*`, `Phase*`, etc. is a deliberate future-gap call."*
- [G-0126](./G-0126-typed-status-kind-phase-enums-mechanically-enforced-at-comparison-sites.md) — the status-enum precedent.
- [M-0123](../epics/E-0033-pin-legal-kernel-verb-workflows-mechanically/M-0123-pass-c-reconcile-to-canonical-go-spec-table-drift-policy.md) — the spec→impl drift policy whose asymmetry this gap closes.

## Target shape sketch (not prescription)

1. Declare typed constants for every finding code in the package where it's emitted. Match the `provenance.go` pattern — a `const ( CodeXxx = "xxx" ... )` block at the top of each file or a shared `codes.go`. The grouping is a judgment call left to the implementing milestone.
2. Replace bare-string `Code:` emit sites with the constant.
3. Replace bare-string `Code ==` / `Code !=` / `switch Code` sites in tests with the constant.
4. Extend [`internal/policies/enum_literal_adoption.go`](../../../internal/policies/enum_literal_adoption.go) to also enumerate `Code*` constants from `internal/check/*.go` and require adoption at comparison sites (including test files — the emit-vs-test drift is one of the main wins). The existing AST walk is the right shape; broaden the constant-source enumeration to a small set of (package, name-prefix) pairs rather than the hardcoded `internal/entity/entity.go` + `Status*` pair.
5. Optionally: enforce that finding codes never appear as bare-string `*ast.BasicLit` at `Code:` keyed-field-value sites in struct literals, not just at comparison sites. That's a tighter chokepoint — it would have caught the original drift at emit time. Whether it's worth the AST cost is an implementing-milestone call.
