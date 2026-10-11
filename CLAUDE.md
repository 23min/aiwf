# CLAUDE.md — ai-workflow repo

This repo carries `aiwf` — a small framework that helps humans and AI assistants track what's planned, decided, and done, by validating mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/design/design-decisions.md`](docs/design/design-decisions.md) for what aiwf commits to and §"What's deliberately not in the PoC" for scope limits. Gaps live as entities under `work/gaps/` — run `aiwf list --kind gap` or `aiwf show G-NNNN`.

---

## Engineering principles

- **Framework correctness must not depend on LLM behavior.** Skills are advisory; the pre-push hook and `aiwf check` are authoritative. A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee.
- **Provenance is principal × agent × scope, not just operator.** When the human directs the LLM in conversation, the LLM is a *tool* (human = principal, no co-actor inflation). When the human runs `aiwf authorize E-NNNN --to ai/claude`, the agent works within that scope until the scope-entity is terminal or the human pauses. `--force` is sovereign — humans only. See [`docs/design/provenance-model.md`](docs/design/provenance-model.md).

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

## Working in this repo

**Trunk-based development on `main` for maintainers.** Commit directly to trunk; no PR ceremony. Validation is mechanized: `aiwf check` runs pre-commit (shape-only) and pre-push (full), and CI runs the full matrix. Outside contributors use GitHub PRs (see [`CONTRIBUTING.md`](CONTRIBUTING.md)). Conventional Commits subjects are mandatory both paths. When in doubt, the smaller change is the right change.

**Derived artifacts — regenerate, don't hand-edit.** Change the source, then regenerate:
- **`ROADMAP.md`** — committed, regenerated by `aiwf render roadmap` (the trap: it's tracked but not hand-maintained).
- **`STATUS.md`** — gitignored, regenerated by the post-commit hook (`status_md.auto_update`).
- **`site/`** — gitignored, the `aiwf render --format=html` output.
- Materialized host artifacts — regenerate through `aiwf init`/`update`; see the [host setup guide](README.md#2-host-setup-and-embedded-rituals) for selected-host paths and managed root guidance.
- **Not derived:** the embedded ritual snapshot under `internal/skills/embedded-rituals/` is the *authoring source of truth* — you hand-edit it (see `.guidance/project/shipped-content.md`).

---

## Default to a worktree for any branch work in this repo

When a ritual (`wf-patch`, milestone, epic) or an ad-hoc fix creates a branch, default to an in-repo worktree (`.claude/worktrees/<branch>/`, per [the worktree-placement ADR](docs/adr/ADR-0023-default-to-in-repo-worktree-placement-under-claude-worktrees.md)) rather than switching the main checkout in place — create it with `aiwf worktree add <branch> --base <base>`, which materializes the selected hosts’ supported artifacts into the new worktree atomically, in one step. This applies to the session's own direct work, not just subagent dispatch. Skip only when explicitly told to work in the main checkout for that invocation.

---

## Go conventions

### Commit conventions

Subjects follow Conventional Commits (`feat(plan): …`, `chore(plan): …`, `docs(adr): …`) — say *why*, one logical change per commit.

**No `Co-Authored-By:` line names a non-human agent** ([the co-author decision](work/decisions/D-0096-refuse-ai-co-author-trailers-behind-an-opt-in-address-list.md)). The agent that ran a verb is recorded in `aiwf-actor:`, and the principal it acts for is the human; a co-author line is a second copy of that fact which nothing re-derives, and it contradicts the provenance model's reading of an LLM directed in conversation — a tool, not a co-author. This **overrides an assistant harness's own attribution default**: where a harness instructs the session to append such a line, this rule wins and the line is omitted. A human co-author is unaffected. Enforced rather than remembered: `provenance.refuse_coauthors` in `aiwf.yaml` lists the addresses, the `commit-msg` hook refuses one at composition, and `coauthor-trailer-ban` (`internal/policies/`) judges the same property over the pushed range for the cases the hook cannot reach — an uninstalled hook, `--no-verify`, or a hook shelling a release older than the refusal. The knob is absent by default, so a consumer repo is unaffected until it names an address.

### Type design

- **The six kinds and their status sets are hardcoded in Go**, not external YAML (deferred until a real consumer needs custom vocabulary).

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
