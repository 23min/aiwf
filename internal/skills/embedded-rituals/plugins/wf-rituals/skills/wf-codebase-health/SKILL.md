---
name: wf-codebase-health
description: Stack-agnostic field guide of code-health principles — module boundaries, contracts, data discipline, tests that pin behavior, errors/logs/audit, reasoning aids, operational properties. Use when designing a new module, planning a refactor, reviewing a non-trivial diff, writing a spec that introduces new boundaries, or scoring an inherited codebase Strong/Weak/Missing with file:line evidence. These are advisory forces, not rules — consult them, don't enforce them; the project's own conventions win. Complements wf-review-code (the per-diff gate) at the whole-codebase altitude.
---

# wf-codebase-health

A field guide for keeping a codebase legible — the small set of properties
that tend to be true of code a senior engineer can pick up in hours rather
than weeks, regardless of stack. Use it to undo "vibe coding" decay, or to
prevent it.

## This is advisory — forces, not rules

These principles are **forces, not commandments.** When forces conflict,
judgment wins. Nothing here is a pass/fail gate, and this skill does not
enforce anything — it is a reference the assistant *consults*. The consuming
project's own conventions always win where they differ. Delete the
principles that don't fit your domain; add the ones your team keeps
re-deriving.

If you find yourself wanting to *mechanically enforce* one of these, that is
a project-specific decision the project owns — write a linter rule, a CI
check, or a test in that project's own terms. This skill stays advisory by
design.

## Two faces — prime first, score second

This rubric is used at **two ends** of the lifecycle, and the *priming* end is
primary:

- **Prime ("do it this way")** — consult it *while designing and writing* code,
  so the structure comes out right the first time. This is the main use. The five
  highest-leverage forces (D1, C1, C3, B1/B2, E1) are also primed every turn via
  aiwf's guidance fragment; this skill is the full set, reached when you are
  shaping a module or a boundary.
- **Score ("did we do it this way")** — use it *after*, as a review checklist or
  a Strong / Weak / Missing scorecard (see *Scoring a codebase against this*). Useful, but catching a
  structural problem at review or wrap is far more expensive than not introducing
  it — so lean on the priming end.

## When to use

Reach for this skill when you are **shaping or auditing structure**, not fixing a
line:

- **Priming:** designing a new module/package/boundary; planning a refactor;
  writing a spec that introduces new seams; implementing an AC that adds
  structure.
- **Scoring:** reviewing a non-trivial diff for design; scoring an inherited or
  "vibe-coded" codebase to decide where to start.

## How to use it

- **Prompt list** *(prime)* — hold the principles while writing something new.
  The primary use.
- **Review checklist** *(score)* — while reading a diff.
- **Scorecard** *(score)* — score each principle Strong / Weak / Missing with
  `file:line` evidence (see *Scoring a codebase against this*).

The right next step is rarely "fix all of these." It's: pick the one or two that
fail hardest right now, fix those, then re-score.

## Relation to other skills

- **`wf-review-code`** is the **per-diff gate** — correctness, AC coverage,
  branch-coverage discipline, conventions, doc hygiene on the change in
  front of you. This skill is the **whole-codebase altitude** — the
  structural and operational properties a single diff review skips. They
  cross-reference; they don't duplicate. On a large or boundary-introducing
  diff, run both.
- **`wf-rethink`** re-derives one unit's design from intent. This guide is
  the vocabulary that informs that re-derivation.

## What this is NOT

- A style guide (no formatting rules).
- A framework (no specific libraries).
- A methodology (no agile / TDD / clean-architecture dogma).

---

# The principles

## A. Module boundaries

### A1. High cohesion

Each module has one reason to exist. A change that touches one concern
touches one module; a change that touches another concern touches a
different module.

**Smells:**
- A single file or class that mutates 10+ kinds of state.
- "And-also" function names (`load_config_and_apply_defaults`).
- Top-of-file imports spanning unrelated subsystems (filesystem + HTTP + DB
  + UI in one file).
