---
id: D-0005
title: 'AC mechanical-evidence: promote-time --evidence flag binds claim to test symbol'
status: proposed
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0066 (legal-workflows-first-principles.md, §2c AC × milestone composition)
- Audit: R-AUDIT-0195 (legal-workflows-audit-r1.md, §7 CLAUDE.md repo principles — reviewer-discipline)
- Class: FP-only (mechanization elevation) — Pass A captures the rule as reviewer-discipline; CLAUDE.md itself contemplates elevation (*"Discipline is the chokepoint until a kernel finding-rule lands that polices test-existence per AC"*).

## Resolution

Mechanize via promote-time flag (mechanism D from the option-space walk): `aiwf promote M-NNNN/AC-N met --evidence <test-symbol>`.

Mechanism shape:

1. New flag `--evidence <symbol>` on `aiwf promote ... met` (and possibly the `--audit-only` variant). Accepts a Go test function name or `pkg.TestFuncName` form. Repeatable (an AC may have multiple evidence symbols).
2. Kernel projects the flag values into `acs[].evidence: []string` frontmatter field on the AC, in the same one-commit.
3. The promote commit carries `aiwf-evidence: <symbol>` trailer (one trailer line per symbol) alongside the standard trailer set.
4. Two-stage verification:
   - **Write-time**: verb refuses `met` without `--evidence` (hard-reject). Sovereign override available via `--force --reason "..."` for genuinely exceptional cases (e.g., docs-shaped ACs that have no Go test).
   - **Read-time**: `aiwf check` resolves each AC's recorded evidence symbols against `go test -list ./...` output; finding fires if any symbol no longer exists in the test binary.

Rationale:

- The design space has two axes: WHERE evidence lives (frontmatter / body / trailer / test-file naming), and WHEN it's checked (write-time / read-time / never). Mechanism D occupies the WHERE=frontmatter-via-trailer + WHEN=both cells, which is the tightest configuration for the chokepoint.
- Binds the claim to the artifact at the moment the claim is made. No drift later: the operator must name the test when promoting; can't promote-then-forget.
- AI-discoverable: a structured field is machine-readable; tab-completion can list available test symbols from `go test -list`; the trailer is greppable.
- Modest impl scope: one new flag, one new finding code (or two: missing vs stale), one new frontmatter field, one new trailer key.
- Honest about the semantic gap: the kernel verifies *"the named test exists in the binary"*, NOT *"the test asserts the AC's claim."* The latter remains reviewer-only. This is mechanism D's load-bearing limit; alternatives that pretended otherwise (e.g., grep for AC id substring) were rejected as illusion-of-enforcement.

Alternatives considered:

- B (frontmatter field, hand-edited): same final field shape, weaker ergonomics; operator might fill the field after the promote rather than at it.
- C (body section convention): less mechanical (kernel only checks "section present"), same migration cost.
- E (grep for AC id substring in `*_test.go`): zero schema change, noisy chokepoint (false positives from comments, related-but-not-evidence references).
- F (test-function naming convention `TestM0123_AC1_*`): high migration cost (rename existing tests); naming drift is easy.
- A (status quo, reviewer-discipline): rejected because the AC mechanical-evidence rule IS load-bearing per CLAUDE.md, and reviewer-discipline alone doesn't satisfy the kernel principle *"framework correctness must not depend on the LLM's behavior."*

## Spec cell

`internal/workflows/spec` — `Rule{Kind: <AC via composite id>, FromState: entity.StatusOpen, Verb: "promote", Preconditions: [self.evidence non-empty AND evidence-symbols-resolvable], Outcome: Legal, RejectionLayer: VerbTime (no-flag) + CheckTime (stale-symbol), BlockingStrict: true, ExpectedErrorCode: "ac-evidence-missing" (verb-time) | "ac-evidence-stale" (check-time)}` (legal transition target: `entity.StatusMet`).

## Follow-up

Impl change scope-out of M-0123. File a gap → milestone under E-0033 for: `--evidence` flag in `aiwf promote`, frontmatter field on AC, new finding codes, `go test -list` integration, migration verb for backfilling existing met-ACs (~200+ to backfill).
