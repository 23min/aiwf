# CLAUDE.md — ai-workflow repo

This repo carries `aiwf` — a small experimental framework that helps humans and AI assistants keep track of what's planned, decided, and done, by validating a small set of mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) for what aiwf commits to. Read [`docs/pocv3/archive/poc-plan-pre-migration.md`](docs/pocv3/archive/poc-plan-pre-migration.md) for the four sessions of work that produced it. **Gaps live as `aiwf` entities under `work/gaps/` (per G38 dogfooding the kernel against itself); run `aiwf list --kind gap` or `aiwf show G-NNN` to inspect them. The pre-migration text record is archived at [`docs/pocv3/archive/gaps-pre-migration.md`](docs/pocv3/archive/gaps-pre-migration.md) for historical reference.**

---

## Engineering principles

- **KISS — keep it simple.** Prefer the boring solution. Three similar lines beats a premature abstraction. Avoid cleverness — reflection, metaprogramming, deeply nested generics, control-flow tricks — unless the simple version is demonstrably worse.
- **YAGNI — don't build for tomorrow.** No speculative interfaces, no "we might need this later" config knobs, no plugin architectures for a single implementation. Add the second case when it shows up; abstract on the third.
- **No half-finished implementations.** If a feature lands, it lands tested. Stubs and TODOs in shipped code are a smell, not a milestone.
- **Errors are findings, not parse failures.** `aiwf check` loads inconsistent state and reports it; it does not refuse to start. Validation is a separate axis from loading.
- **The framework's correctness must not depend on the LLM's behavior.** Skills are advisory; the pre-push git hook and `aiwf check` are authoritative. If a guarantee depends on the LLM remembering to invoke a skill, it is not a guarantee.
- **Kernel functionality must be AI-discoverable.** Every verb, flag, JSON envelope field, body-section name, finding code, trailer key, and YAML field is reachable through channels an AI assistant routinely consults: `aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, this `CLAUDE.md`, or the design docs cross-referenced from it. If an AI assistant has to grep source to learn a kernel capability, the capability is undocumented. New capabilities ship with their `--help` text and skill-level documentation alongside the implementation, not after.
- **CLI surfaces must be auto-completion-friendly.** Every verb, subverb, flag, and closed-set value (kinds, statuses, format names, entity ids) is reachable via shell tab-completion. Static enumerations are wired through Cobra's `ValidArgs` and `RegisterFlagCompletionFunc`; entity ids enumerate dynamically by shelling back to `aiwf` from the generated completion script. Auto-completion is the human-facing peer of the AI-discoverability rule above: humans tab through the surface, AI assistants read `--help` and skills — both must traverse the *same* canonical surface, not two parallel ones. The drift-prevention test in `internal/policies/` (E-14 / M-054) is the chokepoint — a verb or flag added without completion wiring fails CI, so the guarantee does not depend on reviewer vigilance.
- **Provenance is principal × agent × scope, not just operator.** When the human directs the LLM in conversation ("add a gap that says X"), the LLM is a *tool* — the human is the principal, the LLM is the agent, no co-actor inflation. When the human authorizes autonomous work (`aiwf authorize E-03 --to ai/claude`), the agent operates within that scope until the scope-entity reaches a terminal status or the human pauses. `--force` is sovereign: only humans wield it. See [`docs/pocv3/design/provenance-model.md`](docs/pocv3/design/provenance-model.md) for the full model.

For Go-specific rules (formatting, linting, testing, coverage, error handling, CLI conventions, commit-trailer convention), see the *Go conventions* section below.

---

## Working with the user

- **Q&A / interview format.** When the user says "Q&A", "interview me", or anything similar — or whenever you have ≥3 distinct decisions queued and would otherwise dump them in one message — present questions or findings **one at a time**, not as a batch. For each item, give:
  1. **Context** — what the question is about and why it matters here.
  2. **Pros / cons** (or whys / why-nots) for each option.
  3. **Risks**, if any.
  4. **Your lean** and the reasoning behind it. State it plainly; don't hedge to the point of uselessness.
  5. A **numbered list of options** the user can pick from (including "something else").

  Wait for the user's choice before moving to the next item. Once they pick, confirm the decision in one line and move on.

---

## Authoring an ADR

An ADR captures an architectural decision. **Decision is decision.** Once written down, what the ADR records is the choice — not the schedule for acting on it.

Don't write gate language into ADR bodies — *"ratify after X happens,"* *"status remains proposed through Y wraps,"* *"accept after the implementing gaps' resolution shapes prove the contract works in practice."* Either you're committed (ratify it via `aiwf promote ADR-NNNN accepted`) or you're not (keep it `proposed` and let the conversation continue). When to *act on* the decision — what milestone, in what sequence, gated on what — is a planning concern that lives in the planning surface (`aiwf status`, the whiteboard, the rituals plugin's planning skills), not in the ADR body.

The prior pattern (gate language in ADR bodies, sometimes backed by bespoke `internal/policies/` tests pinning status) conflated *"what's the decision?"* with *"are we ready to act on it?"*. Reviewers and LLM agents reading the ADR couldn't tell whether the gate was decision rationale ("we'll ratify if the design holds up") or schedule artifact ("we'll ratify when the epic closes"). The clean separation: **ADR captures the choice; planning sequences the action.**

The FSM (`proposed → accepted | rejected`; `accepted → superseded`) and `aiwf promote` are the only mechanical surfaces that should constrain ADR status transitions. No bespoke per-ADR test pins. Sovereign override (`--force --reason`) remains available when an exceptional ratification path is genuinely needed.

---

## What aiwf commits to

These are the load-bearing properties any change must preserve. They are distilled from the research arc and recorded in [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md).

1. **Six entity kinds** — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for legal transitions. Hardcoded; not driven by external YAML.
2. **Stable ids that survive rename, cancel, and collision.** Every kernel id kind emits at a uniform canonical 4-digit width (per ADR-0008) — epics, milestones, gaps, decisions, contracts, ADRs, and findings all follow the same `<prefix>-NNNN` shape. Parsers tolerate narrower legacy widths on input so pre-migration trees, branches, and commit trailers continue to validate without history rewrite; renderers and allocators always emit canonical width. Consumers carrying narrow-legacy trees migrate via `aiwf rewidth --apply` (one commit, idempotent, archive entries preserved per forget-by-default). The id is the primary key; the slug is just display. Renames preserve the id. "Removal" means flipping status to a terminal value, not deleting the file. Collisions are detected by `aiwf check` and resolved by `aiwf reallocate`.
3. **`aiwf check` runs as a pre-push git hook.** Validation is the chokepoint. The hook is what makes the framework's guarantees real; without it, skills are just suggestions.
4. **`aiwf history <id>` reads `git log`.** No separate event log file. Structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable.
5. **Marker-managed framework artifacts in the consumer repo, regenerated only on explicit `aiwf init` / `aiwf update`.** Skills under `.claude/skills/aiwf-*` (gitignored) and git hooks under `.git/hooks/<hook>` (untracked, identified by an `# aiwf:<hook>` marker so user-written hooks are left alone). `aiwf update` is the upgrade verb — it refreshes every artifact the consumer is opted into. Stable across `git checkout` by design.
6. **Layered location-of-truth.** Engine binary lives external (machine-installed via `go install`). Per-project policy and planning state live in the consumer repo. Materialized skill adapters live in the consumer repo but are gitignored.
7. **Every mutating verb produces exactly one git commit.** That gives per-mutation atomicity for free. A failed mutation aborts before the commit.
8. **Acceptance criteria as namespaced sub-elements of milestones; TDD opt-in per milestone.** ACs are not a seventh kind — they're structured sub-elements addressed by composite id `M-NNN/AC-N`, validated by `aiwf check`, with the audit rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`.
9. **Principal × agent × scope provenance.** The kernel separates *who is accountable* (principal, always human) from *who ran the verb* (operator/actor, may be LLM or bot). Authorized agent work is gated by a typed scope FSM (`active | paused | ended`) opened with `aiwf authorize`. `--force` requires a human actor — sovereign acts always trace to a named human. Identity is runtime-derived from `git config user.email`, not stored in `aiwf.yaml`.
10. **Uniform archive convention for terminal-status entities** (per [ADR-0004](docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)). Every kind stores terminal entities under a per-parent `archive/` subdirectory — `work/gaps/archive/`, `work/decisions/archive/`, `work/contracts/archive/`, `work/epics/archive/`, `docs/adr/archive/` — so the active directory listing reflects what is currently in-flight without filter ceremony. Movement is decoupled from FSM promotion: `aiwf promote` and `aiwf cancel` flip status only; `aiwf archive` sweeps qualifying entities into their archive subdirs as a single commit per invocation. The loader resolves ids across active and archive, so cross-references stay live indefinitely. Reversal is deliberately absent — file a new entity referencing the archived one. Drift is policed via the `archive-sweep-pending` advisory finding, with an opt-in `archive.sweep_threshold` knob in `aiwf.yaml` that flips the finding to blocking past the named count.

