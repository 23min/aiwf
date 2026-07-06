---
title: Entity id lifecycle — minting, confirmation, and reconciliation under concurrency and network failure
status: captured
date: 2026-07-04
---

# Entity id lifecycle — minting, confirmation, and reconciliation under concurrency and network failure

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind, so this file lives under `docs/initiatives/` as an umbrella capture,
following the precedent of
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md).

This is not an ADR: it does not ratify a decision — if anything, it surfaces
that **three existing, unratified designs already answer overlapping parts of
this question** (a count that grew by one, EMB, within this same document's
lifetime — see §4 below), and the first job of any follow-on work is
reconciling them, not inventing a fifth.

This is not research: the need is not speculative — it emerged from a live
design conversation (2026-07-04) about G-0281's gaps-inbox that kept growing
until it was clearly bigger than one gap.

This is not an exploration: this repo already has three artifacts on this
exact axis (`docs/pocv3/design/id-allocation.md`, ADR-0001, G-0281 /
ADR-0022 / M-0186 / M-0187) — the concern is not new, it is scattered.

This is not a plan: it intentionally avoids committing to epics, milestones,
sequencing, or timeframes. Its job is to hold the shape of the problem still
long enough that a plan can be drafted from a coherent center.

## Initiative statement

aiwf has, across separate sessions and separate artifacts, designed **four
different answers** to "how does an entity get a stable id when more than one
branch or machine might be allocating at once":

1. **Incremental allocator widening** (`docs/pocv3/design/id-allocation.md`,
   shipped via E-0052) — read a wider cross-branch view before allocating, so
   most collisions never happen; `aiwf reallocate` remains the backstop for
   the residual race.
2. **Deferred minting at trunk integration** (ADR-0001, `status: proposed`) —
   don't allocate a real numeric id at all until the entity's branch actually
   merges to trunk; use a slug as the pre-mint handle; provide a `--to-trunk`
   escape hatch for gaps/ADRs/decisions that need a real id immediately.
3. **Eager allocation via a coordinated side channel** (G-0281 /
   ADR-0022 §Consequences / M-0186 / M-0187, `status: open`/`proposed`/`draft`)
   — allocate a real numeric id immediately by fetching, allocating, and
   pushing to a dedicated never-checked-out ref, retrying on collision, then
   reconciling the result back into the working branch.
4. **Ephemeral mutation branch, "EMB"** (this document, `status: captured`,
   not yet filed as its own gap or ADR — surfaced pressure-testing (3)
   against real push-cost data) — wrap every Tier-1 (main-direct)
   mutation in a short-lived branch, created and checked out **in the
   operator's own worktree** at current HEAD, committed via the existing
   ADR-0022 primitive, then landed onto trunk via an ordinary
   fetch → attempt → reject-and-retry cycle — the same CAS-guarded
   confirmation `--to-trunk` and G-0281 already use, just targeting trunk
   directly from a branch that never leaves the operator's own worktree. See
   §4 below.

These are not four independent features. They sit on **one axis** —
E-0052's own epic spec says as much for (1) vs (2) ("cheap-now /
structural-later on one axis, not competitors") — but (3) was designed this
session without weighing it against (2), and (2) already has an answer to
several of the hardest questions (3) ran into: confirmation semantics,
avoidance of a reconciliation step, and uniform treatment across all six
kinds rather than gaps-only. (4) was designed later still, specifically
*because* pressure-testing (3) surfaced a cost neither (2) nor (3) had
reckoned with — see "Trunk contention" and the new initiative this spawned,
`check-performance-incremental-revwalk-cache.md` / G-0372.

The initiative is: **before any of G-0281 / M-0186 / M-0187 gets an ADR of
its own, reconcile it explicitly against ADR-0001.** Separately, the
underlying protocol — allocate, confirm, reference, diverge, reconcile,
retry — is precise and small enough to be a genuine candidate for formal
modeling (state machine + safety/liveness properties), not just prose
design. This document also scopes that formalization effort and evaluates
one candidate tool (`loom`, github.com/23min/loom) for it.

## Mission fit

aiwf's mission is mechanically-validated planning state with stable
identity across rename, cancel, and collision (CLAUDE.md commitment #2;
`docs/pocv3/design/design-decisions.md` §"Stable ids and rename ergonomics").
Id allocation under concurrency is not adjacent to that mission — it *is*
that mission, at its least forgiving point: the id is the primary key every
other guarantee (history, provenance, cross-reference integrity, the
`ids-unique` check) is built on top of. Getting the id-lifecycle protocol
wrong doesn't just create friction, it silently corrupts the one property
(id stability) the kernel promises above all others.

## Prior art — four existing answers, not fully reconciled

### 1. Incremental allocator widening — shipped, `E-0052` (`status: done`)

`docs/pocv3/design/id-allocation.md` describes the current mechanism: `aiwf
add` allocates `max(ids) + 1` over a view that (per the E-0052 update) unions
the working tree, every local `refs/heads/*`, every remote-tracking
`refs/remotes/*`, and the configured trunk ref; `--fetch` opt-in-refreshes
it. `ids-unique` (pre-push) catches what still slips through by comparing
working tree against trunk; `aiwf reallocate` fixes it, preserving lineage
via `prior_ids` in frontmatter and `aiwf-prior-entity:` in trailers.

Load-bearing, explicit rejection in that same document, under "What this is
not":

> - A monotonic counter coordinated across branches.
> - **A coordination ref or push-CAS allocator.**
> - ...
>
> Each one was considered, and each one is more code than the problem
> requires. If real friction shows up later, any of them can earn its own
> design.

