# CLAUDE.md — ai-workflow repo

This repo carries `aiwf` — a small framework that helps humans and AI assistants track what's planned, decided, and done, by validating mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/design/design-decisions.md`](docs/design/design-decisions.md) for what aiwf commits to and §"What's deliberately not in the PoC" for scope limits. Gaps live as entities under `work/gaps/` — run `aiwf list --kind gap` or `aiwf show G-NNNN`.

---

## Engineering principles

- **Framework correctness must not depend on LLM behavior.** Skills are advisory; the pre-push hook and `aiwf check` are authoritative. A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee.
- **Provenance is principal × agent × scope, not just operator.** When the human directs the LLM in conversation, the LLM is a *tool* (human = principal, no co-actor inflation). When the human runs `aiwf authorize E-NNNN --to ai/claude`, the agent works within that scope until the scope-entity is terminal or the human pauses. `--force` is sovereign — humans only. See [`docs/design/provenance-model.md`](docs/design/provenance-model.md).

---

## Authoring an ADR

The FSM (`proposed → accepted | rejected`; `accepted → superseded`) and `aiwf promote` are the only surfaces that constrain ADR status — no bespoke per-ADR test pins.

---

## Documentation hierarchy

Every active `docs/` subtree carries one of the authority tiers below, so a reader — human or LLM — can weight a file's content correctly from its path alone, without reading every file to infer intent.