If a proposed change does not preserve one of these, treat it as a kernel-level decision and surface it explicitly — not a quiet refactor.

---

## What is *not* in scope

Not in scope, deliberately. None of these blocks aiwf value; each can be added later when real friction demonstrates the need.

- An events.jsonl file or any append-only event log.
- A graph projection file or hash chain.
- A monotonic ID counter coordinated across branches.
- A module system or capability registry.
- Multi-host adapter generation (aiwf targets Claude Code only).
- A third-party skill registry.
- Tombstones beyond "status = cancelled / wontfix / rejected / retired."
- CRDT primitives, custom merge drivers, server-side hooks.
- GitHub Issues or Linear sync.
- Full FSM-as-YAML.

If you find yourself reaching for any of the above to solve a problem, stop and check [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) §"What's deliberately not in the PoC" — there's almost certainly a simpler way.

---

## How to validate changes

```bash
go test -race ./...                  # unit tests
golangci-lint run                    # linters
go build -o /tmp/aiwf ./cmd/aiwf     # binary builds
```

All three should pass before committing. CI runs all of them on every push.

---

## Operator setup

After cloning, install the framework's companion plugins **for this project's scope** so the planning skills (`aiwfx-start-milestone`, `wf-tdd-cycle`, etc.) and role agents activate in this repo. Without them, `aiwf` is just the planning data layer and `aiwf doctor` warns (per M-070's recommended-plugin check). The expected set is declared in [`aiwf.yaml`'s `doctor.recommended_plugins`](aiwf.yaml).