- A function taking 8+ unrelated parameters because it does 8+ unrelated
  things.

**Moves:**
- Name the module's single concern in one sentence. If you can't, split it.
- Group functions that change together; split functions that don't.
- Push side concerns (logging, metrics, retries) into decorators or
  wrappers, not the core function body.

**Tradeoff:** premature splitting creates fictional boundaries that get
reabsorbed later. If three things change together every time, they are one
thing — leave them.

### A2. Low coupling

Modules talk through narrow, named interfaces. Changing one module's
internals doesn't ripple into others.

**Smells:**
- Module A reaches into module B's private state (`b._cache[...]`).
- Cyclic imports.
- "Helper" modules everyone imports because they accumulated miscellaneous
  utilities.
- Changing one function forces edits in five unrelated files.

**Moves:**
- Define the interface (signatures, dataclasses, protocols) before the
  implementation.
- Prefer data passed in arguments to data fetched from globals.
- If A and B both need the same primitive, push it down to a shared module
  they both depend on — don't let A reach into B for it.

**Tradeoff:** zero coupling means everything is duplicated. Some coupling is
fine — even good — when the cost of indirection exceeds the cost of the link.

### A3. Layered (no upward dependencies)

Higher-level modules depend on lower-level ones. Never the reverse. The CLI
depends on the domain; the domain doesn't import the CLI.

