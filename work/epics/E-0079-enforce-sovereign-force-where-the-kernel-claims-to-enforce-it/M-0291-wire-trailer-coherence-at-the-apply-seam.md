---
id: M-0291
title: Wire trailer coherence at the apply seam
status: done
parent: E-0079
tdd: required
acs:
    - id: AC-1
      title: Forced transition by a non-human actor is refused before commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: Coherence verdicts pinned across the full actor-and-trailer domain
      status: met
      tdd_phase: done
    - id: AC-3
      title: No production path commits without passing the coherence guard
      status: met
      tdd_phase: done
    - id: AC-4
      title: An ADR records that sovereign acts are prevented at the verb route
      status: met
      tdd_phase: done
---

## Goal

Put the trailer-coherence guard at the one seam every verb's commit passes
through, so a forced act by a non-human actor is refused at the moment it is
attempted rather than reported after it has already landed.

## Context

`CheckTrailerCoherence` is reached from two verbs. Every other verb that
constructs a sovereign force trailer commits without consulting it, and the
operator learns of the violation only when the pre-push check walks git history
— by which point the act is in the log and the exits are history rewrites.

The guard was never copyable to the verb layer, which is why it reached one verb
of four rather than drifting there by inattention. A verb's trailer set is
incomplete when the verb returns: the CLI layer appends the principal,
on-behalf-of, authorized-by and scope-ends trailers afterwards, so a guard
placed inside a verb would see no principal and refuse every legitimately
authorized non-human actor. The verbs that do call it assemble a complete set
themselves. `verb.Apply` is the single point downstream of both shapes.

## Acceptance criteria

### AC-1 — Forced transition by a non-human actor is refused before commit

Driven against the real binary, not the library. Every site that constructs a
sovereign force trailer refuses: the shared transition-trailer helper — which
serves promote, cancel, and both AC-granularity transitions — and the inline
sites in `add` and `authorize`.

Refusal is observable rather than inferred: a non-zero exit, a message naming
the rule that refused, and `HEAD` byte-identical before and after. A guard that
refuses but has already written is the failure this milestone exists to prevent,
so the unmoved-`HEAD` assertion is the load-bearing half.

The force-replace verbs stay open to non-human actors. `contract bind`,
`contract recipe` and `update --remove` declare a `--force` that means
force-replace, emit no sovereign trailer, and would break legitimate automation
if swept in — a different word spelled the same.

`update --remove` is named for completeness and cannot carry a case: it
registers no `--actor` flag, builds no plan, and never reaches the seam, so
there is no non-human invocation of it to pass or fail. The evidence below
covers the verbs that do reach it.

Evidence: subprocess-level tests per force-trailer construction site asserting
the unmoved `HEAD`, plus a passing case for each force-replace verb under a
non-human actor.

### AC-2 — Coherence verdicts pinned across the full actor-and-trailer domain

The rule set is a predicate over *(actor role × trailer presence-vector)*. The
test generates that cross product rather than enumerating the cases someone
thought of, so coverage is a property of the test's construction and survives a
tenth rule being added.

The guard at the seam enforces the rules predicated on a force trailer, not the
whole rule set (D-0060). Membership is decided by satisfiability: a rule
belongs at the seam only if every verb reaching it has some invocation that
satisfies it. A verb outside the provenance-decoration layer carries no
principal and registers no flag that could supply one, so enforcing the
principal rules there is a closed door rather than a rule. Audit-only is
sovereign too and is excluded for that same reason, which is why the criterion
is satisfiability rather than sovereignty.

The domain test covers the whole rule set regardless, since that is what the
function still computes for its other callers, and a second property pins the
seam's subset in both directions — refusing too little would let a forced act
commit, refusing too much closes verbs no invocation can reopen.

The epic's open question, whether the check side should grow an
`audit-only-with-force` counterpart, is answered here by measurement: no. A flag
mutex refuses `--force` alongside `--audit-only` as a usage error before a plan
exists, so neither the seam nor the check side is that rule's first barrier.

Evidence: a generated table over the full domain, each combination carrying its
expected verdict.

### AC-3 — No production path commits without passing the coherence guard

A policy under `internal/policies/` fails when a path can reach a commit without
the guard.

The predicate is over commit-construction sites, not over dispatchers reaching
`verb.Apply`. Once the guard sits inside `Apply`, enumerating callers stops
being the right question — every caller is covered by construction — and what
remains checkable is that the seam stays singular and carries the guard.

Placing the guard inside `Apply` is what makes the property structural instead
of policed: the non-dispatcher caller in the cell-coverage fixture is covered
without being named, because it too goes through `Apply`.