In a Claude Code session at this repo's root:

```
/plugin marketplace add 23min/ai-workflow-rituals
/plugin                     # Discover tab → install each at PROJECT scope
/reload-plugins
```

Install both `aiwf-extensions@ai-workflow-rituals` and `wf-rituals@ai-workflow-rituals`. **The CLI form `claude /plugin install <name>@<marketplace>` defaults to *user* scope** — only the interactive `/plugin` menu offers a project-scope choice. Verify with `aiwf doctor`: once both are project-scope-installed, the `recommended-plugin-not-installed:` warnings go silent. (Closes G-064 via M-071, which lives under E-18.)

---

## Working in this repo

The historical sessions and iterations are archived at [`docs/pocv3/archive/poc-plan-pre-migration.md`](docs/pocv3/archive/poc-plan-pre-migration.md). Forward work tracks via epic + milestone entities under `work/`; allocate via `aiwf add epic` / `aiwf add milestone` and run `aiwf status` to see in-flight state.

Trunk-based development on `main`: commit directly, no PR ceremony. Conventional Commits subjects are still useful (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs: ...`).

When in doubt: the smaller change is the right change.

---

## AC promotion requires mechanical evidence

Before `aiwf promote M-NNN/AC-<N> met`, there must be a mechanical assertion that fails if the AC's claim breaks — a Go test under `internal/policies/`, a kernel finding-rule, or a fixture-validation script. *"I read the file and it looks right"* is not evidence; it makes the AC's correctness depend on the reviewer's recall, which is exactly the dependency *"framework correctness must not depend on the LLM's behavior"* forbids.

This applies **even to milestones with `tdd: none`**. The `tdd:` policy controls whether the kernel's `acs-tdd-audit` finding fires; it does not waive the test-discipline obligation. For doc-shaped ACs (ADR content, skill body content), the test is typically a structural assertion on a named markdown section — per Go conventions §"Substring assertions are not structural assertions" below, scope the assertion to the section, don't grep flat over the file.

The chokepoint is the AC-promote command. Discipline is the chokepoint until a kernel finding-rule lands that polices test-existence per AC.

---

## Cross-repo plugin testing

When a milestone's deliverable is a `SKILL.md` (or other content) that lives in the rituals plugin repo at `/Users/peterbru/Projects/ai-workflow-rituals/` (distributed via the Claude Code marketplace), the **canonical authoring location during the milestone is a fixture in this repo** at `internal/policies/testdata/<skill-name>/SKILL.md`. AC tests under `internal/policies/` assert content claims against the fixture; red→green TDD iteration happens against it.

At wrap, the fixture content is copied into the rituals repo as a separate commit there; the wrap-side spec records the rituals-repo commit SHA in *Validation*. A drift-check test in this repo compares the fixture against the local marketplace cache (`~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`) and fires if they diverge — and skips cleanly when the cache is absent (CI without a plugin install).

Subtree and submodule are wrong for this: the rituals repo is the upstream, and vendoring it here would invert the relationship and add CI/contributor friction. **No tests live in the rituals repo** — it stays pure markdown.

---

## Go conventions

These rules apply to all Go code in the module. The repo-wide engineering principles above (KISS, YAGNI, no half-finished implementations, errors-as-findings) cascade in on top of these.

### Formatting and linting

- **`gofumpt`** is the formatter. Run via `golangci-lint`, no separate install. Anything `gofumpt`-clean is also `gofmt`/`goimports`-clean.
- **`golangci-lint`** is the only linter. Enabled set: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, `revive`, `gosec`, `bodyclose`, `unconvert`, `misspell`, `gofumpt`, `goimports`. Config in `.golangci.yml` at repo root.
- CI fails on any lint finding. No `//nolint` directives without a one-line rationale comment.

### Testing

- **`testing` (stdlib) + `github.com/google/go-cmp`** for comparison-heavy assertions. No testify, no assertion DSLs.
- **Table-driven** when ≥2 cases exercise the same function. Single-case tests stay flat.
- **Subtests via `t.Run(name, ...)`** for each table case.
- **Golden files** under `testdata/` for snapshot assertions. Synthetic content only — fixtures must read as obviously fictional, not as anonymized copies of real projects.
- **Race detector on every CI run:** `go test -race ./...`.

#### Test the seam, not just the layer

When a new helper, package, or shared function is wired into an existing caller (verb, dispatcher, hook), the test set must cover **both** the helper's behavior *and* the seam where it integrates. A unit test of the helper alone is necessary but not sufficient — it doesn't catch the case where the caller has a parallel source of truth and never adopts the helper.

Concrete shape: for a new verb-level helper, write at least one test that drives the verb's dispatcher (`run([]string{"<verb>", ...})`) and asserts the output reflects the helper's contract. For a check-rule helper, write a fixture-tree test that exercises the rule through `check.Run`. Test names should make the seam explicit (`TestRunVersion_UsesBuildInfoFallback`, not just `TestResolvedVersion`).

When a verb's output depends on values that only exist in a real binary — `runtime/debug.ReadBuildInfo`, `-ldflags`-stamped globals, `os.Args[0]`, `os.Executable()` — a unit test running under `go test` cannot exercise the production path. Add a binary-level integration test that builds the cmd to a tempfile and runs it as a subprocess: `go build -o $TMP/aiwf ./cmd/aiwf && exec.Command($TMP/aiwf, "version")`. The cost is a few seconds per CI run; the alternative is the bug shipping.

Why this rule exists: v0.1.0 shipped with `aiwf version` returning `"dev"` even though the new `version.Current()` helper returned the correct buildinfo value. The unit test of `version.Current()` was clean. The verb still printed an unrelated package-global. Two parallel sources of truth coexisted; tests covered only the new one.

#### Contract tests for upstream-cached systems

For any external system with caching semantics — HTTP proxies (the Go module proxy is the canonical example), DNS, CDN-fronted APIs — tests must pin "did we ask the right question," not just "did we parse the answer correctly."

Concrete shape: a real-system integration test (gated under `-short` so CI without network can skip) that derives the expected value through an **independent** code path, not from the same endpoint the implementation uses. For the module proxy this means: if the implementation resolves "latest" via `/@v/list`, the test independently fetches `/@v/list`, computes the expected highest semver, and asserts the implementation returns the same value. A test that just asserts "the implementation returned a non-empty version" is parsing-coverage, not resolution-correctness.

When you discover the right endpoint by reading the upstream tool's source (e.g., the Go toolchain's resolver), document that decision in a comment at the call site so future readers don't re-litigate the choice.