**This is precisely the mechanism G-0281 proposes** ("a coordination ref or
push-CAS allocator"), scoped to gaps only. E-0052's own epic spec
anticipates this: it explicitly carves out G-0272 (sibling-worktree scan)
and G-0273 (fetch-before-allocate) as "class 1 and 2" — the cheap,
locally-knowable collision classes — and names class 3 ("different
machines, genuinely concurrent — unknowable locally") as the one
`aiwf reallocate` exists for, with **ADR-0001 named as the structural
endpoint on the same axis** if class-3 friction ever justifies it.

**Housekeeping:** `G-0272` and `G-0273` — both cited by `M-0212`/`M-0213` as
`Source: G-0272` / `Source: G-0273` — are `status: addressed` and archived,
as is `G-0316` (cited in `M-0214` as the gap E-0052's remote-tracking-refs
milestone closes). All three are closed; nothing outstanding in this
cluster. `G-0274` (batch reallocate) remains the one legitimately open gap
here — E-0052 explicitly held it out of scope.

### 2. Deferred minting at trunk integration — proposed, `ADR-0001` (`status: proposed`)

`ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md`
is a complete, general design that predates G-0281 and answers a wider
version of the same question, uniformly across **all six
kinds** (epics, milestones, ADRs, gaps, decisions, contracts):

- **Default path.** On a non-trunk branch, `aiwf add <kind>` doesn't
  allocate a real numeric id at all. It writes `work/<kind>/inbox/<slug>.md`
  with `id: <slug>` — the slug *is* the id, pre-mint. Cross-references use
  kind-prefixed slug form (`gap:auth-redirect-loop`). Nothing numeric exists
  yet, so nothing numeric can collide yet.
- **Mint operation.** Triggered automatically by `post-merge`/`post-commit`
  hooks (plus a CI safety net for hosted PR-merges) the moment inbox files
  land on the configured trunk branch. One commit mints every pending entity
  against trunk's real high-water mark, renames inbox files to canonical
  paths, rewrites frontmatter, and rewrites every cross-reference from slug
  form to canonical id — all mechanically, all at once, in a single
  `aiwf assign-pending` commit.
- **Escape hatch — `--to-trunk`.** For "I want a stable numeric id right
  now, no branch ceremony": `aiwf add <kind> --to-trunk` opens a *throwaway
  detached worktree* at `origin/<trunk>`, fetches, allocates against trunk's
  real state, commits with full canonical trailers, and pushes straight to
  `HEAD:<trunk>` — retrying up to 3 times on non-fast-forward rejection.
  **v1 scope is explicitly gaps, ADRs, and decisions** — the exact set
  G-0281's "collision fear" problem statement is about.
- **History.** `aiwf history` walks pre-mint (slug-keyed) and post-mint
  (id-keyed) commits as one timeline via the `aiwf-mint:` trailer recorded
  on the assignment commit.

**This already solves several of the hardest problems G-0281's design ran
into, structurally, not procedurally:**

| Problem G-0281's design wrestled with | ADR-0001's answer |
|---|---|
| "How do we know for sure we got the id?" (confirmation semantics) | The `--to-trunk` push either lands on the *real* trunk branch or is rejected — no side channel, so no separate "is it confirmed" question distinct from "did the push succeed." |
| Reconciliation — nothing lands in `work/gaps/` until a separate import step | Doesn't exist as a problem for the default path: the entity is already a real file on the operator's own branch (as a slug-keyed inbox file) from the moment it's created. For `--to-trunk`, the file lands directly on trunk with no side ref and no import step at all. |
| Cross-reference rewrite when a provisional id turns out to collide | Only needed for `--to-trunk`'s residual race (rare, single retry loop). The *default* path never has this problem: slug-uniqueness-within-kind collisions are individually rare and fixed by an ordinary `aiwf rename`, and nothing numeric ever needs renumbering because nothing numeric was ever provisional. |
| "Only reference it after confirmed" as an operator discipline | Enforced structurally for the default path: you cannot reference a numeric id that doesn't exist yet, because there isn't one — you reference the slug, which is stable from the moment of creation. |

**The uncomfortable implication, stated plainly:** a meaningful fraction of
G-0281's design work (retry-on-reject semantics, two-layer
push-confirmation, offline-divergence renumbering, cross-reference blast
radius) is solving a problem that **would not exist** if ADR-0001's default
mechanism were adopted for gaps — because ADR-0001 never eagerly mints a
real numeric id pre-integration in the first place. G-0281's side-channel
model is not wrong, but it may be solving a narrower, already-subsumed
problem: "I want a real numeric id sooner than merge-time, without
`--to-trunk`'s network-and-worktree ceremony." Whether that narrow niche is
worth a second, gaps-only mechanism *alongside* ADR-0001, or whether
`--to-trunk` (already scoped to gaps in v1) already fills it, is the central
open question this initiative surfaces. See "Open design questions" below —
deliberately not answered here.

**A placement difference worth naming, so the comparison isn't lopsided:**
G-0281's model, unlike `--to-trunk`, lands the filed entity on **whatever
branch the operator is currently on** (via `aiwf gaps import`, parent =
current HEAD) without requiring trunk involvement at all. `--to-trunk`
always lands on trunk directly — a feature-branch operator who runs it gets
a gap that is not yet on their own branch until they pull trunk. These are
genuinely different placement semantics, not just different mechanisms for
the same outcome; a reconciliation decision needs to weigh which one the
"collision fear" workflow actually wants.

### 3. Eager allocation via a coordinated side channel — `G-0281` (`status: open`) / `ADR-0022` (`status: accepted`) / `M-0186` / `M-0187` (`status: draft`)

Summarized for completeness; full detail lives in the gap/ADR/milestone
files themselves:

- **`ADR-0022`** — verb commits move from `git stash push --staged`
  isolation to a temp-index + `commit-tree` plumbing primitive. Orthogonal
  to the minting-strategy question: whichever of (2) or (3) wins, this
  primitive is what constructs the commit either way (including ADR-0001's
  own `--to-trunk` and mint-hook commits, which are exactly this shape).
  The pre-commit shape-check's fate for hand-made commits, and
  commit-signing parity (`commit.gpgsign`) for `commit-tree`-built commits,
  are both covered in the ADR body.
- **`G-0281`** — file a gap onto a dedicated never-checked-out `refs/aiwf/*`-
  class ref via fetch → allocate → CAS `update-ref` → opt-in push. Its
  current design includes: a **Reconciliation** section (materialization
  needs an explicit `aiwf gaps import` verb — the `git stash pop` analogy,
  not the `git notes` / `ghp-import` / GitHub-API analogy, none of which
  actually match this problem's shape — plus read-only peek surfaces
  (`status`, `show`, `render`) that can show pending inbox entries without
  any change to the mutating-verb surface); a **Risks** section (retry-on-
  reject reuses `aiwf reallocate`'s existing cross-reference-rewrite
  machinery rather than a bespoke rename, because a provisional id may
  already be referenced by other local work before it's confirmed;
  deferred (opt-in) push compounds both collision odds and the blast
  radius of a required renumber); and a **"Why a git ref, not a real
  allocator service"** section identifying the mechanism as a
  compare-and-swap sequence generator built on git's own ref semantics (a
  non-force push already *is* CAS on a pointer — the same primitive
  GitHub's `createCommitOnBranch`/`expectedHeadOid` exposes), explaining
  why that's an acceptable, bounded exception to aiwf's otherwise
  fully-offline verb model, scoped to one entity kind.
- **`M-0186` / `M-0187`** — E-0045's two milestones: the shared
  commit-construction primitive, then the gaps-inbox as its second
  consumer. Neither has started; `M-0187`'s own ADR (not yet written) is
  where G-0281's open questions were always meant to land.

### 4. Ephemeral mutation branch ("EMB") — this document, not yet filed

Surfaced while pressure-testing whether G-0281's side-channel model is
actually the best answer once its discoverability cost (§"Discoverability
contention" below) is taken seriously. The mechanism: every Tier-1
(main-direct, per `ADR-0010`) mutation, instead of
committing straight onto trunk, creates a short-lived branch at current
HEAD, checks it out **in the operator's own worktree** (not a side ref, not
a detached-elsewhere temp worktree), commits via the existing
temp-index/commit-tree primitive (`ADR-0022`), then attempts to land it —
fetch, push (or fast-forward-then-push) straight onto trunk, retry with a
rebase-and-recheck on rejection, bounded. Structurally this is
`--to-trunk`'s retry loop done in place instead of in a throwaway detached
worktree elsewhere — the same `ConfirmAgainstRef(s, trunk)` action from the
protocol below, just with `materialized[s]` populated at commit time instead
of at confirmation time.

**What it solves, that G-0281 and `--to-trunk` don't, together:**
`InstantaneousUniqueness` (same CAS-guarded confirmation as both), *and*
zero discoverability blindness (unlike G-0281 — see the table below), *and*
generalizes to all six kinds with one mechanism instead of gaps-only, *and*
introduces no new git primitive (ordinary branches, no dedicated ref).

**What it does not solve, and should not be sold as solving:**

- **Does not remove contention, only makes each instance mechanical.** Under
  real concurrency you still lose races; EMB just makes losing one cheap to
  recover from (fetch, rebase, recheck id, retry) instead of a manual
  redo — *provided* that recovery cycle itself is cheap, which depends
  entirely on the G-0372 finding below.
- **Does not solve genuine content conflicts.** Two sessions editing the
  *same* entity concurrently still produce an ordinary merge conflict EMB
  can't auto-resolve — same as today, no better, no worse.
- **Does not preserve G-0281's "works in any working-tree state" property.**
  A worktree mid-rebase-of-something-unrelated can't safely branch-and-commit
  in place; EMB needs a fallback for that narrow case (unresolved — see
  "Open design questions").
- **EMB does not implement `ADR-0010`'s "AI chokepoint."** Per
  `internal/verb/allow.go`, the actual authorization gate
  (`scopeAllows(actor, verb, entity)`) is governed by **entity-reference-graph
  reachability** — does the mutated entity (or its outbound references, for a
  creation) reach the scope-entity named in an open `aiwf authorize` scope —
  not by which git ref or branch a commit lands on. EMB's branch-per-mutation
  mechanism has **no bearing whatsoever** on AI-actor authorization, in
  either direction; it solves id-collision and discoverability, a
  completely orthogonal axis. `ADR-0010`'s own "AI chokepoint" section left
  its actual mechanism as a downstream planning question — whatever answers
  it, it's entity-reachability, not branch topology, and EMB should not be
  credited with closing that question.

**Risks, not yet resolved:**

- **Nested/concurrent EMB within one session.** If a second mutation fires
  while a first mutation's ephemeral branch is still checked out mid-retry,
  the Tier-1-vs-ritual-branch detection logic must distinguish "my own
  in-flight ephemeral branch" from "a real ritual branch" — undesigned.
- **The "checked out in place" property is two-sided.** It is EMB's whole
  reason for having zero discoverability blindness, and it is also the
  source of a real, structural cost nothing else in this document's
  mechanisms carries: the ephemeral branch **shares the working tree** with
  whatever else the operator is doing there, and the retry step cannot
  fully avoid touching that shared state. An attempted fix — do the retry's
  recompute via the same commit-tree plumbing every verb already uses,
  never touching the real working tree or index — does not fully work: the
  branch is *currently checked out*, so repointing its ref via raw
  `update-ref` leaves the actual files on disk stale relative to what HEAD
  now claims, and reconciling that requires a checkout/reset refresh — the
  exact working-tree-touching operation the fix was trying to avoid. Two
  concrete consequences follow: (a) if the retry's recompute is a genuine
  `git rebase` and it hits a real content conflict (not just an id
  collision), the operator's actual working directory is left mid-conflict
  — and since EMB can't safely operate in an abnormal working-tree state,
  a *later* EMB-wrapped mutation in the same worktree is blocked by the
  *earlier* one's stuck retry, with no fallback, until a human resolves it
  by hand; (b) even absent any conflict, unrelated uncommitted edits sitting
  in the same worktree at retry time are exposed to whatever the refresh
  touches. G-0281 and `--to-trunk` are structurally immune to both, because
  neither ever shares the operator's primary worktree.
- **Retry cost was, until this same investigation, an open question mark —
  now it's a shared, tracked dependency, not an EMB-specific flaw.**
  Pressure-testing EMB's retry loop against this repo's actual pre-push hook
  found the hook costs ~22 seconds, unconditionally, on every push,
  regardless of contention or of which of these four mechanisms is used —
  see `check-performance-incremental-revwalk-cache.md` and **G-0372**,
  filed directly from this finding. That initiative's prototyped fix (an
  immutable per-commit-sha cache; measured 33x speedup, formally verified
  correct) is a prerequisite for EMB, G-0281's opt-in push, *and* plain
  "push after every commit" discipline alike — it does not favor EMB over
  G-0281 or vice versa. Evaluating any of these four mechanisms against
  *today's* push cost, rather than the cost once G-0372 lands, risks biasing
  the comparison against all of them equally, not just against EMB.

