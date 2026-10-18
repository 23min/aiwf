# CLAUDE.md — ai-workflow repo

This repo carries `aiwf` — a small framework that helps humans and AI assistants track what's planned, decided, and done, by validating mechanical guarantees about a markdown-and-frontmatter project tree. Read [`docs/design/design-decisions.md`](docs/design/design-decisions.md) for what aiwf commits to and §"What's deliberately not in the PoC" for scope limits. Gaps live as entities under `work/gaps/` — run `aiwf list --kind gap` or `aiwf show G-NNNN`.

---

## Engineering principles

- **Framework correctness must not depend on LLM behavior.** Skills are advisory; the pre-push hook and `aiwf check` are authoritative. A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee.

---

## What aiwf commits to

Load-bearing properties any change must preserve, distilled in [`docs/design/design-decisions.md`](docs/design/design-decisions.md). If a change doesn't preserve one, treat it as a kernel-level decision and surface it — not a quiet refactor.

1. **Six entity kinds** — epic, milestone, ADR, gap, decision, contract — each with a closed status set and one Go function for its legal transitions, hardcoded in Go rather than read from YAML until a consumer needs its own vocabulary.
2. **Stable ids that survive rename, cancel, and collision.** The id is the primary key and the slug is display, so a rename keeps the id; removal is a terminal status, not deletion; `aiwf check` detects a collision and `aiwf reallocate` resolves it. Ids emit at canonical 4-digit width ([the canonical-width ADR](docs/adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md)), and parsers accept narrower legacy widths permanently, since no verb widens an id in place; a narrow id in the active tree is an error, `entity-id-narrow-width`.
3. **`aiwf check` runs as a pre-push git hook.** Validation is the chokepoint that makes the guarantees real.
4. **`aiwf history <id>` reads `git log`.** No separate event log; structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) make the log queryable.
5. **Selected-host artifacts, refreshed explicitly.** `aiwf init` and `aiwf update` select Claude Code, Codex, both or neither, preserving user content and the artifacts of hosts no longer selected; the core Git hooks are host-independent. The [host setup guide](README.md#2-host-setup-and-embedded-rituals) holds the paths, the root wiring and its opt-outs.
6. **Layered location-of-truth.** Engine binary external (`go install`); per-project policy/planning state in the consumer repo; materialized skill adapters in the consumer repo but gitignored.
7. **Every mutating verb produces exactly one git commit, or none.** A failed mutation aborts before the commit; a request already satisfied is a NoOp at exit 0 with no commit ([the same-status ADR](docs/adr/ADR-0036-same-status-fsm-transitions-converge-to-noop-not-refusal.md)); `metadata.commit_sha` is present only when a commit was made.
8. **Acceptance criteria as namespaced sub-elements of milestones; TDD opt-in per milestone.** ACs are addressed by composite id `M-NNNN/AC-N`, validated by `aiwf check`, with the `acs-tdd-audit` rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`.
9. **Principal × agent × scope provenance.** The principal, who is accountable, is always human; the actor who ran the verb may be an LLM or a bot. An LLM the human directs in conversation is the human's tool, not a co-actor; `aiwf authorize` opens a scope an agent works within. `--force` requires a human actor. Identity comes from `git config user.email`, never `aiwf.yaml`. The [provenance model](docs/design/provenance-model.md) holds the rest.
10. **Uniform archive convention for terminal-status entities** ([the archive ADR](docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)). `aiwf promote` and `aiwf cancel` flip status only; `aiwf archive` sweeps terminal entities into the `archive/` directory beside them in one commit, a milestone moving with its epic. The loader resolves archived ids, so references stay live. Nothing reverses an archive: file a new entity that references the archived one.

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

`make check-fast` runs `go vet`, the linters and the tests without the race detector; `make ci` is the CI-parity gate, adding the race detector, coverage, the profile-driven gates and the self-check. CI's full gate runs on pull requests and pushes to `main` that touch Go, build or guidance files; a push to an epic or milestone branch runs only the gitleaks scan.

**Cadence.** Run `make check-fast` in the inner loop, a milestone wrap into its epic branch included, and `make ci` at the boundaries that integrate or leave the machine: the epic-to-`main` merge and every push. Skip a run when none of these changed since the last green one: Go and module files, `Makefile`, `.golangci.yml`, `.github/workflows/*`, `internal/**/embedded*` and the guidance (`CLAUDE.md`, `AGENTS.md`, `.guidance/**`); planning-state commits may skip it. A wrap merge may use `git commit --no-verify`, since its tree is the one already validated. Never `--no-verify` a push: that skips the gitleaks scan, whose boundary is the push.

---

## Working in this repo

**Trunk-based on `main`, code on branches.** Planning-state verbs, release commits and doc edits may commit directly to `main`. Every code change goes on a branch in a worktree — `wf-patch` for a small fix, a ritual branch otherwise — and merges into `main` locally, without a PR. Outside contributors use GitHub PRs ([`CONTRIBUTING.md`](CONTRIBUTING.md)). Conventional Commits subjects on both paths. `aiwf check` runs shape-only at pre-commit and in full at pre-push.

**Regenerate derived artifacts; never hand-edit them.** `ROADMAP.md`, which is tracked, by `aiwf render roadmap`; `STATUS.md` by the post-commit hook; `site/` by `aiwf render --format=html`; materialized host artifacts by `aiwf update`. The embedded ritual snapshot under `internal/skills/embedded-rituals/` is not derived: it is the hand-edited source (`.guidance/project/shipped-content.md`).

---

## Default to a worktree for any branch work in this repo

Create every branch, a ritual's or an ad-hoc fix's, in an in-repo worktree under `.claude/worktrees/<branch>/` ([the worktree-placement ADR](docs/adr/ADR-0023-default-to-in-repo-worktree-placement-under-claude-worktrees.md)) with `aiwf worktree add <branch> --base <base>`, which materializes the selected hosts' artifacts into it. This covers your own work, not only a subagent's; work in the main checkout only when told to for that invocation.

---

## Commit conventions

Subjects follow Conventional Commits (`feat(plan): …`, `chore(plan): …`, `docs(adr): …`) — say *why*, one logical change per commit.

**No `Co-Authored-By:` line names a non-human agent** ([the co-author decision](work/decisions/D-0096-refuse-ai-co-author-trailers-behind-an-opt-in-address-list.md)). This overrides an assistant harness's own attribution default: omit the line it asks for. A human co-author is unaffected. `provenance.refuse_coauthors` in `aiwf.yaml` lists the refused addresses; the `commit-msg` hook refuses them at composition, and `coauthor-trailer-ban` reports them over the pushed range.

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
