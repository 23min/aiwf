# Legal workflows — first-principles catalog (Pass B)

> **Status.** This is M-0122's deliverable for E-0033. It is Pass B of the
> three-pass methodology pinned by [ADR-0011](../../adr/ADR-0011-legal-workflow-spec-methodology.md).
> The rules below are derived **from first principles** — the entity model in
> [`design-decisions.md`](design-decisions.md), the archive convention in
> [ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md),
> the id convention in [ADR-0008](../../adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md),
> the branch model in [ADR-0010](../../adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md),
> and the provenance model in [`provenance-model.md`](provenance-model.md) —
> **independently of Pass A** (M-0121's `legal-workflows-audit.md`).
>
> The independence is load-bearing. If this catalog matches Pass A's
> extraction, we have high confidence the rule is real. If it diverges,
> the divergence is an explicit decision-point for Pass C (M-0123) to
> resolve.
>
> The rules here are derived from **what the entity model commits to**,
> not from how the implementation happens to enforce it. Where multiple
> equally-defensible derivations are possible from the model alone, the
> rule is marked *conventional* (a sensible default but could be
> otherwise) rather than *load-bearing* (must hold or the model breaks);
> ambiguities surface in the closing "Open questions for Pass C"
> section.
>
> Scope (per ADR-0011 §Scope): kernel-verb workflows at three layers —
> per-entity FSM transitions, per-verb pre/post conditions beyond FSM
> (cross-entity invariants), and cross-verb sequence legality. Branch
> choreography (ADR-0010 layer 4), rituals-plugin orchestration, and
> random/model-based fuzz testing are explicitly **out of scope**.

## Row schema

Each rule is a row with six columns:

| Column | Meaning |
|---|---|
| Rule id | `R-FP-NNNN` — sequential 4-digit, distinct id-space from Pass A's `R-AUDIT-NNNN`. |
| Scope | Which entity kind, verb, or cross-cutting concern the rule constrains. |
| Statement | The rule itself, stated declaratively. |
| Reasoning | The first-principles derivation: which property of the entity model makes this rule load-bearing or conventional. |
| Load-bearing? | `load-bearing` — the model is incoherent without this rule; or `conventional` — a sensible default that the model could have made differently. |
| Severity if violated | Severity if a `aiwf check` finding fires (or "n/a" if the rule is a verb-time refusal only). |

---

## 1. Per-kind lifecycles

This section enumerates each kind's status set and reasons about which transitions make semantic sense given the names of the states. The derivation is "what would a human reader of `aiwf show E-0001` reasonably expect about which transitions are legal, given the closed status set and English-language convention?"

The general lifecycle shape across all kinds:

- A **proposal/draft/open** state representing "exists but not yet committed-to."
- One or more **active/accepted** states representing "live, in use."
- One or more **terminal** states representing "decided to stop — either by completion, abandonment, or supersession."
- Terminal states stay terminal: there is no demotion. Per `internal/entity/transition.go`'s self-comment quoted in ADR-0004 ("there is no 'demote'"), unwinding a terminal state is a hand-edit / new-entity affair, not a verb.

### 1a. Epic (`E-NNNN`, statuses: `proposed`, `active`, `done`, `cancelled`)

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0001 | epic FSM | `proposed → active` is legal. | "Proposed" means "queued for ratification"; "active" means "work has started." The transition between them is exactly the ratification act. Without this transition the kind is unusable. | load-bearing | error |
| R-FP-0002 | epic FSM | `proposed → cancelled` is legal. | A proposal can be rejected without ever being ratified. This is the symmetric closure of R-FP-0001 — every proposal either enters the active set or doesn't. | load-bearing | error |
| R-FP-0003 | epic FSM | `active → done` is legal. | "Done" is the normal completion of "active" work. Without this transition, completion is impossible. | load-bearing | error |
| R-FP-0004 | epic FSM | `active → cancelled` is legal. | Active work can be abandoned (priorities shift, the underlying assumption falsifies). The four-state closed set includes a non-completion terminal precisely to admit this case. | load-bearing | error |
| R-FP-0005 | epic FSM | `done` is terminal — no outgoing transitions. | Completion is sovereign. Once done, the entity is a historical record. Any subsequent state would dilute the meaning of "done." | load-bearing | error |
| R-FP-0006 | epic FSM | `cancelled` is terminal — no outgoing transitions. | Cancellation is sovereign. Re-opening cancelled work happens via a new entity that references the cancelled one (per ADR-0004 §Reversal), not by demoting `cancelled → active`. | load-bearing | error |
| R-FP-0007 | epic FSM | `proposed → done` is **not** legal. | An epic can't finish without ever being ratified. The "active" state isn't just decoration — it's the visible signal that work is in flight. Skipping it conflates planning state with completion state. | load-bearing | error |
| R-FP-0008 | epic FSM | `cancelled → proposed` (or any backward transition) is not legal. | Terminals stay terminal (R-FP-0005, R-FP-0006). The FSM is one-directional. | load-bearing | error |

### 1b. Milestone (`M-NNNN`, statuses: `draft`, `in_progress`, `done`, `cancelled`)

The milestone lifecycle is structurally identical to the epic's, with `draft ↔ proposed` and `in_progress ↔ active`. The choice of "draft" over "proposed" is conventional (milestones live inside their parent epic's scope and don't need separate ratification ceremony; "draft" reads more naturally for a child entity).

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0009 | milestone FSM | `draft → in_progress` is legal. | Same reasoning as R-FP-0001 — the "draft → live" transition is the kind's primary mode of activation. | load-bearing | error |
| R-FP-0010 | milestone FSM | `draft → cancelled` is legal. | Same as R-FP-0002 — a draft can be abandoned without ever being worked. | load-bearing | error |
| R-FP-0011 | milestone FSM | `in_progress → done` is legal. | Same as R-FP-0003 — completion of live work. | load-bearing | error |
| R-FP-0012 | milestone FSM | `in_progress → cancelled` is legal. | Same as R-FP-0004 — abandonment of live work. | load-bearing | error |
| R-FP-0013 | milestone FSM | `done` is terminal. | Same as R-FP-0005. | load-bearing | error |
| R-FP-0014 | milestone FSM | `cancelled` is terminal. | Same as R-FP-0006. | load-bearing | error |
| R-FP-0015 | milestone FSM | `draft → done` is not legal. | Same as R-FP-0007 — can't complete what was never started. | load-bearing | error |

### 1c. ADR (`ADR-NNNN`, statuses: `proposed`, `accepted`, `superseded`, `rejected`)

ADRs depart from epic/milestone shape because the lifecycle's meaning is different: ADRs are *decisions*, not *work*. The "live" state is "accepted" (the decision stands), and the two non-acceptance terminals are `superseded` (replaced by another ADR) and `rejected` (decided against).

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0016 | ADR FSM | `proposed → accepted` is legal. | The ratification act. Without it the kind is unusable. | load-bearing | error |
| R-FP-0017 | ADR FSM | `proposed → rejected` is legal. | A proposal can be decided against without ever being accepted. | load-bearing | error |
| R-FP-0018 | ADR FSM | `accepted → superseded` is legal. | Decisions evolve. A new ADR may supersede an older one; the older ADR transitions from "live" to "historical-but-named" via this transition. | load-bearing | error |
| R-FP-0019 | ADR FSM | `superseded` is terminal. | Once superseded, the ADR is a historical reference. The fresh ADR (the superseder) carries forward; reviving an old decision happens via a new ADR (per `supersedes` chain), not via demotion. | load-bearing | error |
| R-FP-0020 | ADR FSM | `rejected` is terminal. | Same shape as `cancelled` for epic/milestone: rejected decisions don't get re-proposed under the same id; a new ADR proposes the alternative. | load-bearing | error |
| R-FP-0021 | ADR FSM | `accepted → rejected` is not legal. | Once an ADR is accepted, the route to "not in force" is via supersession (which preserves the chain) or via a new ADR rejecting the implied premise. Direct demote to `rejected` would erase the chain and silently invalidate downstream references. | conventional | error |
| R-FP-0022 | ADR FSM | `proposed → superseded` is not legal. | Supersession is a relation between accepted decisions; superseding something that was never in force is meaningless. | load-bearing | error |

### 1d. Gap (`G-NNNN`, statuses: `open`, `addressed`, `wontfix`)

Gaps have the simplest lifecycle — three states, one entry (`open`), two terminals (`addressed`, `wontfix`).

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0023 | gap FSM | `open → addressed` is legal. | The gap was identified, work happened, the gap closed. The primary success transition. | load-bearing | error |
| R-FP-0024 | gap FSM | `open → wontfix` is legal. | The gap was identified and intentionally declined (out of scope, low ROI, blocked by external constraint). The intentional-non-action terminal. | load-bearing | error |
| R-FP-0025 | gap FSM | `addressed` is terminal. | Once closed by work, the gap is historical. A regression files a new gap referencing the old one (per ADR-0004 §Reversal). | load-bearing | error |
| R-FP-0026 | gap FSM | `wontfix` is terminal. | Same as `addressed` — the decision-to-not-act is sovereign, and a renewed willingness to act files a new gap. | load-bearing | error |
| R-FP-0027 | gap FSM | `addressed → wontfix` (or vice versa) is not legal. | Cross-terminal transitions never make sense — both terminals close the gap by different routes; demoting one to the other erases the closure-record. | load-bearing | error |

### 1e. Decision (`D-NNNN`, statuses: `proposed`, `accepted`, `superseded`, `rejected`)

Decisions share their status set with ADRs but exist at a different governance layer (project-scoped, vs. ADR's architecture-scoped). The lifecycle rules are structurally identical — the same arguments that establish R-FP-0016 through R-FP-0022 apply here.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0028 | decision FSM | `proposed → accepted` is legal. | Mirror of R-FP-0016 for decisions. | load-bearing | error |
| R-FP-0029 | decision FSM | `proposed → rejected` is legal. | Mirror of R-FP-0017. | load-bearing | error |
| R-FP-0030 | decision FSM | `accepted → superseded` is legal. | Mirror of R-FP-0018. | load-bearing | error |
| R-FP-0031 | decision FSM | `superseded` is terminal. | Mirror of R-FP-0019. | load-bearing | error |
| R-FP-0032 | decision FSM | `rejected` is terminal. | Mirror of R-FP-0020. | load-bearing | error |
| R-FP-0033 | decision FSM | `accepted → rejected` is not legal. | Mirror of R-FP-0021. | conventional | error |
| R-FP-0034 | decision FSM | `proposed → superseded` is not legal. | Mirror of R-FP-0022. | load-bearing | error |

### 1f. Contract (`C-NNNN`, statuses: `proposed`, `accepted`, `deprecated`, `retired`, `rejected`)

Contracts have a richer lifecycle than ADRs/decisions because they govern an ongoing operational relationship (validators, schemas, fixtures) rather than a one-shot judgment. The extra state `deprecated` is "in force but discouraged" — a graceful sunset path before full retirement.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0035 | contract FSM | `proposed → accepted` is legal. | The ratification act, mirroring R-FP-0016. | load-bearing | error |
| R-FP-0036 | contract FSM | `proposed → rejected` is legal. | Mirror of R-FP-0017. | load-bearing | error |
| R-FP-0037 | contract FSM | `accepted → deprecated` is legal. | The sunset path: a contract is still in force but consumers are encouraged to migrate. Without this transition, the "deprecated" state would be unreachable. | load-bearing | error |
| R-FP-0038 | contract FSM | `accepted → retired` is legal. | A contract can be retired directly without a deprecation period (e.g., an experimental binding that didn't pan out). Less common path but the model's three-terminal structure admits it. | conventional | error |
| R-FP-0039 | contract FSM | `deprecated → retired` is legal. | The completion of the sunset path. After deprecation, full retirement removes the contract from active enforcement. | load-bearing | error |
| R-FP-0040 | contract FSM | `retired` is terminal. | Mirror of R-FP-0019/0031 — the historical-but-named state. | load-bearing | error |
| R-FP-0041 | contract FSM | `rejected` is terminal. | Mirror of R-FP-0020/0032. | load-bearing | error |
| R-FP-0042 | contract FSM | `deprecated → accepted` (revive) is not legal. | Demote is forbidden universally (R-FP-0008 generalized); a deprecated contract returning to full acceptance would need a fresh `C-NNNN` to record the new commitment. | load-bearing | error |
| R-FP-0043 | contract FSM | `proposed → deprecated` is not legal. | Deprecation only makes sense for a contract that was first accepted; deprecating something that was never in force is meaningless (parallel to R-FP-0022). | load-bearing | error |
| R-FP-0044 | contract FSM | `proposed → retired` is not legal. | Same shape as R-FP-0043 — retirement implies prior force. | load-bearing | error |
| R-FP-0045 | contract FSM | `accepted → rejected` is not legal. | Mirror of R-FP-0021 — rejection is a pre-acceptance terminal. | conventional | error |

**Total: 45 rules in §1.**

---

## 2. Acceptance criteria and TDD phase

ACs are namespaced sub-elements of milestones with composite ids `M-NNNN/AC-N`. Their lifecycle is constrained by the parent milestone's lifecycle. The status set is `open | met | deferred | cancelled` per design-decisions.md §"Acceptance criteria and TDD"; the TDD phase set is `red | green | refactor | done`.

### 2a. AC FSM

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0046 | AC FSM | `open → met` is legal. | The primary success transition: the AC's claim has mechanical evidence. | load-bearing | error |
| R-FP-0047 | AC FSM | `open → deferred` is legal. | An AC can be postponed when scope shifts but the work hasn't been formally cancelled. | load-bearing | error |
| R-FP-0048 | AC FSM | `open → cancelled` is legal. | An AC can be removed from scope outright. | load-bearing | error |
| R-FP-0049 | AC FSM | `met → deferred` is legal. | Per design-decisions.md §"AC FSM," "met may move to deferred/cancelled if scope changes after the fact." This admits the practical case where an AC was satisfied, then later re-evaluated as no-longer-applicable. | load-bearing | error |
| R-FP-0050 | AC FSM | `met → cancelled` is legal. | Same reasoning as R-FP-0049 — the model deliberately allows post-met scope changes. | load-bearing | error |
| R-FP-0051 | AC FSM | `deferred` is terminal w.r.t. the milestone's "done" gate but flexible: `deferred → open` may be legal if work resumes. | The model is less explicit on whether deferred can return to open; design-decisions.md only lists `deferred` and `cancelled` as terminals for the AC's lifecycle. **Marked conventional because the precise terminality of `deferred` is a Pass C decision point.** | conventional | n/a |
| R-FP-0052 | AC FSM | `cancelled` is terminal. | Cancellation is sovereign across all kinds; ACs inherit this property. | load-bearing | error |
| R-FP-0053 | AC FSM | `open → open` (no-op) is not a verb-allowed transition (or, if allowed, is a no-op). | Spurious self-promotion is never useful and would clutter history; the FSM should refuse it. **Conventional** — the model is silent; a permissive FSM might allow it. | conventional | error |

### 2b. TDD phase FSM

The TDD phase is linear: `red → green → refactor → done`. Required only when the parent milestone has `tdd: required`. Per design-decisions.md, when the milestone is `tdd: none`, the phase field is tolerated as absent or any value.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0054 | TDD phase FSM | `red → green` is legal. | The TDD cycle's primary forward step: failing test becomes passing. | load-bearing | error |
| R-FP-0055 | TDD phase FSM | `green → refactor` is legal. | The cycle's improvement step: passing test, code cleanup. | load-bearing | error |
| R-FP-0056 | TDD phase FSM | `refactor → done` is legal. | The cycle's completion. | load-bearing | error |
| R-FP-0057 | TDD phase FSM | `done` is terminal. | Once the phase reaches done, the cycle is over; further phase mutations would mean re-opening the AC's work. | load-bearing | error |
| R-FP-0058 | TDD phase FSM | Skipping phases (e.g., `red → done`) is not legal. | The phases name the discipline of the cycle. Skipping them defeats the policy's purpose. Forcing skip requires `--force --reason`, surfaced as a sovereign override. | load-bearing | error |
| R-FP-0059 | TDD phase FSM | When parent milestone has `tdd: none`, the phase field is tolerated as absent or any value — no FSM enforcement. | Per design-decisions.md: the kernel guards the *outcome* (met requires done) only when `tdd: required`. Under `tdd: none`, the field is informational. | load-bearing | n/a |

### 2c. AC × milestone composition

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0060 | AC × milestone | When parent milestone has `tdd: required`, an AC with `status: met` requires `tdd_phase: done`. | This is the kernel's single audit rule across the two FSMs — codified in design-decisions.md as load-bearing. The kernel guards the outcome; the rituals plugin drives the flow. | load-bearing | error |
| R-FP-0061 | AC × milestone | A milestone with `status: done` requires every AC to be in a terminal AC status (`met`, `deferred`, `cancelled`) — no `open` ACs at done time. | Codified in design-decisions.md as the `milestone-done-incomplete-acs` finding. The reasoning: "done" claims completion, and an open AC contradicts the claim. | load-bearing | error |
| R-FP-0062 | AC × milestone | AC ids are position-stable: `acs[i].id == "AC-{i+1}"`. Cancelled ACs stay in `acs[]` at their original position. | Codified in design-decisions.md. Necessary for `M-NNNN/AC-N` references in trailers and gap.addressed_by fields to remain valid across deletions. | load-bearing | error |
| R-FP-0063 | AC × milestone | AC ids are allocated per-milestone starting at 1; no global allocator. | Codified in design-decisions.md. The composition-not-reference relationship between AC and milestone means ACs share no namespace with each other across milestones. | load-bearing | error |
| R-FP-0064 | AC × milestone | An AC's lifecycle is bounded by its parent milestone's lifecycle: cancelling the milestone implicitly cancels all of its ACs (or terminates them otherwise — the model doesn't pin the exact mechanism, but ACs cannot outlive the milestone). | Composition means the AC has no existence apart from the milestone. The milestone's terminality is the AC's terminality. **Marked conventional** because the exact cascade mechanism — auto-cancel ACs vs. inherit-terminality — is a Pass C decision point. | conventional | error |
| R-FP-0065 | AC × milestone | The milestone's frontmatter `acs[]` array and the body's `### AC-N — <title>` headings should agree (one heading per `acs[]` entry, matched by id). | Codified in design-decisions.md as the `acs-body-coherence` warning. Necessary for `aiwf show` and the render's manifest tab to surface AC bodies. | load-bearing | warning |
| R-FP-0066 | AC × milestone | AC promotion to `met` is allowed only when mechanical evidence exists for the AC's claim. | Codified in CLAUDE.md §"AC promotion requires mechanical evidence" as a discipline rule — not (yet) a kernel finding. **Marked conventional** since the kernel does not (per the current design) enforce a test-existence check; this is reviewer/skill discipline. Pass C should decide whether to elevate to a kernel rule. | conventional | n/a |

### 2d. AC verbs

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0067 | `aiwf add ac` | Requires an existing milestone id as the parent argument. The verb refuses if the parent milestone does not exist or has a terminal status. | Composition: ACs cannot exist apart from a milestone. Adding an AC to a `done` milestone would create the contradiction R-FP-0061 polices. | load-bearing | error |
| R-FP-0068 | `aiwf add ac` | When the parent milestone has `tdd: required`, the verb seeds `tdd_phase: red` (the legal starting state). When `tdd: none` or `advisory`, the field is omitted. | Codified in design-decisions.md. The kernel writes the only legal starting state and otherwise doesn't impose policy. | load-bearing | error |
| R-FP-0069 | `aiwf add ac` | Allocated id is `max(acs[].id_number) + 1`, including cancelled entries. | Per R-FP-0062 (position-stable ids). | load-bearing | error |
| R-FP-0070 | `aiwf promote <composite-id>` | Refuses an illegal AC-FSM transition unless `--force --reason` is supplied. | The general FSM-refusal rule applies to composite ids. | load-bearing | error |
| R-FP-0071 | `aiwf rename <composite-id>` | Updates `acs[].title` in the parent milestone and rewrites the matching `### AC-N — <title>` body heading in one commit. The bare-id form keeps path-rename behavior. | Codified in design-decisions.md. The dispatch on composite-vs-bare id is necessary for the AC sub-element to share the verb namespace with milestones. | load-bearing | n/a |

**Total: 26 rules in §2.**

---

## 3. Cross-entity invariants

These rules pin relationships across kinds that the entity model commits to.

### 3a. Milestone ↔ epic composition

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0072 | milestone | Every milestone has a `parent` frontmatter field pointing at an existing epic. | Codified in design-decisions.md's per-kind reference fields table. The milestone is composed by its epic; without the parent reference, the composition is invisible. | load-bearing | error |
| R-FP-0073 | milestone × epic | An epic may not transition to `done` while one or more of its milestones is in a non-terminal status (`draft` or `in_progress`). | Composition implies completion-coherence: a "done" epic with `in_progress` milestones contradicts the claim that the parent's work is finished. | load-bearing | error |
| R-FP-0074 | milestone × epic | `aiwf cancel` on an epic implicitly cancels (or otherwise terminalizes) its non-terminal milestones. **Marked conventional** because the exact cascade mechanism is a Pass C decision point. | Cancellation propagating to children is the most natural semantics, but the model could instead require the operator to cancel each milestone explicitly. | conventional | error |
| R-FP-0075 | milestone × epic | A milestone may not be added to an epic whose status is terminal (`done` or `cancelled`). | A "done" epic has its child set frozen; adding a milestone changes the scope retroactively, which contradicts the "done" claim. | load-bearing | error |
| R-FP-0076 | milestone × epic | `depends_on` between milestones is a DAG — cycles are forbidden. | Codified in design-decisions.md's no-cycles invariant. A cycle would make dependency-ordering impossible. | load-bearing | error |
| R-FP-0077 | milestone × epic | A milestone may declare `depends_on` only on other milestones (not on epics, gaps, ADRs, decisions, or contracts). | Codified in design-decisions.md's per-kind reference fields table. The dependency relation is between work units of the same kind. | load-bearing | error |

### 3b. Reference resolution and id stability

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0078 | refs | Every id-form reference in frontmatter must resolve to an extant entity (in either active or archive location). | Codified in design-decisions.md as `refs-resolve`. Stale refs break the framework's "referential stability is real" commitment. | load-bearing | error |
| R-FP-0079 | refs | The id space is unique across active and archive locations per kind. Two entities cannot share an id. | Codified in design-decisions.md as `ids-unique`. The id is the primary key; collisions break primary-key semantics. | load-bearing | error |
| R-FP-0080 | refs | References resolve across both active and archive directories. | Codified in ADR-0004 §"Id resolver." Archive movement is location-only; references stay live by id. | load-bearing | n/a |
| R-FP-0081 | refs | An archived entity's references *into* still-active entities are not health-linted (out of scope for active-tree linting). | Codified in ADR-0004 §"`aiwf check` shape rules." The forget-by-default principle for archive. | conventional | n/a |
| R-FP-0082 | refs | `prior_ids: []` on an entity preserves history across reallocations: `aiwf history <old-id>` matches `aiwf-prior-entity: <old-id>` trailers. | Codified in design-decisions.md §"Stable ids and rename ergonomics." Without the prior_ids list and the prior-entity trailer, reallocation would silently break id-history queries. | load-bearing | error |

### 3c. ADR supersession chain

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0083 | ADR | `supersedes` on an ADR is a list of older ADR ids; `superseded_by` is a single newer ADR id pointing back. | Codified in design-decisions.md's per-kind reference fields. The chain is bidirectional by convention so that "what does this ADR replace?" and "what replaced this ADR?" both have answers. | load-bearing | error |
| R-FP-0084 | ADR | When ADR-X supersedes ADR-Y, ADR-Y's status must transition to `superseded` (or already be terminal — `rejected` is acceptable since rejection precedes supersession on the FSM). | The supersession relation is meaningless unless the older ADR's status reflects it. The transition is the load-bearing pairing of frontmatter (`superseded_by`) with status. | load-bearing | error |
| R-FP-0085 | ADR | Supersession chains form a DAG — no cycles (`A supersedes B` then `B supersedes A` is forbidden). | A cycle would make "which decision is currently in force?" unanswerable. | load-bearing | error |

### 3d. Gap addressing

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0086 | gap | `addressed_by` may reference any kind: epic, milestone, ADR, decision, contract, or AC (composite id). | Codified in design-decisions.md's per-kind reference fields ("any kind") and §"Acceptance criteria and TDD" (composite ids accepted on open-target fields). | load-bearing | error |
| R-FP-0087 | gap | When a gap transitions to `addressed`, at least one `addressed_by` reference should be set (the entity that addresses the gap). **Marked conventional** because the model does not strictly require the reference to exist before promotion. | A gap can be addressed by hand-edited prose plus a status flip; the structured reference is the recommended form but not load-bearing. | conventional | warning |
| R-FP-0088 | gap | `discovered_in` may reference a milestone or epic — the work that surfaced the gap. | Codified in design-decisions.md's per-kind reference fields. | load-bearing | error |

### 3e. Decision relations

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0089 | decision | `relates_to` may reference any kind. | Codified in design-decisions.md. Decisions are governance-layer; they cut across kinds. | load-bearing | error |

### 3f. Contract relations

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0090 | contract | `linked_adrs` references ADRs that motivate the contract. | Codified in design-decisions.md's per-kind reference fields. | load-bearing | error |
| R-FP-0091 | contract × binding | A contract entity (`C-NNNN`) and a contract binding in `aiwf.yaml.contracts.entries[]` are distinct: the entity is registry state; the binding is operational state pointing at schema/fixtures/validator. | Codified in design-decisions.md §"Contracts." Conflating them would mix registry concerns with operational concerns. | load-bearing | error |
| R-FP-0092 | contract verify | The verify pass requires every `valid/` fixture to pass and every `invalid/` fixture to fail at the contract's current version. | Codified in design-decisions.md §"Contracts." The pairing is what makes the fixtures-tree meaningful. | load-bearing | error |
| R-FP-0093 | contract evolve | The evolve pass runs every historical `valid/` fixture against the *current* schema, catching silent breakage. | Codified in design-decisions.md §"Contracts." Without it, schema evolution is unaudited. | load-bearing | error |

**Total: 22 rules in §3.**

---

## 4. Frontmatter schema invariants

These rules pin the YAML frontmatter shape every entity must satisfy.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0094 | frontmatter | Every entity has the three required fields: `id`, `title`, `status`. | Codified in design-decisions.md's "Common to every kind" table. Without these, the entity has no primary key, no display name, and no lifecycle position. | load-bearing | error |
| R-FP-0095 | frontmatter | `id` matches the kind's id format. | Codified in design-decisions.md and ADR-0008. The id format is `<prefix>-NNNN` canonical, with narrower legacy widths tolerated on input. | load-bearing | error |
| R-FP-0096 | frontmatter | `status` is in the kind's closed status set. | Codified in design-decisions.md §"Six entity kinds." Out-of-set values would make the FSM checks return nondeterministic results. | load-bearing | error |
| R-FP-0097 | frontmatter | `title` is non-empty and within the configured `entities.title_max_length` cap (default 80 chars). | Codified in CLAUDE.md §"Type design." The cap is hard-reject at write time, with grandfathering for pre-cap titles. | load-bearing | error |
| R-FP-0098 | frontmatter | `created` and `updated` timestamps are **absent** from frontmatter — `git log` carries them. | Codified in design-decisions.md ("deliberately absent... Putting them in YAML would be redundant state and a future drift target"). | load-bearing | error |
| R-FP-0099 | frontmatter | Per-kind reference fields are typed: `parent: id`, `depends_on: []id`, `supersedes: []id`, `superseded_by: id`, `discovered_in: id`, `addressed_by: []id`, `relates_to: []id`, `linked_adrs: []id`. | Codified in design-decisions.md's per-kind reference fields table. Type mismatches (string vs. list) would make refs-resolve return unstable. | load-bearing | error |
| R-FP-0100 | frontmatter | Bodies are not validated by the kernel. They are human prose with template stubs. | Codified in design-decisions.md ("The framework guarantees structural and referential stability of frontmatter; prose is the human's responsibility"). | load-bearing | n/a |
| R-FP-0101 | frontmatter | The milestone's `acs[]` items have valid `id` (`AC-N`, position-equal including cancelled entries), `status` in the closed set, and `tdd_phase` in the closed set when present. | Codified in design-decisions.md as the `acs-shape` finding. The composite-id grammar is fragile without this. | load-bearing | error |
| R-FP-0102 | frontmatter | The milestone's `tdd` field is in `{required, advisory, none}` (default `none` when absent). | Codified in design-decisions.md §"Acceptance criteria and TDD." | load-bearing | error |

**Total: 9 rules in §4.**

---

## 5. ID format and stability

These rules pin the id namespace and its semantics.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0103 | id format | The canonical width is 4 digits across all kernel kinds (`E-NNNN`, `M-NNNN`, `G-NNNN`, `D-NNNN`, `C-NNNN`, `ADR-NNNN`). Composite ACs are `M-NNNN/AC-N`. | Codified in ADR-0008. Uniform width simplifies the parser, the renderer, and downstream tooling. | load-bearing | error |
| R-FP-0104 | id format | Parsers accept narrower legacy widths on input (`E-22`, `M-007`) and canonicalize to 4-digit on output. | Codified in ADR-0008 §"Parser tolerance." Pre-migration consumers continue to validate without history rewrite. | load-bearing | n/a |
| R-FP-0105 | id format | Allocators always emit 4-digit form for new entities. | Codified in ADR-0008 §"Allocator behavior." | load-bearing | error |
| R-FP-0106 | id format | An id, once allocated, is permanent. Even after rename, cancel, or supersession, the id keeps its mapping to that entity-history. | Codified in design-decisions.md §"Cross-cutting properties" and §"Stable ids and rename ergonomics." | load-bearing | error |
| R-FP-0107 | id allocation | `aiwf add <kind>` allocates `max(existing_ids_for_kind) + 1` over the union of the working tree and the configured trunk ref. | Codified in design-decisions.md §"Stable ids and rename ergonomics." Scanning trunk closes the dominant collision case. | load-bearing | error |
| R-FP-0108 | id allocation | The slug-as-collision-buffer property: two branches that allocate the same id for different titles produce different paths because the slug is part of the path. Git merges both cleanly; collision surfaces only when `aiwf check`'s `ids-unique` runs. | Codified in design-decisions.md §"Stable ids and rename ergonomics." This is what makes the simple allocator viable without coordination. | load-bearing | error |
| R-FP-0109 | id collision | `aiwf reallocate` picks the next free id, `git mv`s the file, walks every entity's frontmatter to rewrite reference fields, and writes `aiwf-prior-entity: <old-id>` on the resulting commit. Body-prose refs surface as findings for human review (not auto-rewritten). | Codified in design-decisions.md §"Markdown is the source of truth" and §"Stable ids and rename ergonomics." | load-bearing | error |
| R-FP-0110 | id collision | The id format is never extended with suffixes (no `M-007a`/`M-007b`); collision recovery always renumbers. | Codified in design-decisions.md §"Stable ids and rename ergonomics." Suffixes would expand the id grammar past the regular `<prefix>-NNNN` shape. | load-bearing | error |

**Total: 8 rules in §5.**

---

## 6. Provenance model rules

These rules pin the principal × agent × scope provenance commitments. Codified in `provenance-model.md`.

### 6a. Identity sourcing

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0111 | identity | Operator identity is derived at runtime from `git config user.email` (or `--actor <role>/<id>` override), not stored in `aiwf.yaml`. | Codified in provenance-model.md. Per-checkout identity is required for the multi-clone case. | load-bearing | error |
| R-FP-0112 | identity | The `<role>/<id>` regex is `^[^\s/]+/[^\s/]+$` — exactly one `/`, no whitespace, both sides non-empty. | Codified in provenance-model.md. The parsed-shape rule is a precondition for the human-vs-non-human distinction. | load-bearing | error |
| R-FP-0113 | identity | Roles starting with `human/` are humans; everything else is a non-human actor. The kernel makes one structural distinction. | Codified in provenance-model.md. The structural distinction is what gates `--force` and the principal trailer. | load-bearing | error |

### 6b. Trailer rules

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0114 | trailers | Every mutating verb writes `aiwf-verb:` + `aiwf-entity:` + `aiwf-actor:` trailers on its commit. | Codified in design-decisions.md §"Markdown is the source of truth" and CLAUDE.md §"Commit conventions." Without these the trailer-driven `aiwf history` is incomplete. | load-bearing | error |
| R-FP-0115 | trailers | When `aiwf-actor:` is a non-human role, `aiwf-principal:` is required (and is a `human/...` role). | Codified in provenance-model.md. Non-human acts must be attributable. | load-bearing | error |
| R-FP-0116 | trailers | When `aiwf-actor:` is `human/...`, `aiwf-principal:` must be absent. | Codified in provenance-model.md. A direct human act has no second-actor split. | load-bearing | error |
| R-FP-0117 | trailers | `aiwf-on-behalf-of:` and `aiwf-authorized-by:` are required-together: both present or both absent. | Codified in provenance-model.md. Either both signal scope membership, or neither does. | load-bearing | error |
| R-FP-0118 | trailers | `aiwf-force:` requires `aiwf-actor: human/...`. The kernel refuses `--force` from non-human actors. | Codified in provenance-model.md §"The `--force` rule." Sovereign acts always trace to a named human. | load-bearing | error |
| R-FP-0119 | trailers | `aiwf-force:` and `aiwf-on-behalf-of:` are mutually exclusive. | Codified in provenance-model.md. Force is human-only; on-behalf-of implies an agent. They cannot coexist. | load-bearing | error |
| R-FP-0120 | trailers | `aiwf-on-behalf-of:` and `aiwf-actor: human/...` are mutually exclusive. | Codified in provenance-model.md. A direct human act has no on-behalf-of. | load-bearing | error |
| R-FP-0121 | trailers | `aiwf-authorized-by:` SHAs are validated at read time (every `aiwf check` pass), not write time. Three sub-cases: missing SHA (`provenance-authorization-missing`), out-of-scope reference (`provenance-authorization-out-of-scope`), ended scope (`provenance-authorization-ended`). | Codified in provenance-model.md. Write-time SHA validation gives only weak guarantees because SHAs can become stale via rebase/force-push. | load-bearing | error |
| R-FP-0122 | trailers | `aiwf-to:` records the target state of a promote (and the agent for an authorize). | Codified in design-decisions.md §"Acceptance criteria and TDD." Target state belongs in the structured trailer, not the commit subject. | load-bearing | error |
| R-FP-0123 | trailers | `aiwf-prior-entity:` is written by `aiwf reallocate` alongside `aiwf-entity:`. The bridge that keeps both ids' histories complete. | Codified in design-decisions.md §"Markdown is the source of truth." | load-bearing | error |
| R-FP-0124 | trailers | `aiwf-reason:` is non-empty after trim. Required on `--pause`, `--resume`, and `--force`; optional on `--to`. | Codified in provenance-model.md §"Trailer set." Empty reasons defeat the audit purpose. | load-bearing | error |

### 6c. Scope FSM

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0125 | scope FSM | Scope states are closed: `active`, `paused`, `ended`. | Codified in provenance-model.md. The closed set means the parser/renderer is bounded. | load-bearing | error |
| R-FP-0126 | scope FSM | `active → paused` is legal via `aiwf authorize <id> --pause "<reason>"`. | Codified in provenance-model.md. | load-bearing | error |
| R-FP-0127 | scope FSM | `paused → active` is legal via `aiwf authorize <id> --resume "<reason>"`. | Codified in provenance-model.md. | load-bearing | error |
| R-FP-0128 | scope FSM | `active → ended` and `paused → ended` are automatic when the scope-entity reaches a terminal status. Recorded by `aiwf-scope-ends: <auth-sha>` on the terminal-promote commit. | Codified in provenance-model.md §"Scope termination." | load-bearing | error |
| R-FP-0129 | scope FSM | `ended` is terminal. Un-canceling the scope-entity does not resurrect the ended scope; the human issues a new authorization. | Codified in provenance-model.md ("Q3.5: strict end-on-terminal"). | load-bearing | error |
| R-FP-0130 | scope FSM | A non-human actor's verb succeeds only if at least one active scope's reachability check passes (the verb's target entity reaches the scope-entity via the reference graph). | Codified in provenance-model.md §"Scope check." Without this, the gating function is trivially true and the scope concept is decorative. | load-bearing | error |
| R-FP-0131 | scope FSM | Human actors with no `--principal` flag bypass the scope check. Humans need no authorization to act. | Codified in provenance-model.md. Scopes constrain agents-acting-for-humans; humans are sovereign by themselves. | load-bearing | error |
| R-FP-0132 | scope FSM | When multiple active scopes match the same verb, the kernel picks the most-recently-opened scope deterministically. | Codified in provenance-model.md §"Multiple parallel scopes." Determinism is necessary for the recorded `aiwf-authorized-by:` to be reproducible. | load-bearing | error |

### 6d. Authorize verb

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0133 | `aiwf authorize` | `--to <agent>` opens a new scope on `<id>`. The verb refuses if `<id>` is in a terminal status (unless `--force --reason` overrides). | Codified in provenance-model.md. Authorizing work on a `done` epic is incoherent without explicit override. | load-bearing | error |
| R-FP-0134 | `aiwf authorize` | The verb refuses if the actor is not `human/...`. Only humans authorize. | Codified in provenance-model.md. Sub-agent delegation is deferred to G22. | load-bearing | error |
| R-FP-0135 | `aiwf authorize` | `--pause "<reason>"` requires a most-recently-opened active scope for `<id>`. If none, the verb refuses with `provenance-no-active-scope-to-pause`. | Codified in provenance-model.md. | load-bearing | error |
| R-FP-0136 | `aiwf authorize` | `--resume "<reason>"` requires a most-recently-paused scope for `<id>`. If none, the verb refuses with `provenance-no-paused-scope-to-resume`. | Codified in provenance-model.md. | load-bearing | error |
| R-FP-0137 | `aiwf authorize` | A scope is addressed by the SHA of its `authorize` commit. No separate scope id namespace. | Codified in provenance-model.md §"Scope id." | load-bearing | error |

**Total: 27 rules in §6.**

---

## 7. Verb execution invariants

These rules pin the cross-cutting properties every mutating verb must satisfy.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0138 | every mutating verb | Produces exactly one git commit (or no change at all on findings). | Codified in design-decisions.md §"One git commit per mutating verb." Atomicity of the unit-of-merge. | load-bearing | error |
| R-FP-0139 | every mutating verb | Validate-then-write pattern: the verb computes the projected new tree in memory, runs `aiwf check` against the projection, and only writes when the projection is clean. | Codified in design-decisions.md §"One git commit per mutating verb." Without this, partial mutations land before validation catches them. | load-bearing | error |
| R-FP-0140 | every mutating verb | On findings introduced by the projection, the working tree is never touched. | Codified in design-decisions.md. The rollback discipline preserves the "no change at all on findings" guarantee. | load-bearing | error |
| R-FP-0141 | every mutating verb | Pre-existing tree errors (a broken reference left over from a prior hand-edit) do not refuse an unrelated `aiwf add`. The diff is by `code + subcode + path + entity + message`. | Codified in design-decisions.md. Lets users incrementally fix partial breakage rather than first cleaning up by hand. | load-bearing | error |
| R-FP-0142 | every mutating verb | Acquires an exclusive lock on `<root>/.git/aiwf.lock` (POSIX `flock`) before reading the tree. Read-only verbs do not lock. | Codified in design-decisions.md §"One git commit per mutating verb." Prevents two mutations racing on id allocation. | load-bearing | error |
| R-FP-0143 | every mutating verb | Writes the standard trailer set (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`, plus context-specific trailers per §6b). | Codified in design-decisions.md and CLAUDE.md §"Commit conventions." | load-bearing | error |
| R-FP-0144 | every read-only verb | Does not lock; remains free to run concurrently with mutations. Read-only set: `check`, `history`, `status`, `render` without `--write`, `doctor`, `whoami`. | Codified in design-decisions.md §"One git commit per mutating verb." | load-bearing | n/a |
| R-FP-0145 | sovereign override | `--force --reason "<text>"` allows any-to-any FSM transition. Reason is required, non-empty after trim. Coherence checks (id format, closed-set membership, ref resolution) still run. | Codified in design-decisions.md §"Acceptance criteria and TDD." Force relaxes only the transition rule, not the coherence rules. | load-bearing | error |
| R-FP-0146 | sovereign override | The `aiwf-force: <reason>` trailer lands on the forced commit, alongside the standard trailers. | Codified in design-decisions.md and provenance-model.md. The trailer is what makes forced acts auditable. | load-bearing | error |

**Total: 9 rules in §7.**

---

## 8. Archive convention

These rules pin the per-parent archive subdirectory convention codified in ADR-0004.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0147 | archive | Terminal-status entities live under per-parent `archive/` subdirectories. Active entities live in the parent directory itself. | Codified in ADR-0004. The on-disk projection of terminality. | load-bearing | error |
| R-FP-0148 | archive | Movement is decoupled from FSM promotion: `aiwf promote` and `aiwf cancel` flip status only; `aiwf archive` is the sweep verb. | Codified in ADR-0004 §"Decision." Promotion verbs stay one-purpose. | load-bearing | error |
| R-FP-0149 | archive | `aiwf archive` produces exactly one commit per `--apply` invocation, regardless of how many entities are swept. | Codified in ADR-0004 §"`aiwf archive` verb." Single-commit-per-verb specialization. | load-bearing | error |
| R-FP-0150 | archive | The default mode is dry-run. `--apply` is required to commit. | Codified in ADR-0004. Mechanical safety: the verb name's affordance is "sweep what's terminal," not "guess what should be archived." | load-bearing | n/a |
| R-FP-0151 | archive | Milestones do not archive independently — they ride with the parent epic when the epic archives. | Codified in ADR-0004 §"Storage." Milestones are flat files inside the epic directory; per-kind archive doesn't apply at the sub-element level. | load-bearing | error |
| R-FP-0152 | archive | Reversal is not provided. No `aiwf reactivate` or `aiwf un-archive` verb exists. The canonical pattern is to file a new entity referencing the archived one. | Codified in ADR-0004 §"Reversal." Forward-only by design. | load-bearing | n/a |
| R-FP-0153 | archive | A file in `archive/` whose frontmatter status is not terminal fires `archived-entity-not-terminal` (error). Remediation is to revert the status, not relocate the file. | Codified in ADR-0004 §"`aiwf check` shape rules." | load-bearing | error |
| R-FP-0154 | archive | A file in an active dir whose status is terminal fires `terminal-entity-not-archived` (advisory). Counted by `archive-sweep-pending`. | Codified in ADR-0004. The normal transient state under the decoupled model. | load-bearing | warning |
| R-FP-0155 | archive | The `archive.sweep_threshold` config knob in `aiwf.yaml` flips `archive-sweep-pending` from advisory to blocking past the named count. | Codified in ADR-0004. Teams choose their own discipline. | conventional | error |
| R-FP-0156 | archive | Tree-integrity rules (`ids-unique`, parse-level errors) traverse both active and archive. Shape and health rules (`acs-shape`, `entity-body-empty-ac`, etc.) skip archive. | Codified in ADR-0004 §"`aiwf check` shape rules." Forget-by-default for archive. | load-bearing | error |
| R-FP-0157 | archive | `aiwf list` shows active by default; `--archived` includes archived. `aiwf status` is strictly active-only. `aiwf show <id>` resolves regardless of location. | Codified in ADR-0004 §"Display surfaces." Active-by-default discoverability inversion. | load-bearing | n/a |

**Total: 11 rules in §8.**

---

## 9. Validation chokepoint rules

These rules pin the pre-push hook and `aiwf check` as the mechanical chokepoint.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0158 | check | `aiwf check` is a pure function from the working tree to a list of findings. It loads inconsistent state and reports it; it does not refuse to start. | Codified in CLAUDE.md §"Engineering principles" ("Errors are findings, not parse failures"). | load-bearing | error |
| R-FP-0159 | check | `aiwf check` runs as a pre-push git hook installed by `aiwf init`. The hook is the mechanical chokepoint. | Codified in design-decisions.md §"Validation is the chokepoint." The hook is what turns guarantees into enforcement. | load-bearing | error |
| R-FP-0160 | check | `aiwf check --shape-only` runs as a pre-commit hook. The shape pass is fast and catches structural issues before they accumulate. | Codified in CLAUDE.md §"What's enforced and where." | load-bearing | error |
| R-FP-0161 | check | `--no-verify` bypasses the hook (standard git behavior). The framework does not try to prevent bypass — bypassing is sometimes the right call. | Codified in design-decisions.md §"Validation is the chokepoint." Sovereign override at the git layer. | load-bearing | n/a |
| R-FP-0162 | check | The exit code is 0 (clean), 1 (findings), 2 (usage error), or 3 (internal error). Verbs return `int`; the CLI adapter wraps to `*exitError`. | Codified in CLAUDE.md §"CLI conventions." Distinct codes let CI scripts and downstream tools branch correctly. | load-bearing | error |
| R-FP-0163 | check | Findings produced by a verb's projection (the new finding set minus the pre-existing set, diffed by code+subcode+path+entity+message) block the verb. Pre-existing findings do not block unrelated verbs. | Codified in design-decisions.md §"One git commit per mutating verb." | load-bearing | error |
| R-FP-0164 | check | Closed-set finding codes; each code has a known severity (error/warning/info). | Codified in design-decisions.md §"Validation is the chokepoint" and ADR-0004 §"`aiwf check` shape rules." | load-bearing | error |
| R-FP-0165 | check | Contract verify+evolve passes run as part of `aiwf check` when `aiwf.yaml.contracts` declares bindings. The pre-push hook stays a single chokepoint. | Codified in design-decisions.md §"Contracts." | load-bearing | error |

**Total: 8 rules in §9.**

---

## 10. Anti-rules (explicitly NOT kernel rules)

This section catalogs things that are deliberately *not* kernel rules — sometimes because they were considered and rejected, sometimes because they live in a separate layer (skills, rituals, CI). These rules clarify the kernel's scope by negation; without them, contributors are likely to introduce constraints that the model does not commit to.

| Rule id | Scope | Statement | Reasoning | Load-bearing? | Severity if violated |
|---|---|---|---|---|---|
| R-FP-0166 | anti-rule | A milestone is **not** required to have ≥1 AC. ACs are optional. | Codified in design-decisions.md §"What's not a kernel rule." The kernel guards the AC outcome, not its existence. | load-bearing | n/a |
| R-FP-0167 | anti-rule | A milestone is **not** required to enter `in_progress` with all ACs in `tdd_phase: red`. The kernel guards only the outcome (`met` requires `done`). | Codified in design-decisions.md §"What's not a kernel rule." The flow is the rituals plugin's concern. | load-bearing | n/a |
| R-FP-0168 | anti-rule | There is **no** global AC allocator. AC ids are per-milestone (R-FP-0063). | Codified in design-decisions.md. | load-bearing | n/a |
| R-FP-0169 | anti-rule | There is **no** AC tombstone beyond status-cancel. The position-stable position-in-`acs[]` retains the cancelled AC. | Codified in design-decisions.md §"What's not a kernel rule." | load-bearing | n/a |
| R-FP-0170 | anti-rule | There is **no** `aiwf reactivate` or `aiwf un-archive` verb. Archive is forward-only (R-FP-0152). | Codified in ADR-0004 §"Reversal." | load-bearing | n/a |
| R-FP-0171 | anti-rule | There is **no** event log file, no graph projection file, no hash chain, no monotonic ID counter. | Codified in design-decisions.md §"What the framework needs to do" and §"What is deliberately not in the PoC." | load-bearing | n/a |
| R-FP-0172 | anti-rule | There is **no** kernel rule about which branch a verb is legal on. Branch choreography is ADR-0010's layer 4, out of E-0033's scope. | Codified in ADR-0010 and ADR-0011 §Scope. | load-bearing | n/a |
| R-FP-0173 | anti-rule | The kernel makes **no** assumption about which Claude Code plugins a consumer should have installed. `aiwf.yaml.doctor.recommended_plugins` is opt-in; default empty. | Codified in design-decisions.md §"aiwf.yaml config." | load-bearing | n/a |
| R-FP-0174 | anti-rule | The kernel does **not** ship validator binaries (`cue`, `ajv`). Validators are declared in `aiwf.yaml.contracts.validators` and installed via the user's toolchain. | Codified in design-decisions.md §"Contracts." The engine owns orchestration; the user owns validators. | load-bearing | n/a |
| R-FP-0175 | anti-rule | `--force` cannot be wielded by a non-human actor. A future delegated-force flag (G23) is deferred. | Codified in provenance-model.md. Sovereign acts always trace to a named human. | load-bearing | n/a |
| R-FP-0176 | anti-rule | There is **no** "milestone-must-pre-fail-tests-before-in_progress" rule. The TDD discipline is rituals-plugin-driven, not kernel-driven. | Codified in design-decisions.md §"What's not a kernel rule." | load-bearing | n/a |

**Total: 11 rules in §10.**

---

## Grand total

- §1 Per-kind lifecycles: 45 rules
- §2 ACs and TDD: 26 rules
- §3 Cross-entity invariants: 22 rules
- §4 Frontmatter schema: 9 rules
- §5 ID format and stability: 8 rules
- §6 Provenance model: 27 rules
- §7 Verb execution invariants: 9 rules
- §8 Archive convention: 11 rules
- §9 Validation chokepoint: 8 rules
- §10 Anti-rules: 11 rules

**Grand total: 176 rules.**

---

## Open questions for Pass C

These are points where first-principles reasoning was ambiguous or where multiple equally-defensible derivations are possible. Pass C (M-0123) reconciles each against Pass A's catalog and produces an explicit decision entity per unresolved case.

1. **Q1 — AC `deferred` terminality.** R-FP-0051 marks `deferred → open` as conventional rather than load-bearing. Design-decisions.md only lists `deferred` and `cancelled` as terminals for AC, but doesn't pin whether `deferred` can return to `open` when work resumes. Pass A's extraction from `internal/entity/transition.go` should reveal whether the implementation allows the reverse. **Decision needed:** is `deferred → open` a legal transition, or do operators file a new AC?

2. **Q2 — AC self-promote no-op (`open → open`).** R-FP-0053 assumes the FSM refuses self-promotion as a usability concern. The model doesn't pin this either way. **Decision needed:** does the FSM admit `X → X` as a no-op, refuse it, or commit a redundant promote with a trailer?

3. **Q3 — ADR `accepted → rejected` legality.** R-FP-0021 marks this conventional (not load-bearing). The supersession path preserves history; direct demotion would not. **Decision needed:** is direct `accepted → rejected` admitted (perhaps gated by `--force`), or is it forbidden outright?

4. **Q4 — Contract `accepted → rejected` legality.** R-FP-0045 mirrors Q3 for contracts. Same shape of decision.

5. **Q5 — Epic cancellation cascade to milestones.** R-FP-0074 marks the exact cascade mechanism (auto-cancel children vs. require explicit per-child cancellation) as conventional. **Decision needed:** when `aiwf cancel E-NNNN` runs, what happens to that epic's `in_progress` and `draft` milestones — auto-cancel, refuse with a listing, or warn-and-proceed?

6. **Q6 — Milestone cancellation cascade to ACs.** R-FP-0064 mirrors Q5 at the AC layer. **Decision needed:** when a milestone is cancelled, do its `open` ACs auto-cancel, inherit some other terminal, or refuse the cancel?

7. **Q7 — AC promotion mechanical-evidence rule.** R-FP-0066 marks the "AC promotion requires mechanical evidence" rule as conventional because the kernel doesn't enforce a test-existence check. CLAUDE.md treats it as discipline. **Decision needed:** is this elevated to a kernel finding-rule (and how is "test exists" detected mechanically), or does it stay as reviewer-discipline?

8. **Q8 — `gap.addressed_by` requirement at addressed-promotion.** R-FP-0087 marks the addressed-by requirement as conventional rather than load-bearing — a gap could be addressed by prose alone, without a structured reference. **Decision needed:** is at least one `addressed_by` reference required at the moment a gap transitions to `addressed`?

9. **Q9 — Archive sweep threshold default.** R-FP-0155 notes the `archive.sweep_threshold` config knob. **Decision needed:** is there a sensible default beyond "unset" — e.g., "warn past 50 entries"? Or does the kernel deliberately not opine?

10. **Q10 — Anti-rule completeness.** §10 enumerates eleven anti-rules. **Decision needed:** are there other near-rules — patterns that contributors might assume are kernel-enforced but aren't — worth explicitly cataloging? Examples that surfaced during derivation but weren't included:
    - No "every gap must reference a `discovered_in`" rule (the field is optional).
    - No "every ADR must be linked from at least one entity" rule.
    - No "epic must have ≥1 milestone before transitioning to active" rule.

11. **Q11 — Verb-output scope.** This catalog focuses on entity-state invariants and trailer rules. It deliberately does not catalog:
    - Per-verb `--format=json` envelope shape.
    - Per-finding code's exact message string.
    - The closed-set list of finding codes (only the categories are derived; the exact code strings are an implementation choice).
    
    **Decision needed:** which of these belong in the spec table that M-0123 produces? The argument for inclusion is mechanical regression protection; the argument for exclusion is YAGNI for the PoC.

12. **Q12 — Read-only verb set.** R-FP-0144 lists `check`, `history`, `status`, `render` without `--write`, `doctor`, `whoami` as read-only. The list could grow (`list`, `show`, `version`, `whoami`). **Decision needed:** is the read-only set itself part of the spec, or is it derived from "does this verb call into the lock"?

13. **Q13 — Closed-set status names per kind.** §1's per-kind tables enumerate transitions but assume the status names are exactly as written in design-decisions.md. Pass A may surface implementation deviations (e.g., an implementation that recognizes `in-progress` vs `in_progress` for backwards compatibility). **Decision needed:** is the spec the authoritative status-name source, or are aliases tolerated?

14. **Q14 — Scope reachability completeness.** R-FP-0130 says "the verb's target entity reaches the scope-entity via the reference graph." The reference graph's exact composition (which fields participate: `parent`, `depends_on`, `addressed_by`, `relates_to`, `discovered_in`?) is described in provenance-model.md as "the chain is bounded by the existing kind reference grammar." **Decision needed:** is the participating-fields set part of the spec table, or is it derived from "every field listed in the per-kind reference fields table"?

15. **Q15 — Authorize verb scope-entity restrictions.** R-FP-0133 says the scope-entity must not be in terminal status. The model doesn't pin whether all six kinds are equally authorizable, or whether only epics/milestones make sense as scope-entities. **Decision needed:** is there a kind restriction on `aiwf authorize <id>`, or does the verb accept any kind?

---

*End of Pass B catalog. Pass C (M-0123) reconciles this against Pass A's `legal-workflows-audit.md` and produces the canonical Go spec table.*