**Trunk-contention profile, stated once here since it changes the "Trunk
contention" table below:** EMB lands directly on trunk (the ephemeral
branch's tip is pushed straight onto `refs/heads/<trunk>` at origin, or
fast-forwards local trunk then pushes) — it is on the **same side** of that
axis as `--to-trunk`, not G-0281's decoupled side. It does *not* inherit
G-0281's "no population-wide resync tax" property just because it also
avoids G-0281's discoverability cost; those are two independent axes, and
EMB trades one for the other differently than either existing mechanism
does, rather than winning on both.

## Trunk contention — the friction the operator actually felt

A fourth comparison axis, surfaced by lived experience rather than by reading
the existing designs: **does confirming an entity advance a ref that other
concurrent sessions must resync against, and how often?** This is a real
cost distinct from the safety/liveness properties above — it doesn't affect
whether the protocol is *correct*, it affects how much unrelated friction it
imposes on everyone else working in the repo at the same time, which is
exactly the kind of thing multiple worktrees make visible.

| Mechanism | Touches `trunkRef` at filing time? | Who pays the resync cost, and how often? |
|---|---|---|
| E-0052 (shipped) | No — allocation only *reads* trunk, and pushes nothing until the ordinary epic-wrap merge. | Nobody, beyond the merges that were already going to happen. |
| ADR-0001 default (slug + mint-at-merge) | No — mint fires inside a merge that was already happening. | Nobody new; same as above. |
| ADR-0001 `--to-trunk` | **Yes** — every successful call pushes straight to `origin/<trunk>`. | Every worktree/checkout that was caught up with trunk, every time *any* session uses it — the cost scales with population-wide usage, not with the caller's own usage. |
| G-0281 side channel | No — pushes only advance the dedicated, never-checked-out ref. | Only sessions that plan to `aiwf gaps import`, on their own schedule; nobody else's view of trunk is ever invalidated by it. |
| EMB | **Yes** — same as `--to-trunk`: every landing attempt pushes straight onto `refs/heads/<trunk>`. | Same as `--to-trunk` — population-wide, scales with usage. EMB's zero-blindness win (below) is a *different* axis from trunk-contention; it does not also buy G-0281's decoupling. |

This means the "uncomfortable implication" drawn earlier — that `--to-trunk`
might make G-0281 redundant — was incomplete. `--to-trunk` and G-0281 solve
the same *confirmation* problem, but at different *trunk-contention* cost.
Whether that cost is worth paying for `--to-trunk`'s simplicity (no separate
import step, no side ref to maintain) versus G-0281's decoupling (no
population-wide resync tax, at the cost of an explicit import step per
session) is now a genuine, weighable tradeoff — not a strict dominance
either way. This is folded into the protocol specification below as a
first-class, checkable quantity, not just a comparison-table entry.

## Discoverability contention — search and LLM blindness for un-materialized entities

A fifth comparison axis, surfaced independently of the trunk-contention one
above: **can a plain filesystem search — `grep`/`rg`, an editor's
search-in-files, or an LLM agent reading the working tree cold — see a
pending entity at all, without invoking any aiwf-specific verb?** This is
not a safety property (no kernel guarantee depends on it) and it is not the
`ResyncBurden` cost metric either — it is a third, independent kind of cost,
orthogonal to both confirmation-instantaneousness and trunk-contention.

It is not a new state in the six-stage lifecycle or the protocol
specification below. It is a named cost attached to the existing gap
between `confirmed` (stages 3–4) and `materialized[s]` (stage 5) —
`materialized[s]` was already defined above as "everything session `s` can
currently see as a real file in its own working tree," which is precisely
the grep/editor-search-visible set. What is new here is naming the
*practical* consequence of that gap's size, not its formal shape: how long
an entity can sit confirmed-but-unmaterialized (or, for the inbox model,
how long it can sit proposed-but-not-yet-a-file — moot there, see below)
differs enormously across mechanisms, and that duration is exactly the
window during which grep, editor search, and an LLM agent's cold read
return a false negative for a pending entity that genuinely exists.

- **E-0052 / ADR-0001 default (inbox file).** No blindness window at all.
  The moment `Propose` fires, the entity is an ordinary file at
  `work/<kind>/inbox/<slug>.md` in the operator's own working tree —
  `materialized[s]` is populated at propose time, not at some later confirm
  step. Grep, ripgrep, VSCode search, and any LLM reading the repo see it
  exactly as they would a fully-minted entity; the slug-keyed
  cross-references (`gap:auth-redirect-loop`) are themselves greppable
  strings.
- **ADR-0001 `--to-trunk`.** A brief, bounded blindness window: during the
  retry loop (steps 2–4 of the escape hatch) the entity exists only inside a
  detached temp worktree that most tooling never looks at. Short-lived
  (bounded retries) and self-resolving, but not zero.
