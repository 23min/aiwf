---
id: G-0474
title: No chokepoint enforces the ban on blank-identifier unused-silencers
status: open
priority: low
---
## What's missing

CLAUDE.md's Go conventions ban one pattern by name: a blank-identifier alias kept
solely to stop `unused` or `staticcheck` flagging reserved-for-future code. The
rule's reasoning is that such an alias defeats the very check meant to find that
code; its remedy is to delete the symbol and re-add it with its first real caller.

That rule arrived by decision, not by accident — G-0451 asked for it, naming the
`var _ = fn` form explicitly as a write-time force. G-0449 then removed two such
guards, arguing that they "exist *to suppress* the signal" and that "only a
reachability sweep" finds what they hide. What neither produced was a detector.

The rule is unenforced, and the tree carries a live instance:
`internal/verb/rewidth_test.go:738` reads `var _ = rewriteRewritesToPaths`, under
the comment "rewriteRewritesToPaths is exported for cross-test reuse if needed."
The helper has no caller, and it is unexported — the comment is wrong on both
counts.

This instance is not a recurrence. It was introduced 2026-05-10, months before
G-0449's sweep, which touched only `resolver.go`, `entity.go` and their tests. It
survived because it lives in a `_test.go` file and the sweep looked at production
code. That is the sharper argument for a detector: the one prior cleanup was
scoped by hand, and hand-scoping is what missed this.

The alias is why no gate caught it. `unused` sees a reference and stops. Only
whole-program reachability with tests as roots looks past the alias to ask whether
anything reachable calls the function — `deadcode -test ./...` reports it, and
`deadcode` is neither in the linter set nor run by any gate, nor provisioned by the
devcontainer.

## Why it matters

This is a rule the repo states as a rule, in the file it treats as authoritative
for Go conventions, having filed a gap to put it there. The repo also holds that a
guarantee depending on someone remembering is not a guarantee. The pattern is
mechanically detectable, so the distance between the stated rule and the enforced
rule is closable rather than inherent.

The consequence of leaving it is narrow but self-concealing, which is the part
worth fixing. A silenced symbol looks maintained: it compiles, it carries a doc
comment, and the linter is quiet. The one live instance carries a justification
that reads plausibly and is factually wrong twice over, which is what that class of
comment looks like once the code it describes has moved on.

## Options

1. **An AST policy in `internal/policies/`** flagging a package-level
   `var _ = <ident>` whose right-hand side is a bare identifier resolving to a
   function or type declared in the same package. Matches how comparable
   conventions here are enforced, runs in the existing suite, and needs a firing
   fixture to avoid being vacuous. The bare-identifier restriction is
   load-bearing: `var _ = entity{Priority: "high"}`
   (`internal/policies/closed_set_status_constants_test.go:102`) is a deliberate
   policy fixture and must not fire, and typed interface assertions of the form
   `var _ Iface = (*impl)(nil)` must not either. Measured tree-wide, a regex for
   the bare form matches exactly one site — the true positive — with no false
   positives.
2. **Add `deadcode` to the gate.** Catches this class and genuine orphans together.
   The triage cost is smaller than it sounds: `deadcode -test ./...` returns three
   hits tree-wide today, and the two that are not this instance are both owned by
   open G-0417 and accepted D-0018 and already annotated as known-dead. So the
   ownership pass is a one-time three-line exercise, not an ongoing tax. It finds
   strictly more than option 1 — including the G-0417 surface, which option 1
   structurally cannot see.
3. **Delete the one instance and leave the rule unenforced.** Honest about cost and
   it restores the tree, but it leaves a documented rule with no detector, which is
   the condition this gap is about rather than a fix for it.

Option 1 is the lean: it is the enforcement shape this repo already uses, it is
narrow enough to be precise, and it closes the rule-versus-detector gap rather than
only the instance. Option 2 is a genuine companion rather than a rival — its cost
is now measured and low, and it reaches what option 1 cannot; taking both is
defensible.

Either way the live instance goes: no caller, no open issue naming it, no decision
retaining it.

One thing an implementer will hit: ADR-0019 ships the code-health rubric as an
*advisory* skill, explicitly "not a `check` rule, not a hook." That governs the
rubric's own surface, not this convention — an `internal/policies/` test is
kernel-internal and adds no consumer surface — but the ADR reads like a blocker
until that distinction is made.

## Related

- G-0451 — asked for this rule, naming the `var _ = fn` form
- G-0449 — removed two instances by hand; its scoping is what missed this one
- G-0465, G-0471 — the same rule-without-a-detector shape, on other rules
- G-0417, D-0018 — own the two other `deadcode` hits, so option 2's triage is bounded
- ADR-0019 — the advisory-rubric decision to distinguish, not to override

## Scope

Surfaced by a `wf-structural-sweep` pass after E-0073 wrapped.