Why this rule exists: v0.1.0's `version.Latest()` queried the proxy's `/@latest` endpoint, which is cached separately from `/@v/list` and can serve stale pre-tag pseudo-versions for hours after a tag lands. The unit tests served whatever JSON the implementation expected and never asked whether the chosen endpoint was the right one. The Go toolchain uses `/@v/list`-first for exactly this reason — documented behavior we re-learned by failing in production.

#### Spec-sourced inputs for upstream-defined input spaces

When test cases enumerate an upstream-defined input grammar — semver shapes, RFC fields, error-code families, on-disk format variants — the test must cite the spec and cover the full enumerated space, not "the example I had in mind."

Concrete shape: prefix the test data with a comment pointing at the canonical spec (e.g., `// per https://go.dev/ref/mod#pseudo-versions`), then list every case the spec defines. If you cannot cite a single source for the input space, the space isn't pinned and the tests are example-driven; either find the spec or document the omission explicitly as a known limitation.

Why this rule exists: v0.1.0's pseudo-version regex initially only matched the basic `v0.0.0-DATE-SHA` form. The Go module spec defines three shapes (basic, post-tag, pre-release-base); VCS stamping adds the `+dirty` suffix. Smoke tests caught the gaps mid-implementation. A spec-sourced test pass at design time would have exercised all four cases on the first commit.

#### Substring assertions are not structural assertions

A test that greps for a literal in rendered output (HTML, Markdown, JSON) proves the literal exists *somewhere*. It does not prove the literal is in the right *place*. The right place is what the user (or the next renderer) actually consumes; the literal floating in the wrong section is still a bug that ships.

Concrete shape:
- For HTML output, parse the document with `golang.org/x/net/html` (or an equivalent) and assert presence inside the named `<section>`, attribute, or descendant chain. A standalone substring match is acceptable only when the value is unique and the location is irrelevant (e.g., a stable token in a JSON envelope).
- For markdown output, walk the heading hierarchy and assert the prose appears under the expected section, not just on the page.
- For multi-tab / multi-section pages, every substring assertion must name *which* section it expects the value in. "AC anchor exists" is not enough; "AC anchor exists inside `data-tab=manifest`" is.

If the literal under test is short or generic enough to plausibly appear in unrelated places (e.g. an id="ac-1" attribute, the word "strict", a status name like "active"), assume it does and use a structural assertion.

Why this rule exists: I3 step 5 shipped milestone-page tests that asserted `id="tab-overview"`, `href="#ac-1"`, `policy-strict">strict` etc. as plain substring matches. Two of those would have passed even with the AC rendered in the wrong tab, the policy badge swapped, or the anchor wired backwards. The user caught this in audit; the tests were structurally weak from the start.

#### Render output must be human-verified before the iteration closes

Test suites pin code correctness — they do not pin *feature* correctness. For UI / rendered output (HTML pages, generated docs, status outputs that the user reads), running the binary against a real fixture and visually inspecting the result is part of "done," not an optional follow-up. A green test suite says "no regressions in what we asserted"; only a manual look says "the page actually communicates what it should."

