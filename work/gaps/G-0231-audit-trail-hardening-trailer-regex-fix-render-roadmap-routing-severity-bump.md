---
id: G-0231
title: 'Audit-trail hardening: trailer-regex fix, render-roadmap routing, severity bump'
status: open
---
## What's missing

Four distinct softness points in the audit-trail chokepoint stack, all surfaced by the E3 adversarial pass:

1. **`PolicyTrailerKeysViaConstants` regex is structurally broken.** `internal/policies/trailer_keys.go`'s regex `"([^"\\]*)"` produces zero violations against the live repo despite `internal/cli/render/render.go:277-278` containing literal `"aiwf-verb"` / `"aiwf-actor"`. Fix the regex (escape handling); add a positive-control test case (a known violation in `testdata/` that the policy must catch) so the next regex regression is caught immediately.
2. **`render-roadmap` bypasses `verb.Apply`.** `internal/cli/render/render.go:267-283` invokes `gitops.Commit` directly with hand-built trailers — escapes the verb-validate-then-write chokepoint, escapes the rollback envelope, and is the only literal-trailer-string site in production code (which is why the broken regex matters). Route through `verb.Apply` with a proper `Plan`; let the kernel's existing trailer-emission path handle the trailer construction.
3. **`CodeProvenanceUntrailedEntityCommit` severity bump.** Today the finding emits at `SeverityWarning`; pre-push exits 0 even when manual entity edits land untrailered. Bump to `SeverityError` once the historical fabricated-trailer cleanup is complete. Pair with a one-shot audit on the trunk window naming the commits that need addressing first (the bump is gated on the audit being clean).
4. **Trailer-shape test coverage expansion.** `internal/cli/integration/trailer_shape_test.go` doesn't cover `render-roadmap`, `archive`, `acknowledge-illegal`, `audit-only`, `rewidth`, `milestone-depends-on`. Parameterize the existing test or add per-verb cases.

## Why it matters

E3's adversarial verdict ("the Strong holds but several chokepoint claims are softer than first-review framed them") is the canonical sign that the chokepoints exist but aren't tight. A broken regex producing zero violations is silently worse than no regex at all — it shows up as a green CI signal. The four items above tighten the chokepoint without changing the architecture.

## Related

G-0218 ("Operator-typed commit messages bypass aiwf-verb registry at composition") is adjacent but doesn't cover the trailer-regex bug or the render-roadmap bypass. Land G-0218 alongside this if you want all "audit-trail edges harden together"; otherwise sequence them however.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §E3 (recommended moves 1–3; refuting evidence list — items 2, 3, 4, 5).