- **Normative** — current-truth, kept in lockstep with the code: `docs/adr/` (terminal-status ADRs move to `docs/adr/archive/` under the same forget-by-default convention as the Archival tier below), `docs/design/`, `docs/reference/` (generated references), `docs/architecture.md`, `docs/overview.md`, `docs/workflows.md`, `docs/skill-author-guide.md`.
- **Forward-looking** — captured ideas awaiting promotion to a real epic/gap entity: `docs/initiatives/` (an initiative whose every thread has been promoted to a tracked entity moves to `docs/initiatives/archive/` with `status: realized`, under the same forget-by-default convention as the Archival tier below — promotion, not closure of every descendant, is the bar).
- **Observational** — dated measurements of the repo, true as of the date or commit they name and worked down rather than kept in lockstep with the code: `docs/audits/`.
- **Exploratory** — synthesis/thesis/proposal genre, not kernel-binding regardless of internal rigor: `docs/explorations/` (including [`explorations/loom/loom-by-example.md`](docs/explorations/loom/loom-by-example.md), [`loom-light-plan.md`](docs/explorations/loom/loom-light-plan.md), `explorations/surveys/`), `docs/research/`, `docs/working-paper.md`.
- **Archival** — frozen historical snapshot per [the archive convention](docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md); cross-references inside it are not fixed when their targets move (same forget-by-default convention as `CHANGELOG.md`'s append-only exemption): `docs/archive/` (including `docs/archive/pocv3/`, and `docs/archive/migration/`, entry point [`from-prior-systems.md`](docs/archive/migration/from-prior-systems.md) — it specifies the `aiwf import` manifest contract, and that verb still ships, so a reader can reach the surface it documents).

---

## What aiwf commits to

Load-bearing properties any change must preserve, distilled in [`docs/design/design-decisions.md`](docs/design/design-decisions.md). If a change doesn't preserve one, treat it as a kernel-level decision and surface it — not a quiet refactor.

1. **Six entity kinds** — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for legal transitions. Hardcoded, not driven by external YAML.
2. **Stable ids that survive rename, cancel, and collision.** Every kernel id emits at a uniform canonical 4-digit width ([the canonical-width ADR](docs/adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md)); parsers tolerate narrower legacy widths on input, while renderers and allocators always emit canonical width. That tolerance is permanent: it is what keeps live cross-references into entities archived before the width convention resolving, since no verb widens an id in place. A narrow id in the *active* tree is a defect, reported at error severity by `entity-id-narrow-width`. The id is the primary key, the slug is display; rename preserves the id; "removal" means a terminal status, not deletion; collisions are detected by `aiwf check` and resolved by `aiwf reallocate`.
3. **`aiwf check` runs as a pre-push git hook.** Validation is the chokepoint that makes the guarantees real.
4. **`aiwf history <id>` reads `git log`.** No separate event log; structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable.
5. **Selected-host framework artifacts, refreshed explicitly.** `aiwf init` / `aiwf update` select Claude Code and/or Codex from an explicit `hosts` list or executable detection on PATH; `hosts: []` selects none. Claude uses `.claude/` and a managed guidance import in `CLAUDE.md`; Codex uses `.agents/` and native managed guidance in `AGENTS.md`. Root wiring defaults on, with `guidance.wire_claudemd` and `guidance.wire_agentsmd` opt-outs. Preserve user content and retained unselected artifacts. Core Git hooks remain host-independent. See [host setup](README.md#2-host-setup-and-embedded-rituals).
6. **Layered location-of-truth.** Engine binary external (`go install`); per-project policy/planning state in the consumer repo; materialized skill adapters in the consumer repo but gitignored.
7. **Every mutating verb produces exactly one git commit, or none** — per-mutation atomicity for free. A failed mutation aborts before the commit; a request already satisfied converges to a NoOp at exit 0 and writes none ([the same-status ADR](docs/adr/ADR-0036-same-status-fsm-transitions-converge-to-noop-not-refusal.md), and the same-state convergence rule in `.guidance/project/verb-design.md`). The invariant the oracles check is *one mutation ⇒ exactly one commit; a NoOp ⇒ zero*, with `metadata.commit_sha` present only in the first case.
8. **Acceptance criteria as namespaced sub-elements of milestones; TDD opt-in per milestone.** ACs are addressed by composite id `M-NNNN/AC-N`, validated by `aiwf check`, with the `acs-tdd-audit` rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`.
9. **Principal × agent × scope provenance.** The kernel separates *who is accountable* (principal, always human) from *who ran the verb* (actor, may be LLM/bot). Authorized work is gated by a scope FSM (`active | paused | ended`) opened with `aiwf authorize`; `--force` requires a human actor. Identity is runtime-derived from `git config user.email`, not stored in `aiwf.yaml`.
10. **Uniform archive convention for terminal-status entities** ([the archive ADR](docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)). Every kind stores terminal entities under a per-parent `archive/` subdirectory; `aiwf promote` / `aiwf cancel` flip status only, and `aiwf archive` sweeps qualifying entities into their archive subdirs as one commit. The loader resolves ids across active and archive, so cross-references stay live. Reversal is absent — file a new entity referencing the archived one. Drift is policed by the `archive-sweep-pending` advisory finding (opt-in `archive.sweep_threshold` knob in `aiwf.yaml`).

---

## What is *not* in scope

Deliberately excluded (each addable later when real friction demands it): an events.jsonl / append-only event log, a graph-projection or hash-chain file, a cross-branch monotonic ID counter, a module/capability registry, a third-party skill registry, tombstones beyond terminal statuses, CRDT primitives / custom merge drivers / server-side hooks, GitHub-Issues or Linear sync, full FSM-as-YAML. If you reach for any of these, check design-decisions.md §"What's deliberately not in the PoC" — there's almost certainly a simpler way.

---

## How to validate changes

```bash
go test -race ./...              # unit tests
make lint                        # linters (worktree-scoped cache; bare `golangci-lint run` leaks stale findings across worktrees)
go build -o /tmp/aiwf ./cmd/aiwf # binary builds
```

`make check-fast` bundles the first two plus `go vet` for the inner loop; `make ci` runs the full CI-parity gate (race, coverage, the profile-driven gates, self-check). CI runs the full gate on every push.

**Local validation cadence.** `make ci` gates the *integration boundaries* (the epic→main merge and the push); it need **not** run before every local commit:

- **Inner loop** (milestone work on an epic branch, a `wf-patch` branch before wrap): use `make check-fast`.
- **Skip the redundant run** when no Go/build input (`*.go`, `go.mod`, `go.sum`, `Makefile`, `.github/workflows/*`) changed since the last green `make ci`. Planning-state commits (`aiwf promote`/`edit-body`/`add`) touch entity markdown, never build inputs, and don't invalidate a green run.
- **A milestone wrap merges into the *epic* branch, not mainline** — so run `make check-fast`, not the full gate; its safety net is the pre-push hook + CI-on-push when the epic branch is pushed. The wrap merge commit may use `git commit --no-verify` (its tree is byte-identical to the already-validated implementation commit). **Never `--no-verify` a push** — that skips the gitleaks scan whose boundary *is* the push.

The authoritative gate is CI-on-push; local `make ci` is pre-flight insurance at the boundary that integrates or leaves the machine.

---

## Stress-test harness

`cmd/stresstest` is a correctness stress harness — dev-only tooling, never installed alongside `cmd/aiwf`. It builds the `aiwf` binary under test and drives it as a real subprocess against disposable git repos through a catalog of workflow-legality, concurrency, and fault-injection scenarios, each with a pass/fail oracle. The catalog runner itself (`make stress`, `go run ./cmd/stresstest`) is on-demand only, never scheduled and never wired into `make ci`.

**The scenarios' own Go tests split by what their oracle asserts, and the split is the `stress` build tag.**

- **Hermetic scenarios run on every push**, untagged, inside the ordinary `go test ./...`. The workflow-legality group is what the harness exists for: `verb-sequence` (a seeded random walk of `aiwf promote` across all six kinds, asserting the FSM's legality verdict is honored and that `aiwf check` reports no error-severity finding and no warning outside a curated baseline — an absolute allowlist judging each state on its own terms, not a comparison against the prior step) plus `archive-during-active-scope`, `force-override-durability`, `head-drift`, and `promote-on-wrong-branch-detection`. Six more qualify on the same test and stay untagged despite their subject matter: `disk-fault` (permission-denied write, no timing at all), `parallel-branch-reallocate` and `cross-worktree-edit-body-race` (collisions deterministic by construction — two clones computing the same `max(id)+1`, two sequential commits to one body field), `reachability-isolation` (a byte-identical `aiwf check` outcome either side of an invisible sibling commit), and `concurrent-id-allocation` and `concurrent-move`, which race real subprocesses but between them judge only load-invariant facts: refusals carrying repolock's busy code, a commit count matching exactly the actors that succeeded, at least one actor through, and — each in its own scenario — no id allocated twice and every milestone reported moved really landed under the target epic. Category name is not the criterion; oracle shape is.
- **Scenarios that drive real, concurrently-scheduled subprocesses carry `//go:build stress`** and run only via `make stress-tests` (or `go test -tags stress ./internal/stresstest/ ./cmd/stresstest/`). Five qualify: `concurrent-milestone-race`, `concurrent-writer-at-scale`, `cross-worktree-id-race`, `mid-write-kill`, and `lock-kill`. The reason is cost, not correctness: sharing a runner with `go test ./...` roughly doubles their wall time. All five judge load-invariant facts, by different means. `concurrent-milestone-race` excuses the documented `repo-lock-busy` refusal an actor takes when another holds the lock, bounds each transition at at-most-one, and asks only that some actor got through. `concurrent-writer-at-scale` retries past that refusal instead of excusing it. `cross-worktree-id-race` races two worktrees, which hold two separate lockfiles, so lock contention does not arise between them. `mid-write-kill` and `lock-kill` bound their observation on the watched process rather than a clock, so slowness delays their verdict instead of falsifying it. In `cmd/stresstest`, the whole-catalog runner tests (`TestRunRun_ScenarioAll_*`) carry the tag because `--scenario all` executes this class, and so does the lock-kill runner test, which drives a real lock-holding subprocess; `--scenario all`'s *selection* claim stays pinned untagged by `TestResolveScenarios_All_NamesEveryCatalogEntryInOrder`.

Adding a scenario means choosing its lane, and since no oracle now measures the machine that choice is a cost decision rather than a correctness one: a scenario goes tagged when it drives real subprocesses and would slow every push, untagged otherwise. An oracle that asserts how *many* actors get through, or how fast, is the thing to refuse at review — that is the runner's throughput, not aiwf's behavior.

The tag applies per file, so a decision test sharing a file with a scenario driver leaves the every-push path with it — silently, since it still passes whenever anyone runs it. `PolicyStressLaneCensus` (`internal/policies/stress_lane_census.go`) is the chokepoint across both stress packages: a `stress`-tagged test that starts no process, waits on no clock and runs no goroutine fails the policy suite, naming the test to move. Driving `RunScenario` / `RunRepeated` / `Setup` counts only alongside a real scenario constructor, since all three take a `Scenario` and this package's own hermetic tests hand them fakes. Split the file rather than tagging around it.

```bash
make stress                          # run the whole catalog, 5 attempts per scenario by default (override: STRESS_REPEAT=N make stress)
make stress-tests                    # run the `stress`-tagged Go tests: the real-subprocess scenario drivers + the whole-catalog runner tests
go run ./cmd/stresstest list         # enumerate every scenario name --scenario can select
go run ./cmd/stresstest run --scenario <name>|all --repeat N [--out DIR] [--module-root DIR]
go run ./cmd/stresstest compose <raw-report-path>   # render a human-readable summary from a run's raw-report JSONL
```

`run` flags: `--scenario` (required; a catalog name, or `all` for the whole catalog), `--repeat` (attempts per scenario, default 1 — statistical coverage for timing-shaped scenarios comes from repeating, not from one lucky run), `--out` (build output, scenario temp dirs, and the raw-report file; defaults to a fresh temp dir, printed on completion), `--module-root` (the aiwf module root to build the binary under test from; defaults to `.`). A failing attempt prints its preserved repo directory for follow-up.

---

## Working in this repo

**Trunk-based development on `main` for maintainers.** Commit directly to trunk; no PR ceremony. Validation is mechanized: `aiwf check` runs pre-commit (shape-only) and pre-push (full), and CI runs the full matrix. Outside contributors use GitHub PRs (see [`CONTRIBUTING.md`](CONTRIBUTING.md)). Conventional Commits subjects are mandatory both paths. When in doubt, the smaller change is the right change.

**Derived artifacts — regenerate, don't hand-edit.** Change the source, then regenerate:
- **`ROADMAP.md`** — committed, regenerated by `aiwf render roadmap` (the trap: it's tracked but not hand-maintained).
- **`STATUS.md`** — gitignored, regenerated by the post-commit hook (`status_md.auto_update`).
- **`site/`** — gitignored, the `aiwf render --format=html` output.
- Materialized host artifacts — regenerate through `aiwf init`/`update`; see the [host setup guide](README.md#2-host-setup-and-embedded-rituals) for selected-host paths and managed root guidance.
- **Not derived:** the embedded ritual snapshot under `internal/skills/embedded-rituals/` is the *authoring source of truth* — you hand-edit it (see §"Ritual content authoring").

---

## Default to a worktree for any branch work in this repo

When a ritual (`wf-patch`, milestone, epic) or an ad-hoc fix creates a branch, default to an in-repo worktree (`.claude/worktrees/<branch>/`, per [the worktree-placement ADR](docs/adr/ADR-0023-default-to-in-repo-worktree-placement-under-claude-worktrees.md)) rather than switching the main checkout in place — create it with `aiwf worktree add <branch> --base <base>`, which materializes the selected hosts’ supported artifacts into the new worktree atomically, in one step. This applies to the session's own direct work, not just subagent dispatch. Skip only when explicitly told to work in the main checkout for that invocation.

---

## Ritual content authoring

Rituals (`aiwfx-*` / `wf-*` skills, agents, templates) are **authored directly** at `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md`, embedded via `go:embed`, and materialized by `aiwf init` / `aiwf update` into the [selected hosts’ supported paths](README.md#2-host-setup-and-embedded-rituals) ([the embed-and-materialize ADR](docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md)). A ritual edit is one commit here — no cross-repo coordination. The upstream `23min/ai-workflow-rituals` repo is archived ([the upstream-retirement ADR](docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md)); the embedded snapshot IS the single source of truth. When a milestone's deliverable is ritual content, the authoring location is the embedded snapshot itself, and AC tests under `internal/policies/` assert against the embedded bytes via path constants.

**Every embedded-rituals `SKILL.md` edit must ride a commit that names the entity it belongs to.** This is mechanical, not vigilance: the `skill-edit-provenance-backstop` policy fails the profile-driven gate step when a commit adds or modifies a `SKILL.md` under `internal/skills/embedded-rituals/**` carrying no `aiwf-entity:` trailer, or carrying one whose value resolves to no entity in the tree. Commit such an edit with `--trailer "aiwf-entity: <id>"` naming the epic, milestone, gap, or decision that owns it. The gate is diff-scoped over the commits between the base ref and HEAD — an uncommitted edit carries no provenance yet, so it is not judged — and it runs in `make ci` as well as CI (the property is an aiwf-repo invariant, meaningless in a consumer tree).

`aiwf-entity` is the whole of the requirement. No `aiwf-verb:` is asked for, because a ritual `SKILL.md` is source rather than an entity file and no aiwf verb commits it — the closed set `trailer-verb-unknown` enforces carries no value meaning "I edited a shipped surface", so requiring one would mandate a fabricated trailer. No `aiwf-actor:` is asked for either: where no verb ran, "who ran the verb" is undefined — an edit directed in conversation has the human as principal and the assistant as a tool, so there is no separate actor to record. The owning entity's *status* is not consulted, only its existence — see [the provenance-not-content decision](work/decisions/D-0071-enforce-provenance-not-content-at-the-skill-edit-backstop.md) for why keying on status would rot, and [the shipped-prose decision](work/decisions/D-0070-retire-prose-content-assertions-over-shipped-surfaces.md) for why content correctness in shipped prose is held at review instead.

---

## Consumer-operating guidance vs repo-development guidance

Two audiences read guidance, each with a different shippable home. The dividing line is **audience, not importance**:

- **"How to OPERATE aiwf in any repo"** (gate-per-mutation, reallocate-not-`git mv`, AC-mechanical-evidence, one-decision-at-a-time, never-pause, `body-prose-id`, cross-branch id allocation) is consumer-facing and **ships**: the always-on subset lives in the embedded guidance source (`internal/skills/embedded-guidance/aiwf-guidance.md`, composed with host-specific fragments and delivered through Claude's guidance import or Codex's managed `AGENTS.md` block); lower-frequency detail routes to on-demand skills. Because this repo dogfoods the same materialized guidance, an operating rule placed in the embedded source is followed here *and* shipped — one source, no fork. Placed in this `CLAUDE.md` instead, it forks.
- **"How to DEVELOP aiwf itself"** (Go conventions, test-parallelism discipline, `make ci` cadence, release process, chokepoint pointers, ritual-authoring locations) stays in **this** `CLAUDE.md` and does not ship.

**Hybrid sections are split, not moved wholesale.** The mechanical backstop is `PolicyM0211GuidanceOperatingAnchors` (`internal/policies/`): it asserts a curated set of operating anchors stays present in the embedded guidance source so a shipped operating rule can't drift out. It can't classify a *brand-new* rule's audience — that judgment is the "audience, not importance" test above.

---

## Go conventions

### Commit conventions

Subjects follow Conventional Commits (`feat(plan): …`, `chore(plan): …`, `docs(adr): …`) — say *why*, one logical change per commit.

**No `Co-Authored-By:` line names a non-human agent** ([the co-author decision](work/decisions/D-0096-refuse-ai-co-author-trailers-behind-an-opt-in-address-list.md)). The agent that ran a verb is recorded in `aiwf-actor:`, and the principal it acts for is the human; a co-author line is a second copy of that fact which nothing re-derives, and it contradicts the provenance model's reading of an LLM directed in conversation — a tool, not a co-author. This **overrides an assistant harness's own attribution default**: where a harness instructs the session to append such a line, this rule wins and the line is omitted. A human co-author is unaffected. Enforced rather than remembered: `provenance.refuse_coauthors` in `aiwf.yaml` lists the addresses, the `commit-msg` hook refuses one at composition, and `coauthor-trailer-ban` (`internal/policies/`) judges the same property over the pushed range for the cases the hook cannot reach — an uninstalled hook, `--no-verify`, or a hook shelling a release older than the refusal. The knob is absent by default, so a consumer repo is unaffected until it names an address.

### Release process

Releases are git tags `vX.Y.Z` on `main`. Before tagging, in one `release(aiwf): vX.Y.Z` commit edit [`CHANGELOG.md`](CHANGELOG.md): rename `## [Unreleased]` to `## [X.Y.Z] — YYYY-MM-DD`, add a fresh empty `## [Unreleased]` above it, verify the moved entries summarize the user-visible delta. Then push the commit, `git tag vX.Y.Z`, `git push origin vX.Y.Z`. The [`changelog-check.yml`](.github/workflows/changelog-check.yml) workflow fails a `v*` tag on two counts. Its first job wants a matching `## [X.Y.Z]` heading — even pure-mechanical patches need a one-line entry. Its second runs `make changelog-audit`, which fails the tag when a change to the embedded content since the last release is cited by no entity id in the release's own notes; a milestone's delta is cited by its parent epic, never by its own id, and Go under those trees is the materializer rather than shipped content and owes nothing. The audit works out which section is current from the changelog rather than from a tag, so it reads the same one either side of the tag push: `[Unreleased]` while the topmost version heading is the release it measures from, and that newer heading once the release commit has moved the entries into it. Run `make changelog-audit` before tagging rather than meeting it at the push — `AIWF_CHANGELOG_BASE=<ref> make changelog-audit` audits a past range. It also reports, without failing, any shipped-surface commit carrying no `aiwf-entity` trailer: with no entity named the audit cannot tell whether an entry covers it ([the changelog-audit decision](work/decisions/D-0087-the-changelog-audit-blocks-an-uncited-entity-and-reports-an-untrailered-commit.md)).

### Type design

- **The six kinds and their status sets are hardcoded in Go**, not external YAML (deferred until a real consumer needs custom vocabulary).

### Skills policy

Every top-level Cobra verb is reachable through some AI-discoverable channel, in one of four shapes (per-verb skill / topical multi-verb skill / no skill when `--help`+completion suffice / discoverability-priority split). Judgment rule in [the skills-policy ADR](docs/adr/ADR-0006-skills-policy-per-verb-default-or-help-only.md); mechanical companion `internal/policies/skill_coverage.go` (every verb has an `aiwf-<verb>` skill or an allowlist entry; every skill carries valid `name:`/`description:` frontmatter; every backticked `` `aiwf <verb>` `` in a skill body resolves).

**Shipped surfaces cite no real entity id, filesystem path, or inline lifecycle status, and carry no development history, provenance narrative, or rationale/war-story — only imperative, consumer-scoped instruction.** "Shipped surface" is the full set the id chokepoint scans, not just `SKILL.md` bodies: those bodies *and* their `description:` frontmatter, entity templates, role-agent cards, the always-on guidance fragment, and the statusline's `#` comments. Every one materializes into consumer repos (via `aiwf init` / `aiwf update`), where aiwf's own ids are meaningless and rot as entities change status or move to archive — and where a note about *this* repo's development history, or an argued rationale for a past choice, is context-free noise. Illustrative content uses canonical `<prefix>-NNNN` placeholders (the letter-N form) or shape-descriptions; the one carve-out is a markdown link to a design/ADR doc (id rides in the destination, visible text stays descriptive). Provenance, history, and rationale belong in this `CLAUDE.md`, the design docs, and commit trailers — not in a consumer-facing surface. Chokepoint: the `skill-body-id` check (`internal/check`) fires pre-push over every `*.md` under `internal/skills/embedded{,-rituals,-guidance}/**` (frontmatter included) plus the `#` comments of `embedded-statusline/*.sh`, is the mirror of `body-prose-id`, and is inert in a consumer repo. It masks non-prose link carriers, which is what gives the doc-link carve-out; code constructs are NOT exempt, since a command example ships to a consumer exactly as prose does. It fires two shapes — a real id, and a placeholder at any width other than canonical — at **error** severity, so either one blocks the push. Only the id-shaped-token subset has a stable machine shape and is enforced there; the history/provenance/rationale content class is held at review.

### What's enforced and where

Rules are enforced at named chokepoints, not by remembering a checklist. **Fire as early as the class allows:** pre-commit (shape-only) → pre-push (full `aiwf check` + lint + gitleaks) → CI (`internal/policies/` Go tests, coverage gates, build/vet/race matrix) → release tag. The last tier holds the one rule that cannot fire earlier: a milestone's delta is legitimately absent from the notes until its epic wraps, so asking before a release would need an in-flight-epic exemption. An `aiwf check` finding catches in-context before the work leaves the machine; a CI-only policy test is a backstop that fires after the trunk push has landed. For judgment classes no check can cover (prose coherence, design soundness, cross-reference correctness) the timely catch is the in-context wrap-ritual review, which feeds the human gate.

- **Blocking via CI lint / test / build:** the full `golangci-lint` set + `go vet` + `go test -race` + `go build` + `aiwf doctor --self-check` + `govulncheck`; and the `internal/policies/` Go tests — repo-specific invariants (trailer keys, sovereign acts, FSM wiring/no-cycle, `-parallel 8` cap, setup_test presence, git-test-env harden, stress-lane census, enum-literal adoption, closed-set status constants, trailer-order sync, atomic-write chokepoint, version single-source, validate/is-never-writes, layering direction, no-`time.Now`-in-core, `Violation.Policy` spelled as a string literal so the firing-fixture inventory can see every policy), the diff-scoped coverage gate + firing-fixture meta-gate + `test-executable-write` scan, and the skill/finding-code discoverability + ritual `skill_edit_provenance_backstop.go` backstop + `shipped_prose_assertion.go` ban + `coauthor_trailer_ban.go` range check + guidance-anchor + trailer-commit-drift policies. The `shipped-prose-assertion` ban judges *test source*, not prose, so it is immune to rewording; it scans every test package rather than `internal/policies` alone, because [the shipped-prose decision](work/decisions/D-0070-retire-prose-content-assertions-over-shipped-surfaces.md) scopes itself by surface and a package-scoped ban would make "wrote the test elsewhere" an exception nothing records. `test-executable-write` is CI-tier rather than pre-push only because the hook's existing diff-scoped block is comment-history-specific down to its failure message; `internal/stresstest`, `internal/contractverify` and `internal/policies` — the packages where the `ETXTBSY` flake has been observed firing, rather than merely being reachable — are additionally held clean whole-file by `TestAdoptedPackagesRouteThroughWriteExecutable`, since a diff-scoped rule forgives every line a change does not touch.
- **Blocking at commit composition:** the `commit-msg` hook's refusal grounds — a fabricated `aiwf-verb:` value, a subject claiming an AC its `aiwf-entity:` trailer does not name, an aiwf trailer block a blank line has hidden from git, a shipped-ritual edit naming no entity, and a `Co-Authored-By:` address listed under `provenance.refuse_coauthors` ([the co-author decision](work/decisions/D-0096-refuse-ai-co-author-trailers-behind-an-opt-in-address-list.md)). Each costs a retype here and an amend or a rebase once the commit exists.
- **Blocking pre-commit / pre-push:** `aiwf check --shape-only` (pre-commit); full `aiwf check` incl. `body-prose-id` and `skill-body-id` (pre-push; both block); the `golangci-lint` pre-push hook (`make install-hooks`, pinned by `prepush_lint_hook_test.go`); gitleaks path/secret scan (pre-push + CI); the diff-scoped `comment-history-attrition` scan (pre-push on Go-source pushes + CI coverage-gate) — a comment on a changed line narrating a superseded state blocks the push; `//history:ok <reason>` is the deliberate exception. Its whole-tree sibling `PolicyCommentHistoryAttritionTree` runs in the ordinary policy suite (`make check-fast` / `make ci` / CI), so the tree stays clean rather than merely not-getting-worse. It enumerates the tree through git, which Go's test cache cannot observe, so a local run can replay a pass that predates a newly-committed file; CI never caches (the workflows name the wrapper) and both hooks pass `-count=1`, so the stale window is local and add-only. `make comment-history-audit` is the focused run of it, passes `-count=1`, and is what the `wf-codebase-health` rubric's F2 reaches for during an audit.
- **Blocking at the release tag:** `make changelog-audit`, the second job of [`changelog-check.yml`](.github/workflows/changelog-check.yml) — a change to embedded content since the last release that the release's notes cite by no entity id fails the pushed `v*` tag. Its other finding, a shipped-surface commit carrying no `aiwf-entity` trailer, is reported without failing: with no entity named the audit cannot say whether an entry covers it ([the changelog-audit decision](work/decisions/D-0087-the-changelog-audit-blocks-an-uncited-entity-and-reports-an-untrailered-commit.md)).
- **Advisory (code review):** `context.Context` first-arg, no new package-level mutable state, one-line dep justification, deliberate `go.mod` floor bumps, docs, entity bodies, and code comments stating the conclusion rather than the drafting history — the *judgment* half only; the phrase-shaped subset of the code-comment rule is mechanically gated (see the two bullets above).

Relax a blocking rule for a specific call site via the linter's allowlist with a one-line rationale, never a bare `//nolint`.

<!-- aiwf:guidance:START - DO NOT EDIT, regenerated by aiwf update -->
@.claude/aiwf-guidance.md
<!-- aiwf:guidance:END -->

<!-- aiwf:engineering-guidance:START - DO NOT EDIT, regenerated by aiwf update -->
Before using engineering guidance, check for .guidance/.aiwf-pending. If it exists,
report that installation is incomplete and ask the maintainer to rerun aiwf update;
do not treat the installed pack files as a complete policy set.
Otherwise read [.guidance/project.md](.guidance/project.md) first when it exists,
then [.guidance/index.md](.guidance/index.md) and the packs relevant to the task.
Handwritten project overrides take precedence over pack guidance. Load supporting
rubrics only when relevant to the task. Do not also load legacy engineering guidance.
<!-- aiwf:engineering-guidance:END -->