Concrete shape:
- Before claiming a render-iteration step closed, render against a non-trivial real fixture (the kernel repo's own planning tree is the canonical one), open the result, exercise the interactive surface (every tab, every link, every conditional content path), and only then mark the step done.
- If you cannot run the binary in your environment (sandbox, CI-only), say so explicitly to the user instead of declaring success — the tests do not stand in for that pass.
- An end-to-end golden snapshot of one full page (HTML byte-equal to a known-good fixture) is a good auxiliary safety net, but it doesn't replace the human look-through. Snapshots only catch *changes*; they don't catch "this was wrong on day one."

Why this rule exists: I3 step 5 shipped six milestone tabs with placeholder Build/Tests/Provenance content paths that no test exercised. The rendered output was never opened in a browser. A green `go test ./...` was treated as completion; the user's audit caught the gap.

#### Test untested code paths before declaring code paths "done"

When a function has a branch (a `switch`, an `if`, a filter), every reachable branch must have a test that traverses it — or the branch must be marked `//coverage:ignore` with a one-line rationale. "Tests pass" with code paths not exercised is "tests pass for the paths I happened to think about."

Concrete shape:
- Before committing a feature, run `go test -coverprofile=cov.out ./<pkg>/...` and skim the uncovered lines. Each uncovered line is either: (a) a missing test (write it), (b) defensive code that can't fire (delete it), or (c) genuinely unreachable in production (mark it `//coverage:ignore <reason>`).
- For typed view-builders that filter or branch on input (e.g. "is this a phase event?", "does this commit have an authorize trailer?"), the test set must include at least one input that takes each branch. A fixture with no scopes, no phase events, no force trailers exercises only the empty-state branch — that's not coverage of the populated path.
- When the package gains a new typed input (a new trailer, a new field on a struct), audit the consumers' branches the same way: which call sites now have an unexercised arm? Write the missing test before the next commit.

Why this rule exists: I3 step 5's `phaseEventsFromHistory`, `firstTestsTrailer`, `provenanceFor`, and `linkedEntitiesFor` were all wired in but never exercised by any test fixture that produced phase history, test trailers, scopes, or cross-kind references. The functions could have returned wrong shapes silently and nothing would have failed.

#### Don't paper over a test failure — root-cause it

When a test fails in a way that doesn't match its premise, the failure is information about the system, not about the test. Working around it (changing the test setup until it passes, adding manual git commits the production path doesn't make, sleeping until a race resolves) leaves the original signal unread. The test now passes for a reason other than what it was supposed to verify.

Concrete shape:
- If a test fails with a state error (lock contention, projection mismatch, "not found"), the first action is to dump the state at the point of failure (`t.Logf` the on-disk content, the trailer set, the lock holder) and read the actual cause. Only after you understand it should you decide whether the test or the production code needs to change.
- A "manual git commit to keep things clean" inside a test is a yellow flag — the production verb is not making that commit and the user won't either. Either the verb should make it (production bug) or the test fixture should be set up so it isn't needed (test bug); not both.
- "Workaround applied; investigation deferred" comments are owed an issue / gap entry; otherwise they accumulate as silent debt.

Why this rule exists: I3 step 2A's `TestRun_AddACWithTestsFlag` originally hit a verb error after a hand-edit; I added a manual `git add -A && git commit` to make the test pass without diagnosing why the verb's projection ran in the wrong direction. The test now passes for a different reason than its assertion claims.

#### Policy tests that read entity files must resolve via the loader

When a test under `internal/policies/` (or any test that reads a live planning-tree entity) needs the file path of an epic, milestone, gap, decision, contract, or ADR, the path must come from the loader — `tree.Load(ctx, root)` + `Tree.ByID(id)` + `entity.Path`. **Never** hardcode a literal path like `filepath.Join(root, "work", "epics", "E-NNNN-...", "M-NNNN-....md")`.

The reason is ADR-0004 (uniform archive convention): `aiwf archive --apply` moves terminal-status entities into a per-kind `archive/` subdirectory, and the loader resolves ids transparently across active and archive. A test that hardcodes the active-tree path appears to work right up until the parent entity reaches terminal status and the next sweep moves the file — at which point the test fails with a confusing file-not-found error inside a pre-commit policy hook, blocking the very `aiwf archive` commit that should have been routine.

Concrete shape:

```go
tr, _, err := tree.Load(context.Background(), root)
if err != nil { t.Fatalf("tree.Load: %v", err) }
e := tr.ByID("M-0090")
if e == nil { t.Fatal("M-0090 not found") }
specPath := filepath.Join(root, e.Path)  // active or archive, transparent
```

The chokepoint is `PolicyNoHardcodedEntityPaths` in `internal/policies/`. It scans Go source under that package for `filepath.Join(...)` calls whose string-literal args match the entity-slug shape (`E-/M-/G-/D-/C-/ADR-` followed by digits and a dash). Any such call fails CI with a finding pointing at the offending line and citing the loader-based resolution above. The policy's scope is `internal/policies/*.go` — that's where this bug class lives in practice; if the pattern leaks into other packages, widen the scope in one commit rather than chasing it per-file.