- **G-0281 side channel.** Full blindness by construction, for as long as
  the operator defers import. "Never checked out" is the load-bearing design
  choice (see "Design direction" above) — deliberately, nothing lands in the
  operator's working tree until `MaterializeFromRef` (`aiwf gaps import`)
  runs. Between `Propose` and that action, the entity is a real git object
  (reachable via `git show refs/aiwf/gaps:work/gaps/G-NNNN-slug.md`) but
  invisible to every path-based tool. G-0281's own "Read-only visibility"
  mitigation (peek via `aiwf status`/`show`/`render`) only closes this for a
  human or agent that already knows to run an aiwf-specific command against
  the inbox ref — it does nothing for the default discovery path (grep,
  editor search, or an LLM agent listing/searching the working tree), which
  is exactly the path an agent exploring a repo cold, or an operator
  scanning for "what gaps mention X," takes first. Because import is
  explicitly opt-in and un-scheduled ("the operator decides when to
  reconcile" — see "Reconciliation" above), this window has no upper bound
  in the design as written.
- **EMB.** No blindness window, structurally — for the same reason as the
  inbox-file default, not for G-0281's reason: `materialized[s]` is
  populated at commit time because the branch is checked out in the
  operator's own worktree the entire time, not because content is
  eagerly written before confirmation. This is EMB's central
  differentiator from G-0281: the same CAS-guarded confirmation
  mechanism, targeting trunk directly like `--to-trunk`, without either
  G-0281's unbounded blindness or `--to-trunk`'s brief detached-worktree
  window.

The practical failure mode this produces: a pending gap parked in the inbox
ref is, for any tool or agent that discovers work by reading the filesystem
rather than by knowing to ask aiwf, functionally identical to a gap that was
never filed. An LLM asked "are there any known gaps about X" that greps
`work/gaps/` gets a false negative for anything still parked in the side
channel — a wrong answer to a real question, even though no kernel safety
property is violated and `aiwf check` would report the tree as clean.

| Mechanism | `materialized[s]` populated at `Propose` time? | Grep/editor-search visible? | LLM-agent-cold-read visible? |
|---|---|---|---|
| E-0052 / ADR-0001 default | Yes | Yes | Yes |
| ADR-0001 `--to-trunk` | Briefly no (temp worktree, bounded by retry count) | No, transiently | No, transiently |
| G-0281 side channel | No — deferred to an unscheduled, opt-in import | No, until import | No, until import |
| EMB | Yes — checked out in the operator's own worktree from commit time | Yes | Yes |

## The lifecycle this initiative is actually about

Stripped of which specific mechanism implements it, every one of the three
designs above is answering the same six-stage question, and any formal
model needs to cover all six stages, not just "allocation":

1. **Propose** — an operator (human or agent) decides to create an entity
   and picks (or is given) a candidate identity: a slug, a provisional
   numeric id, or nothing yet.
2. **Commit locally** — the entity's content is written and committed
   somewhere reachable only by this operator's machine.
3. **Attempt confirmation** — the candidate identity is checked against a
   shared authority (trunk, or a coordination ref) via a network operation
   that can succeed, be cleanly rejected, or fail ambiguously (timeout,
   partition).
4. **Reconcile on rejection or ambiguity** — on a clean rejection, retry
   against fresh state. On an ambiguous outcome, re-derive ground truth
   independently (never trust a transient acknowledgment) — durably, by
   comparing the shared authority's actual current state against the exact
   object this operator built.
5. **Materialize / reference** — the entity becomes visible and safely
   referenceable to the rest of the tree (other entities' prose,
   frontmatter, cross-references) — and this must not be possible *before*
   stage 4 has actually succeeded, or a rename at stage 4 corrupts
   references that never should have existed yet.
6. **Diverge and re-reconcile under extended offline operation** — an
   operator who proposes and commits several entities while disconnected,
   then reconnects, faces the multi-entity version of stage 4: not just "my
   one candidate collided," but "some prefix of my locally-built chain
   collided, and anything I built *on top of* the colliding portion, still
   offline, needs to move with it."

Each of the three existing designs answers a different subset of these six
stages with a different mechanism, and none of them has been checked against
the others for interaction effects — e.g., what happens if ADR-0001's
mint-at-merge hook fires while G-0281's inbox-import verb is also active on
the same repo; whether `aiwf reallocate`'s cross-reference rewrite is
reentrant-safe if triggered automatically by a retry loop rather than by a
human running it once.

## Protocol specification (for formal modeling)

This section is written at the level of detail a TLA+ module needs —
explicit state, explicit actions with enabling conditions and effects — so
that translating it is mechanical rather than a second design pass.

**The core protocol below is fully generic over an abstract set of
coordination refs — it contains no git, worktree, branch, or merge
vocabulary at all.** Earlier drafts of this section named `trunkRef` and
`sideRef` as separate pieces of state with separate, near-duplicate confirm
actions (`ConfirmToTrunk` / `ConfirmToSideRef`, `MergeToTrunk`); that baked
one specific mechanism's git mechanics into what should be an
implementation-agnostic coordination protocol. A worktree is fully captured
by the existing `Session` unit; a branch merge is fully captured by "confirm
a whole batch against a ref instead of one candidate." Nothing about either
needs its own state or its own action. The core protocol is proved once,
generically, over an arbitrary set of refs; *which* refs exist, and what
else touches them, is pushed into a separate, purely descriptive
instantiation table at the end of this section — that table is where
"trunk," "worktree," and "merge" belong, and it is data the properties are
checked *against*, not new machinery the properties are stated *in terms
of*.

### Actors

- `Session` — a finite set of concurrent working contexts, one per active
  worktree/checkout (**not** per human; one human running two worktrees is
  two sessions). This is the right unit of concurrency because the friction
  motivating this initiative is a per-worktree phenomenon, not a per-human
  one, and it is the *only* concept a worktree needs in this model.
- `Ref` — a finite set of abstract coordination refs. The protocol doesn't
  care what a `Ref` "is" in git terms (a branch, a dedicated never-checked-out
  ref, anything else) — only that it has a value that advances monotonically
  and that a population of sessions may or may not track it. E-0052 is the
  degenerate case of using *no* `Ref` at all (see the instantiation table).
- `Kind` — a finite set of entity kinds (gap, milestone, ADR, decision,
  epic, contract) — orthogonal to the protocol; id sequences are scoped per
  kind, so a single-kind model is sufficient for a first spec and the
  multi-kind case is a straightforward product. **Scoping note:** this
  collapses "slug" and "provisional numeric id" into one abstract
  `candidateId` notion. The real ADR-0001 design has two genuinely distinct
  collision domains — slug collision (resolved by `aiwf rename`) and numeric
  id collision (resolved by `aiwf reallocate`) — this is a deliberate
  reduction for a first spec, not an oversight, and should be named as such
  if the spec is ever extended to distinguish them.

### Global (shared, remote-observable) state

- `refValue : Ref → Nat` — each ref's current abstract version. Advances
  only via `ConfirmAgainstRef` or `BatchConfirmAgainstRef` (below).
- `confirmed ⊆ Id` — the set of ids that are, as of the current global
  state, real and exclusively claimed. Grows, or has an element atomically
  replaced by `RenumberOnCollision`; never loses an element outright.
- `owner : confirmed → Session` — which session's candidate produced each
  confirmed id. **Updated by `RenumberOnCollision` too:** when `old`
  becomes `new`, `owner[new] := owner[old]` — the owning session doesn't
  change, only the id does.

### Per-session local state

- `localView[s] : Ref → Nat` — session `s`'s last-fetched value of each
  ref. Starts equal to `refValue` at session creation; goes stale
  per-ref, independently, whenever some other session advances that
  specific ref without `s` resyncing it. Keeping this per-ref (rather than
  one bundled "resync everything" value) is what makes `ResyncBurden`
  attributable to a specific ref rather than an undifferentiated total.
- `pending[s]` — a **sequence** (order matters — this is what lets the
  offline-divergence property be stated precisely) of candidate entities
  session `s` has committed locally but not yet confirmed. Each element is
  a record `[candidateId, refs]` where `refs` is the set of ids/candidates
  it cross-references.
- `materialized[s] ⊆ Id ∪ CandidateId` — everything session `s` can
  currently see as a real file in its own working tree: its own
  `pending[s]` entries plus whatever confirmed entities it has actually
  materialized in from some ref. Deliberately **not** the same set as
  `confirmed` — that gap between "globally true" and "locally visible" is
  the whole reason `NoPrematureReference` below is non-trivial.

### Actions

- **`Propose(s, kind)`.** Always enabled, no network. Appends a fresh
  candidate to `pending[s]`, immediately added to `materialized[s]`.
  **Candidate-identity note:** this spec deliberately leaves open whether a
  candidate carries a numeric guess at propose time (the *guess, CAS,
  possibly re-guess-on-reject* shape — what `--to-trunk` actually does: it
  fetches, computes a specific number, writes the entity with that number
  baked in, and only revises it if rejected) or carries no numeric identity
  at all until confirmation (the *mint-fresh-only-on-success* shape — what
  ADR-0001's slug-keyed default path does — there is no "wrong guess" case
  because there was never a guess). Both shapes satisfy the same action
  signature here; which one a given `ConfirmAgainstRef` instance implements
  is instantiation-layer detail, not core-protocol detail — but a spec
  author should pick one per mechanism explicitly rather than let it stay
  ambiguous, since the two shapes have different retry-failure modes.
- **`Reference(s, from, to)`.** Enabled only if `to ∈ materialized[s]`.
  Effect: records a `refs` edge from `from` to `to`. This enabling
  condition *is* `NoPrematureReference`, stated as a precondition rather
  than an invariant to check after the fact.
- **`ResyncRef(s, ref)`.** Always enabled. Effect: `localView[s][ref] :=
  refValue[ref]`. Parameterized by which ref, so `ResyncBurden` can be
  computed per ref rather than conflating a trunk-forced resync with a
  side-ref resync.
- **`ConfirmAgainstRef(s, ref)`.** Enabled if `pending[s]` is non-empty.
  Effect: if `localView[s][ref] = refValue[ref]` (the CAS guard), pop the
  head of `pending[s]`, add it to `confirmed` with `owner = s`, set
  `refValue[ref] := refValue[ref] + 1` and `localView[s][ref] :=
  refValue[ref]`. If the guard fails (another session advanced `ref` first),
  the action instead leaves `localView[s][ref]` stale — modeling a rejected
  push — and `s` must `ResyncRef(s, ref)` before trying again. **This one
  action covers both `--to-trunk` and G-0281** — they differ only in which
  `ref` gets passed, which is exactly the point of the refactor.
- **`BatchConfirmAgainstRef(s, ref)`.** Enabled if `pending[s]` is
  non-empty. Effect: same CAS guard as `ConfirmAgainstRef`, but on success
  the *entire* `pending[s]` sequence moves to `confirmed` atomically, in
  order, preserving each entry's `refs`; `refValue[ref]` advances by
  `Len(pending[s])`. This covers ADR-0001 default's mint-at-merge trigger.
  **This action is *not* exempt from the CAS guard**, even though a real
  git merge might seem to serialize automatically. That's only true if the
  guard is actually present in git-terms (an ordinary non-fast-forward push
  rejection on `ref`) *and* the triggering mint logic recomputes ids
  against the post-rebase state rather than replaying a cached decision
  from before the rejection. Both are real requirements on the
  implementation, not automatic consequences of "it's a merge" — so the
  model keeps the same guard here as the single-candidate action,
  deliberately, rather than assuming git's plumbing makes it unnecessary.
- **`MaterializeFromRef(s, ref)`.** Enabled if `confirmed` contains ids
  (confirmed via `ref`) not yet in `materialized[s]`. Effect: adds them to
  `materialized[s]`. Covers both `aiwf gaps import` (G-0281's side ref) and
  the ordinary "pull trunk into my branch" action that ADR-0001-based
  mechanisms need before another session can reference something someone
  else minted — both must be modeled, or `NoPrematureReference` is
  untestable for the trunk-based mechanisms.
- **`DetectCollision(s1, s2)`.** Enabled when two sessions' locally-visible
  states each believe they hold the same id in `confirmed` — possible only
  under mechanisms that skip `ConfirmAgainstRef`'s CAS guard entirely (see
  `EventualUniqueness` below). Effect: marks the pair for
  `RenumberOnCollision`. Models `aiwf check`'s `ids-unique`/`trunk-collision`
  finding. **Necessary because `EventualUniqueness` (below) is a real,
  non-vacuous property for two of the four mechanisms** — without this
  action, a single `Uniqueness` property would incorrectly claim to hold for
  mechanisms that only ever provide the weaker guarantee.
- **`RenumberOnCollision(s, old, new)`.** Enabled after `DetectCollision`,
  or synchronously after a failed `ConfirmAgainstRef` guard for a candidate
  that, on resync, turns out to collide with something already confirmed.
  Effect: replace `old` with `new` everywhere it appears as a `refs` target
  in every session's `pending`; `owner[new] := owner[old]`. **Tiebreaker
  left deliberately nondeterministic** (either side may be renamed) — a
  first safety pass should hold regardless of which side a tiebreaker picks.
  The real system's actual rule (ancestor-of-trunk wins; "both or neither in
  trunk" stops and prompts a human — `docs/pocv3/design/id-allocation.md`
  §"Reallocate when both branches did real work") is a refinement to layer
  on afterward, and its "stops and prompts a human" outcome is a genuine
  third terminal state this model's liveness properties need to account for
  explicitly (see `Convergence` below) rather than silently assume away.

