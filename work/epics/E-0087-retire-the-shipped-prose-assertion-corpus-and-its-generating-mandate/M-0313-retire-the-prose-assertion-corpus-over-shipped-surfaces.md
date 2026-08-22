---
id: M-0313
title: Retire the prose-assertion corpus over shipped surfaces
status: in_progress
parent: E-0087
depends_on:
    - M-0312
tdd: advisory
acs:
    - id: AC-1
      title: A phrase assertion over shipped prose fails the policy suite
      status: met
    - id: AC-2
      title: The cross-document citation walk still fails on a dangling reference
      status: met
    - id: AC-3
      title: The dispatch trigger-phrase checks still fail when a phrase is removed
      status: met
---
## Goal

Delete the prose- and heading-presence assertions over shipped surfaces across the
policy suite, preserving the two exception classes D-0070 names and demonstrating that
each surviving exception still fails when its property breaks.

## Context

Measured while scoping G-0596: the policy suite carries a body of test functions that
assert shipped prose still says particular things, spanning several thousand lines, with
no catch recorded across roughly fourteen months and several filed gaps recording drift
it failed to prevent. D-0050 fixed the rule for new tests but declined to retrofit;
D-0070 mandates the retrofit and settles that the disposition is deletion rather than
conversion.

M-0312 removes the mandate that regrows this corpus and must land first. Once it has,
deletion is unconstrained — whole files can go, including the package-level path
constants that only exist to satisfy the old predicate.

## Acceptance criteria

### AC-1 — A phrase assertion over shipped prose fails the policy suite

A policy fails the suite when a test that reads a shipped-surface fixture asserts its
prose contains a particular phrase, naming the file and line. The check runs over **test
source**, not over prose, which is what makes it immune to rewording and distinguishes it
from the class D-0070 retires.

**The allowlist is closed and carries no grandfather entries.** This clause is what makes
the deletion real rather than incidental. Without it the ban is satisfiable by exempting
everything that already exists — a move this repo has precedent for in the firing-fixture
gate's own ledger — and the corpus survives with every acceptance criterion green.

**What a green suite means, stated at the strength it actually has.** The check is
syntactic: it reasons about the shape of test source, not about types or the values that
flow through it. So it recognizes the spellings this repo writes — a literal or a phrase
list compared against content from a shipped read, including through a table's struct
field, a rebound path, a package-level needle, a case-folding wrapper, or an assertion
helper where the document is not the first argument — and it does not recognize every
spelling that could express the same claim. A comparison against a literal with no call
in it, a `cmp.Diff`, and a regexp match all sit outside the predicate.

Green therefore means *the common spellings are blocked and the retrofit was done by
hand*, not *no such assertion can exist*. The retrofit is what removed the corpus; the
ban is what stops the familiar shape being reached for again. Content drift beyond that
is held at review — the disposition D-0070 §Consequences and D-0071 already chose, not a
gap this milestone opens.

Two limits are known and recorded rather than closed. The check scans test functions, so
the same assertion written as a production policy is outside it — `m0211-guidance-operating-anchors`
is exactly that, and CLAUDE.md documents it as a deliberate anchor set. And a helper that
returns a filesystem path rather than document text can carry taint onto a value that is
not a document, which would misreport an assertion over command output; no instance
exists in the tree, and the fix needs type information rather than more syntax (G-0605).

Closure is a property of the code rather than a claim in a comment: the exemptions live
in one map per admitted class, so adding one means choosing a class and a case fitting
neither has nowhere to go. Two classes are admitted, and they are not D-0070's two — one
of D-0070's needs no exemption at all:

- **Trigger phrases.** The phrasings in a skill's `## When to use` section and its
  `description:` frontmatter that decide whether an assistant reaches for the skill.
- **Relationship checks reaching their second artefact through code**, by computing the
  expectation from the behaviour under test rather than restating it. D-0070 keeps the
  cross-document relationship check for its reach — it fails when either side moves, and
  no rewording satisfies it falsely — and that reasoning does not depend on the second
  artefact being a file. The needle is then a stable identifier rather than a phrase.
- **Cross-document relationship checks are not exempted, because the rule never fires on
  them.** Their needle is drawn from the second document rather than written into the
  test, so they fall outside the predicate by construction.

Fixture: plant a long-literal containment assertion in a test reading an embedded-skill
fixture; the policy fires and names it. Removing the plant returns the suite to green.

### AC-2 — The cross-document citation walk still fails on a dangling reference

The check that walks every ritual and fails a section reference naming a heading no
ritual defines survives this milestone intact, and still bites.