Why this rule exists: M-0090's first archive sweep aborted because `TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap` read the milestone spec via a hardcoded `filepath.Join` literal that the sweep invalidated. The test had passed every commit up to that point — it broke the instant the milestone's parent epic became archive-eligible. ADR-0004's whole point is that archive movement should be transparent; a test that opts out of the loader opts out of that guarantee.

### Coverage

- **High coverage on `internal/...` packages.** PoC target is 90%; failing checks for low coverage are advisory at this stage.
- **Exclusions** (intentionally small):
  - `cmd/aiwf/main.go` — covered by integration tests against the binary, not unit tests.
  - Generated code.
  - Specific lines marked `//coverage:ignore <reason>`.
- The PoC is small enough that 100% coverage on internal packages is realistic; aim for it but don't block on it.

#### Beyond line coverage: fuzz, property, and mutation testing

Line coverage measures *what code runs under the test suite*; it does not measure whether the assertions are strong enough to catch a regression. Three additional layers complement coverage on the load-bearing paths:

- **Fuzz tests (G44 item 1)** — `Fuzz*` functions in `internal/{entity,gitops,version,pathutil}/` exercise high-value parsers (Slugify, Split, parseTrailers, Parse, Inside) against arbitrary input. Seed corpora run as part of the routine `go test`; full fuzzing (`-fuzztime=2m` per target) runs on the `fuzz` workflow (`workflow_dispatch` + weekly cron). New corpus seeds discovered by fuzzing belong under `testdata/fuzz/Fuzz<Name>/` and get committed alongside any related fix.
- **Property tests (G44 item 2)** — `internal/entity/transition_property_test.go` asserts the FSM closed-set invariants exhaustively (state-set agreement, terminality, reachability, totality of `ValidateTransition`). The drift-prevention follow-up (`PolicyFSMInvariants` in `internal/policies/`) catches "new Kind without FSM wiring" and "FSM cycle introduced" — invariants the original tests miss because their iteration source is also their test target.
- **Mutation testing (G44 item 3)** — `mutate-hunt` workflow runs gremlins against a chosen package pattern. **Workflow_dispatch only**; never on push. Use before tagging a release or after a substantive test-suite change. Read survivors carefully: equivalent-mutant noise (e.g., `a > b` and `a >= b` after a `a != b` guard) and unreachable-branch noise are common false positives — don't chase them. Real survivors are concrete file:line entries that warrant either a new test or a refactor that eliminates the mutation site. Required tuning for this repo: `--workers 1` (default parallelism contends on the test-binary build cache and times out) and `--timeout-coefficient 15`.

### Error handling

- Wrap every error returned across a function boundary with context: `fmt.Errorf("loading frontmatter from %s: %w", path, err)`.
- Compare errors with `errors.Is(err, ErrFoo)` and `errors.As(err, &target)`. Never `err == ErrFoo` except for sentinels you own.
- Sentinel errors (`var ErrNotFound = errors.New("not found")`) for stable conditions. Typed errors (`type ValidationError struct{...}`) when the error carries data.
- **Library code never panics or `os.Exit`s.** Only `cmd/<tool>/main.go` calls `os.Exit`.

### Concurrency

- Pass `context.Context` as the first argument of every IO-touching function.
- Use `context` for cancellation only. Don't stuff request-scoped data via `context.WithValue` except across API boundaries.
- Never hold a mutex across an IO call.

### CLI conventions

Every binary (currently just `aiwf`) follows:

