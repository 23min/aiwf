---
id: M-069
title: Retrofit TDD-shaped tests for E-14
status: in_progress
parent: E-14
tdd: required
acs:
    - id: AC-1
      title: Envelope conforms to documented schema for every --format=json verb
      status: met
      tdd_phase: done
    - id: AC-2
      title: Single-commit-per-verb invariant asserted per mutating verb
      status: met
      tdd_phase: done
    - id: AC-3
      title: Trailer-key shape asserted per mutating verb
      status: met
      tdd_phase: done
    - id: AC-4
      title: Pre-push hook byte-golden plus template-equals-installed cross-check
      status: met
      tdd_phase: done
    - id: AC-5
      title: init then doctor --self-check seam in a fresh tempdir repo
      status: open
      tdd_phase: done
    - id: AC-6
      title: Native-Cobra drift test fails CI on passthrough-adapter regression
      status: open
      tdd_phase: done
    - id: AC-7
      title: Help-quality drift asserts Example present and no migration prose
      status: open
      tdd_phase: done
---
## Goal

Retrofit the load-bearing tests E-14 (Cobra and completion) shipped without. The audit closed via G-055 found that several E-14 milestones were promoted `done` while the AC text referenced behavior that no test exercises. This milestone is `tdd: required`; each AC walks `red` → `green` → `done` so the gap is closed by mechanical evidence, not narrative.

The seven ACs each pin one gap class identified in the audit. Production code is, in the audit's premise, already correct — what's missing is the test that would have failed before E-14 shipped and would fail again on regression. Where the production path turns out *not* to satisfy the test, that's a real second-order finding and gets fixed inline.

## Acceptance criteria

### AC-1 — Envelope conforms to documented schema for every --format=json verb

`internal/render/render.go` documents the JSON envelope contract: every `--format=json` invocation emits a single object with `tool` (always `"aiwf"`), `version` (non-empty string), `status` (one of `"ok"` / `"findings"` / `"error"`), `findings` (array, never null/missing — empty when none), optional `result` (verb-specific payload), and optional `metadata`. The contract is the load-bearing thing CI scripts and downstream tools key off — `findings` is grepped the same way across verbs, `result` is switched on by verb name.

The existing per-verb tests check the envelope loosely. `TestRun_ShowJSONEnvelope` asserts `tool == "aiwf"` and `status == "ok"` for one show invocation; nothing pins the contract across the *full* set of verbs that emit it. A regression where, say, `aiwf status --format=json` started omitting the `findings` array, or `aiwf contract verify` returned a status string outside the closed set, would not be caught by any current test — `findings` is the field downstream consumers grep, so its disappearance is a silent breaking change.

This AC adds a structural conformance test that exercises every verb supporting `--format=json`, parses the envelope, and asserts the schema:

- top-level keys are exactly the documented set (`tool`, `version`, `status`, `findings`, optionally `result`, optionally `metadata`); unknown keys fail the test.
- `tool` equals `"aiwf"` exactly.
- `version` is a non-empty string.
- `status` is one of the three closed-set values.
- `findings` is a JSON array (decodes into `[]any`, may be empty, must not be `null` or missing).
- `result` and `metadata`, when present, are JSON objects.

The matching uses `go-cmp.Diff` against a canonical schema-shaped value with `cmpopts.IgnoreFields` (and a comparer for the `findings` array contents) so the test pins **structure**, not specific run-varying values like the build-info version string or metadata timing fields. The verb table is the source of truth: a new `--format=json` verb that lands without an entry is the regression we want this test to catch on the next CI run.

The test drives the verbs through `run([]string{...})` (the same dispatcher production uses) and captures stdout, so it sits at the seam between the verb's flag-binding and `render.JSON` — a verb that constructed its envelope manually, or skipped `render.JSON` and emitted ad-hoc JSON, would fail the conformance assertion even if its godoc still claimed the contract.

The verbs covered: `check`, `show`, `history`, `status`, `schema`, `template`, `contract verify`, and `render --format=html` (which emits the envelope on stdout while writing HTML to disk).

### AC-2 — Single-commit-per-verb invariant asserted per mutating verb

CLAUDE.md design decision §7: "Every mutating verb produces exactly one git commit. That gives per-mutation atomicity for free. A failed mutation aborts before the commit." This is one of the load-bearing properties any change must preserve — together with stable ids, layered location-of-truth, and `aiwf check` as the chokepoint — and the audit closed via G-051 ("Planning sessions emit one commit per entity, not per logical mutation") was the user-visible symptom of an earlier era when this invariant was not enforced.