The scan resolves calls rather than matching text, and covers every
commit-construction primitive rather than only the one a verb is meant to use.
Both matter: an aliased import renames the selector so a name-based search finds
nothing, and holding only the intended call would watch the front door while
leaving three side doors open. Resolving calls also stops a primitive named in a
comment or a string literal from reading as a call site — `internal/verb/apply.go`
names `CommitTree` in comments inside function bodies, which a textual scan
cannot tell from a call.

Evidence: the policy failing against a fixture that reaches a commit off-seam —
directly, through an alias, and through each lower-level primitive — and passing
against the tree.

### AC-4 — An ADR records that sovereign acts are prevented at the verb route

The stance has two sides, and recording only the half this milestone builds
would misstate it: sovereign acts are *prevented* at the verb route and
*ratifiable* at the history route, the second being M-0292's subject.

The ADR also records why one seam suffices here where ADR-0038 needed two.

Evidence: the ADR at `accepted` status, with a structural assertion scoped to
its named sections.

## Constraints

- The wiring changes who may wield `--force`, never what it overrides. Tier-1 /
  Tier-2 semantics belong to G-0333 and stay untouched.
- No finding is downgraded to make this pass. `provenance-force-non-human` stays
  at error severity; the epic adds a way to clear it, not a way to ignore it.
- `contract bind`, `contract recipe` and `update --remove` must keep working for
  non-human actors.

## Design notes

- **One seam, not the two ADR-0038 uses.** That ADR needed a claim-side and a
  commit-side guard because a converging verb returns before a plan exists. A
  converging verb also writes no commit, so it emits no trailer and has no
  coherence to violate — the case that forced the second seam there cannot arise
  here.
- **The guard goes in `verb.Apply`, not in `cliutil`.** The self-assembling
  verbs and the cell-coverage fixture reach `Apply` without passing through the
  dispatcher layer, so a `cliutil` placement would leave exactly the paths this
  milestone exists to close.

## Surfaces touched

- `internal/verb/apply.go` — the seam.
- `internal/verb/coherence.go` — the rule set, and the force-predicated subset
  the seam enforces.
- `internal/cli/cliutil/apply.go` — the exit class a seam refusal reports.
- `internal/policies/coherence_guard_chokepoint.go` — AC-3's policy.
- `docs/adr/` — AC-4's record.

## Out of scope

- Delegated force (G-0023). That changes the provenance model; this makes the
  current model true.
- The ratification path (M-0292) and the surface corrections (M-0293).

## Dependencies

- None. This milestone is the epic's foundation; M-0293 and M-0294 depend on it.

---

## Work log

### AC-1 — Guard at the apply seam

Every force-trailer site now refuses a non-human actor before writing · commit
293dee60d · tests 5/5 subtests, full suite green

Measured before the change: `promote`, `cancel`, the AC phase transition, and
`add` each committed the forced act at exit 0. `authorize` alone refused, via
its own human-actor check rather than coherence — pinned in the same table so
that site stays covered whichever guard holds it.

Two claims recorded here during implementation were wrong, and the wrap review
caught both. Enforcing all nine rules at the seam broke no existing test, and
that was read as a measured blast radius of zero — true about the suite, false
about the behavior, because the suite never exercised a contract verb under a
non-human actor. And the rule order that keeps `force-non-human` from being
reported first was recorded as a fact to accommodate; it was a choice nobody had
made. Both are corrected below.

The readiness pass added a direct in-process test at the seam (commit 27ffa49e8)
after the diff-scoped coverage gate reported the guard's own return untested.
Both statements were true at once: the binary-level table proves the refusal
reaches an operator, and a subprocess contributes nothing to the coverage
profile. A behavioral audit cannot substitute for the mechanical one here, which
is the reusable lesson — the two answer different questions.

### AC-2 — Domain pinned, and a rule corrected against the design doc

Verdicts pinned at every point of the generated domain · commit 71e3f8f9c ·
tests green across the module

The domain is generated from two axes — actor role, and the presence subset of
the trailers any rule reads — so a rule added against a new trailer extends
it by one line rather than by a fresh set of hand-written cases. Three
assertions sit on it: a golden recording which rule fires where, invariants
sourced from the provenance design doc rather than from the code, and a
reachability check that fails when a rule is shadowed into never firing.

The doc-sourced invariants earned their keep immediately by failing: a principal
with no actor at all was reported coherent, which the doc calls incoherent.
Recorded as D-0059 and fixed here.

The reachability check exists because the rules return the first violation only,
so a broadened earlier condition can shadow a later rule entirely — leaving a
rule that reads as enforced, passes its own unit test through a direct call, and
never fires for any real trailer set. That is the same claim-without-enforcement
shape this milestone exists to remove, one layer down.

