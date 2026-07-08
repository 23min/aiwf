# CLAUDE.md — ai-workflow repo

This repo carries `aiwf` — a small framework that helps humans and AI assistants track what's planned, decided, and done, by validating mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) for what aiwf commits to and §"What's deliberately not in the PoC" for scope limits. Gaps live as entities under `work/gaps/` — run `aiwf list --kind gap` or `aiwf show G-NNNN`.

---

## Engineering principles

- **KISS.** Prefer the boring solution. A few similar lines beat a premature abstraction. Avoid cleverness (reflection, metaprogramming, deep generics, control-flow tricks) unless the simple version is demonstrably worse.
- **YAGNI.** No speculative interfaces, no "might need it later" knobs, no plugin systems for one implementation. Add the second case when it shows up; abstract on the third.
- **No half-finished implementations.** A feature that lands, lands tested. Stubs and TODOs in shipped code are a smell.
- **Errors are findings, not parse failures.** `aiwf check` loads inconsistent state and reports it; it never refuses to start. Validation is a separate axis from loading.
- **Framework correctness must not depend on LLM behavior.** Skills are advisory; the pre-push hook and `aiwf check` are authoritative. A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee.
- **Kernel functionality must be AI-discoverable.** Every verb, flag, JSON field, body-section name, finding code, trailer key, and YAML field is reachable via `aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, this file, or the cross-referenced design docs. If an AI must grep source to learn a capability, it's undocumented — ship `--help` + skill docs alongside the implementation.
- **CLI surfaces must be auto-completion-friendly.** Every verb, subverb, flag, and closed-set value is reachable via tab-completion (Cobra `ValidArgs` / `RegisterFlagCompletionFunc`; entity ids enumerate dynamically). The drift test in `internal/policies/` fails CI on a verb/flag added without completion wiring.
- **Provenance is principal × agent × scope, not just operator.** When the human directs the LLM in conversation, the LLM is a *tool* (human = principal, no co-actor inflation). When the human runs `aiwf authorize E-NN --to ai/claude`, the agent works within that scope until the scope-entity is terminal or the human pauses. `--force` is sovereign — humans only. See [`docs/pocv3/design/provenance-model.md`](docs/pocv3/design/provenance-model.md).
- **Never emit a malformed or fabricated id-shaped token in committed prose.** No letter/placeholder suffixes (`M-a`, `M-alpha`, `M-NNNN`), no canonical-width fabrications for entities that don't exist, no pseudo-formal sequence labels ("Phase 1", "alpha/beta"). In planning conversations, short numeric labels (`M-1`, `M-2`) are allowed shorthand for not-yet-allocated milestones. Wrap any id-shape in backticks when discussing syntax rather than referencing a real entity; the `body-prose-id` check enforces this.

---

## Working with the user

- **Q&A / one decision at a time.** When the user says "Q&A"/"interview me", or you have ≥3 queued decisions, present them **one at a time**: context, pros/cons, risks, your plain lean, then a numbered option list (incl. "something else"). Wait for the pick, confirm in one line, move on.
- **Never suggest the user pause.** Don't propose pausing, stopping, banking progress, or resuming later. When to stop is the user's call. If continuing carries real risk, state it plainly and keep going / ask what's next — surface the information, don't own the decision to stop.
- **Gate discipline survives compaction.** Every mutating action — commit, push, merge, promote, archive, cancel, tag, `gh pr create`, branch delete — is its own gate; prior approvals never carry forward, including across `/compact`. Don't collapse several mutations into one `AskUserQuestion` ("Yes — commit + push + merge" is the wrong question); one action per gate. The sanctioned exception is the **declared-sequence gate**: a single gate MAY cover a sequence of **local, reversible** mutations at one moment *provided the gate enumerates every action verbatim* (the user can then approve a subset). Batchable: promotes, an `aiwf archive` sweep, a local merge to mainline, a tracker-closure `aiwf promote G-NNNN addressed --by-commit <sha>`, local branch/worktree deletion. Never batch **outward / irreversible** actions — push, `gh pr create`, tag-push, remote-branch delete, `--force` (each its own gate; push is the only action that leaves the machine; `--force` is additionally human-only). Never batch **timing-bearing** mutations whose signal *is* their timestamp — `tdd: required` AC phase promotes fire live, and (per the shipped guidance fragment) live also means ungated by default: no per-promote approval ask, only the wrap review and the push as control points. Any deviation (conflict, finding, unexpected dirty state, an action not on the list) aborts and re-gates. Canonical instance: the `wf-patch` wrap; the `aiwfx-wrap-*` rituals use the same gate for their terminal local sequence (local merge + promote-done + cleanup — promote lands last so a delegated scope stays live for the merge commit, per G-0119; push excluded).
- **Finish in-context, don't paper over.** When you notice a closely-related issue in text you're editing — especially one you just authored — fix it inline. File a separate gap only when the issue is *architecturally distinct*. Surface what you noticed plainly; don't silently defer.

---

## Authoring an ADR

**Decision is decision.** An ADR records the choice, not the schedule for acting on it. Don't write gate language into ADR bodies (*"ratify after X"*, *"status stays proposed through Y"*). Either you're committed (`aiwf promote ADR-NNNN accepted`) or you're not (keep it `proposed`). When to *act* is a planning concern that lives in the planning surface, not the ADR body. The FSM (`proposed → accepted | rejected`; `accepted → superseded`) and `aiwf promote` are the only surfaces that constrain ADR status — no bespoke per-ADR test pins. `--force --reason` remains for exceptional ratification paths.

---

## Writing docs and entity bodies — state the conclusion, not the drafting history

**State the conclusion, not the drafting history.** Initiative docs, ADR / gap / decision bodies, and milestone specs record the current design, not how the text got there. Don't narrate a document's own revision history inline — no *"this session added…"*, *"an earlier draft claimed X, that's wrong"*, *"as of this session"*, *"in a later pass"*. If a corrected assumption explains a non-obvious aspect of the current design, keep the *reasoning* (why X holds, why Y isn't automatic) and state it as direct exposition — drop the draft-history framing around it. Provenance (when, from what conversation, superseding what) belongs in one clearly-labeled section (`## Provenance`, an ADR's date/decided-by header) — not scattered through the substantive prose as asides. Judgment call, not mechanically checkable (`wf-doc-lint` is grep-based); caught at review the same way prose coherence and design soundness already are.