It is the one assertion in the corpus with a recorded catch — two dangling citations on
its first run — and D-0050 names its shape as the one to prefer: a relationship between
documents rather than a reading of one. No rewording makes it pass falsely.

Probe: introduce a section reference naming a heading that does not exist, confirm the
walk goes red, revert. An unproven survivor is a finding, not a completion.

### AC-3 — The dispatch trigger-phrase checks still fail when a phrase is removed

The assertions over trigger phrases in a skill's `## When to use` section and its
`description:` frontmatter survive, and still bite.

These pin dispatch behaviour rather than prose style: G-0353's session mining measured
the deployer agent at approximately zero dispatches before those phrasings existed. The
limit is worth restating, because it bounds how much the exception is worth — nothing
mechanical consumes a trigger phrase, so the property rests on an assistant's judgment.
D-0070 keeps the class on the strength of the evidence, not the soundness of the
mechanism.

Probe: remove one trigger phrase from the deployer card's `description:` and one from
`aiwfx-release`'s `## When to use`, confirm each goes red, revert.

## Constraints

- The two exception classes survive intact: cross-document relationship checks, and the
  trigger phrases in a skill's `## When to use` section and `description:` frontmatter.
  A pass that removes either has overshot.
- Disposition is per assertion against D-0070, not per file. Several files hold a
  genuine structural assertion within a few lines of a prose one.
- No test function is deleted merely to make a file pass.
- Every surviving exception is probed: break the property in the source document, confirm
  the test goes red, revert. An unproven survivor is a finding, not a completion.
- Coverage must not regress; the diff-scoped gate names any regression by file and line.

## Design notes

- D-0070 carries the disposition rules, the measurement, and the rejected alternatives
  (convert to shape assertions; limit the retrofit to headings; keep everything and hold
  content at review).
- Heading-presence assertions are in scope for deletion. A heading check exists to scope
  a body assertion; once the body assertion is gone it degrades to asserting the heading
  exists.
- The trigger-phrase exception rests on behavioural evidence rather than a mechanical
  consumer — D-0070 records that limit explicitly.

## Surfaces touched

- `internal/policies/` — the test files asserting over embedded skill, ritual, template,
  agent-card, and guidance prose
- `internal/skills/` — the same class of assertion, reached through the `go:embed` roots
  rather than a repo-relative path. D-0070 scopes itself by surface, not by package, so a
  ban stopping at one package would make "wrote the test elsewhere" an exception no
  allowlist records
- `internal/policies/d5_structure_test.go` — the citation walk, preserved

## Out of scope

- Retiring the exposition-tier design documents that some of these tests lock. Removing
  the lock is in scope; what becomes of the documents is separate work with its own
  decision.
- Any change to the `skill-body-id` check or the shipped-surface id rule.
- Re-pointing the backstop, which is M-0312's deliverable.

## Dependencies

- D-0070, accepted.
- M-0312, done — the backstop must be re-pointed before deletion, per E-0087's
  constraints.

## Coverage notes

Deletion removes test code rather than production code, so the diff-scoped gate should
report no newly-uncovered statements. Where deleting a test does drop the last cover for
a production line, that line is a candidate for deletion in its own right rather than a
reason to keep a vacuous assertion.

## Work log

### AC-1 — the ban, and the deletion it made checkable

`shipped-prose-assertion` landed with the corpus removed in the same commit,
because the two are one change: the ban is red until the deletion lands, and the
deletion is only durable because the ban exists. Commit `d3f1d53f6`.

The work-list was not in the tree and was re-derived by an AST pass rather than
by grep, which mattered: a literal-only scan miscounts in both directions,
condemning tests that assert over a `Violation.Detail` and missing every needle
that arrives through a variable.

### AC-2 — the citation walk, preserved

No change was needed. The walk draws its needle from the document it cites, so
it falls outside the ban's predicate rather than being exempted from it — a
property confirmed by measurement, not assumed.

### AC-3 — the dispatch trigger phrases, preserved

Five assertions across four carriers, each on `triggerPhraseExemptions` with the
dispatch claim that earns it.

### Post-review corrections

Three independent reviewers ran over the change-set; their findings are in
`## Reviewer notes`. The corrective work: four protected checks restored, the
ban's discriminator replaced twice, four recall holes closed, three false
positives fixed, two vacuous tests repaired, and the remaining survivors
deleted.

## Decisions made during implementation

- **The allowlist enumerates test functions rather than exempting a location.**
  Exempting anything read out of `## When to use` or `description:` was measured
  first: it would have covered fourteen assertions of which only five bear on
  dispatch, so location would have kept nine members of the corpus.
