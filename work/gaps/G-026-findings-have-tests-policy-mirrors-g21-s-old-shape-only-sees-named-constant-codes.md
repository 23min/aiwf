---
id: G-026
title: '`findings_have_tests` policy mirrors G21''s old shape — only sees named-constant codes'
status: addressed
---

Resolved in commit `f37dc07` (feat(aiwf): G26 — extend findings_have_tests to inline-literal codes). After G21 broadened `PolicyFindingCodesAreDiscoverable` to enumerate every kebab-case finding code (named constants + inline `Code: "..."` literals across `check/` and `contractcheck/`), `PolicyFindingCodesHaveTests` was left on the old narrow enumeration: it only verified test references for named-constant codes (i.e., the `provenance-*` family). Inline-literal codes — most of the pre-I2.5 surface, including `acs-tdd-audit`, `acs-shape`, `case-paths`, `load-error`, etc. — could be production-emitted without any test asserting the exact code string. A typo in the emission site would slip through every existing test.

The fix shares `loadCheckCodeLiterals` with the discoverability policy so the two now operate on the same code population. For inline-literal codes the only acceptable test reference is the quoted string value (no constant name to fall back on). Codes also declared as constants are deduped against the named pass.

The broadened policy immediately surfaced one real violation: `acs-tdd-audit` was emitted in `check/acs.go` and exercised by three tests in `acs_test.go` that asserted severity and entity-id but never the code string. Two of those tests were tightened to also assert `Code == "acs-tdd-audit"` — a typo at the emission site would now fail them.

Severity: Low. The policy gap was structural (test-coverage rule didn't match the docs-coverage rule's scope); the one real violation it found was a single under-tested code, not a correctness regression. But the symmetry is the point: G21 and G26 together now mean every kernel finding code is *both* documented in at least one channel *and* asserted by string in at least one test — the kind of pair where letting one half drift while the other tightens is exactly how subtle holes open up.

---

<a id="g24"></a>