---

## What aiwf commits to

Load-bearing properties any change must preserve, distilled in [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md). If a change doesn't preserve one, treat it as a kernel-level decision and surface it — not a quiet refactor.

1. **Six entity kinds** — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for legal transitions. Hardcoded, not driven by external YAML.
2. **Stable ids that survive rename, cancel, and collision.** Every kernel id emits at a uniform canonical 4-digit width (ADR-0008); parsers tolerate narrower legacy widths on input, while renderers and allocators always emit canonical width. Migrate legacy trees with `aiwf rewidth --apply` (one idempotent commit). The id is the primary key, the slug is display; rename preserves the id; "removal" means a terminal status, not deletion; collisions are detected by `aiwf check` and resolved by `aiwf reallocate`.
3. **`aiwf check` runs as a pre-push git hook.** Validation is the chokepoint that makes the guarantees real.
4. **`aiwf history <id>` reads `git log`.** No separate event log; structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable.
5. **Marker-managed framework artifacts, regenerated only on explicit `aiwf init` / `aiwf update`.** Skills under `.claude/skills/aiwf-*` (gitignored) and git hooks under `.git/hooks/<hook>` (identified by an `# aiwf:<hook>` marker so user hooks are left alone). The per-turn guidance fragment (ADR-0018): `init`/`update` materialize `.claude/aiwf-guidance.md` and maintain a marker-wrapped `@.claude/aiwf-guidance.md` import in the consumer's root `CLAUDE.md` — line-anchored, self-healing, default-on with an `aiwf.yaml` opt-out (`guidance.wire_claudemd: false`). `aiwf update` refreshes every opted-in artifact.
6. **Layered location-of-truth.** Engine binary external (`go install`); per-project policy/planning state in the consumer repo; materialized skill adapters in the consumer repo but gitignored.
7. **Every mutating verb produces exactly one git commit** — per-mutation atomicity for free; a failed mutation aborts before the commit.
8. **Acceptance criteria as namespaced sub-elements of milestones; TDD opt-in per milestone.** ACs are addressed by composite id `M-NNNN/AC-N`, validated by `aiwf check`, with the `acs-tdd-audit` rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`.
9. **Principal × agent × scope provenance.** The kernel separates *who is accountable* (principal, always human) from *who ran the verb* (actor, may be LLM/bot). Authorized work is gated by a scope FSM (`active | paused | ended`) opened with `aiwf authorize`; `--force` requires a human actor. Identity is runtime-derived from `git config user.email`, not stored in `aiwf.yaml`.
10. **Uniform archive convention for terminal-status entities** (ADR-0004). Every kind stores terminal entities under a per-parent `archive/` subdirectory; `aiwf promote` / `aiwf cancel` flip status only, and `aiwf archive` sweeps qualifying entities into their archive subdirs as one commit. The loader resolves ids across active and archive, so cross-references stay live. Reversal is absent — file a new entity referencing the archived one. Drift is policed by the `archive-sweep-pending` advisory finding (opt-in `archive.sweep_threshold` knob in `aiwf.yaml`).

---

## What is *not* in scope

Deliberately excluded (each addable later when real friction demands it): an events.jsonl / append-only event log, a graph-projection or hash-chain file, a cross-branch monotonic ID counter, a module/capability registry, multi-host adapter generation (aiwf targets Claude Code only), a third-party skill registry, tombstones beyond terminal statuses, CRDT primitives / custom merge drivers / server-side hooks, GitHub-Issues or Linear sync, full FSM-as-YAML. If you reach for any of these, check design-decisions.md §"What's deliberately not in the PoC" — there's almost certainly a simpler way.

---

## How to validate changes

```bash
go test -race ./...              # unit tests
make lint                        # linters (worktree-scoped cache; bare `golangci-lint run` leaks stale findings across worktrees, G-0179)
go build -o /tmp/aiwf ./cmd/aiwf # binary builds
```

`make check-fast` bundles the first two plus `go vet` for the inner loop; `make ci` runs the full CI-parity gate (race, coverage, self-check). CI runs the full gate on every push.

**Local validation cadence.** `make ci` gates the *integration boundaries* (the epic→main merge and the push); it need **not** run before every local commit:

- **Inner loop** (milestone work on an epic branch, a `wf-patch` branch before wrap): use `make check-fast`.
- **Skip the redundant run** when no Go/build input (`*.go`, `go.mod`, `go.sum`, `Makefile`, `.github/workflows/*`) changed since the last green `make ci`. Planning-state commits (`aiwf promote`/`edit-body`/`add`) touch entity markdown, never build inputs, and don't invalidate a green run.
- **A milestone wrap merges into the *epic* branch, not mainline** — so run `make check-fast`, not the full gate; its safety net is the pre-push hook + CI-on-push when the epic branch is pushed. The wrap merge commit may use `git commit --no-verify` (its tree is byte-identical to the already-validated implementation commit). **Never `--no-verify` a push** — that skips the gitleaks scan whose boundary *is* the push.

The authoritative gate is CI-on-push; local `make ci` is pre-flight insurance at the boundary that integrates or leaves the machine.

---

## Operator setup

After cloning, run **`aiwf init`** (first time) or **`aiwf update`** (existing repo) at the repo root. That single command materializes everything aiwf ships into `.claude/`: verb skills (`aiwf-*`), ritual skills (`aiwfx-*`), engineering skills (`wf-*`), role agents (planner/builder/reviewer/deployer), and entity templates — all gitignored, marker-managed, byte-refreshed on `aiwf update`. Rituals are embedded in the binary from a pinned snapshot (ADR-0014), so there is no separate install step and the ritual version always equals the binary version.

Verify with `aiwf doctor` (the `rituals:` line confirms materialization). aiwf does **not** edit your `.claude/settings.json` without **explicit per-invocation consent**. The narrow exception is the statusline opt-in (`aiwf init/update --statusline`): settings edits are gated by an interactive `[y/N]` confirm on a TTY or the explicit `--wire-settings` flag. See [ADR-0015](docs/adr/ADR-0015-settings-json-edits-require-explicit-per-invocation-consent.md).

### Devcontainer

If you use the devcontainer (see [`.devcontainer/README.md`](.devcontainer/README.md)), no separate install is needed inside the container either — `aiwf init` / `aiwf update` materializes the rituals into the container's `.claude/` exactly as on the host. Mechanics live in [`.devcontainer/initialize.sh`](.devcontainer/initialize.sh) and [`.devcontainer/devcontainer.json`](.devcontainer/devcontainer.json).

---

## Working in this repo

**Trunk-based development on `main` for maintainers.** Commit directly to trunk; no PR ceremony. Validation is mechanized: `aiwf check` runs pre-commit (shape-only) and pre-push (full), and CI runs the full matrix. Outside contributors use GitHub PRs (see [`CONTRIBUTING.md`](CONTRIBUTING.md)). Conventional Commits subjects are mandatory both paths. When in doubt, the smaller change is the right change.

**Derived artifacts — regenerate, don't hand-edit.** Change the source, then regenerate:
- **`ROADMAP.md`** — committed, regenerated by `aiwf render roadmap` (the trap: it's tracked but not hand-maintained).
- **`STATUS.md`** — gitignored, regenerated by the post-commit hook (`status_md.auto_update`).
- **`site/`** — gitignored, the `aiwf render --format=html` output.
- Materialized `.claude/` artifacts (skills, `aiwf-guidance.md`, hooks) — gitignored, regenerated by `aiwf init`/`update`.
- **Not derived:** the embedded ritual snapshot under `internal/skills/embedded-rituals/` is the *authoring source of truth* — you hand-edit it (see §"Ritual content authoring").

---

## AC promotion requires mechanical evidence

Before `aiwf promote M-NNNN/AC-<N> met`, there must be a mechanical assertion that fails if the AC's claim breaks — a Go test under `internal/policies/`, a kernel finding-rule, or a fixture-validation script. *"I read it and it looks right"* is not evidence. This applies **even to `tdd: none` milestones**: the `tdd:` policy only controls whether `acs-tdd-audit` fires; it never waives the test obligation. For doc-shaped ACs, the test is a structural assertion scoped to a named markdown section (see §"Substring assertions are not structural assertions"). The chokepoint is the AC-promote command.

---

## Default to a worktree for any branch work in this repo

When a ritual (`wf-patch`, milestone, epic) or an ad-hoc fix creates a branch, default to an in-repo worktree (`.claude/worktrees/<branch>/`, per ADR-0023) rather than switching the main checkout in place — create it with `aiwf worktree add <branch> --base <base>`, which materializes rituals (skills, agents, templates, guidance) into the new worktree atomically, in one step. This applies to the session's own direct work, not just subagent dispatch. Skip only when explicitly told to work in the main checkout for that invocation.

---

## Subagent worktree isolation

When dispatching a subagent that must work in an isolated worktree, **the parent bootstraps the worktree before invoking `Agent`** — never rely on the `isolation: "worktree"` kwarg (it has been observed to silently drop, leaving work in the live tree undetected; [G-0099](work/gaps/G-0099-worktree-isolation-parent-side-precondition.md)):

1. Parent runs `aiwf worktree add <branch> [<path>] --base <base>` (an explicit `<path>` for a sibling placement; `.claude/worktrees/<name>` — the default — for transient agent worktrees) — creates the worktree and materializes rituals into it atomically, in one step.
2. Parent verifies with `aiwf doctor --root <path>`, which reports rituals as materialized with no separate `aiwf init`/`aiwf update` step needed.
3. Parent invokes `Agent` *without* the `isolation` kwarg; the prompt names the worktree path so the subagent uses absolute paths / `git -C <path>`.
4. On return, parent verifies the subagent's commits live on the worktree branch.

The chokepoint is [`.claude/hooks/validate-agent-isolation.sh`](.claude/hooks/validate-agent-isolation.sh), a `PreToolUse` hook that denies any `Agent` call passing `isolation: "worktree"`. Contract pinned by `TestAgentIsolationHook_*`.

---

## Worktree binary discipline

When diagnosing `aiwf`'s own behavior against a worktree with uncommitted/unmerged code changes, the `aiwf` on PATH was built from earlier state — `aiwf check`/`doctor` then compute results from stale code, silently. **Build a worktree-scoped binary and invoke it by path**, don't rely on PATH: run `make diag-aiwf` (builds `./bin/aiwf-diag` from current source, prints its absolute path), then invoke that path throughout the session. Rebuild after any commit touching the packages under diagnosis (typically `internal/check`, `internal/entity`, `internal/cli`, `internal/verb`). For AI sessions, `go build -o "$CLAUDE_JOB_DIR/aiwf" ./cmd/aiwf` and invoke that path. Operator discipline — nothing mechanically blocks a stale-PATH call.

---

## Ritual content authoring

Rituals (`aiwfx-*` / `wf-*` skills, agents, templates) are **authored directly** at `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md`, embedded via `go:embed`, and materialized into consumers' `.claude/` by `aiwf init` / `aiwf update` (ADR-0014). A ritual edit is one commit here — no cross-repo coordination. The upstream `23min/ai-workflow-rituals` repo is archived (ADR-0016); the embedded snapshot IS the single source of truth. When a milestone's deliverable is ritual content, the authoring location is the embedded snapshot itself, and AC tests under `internal/policies/` assert against the embedded bytes via path constants.

**Every embedded-rituals `SKILL.md` edit must land alongside a referencing structural test under `internal/policies/`.** This is mechanical, not vigilance: the `skill-edit-structural-test-backstop` policy fails the CI coverage-gate step when a commit modifies a `SKILL.md` under `internal/skills/embedded-rituals/**` whose path no `internal/policies/*_test.go` references. It's diff-scoped and CI-tier (the property is an aiwf-repo invariant, meaningless in a consumer tree). v1 granularity is file-existence + skill-reference.

---

## Consumer-operating guidance vs repo-development guidance

Two audiences read guidance, each with a different shippable home. The dividing line is **audience, not importance**:

- **"How to OPERATE aiwf in any repo"** (gate-per-mutation, reallocate-not-`git mv`, AC-mechanical-evidence, one-decision-at-a-time, never-pause, `body-prose-id`, cross-branch id allocation) is consumer-facing and **ships**: the always-on subset lives in the embedded guidance source (`internal/skills/embedded-guidance/aiwf-guidance.md`, materialized into a consumer's `.claude/aiwf-guidance.md` and `@`-imported); lower-frequency detail routes to on-demand skills. Because this repo dogfoods the same materialized guidance, an operating rule placed in the embedded source is followed here *and* shipped — one source, no fork. Placed in this `CLAUDE.md` instead, it forks.
- **"How to DEVELOP aiwf itself"** (Go conventions, test-parallelism discipline, `make ci` cadence, release process, chokepoint pointers, ritual-authoring locations) stays in **this** `CLAUDE.md` and does not ship.

**Hybrid sections are split, not moved wholesale** — the *Id-collision resolution* section below is the canonical example (the allocation/avoidance workflow ships; the merge-time `git mv` mechanics stay here behind a pointer). The mechanical backstop is `PolicyM0211GuidanceOperatingAnchors` (`internal/policies/`): it asserts a curated set of operating anchors stays present in the embedded guidance source so a shipped operating rule can't drift out. It can't classify a *brand-new* rule's audience — that judgment is the "audience, not importance" test above.

---

## Id-collision resolution at merge time

When `aiwf check` reports `ids-unique/trunk-collision` (or the pre-push hook blocks) after a merge, first rule out the stale-branch case (G-0378/ADR-0031): a rename landed on trunk (`aiwf retitle`/`aiwf rename`) after the branch forked, invisible to the branch's stale copy. Check with `git log --diff-filter=R --follow origin/main -- <trunk-path>` (the trunk ref must precede the pathspec, or the check runs against HEAD and misses the rename), or just attempt the merge/pull and see whether it resolves the divergence cleanly — if so, merging trunk into the branch is the fix, and `aiwf reallocate` on the branch's copy would only produce a genuine duplicate entity. Only once the two paths are confirmed to be genuinely unrelated entities does the collision resolve via **`aiwf reallocate <id>`**, not `git mv` + a manual frontmatter edit. The allocator picks the next free id by scanning the working tree, all local/remote-tracking refs, and the trunk ref (that scan feeds *allocation only*, never the `ids-unique` check, which compares working tree against trunk; `allocate.trunk` configures the ref).

The general allocation & collision-*avoidance* workflow (allocate on your working branch; `aiwf add --fetch` and push promptly in a multi-clone setup; the unpushed-peer and unmerged-branch-prose caveats) is consumer-operating guidance and now **ships** via the embedded guidance source and the `aiwf-add` skill — no longer duplicated here. What follows is the merge-time collision-*resolution* specialization.

The `git mv` move that compiles (`git mv work/gaps/G-NNNN-slug.md work/gaps/G-MMMM-slug.md` + `id`/`prior_ids` frontmatter edit) clears the immediate `git diff -M` finding but is **not the canonical path**: it leaves the renumber invisible to `aiwf history`, doesn't rewrite cross-references, and carries no `aiwf-verb: reallocate` trailer — if you miss a reference elsewhere, no test catches it. So: **collision → `aiwf reallocate <id>`.** The verb renames + rewrites frontmatter atomically, walks the tree rewriting every cross-reference to the old id, and stamps `aiwf-verb: reallocate` + `aiwf-prior-entity:` trailers. This applies whether the collision surfaces during or after the merge. Operator discipline only — no mechanical chokepoint blocks a `git mv`; file a gap if the pattern recurs. (Surfaced in E-0033 after a main→epic merge produced two independently-allocated ids.)

---

## Go conventions

Repo-wide principles (KISS, YAGNI, no half-finished implementations, errors-as-findings) cascade in on top of these.

### Formatting and linting

- **`gofumpt`** formats (via `golangci-lint`, no separate install); gofumpt-clean implies gofmt/goimports-clean.
- **`golangci-lint`** is the only linter. Enabled: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, `revive`, `gosec`, `bodyclose`, `unconvert`, `misspell`, `gofumpt`, `goimports`. CI fails on any finding. No `//nolint` without a one-line rationale.

### Testing

- **stdlib `testing` + `github.com/google/go-cmp`.** No testify, no assertion DSLs.
- **Table-driven** when ≥2 cases share a function (subtests via `t.Run`); single-case tests stay flat.
- **Golden files** under `testdata/`, synthetic (obviously-fictional) content only.
- **Race detector on every CI run:** `go test -race ./...`.

#### Running tests in the devcontainer (primary)

Inside the devcontainer (Linux) there is no signing requirement and `go test` runs unwrapped. Use `make test` / `make test-race` / `make coverage`, bare `go test ./pkg/...`, or focused `go test -run TestX -count=1 ./pkg/...`.

#### Running tests on macOS host (fallback)

On the macOS host, `go test` must route the per-package test binary through [`scripts/sign-and-run.sh`](scripts/sign-and-run.sh) via `-exec` (it ad-hoc-signs on Darwin; no-op on Linux/CI). Skipping it lands you in a Sonoma 14.8.x syspolicyd crash loop that stalls every new process launch (see G-0128 / G-0133). **Do:** `make test`, `make test-race`, `make coverage` (CI carries `-exec=./scripts/sign-and-run.sh` equivalently). **Don't:** bare `go test ./...` (bypasses the wrap; `-race` also hits the G-0127 fork/exec deadlock). Focused runs outside `make`: `export GOFLAGS="-exec=$(pwd)/scripts/sign-and-run.sh"` then bare `go test`. **Defaults, not a chokepoint** — nothing catches a bare `go test` on the host. (Installing the production binary on Darwin needs the same signing: `make install` and `aiwf upgrade` sign automatically; bare `go install …@ver` does not — sign manually with `codesign -s - -f "$(go env GOPATH)/bin/aiwf"`.)

### Test design rules

- **Test the seam, not just the layer.** When a helper is wired into a caller, cover the helper's behavior *and* the integration seam (drive `run([]string{"<verb>", …})` or `check.Run`). When output depends on values only a real binary has (`ReadBuildInfo`, ldflags globals, `os.Args[0]`), add a binary-level subprocess integration test.
- **Contract tests for upstream-cached systems.** Pin "did we ask the right question," not just "did we parse the answer": derive the expected value through an *independent* code path (gated under `-short`).
- **Spec-sourced inputs for upstream-defined input spaces.** Cite the spec in a comment and cover the full enumerated grammar, not "the example I had in mind."
- **Substring assertions are not structural assertions.** A grep for a literal proves it exists *somewhere*, not in the right *place*. Parse HTML (`x/net/html`) / walk the heading hierarchy and assert presence inside the named section/attribute; for multi-section pages name *which* section. Use a structural assertion whenever the literal is short/generic.
- **Render output must be human-verified before an iteration closes.** For rendered UI/docs, run the binary against a real fixture (the kernel's own planning tree), exercise every tab/link/conditional path, then mark done. If you can't run the binary, say so — tests don't stand in for the look.
- **Test every reachable branch.** Each `if`/`switch`/filter arm needs a test that traverses it, or a `//coverage:ignore <reason>`. Skim uncovered lines before committing; each is a missing test, dead code to delete, or genuinely unreachable (annotate).
- **Don't paper over a test failure — root-cause it.** A state error (lock contention, projection mismatch, not-found) is information about the system. Dump the state, read the cause. A "manual git commit to keep things clean" inside a test is a yellow flag — the production verb isn't making it.
- **Policy tests that read entity files resolve via the loader** — `tree.Load(ctx, root)` + `Tree.ByID(id)` + `entity.Path`, never a hardcoded `filepath.Join(root, "work", …)` literal (archive sweeps move files; the loader resolves across active and archive). Chokepoint: `PolicyNoHardcodedEntityPaths`.

### Test discipline

Test files run **parallel-by-default**; new test files follow suit. The six load-bearing rules:

- **`setup_test.go` per package** with a `func TestMain(m *testing.M)` that `os.Setenv`s the four GIT identity vars once (`GIT_AUTHOR_NAME`/`EMAIL`, `GIT_COMMITTER_NAME`/`EMAIL`) then `os.Exit(m.Run())`. `os.Setenv`, not `t.Setenv` (which panics under `t.Parallel`).
- **`t.Parallel()` first-line** on every parallelizable test (nested inside independent table subtests too).
- **Serial skip-list** in `setup_test.go`'s comment block, one rationale line per test that must stay serial (`t.Setenv`/`t.Chdir`, mutates a package var, shares stdout/stderr capture).
- **`sync.Once` for expensive read-only shared fixtures** (cmd-binary build, live-repo `*Tree`), with a `// do not mutate` comment.
- **`-race -parallel 8` cap** uniform across `Makefile` + workflows (race + git-subprocess fan-out flakes at default parallelism). Chokepoint: `internal/policies/race_parallel_cap.go`.
- **`testsupport.HardenGitTestEnv()` in exec-bearing TestMains** (after identity seeding, before `m.Run()`): scrubs inherited git-locator env vars and disables auto-gc. Chokepoint: `internal/policies/git_test_env_harden.go`.

Presence chokepoint: `internal/policies/test_setup_presence.go` (AST walk of `internal/*` test packages; fails CI if `setup_test.go`/`TestMain` is missing). Scope is `internal/*`; `cmd/aiwf/` uses a per-file skip-list in its own `setup_test.go`.

### Coverage

High coverage on `internal/...` (PoC target 90%; total-coverage check advisory). Exclusions: `cmd/aiwf/main.go` (integration-tested), generated code, `//coverage:ignore <reason>` lines.

- **Diff-scoped coverage gate.** Every statement on a line changed since the base ref must be tested or `//coverage:ignore`'d — an untested changed branch fails CI naming the `file:line`. Engine: `internal/policies/branch_coverage_audit.go`; run locally with `make coverage-gate` (compares committed `HEAD` to the merge-base, so commit first). It's *statement* coverage with the ignore-escape, not true per-arm branch coverage.
- **Firing-fixture meta-gate.** Every policy's `Policy: "<id>"` construction line must be covered by some test (no vacuous chokepoints), else it fails unless its id is in the shrinking `grandfatherDark` ledger. Engine: `internal/policies/firing_fixture_presence.go`; runs with `make coverage-gate`. Fail-closed if the profile carries no `internal/policies` blocks.
- **Beyond line coverage.** Fuzz targets (`Fuzz*` in `internal/{entity,gitops,version,pathutil}/`, full runs on the `fuzz` workflow); property tests (`internal/entity/transition_property_test.go` + `PolicyFSMInvariants`); mutation testing (`mutate-hunt`, workflow_dispatch only, `--workers 1 --timeout-coefficient 15`; ignore equivalent-mutant / unreachable-branch noise).

### Error handling

- Wrap across boundaries: `fmt.Errorf("loading %s: %w", path, err)`. Compare with `errors.Is`/`errors.As`, never `==` (except sentinels you own).
- Sentinel errors for stable conditions; typed errors when the error carries data.
- **Library code never `panic`s or `os.Exit`s** — only `cmd/<tool>/main.go` exits.

### Concurrency

- `context.Context` is the first arg of every IO-touching function (cancellation only; don't stuff request data via `WithValue` except across API boundaries).
- Never hold a mutex across an IO call.

### CLI conventions

- **`github.com/spf13/cobra`.** Each verb is a `cobra.Command` with a `runXCmd(...)` body called from `RunE`; wire from `newRootCmd`. Closed-set flag values bind completion via `cobra.FixedCompletions(...)`; entity-id flags use `completeEntityIDFlag(kind)`. Chokepoint: `cmd/aiwf/completion_drift_test.go`.
- **Exit codes:** `0` ok, `1` findings, `2` usage, `3` internal. Verbs return `int`; the `RunE` adapter shuttles via `*exitError`.
- **Output:** human-readable by default; `--format=json` emits `{ tool, version, status, findings, result, metadata }` (`status` ∈ `ok`/`findings`/`error`); `--pretty` indents; tool output → stdout. **Diagnostic/operator logging** is opt-in and default-off ([ADR-0017](docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md)): `log/slog`, configured via `AIWF_LOG`/`AIWF_LOG_FORMAT`/`AIWF_LOG_FILE` (env beats `aiwf.yaml`'s `logging:` block beats the default), writes structured records to a daily-rotated file under the XDG state home by default — never stderr unless explicitly configured there. `forbidigo` (backed by the `logging-chokepoint` AST policy, which also catches `fmt.Fprintln`/`fmt.Fprintf` to `os.Stdout`/`os.Stderr`) bans bare `fmt.Print*` call sites outside the sanctioned `cliutil` text wrappers, so operator-facing text can't silently drift back onto ad hoc prints; diagnostic logging goes through `log/slog` in `internal/logger`, never a bare print.
- **Flags:** `--help`, `--version`, `-v`, `--pretty` plus verb-specific. No global config files; config via flags, env, or `aiwf.yaml`.
- **No package-level mutable state.** Inject dependencies via struct fields / constructors (`func New(deps Deps) *T`) — never a package var tests mutate.

### Commit conventions

Every mutating verb writes trailers so `aiwf history` can render timelines:

```
aiwf-verb: promote
aiwf-entity: M-0001
aiwf-actor: human/peter
```

Subjects follow Conventional Commits (`feat(plan): …`, `chore(plan): …`, `docs(adr): …`) — say *why*, one logical change per commit.

### Release process

Releases are git tags `vX.Y.Z` on `main`. Before tagging, in one `release(aiwf): vX.Y.Z` commit edit [`CHANGELOG.md`](CHANGELOG.md): rename `## [Unreleased]` to `## [X.Y.Z] — YYYY-MM-DD`, add a fresh empty `## [Unreleased]` above it, verify the moved entries summarize the user-visible delta. Then push the commit, `git tag vX.Y.Z`, `git push origin vX.Y.Z`. The [`changelog-check.yml`](.github/workflows/changelog-check.yml) workflow fails a `v*` tag whose commit's `CHANGELOG.md` lacks a matching `## [X.Y.Z]` heading — even pure-mechanical patches need a one-line entry.

### Dependencies

Minimize external deps (one-line justification per new one). `CGO_ENABLED=0` (static binaries). `go 1.24` minimum in `go.mod` (the consumer-compatibility floor); CI toolchain is pinned separately to `go-version: "1.25"` (bump CI for stdlib CVEs, bump the floor only for a needed language feature). One `go.mod` for the module.

### Naming

Package names short, lowercase, singular (`entity` not `entities`). Avoid stuttering (`entity.Load`, not `entity.Entity`). Exported identifiers get a doc comment starting with the name. Acronyms stay capitalized (`parseURL`, `jsonOut`).

### Type design

- **Closed-set enums ship only values with a current call site.** Speculative future values violate YAGNI.
- **The six kinds and their status sets are hardcoded in Go**, not external YAML (deferred until a real consumer needs custom vocabulary).
- **`entities.title_max_length` in `aiwf.yaml`** caps title (at `add`/`retitle`/`import`) and slug (at `rename`); default 80. Hard-reject, not truncate — the error points at `--body-file`. Title and slug share the budget so filenames and frontmatter stay in sync. Pre-cap titles are grandfathered; clean up with `aiwf retitle` (re-derives the slug), which the check treats as a git rename so a retitle batch pushes cleanly.

### Designing a new verb

The design isn't done until you can answer **"what verb undoes this?"** Acceptable: another invocation of the same verb; an explicit terminal transition (`aiwf cancel`, `aiwf reallocate`); "you can't, deliberately — here's why" (`aiwf init` is one-shot; written down); "you'd open a new entity for the inverse." Not acceptable: "we'll figure it out later." See [docs/pocv3/design/design-lessons.md](docs/pocv3/design/design-lessons.md) §"On reversal".

### Skills policy

Every top-level Cobra verb is reachable through some AI-discoverable channel, in one of four shapes (per-verb skill / topical multi-verb skill / no skill when `--help`+completion suffice / discoverability-priority split). Judgment rule in [ADR-0006](docs/adr/ADR-0006-skills-policy-per-verb-default-or-help-only.md); mechanical companion `internal/policies/skill_coverage.go` (every verb has an `aiwf-<verb>` skill or an allowlist entry; every skill carries valid `name:`/`description:` frontmatter; every backticked `` `aiwf <verb>` `` in a skill body resolves).

**Shipped surfaces cite no real entity id, filesystem path, or inline lifecycle status, and carry no development history, provenance narrative, or rationale/war-story — only imperative, consumer-scoped instruction.** "Shipped surface" is the full set the id chokepoint scans, not just `SKILL.md` bodies: those bodies *and* their `description:` frontmatter, entity templates, role-agent cards, the always-on guidance fragment, and the statusline's `#` comments. Every one materializes into consumer repos (via `aiwf init` / `aiwf update`), where aiwf's own ids are meaningless and rot as entities change status/archive/rewidth — and where a note about *this* repo's development history, or an argued rationale for a past choice, is context-free noise. Illustrative content uses canonical `<prefix>-NNNN` placeholders (the letter-N form) or shape-descriptions; the one carve-out is a markdown link to a design/ADR doc (id rides in the destination, visible text stays descriptive). Provenance, history, and rationale belong in this `CLAUDE.md`, the design docs, and commit trailers — not in a consumer-facing surface. Chokepoint: the `skill-body-id` check (`internal/check`) fires pre-push over every `*.md` under `internal/skills/embedded{,-rituals,-guidance}/**` (frontmatter included) plus the `#` comments of `embedded-statusline/*.sh`, is the mirror of `body-prose-id`, is prose-scoped (masks code spans / fenced blocks / link destinations), and is inert in a consumer repo. Only the id-shaped-token subset has a stable machine shape and is enforced there; the history/provenance/rationale content class is held at review.

### What's enforced and where

Rules are enforced at named chokepoints, not by remembering a checklist. **Fire as early as the class allows:** pre-commit (shape-only) → pre-push (full `aiwf check` + lint + gitleaks) → CI (`internal/policies/` Go tests, coverage gates, build/vet/race matrix). An `aiwf check` finding catches in-context before the work leaves the machine; a CI-only policy test is a backstop that fires after the trunk push has landed. For judgment classes no check can cover (prose coherence, design soundness, cross-reference correctness) the timely catch is the in-context wrap-ritual review, which feeds the human gate.

- **Blocking via CI lint / test / build:** the full `golangci-lint` set + `go vet` + `go test -race` + `go build` + `aiwf doctor --self-check` + `govulncheck`; and the `internal/policies/` Go tests — repo-specific invariants (trailer keys, sovereign acts, FSM wiring/no-cycle, `-parallel 8` cap, setup_test presence, git-test-env harden, enum-literal adoption, closed-set status constants, trailer-order sync, atomic-write chokepoint, version single-source, validate/is-never-writes, layering direction, no-`time.Now`-in-core), the diff-scoped coverage gate + firing-fixture meta-gate, and the skill/finding-code discoverability + ritual `skill_edit_structural_test_backstop.go` backstop + guidance-anchor + trailer-commit-drift policies.
- **Blocking pre-commit / pre-push:** `aiwf check --shape-only` (pre-commit); full `aiwf check` incl. `body-prose-id` and `skill-body-id` (pre-push); the `golangci-lint` pre-push hook (`make install-hooks`, pinned by `prepush_lint_hook_test.go`); gitleaks path/secret scan (pre-push + CI).
- **Advisory (code review):** `context.Context` first-arg, no new package-level mutable state, one-line dep justification, deliberate `go.mod` floor bumps, docs/entity bodies stating the conclusion rather than the drafting history (§"Writing docs and entity bodies").

Relax a blocking rule for a specific call site via the linter's allowlist with a one-line rationale, never a bare `//nolint`.

<!-- aiwf:guidance:START - DO NOT EDIT, regenerated by aiwf update -->
@.claude/aiwf-guidance.md
<!-- aiwf:guidance:END -->

<!-- ai-dotfiles:stack:START (generated by dotfiles-sync; do not edit) -->
@~/.agents/guidance/200-go.md
@~/.agents/guidance/201-python.md
@~/.agents/guidance/203-typescript.md
<!-- ai-dotfiles:stack:END -->