- **The scan covers every test package, not `internal/policies` alone.** D-0070
  scopes itself by surface; a package-scoped ban makes "wrote the test
  elsewhere" an exception nothing records.
- **A second exemption class was admitted: relationship checks that derive their
  expectation by running the code.** D-0070 names the cross-document form and
  keeps it for its reach — it fails when either side moves — and that reasoning
  does not depend on the second artefact being a file.
- **The claim was narrowed rather than the analyzer rebuilt.** Reviewers proposed
  `go/types`; that closes the resolution holes but not the dataflow ones, and the
  honest version needs SSA, which is more code than it replaces and a permanent
  cost on every test run. Recorded as **G-0605** with the full trade in
  [`docs/explorations/10-type-aware-static-analysis.md`](../../../docs/explorations/10-type-aware-static-analysis.md).

## Validation

- `go build ./...` — clean
- `go test ./...` — all packages green
- `make lint` — 0 issues
- `make coverage-gate` — diff-scoped statement coverage and the firing-fixture
  meta-gate both green
- `aiwf check` — 0 errors
- Survivor probes, each broken in the source document and reverted: the dangling
  `§"…"` citation, `(step 6)` → `(step 99)`, a fabricated `--kind`, a
  non-existent recipe path, and a removed trigger phrase in each of the two
  carriers. Every one went red and returned green on revert.

## Deferrals

- **G-0605** — whether aiwf's self-validation should move to type-aware analysis.
  Raised when reviewers proposed `go/types` for the recall holes; the analysis of
  what types and dataflow would each buy is in
  [`docs/explorations/10-type-aware-static-analysis.md`](../../../docs/explorations/10-type-aware-static-analysis.md).
  Aspirational: the blind spot is shared by every syntax-only check in the repo,
  which is the argument for doing it and the reason it is not this milestone's.
- **G-0606** — a prose assertion written as a production policy is outside the
  scan. `m0211-guidance-operating-anchors` is the live instance and is deliberate,
  so the class needs a disposition rather than a deletion.
- **G-0607** — a regexp match over shipped content bypasses the ban. Detection is
  easy; telling a structural pattern from a prose one inside a regexp literal is
  the part that needs deciding.
- **G-0608** — the negative regression pin is a shape D-0070 does not name. One
  shipped-surface site was dropped from a dead-CLI-form guard rather than invent a
  third exemption class; the coverage loss is recorded there.

## Reviewer notes

Three independent reviewers ran over the change-set: two on correctness, sliced
by concern, and one on design. They converged, and the milestone would have
closed on a false claim without them.

**What they found, and what changed.** AC-1 claimed a green suite proved the
corpus gone; it did not, and eleven members were still in the tree, invisible
because a path reaching a reader through a table's struct field or a `:=`
rebinding formed no taint. Four protected checks had been deleted — one
cross-document citation resolver and three deriving expectations from the
kernel — because the triage classified whole functions as prose by counting
assertion sites rather than reading them. That is the risk E-0087's own
constraints name, and applying disposition per file when the spec says per
assertion is what produced it. Both are fixed: the survivors are gone, the four
checks are restored and probed, and the claim is narrowed to what the check
delivers.

**Two design cuts were replaced rather than patched.** Position was doing work
meaning should do — a call in a condition was treated as an assertion, and a call
bound to a name as scoping. Both now key on semantics: a call returning document
text is narrowing wherever it sits, and a call is an assertion only when failing
its branch fails the test. Each replaced a hand-maintained list with a property.

**Declined, with reasons.** Rebuilding on `go/types` was proposed and is right in
principle, but `go/types` alone closes the resolution holes and not the dataflow
ones; the honest version needs SSA, costs more code than it replaces, and taxes
every test run — recorded as G-0605 rather than done here. The `acs-tdd-audit`
warning on the met ACs is left standing: back-filling a phase ladder after the
fact is indistinguishable from fabricating it, which defeats what the ladder
proves. D-0053 is left unchanged: its retirement trigger fired when
`verb_skill_factual_test.go` was deleted, and its body already names D-0070 as
what retires it, so the record reads true as written.

**Two vacuous tests were repaired.** Two "does not fire" cases passed for the
wrong reason — their fixture called an undeclared helper, so the haystack was
untainted incidentally rather than by the rule under test. Fixed and
mutation-checked: treating `Hint` as document text now reddens them, where before
it did not.

**Numbers stated in progress reports were wrong twice** and are corrected here:
129 test functions deleted and 16 added, not roughly 200; 18 files removed, not
20. Coverage rose over the change rather than falling — `internal/policies` from
92.8% to 93.2%, `internal/skills` unchanged.
