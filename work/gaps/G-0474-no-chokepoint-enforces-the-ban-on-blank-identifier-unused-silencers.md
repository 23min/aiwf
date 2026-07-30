---
id: G-0474
title: No chokepoint enforces the ban on blank-identifier unused-silencers
status: open
---
## What's missing

CLAUDE.md bans one pattern by name in its Go conventions: a blank-identifier alias
kept solely to stop `unused` or `staticcheck` flagging reserved-for-future code.
The rule's reasoning is that such an alias defeats the very check meant to find
that code, and its remedy is to delete the symbol and re-add it with its first real
caller.

No chokepoint enforces it, and the tree carries a live instance:
`internal/verb/rewidth_test.go:738` reads `var _ = rewriteRewritesToPaths`, under
the comment "rewriteRewritesToPaths is exported for cross-test reuse if needed."
The helper has no caller. It is also not exported — the identifier is unexported,
so the comment is wrong about that too.

The alias is why nothing caught it. `unused` sees a reference and stops. Only a
whole-program reachability analysis with tests as roots looks past the alias to
ask whether anything reachable calls the function; running `deadcode -test ./...`
reports it, and that tool is not installed, not in the linter set, and not run by
any gate.

## Why it matters

This is a rule the repo states as a rule, in the file it treats as authoritative
for Go conventions, and it holds elsewhere that a guarantee depending on someone
remembering is not a guarantee. The pattern is mechanically detectable — an AST
walk for a `var _ = <ident>` where the identifier resolves to a function or type
in the same package is a small policy — so the gap between the stated rule and the
enforced rule is closable rather than inherent.

The consequence of leaving it is narrow but self-concealing, which is the part
worth fixing. A silenced symbol looks maintained: it compiles, it has a doc
comment, and the linter is quiet. The one instance in the tree carries a
justification that reads plausibly and is factually wrong in two ways, which is
what that class of comment tends to look like once the code it describes has moved
on.

## Options

1. **An AST policy in `internal/policies/`** flagging a package-level
   `var _ = <ident>` whose right-hand side is a bare identifier resolving to a
   function or type declared in the same package. Matches how every comparable
   convention in this repo is enforced, runs in the existing suite, and needs a
   firing fixture to avoid being vacuous. Composite literals used deliberately as
   policy fixtures (`var _ = entity{...}`) must not fire, so the check is on a
   bare identifier, not on `var _ =` generally.
2. **Add `deadcode` to the gate** — install it and run whole-program reachability
   in CI. Catches this class and genuine orphans together, at the cost of a new
   tool in the build and a triage burden: reachability flags deliberately-retained
   code identically to rot, so every hit needs an ownership check before anyone
   acts on it.
3. **Delete the one instance and leave the rule unenforced.** Honest about cost,
   and it restores the tree — but it leaves a documented rule with no detector,
   which is the condition this gap is about rather than a fix for it.

Option 1 is the lean: it is the shape of enforcement this repo already uses, it is
narrow enough to be precise, and it closes the rule-versus-detector gap rather
than only the instance. Option 2 is worth considering as a companion — it finds
strictly more, and the triage cost is real work rather than a reason not to — but
it answers a broader question than the one this gap asks.

Either way the live instance goes: the helper has no caller, no open issue names
it, and no decision retains it.

## Scope

Surfaced by a `wf-structural-sweep` pass after E-0073 wrapped. Sits with G-0465
and G-0471 as a third instance of one pattern — a rule this repo states, holds
itself to at review, and has no mechanical detector for.
