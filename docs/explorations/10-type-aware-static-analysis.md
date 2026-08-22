# Type-aware static analysis — what `go/types` and `go/ssa` would buy aiwf

This captures the question raised while reviewing M-0313's `shipped-prose-assertion`
ban: aiwf's self-validation is built entirely on **syntax**, and syntax matches
spellings rather than meaning. Two independent reviewers defeated a freshly written
AST analyzer in an afternoon, and the holes they found were not bugs in that analyzer
so much as the ceiling of its substrate.

It is an exploration, not a proposal with a plan. It states the question, the
evidence, and the trade, and deliberately stops short of prescribing an
implementation — the point is to make the decision *possible*, not to pre-empt it.
Tracked as **G-0605**.

---

## The question

Every Go-source check aiwf runs is a pattern match over a parsed syntax tree. Should
some of them instead be type-aware — resolving what identifiers *mean* and what
values *flow* — and if so, at what cost?

---

## What the three terms are

**Static typing** is a property of Go the language: every expression has a type known
before the program runs. This is already true of aiwf and is not the subject here.

**`go/types`** is the standard library's type checker exposed as a library — the same
engine the compiler uses. Given parsed syntax plus the packages it imports, it answers
three questions the syntax tree cannot:

- *what type does this expression have?* — `[]string` is distinguishable from `string`
- *which declaration does this identifier refer to?* — `clock.Now` resolves to
  `time.Now` regardless of the alias the file chose
- *what is this constant expression's value?* — `"### " + stepPrefix` folds to a string
  when both operands are constants

**Dataflow analysis** answers a fourth question neither syntax nor types answers: *what
value reached here, and where did it come from?* Types tell you a variable is a
`string`; dataflow tells you it holds the bytes of a particular file. Taint tracking is
the common form — mark values at a source, follow them through assignments, calls and
returns, and report when one arrives at a sink.

**`go/ssa`** (in `golang.org/x/tools`) is the substrate dataflow is normally built on.
It lowers a type-checked program into single-static-assignment form, where every value
has exactly one definition, which is what makes "follow this value" tractable instead
of heuristic.

The distinction that matters: **`go/types` is about resolution, `go/ssa` is about
propagation.** They solve different halves, and most of what defeated the M-0313
analyzer was the propagation half.

---

## What the current substrate cannot see

Read from the code, not inferred:

- `internal/policies/no_time_now_in_core.go:99` compares `ident.Name != "time"`. An
  aliased import — `import clock "time"` — makes the identifier `clock`, and the check
  skips it. The rule means "don't mint wall-clock time in the core"; the code asks
  "is the identifier spelled `time`".
- `internal/policies/atomic_write_chokepoint.go:114-118` matches the same way, on
  package identifier plus selector name.
- The M-0313 ban could not tell that `tc.path` held a shipped-surface path, because
  the value arrived through a struct field in a table literal. Nor that `p := ritualPath`
  carried it, because `p` is a variable rather than a constant.

The common shape: these checks are written about *meaning* and implemented against
*spelling*. Where the two agree the check works; where a rename, an alias, or an
intermediate binding separates them, it silently does not.

---

## Scope — is this wider than policies?

Measured across the repo: `go/ast` is imported by 66 files in `internal/policies` and
three elsewhere (`internal/cli/check/isolation_escape_test.go`,
`internal/stresstest/no_prod_change_test.go`, `internal/gitops/no_stash_test.go`), all
of them guard tests.

So the honest scope statement is narrower than "the whole codebase" and broader than
"the policy package": **every Go-source analysis aiwf performs is self-validation, and
aiwf ships no Go analysis to consumers.** A type-aware layer would change how this
repo checks itself. It would not change the product, the kernel, or anything a consumer
installs.

That cuts both ways. It bounds the blast radius of adopting it, and it bounds the
payoff — no consumer ever benefits.

---

## Advantages

- **Rules stop being defeated by renaming.** Alias-blindness, intermediate bindings and
  struct-field indirection are resolution problems, and resolution is exactly what a
  type checker does. The fix is structural rather than another arm on a matcher.
- **One fix, many consumers.** The blind spot is shared by ~66 checks. An analysis
  layer built once amortizes across all of them, which is a materially different
  proposition from building it for a single rule.
- **Rules that cannot be written today become writable.** Anything phrased as "this
  value must not reach there" — a secret into a log, an unvalidated payload across a
  boundary, a request-scoped value into a cache — is a dataflow question. Today such a
  rule can only be approximated by naming conventions.
- **False positives fall too, not just false negatives.** Much of the M-0313 analyzer's
  bulk exists to avoid firing wrongly; several of its heuristics are proxies for facts
  a type checker simply knows. Precision and recall improve together, which is unusual.
- **It retires proxies that are quietly wrong.** That analyzer treats "function returns
  a lone `string`" as "function returns document text". Types would replace the proxy
  with the fact.

## Disadvantages

- **It is bigger, not smaller.** The standing objection to the M-0313 ban is
  proportionality — ~840 lines for a partial guarantee. A types-plus-SSA analysis is
  *more* code, not less. It answers the recall objection while worsening the cost one.
- **A permanent cost on the inner loop.** `packages.Load` with full type information,
  plus SSA construction, is orders of magnitude slower than parsing. The current policy
  suite is seconds; type-aware loading of a module this size is tens of seconds, paid
  on every run by everyone.
- **A large new dependency.** `golang.org/x/tools` in a repo whose convention is to
  justify each dependency against real need. It is a well-maintained
  quasi-standard-library module, which softens but does not remove the objection.
- **`go/types` alone does not deliver the headline benefit.** Resolution and constant
  folding are the minority of what defeats these checks. The majority needs the
  dataflow layer, and that is where the engineering actually is.
- **It does not close the category.** "Is this expression an assertion?" stays a
  judgment call. Regexp-encoded matching, comparisons against literals, and content
  reached through a materialized temp file are design questions that no amount of type
  information settles.
- **Sophistication invites over-trust.** A more capable analyzer produces a greener
  suite that means *more* but still not everything — and the failure D-0050 named is a
  green run implying an assurance it cannot deliver. A stronger tool raises that stake.

---

## What would have to be true to adopt this

Not a plan — the conditions under which the trade flips:

1. **More than one check wants it.** One rule cannot carry the cost. A candidate list of
   existing checks that are demonstrably spelling-bound, with the drift each currently
   misses, is the first piece of evidence.
2. **The runtime cost is measured, not assumed.** A spike loading this module with
   `packages.NeedTypes|NeedSyntax|NeedDeps` and timing it settles the largest
   disadvantage either way. Everything above treats that number as an estimate.
3. **A tier boundary exists for it.** If type-aware checks run only in CI while
   syntax-only checks stay in the inner loop, the cost objection mostly dissolves — at
   the price of moving some catches later than pre-push.
4. **The layer is shared, not per-policy.** The value is amortization. A second bespoke
   analyzer with its own loading and its own taint model is the current situation with
   extra steps.

---

## Provenance

Raised during M-0313's independent review (E-0087), where two reviewers independently
defeated a syntax-only analyzer and one proposed `go/types` as the remedy. That
proposal understated the work — the holes were predominantly dataflow rather than
resolution — and the correction is what prompted writing the distinction down. The
milestone itself took the narrower path, keeping a syntactic check and narrowing its
claim to match what it delivers.