### AC-3 — The seam held singular and guarded

Policy asserts no production path commits off-seam · commit c0b45f3d7 · tests
green across the module

Placing the guard inside `Apply` made the no-bypass property structural, so the
policy does not enumerate callers: the cell-coverage fixture reaches `Apply`
without being a CLI dispatcher and is covered without being named. Measured
basis for scoping it to one call — `gitops.CommitVerbChange` has exactly one
production caller, and the other three commit-construction primitives have none
outside their own package.

Two clauses rather than one, because either alone is satisfiable while the
property is false: one commit site proves nothing once the guard is deleted from
it, and a present guard proves nothing if a second function commits alongside.

The scan is fail-closed in both directions it can fail silently. Finding no
commit site reports an orphaned scope rather than a clean tree. A file that
carries the commit call but does not parse is reported rather than skipped —
the sibling lock policy skips such a file, which is defensible for a scan that
asks whether callers take a lock, but not for one whose whole job is finding
that call.

The wrap review measured the first version as narrower than the AC claims: an
aliased import, and any of the three lower-level commit primitives, produced no
violation from this policy or from either of its neighbours. Rewritten to
resolve calls against the file's own import block, with a fixture per bypass.
The rewrite also removed a false-positive class nobody had hit yet — a policy
source file quoting these primitive names as string literals reads as a call
site to a textual scan.

### AC-4 — The stance recorded and ratified

ADR-0040 accepted, pinned by section · commit 30091160b · tests green across the
module

The ADR records both halves, because writing only the prevention half would
state the stance as a closed door rather than a gated one: the verb route is
closed, and the history route stays open to ratification, since the check rule
fires on commits no verb produced — an imported repo, a hand-crafted commit,
history predating the guard — which cannot be prevented retroactively.

Its Consequences section carries the things a reader would otherwise meet by
surprise: that the seam enforces only the force-predicated rules and why, and
that the refusal exits as a legality refusal rather than an internal failure.
Both are pinned by section.

### Wrap review — the seam's scope corrected

Seam narrowed to the force-predicated rules; refusal reclassified · commits
7ccfb535f, ab95ea1a9, 0559eecf4 · full gate green

The independent two-lens review found the seam closed four verbs it was
required to leave open. `contract bind`, `contract unbind` and both recipe
verbs never pass through the provenance-decoration layer, so they carry no
principal and register no flag that could supply one — a guard enforcing the
whole rule set refused all four with no invocation that could pass. Measured
against both binaries, base and head, before anything was changed.

The narrowing is D-0060. Two further defects landed with it: a seam refusal
exited as an internal error with no code in the envelope, and the rule order
made the refusal name a trailer pair rather than the force the operator
wielded.

AC-1's evidence clause named a second half that was never built — a passing
case per force-replace verb under a non-human actor. That absence is why the
closure went unnoticed, and the test now exists.

## Decisions made during implementation

- D-0059 — widen the principal rule to require a non-human actor, closing the
  gap between the design doc's required-together statement and a condition that
  only covered the human half.
- D-0060 — scope the apply-seam guard to the sovereign-force rules, so that
  every refusal it raises is one some invocation could satisfy.

## Validation

- `make check-fast` — vet across build tags, `golangci-lint` 0 issues, full
  suite green.
- `make ci` — race suite, coverage 90.2%, profile-driven gates,
  `aiwf doctor --self-check` 29/29.
- `AIWF_COVERAGE_BASE=epic/E-0079-… make coverage-gate` — diff-scoped statement
  coverage green, firing-fixture meta-gate green.
- `aiwf check` — 0 errors. The one warning,
  `provenance-untrailered-scope-undefined`, reports that the provenance audit
  was skipped for want of an upstream ref; the branch has never been pushed.
- Behavioral spot-checks against a real binary: a forced act by an agent exits
  1 carrying `provenance-force-non-human`, with HEAD unmoved; all four contract
  verbs exit 0 with a landed commit under `--actor ai/claude`.

## Deferrals

- G-0544 — wire the contract verbs through the provenance decoration layer.
  The hole the seam surfaced and the narrowing left in place: those four verbs
  commit an actor with no principal, and the push rejects the result. Deferred
  because supplying a principal is a provenance-policy change that needs its
  own branch, not one improvised at a wrap.
- G-0545 — fold this milestone's seam policy into the existing
  commit-construction seam policy. Both assert the seam is singular; only the
  guard-presence clause is new. Deferred as a refactor with its own review.
