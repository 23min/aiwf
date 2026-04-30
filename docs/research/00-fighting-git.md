# Fighting git: branching, merging, and the limits of a totally-ordered event log

> **Status:** defended-position
> **Hypothesis:** A totally-ordered hash-chained event log layered onto git fights git's branching model and cannot be made to survive merges by construction; the framework must either lift the substrate (CRDT), lower the abstraction (markdown-canonical), mediate at merge (custom merge driver), or concede to git (use `git log` as the event log).
> **Audience:** anyone proposing changes to the event log, ID allocator, projection-hash chain, or `aiwf verify`.
> **Premise:** the framework as designed in [`docs/architecture.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md) is examined for consistency with git's branching model.
> **Tags:** #thesis #git #aiwf #state-model #research

---

## Abstract

The framework's original architecture persists structural state in three coordinated artifacts: markdown specs, an append-only `events.jsonl`, and a derived `graph.json` projection. The event log is specified as a totally ordered sequence with monotonic sequence numbers and a hash chain over post-state. Git, however, is a Merkle DAG of file-tree snapshots in which branches are first-class persistent divergent histories. This document shows mechanically why the architecture's chosen invariants — total order, hash-chained linearity, monotonic IDs — do not compose cleanly with git's text-merge model: sequence numbers collide, hash chains break, and the monotonic ID allocator behaves like a multi-master replication primitive without coordination. ("Fighting git" in the title is shorthand for *"the original invariants do not compose with git's merge model"* — the doc is not arguing that custom merge drivers, lock-file mergers, or other text-in-git extensions are themselves illegitimate.) It surveys the relevant literature (CRDTs, local-first, patch theory, Bayou-style application-defined merge), then enumerates a tiered solution space (lift / lower / mediate / concede) without picking. The follow-on documents in this series do the picking; this one names the substrate problem and the admissible responses.

---

## 1. Problem statement

The framework persists structural state in three coordinated artifacts inside a consumer's git repository (`docs/architecture.md` §2.2):

| Artifact | Path | Tracked? | Role |
|---|---|---|---|
| Markdown specs (frontmatter + body) | `work/**/*.md`, `docs/decisions/*.md` | yes | Declared **source of truth** (§2.2) |
| Append-only event log | `.ai-repo/events.jsonl` | yes | "Durable kernel" (§2.2); also described as canonical at runtime (§3) |
| Graph projection | `.ai-repo/graph.json` | gitignored | Derived read-model with a SHA-256 over RFC 8785 canonical form |

The event log is specified as a totally ordered sequence. Each event carries a monotonic `seq`, a logical post-state hash (`post_state_hash`), and an actor; mutations are trace-first (event appended **before** any side-effect) under `flock`+`O_APPEND`. Entity IDs are allocated by an engine-side monotonic counter per kind (`E-19`, `M-NN-…`, `ADR-NNNN`, `D-YYYY-MM-DD-NNN`, `G-…`).

Git, however, is not a totally ordered system. It is a Merkle DAG of file-tree snapshots. **Branches are first-class, persistent, deliberately divergent histories.** A merge is a 3-way textual operation between two such histories and their common ancestor.

The framework's promises (total order, hash-chained linearity, monotonic IDs) are properties of *file content*. Git merges file content. Therefore: **the framework's promises do not survive git merges *as currently specified*; some form of framework participation in the merge is required to preserve them.**

This document examines the consequences in detail, names the relevant theoretical frameworks, surveys related systems and literature, and proposes a tiered space of solutions. It does not pick one — that is the job of the per-tier architecture proposals it seeds.

---

## 2. Scenarios — what actually happens

We walk four scenarios, in order of increasing pain. For each we say what happens to (a) markdown, (b) `events.jsonl`, (c) `graph.json`, (d) ID allocator, (e) `aiwf verify`.

### Scenario A — Branch switch, no merge

User on `feat/x` runs `aiwf add milestone …`. Then `git checkout main`.

- (a) Markdown: git removes the new entity files. Correct.
- (b) `events.jsonl`: revision changes; the appended events are gone from the working tree. Correct for that branch's worldview.
- (c) `graph.json`: gitignored, so the working copy reflects `feat/x`. **Stale.** Future `aiwf` invocations would see a projection that disagrees with the on-disk log.
- (d) Allocator state: depends on where the counter is persisted. If it lives in `.ai-repo/config/` and is gitignored, it's also stale. If tracked, switches with the branch (correct).
- (e) `aiwf verify` if run immediately after checkout: would report drift between projection and event log.

**Tractable.** Solved by a `post-checkout` git hook that runs `aiwf rebuild` (regenerate `graph.json` from `events.jsonl`).

### Scenario B — Merging main into a feature branch (or vice versa)

Both sides have appended new events to `.ai-repo/events.jsonl` since the merge base.

- (a) Markdown: git's per-file 3-way merge handles most cases. New entities live in separate files, so most pairs do not conflict. Same-entity edits to frontmatter conflict like any YAML edit conflict — surfaced normally. **Mostly OK.**
- (b) `events.jsonl`:
  - Default git merge of an append-only file usually concatenates without textual conflict. The resulting file is *syntactically* valid JSONL.
  - **`seq` numbers collide.** Both sides allocated `seq=42, 43, 44…` from the merge base.
  - **`post_state_hash` chain breaks.** Each event recorded the canonical hash of the projection *as observed when written*. After concatenation, replaying the merged log produces, at every step past the merge base, a projection whose hash matches *neither* recorded hash.
  - **Total ordering is destroyed.** §13 of the architecture promises "the event log preserves total ordering"; after a textual 3-way merge the order is whatever git produced — not a meaningful causal order.
- (c) `graph.json`: gitignored; will be rebuilt. The rebuild itself is mechanical; the *meaning* of the rebuilt projection is the question.
- (d) Allocator: if both sides created `E-19` (different entities, same id), we get either two directories `work/epics/E-19-foo/` and `work/epics/E-19-bar/` (git directory merge: usually fine because paths differ), or a true path collision (two `work/epics/E-19-foo/epic.md` from different branches). **Identity collision is possible by construction.**
- (e) `aiwf verify`: under current spec, every event past the merge base is drift. The findings are not wrong — but they are not actionable either, because the drift is *expected* after a merge and the framework has no vocabulary to say so.

**This is the core failure mode.** The hash chain's linearity is a content property, but content is what git merges.

### Scenario C — Main is far behind; branch carries weeks of work

Same as B, amplified. Additionally:

- The framework version on main may not understand action types the branch's events use (schema added in the branch). The architecture's "errors are findings, not parse failures" principle (CLAUDE.md, §"Engineering principles") helps — the engine should still load — but `aiwf verify` will report many findings.
- Stale gitignored state (allocator counter, snapshots) compounds.

### Scenario D — Concurrent PRs, rebase before merge

Two PRs both modify state, both rebase onto main before merge. Rebase rewrites the branch's event log on top of main:

- The replayed events get new timestamps (or keep original — depends on choice), new `seq` numbers (forced renumbering during rebase requires the framework to rewrite the file), and new `post_state_hash` values (because they replay onto a different base projection).
- Provenance is degraded: the actor and original time are preserved only if rebase is content-aware.
- "What actually happened, when, by whom" becomes ambiguous.

### What the markdown buys us

Of the three artifacts, **only markdown survives git merges with semantics intact**, and only because:

- entities live in separate files (most cross-branch work touches disjoint sets);
- frontmatter is a small, declarative YAML block;
- bodies are prose, which git's text merge handles in a domain-appropriate way.

The architecture's hierarchy of derivability — `graph.json` from `events.jsonl`, `events.jsonl` from markdown — means that **if the event log is corrupted by a merge, the framework can in principle rebuild it from a rescan.** This is the saving grace.

---

## 3. Is the framework "fighting git"?

**Yes, in two specific places. Not structurally — provided one ambiguity is resolved.**

### 3.1 The two places it fights git

1. **Monotonic ID allocation across uncoordinated writers.** Functionally identical to auto-increment primary keys in a multi-master replicated database. UUIDs solved this in databases for the same reason.
2. **The `seq` and `post_state_hash` chain inside `events.jsonl`.** These encode total order as a property of file content; file content is what git merges. The chain cannot be linear across a merge boundary *under the architecture's chosen invariants* — without a custom merge driver, content-derived event IDs, or a different on-disk format, the linearity property does not hold.

### 3.2 The unresolved design ambiguity

`docs/architecture.md` §2.2 declares **markdown the source of truth**. §3 (the event-sourced kernel) and the §"Architectural commitments" in `CLAUDE.md` (trace-first writes, recovery-is-forward-only-via-the-trace) treat **`events.jsonl` as canonical at runtime**. Both stances are defensible. The framework currently reads as one in §2 and the other in §3+. Every merge question has two answers depending on which paragraph you privilege.

The two stances:

- **Stance A — Markdown is canonical.** `events.jsonl` is a runtime cache and audit trail. On merge it is expected to be regenerable; trace-first writes still apply within a branch but the log is not a god-log. Provenance (actor, timestamp, intent) becomes a best-effort property, not a hard guarantee, after a merge.
- **Stance B — `events.jsonl` is canonical.** Markdown is itself derived (or co-canonical with explicit reconciliation). This stance owes a real story for how the log survives concurrent writers across branches: custom merge driver, branch-aware seq, possibly a different on-disk format altogether.

Resolving this ambiguity is a **prerequisite** to picking a tier in §7. Different tiers belong to different stances.

### 3.3 Where it is *not* fighting git

The graceful-degradation hierarchy means the framework is **not reinventing the canonical store**. Git is still the time machine; markdown is still git-tracked; the framework is maintaining derived stores. Indices need rebuild-on-divergence stories. Yours will too.

The architecture also explicitly disclaims being a git replacement (§13: "Branches, commits, and merges are the assistant's job using normal git tooling. The framework records *which* milestone a branch corresponds to, not the contents of the branch.") This is the right disclaimer; the gap is that §13 does not yet say what *does* happen across git operations.

---

## 4. Theoretical framing

This problem has been studied for decades under multiple names. We name the relevant frames so future proposals can be precise.

### 4.1 ACID — within and across branches

- **Within a single branch**, the framework gets ACID for free from the trace-first protocol + `flock` + `O_APPEND` write semantics. Each verb is atomic; invariants hold; isolation between sessions on the same checkout is serialized through the lock; durability is git + filesystem.
- **Across branches, isolation breaks at merge.** The framework commits to invariants (one entity per id, FSM-legal status transitions, hash-chained log). The merge operator (git's textual 3-way merge) is unaware of those invariants. Result: post-merge state can violate them silently.

This is the classic **multi-master replication serializability problem**. See Bayou (Terry et al., 1995) for the canonical "application-defined merge procedures" model.

### 4.2 CAP — branches are partitions, but permanent ones

CAP states that under network partition, a system must choose between Consistency and Availability. Git's twist: **partitions are first-class and deliberate, not transient failures.** A branch is not a network split waiting to heal; it is a chosen alternate reality.

While "partitioned," each branch is internally Available and internally Consistent. On merge, you reconcile. So the framework is not really choosing CP or AP — it is conceding that **all writes are concurrent until proven otherwise**, and convergence is an explicit, application-aware step.

This places the framework squarely in the **local-first software** model (Kleppmann, Wiggins, van Hardenberg, McGranaghan, 2019). Local-first treats network partition as the normal case and convergence as application-defined.

### 4.3 CRDTs — the convergence story

A **Conflict-free Replicated Data Type** (Shapiro, Preguiça, Baquero, Zawirski, 2011) is a data type whose operations are designed to commute, so concurrent replicas converge to the same state regardless of the order in which they observe operations.

Applied to this framework:

- **Append-only event sets** are trivially CRDTs (G-Set: union always converges). If `events.jsonl` were treated as a *set* of events keyed by content-derived ID, merge would converge automatically.
- **The `seq` field is not a CRDT.** It encodes a global ordering decision that cannot commute.
- **The `post_state_hash` chain is not a CRDT.** It encodes a linear causal history that branching destroys.
- **The ID allocator is not a CRDT.** Monotonic per-kind counters collide under partition.

The repair work is, essentially, **demoting the non-CRDT properties to branch-local properties** and providing application-level merge logic for the joins. This is the Bayou pattern.

### 4.4 Operational Transformation and patch theory

**Operational Transformation** (Ellis & Gibbs, 1989) is the model behind Google Docs: operations are transformed against concurrent operations to preserve user intent. Less directly applicable here because git already gives you snapshot-based concurrency, not operation-based.

**Patch theory** (Pijul; Mimram & Di Giusto, 2013) treats patches as morphisms in a category, with merges as pushouts. Merges become well-defined by construction (no conflicts in the git sense — only patches that don't compose). This is the mathematically rigorous version of "what does merging two event logs even mean."

### 4.5 Lamport time and causal order

`seq` is a global counter. It cannot be globally consistent without coordination (Lamport, 1978). Replacing `seq` with a **vector clock** or a **commit-graph-derived causal hash** removes the cross-branch coordination requirement: two events are concurrent if neither is in the other's causal past, and the framework can choose how to order them deterministically when needed (e.g., by `(actor, content_hash)` tiebreak).

### 4.6 "Branches are CRDTs for files"

Git's per-file 3-way merge already behaves like a CRDT for line-structured text. This is why the markdown layer survives merges: the encoding is already merge-aware. Where the framework makes a mistake at this layer, it is in layering a non-CRDT abstraction (a totally ordered hash-chained log) on top of a mergeable substrate without a corresponding merge story.

The architectural choice space, then, is:

1. **Lift the substrate**: replace `events.jsonl` with a CRDT-aware encoding (Automerge, Yjs, etc.).
2. **Lower the abstraction**: drop the linearity claim, encode events so the existing substrate (text-in-git) handles them naturally.
3. **Mediate the boundary**: keep both, but provide a custom merge driver that implements the join.
4. **Concede entirely**: stop having a separate event log; let `git log` *be* the event log.

§7 elaborates each option.

---

## 5. UX and semantic stability — why this matters beyond correctness

Correctness is not the only axis. Even a theoretically correct merge story will fail in practice if it surprises users at the wrong moment.

### 5.1 The user's expected model

Users (humans and AI assistants) expect git to behave like git:

- "I switch branches and the world changes." Familiar.
- "I merge and conflicts surface as text." Familiar.
- "I rebase and history is rewritten." Familiar but already a known foot-gun.

The framework should honor these expectations. Any solution that makes branch operations feel non-git-like (e.g., "you must run `aiwf checkout` instead of `git checkout`") will be ignored, worked around, or break when CI / IDEs / hooks do plain git operations.

### 5.2 Semantic stability — what "the same project" means

A user expects that two checkouts of the same commit produce the same observable state. Today this is true for tracked artifacts; not for `graph.json` (gitignored, may be stale from a prior branch). Any tier must preserve **commit-determinism**: same commit ⇒ same engine-visible state, modulo a deterministic rebuild step.

A reasonable expectation is that **merging does not silently change semantics of past entities**. If a branch's E-19 and main's E-19 collide and one is renamed to E-19a, every reference (including in prose, including in CHANGELOG.md, including in PR descriptions) must be updated or the framework must surface the inconsistency. This is the *propagation preview* concept in the existing architecture, but it currently does not contemplate merge as a propagation event.

### 5.3 Failure modes and their human cost

A taxonomy of how merge problems can hurt users:

| Failure | Cost | Recovery |
|---|---|---|
| Silent drift (verify finds nothing wrong, state is wrong) | High — users build on bad state | Hard — requires forensics |
| Loud drift (verify reports findings) | Low if findings are actionable; high if not | Easy if framework gives a verb to resolve |
| Path collision (two branches created the same id) | Medium — git surfaces the collision but framework must resolve | Manual unless the framework owns rename |
| Lost provenance (rebase rewrites timestamps/actors) | Variable — depends on whether anyone cares | Audit trail in PR description, not in events.jsonl |
| Hash-chain breakage post-merge | Low if expected; high if treated as corruption | Document; teach `verify` about merges |
| Schema-version skew (branch uses new event types) | Medium — main can't replay branch's log | Schema migration, versioned events |

**The biggest UX risk is the second-to-last row.** If `aiwf verify` after every merge screams about drift, users will learn to ignore it, defeating the purpose. The framework must distinguish **expected post-merge reconciliation findings** from **actual drift findings**.

### 5.4 The AI-assistant angle

The framework is consumed by AI assistants. AI assistants are particularly vulnerable to:

- **Stale projections cached in conversation context.** If an assistant ran `aiwf query` before a merge and reasons from that result after, it will be wrong. The framework should make freshness checks cheap and obvious.
- **Identity ambiguity.** If `E-19` means different things on different branches and an assistant generates a PR description referencing `E-19`, ambiguity propagates into prose. References should be resolvable to commit-stable identities.
- **Verbosity of merge reconciliation.** A long list of findings after every merge will eat context budgets. Reconciliation findings benefit from being batched and summarizable.

---

## 6. Constraints any solution must respect

Lifted from `CLAUDE.md` and the architecture document. A solution that violates these is not admissible without a corresponding architecture amendment.

1. **The engine never generates prose.** Reconciliation surfaces structured findings, not "smart commit messages."
2. **The assistant never writes the projection or event log directly.** Reconciliation is a verb (`aiwf reconcile` or similar), not free-form editing.
3. **Trace-first writes within a branch.** Whatever solution is chosen, mutations within a single linear history still record events before applying effects.
4. **Hash-verifiability of the projection.** The post-merge projection must be canonicalizable and hashable. The hash *chain* may be branch-local; the *projection hash itself* remains a verifiability primitive.
5. **Closed-set vocabularies in YAML, validated in Go.** Merge logic that requires a Turing-complete merge spec per kind is a smell — escalate it to engine code, not contract YAML.
6. **Engine is invocable without an AI assistant.** Merge reconciliation must be drivable by `gh`, by `make`, by a human at the CLI.
7. **Errors are findings, not parse failures.** Post-merge inconsistencies are findings; the engine still loads, still answers queries, still runs verify.
8. **No half-finished implementations.** A tier ships with: spec, engine code, test fixtures, doc updates, CHANGELOG entry, and a migration plan if it changes on-disk format.

---

## 7. Solution space — tiered

Each tier is a coherent direction. Tiers are not strictly ordered; (a) is least invasive, (g) is most. Pick after reading the literature in §8.

### Tier 1 — Mediate: stay text-in-git, teach git the framework's join

**(a) Custom git merge driver for `events.jsonl`.**

`.gitattributes` registers `events.jsonl merge=aiwf-eventlog`. The driver is a verb: `aiwf merge-eventlog --base <path> --ours <path> --theirs <path>`. It performs:

1. Load three sides (base, ours, theirs).
2. Compute the set difference: events in `ours \ base` and `theirs \ base`.
3. Union, deduplicating by content-derived event ID (e.g., SHA-256 of canonical payload + parent commit).
4. Order the union deterministically (causal order from git commit DAG; tiebreak by `(actor, content_hash)`).
5. Renumber `seq` in the merged file (or drop `seq` in favor of the content-derived ID).
6. Append a synthetic `events.merge.applied` event recording: parent commits, conflict resolutions (especially id collisions), the new post-state hash.
7. Recompute the projection.

**Pros:** stays in git's intended extension model (merge drivers are how lockfiles, generated files, etc. are handled across the ecosystem); preserves text-in-git readability; minimal on-disk format change; ships in a few hundred lines of Go.
**Cons:** the driver has to live somewhere and be installed per-clone (`aiwf init` writes `.gitattributes` and a path-to-binary); rebase replays events one at a time and the driver must handle each replay step; doesn't compose well with non-git tools that don't understand merge drivers (e.g., GitHub web UI's merge button uses a server-side merge that may or may not respect drivers — verify per host).
**Open risks:** does GitHub's server-side merge invoke custom drivers? (Mostly **no**.) If users rely on the green button, the driver is bypassed; the framework must detect post-merge inconsistency and reconcile after the fact.

**(b) Branch-aware ID allocation.**

Two sub-options that compose:

- **Local counter + collision suffix.** Each branch allocates from a per-branch counter. On merge, the driver detects collision (same id, different content-hash) and renames one side (e.g., `E-19` → `E-19a`). The existing example schema for sibling variants (`E-NN[a-z]?`) generalizes to merge-induced renames.
- **Content-addressed IDs** (ULID, k-sortable UUID). Collision-free by construction. Loses the human-friendly `E-19` shorthand. Mitigate with a "display id" mapped from a stable content id.

A hybrid (human-friendly within a branch, suffixed at merge) preserves the assistant's UX while removing collision-as-correctness-bug.

**(c) Demote the hash chain.**

Replace "globally linear `post_state_hash` chain" with "branch-local linear chain, joined at merge events." The merge driver's synthetic `events.merge.applied` event terminates one chain and begins a new one with a `parent_hashes: [hash_main, hash_branch]` field. `aiwf verify` walks the chain as a DAG, not a list.

**(d) `aiwf rebuild` and post-checkout / post-merge git hooks.**

`aiwf init` installs `post-checkout` (rebuild `graph.json`), `post-merge` (run merge-aware verify), `post-rewrite` (rebase support). Hooks are idempotent and fast (target sub-second).

**Tier 1 verdict.** Among the tiers, this is the smallest viable fix. It keeps the architecture's bones intact, addresses the git incompatibilities named in §3, and is incrementally shippable. It is a reasonable fallback if the team is not willing to take a dependency on a CRDT library.

### Tier 2 — Lift: replace the substrate with a CRDT

**(e) Replace `events.jsonl` with an Automerge document** (`.ai-repo/events.automerge`).

Automerge (Kleppmann & Beresford, 2017) is a JSON CRDT with formal convergence guarantees. Two replicas of an Automerge document with disjoint edits merge automatically and deterministically, regardless of the order in which they observe each other's changes.

**Pros:** every concurrent-write concern in §2 evaporates; provenance is preserved (Automerge tracks per-change actor and lamport-style timestamps natively); merges are automatic, not application-defined.
**Cons:** on-disk format is no longer human-readable (binary); take a dependency on the Automerge ecosystem (Rust core, language bindings); lose the "tail an event log with `cat`" property; binary diffs in PRs are unreviewable, requiring tooling for "show me the events this PR adds."
**Open risks:** Automerge's invariants are about *convergence*, not about *application invariants* — the framework still owes a story for "exactly one E-19" if it wants that property. So Automerge solves the merge of the log, but not necessarily the merge of the meaning.

### Tier 3 — Theoretically rigorous: patch theory

**(f) Pijul-style patch model for events.**

Each event is a patch over the projection, with explicit dependencies. Patches that touch disjoint state commute; patches that touch overlapping state must declare conflicts explicitly. Merges are pushouts; well-defined by construction.

**Pros:** mathematically gorgeous; closest to "actually solving the problem."
**Cons:** research-grade; very few production examples (Pijul itself is the main one); team would be doing significant primary engineering; tooling ecosystem is thin.
**Verdict:** unlikely to be the right call for this framework today, but worth knowing about as the theoretical north star against which Tiers 1 and 2 are pragmatic compromises.

### Tier 4 — Concede: stop having an event log file

**(g) Use `git log` as the event log.**

Drop `.ai-repo/events.jsonl` entirely. Each `aiwf` write verb produces one git commit on the user's behalf. The "event log" is `git log -- work/ .ai-repo/` with a custom formatter (`aiwf history`). The "projection" is rebuilt from the working tree.

**Pros:** zero merge problems by construction (git already merges its own log); zero state-of-the-world divergence; total alignment with git's model; no custom merge driver needed.
**Cons:** every write becomes a commit, which conflicts with how humans batch their work into commits ("I want to think before I commit"); the assistant becomes a heavy git user, which requires git commit/push permissions in places that may not have them; loses sub-commit atomicity (a complex `aiwf hotfix` flow that touches multiple entities now spans multiple commits or one chunky commit); audit trail is git history, which is mutable via rebase/amend (so trace-first becomes "trace-first within a commit, mutable across history").
**Verdict:** the most direct answer to "stop fighting git" — the only tier that doesn't keep a separate event log at all. Likely too disruptive to current UX for the framework's target users, but it forces clarity about what the event log is *for*.

### Tier 5 — Hybrid: declare markdown canonical, treat events as advisory

**(h) Stance A taken to its conclusion.**

Markdown is the source of truth for *both* structural fields and prose. `events.jsonl` is an advisory append-only audit log, never authoritative. On merge, the log is regenerated by rescanning the markdown of the merged tree (synthesizing minimal events that explain "this is the state that resulted from a merge of these two histories"). Provenance is preserved on a best-effort basis from git history (commit author, commit time) rather than from explicit event records.

**Pros:** ruthless simplification; the framework stops fighting git because it stops trying to be the time machine; verify becomes "does the projection match the markdown?" — a single comparison; merge becomes a non-event for the framework (the markdown merged or it didn't; if it merged, rescan).
**Cons:** loses fine-grained provenance (who clicked promote, with what idempotency key, at what wall-clock time); loses the ability to record "attempts" separately from "confirmations" (no trace-first-then-confirm); some auditability stories that the architecture was leaning on (forensic replay of partial failures) become weaker.
**Verdict:** a clean answer if the team is willing to scope down the event log's role. Worth comparing seriously against Tier 1 before committing engineering effort either way.

### Cross-tier elements

Several pieces are useful regardless of tier:

- **Post-checkout / post-merge hooks** (from (d)) are useful in every tier.
- **Branch-aware IDs** (from (b)) are useful in every tier except (g), where commits provide the namespace.
- **Schema versioning of events** is useful in every tier where events exist (a, e, h).
- **A `propagation preview` for merges** — show the user "here's what reconciliation will do before I do it" — is useful in every tier and is consistent with the existing architecture's preference for previews over auto-do.

---

## 8. Annotated bibliography

Listed roughly in order of relevance to this framework's specific problem.

### Direct hits

- **Kleppmann, M., & Beresford, A. R.** (2017). *A Conflict-Free Replicated JSON Datatype.* IEEE TPDS, 28(10), 2733–2746.
  The Automerge paper. JSON CRDT with convergence guarantees. **Read first** — it is the closest existing solution to "what if `events.jsonl` were merge-aware out of the box?"
- **Kleppmann, M., Wiggins, A., van Hardenberg, P., & McGranaghan, M.** (2019). *Local-First Software: You Own Your Data, in Spite of the Cloud.* Onward! 2019.
  Articulates the problem class. Argues collaborative-offline-first is the right model for software-that-edits-shared-state-without-a-central-server. The framework is local-first software whether or not it has acknowledged it.
- **Mimram, S., & Di Giusto, C.** (2013). *A Categorical Theory of Patches.* ENTCS 298:283–307.
  Patches as morphisms; merges as pushouts. The rigorous answer to "what does merging two logs mean." The Pijul VCS (Pierre-Étienne Meunier and Florent Becker, ongoing) is the production realization; the theory is the deeper contribution.

### Foundational

- **Shapiro, M., Preguiça, N., Baquero, C., & Zawirski, M.** (2011). *Conflict-Free Replicated Data Types.* SSS 2011.
  The CRDT paper. Background reading for Automerge.
- **Lamport, L.** (1978). *Time, Clocks, and the Ordering of Events in a Distributed System.* CACM, 21(7), 558–565.
  Why a total order on `seq` cannot be globally consistent without coordination. Replace `seq` with vector clocks or causal hashes.
- **Ellis, C. A., & Gibbs, S. J.** (1989). *Concurrent Operational Transformation in Distributed Editors.* SIGMOD 1989.
  Operational transformation. The Google Docs model. Less directly applicable here but useful framing.
- **Gilbert, S., & Lynch, N.** (2002). *Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services.* ACM SIGACT News, 33(2), 51–59.
  The CAP formalization.

### Systems with the right shape

- **Terry, D. B., et al.** (1995). *Managing Update Conflicts in Bayou, a Weakly Connected Replicated Storage System.* SOSP 1995.
  Application-defined merge procedures. The pattern the framework will end up using regardless of tier — every merge driver in §7 is a Bayou merge proc.
- **Dolt** (Liquidata, ongoing). *Dolt is Git for Data.*
  A versioned SQL database with branch/merge primitives at the table level. Their public design notes on schema merge, primary-key collision, and three-way merge for relational data are directly applicable to the projection.
- **Irmin** (Tarides, ongoing). *Irmin: a distributed database that follows the same design principles as git.*
  Mergeable types parameterized over a merge function. The closest answer to "what would `events.jsonl` look like if designed merge-aware from day one."
- **Datomic** (Cognitect). *as-of, with, history.*
  Point-in-time queries over an immutable database. Not branchy in the git sense, but the model of "time is a first-class axis of the database" is a useful counterpoint.
- **Fossil SCM** (Hipp). *VCS + bug tracker + wiki, all in one repo, all merged together.*
  Interesting precedent for "structured records live in the VCS and merge with it." Sees use in the SQLite project.

### Useful adjacent work

- DeCandia, G., et al. (2007). *Dynamo: Amazon's Highly Available Key-Value Store.* SOSP 2007.
- Schneider, F. B. (1990). *Implementing Fault-Tolerant Services Using the State Machine Approach.* ACM Computing Surveys, 22(4).
- Hickey, R. (various talks). *The Database as a Value.* (Datomic.)
- Kleppmann, M. (2017). *Designing Data-Intensive Applications.* O'Reilly. Chapters 5 and 9 in particular cover replication and consistency.

---

## 9. Recommended next steps (for proposal authors)

This document does not pick a tier. The follow-on work is one architecture proposal per admissible tier (1, 2, 4, 5 — tier 3 is research, document but do not propose). Each proposal should:

1. **Take a stance on §3.2.** State explicitly whether markdown or `events.jsonl` is canonical. Justify.
2. **Walk all four scenarios from §2** under the proposed design. State concretely what happens to each artifact.
3. **Address §6 constraints** one by one. Flag any that the proposal violates and propose a corresponding architecture amendment.
4. **Address §5 UX risks.** Especially: how does the proposal avoid post-merge `verify` noise that trains users to ignore findings? How does it handle GitHub's server-side merge button?
5. **Include a migration plan.** What does an existing `.ai-repo/` look like before and after? Is the migration automatic, prompted, or manual?
6. **Cite at least one prior system from §8** as the design's lineage.
7. **Specify the CHANGELOG entry** the change would carry — a forcing function for getting the user-observable effect right.

Until at least one such proposal exists and is accepted, no further work on `events.jsonl`'s seq numbering, hash chain, or ID allocation should land — the design space is too unsettled to commit code to one spot in it.

---

## Appendix A — Glossary specific to this document

| Term | Meaning |
|---|---|
| Branch-local linearity | A property that holds along any single linear lineage in the git DAG, joined explicitly at merge events. |
| Causal order | Partial order in which event A precedes event B iff A is in B's transitive history. |
| Content-derived ID | An identifier computed deterministically from the event's content (and optionally its parent), making the same event have the same id everywhere. |
| Merge driver | A program git invokes for a specific path/pattern during 3-way merge, replacing the default text merger. Configured via `.gitattributes`. |
| Propagation preview | The architecture's existing pattern of showing a user what reconciliation will do before doing it. Should extend to merges. |
| Reconciliation finding | A `verify` finding that records "this state results from a merge and was reconciled by the engine," distinct from drift findings. |
| Stance A / Stance B | The two unresolved positions in §3.2 on whether markdown or `events.jsonl` is canonical. |

---

## Appendix B — Open questions deferred to per-tier proposals

- Where does the ID allocator's counter live? (Currently unspecified in `architecture.md`.)
- Is `graph.json` *always* gitignored, or is committing it a supported configuration? (§2.2 says "gitignored by default"; Appendix A says "gitignored or committed.")
- What is the relationship between `aiwf verify` findings and CI gate severity? (§8.1 hints at this; not fully specified.)
- Does the framework attempt to detect partial merges (e.g., conflict markers left in place by a careless user), or trust that git's exit codes were respected?
- How are schema-version-skew events handled when main is behind a feature branch? (Per-tier.)
- Is there a `aiwf reconcile` verb, distinct from `aiwf verify`? Does it write events, or only emit findings? (Per-tier.)

---

## In this series

- Previous: [introduction](https://proliminal.net/theses/ai-workflow-research/)
- Next: [01 — Git-native planning](https://proliminal.net/theses/git-native-planning/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