- **Framework:** [`github.com/spf13/cobra`](https://github.com/spf13/cobra) is the standard CLI library. Each verb is a `cobra.Command`; a `runXCmd(...)` helper holds the verb's body and is called from `RunE`. Closed-set flag values bind to completion via `cmd.RegisterFlagCompletionFunc(name, cobra.FixedCompletions(...))`; entity-id flags use the dynamic `completeEntityIDFlag(kind)` helper. New commands are added under `cmd/aiwf/` and wired from `newRootCmd`. The drift test in `cmd/aiwf/completion_drift_test.go` is the chokepoint — a flag added without completion wiring (or an entry in the opt-out list) fails CI.
- **Exit codes:** `0` ok, `1` findings (validation succeeded but reported issues), `2` usage error, `3` internal error. Verbs return `int`; the Cobra `RunE` adapter wraps the value via `wrapExitCode` and `run()` translates it back through the `*exitError` typed shuttle.
- **Output:** Human-readable text by default; `--format=json` emits a structured JSON envelope for CI scripts and downstream tools. `--pretty` (with `--format=json`) indents the envelope. `aiwf` is an interactive CLI first; the JSON shape is the secondary surface.
- **JSON envelope:** `{ tool, version, status, findings, result, metadata }`. `status` is one of `ok`, `findings`, `error`. `findings` is an array (possibly empty); `result` carries the verb's payload; `metadata` carries timing, counts, and the calling correlation_id when present.
- **Logging:** `log/slog` to stderr (default level `INFO`). Tool output goes to stdout. `fmt.Fprintln(os.Stderr, …)` is not a substitute.
- **Flags:** `--help`, `--version`, `-v`, `--pretty` plus verb-specific. No global config files; everything via flags, env, or `aiwf.yaml` at the consumer repo root.
- **No package-level mutable state.** Pass dependencies via struct fields. In particular, don't introduce production patterns purely to satisfy test-injection — if tests need to swap a dependency, the production code uses constructor injection (`func New(deps Deps) *T`); never a package-level `var registry = map[…]` that tests mutate.

### Commit conventions

Every mutating verb writes a structured trailer in its commit message so `aiwf history` can render per-entity timelines:

```
aiwf-verb: promote
aiwf-entity: M-001
aiwf-actor: human/peter
```

Commit subject lines follow Conventional Commits (`feat(plan): ...`, `chore(plan): ...`, `docs(adr): ...`).

### Release process

Releases of `aiwf` are git tags on `poc/aiwf-v3` of the form `vX.Y.Z`. The Go module proxy resolves them when a consumer runs `aiwf upgrade` or `go install <pkg>@latest`. There is no separate release artifact to publish, but the user-facing changelog must stay in step.

Before tagging `vX.Y.Z`:

1. In a single release-prep commit, edit [`CHANGELOG.md`](CHANGELOG.md):
   - Rename the `## [Unreleased]` heading to `## [X.Y.Z] — YYYY-MM-DD`.
   - Add a fresh empty `## [Unreleased]` heading at the top (above the new version section).
   - Verify the moved entries summarize the user-visible delta — gaps closed, verbs added, behavior changes. Internal refactors that change nothing observable can be omitted.
2. Use commit subject `release(aiwf): vX.Y.Z`.
3. Push the commit, then `git tag vX.Y.Z` pointing at it, then `git push origin vX.Y.Z`.

Skipping the changelog edit means the tag-push CI check fails: the workflow at [`.github/workflows/changelog-check.yml`](.github/workflows/changelog-check.yml) verifies that every pushed `v*` tag is reachable from a commit whose `CHANGELOG.md` contains a matching `## [X.Y.Z]` heading. Per the kernel's "framework correctness must not depend on the LLM's behavior" rule, the check is the guarantee — the human-facing rule above is just the convenient version.

Patch releases that are pure-mechanical (e.g. a `go.sum` refresh with no behavior delta) still require a CHANGELOG entry, even if it is a single line saying "no functional changes" — the workflow does not distinguish empty from missing.

### Dependencies

- Minimize external deps. Each new dep needs a one-line justification in the commit message or PR description.
- `CGO_ENABLED=0` — binaries must be statically linked.
- `go 1.24` minimum (consumer floor in `go.mod`). Bump deliberately. Last bumped from 1.22 → 1.24 in G43; rationale on file.
- **CI build toolchain is decoupled** from the `go.mod` floor: the workflow files pin `go-version: "1.25"` so `actions/setup-go` resolves to the latest patched 1.25.x, picking up stdlib backports the 1.24.x line lacks. The directive in `go.mod` is the consumer-compatibility commitment; the CI version is the build-time toolchain. Bump CI when `govulncheck` reports stdlib CVEs only fixed in a newer minor; bump `go.mod` only when the codebase needs a feature that `go.mod`'s current floor doesn't provide.
- One `go.mod` for the entire module.

### Naming

- Package names: short, lowercase, no underscores, no plurals (`entity` not `entities`).
- Avoid stuttering: `entity.Entity` is wrong; `entity.Load` or `entity.Result` is right.
- Exported identifiers must have a doc comment starting with the identifier name.
- Acronyms stay capitalized: `parseURL`, `httpClient`, `jsonOut` — not `parseUrl`.

### Type design

- **Closed-set enums ship only used values.** When defining a closed set of constants or enum values (status, kind, action), ship only the values that have a current call site. Speculative future values violate YAGNI even when they're "just constants."
- **The PoC's six kinds** (epic, milestone, ADR, gap, decision, contract) and their **status sets** are hardcoded in Go for the PoC. They are intentionally not driven by external YAML — that move is deferred until a real consumer needs to customize the vocabulary.

### Designing a new verb

Before adding a verb to `cmd/aiwf/`, the design isn't done until you can answer **"what verb undoes this?"** Acceptable answers:

- *Another invocation of the same verb with different inputs.* Most state-transition verbs reverse this way (e.g. `aiwf promote E-01 active` undoes `aiwf promote E-01 done` if the kind allows it).
- *An explicit terminal-state transition.* `aiwf cancel`, `aiwf reallocate` (renumbers; the old id's history terminates with the rename event).
- *"You can't, and that's deliberate — here's why."* `aiwf init` is one-shot; `aiwf import` for already-present ids needs `--on-collision`. The reason gets written down.
- *"You'd open a new entity for the inverse."* Bug-fix-style reversals (e.g., add a hotfix milestone) belong here.

Not acceptable: *"we'll figure that out later"* — the verb isn't ready. See [docs/pocv3/design/design-lessons.md](docs/pocv3/design/design-lessons.md) §"On reversal" for the principle this comes from.

### Skills policy

Every top-level Cobra verb is reachable through some AI-discoverable channel. The shape of that coverage follows four cases — **per-verb skill** (default for mutating verbs that carry decision logic), **topical multi-verb skill** (when users reach for the concept rather than the verb; precedent: `aiwf-contract`), **no skill** (when `--help` plus tab-completion fully cover the surface; e.g. `aiwf version`, `aiwf init`), and **discoverability-priority split** (when a topical group's prompt shapes diverge enough to dilute one bundled description; precedent: `aiwf-list` and `aiwf-status`).

The judgment rule lives in [ADR-0006](docs/adr/ADR-0006-skills-policy-per-verb-default-topical-multi-verb-when-concept-shaped-no-skill-when-help-suffices.md). The mechanical companion is [`internal/policies/skill_coverage.go`](internal/policies/skill_coverage.go) — it asserts that every verb has either a same-named `aiwf-<verb>` skill or an entry in `skillCoverageAllowlist` with a one-line rationale, that every embedded skill carries valid `name:` and `description:` frontmatter, and that every backticked `` `aiwf <verb>` `` mention inside a skill body resolves to a registered top-level verb. CI fails any future PR that adds a verb without satisfying both bars.

When designing a new verb, answer the *what verb undoes this?* question above first, then *which case from ADR-0006 applies, and where does this verb's skill live?*

### What's enforced and where

The kernel's "framework correctness must not depend on LLM behavior" principle applies here too: the rules below are enforced by tooling at named chokepoints, not by remembering to tick a checklist. This section names the chokepoint for each rule so a contributor (human or LLM) can see what will block a bad commit and what is still advisory.

| Rule                                                         | Chokepoint                                                       | Status                  |
|--------------------------------------------------------------|------------------------------------------------------------------|-------------------------|
| `gofmt` / `goimports` / `gofumpt` clean                      | `golangci-lint run` (formatters block) — CI `lint` job           | Blocking via CI         |
| Lint set passes (`errcheck`, `govet`, `staticcheck`, …)      | `golangci-lint run` — CI `lint` job                              | Blocking via CI         |
| `go vet` clean                                               | `go vet ./...` — CI `vet` job                                    | Blocking via CI         |
| Tests pass with race detector                                | `go test -race ./...` — CI `test` job                            | Blocking via CI         |
| Build succeeds (`CGO_ENABLED=0`)                             | `go build` — CI `build` job                                      | Blocking via CI         |
| End-to-end verb regressions                                  | `aiwf doctor --self-check` — CI `selfcheck` job (G9)             | Blocking via CI         |
| Vulnerable transitive deps                                   | `govulncheck ./...` — CI `vuln` job (G43)                        | Blocking via CI         |
| Library code does not `panic` or `os.Exit`                   | `forbidigo` (G43)                                                | Blocking via CI lint    |
| Test helpers call `t.Helper()`                               | `thelper` (G43)                                                  | Blocking via CI lint    |
| Errors compared with `errors.Is`/`As`, wraps use `%w`        | `errorlint` (G43)                                                | Blocking via CI lint    |
| Planning-tree shape (no stray files under `work/`)           | `aiwf check --shape-only` — pre-commit hook (G41)                | Blocking pre-commit     |
| Full planning-tree validation (refs, ids, FSM, contracts)    | `aiwf check` — pre-push hook                                     | Blocking pre-push       |
| Repo-specific invariants (trailer keys, sovereign acts, etc.) | `internal/policies/` — runs as a Go test package                 | Blocking via CI test    |
| Every verb has skill coverage or an allowlist entry; every `aiwf <verb>` mention in a skill resolves | `internal/policies/skill_coverage.go` — runs as a Go test (M-074) | Blocking via CI test    |
| `context.Context` as first arg of new IO function            | Code review                                                      | Advisory                |
| No new package-level mutable state                           | Code review                                                      | Advisory                |
| Each new dep has a one-line justification                    | Code review (commit message / PR description)                    | Advisory                |
| Mutating-verb commits carry `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers | `internal/policies/trailer_keys.go` + the `principal_write_sites` policy + the untrailered-entity audit (G24, G31, G32) | Blocking via CI test    |
| Bumping the Go floor in `go.mod`                             | Deliberate decision; document rationale in commit message        | Advisory                |

The four advisory lines are the items where mechanical enforcement is either too noisy (context.Context first arg — generics make a literal regex unreliable), too contextual (package-level mutable state — sometimes legitimate behind a guard), or self-policing (dep justification, deliberate floor bumps). Reviewers and the `Go conventions` section above are the chokepoint there.

If a blocking rule needs to be relaxed for a specific call site, route it through the linter's allowlist (e.g., `forbidigo` exclusion for the verb/apply.go re-panic site) with a one-line rationale, not a `//nolint:rule` directive without explanation.