- G-0550 — a commit carrying force and no actor at all is refused by neither
  rule set, though the design doc says it should be. Not verb-reachable, so it
  concerns hand-crafted and imported history; the domain test cannot catch it
  today because its doc-sourced invariant routes through the implementation's
  own notion of a non-human actor.
- G-0551 — nothing checks that the verb-side and check-side rule sets agree.
  Two divergences already exist and both are currently benign, which is the
  state in which a third would go unnoticed.
- G-0552 — nothing bounds the Go build cache in the devcontainer. It filled the
  filesystem during this wrap and surfaced as three consecutive runs failing
  three different packages, which reads as a flaky suite rather than as a full
  disk.
- G-0546 — the two-shape trailer assembly. The CLI layer completes a plan the
  verb layer produced, which is why no verb can validate its own provenance and
  why a refusal speaks in trailer keys. Deferred as epic-sized.

## Reviewer notes

Two review rounds ran, each a fresh two-lens pass over the full change-set. The
second round's code lens found no surviving mutant across five probes — guard
deleted, force rules swapped back, seam re-widened, exit routing removed,
`audit-only-non-human` deleted — with the re-widening caught by a test that did
not exist before the first round.

The second round corrected one code defect and one framing error, both authored
in the first round's fix. `CoherenceError.Code()` mapped `audit-only-non-human`
to the generic incoherent-trailer code although the audit reports it under a
code of its own, breaking the one-identifier promise for a violation nothing
else names imprecisely. And the subset's criterion was stated as sovereignty,
which the kernel contradicts in its own refusal message and audit catalogue:
audit-only is sovereign and is deliberately *not* at the seam. The criterion is
satisfiability, which gives the same cut, gives the right answer for the next
rule, and survives its own repair — G-0544 makes the principal rules satisfiable
and they could then move without re-deriving the argument.

A third pass found two more, both about whether a claim was held rather than
about behavior. `TestPolicyCoherenceGuardChokepoint_IgnoresAPrimitiveNamedInAString`
could not fail: its fixture was written under `internal/policies/`, which the
shared walker skips wholesale so that scanners do not trip the policies they
implement, so the scan never read the file the test was about. Relocated to a
scanned path and mutation-probed. The other was the sole survivor of the guard's
rename, in D-0060's Decision paragraph — the accepted record both the ADR and
AC-2 defer to for the subset's boundary, pointing at a symbol nothing resolves.

That pass also found the design doc missing two rules the code enforces
(`audit-only` with force, and `audit-only` by a non-human actor) and its
finding-code table missing one code. AC-2's claim that its invariants are
doc-sourced rather than code-sourced rested on that section, so the doc was
corrected rather than the claim softened.

A fourth pass ran 13 mutation probes with no survivors, which is what closed
the vacuity question the third pass opened. Its one blocking finding was in the
change's blast radius rather than the guard: the sovereign-act refusal told a
non-human actor to `use --force --reason` to override, and this milestone made
that remedy refuse. The message is reachable only for a non-human actor, so it
was wrong every time it was shown — an operator following it moved from exit 2
to exit 1 and no further. Corrected to name only the human-run path, and pinned.
The same false remedy appears in two surfaces outside this milestone's reach;
both are now rows in M-0293's table.

That pass also found the seam policy treating a *method* named `Apply` under
`internal/verb/` as the seam, so a second commit site wearing the seam's name
was exempt from the policy looking for exactly that. One condition, and a
fixture that measured zero violations before it.

Two review findings were considered and declined, recorded here so the next
reader meets a decision rather than a blank.

- **The history-walking audit still states the principal rule in its narrower
  form.** Equivalent in effect for every commit that audit can see, since it
  returns early for a commit carrying no actor. A textual divergence, not a
  behavioral one; noted in D-0059's follow-ups rather than filed.
- **`import --dry-run` previews a trailer set the guard never validates.** A
  preview is not a commit, and no dry-run-capable verb builds a force trailer.
  Filing it would be ledger padding.

Two shapes in the test suite are worth knowing about. The refusal table's five
cases carry two independent failure reasons, not five — the first three share
one construction site, and `authorize` is held by its own guard; it is a
per-verb roster proving each verb reaches its site, not five independent
proofs. And `TestCheckTrailerCoherence_Rules` was reviewed as a partial-
retirement candidate, since its rule-firing cases are now a subset of the
generated domain; it was kept for the by-name tie to the design doc's worked
examples, which the domain does not carry.

The policy at `coherence_guard_chokepoint.go` pins the guard's presence in the
seam, not its position: a textual scan cannot see ordering. Ordering is pinned
behaviorally instead, by the unmoved-HEAD and no-write-landed assertions in
`apply_coherence_test.go`.