### Safety properties

- **`InstantaneousUniqueness`.** `∀` reachable state, `∀ id ∈ confirmed`:
  exactly one session ever owns it. Holds **only** for mechanisms whose
  every confirm action is `ConfirmAgainstRef`/`BatchConfirmAgainstRef`
  (i.e., every path to `confirmed` is CAS-gated) — this is what `--to-trunk`
  and G-0281 actually provide.
- **`EventualUniqueness`.** A strictly weaker property for mechanisms that
  allow `confirmed` to transiently hold the same id from two sessions: any
  such duplicate is always eventually resolved via `DetectCollision` →
  `RenumberOnCollision`. This is what E-0052 and ADR-0001's default path
  actually provide — **not** `InstantaneousUniqueness`, matching
  `id-allocation.md`'s own description of collisions surfacing at merge
  time and being fixed by `aiwf reallocate` after the fact. Stating both
  properties, rather than one `Uniqueness` property applied uniformly,
  matters: a single invariant would be simply false for two of the four
  mechanisms, which only ever provide the weaker guarantee.
- **`NoPrematureReference`.** Implied by construction via `Reference`'s
  enabling condition; stated as an invariant for defense in depth: `∀`
  recorded edge `(from, to)`: `to` was in `materialized[s]` (for the owning
  session `s`) at the time the edge was recorded.
- **`ReferentialIntegrityUnderRenumber`.** `∀` state reachable after any
  `RenumberOnCollision(s, old, new)`: no `refs` set anywhere contains `old`,
  and `owner[new] = owner[old]` (pre-renumber).

### Liveness properties

- **`Convergence`.** Under weak fairness on `ResyncRef` and the confirm
  actions for every session `s`: `◇(pending[s] = ⟨⟩)` — every session's
  backlog eventually empties, for any finite `Session`, **provided every
  collision is resolvable by the automatic tiebreaker.** For the "stops and
  prompts a human" terminal case, this property needs an explicit
  `HumanResolvesTiebreak(s1, s2)` action in the fairness assumption, or it
  needs to be weakened to "converges, or reaches an explicit
  human-actionable stuck state" — claiming full automatic convergence
  without this is overclaiming what the real system does.
