# Git-native planning: throwing out the event log and the graph

> **Status:** research / first-principles redesign sketch. Not normative. Companion to `fighting-git.md`. Where that document examines what breaks under git operations *given* the current architecture, this one asks: what would we design if we started over and refused to assume an event log or a derived graph at all?
> **Audience:** anyone questioning the foundations of `docs/architecture.md` §2–§3 (the three coordinated representations and the event-sourced kernel).
> **Premise:** the user's instinct to model planning with clay and to garden a backlog rather than execute a waterfall is *exactly* git's native model. The framework's job is to make that ergonomic, not to build a parallel reality alongside it.

---

## 1. Restart: what does the framework actually need to do?

Strip every assumption about *how* the framework works and ask only what it must accomplish for the user. The framework exists to:

1. **Record planning state** — what epics exist, what milestones, what's in flight, what's done, what's blocked, what was decided.
2. **Express relationships** — milestones belong to epics; milestones depend on milestones; decisions ratify scope; gaps motivate new work.
3. **Support evolution** — insert a milestone between two others; rewrite a milestone when a dependency changes; spawn a new epic when a discovered gap is too large; supersede a decision.
4. **Keep history honest** — when a thing changed, who changed it, why.
5. **Validate consistency** — references resolve, status transitions are legal, terminal states stay terminal, cycles don't form.
6. **Generate human-readable views** — the ROADMAP table, dependency diagrams, status reports.
7. **Coordinate AI behavior** — skills, rules, contracts shape how AI assistants act on the project.
8. **Survive parallel work** — multiple humans, multiple AI assistants, multiple branches, possibly weeks of divergent work.

Notice what is **not** on this list: "maintain a totally ordered event log," "keep a derived graph projection," "compute hash-chained provenance." Those are *implementation choices* the original architecture made in service of items 4 and 5. They are not requirements.

If we can satisfy items 1–8 without them, we should — because as `fighting-git.md` showed, those choices fight git, and git is going to win.

---

## 2. The first-principles question: where does state belong?

A repository already has a state-management system: **git itself.** Git tracks every file, records every change, attributes every edit, supports branching, merging, blame, bisect, and time-travel checkout. It is, fundamentally, an event-sourced database with a Merkle DAG of states.

The architecture's mistake — proposing this carefully — is **building a second state-management system inside the first one.** `events.jsonl` is a parallel transaction log. `graph.json` is a parallel projection. Both invent what git already provides, and pay the cost of having to reconcile with git at every merge.

The first-principles alternative: **let git be the state-management system, and let the framework be only what git is not.**

What is git not?

- Git does not understand the *semantics* of a status transition. (`git diff` shows `status: in_progress` became `status: complete`; it does not check whether that transition is legal.)
- Git does not validate **cross-file invariants**. (It will happily commit a milestone whose `depends_on: [M-99]` points at nothing.)
- Git does not allocate **identifiers** that don't collide across branches.
- Git does not produce **rendered views** (the ROADMAP table, the dependency graph).
- Git does not carry **AI-assistant context** (skills, prompts, rules) — they are content git stores, but git does not know they are special.

This list is the framework's actual surface. Everything else can be deleted.

---

## 3. The proposed model — every fact is a file, git is the time machine

### 3.1 Storage

There is one tier of canonical state: **markdown files in `work/` (and equivalents).** Each entity is a file with YAML frontmatter (typed structural fields) and a prose body (narrative). That is *all* that is canonical. There is no event log file, no graph projection file, no separate runtime state file.

```
work/
├── epics/
│   └── E-019-payments-rewrite/
│       ├── epic.md
│       ├── M-001-extract-pricing.md
│       ├── M-002-cutover.md
│       └── decisions/
│           └── D-2026-04-26-001-defer-eu-vat.md
└── gaps/
    └── G-2026-04-26-003-no-fraud-rules.md
```

Each file's YAML frontmatter declares its kind, status, parent, dependencies, dates. The body is prose — opaque to the engine.

### 3.2 History

History is `git log -- <path>`. The framework provides a friendlier renderer:

```
$ aiwf history E-019/M-002
2026-04-26  feat(planning): promote M-002 to in_progress    Peter Bruinsma
2026-04-25  feat(planning): add M-002 (depends on M-001)    claude/session-abc
2026-04-22  feat(planning): rewrite M-002 scope after gap   Peter Bruinsma
```

The renderer reads `git log` plus per-commit YAML diffs to produce structured-event output. The "events" exist — they are just **reified from git** on demand, not stored separately. **The git log *is* the event log.**

### 3.3 Identity

Identifiers are allocated to be collision-free across branches. Three viable schemes (pick one in the proposal that adopts this approach):

- **Content-derived** — `E-` + first 6 chars of SHA-256(title + creation timestamp). Stable across branches if the same entity is created from the same inputs (rare); otherwise distinct. Human-friendly enough.
- **Time-sortable, branch-mixed** — ULIDs prefixed by kind. `E-01HK4YT8…`. Always unique by construction. Less friendly.
- **Sequential per branch + suffix on collision** — `E-19` allocated locally on each branch from a per-branch counter; if both branches' `E-19` survives to merge, one is renamed `E-19a`. Friendly within a branch; merge surfaces the rename as a finding.

In every scheme, **the engine is the allocator within a branch** (so parallel sessions on the same branch don't collide), but **branches do not coordinate** — collision-detection happens at merge time, not at allocation time.

### 3.4 Atomicity (the part that worried us most)

The original architecture introduced trace-first writes (event-before-effect) to handle partial-failure recovery. That problem is real — if a verb edits three files and crashes after two, the working tree is inconsistent.

The git-native solution: **every `aiwf` verb produces exactly one git commit.** The verb writes its file changes to the working tree, validates, and commits in one atomic operation. If the process crashes mid-write, `git status` reveals the partial state and the verb (or `aiwf recover`) can either complete or rollback to the last commit.

Within a single verb's execution, a journal file in `.ai-repo/journal/<verb-id>.jsonl` (gitignored) records intent before file edits, and is deleted on commit success. On startup, orphaned journals are surfaced as "incomplete verb attempts" and the user is prompted to retry or discard. **This is the same trace-first pattern, but localized to crash recovery — not maintained as a permanent ledger.**

### 3.5 Validation

`aiwf verify` is a **pure function** from the working tree to a list of findings. It reads the markdown files, parses frontmatter, walks references, checks invariants. It compares to nothing because there is nothing to compare to — the tree is the truth.

This eliminates the entire category of "drift between projection and event log" findings. Drift was a property of having two stores; with one store, there is nothing to drift from.

What `verify` checks:
- Frontmatter conforms to the kind's contract YAML.
- All references resolve (`depends_on`, `parent`, `supersedes`, `cites`).
- Status transitions implied by `git log` are FSM-legal (e.g., never `complete → in_progress`).
- Terminal states are not modified.
- No reference cycles.
- Cross-entity invariants from contracts (e.g., "an epic in `active` has at least one milestone in `draft`").

All of this is a single pass over the working tree. Sub-second on any project the framework targets.

### 3.6 Verbs

`aiwf` verbs become very simple:

- They edit files.
- They validate the resulting tree.
- They commit, with a structured message.

Example: `aiwf promote M-002 in_progress`:

1. Read `work/epics/E-019/M-002.md`.
2. Update its frontmatter `status: in_progress` and append a `transitions:` log entry with timestamp/actor.
3. Run `aiwf verify --staged`. If findings of severity ≥ error, abort with the findings.
4. `git commit -m "feat(planning): promote M-002-cutover to in_progress"` with a structured trailer (`aiwf-verb: promote`, `aiwf-entity: M-002`, `aiwf-actor: claude/session-abc`).

The structured commit-message trailer is what makes commits machine-greppable for `aiwf history` and audit. **The audit log is the git log.**

### 3.7 Rendered views

The ROADMAP table, dependency graphs, status reports are **never stored**. They are rendered on demand from the working tree:

```
$ aiwf render roadmap > ROADMAP.md   # only if you want to commit a snapshot
```

If a project wants the rendered ROADMAP.md committed for human readability in GitHub's web UI, fine — render-and-commit is a verb. But the source of truth is the underlying entity files. The committed ROADMAP.md is a courtesy, not authority.

This eliminates the "regenerate fenced sections" problem. There are no fenced sections in canonical files — only in optional rendered snapshots.

---

## 4. How this answers the user's specific worries

### 4.1 "Plans change while implementing"

A milestone in flight discovers a gap. The user wants to insert a new milestone, rewrite the next one, or spawn a new epic.

Under the git-native model:

- **Insert** = new file. `aiwf add milestone --depends-on M-002 --before M-003`. New file `M-002a-discovered-gap.md`. Commit. Done.
- **Rewrite** = edit the file. Body changes freely (engine doesn't care). Frontmatter changes go through `aiwf` verbs that validate transitions.
- **New epic** = new directory. `aiwf add epic`. New `epic.md` + an empty milestones directory. Commit. The discovered-from relationship is recorded as a frontmatter field (`discovered_from: G-2026-04-26-003`).

All of these merge cleanly because they touch disjoint files.

The interesting case is **dependency rewrites that span branches**. If branch A says M-005 depends on M-003 and branch B says M-005 depends on M-004, git's per-file merge will conflict on the `depends_on:` line. **This is exactly what should happen** — it is a real semantic conflict that requires a human to decide. The framework's job is to (a) make the conflict visible, (b) provide `aiwf verify` to confirm the post-resolution tree is consistent, (c) surface ripple effects ("M-005 depends on M-004 means this branch can't proceed until M-004 is done").

### 4.2 "Backlog is not a linear list"

Right — and a per-file-per-entity model treats it that way. There is no `backlog.txt` to fight over. Each item is a file. Reordering is just changing `priority:` or `next:` fields, which are small frontmatter edits with low conflict surface. Adding an item is creating a file. The implicit "order" of the backlog is computed at render time from the structural fields — not stored as a global sequence.

### 4.3 "Framework itself diverges across branches"

This is the deepest worry. Branch A pins ai-framework@v1.5; branch B pins v1.7 (which adds a new entity kind). What happens?

The framework lives as a git submodule under `.ai/` (per architecture §11). The submodule's commit pin is part of the consumer's git tree. So:

- Branch A's `.gitmodules` (or submodule pointer) → ai-framework@v1.5.
- Branch B's submodule pointer → ai-framework@v1.7.

Each branch, when checked out, recursively updates the submodule and uses **that branch's pinned framework version**. No divergence within a branch. Behavior is reproducible: `git checkout <commit>` always produces the same framework + content state.

When branches merge:

- The submodule pointer is a single line in `.gitmodules` (and a treeish in the supermodule). Git merges it as text. If both branches updated the pin, you get a conflict — resolve to one version (typically the higher).
- After merge, the framework version is whichever was chosen. It must be able to read the merged content. **This is a version-compatibility question, not a merge-mechanics question.** Standard answer: the framework supports reading content one or two minor versions older; major versions require a migration verb (`aiwf migrate`).

The principle: **the framework version is part of the tree**, so it switches with `git checkout` and merges with `git merge`, like any dependency.

### 4.4 "AI assistant behaves differently on different branches"

This is real and important. Skills, rules, and adapter surfaces in `.claude/skills/`, `.github/skills/`, `CLAUDE.md` are tree state. They differ across branches. An AI assistant reading the tree sees branch-specific instructions.

**Within a single conversation, branch switches change the assistant's effective policies mid-session.** A user might ask "implement the next step" on branch A, then `git checkout B` to test something, and the assistant — if it re-reads the rules — now operates under different rules.

This is *correct behavior*, but surprising. It mirrors what happens with code dependencies: switching branches can change the version of every library you're using, with corresponding behavior changes. We accept that for code; we should accept it for prompts and rules.

What the framework owes:

- **Branch awareness in the assistant's worldview.** When a skill is invoked, the engine should be able to answer "what is the current branch and what version of the framework is pinned?" so the assistant can frame its work accurately.
- **Re-read on branch-change.** A `post-checkout` hook can prompt the assistant to refresh skills and rules. The assistant should not cache rules across a branch switch.
- **Merge-time review of rule changes.** If a PR changes any `.claude/skills/`, `CLAUDE.md`, contract YAML, or other behavior-shaping file, the PR description should call this out (the framework can detect this and flag it). This is normal "review changes to your tooling" hygiene.

---

## 5. What the literature says about branch-divergent AI rules

The user asked specifically: is there research on AI assistants whose agents/skills/rules diverge across branches?

**Honest answer: there is no formal academic treatment of this exact problem that I am aware of.** The community's awareness is practical, scattered across blog posts, GitHub issue threads, and tool documentation. Below is what I know to be relevant, with the caveat that this is a young problem and the literature is thin.

### 5.1 Adjacent work that frames the problem

- **"Prompt as code" / "prompts as artifacts."** A loose movement in the LLM tooling community (LangChain, LMQL, Mirascope, Promptfoo, AgentOps) treating prompt strings as first-class versionable artifacts. Acknowledges branching but does not formalize merge semantics for prompts.
- **In-context learning is sensitive to small prompt changes** (Lu, Bartolo, Moore, Riedel, Stenetorp, "Fantastically Ordered Prompts and Where to Find Them," ACL 2022). Empirical demonstration that even reordering examples in a prompt changes model behavior measurably. By extension: a one-line change to a `.claude/skills/*.md` rule can shift agent behavior in ways that aren't obvious from the textual diff.
- **Constitutional AI and rule-following research** (Bai et al., 2022; Anthropic). Studies how models internalize and follow declarative principles — relevant because skills/rules in the consumer repo *are* declarative principles whose effect is what the assistant does next. Branch-divergent rules are branch-divergent constitutions.
- **AI evaluation under prompt drift.** OpenAI evals, Anthropic evals, and the broader eval ecosystem have recently begun treating "prompt version" as a first-class axis in evaluation. Not branch-aware specifically, but the same fundamental concern.

### 5.2 Practical surfaces with the same problem

- **Cursor `.cursorrules`** — per-repo rules for Cursor's AI. Lives in git, branches with the repo, no formal merge story documented.
- **Aider's `CONVENTIONS.md`** — same shape, same lack of formal treatment.
- **Continue.dev `.continuerc.json`** — same.
- **GitHub Copilot custom instructions** (`.github/copilot-instructions.md`) — same. Note: this is the file the architecture's `.github/skills/` adapters target. It is git-tracked, branch-divergent, and has no merge specification.
- **Claude Code's `CLAUDE.md`, `.claude/skills/*.md`, `.claude/agents/*.md`** — Anthropic's own surfaces. Branch-divergent. The user's framework is one of the more thoughtful attempts to manage this surface declaratively, but the divergence problem is identical.

### 5.3 Why nobody has formalized this yet

A guess: the problem is one to two years old at any scale. Multi-agent repos with branch-divergent rules became common only as IDE-integrated assistants (Copilot, Cursor, Claude Code) shipped widely (2023–2025). The research community hasn't yet caught up to "prompts and rules as a versioned artifact in collaborative codebases."

**The framework being built here is an opportunity to be one of the first articulations of this problem and a worked solution.** The research doc you are reading now plus a future formal write-up of how this framework handles rule divergence would be a contribution to that literature, not a derivative of it.

### 5.4 What we can borrow from elsewhere

- **From software dependency management:** lockfiles + semver. Treat skills/rules as versioned dependencies; pin them; surface upgrades as deliberate diffs.
- **From feature-flag systems:** rule changes can be progressive ("new rule applies to AI sessions after timestamp T"); but this rarely makes sense in a per-branch world where branches *are* the rollout unit.
- **From CRDTs:** for the *content* of rules (markdown), git's text merge is good enough — the semantic merge is the human's job in PR review.
- **From the configuration management literature** (Puppet, Chef, Terraform): declarative state with reconciliation. The framework's verbs are "reconcilers" that keep the tree in a valid state.

---

## 6. What the framework still owes (unchanged from the original)

Dropping the event log and graph does not delete the framework's reason to exist. It still owes:

1. **Schemas (contracts)** — what valid frontmatter looks like per kind. Same as today.
2. **Validators** — the checks `aiwf verify` runs over the tree. Same as today, but stateless.
3. **Verbs** — the safe, atomic operations users invoke instead of editing files by hand. Same shape, but each is now "edit + validate + commit" instead of "append-event + apply-effect + append-confirmation."
4. **Renderers** — ROADMAP, dependency graphs, status reports, on-demand.
5. **Skills (the assistant adapters)** — how AI assistants invoke verbs and reason about state. Same as today.
6. **Crash-recovery journal** — `.ai-repo/journal/` for verbs that crash mid-write. Gitignored, ephemeral, scoped to a single verb attempt.
7. **History renderer** — `aiwf history <id>` over `git log` with structured commit trailers.
8. **Migration** — `aiwf migrate` for framework-version transitions that change the contract YAMLs (rare, deliberate).

What it stops owing:
- The event log file and its lifecycle.
- The graph projection file and its lifecycle.
- Hash-chained linearity.
- Drift detection between two stores.
- A custom merge driver for events.jsonl.
- A monotonic ID allocator that requires cross-branch coordination.

---

## 7. Comparison: this model vs. the architecture.md model

| Concern | architecture.md model | Git-native model |
|---|---|---|
| Source of truth | Ambiguous (markdown vs events.jsonl) | Markdown files only |
| Event log | Append-only `events.jsonl`, git-tracked | None — `git log` *is* the event log |
| Graph projection | `graph.json`, gitignored, hash-verified | None — `aiwf verify` walks the tree directly |
| Drift detection | Hash compare projection vs replay | Concept does not apply — no second store |
| ID allocation | Monotonic per-kind counter | Branch-local + collision suffix on merge, or content-derived, or ULID |
| Cross-branch ID collision | Possible, breaks identity | Resolved at merge time as a finding (or impossible by construction with ULIDs) |
| Atomicity | Trace-first to events.jsonl, then apply | One git commit per verb, journal for in-flight crash recovery |
| History | Replay events.jsonl | `git log` per file, rendered by `aiwf history` |
| Audit | events.jsonl actor/time/payload | git commit author/date + structured trailer |
| Provenance after merge | Hash chain breaks; needs reconciliation | git merge already encodes provenance correctly |
| Behavior under `git checkout` | `graph.json` stale until rebuild | All canonical state switches with the tree; nothing to rebuild |
| Behavior under `git merge` | Hash chain breaks; needs custom merge driver | Per-file merge of markdown; semantic conflicts surface as findings |
| Framework version divergence | Submodule pin; not addressed in arch | Submodule pin; explicitly part of the tree, switches with the branch |
| AI rule divergence | Not addressed | Acknowledged; treated as part of tree; reviewed in PRs |
| Lines of Go code (rough) | Several thousand (eventlog, projection, verify, mutate, …) | Several hundred (parse, validate, edit, commit) |
| External dependencies | None beyond stdlib | None beyond stdlib (or `git2go`/shell-out for git operations) |

The git-native model is materially smaller, materially simpler, and **does not fight git** because it has nothing parallel to git to reconcile.

---

## 8. What we lose (the honest list)

This is not free. Here is what the git-native model gives up vs. the architecture.md model:

1. **Sub-commit atomicity record.** A complex verb that touches five files lands as one commit; the story of "I tried this then confirmed it" is not recorded separately. Mitigation: the journal file during execution provides crash recovery; commit messages provide the after-the-fact narrative.
2. **Action-type provenance after the fact.** Once committed, you know `M-002.status` changed from `draft` to `in_progress`, but you don't have a separate record saying "this was via the `promote` verb." Mitigation: structured commit-message trailers (`aiwf-verb: promote`) preserve this and are greppable.
3. **Replay as a debugging primitive.** Can't say "replay events 1–47 to inspect intermediate state." Mitigation: `git checkout <commit>` is exactly this, with a friendlier UX.
4. **Idempotency keys for distributed retry.** Less of an issue because verbs are now scoped to a single working tree; cross-machine retry isn't really in scope. If it becomes scope, content-addressing of attempted writes is the standard answer.
5. **A clean answer to "what is the order of these two events."** Cross-branch, this question genuinely has no answer in either model — but the git-native model is honest about it (the events are incomparable until merge), while the events.jsonl model lies (it gives them a `seq` number that merges break).
6. **Performance under enormous histories.** If a project has 100,000 entities and 1M historical commits, `aiwf history` walking `git log` is slower than reading a precomputed events.jsonl. Mitigation: the framework targets project-management state, not log-scale data — keep an eye on performance, add caching layers (gitignored, rebuildable) if and when it matters.

These costs are real but bounded. The original model's costs (everything in `fighting-git.md`) are unbounded — they grow with branching, merging, schema evolution, and team size.

---

## 9. UX implications

### 9.1 What the user sees

In the git-native model, the user's interaction is:

```
$ aiwf add milestone --epic E-019 --title "Cutover safety net"
created work/epics/E-019/M-003-cutover-safety-net.md
committed (a4f8c21): feat(planning): add M-003-cutover-safety-net

$ aiwf promote M-003 in_progress
updated work/epics/E-019/M-003-cutover-safety-net.md
committed (b7e9d04): feat(planning): promote M-003 to in_progress

$ aiwf history M-003
2026-04-27  feat(planning): promote M-003 to in_progress    Peter Bruinsma
2026-04-27  feat(planning): add M-003-cutover-safety-net    Peter Bruinsma

$ git checkout main
$ aiwf history M-003
(no commits — this entity does not exist on main)
```

This is consistent with how every other git-tracked thing behaves. There is no "the framework's view of state diverges from git's view" surprise.

### 9.2 What the AI assistant sees

When invoked, the assistant reads the tree (via skills + verbs that query the tree). On branch switch, it re-reads. There is no in-memory state to invalidate beyond the conversation context, which the user owns.

If the assistant needs to know "what did I do in the last hour," it reads `git log --since=1.hour --author=claude` filtered for `aiwf-verb:` trailers. This is exactly the audit story the events.jsonl model promised, with a different storage mechanism.

### 9.3 What CI sees

CI runs `aiwf verify` on every PR. Findings gate the merge. No state to bootstrap; no projection to rebuild. The verify pass is a pure function of the tree at the PR head.

CI can also run `aiwf verify` post-merge (as a job on `main`) to confirm the merge produced a consistent state. If it didn't, the merger is asked to follow up with reconciliation commits.

### 9.4 Conflict resolution

When git surfaces a merge conflict in entity files:

- The user resolves the YAML/text conflict using normal tools (their editor, `git mergetool`).
- They run `aiwf verify` to confirm consistency.
- They commit the resolution.

The framework does not own conflict resolution. It owns *consistency-after-resolution*. This is a much smaller, much more achievable surface.

---

## 10. What we still need to figure out (open questions)

This document does not solve everything. Open questions to address in any proposal that adopts the git-native model:

1. **Where does the per-branch ID counter live?** Options: `.ai-repo/counters.json` gitignored (works within a checkout but loses on fresh clone — recompute by scanning `work/`); a hidden tag or note in git; embed in entity filenames.
2. **What happens to the `aiwf-verb` commit trailer if a user squashes commits?** Squash loses individual verb history. Either accept it (squash is a deliberate rewrite of history) or recommend rebase-merge over squash-merge.
3. **What about non-`aiwf` edits to entity files?** If a human edits `M-003.md` directly in their editor and commits, there's no `aiwf-verb` trailer. `aiwf verify` still validates the result. Should the framework flag direct edits as warnings? Probably not — it would be hostile to user agency. Just verify the result.
4. **How are large refactors (rename a kind, change a field's name) handled?** With a `aiwf migrate` verb that runs codemods on the tree and produces a single migration commit. Same shape as a Rails migration, in spirit.
5. **How does the framework version (in `.ai/` submodule) coordinate with the on-tree contract YAMLs?** Probably: contracts ship in the framework submodule; consumer overrides go in `.ai-repo/contracts-override/`. This needs articulating but isn't different from today.
6. **What is the story for multi-repo planning?** Out of scope here. The architecture's existing answer (each repo is independent; cross-repo coordination via narrative or external systems) carries over.

---

## 11. How this relates to `fighting-git.md`

`fighting-git.md` enumerated tiers of solutions assuming the events.jsonl + graph.json model and asking how to defend it from git's branching. Tier 5 ("Hybrid: declare markdown canonical, treat events as advisory") is the closest sibling to the model proposed here.

This document goes further than Tier 5: it proposes to **delete** the event log and graph rather than demote them. The user's question — "what if I started without those?" — invites that step.

If a future proposal adopts the git-native model, it should:

- Cite this document and `fighting-git.md` as its lineage.
- Take a clear stance on the open questions in §10.
- Show the migration path for any consumer that already has `.ai-repo/events.jsonl` and `.ai-repo/graph.json`.
- Update `docs/architecture.md` §2 (the three coordinated representations becomes one) and §3 (the event-sourced kernel becomes "the validating engine over the working tree").
- Update `CLAUDE.md` "Architectural commitments" — trace-first writes become localized to in-flight verb crash recovery, not a permanent ledger; hash-verified projections become "validated tree, computed views."

It should *not* feel obligated to preserve every architectural commitment. Several of them — trace-first writes to a permanent log, hash-verified projection drift detection, monotonic ID allocation — exist to defend the events.jsonl + graph.json model. If that model goes, those commitments lose their reason and should be revised.

---

## 12. Summary — the move in one paragraph

The original architecture invented an event log and a derived graph because it wanted total ordering, atomicity, hash-verifiable provenance, and replayable history. Git already provides all four — for files. By making the canonical state live exclusively in markdown files (which git understands), the framework gets total ordering, atomicity, provenance, and replay *for free, from git*, and pays no merge-reconciliation cost. The framework's residual job is what git cannot do: validate cross-file semantic invariants, render derived views, and provide ergonomic verbs that produce well-shaped commits. This is a smaller, simpler, and git-native framework that does not fight the substrate it lives on. The cost is giving up sub-commit-granularity provenance and accepting that some merge questions (cross-branch ordering of independent decisions) genuinely have no canonical answer — but the original model lied about both, while this model is honest.

---

## Appendix A — Bibliography (for AI rule divergence)

The author of any proposal adopting this model should cite:

- Lu, Y., Bartolo, M., Moore, A., Riedel, S., Stenetorp, P. (2022). *Fantastically Ordered Prompts and Where to Find Them.* ACL 2022. — Empirical evidence that small prompt changes shift behavior.
- Bai, Y., et al. (2022). *Constitutional AI: Harmlessness from AI Feedback.* Anthropic. — Models internalizing declarative principles; relevance to "branch-divergent constitutions."
- Kleppmann, M., Wiggins, A., van Hardenberg, P., McGranaghan, M. (2019). *Local-First Software.* Onward! 2019. — Branches as legitimate concurrent realities.
- Kleppmann, M., Beresford, A. R. (2017). *A Conflict-Free Replicated JSON Datatype.* IEEE TPDS. — For comparison: the merge-aware substrate alternative.
- (Practical) Documentation for Cursor `.cursorrules`, Aider `CONVENTIONS.md`, GitHub Copilot custom instructions, Claude Code `CLAUDE.md` and `.claude/skills/`. — All exhibit the same branch-divergence problem; none formalize a solution.

The lack of academic treatment is itself a finding. A formal write-up of how this framework handles rule divergence would, to the author's knowledge, be among the first.

---

## Appendix B — Glossary

| Term | Meaning |
|---|---|
| Git-native model | The proposal in this document: markdown files as sole canonical state, git as the time machine, framework as validator + verbs + renderers. |
| Working tree | The current state of files in the user's checkout; in this model, the entirety of canonical state. |
| Verb | A framework operation (`aiwf add`, `aiwf promote`, etc.) that edits the tree, validates, and commits in one atomic step. |
| Journal | Ephemeral, gitignored crash-recovery record for an in-flight verb. Distinct from a permanent event log. |
| Structured commit trailer | A line in a git commit message like `aiwf-verb: promote` that makes commits machine-greppable for audit and history rendering. |
| Branch-divergent rules | The phenomenon of `.claude/skills/`, `CLAUDE.md`, contract YAMLs differing across branches, causing AI assistants to behave differently per branch. |
| Reified event | An "event" computed on demand from `git log` + per-commit YAML diffs, rather than stored as a persistent record. |