`TestBinary_MutatingVerbs_Subprocess` already runs every migrated mutating verb as a subprocess sequence and asserts that each invocation exits cleanly. It does *not* assert the commit count delta per verb. A regression where `aiwf promote` started emitting two commits (one for the entity update, one for a side-effect projection) — or where `aiwf cancel` emitted zero commits and stamped its mutation as part of the *next* verb's commit — would still pass that test. The kernel's atomicity guarantee, the property `aiwf history` projects against, and the per-mutation rollback story all depend on this delta being exactly 1.

This AC adds a sequence test that drives each mutating verb through the in-process dispatcher (`run([]string{...})`), records `git rev-list --count HEAD` before and after the verb, and asserts the delta is exactly 1. The sequence mirrors a typical consumer-repo lifecycle and is exhaustive over the user-facing mutating verb surface: `add` (each kind), `promote` (entity status, AC status, AC tdd_phase), `rename`, `edit-body`, `move`, `cancel`, `authorize` (open / pause / resume), `import` (default bundled-commit mode — multi-entity manifest must still be one commit), `reallocate`, and the `contract` family (`recipe install`, `bind`, `unbind`, `recipe remove`).

The "default-mode `import`" subcase is the audit's namesake: a manifest with N entities must produce exactly one commit, not N. The test seeds a multi-entity manifest specifically to pin that.

The assertion is `delta == 1` (strict equality), not `delta >= 1`. Strict equality catches both ends of the regression: a verb that silently produces a *second* commit (e.g. an audit-trail commit), and a verb that emits *zero* commits (the "deferred to next verb" smell). The verb table is, again, the source of truth — adding a new mutating verb without a row here is the regression this test surfaces.

### AC-3 — Trailer-key shape asserted per mutating verb