- **`RetryTermination`.** A bounded-steps refinement of `Convergence`: for
  any finite `Session`, there exists `N` such that every candidate confirms,
  reaches the human-stuck state above, or surfaces an explicit terminal
  failure, within `N` of its own session's confirm attempts.
- **Fairness, stated as a decision rather than an assumption.** "Weak
  fairness on `ResyncRef` and the confirm actions" needs to be chosen
  deliberately against what's actually being modeled, not applied as
  boilerplate TLA+ liveness incantation — in particular, whether a session
  that gives up after a bounded number of retries (a legitimate real
  behavior) should be modeled as *violating* fairness (and therefore outside
  what `Convergence` claims) or as a distinct, intentionally-unfair session
  class the spec allows and still expects `RetryTermination`'s bounded
  version to hold for.

### The resync-burden metric (cost, not correctness)

Not a safety or liveness property — a derived quantity, per ref, that turns
the trunk-contention comparison into something checkable:

```
ResyncBurden(ref) ==
  the number of ResyncRef(s, ref) actions, summed over all s, that were
  forced (i.e., localView[s][ref] # refValue[ref] at the time) as a
  downstream consequence of some OTHER session's ConfirmAgainstRef or
  BatchConfirmAgainstRef action on that same ref.
```

Because `ResyncRef` is now parameterized by `ref` rather than bundled, this
is directly computable per ref from one TLC run over the instantiation
table below — no separate spec needed per mechanism.

### Instantiation table — where "trunk," "worktree," and "merge" actually live

Purely descriptive metadata the properties above are checked *against*; not
new state-machine actions. This is the *only* place git-specific vocabulary
belongs in this section:

| Mechanism | Which `Ref`(s) | Confirm action used | `InstantaneousUniqueness` or `EventualUniqueness`? | Also touched by unrelated activity (drives `ResyncBurden`) |
|---|---|---|---|---|
| E-0052 (shipped) | none — no CAS ref at all | *(none; relies solely on `DetectCollision`/`RenumberOnCollision`)* | Eventual | n/a |
| ADR-0001 default | `trunk` | `BatchConfirmAgainstRef(s, trunk)`, fired by an ordinary merge | Eventual (only truly `Instantaneous` if the merge-time CAS guard and id-recompute-on-rebase are both actually implemented, per the correction above) | Yes — every ordinary branch integration touches `trunk` |
| ADR-0001 `--to-trunk` | `trunk` | `ConfirmAgainstRef(s, trunk)` | Instantaneous | Yes — every checkout tracking `main` |
| G-0281 | `refs/aiwf/gaps` | `ConfirmAgainstRef(s, gapsRef)` | Instantaneous | No — dedicated, never checked out |
| EMB | `trunk` | `ConfirmAgainstRef(s, trunk)`, from a branch checked out in `s`'s own worktree | Instantaneous | Yes — every checkout tracking `main`, same as `--to-trunk` |

**Mixed adoption falls out for free.** Because the confirm action is
generic over `ref`, a realistic near-term deployment — some sessions using
G-0281's ref, others using plain E-0052, concurrently, for the same kind,
since G-0281 is opt-in rather than a universal replacement — doesn't need a
separate model variant the way it would have under the earlier
per-mechanism-named-action draft. It's simply a TLC run where different
sessions' `Next` choices pick different rows of this table. That's a direct
consequence of the genericization, not an additional design task.

## Formal methods fit

The user's instinct that this is "FSM territory" is correct, and it's worth
being precise about *which* formal tool fits *which* part of the six-stage
lifecycle, because the two mainstream choices are good at different halves
of the problem:

- **TLA+ (with the TLC model checker)** is the natural fit for the
  **concurrency and interleaving** questions: multiple operators, each
  running the propose → commit → confirm → reconcile cycle, in every
  possible interleaving, exhaustively checked against the safety properties
  above (uniqueness, no-premature-reference, referential integrity) and the
  liveness properties (convergence, retry termination) via temporal
  operators. This is exactly the class of problem TLA+ was built for —
  Lamport's own canonical examples are lock and consensus protocols, and a
  CAS-based id allocator is structurally a lock/consensus protocol wearing
  a git costume. A TLA+ spec would let this initiative *search for*
  counterexamples (e.g., "can two operators both believe they hold gap
  G-50 under some interleaving of network delays") rather than reasoning
  about them by hand.
- **Dafny** (or F*) is the natural fit for **per-function contract
  verification** once the protocol design is settled: proving that a
  specific `AllocateID`, `Reconcile`, or `Reallocate` implementation
  actually satisfies its pre/postconditions and terminates, given the
  abstract protocol TLA+ has already validated. Dafny reasons well about a
  single function's correctness; it is not the tool for exhaustively
  exploring many concurrent actors' interleavings the way TLC is.

Recommended sequencing, if this initiative becomes a plan: **TLA+/TLC first**
to validate the protocol shape and hunt for the interaction bugs between
ADR-0001 and G-0281's mechanisms (before committing to build either), **then
Dafny** to pin the actual Go implementation's core allocate/confirm/reconcile
functions against the properties TLA+ validated.

The "Protocol specification" section above is written so the TLA+ module has
one `Next` relation with `ConfirmAgainstRef`/`BatchConfirmAgainstRef`
generic over which `Ref` they target, with the instantiation table
supplying the mechanism-specific bindings, which lets a single spec answer
both the correctness question (do `InstantaneousUniqueness` /
`EventualUniqueness`, `NoPrematureReference`,
`ReferentialIntegrityUnderRenumber`, `Convergence`, `RetryTermination` hold
under each row of that table?) and the cost question
(`ResyncBurden`, from the trunk-contention comparison) in one model, run
under TLC once per mechanism.

## Loom (github.com/23min/loom) — assessed fit

Loom is the user's own project, and a fast-moving one — see the risk entry
below on re-checking this section before trusting it.

- **What it is.** A three-layer model (prose → *umbrella* of structured
  formal claims → LLM-authored implementation), where the umbrella is small
  enough for a human to read end-to-end and precise enough for a verifier
  to check the implementation against, claim by claim. The umbrella
  language (`.lm`) has five registers — `knows` (vocabulary), `relates`
  (operation contracts), `shows` (examples), `does` (implementation),
  `proves` (universal properties) — with cross-register coverage rules
  designed to resist an LLM gaming a weak spec.
- **What it verifies against — now dual-backend.** Originally Dafny-only;
  `M-0018` ("force the second substrate: add a model-checker backend
  through the seam," `status: done`) added a **TLC model-checker backend
  through the same seam**, so an umbrella's `proves` claims can now be
  checked by whichever backend fits the claim's shape — Dafny for
  per-function contracts, TLC for concurrency/interleaving properties. This
  is exactly the Dafny/TLA+ split this initiative's "Formal methods fit"
  section argues for, now available through one tool instead of two
  separately-run ones.
- **Current status — active development, not seed stage.** E-0001 through
  E-0004 (validating the differentiator, re-validating the value gate
  twice, dogfooding the umbrella loop on real aiwf code) are all `done`.
  `E-0005` ("build loom-light: the opt-in, contained verification overlay
  and runner," `status: active`) has landed five of its six milestones:
  the overlay/runner and frozen contracts (`M-0016`), **self-hosting** —
  loom's own trust-critical plumbing (atomic crash safety, dispatch
  totality, parser totality) verified *by loom itself* (`M-0017`), the TLC
  backend above (`M-0018`), and an authoring loop, `loom author`, that
  turns prose and examples into a verified umbrella end to end (`M-0019`,
  `M-0020`). Only `M-0021` (`loom suggest` — proposing loomable invariants
  from source) remains `draft`. Code exists, runs, and verifies itself.
- **A reciprocal finding.** Loom's own tree already carries `G-0018`
  ("scope the aiwf id-lifecycle protocol as loom's next worked example"),
  filed against this exact document — it names this initiative doc by path,
  quotes its "Formal methods fit" recommendation back, and observes that
  loom's `examples/` directory has no worked example exercising *both*
  backends together on a single non-trivial, externally-motivated protocol
  yet (`examples/05-composition/` is Dafny-only). It also names two
  supporting gaps already filed: `G-0013` (automatic Dafny-vs-TLC substrate
  selection in `loom author`) and `G-0014` (assembling the TLA+
  model/`.cfg` overlay via the interface) — i.e., the specific mechanics
  this protocol's `Ref`-parameterized, per-mechanism instantiation table
  would need are already scoped on loom's side, not hypothetical.

**Fit assessment:** loom is usable *today*, not a future possibility. The
protocol specification above is already written at the granularity an
umbrella needs (`knows`: `Session`, `Ref`, `Kind`, `Id`; `relates`:
`ConfirmAgainstRef`, `BatchConfirmAgainstRef`, `ResyncRef`,
`MaterializeFromRef`, `DetectCollision`, `RenumberOnCollision`; `proves`:
`InstantaneousUniqueness`/`EventualUniqueness`, `NoPrematureReference`,
`ReferentialIntegrityUnderRenumber`, `Convergence`, `RetryTermination`) —
authoring it as a loom umbrella, rather than a hand-written standalone
`.tla` file plus a separately hand-written Dafny model, gets the two
backends through one seam and one gap report instead of two disconnected
artifacts.

**Recommendation:** author this protocol as a loom umbrella directly.
Loom's own `G-0018` already proposes this from the other side — the two
repos are now pointing at each other, and the sequencing question (TLA+/TLC
first, Dafny second, per "Formal methods fit" above) maps onto loom's
existing per-property backend selection rather than onto two separate
manual tool invocations.

## Existing aiwf surfaces this touches

### ADRs and docs

- `docs/pocv3/design/id-allocation.md` — the shipped incremental-widening
  design; explicitly rejects "a coordination ref or push-CAS allocator" as
  more code than the general problem needs. This initiative's central
  tension is that G-0281 proposes exactly that, narrowly.
- `docs/adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md`
  (`status: proposed`) — the general, all-six-kinds deferred-mint design;
  the primary thing this initiative needs reconciled against G-0281 before
  either moves further.
- `docs/adr/ADR-0022-verb-commits-built-via-a-temp-index-commit-tree-primitive.md`
  (`status: accepted`) — orthogonal commit-construction primitive, useful
  regardless of how the minting-strategy question resolves.
- `docs/pocv3/design/design-decisions.md` §"Stable ids and rename
  ergonomics" — the kernel commitment this whole initiative serves.
- `docs/pocv3/design/provenance-model.md` — principal × agent × scope
  provenance that any new commit-construction path (mint hooks, `--to-trunk`,
  the gaps-inbox, EMB) must continue to stamp correctly.
- `docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md`
  (`status: accepted`) — the Tier-1/Tier-2 branch model EMB operates
  *within* (it only wraps Tier-1, main-direct mutations). Its "AI chokepoint"
  section left the actual AI-authorization mechanism as a downstream
  planning question; `internal/verb/allow.go` (entity-reference-graph
  reachability, not branch topology) is what actually answers it — EMB
  does not. See §4 above.
- `docs/initiatives/check-performance-incremental-revwalk-cache.md` — a
  sibling initiative, spawned directly from pressure-testing EMB's retry
  cost against this repo's real pre-push hook. Its two independently-safe
  findings (the dead `-m` fan-out, the redundant mistag walk) have since
  shipped, cutting real cost from ~25.5s to ~19.1s (~25% faster); the
  structural fix underneath — making the cost scale with what changed since
  the last check, not total history — remains unsolved (G-0372, still
  open) and is the shared prerequisite for EMB, G-0281's opt-in push, and
  plain frequent-push discipline alike — not an EMB-specific concern,
  despite surfacing from an EMB pressure-test.

### Open gaps

- `G-0372` — aiwf check's full-history git revwalks run unconditionally on
  every push; the concrete performance blocker this initiative's EMB
  discussion surfaced, filed against `check-performance-incremental-revwalk-cache.md`.
- `G-0281` — opt-in gaps inbox on a never-checked-out ref; the design
  thread this initiative was extracted from.
- `G-0274` — batch reallocate; legitimately still open, explicitly deferred
  by E-0052; relevant to the "offline-divergence bound" property above
  (reconciling N colliding entities in one pass is exactly a batch-reallocate
  shape).
- `G-0272` / `G-0273` / `G-0316` — **resolved, not part of this initiative's
  open surface.** All three are `status: addressed` and archived. See the
  Housekeeping note under §1 above.

### Milestones and epics

- `E-0052` (`done`, archived) — the shipped, cheaper point on the same axis;
  this initiative's comparison table leans on its epic spec's own framing.
- `E-0045` / `M-0186` / `M-0187` (`draft`) — the commit-construction epic
  G-0281 sits under; `M-0187`'s not-yet-written ADR is where this
  initiative's central open question ultimately needs to be decided.

## Risks and boundaries

### Risk: building G-0281's mechanism before resolving the ADR-0001 question

The concrete, immediate risk this initiative exists to prevent: `M-0187`
starting implementation, writing its own ADR, and shipping a gaps-only
coordination-ref mechanism that ADR-0001 — if accepted — would make
partially or wholly redundant. Avoid by treating "reconcile against
ADR-0001" as a precondition for `M-0187`'s ADR, not a parallel concern.

### Risk: formal modeling scope creep

A TLA+ spec for "id allocation" could balloon into modeling all of git's
ref-update semantics, network partition models, or Byzantine actors — none
of which this problem needs. Avoid by scoping the model to the six stages
named above and the properties listed, not the full generality of
distributed consensus theory.

### Risk: this document's loom assessment goes stale again

It already did once: loom moved from "seed stage, no code" to "active
development, dual-backend, self-hosting" within days, because it's a
fast-moving sibling project the user also actively develops. Avoid trusting
this section as a permanent snapshot — re-check loom's actual `README.md`
and epic/milestone status against this section before acting on it if any
real time has passed, rather than assuming the assessment above still
holds.

### Risk: the comparison table oversimplifies a genuine placement difference

G-0281's "lands on my current branch" versus `--to-trunk`'s "lands on
trunk directly" are not strictly ranked — one isn't simply better. Avoid
collapsing this initiative's central question into "ADR-0001 wins" without
weighing which placement semantic the actual gaps-filing workflow needs.

### Risk: EMB's "checked out in place" property can produce the very abnormal working-tree state it can't operate in

EMB's zero-discoverability-blindness win depends on the ephemeral branch
staying checked out in the operator's own worktree throughout a retry, not
just at initial commit. But the retry's recompute step, done safely, cannot
fully avoid touching that same working tree (see §4's "checked out in place
is two-sided" above) — so a retry that hits a genuine content conflict
leaves the worktree mid-conflict, and any *later* EMB-wrapped mutation in
that same worktree is then blocked by the earlier one's stuck state, with no
designed fallback. Avoid treating this as a minor implementation detail —
it's a structural tension between EMB's headline property and its own
recovery mechanism that needs a real answer (abort-and-cleanly-report? a
narrower detached-worktree fallback for the conflict case only, à la
`--to-trunk`?) before EMB is built, not discovered after.

### Risk: evaluating any of these four mechanisms against today's push cost, rather than G-0372's prospective structural fix

`aiwf check` costs ~19.1s unconditionally on every push today, after the two
safe G-0372 fixes shipped (down from ~25.5s). Every comparison in this
document that touches retry cost or "push often" as a mitigation was
reasoned about against a number in that range. Once G-0372's structural
fix — the incremental per-commit-sha cache, still unsolved after two
attempted designs were set aside (see `check-performance-incremental-revwalk-cache.md`) —
lands, the real cost is likely closer to the incremental-walk numbers
measured there (tens of ms to a couple seconds for a realistic delta) — a
large enough change that conclusions drawn under today's cost (e.g., "EMB's
retry loop is too expensive to trust") may not hold once it ships. Avoid
treating a conclusion reached under today's cost as durable; re-check it
once G-0372's structural fix lands, not just once per mechanism.

## Open design questions

These are intentionally not answered here — they are the reason this stays
an initiative document rather than an ADR:

- Does accepting ADR-0001 (for gaps, at minimum — its v1 `--to-trunk` scope
  already includes gaps) remove the need for G-0281's coordination-ref
  mechanism entirely, or does the "land on my current branch without
  trunk ceremony" placement semantic remain a distinct, worth-keeping need?
- Given the trunk-contention comparison, is `--to-trunk`'s population-wide
  resync cost acceptable for how often gap-filing actually happens in
  practice, or does that cost alone justify keeping G-0281's side-channel
  design specifically *because* it decouples from trunk — independent of
  which model wins on placement semantics? A `ResyncBurden` run under TLC
  would turn this from a judgment call into a measured comparison.
- If both survive, do they compose (e.g., G-0281's inbox becomes an
  alternate front-end that still mints via ADR-0001's trunk authority
  rather than its own side ref), or do they remain genuinely separate
  mechanisms for genuinely separate use cases?
- Should ADR-0001 itself be revisited/ratified before `M-0186`/`M-0187`
  proceed, given it already answers a superset of what `M-0187` set out to
  design?
- Is the six-stage lifecycle and the seven properties listed above the
  right scope for a first TLA+ spec, or does it need splitting (e.g.,
  model per-kind minting separately from cross-reference rewrite)?
- Who owns writing the TLA+ spec, and what's the bar for treating it as
  "done" — a hand-checked set of invariants, or an actual TLC run over a
  bounded model with N operators?
- Should the Dafny modeling target the eventual Go implementation directly
  (as a correctness oracle checked in CI) or remain a standalone proof
  artifact that informs the Go implementation without being mechanically
  linked to it?
- If G-0281 proceeds despite the discoverability-contention cost above,
  what closes the grep/LLM-cold-read blindness window — a bounded
  auto-import cadence, a materialized read-only mirror of the inbox ref
  that ordinary tooling can see without going through `aiwf`, an operator
  habit of frequent `aiwf gaps import`, or is the window accepted as a
  deliberate, documented tradeoff for the trunk-decoupling benefit? Is this
  cost alone enough to prefer ADR-0001's inbox-file model for gaps
  specifically, independent of the trunk-contention and placement-semantic
  questions above?
- **EMB-specific, all unresolved:** how does the Tier-1-vs-ritual-branch
  detection logic recognize "this is my own in-flight ephemeral branch,
  don't wrap it again" for nested/concurrent invocations within one session?
  What does EMB fall back to when the current worktree is genuinely in an
  abnormal state (mid-rebase, mid-conflict) at mutation time — a narrower
  detached-worktree escape hatch scoped to just that case, à la
  `--to-trunk`? Should EMB's retry-recompute step ever touch the checked-out
  working tree at all, or is the tension between "stays checked out
  throughout" and "retry can't safely touch the working tree" fundamentally
  unresolvable, meaning EMB has to give up one of those two properties
  rather than have both?
- Now that G-0372's structural fix exists as a separately-tracked,
  prototyped design (the two safe quick fixes already shipped): should
  evaluating EMB / G-0281 / `--to-trunk` against each other wait until it
  ships, or proceed now on the understanding that all four mechanisms in
  this document inherit today's ~19.1s-per-push cost until then?

## Recommendation — defer all three unratified mechanisms, pending measured evidence

The EMB pressure-testing above prompts the actual question underneath all
four mechanisms: is any of this solving a *measured* problem, or an
anticipated one? Consistent with the classifier note's "not a plan, not an
ADR" stance — this recommends *building nothing new yet*, which is a
deferral, not a commitment to any epic, milestone, or timeline.

**The measurement.** This repo's own git history is the only real data
source available:

```
git log --all --grep="^aiwf-verb: reallocate$" --format="%ad" --date=short   # 34 events
git log --all --grep="^aiwf-verb: add$"        --format="%H"                 # 986 events
```

**34 reallocate events against 986 add events — a ~3.4% collision rate** —
over this repo's ~2.3 months of history (2026-04-26 to 2026-07-05). More
telling than the rate: those 34 events land on only 11 distinct calendar
days, clustering into roughly five or six identifiable multi-day episodes
(e.g., six on 2026-06-28, four the day before, three the day before that —
the exact window G-0281 itself was designed in and cites as its own
provenance) rather than spreading evenly across ~70 days of activity. This
is bursty, tied to specific known episodes of heavy concurrent-worktree
work, not a steady background tax. Every one of the 34 was resolved by a
single `aiwf reallocate` invocation.

**The verdict.** This document's own words, from the "Relationship to the
collision cluster" note already carried in G-0281 itself, apply better here
than anything new: *"Collisions are friction, not correctness... this whole
effort optimizes file-time friction, not a correctness hole."* A ~3.4% rate,
fixed in one command each time, concentrated in identifiable episodes, is
friction `E-0052` + `aiwf reallocate` is already absorbing adequately. None
of ADR-0001, G-0281, or EMB should be built on the strength of anticipated
"collision fear" alone — each is real, non-trivial engineering (a new
minting model touching all six kinds; a dedicated coordination ref plus an
import verb; a branch-per-mutation wrapper with two open design gaps) being
proposed against a problem the data says is small and already handled.

**What to actually do now, in order, none of it contingent on resolving
which of the three wins:**

1. ~~Land the two provably-safe fixes from `G-0372` /
   `check-performance-incremental-revwalk-cache.md` — dropping the dead
   `-m` fan-out and de-duplicating the mistag walk.~~ Shipped: cut real
   cost from ~25.5s to ~19.1s (~25% faster), no dependency on any
   id-minting decision. `G-0372` stays open for the structural fix
   underneath, which this did not attempt.
2. Reinstate tight push-after-every-Tier-1-mutation discipline. This
   targets the *original* complaint this whole initiative grew from ("main
   moved" warnings) with zero new engineering, using `E-0052` — already
   shipped — at its already-measured ~96.6% first-try success rate.
3. Explicitly defer `ADR-0001`, `G-0281`, and EMB. Not reject — defer, with
   a named trigger below. `M-0187` (`G-0281`'s implementation milestone)
   has since been cancelled on this reasoning, pending that trigger or a
   ratifying decision.

**The trigger for revisiting**, stated so it's checkable rather than a
vibe: the reallocate rate climbing meaningfully above the measured ~3-4%,
or the bursts stopping being tied to identifiable concurrent-work episodes
and becoming constant background noise across ordinary activity. That
mirrors `E-0052`'s own rejected-alternatives text near-verbatim: *"If real
friction shows up later, any of them can earn its own design."* Re-run the
same two `git log --grep` queries above periodically (e.g., at each epic
wrap) rather than relying on memory or anecdote to judge whether the
trigger has fired.

**If the trigger does fire:** this document's comparison work above is not
wasted by this deferral — it's exactly what a future decision would need,
already reconciled. The lean, per the EMB pressure-testing, would be EMB
first, contingent on `G-0372` having landed by then (evaluating any of the
three against today's push cost rather than the post-`G-0372` cost risks
biasing the decision — see the risk entry above).

## Desired future property

A future human or AI agent deciding how to file a gap, ADR, or milestone
under concurrent, possibly-offline conditions should be able to point at
one settled design — not reconstruct, from three scattered documents and a
long conversation, which of three overlapping mechanisms currently applies.
That design should carry a machine-checked argument (via TLA+/TLC, and
later Dafny) for why it cannot silently duplicate an id, cannot let a
reference outlive the id it points at, always converges after any finite
amount of offline divergence, and never leaves a genuinely-filed entity
invisible to ordinary search or agent tooling for longer than the chosen
mechanism's own confirmation window requires — not just a prose argument
that merely reads convincingly.
