---
id: D-0072
title: The shipped-prose ban is partial by design and exempts derived expectations
status: proposed
---
## Question

D-0070 retires prose-presence assertions over shipped surfaces and names two
surviving classes: cross-document relationship checks, and the trigger phrases
that drive skill dispatch. It also rules out a third disposition — "Conversion
is not a third disposition."

Implementing the ban raised two questions D-0070's text does not settle, and
both change what the accepted decision means in practice. They belong in the
record rather than only in a milestone's acceptance criterion.

First: a check can reach its second artefact through *code* rather than through
a second document, computing the expectation by running the thing under test.
Is that the cross-document class or a new one?

Second: a syntactic check cannot recognize every spelling of the assertion it
forbids. What, then, does a green suite mean?

## Decision

**Derived-expectation relationship checks are exempt, as a named class.** A check
that computes its expectation by running the code — rather than restating it as
a literal — is exempt alongside the trigger phrases. The exemption sets live in
one map per class, so admitting a case means choosing a class and a case fitting
neither has nowhere to go.

The reasoning is D-0070's own. It keeps the cross-document check for its reach:
it fails when either side moves, and no rewording satisfies it falsely. That
argument does not depend on the second artefact being a file. Where the
expectation is derived, the only literal left is a stable identifier, and the
failure mode D-0070 measured — a phrase pinned, then reworded past — cannot
arise.

Note the asymmetry this creates with the cross-document class, which needs no
exemption at all: its needle comes from the second document, so the rule never
fires on it. Only the derived form needs naming, because its needle is a literal.

**The ban is a partial guarantee, and says so.** It is syntactic: it reasons
about the shape of test source, not about types or the values flowing through
it. It recognizes the spellings this repo writes — a literal or a phrase list
against content from a shipped read, reached through a table's struct field, a
rebound path, a package-level declaration, a case-folding wrapper, or an
assertion helper where the document is not the first argument. It does not
recognize a comparison against a literal with no call in it, a `cmp.Diff`, or a
regexp match.

So a green suite means *the common spellings are blocked and the retrofit was
done by hand* — not *no such assertion can exist*. The retrofit removed the
corpus; the ban stops the familiar shape being reached for again. Content drift
beyond that is held at review, which is the disposition D-0070 §Consequences and
D-0071 already chose.

## Reasoning

Two alternatives were measured and declined.

**A coarse ban on reading a shipped path from a non-allowlisted test file.**
Around fifty lines instead of eight hundred, and unevadable by every spelling
that defeats the syntactic rule — a path cannot be read without being named.
Declined on a number: twenty-nine test files name a shipped path, and the corpus
lived in those files. It guards the perimeter and leaves the interior
unguarded, so it stops a *new* file pinning prose and does nothing about prose
re-added to an existing reader. It also exempts per file, and coarse disposition
is what caused this epic's one real overshoot.

**Rebuilding on `go/types`.** The right answer to recall, and proposed by
review. Declined for this epic because `go/types` closes the resolution holes
and not the dataflow ones; the honest version needs `go/ssa` and a taint
analysis, which is more code than it replaces, adds a large dependency, and
costs tens of seconds on every test run. The value ceiling is also low: the
protected content is prose that, measured across fourteen months, no assertion
ever caught drifting. Recorded as G-0605 with the full analysis in
`docs/explorations/10-type-aware-static-analysis.md`, because the blind spot is
shared by every syntax-only check in the repo and is worth deciding on evidence
rather than from this one motivating case.

## Consequences

- A green policy suite means less than the phrase "the corpus is gone" would
  suggest, and the acceptance criterion says so explicitly. Overstating it was
  the defect review caught before this milestone closed.
- Three escapes are known and tracked rather than closed: a prose assertion
  written as a production policy (G-0606), a regexp match (G-0607), and the
  negative regression pin, which is a shape D-0070 does not name at all
  (G-0608).
- Adding an exemption now requires choosing a class. A case fitting neither is a
  signal that the class set needs a decision, not that the allowlist needs an
  entry.
- The exemption maps are load-bearing and small. If either grows, that is
  evidence the predicate is mis-drawn rather than that the exceptions are
  multiplying.