`aiwf history` projects the per-entity timeline by reading `git log` filtered on structured commit trailers (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`, plus the I2.5 provenance keys `aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`). The trailer-key shape is the projection's only contract — a verb that forgets a key, types a key wrong (e.g. `aiwf_verb` snake-case, `aiwf-acto` truncated), or emits a brand-new key the canonical set in `internal/gitops/trailers.go` doesn't know about *silently* breaks `aiwf history`'s rendering of that entity's timeline, the provenance audit's ability to walk authorized-by chains, and the policy-test catalog that greps for trailer values.

The existing infrastructure protects the *source side*. `internal/policies/PolicyTrailerKeysViaConstants` flags any production Go file that string-literals a known trailer name instead of referencing the `gitops.Trailer*` constant. `PolicyIntegrationTestsAssertTrailers` flags integration-test functions that drive a mutating verb without referencing the trailer-assertion API. Both are *static* checks — they prevent source drift but say nothing about runtime behavior. A verb whose code wires `gitops.TrailerVerb` to a literal `"promot"` (typo) compiles, passes the source-policy, and only surfaces when a human reads the resulting commit.

This AC adds a runtime test that drives every mutating verb through the in-process dispatcher and reads HEAD's trailers via `gitops.HeadTrailers`:

- The required keys `aiwf-verb`, `aiwf-entity`, `aiwf-actor` are present on every mutating verb's commit.
- `aiwf-verb` value matches the canonical verb name.
- `aiwf-actor` value matches the supplied `--actor` (`human/test` in the test's setup).
- Every trailer key on the commit is a member of the canonical set declared in `internal/gitops/trailers.go` (the `trailerOrder` slice). A new key landing without a corresponding `Trailer*` constant fails this test on the next CI run.
- `import` is the multi-entity case: the single commit must carry one `aiwf-entity` trailer per imported entity — which is also how `aiwf history` discovers the entity-set on a bundled-import commit.

The assertion uses the canonical set as its source of truth; adding a new trailer key in `trailers.go` automatically broadens what's accepted. Removing one (a deprecation) requires updating both the constant and the test, which is the right order. The test sits at the seam between verb dispatch and `gitops.CommitMessage` — a verb that constructed its commit through the canonical helper produces canonical trailers; a verb that bypassed the helper surfaces here.

### AC-4 — Pre-push hook byte-golden plus template-equals-installed cross-check

CLAUDE.md design decision §3: "`aiwf check` runs as a pre-push git hook. Validation is the chokepoint. The hook is what makes the framework's guarantees real; without it, skills are just suggestions." The pre-push hook is the load-bearing chokepoint that turns kernel rules from advisory into enforced. Hook content drift — between the template `preHookScript` returns and the bytes `ensurePreHook` writes to `.git/hooks/pre-push` — silently weakens that chokepoint. A regression where the install path quietly dropped the chain prelude (G45's `pre-push.local` chaining), or the brownfield guard, or the `exec aiwf check` line, would break the kernel's enforcement story.

The existing tests check the *substring* level only. `TestPreHookScript_HasBrownfieldGuard` greps for the brownfield literal; `TestInit_MigratesAlienPreHook` asserts the marker is present after migration; the broader `initrepo_test.go` assertions are similar `bytes.Contains` checks. CLAUDE.md `Substring assertions are not structural assertions` calls this out: a substring match proves a literal exists *somewhere*, not in the right *place*. The hook body has semantic weight on every line — the chain prelude order matters, the brownfield-guard placement matters, the `exec` is what makes the hook useful. Pinning it byte-for-byte is the right granularity.

This AC adds two paired tests:

- **`TestPreHookScript_ByteGolden`** renders `preHookScript("/AIWF_BIN")` (a sentinel binary path) and diffs the output against `testdata/pre-push.golden`. Any template change — body prose, marker, chain prelude, brownfield guard — surfaces as a failing diff. An intentional change requires regenerating the golden; an accidental one fails CI before merge.
- **`TestPreHookScript_TemplateEqualsInstalled`** runs `Init` in a fresh tempdir, reads the installed `.git/hooks/pre-push` bytes, re-renders `preHookScript(exePath)` with the same path `Init` resolved via `resolveExecutable()`, and asserts byte-equality. This is the seam test (CLAUDE.md `Test the seam, not just the layer`): the install path must use the template function as its sole source of truth. A regression where `ensurePreHook` post-processed the output, took a fallback branch, or quietly maintained a parallel template would surface here even if both halves passed their own unit tests.

The pair caps both regression vectors: byte-golden catches template drift; cross-check catches install-path drift. Together they make the chokepoint's content unambiguous.

### AC-5 — init then doctor --self-check seam in a fresh tempdir repo

`aiwf doctor --self-check` is the kernel's end-to-end smoke test: it spins up an internal throwaway repo, drives every verb through it, and reports pass/fail. Existing tests run it directly against a bare tempdir (`TestRun_DoctorSelfCheck_Passes`, `TestBinary_DoctorSelfCheck_Passes`) — they confirm doctor's *internal* sequence works. They do *not* exercise the consumer's natural quickstart flow: a user clones a fresh repo, runs `aiwf init` to scaffold it, and *only then* runs `aiwf doctor --self-check` to verify the install is healthy.

The seam between `aiwf init` (consumer-tempdir scaffolding) and `aiwf doctor --self-check` (internal-tempdir verb matrix) is where state from one could silently break the other. Concretely: `aiwf init` in the consumer's repo writes `aiwf.yaml`, `.git/hooks/pre-push`, `.gitignore` patterns, a CLAUDE.md scaffold, and skill adapters under `.claude/skills/aiwf-*`. `aiwf doctor --self-check` then runs from inside that repo and creates *its own* throwaway tempdir for the verb matrix. A regression where doctor's initialization implicitly read the consumer's `aiwf.yaml`, the consumer's hooks fired against the throwaway repo's commits, or doctor's tempdir naming collided with the consumer's `.git/` layout would fail this seam without showing up in either of the existing direct tests.

This AC adds `TestSeam_InitThenDoctorSelfCheck`, gated under `-short` (it builds the binary). The flow:

- Build the aiwf binary (`go build`).
- Create a fresh tempdir and `git init` it.
- Run `aiwf init --actor human/test` against that tempdir (full hook install — consumer parity).
- Run `aiwf doctor --self-check` from that same tempdir.
- Assert both invocations exit 0 and the doctor output contains the canonical pass marker (`self-check passed`).

The test runs the binary as a subprocess (the `runBinaryAt` pattern) so the consumer's installed pre-push hook fires correctly during any commits the doctor's internal repo happens to make. In-process dispatch (`run([]string{...})`) would deadlock because the test-binary itself is what `aiwf init` would have baked into the hook script.

A regression where init left state (a file in `.aiwf/state/`, a marker in `aiwf.yaml`, an env var) that doctor read during its own scaffolding, or a hook that intercepted the wrong git invocation, surfaces as a non-zero exit on either step. The existing tests pass even with such a regression because they skip the init step entirely.

### AC-6 — Native-Cobra drift test fails CI on passthrough-adapter regression

E-14 migrated every verb from a hand-rolled passthrough adapter (manual argv parsing wrapped around legacy verbs) to native Cobra commands with declarative flag binding. The migration's user-visible payoff is the AI-discoverability and shell-completion guarantees in CLAUDE.md: `aiwf <verb> --help` is authoritative because Cobra generates it from the same flag-binding source the runtime uses, and tab-completion works because every value-taking flag has a completion function bound to that same source. The completion drift test (`TestPolicy_FlagsHaveCompletion`) pins the completion-wiring half. It does *not* pin the migration's structural half — a regression where a contributor reintroduces a passthrough adapter (sets `DisableFlagParsing: true` on a Cobra command and walks `os.Args` themselves) would silently break flag binding, completion, and help generation, and the existing drift test would not catch it because the bypassed command's flags never reach `cmd.Flags()` for the walker to find.

The comment in `cmd/aiwf/main.go` at `newRootCmd` ("Every verb is a native Cobra command (E-14 left no passthrough adapters)") is currently the only thing pinning that property. CLAUDE.md design decision §"Kernel functionality must be AI-discoverable" treats the drift-prevention test in `internal/policies/` as the chokepoint, but the equivalent runtime check on the Cobra tree itself doesn't exist yet.

This AC adds two paired tests:

- **`TestPolicy_NoPassthroughAdapters`** walks every command in `newRootCmd()`'s tree and asserts `DisableFlagParsing` is false. A regression where a contributor sets that field on any verb fails CI with the verb's command path. The test also asserts `DisableFlagsInUseLine` is false (cosmetic but suspicious — typically paired with manual argv parsing).
- **`TestPolicy_NoPassthroughAdapters_DetectsRegression`** is the rule-test-test pair: it constructs a synthetic Cobra tree with `DisableFlagParsing: true` on one node, runs the same walker, and asserts the violation is reported. This pins both directions: production doesn't trip the rule, and the rule actually fires when it should.

The "test the test" shape closes the silent-no-op trap: a future refactor that broke the walker's loop (e.g., `walkCommands` quietly returning early) would let `TestPolicy_NoPassthroughAdapters` keep passing forever even though the rule had stopped firing. The synthetic-violation test catches that regression too.

### AC-7 — Help-quality drift asserts Example present and no migration prose

`aiwf <verb> --help` is the AI-discoverability surface (CLAUDE.md §"Kernel functionality must be AI-discoverable"). Cobra renders that output from each verb's `Long` / `Short` / `Example` fields, so help quality reduces to those three strings being present, current, and timeless. E-14's migration to native Cobra is closed; help text written during the migration that references the migration itself ("version verb migrated", "newly migrated to Cobra", "since the migration") is now drift — it leaks an implementation-history detail into the user-facing surface and dates the help text in a way that becomes increasingly misleading as the reference recedes.

This AC adds two paired drift tests:

- **`TestPolicy_ExamplePresent`** walks every Runnable command in `newRootCmd()`'s tree and asserts `Example` is non-empty. The Example block is what makes the help self-documenting: a verb without an Example forces every consumer (human or AI) to read source or guess invocation shape. A regression where a contributor adds a new verb without an Example block fails CI here. Non-Runnable parents (`aiwf contract`, `aiwf contract recipe`) and Cobra-generated commands (`completion`, `help`) opt out — they dispatch to children rather than to a verb body.
- **`TestPolicy_NoMigrationProse`** walks `Long` / `Short` / `Example` of every command and asserts none contain the case-insensitive substring `migrat` (which catches `migrated`, `migration`, `migrating`, `migrate` in any capitalization). The pattern is deliberately narrow — it ignores `Cobra` (the framework name, legitimate to reference), entity ids like `E-14`, and runtime-feature descriptions of the G45 hook-migration logic (those live in `fmt.Println` calls inside verb runners, not in Cobra-field strings the walker sees).

The audit found one real production drift: `aiwf add ac --help`'s Example used `--title "version verb migrated"` as a sample AC title — an E-14 planning artifact that escaped the migration's cleanup pass. The fix replaces it with a timeless example AC title (subject-matter neutral, framework-independent). The test's tightening forces this kind of cleanup to be intentional going forward — adding migration-flavored example titles fails CI before merge.