**Smells:**
- A "core" module that imports its UI.
- A library that knows about its callers ("if called from the web handler,
  do X").
- Reverse-engineering required to find the entry point.

**Moves:**
- Draw the dependency graph. Cycles or upward arrows are bugs.
- Inject dependencies down (pass a logger; don't import one with caller
  knowledge baked in).
- Keep the "what" (domain logic) separate from the "how" (transports,
  storage drivers, UI).

**Tradeoff:** strict layering adds indirection. Two layers is usually enough
for small codebases; resist building five.

---

## B. Contracts

### B1. Typed interfaces

Inputs and outputs at module boundaries are named types — dataclasses,
structs, TypedDicts, models. Not loose `dict`s or tuples.

**Smells:**
- Functions returning `dict[str, Any]` where the keys are "the schema."
- Tuple returns where position matters and no one remembers what.
- "Magic" string parameters (`mode="strict"`) with no enum.
- Callers reach into return values with string keys validated nowhere.

**Moves:**
- Promote every "shape that crosses a boundary" to a named type.
- Use enums for closed sets of options.
- Make types immutable by default; mutate via explicit builders.

**Tradeoff:** types have a maintenance cost. For private shapes that live
within one module, loose dicts are fine.

### B2. Schemas at boundaries

Wherever data crosses a process boundary — JSON over HTTP, JSONL in a shared
file, DB rows, queue messages — the shape is declared once and validated.

**Smells:**
- Two languages each define their own version of the same struct; drift is
  invisible until something breaks.
- A file's schema lives only in the writer's code.
- A "version" field exists but nothing checks it.
- The only schema documentation is "see the example file."

**Moves:**
- One declaration per schema, codegen the rest (JSON Schema, Protobuf,
  OpenAPI — pick one).
- Validate at the boundary (when reading), not deep in the consumer.
- Equivalence tests between writer and reader (see D2).

**Tradeoff:** codegen is friction. For one-off internal files one team owns
end-to-end, a hand-written shared model is fine.

### B3. Pre/post conditions and invariants

Functions document what they require and guarantee. Invariants ("this list
is always sorted," "this set never contains duplicates") are named and
tested.

**Smells:**
- Defensive code at every call site because callers can't trust the function.
- "Sometimes this returns None" without docs.
- Behavior that depends on hidden global state.
- A bug fix that says "we forgot it could be empty" — repeatedly.

**Moves:**
- Document inputs ("must be UTF-8, non-empty") and outputs ("returns sorted,
  deduplicated, non-empty").
- Assert the invariant near the construction site, not at every consumer.
- Reach for type narrowing (`NonEmptyList`, `SortedList`) when the invariant
  is load-bearing.

**Tradeoff:** asserts in hot paths cost cycles. Pre/post conditions matter
most at module boundaries; fewer are needed inside a module where one author
controls all call sites.

---

## C. Data discipline

### C1. Single source of truth

Each fact lives in one place. Derived facts are computed, not duplicated.

**Smells:**
- "Why does the UI show 5 when the database says 4?"
- Two stores claiming to own the same record (one is stale).
- A cache without a documented invalidation rule.
- The same data living in three formats across the codebase.

**Moves:**
- Name the canonical store for each piece of state.
- Derive everything else with pure functions.
- If you must cache, document the invalidation: rebuilt on X, valid for Y.

**Tradeoff:** denormalization is sometimes necessary for performance. When
you denormalize, name the master, the copy, and the rule that aligns them.

### C2. Idempotence

Re-running an operation against unchanged inputs converges on the same
state. No cruft, no accumulation, no surprises on retry.

**Smells:**
- Re-running the import script creates duplicate records.
- A retry produces a different result than the first call.
- "Run this once" warnings in the README.
- Crash recovery requires manual cleanup.

**Moves:**
- Identify the "key" of each operation (content hash, request ID, primary
  key) and make it the basis for "have we seen this?"
- Distinguish "create if absent" from "always create."
- Test idempotence directly: run twice, assert state unchanged after the
  second run.

**Tradeoff:** idempotence costs extra reads. For high-throughput hot paths,
log-and-deduplicate-later may be faster than check-before-write.

### C3. Atomic writes

A crash mid-write leaves persisted state either fully-old or fully-new —
never half-written. Same for multi-step state changes: all complete or none.

**Smells:**
- A crash leaves the first half of the new content followed by the tail of
  the old.
- "Sometimes the file has a partial line at the end."
- Recovery code that "tolerates" corruption by skipping it.
- Two writes that must agree (index + content) where one can succeed and the
  other fail.

**Moves:**
- Single-file: write a sibling temp, fsync, rename.
- Multi-file: write all temps, fsync, then rename in one predictable order.
- Multi-store: a transaction or two-phase commit, or accept the asymmetry
  and design recovery around it.

**Tradeoff:** atomic writes cost an fsync and a rename. For short-lived
caches that can be rebuilt, ordinary writes are fine.

### C4. Versioned schemas with migration paths

When stored-data shape changes, there's a declared path forward — not "edit
the file by hand."

**Smells:**
- The README says "delete the old JSON before upgrading."
- A field rename requires a coordinated deploy across writer and every
  reader.
- "Legacy format" handling nobody can remove because we don't know who still
  has old data.
- Silent data loss when a reader skips a field the writer added.

**Moves:**
- Embed a version field from day one.
- For each bump, ship an idempotent, re-runnable migration step (see C2).
- Plan forward-compat (readers tolerate unknown fields) and backward-compat
  (writers can emit the old shape for one cycle).

**Tradeoff:** versioning is bureaucracy. For schemas one process owns
end-to-end with no historical data on disk, you can skip it.

---

## D. Tests that pin behavior, not implementation

### D1. Behavior pinned, not structure

Tests assert what the code does for inputs X — not which helpers got called,
in what order, with which mocks.

**Smells:**
- Tests fail when you rename an internal function.
- Tests mock five things to call the one under test.
- Refactoring breaks 20 tests that should have been one snapshot.
- "Test setup" is longer than the test body.

**Moves:**
- For legacy code: characterization snapshots — capture current output, pin
  it, refactor against the gate.
- Prefer integration over unit when the cost is similar.
- Mock at process / network / filesystem boundaries — not at internal
  function boundaries.

**Tradeoff:** integration tests are slower. For pure-function algorithm code
(parsers, scorers, validators) unit tests are right.

### D2. Equivalence tests at seams

Where two implementations claim to be interchangeable (in-memory fake and
real DB; two readers of one format), prove it. Run both against the same
scenarios; assert equivalent decisions.

**Smells:**
- "The fake works but production breaks differently."
- A reader-writer pair where one side silently drifted.
- Two libraries claiming the same protocol with no shared conformance test.

**Moves:**
- Define a contract (interface, protocol, test matrix).
- One test suite, parameterized over implementations.
- Run the matrix in CI for every change to either side.

**Tradeoff:** equivalence tests double the surface that must stay in sync.
For implementations that genuinely differ, test only the contract you share.

### D3. Branch coverage on touched code

A coverage floor on the lines and branches you change in this PR — not a
retroactive bar on legacy modules.

**Smells:**
- Coverage celebrates 92% overall while the new code is at 30%.
- A test "passes" because the branch under test never ran.
- No coverage report at all — "we test what matters."

**Moves:**
- Gate the merge on coverage-on-diff ≥ threshold, not absolute %.
- Branch coverage, not just statement coverage — `if x:` with no else-test
  is a gap.
- Raise the floor as the codebase improves; don't backfill legacy in one go.

**Tradeoff:** coverage isn't quality. A test can hit a branch without
asserting anything useful. Coverage is necessary, not sufficient.

### D4. Tests at the right altitude

Unit tests for pure functions; integration tests at module boundaries;
end-to-end for externally-observable behavior. Don't mock what you can
integration-test cheaply.

**Smells:**
- Every test mocks the database.
- "Unit tests" that exercise three modules and two real files.
- No end-to-end test — "we tested all the units."
- The same scenario tested at four altitudes, four different ways.

**Moves:**
- Per scenario, pick one altitude and document why.
- Pure functions get unit tests; CLI commands get end-to-end tests.
- Mock at process boundaries (the network call, the LLM API), not internal
  calls.

**Tradeoff:** integration tests are slower and harder to isolate. For
algorithmic correctness with easy-to-construct inputs, unit tests give
better feedback per second.

---

## E. Errors, logs, audit trail

### E1. Structured logs

No bare `print()` or unstructured `log.info("did the thing")`. Events have a
name and a context dict; output is machine-renderable.

**Smells:**
- Log messages full of string interpolation.
- Logs are grepped, never queried.
- "I added logging" means new `print()` calls.
- Production debug means `LOG_LEVEL=DEBUG` and hoping.

**Moves:**
- One structured logger; the rendering format is a config flag.
- Every emit binds context: `log.info("match_succeeded", file=..., score=...)`.
- Capture log events in tests so you can assert "this event fired with these
  fields."

**Tradeoff:** structured logging is more code at the emit site. For
throwaway scripts, `print` is fine.

### E2. Designed failure modes

What happens on missing input, unparseable file, timeout, disk full,
concurrent access — is documented and tested. Not "we'll find out when it
breaks."

**Smells:**
- Exception handlers that say `pass` or `continue`.
- "Why is the file empty?" leading to a swallowed exception three modules
  deep.
- A retry loop with no jitter, no backoff, no max attempts.
- Race conditions discovered in production.

**Moves:**
- Per module, list the failure modes; pick one of surface / retry / fallback
  / fail-fast and document the choice.
- Test the failure paths; a fault-injection test is worth its weight.
- Distinguish "expected" failures (no match) from "unexpected" (parser
  crash) — different handling, different logs.

**Tradeoff:** designing every failure mode upfront is over-engineering for a
prototype. Reach for it when the code matters.

### E3. Audit trail

Every significant state change leaves a record. For systems where trust
matters (financial, medical, security, legal), this is non-negotiable.

**Smells:**
- "Who deleted that record?" — nobody knows.
- The provenance of a value is "the database has it that way."
- A bug surfaced because someone manually edited production.

**Moves:**
- Append-only event record alongside the state-of-the-world store.
- Each event names the actor, action, before, after, timestamp.
- Treat the record as data — query it, replay it, test against it.

**Tradeoff:** auditing every read is overkill. Audit writes and decisions;
infer reads from logs.

### E4. Self-explaining errors

When something fails, the message says what was tried, what was expected,
what was found. Stack traces lead to the real problem, not the re-raise.

**Smells:**
- `raise Exception("error")`.
- Messages containing the function name and nothing else.
- Re-raising in a way that loses the original cause.
- The user's question after an error is always "what does this mean?"

**Moves:**
- Errors carry context: `matched {a}, expected {b}, in {path}` — not
  `"mismatch"`.
- Preserve causes (`raise X from e`, wrap with the original included).
- Review error messages like UI copy, not an afterthought.

**Tradeoff:** rich errors take effort. For internal-only systems with a
small team, "go read the log" may be acceptable.

---

## F. Reasoning aids

### F1. Names that don't lie

A function does what its name says — no more, no less. A variable contains
what its name claims.

**Smells:**
- `get_user` that also creates a user if absent.
- `validate` that mutates.
- A boolean named `loaded` that means "loaded or failed."
- "Util" / "helper" / "manager" / "handler" — names that say nothing.

**Moves:**
- Read function names aloud as sentences. If "`get_user` creates and saves a
  user" sounds wrong, rename.
- Names reveal *intent*, not implementation.
- A function that does N things gets renamed honestly or split.

**Tradeoff:** renaming is invasive. Sometimes the right move is to fix a
lying name the next time you touch the function.

### F2. Comments only for non-obvious "why"

Every comment answers a question the code can't: a hidden constraint, a
subtle invariant, a bug workaround, a historical decision that would be
reverted without context.

**Smells:**
- Comments restating the code in English.
- "Increment counter" above `counter += 1`.
- Parameter/type block comments when the signature already says it.
- Stale comments contradicting the code.

**Moves:**
- Delete comments that restate code.
- Keep comments that explain a surprising choice.
- When tempted to comment, ask "could I rename a variable/function instead?"

**Tradeoff:** for public APIs, docstrings are expected even when "obvious."
Internal code can be sparser.

### F3. Decision records that survive turnover

The "why" of significant choices survives the people who knew it. ADRs,
design docs, decision notes — the format matters less than the practice.

**Smells:**
- "Why is it like this?" answered with "ask [person who left]."
- A code comment that says "see Slack thread from 2022."
- Two parts of the codebase implementing the same thing differently with no
  rationale.
- A refactor reverting a load-bearing choice nobody knew was load-bearing.

**Moves:**
- One short doc per non-obvious decision: context, options, choice,
  consequences, date, author.
- Link the doc from the code where the decision is enforced.
- Update or supersede when the decision changes; don't delete — history
  matters.

**Tradeoff:** an ADR for every choice is paralysis. Reserve them for
decisions future-you (or a new hire) would otherwise re-litigate.

---

## G. Operational properties

### G1. Reproducible

Same inputs → same outputs. No hidden time / random / environment / network
dependencies in business logic.

**Smells:**
- A test passes sometimes, fails sometimes.
- "Works on my machine."
- Output depends on what was in `/tmp` at 3am.
- `datetime.now()` and `random()` scattered through pure-looking functions.

**Moves:**
- Push non-deterministic inputs (time, randomness, env) to the edges; inject
  them.
- Capture and replay: every run can be saved and replayed against future
  code.
- Containers, lockfiles, pinned versions.

**Tradeoff:** strict determinism removes legitimate randomness (jitter,
sampling). When randomness is real, seed it explicitly.

### G2. Reversible

Destructive operations are guarded, undoable, or both. You can recover from
a botched run without restoring from backup.

**Smells:**
- A single typo deletes production data.
- "We don't have undo because nobody asked."
- The only recovery is a week-old backup.
- Dry-run is a separate implementation from real-run.

**Moves:**
- Soft-delete by default; hard-delete is a separate operation.
- `--dry-run` is the same code path as the real run, writes routed to a stub.
- Confirmations match the blast radius (a prompt for one file; a ceremony
  for the whole database).

**Tradeoff:** soft-delete costs storage. For genuinely transient data,
hard-delete is fine.

### G3. Observable in production

When the user says "this looked wrong," you can answer: here's what
happened, here's why, here's the data it saw. Logs, metrics, traces,
provenance fields.

**Smells:**
- "Can you reproduce it?" is the first question after every bug report.
- Production decisions have no recorded reasoning.
- A score is shown with no way to see its inputs.
- Dashboards exist for infrastructure but not business logic.

**Moves:**
- Provenance fields on records: which version / method / inputs produced
  this value.
- Decision logs: every non-trivial branch records why it took the branch.
- Metrics: per-step counts, latencies, success/failure rates.
- Sampling for high-volume paths; full capture for slow ones.

**Tradeoff:** observability has overhead. Sample aggressively on
latency-critical paths; capture everything on correctness-critical ones.

---

# Scoring a codebase against this

1. **One reviewer per principle.** Evaluate one principle end-to-end; find
   evidence. Don't grade everything at once.
2. **Concrete evidence.** Cite `file:line` for every verdict. "Coupling is
   bad" is not a finding; "module X reaches into Y's private state at A:42
   and B:177" is.
3. **Strong / Weak / Missing.** Three levels. Don't grade 1–10; you'll spend
   time defending 6 vs 7.
4. **Adversarial pass.** A second reviewer tries to refute the first's
   "Strong" verdicts. A Strong that survives a real refutation attempt is
   actually strong.
5. **Prioritize by leverage.** Pick the weak principle whose fix unlocks the
   most downstream work. Score informs sequencing; it doesn't dictate it.

## Each principle taken too far

| Principle | Overdone becomes |
|---|---|
| High cohesion | Micro-modules that obscure the flow |
| Low coupling | Indirection everywhere; one-line wrapper functions |
| Layered | Five dispatchers between input and effect |
| Typed interfaces | Every internal dict promoted to a type; "type theater" |
| Schemas at boundaries | A schema registry for files one process writes |
| Single source of truth | A central god-object everyone depends on |
| Idempotence | Check-before-write where conflict is impossible |
| Atomic writes | Two-phase commits for cache rebuilds |
| Versioned schemas | Version field on every transient struct |
| Behavior pinning | Refactor-resistant tests that hide real regressions |
| Equivalence tests | Conformance tests for implementations sharing no contract |
| Branch coverage | 95% coverage of nothing meaningful |
| Structured logs | Event soup nobody queries |
| Designed failure modes | Speculative recovery for failures that never happen |
| Audit trail | Auditing everything; storage explodes |
| Names that don't lie | Renaming as a hobby |
| Comments for "why" | Three-paragraph comments above five-line functions |
| Decision records | An ADR for picking a CSS color |
| Reproducible | Determinism where randomness was legitimate |
| Reversible | Confirmations on harmless reads |
| Observable | Logging every line; signal lost in noise |

## Priority when you can't do everything

Inheriting a vibe-coded codebase, the principles don't pay back equally.
Approximate order of leverage:

1. **D1 Behavior pinning** (characterization tests) — without this, no
   refactor is safe. The gate.
2. **C3 Atomic writes** and **C1 Single source of truth** — correctness
   before structure.
3. **E1 Structured logs** — you can't fix what you can't see.
4. **B1 Typed interfaces** and **B2 Schemas at boundaries** — most bugs
   caught per unit of effort.
5. **A1 High cohesion** + **A2 Low coupling** — the classic refactor;
   attempt only after 1–4.
6. **E3 Audit trail** + **G3 Observability** — once stable, make it operable.
7. Everything else, as bandwidth allows.

The principle this list is itself an instance of: **don't try to fix
everything at once.**

## When NOT to apply this

- One-off scripts.
- Prototypes you'll actually throw away.
- Code where the author is the only reader and the lifetime is days.

The cost of clean code is real; it pays back over time and over multiple
readers. When neither time nor readers exist, do the simple thing.

---

*Meant to be edited. Replace examples with ones from the codebase it's
applied to; add principles your team re-derives; delete those that don't fit.
The goal is shared vocabulary, not orthodoxy.*
