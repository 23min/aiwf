# Parallel TDD subagents with finding-gated AC closure

This is the design synthesis for [E-19](../../../work/epics/E-19-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md). It captures the conversation that produced the epic and its dependent ADRs ([ADR-0003](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md), [ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)).

The intent: future readers (human or LLM) auditing the epic don't have to reverse-engineer the design from commit messages and ADR prose. The forks the design considered, the tradeoffs of the rejected paths, and the kernel principles each choice rested on are written down once, here.

---

## Motivation

The proximate trigger is **M-066/AC-1**, where a long implementation session lost track of branch-coverage discipline mid-cycle. The TDD-cycle skill (`wf-tdd-cycle`) was advisory text — easy to drift through under the pressure of a long conversation. The retrospective finding: the framework's correctness can't depend on the LLM remembering rules over many turns. That's exactly the kernel principle from CLAUDE.md ("framework correctness must not depend on LLM behavior") applied to the TDD cycle itself.

The proposed fix isn't a stricter skill prompt; it's a **structural** one: bound the cycle's lifetime to a subagent invocation. A subagent that starts fresh, sees only the AC contract + the relevant code, and returns when done can't drift the way a long conversation can. The protocol *is* the lifetime of the agent.

That immediately raises four questions:

1. **What happens to concerns the subagent surfaces?** Branch-coverage gaps, weak assertions, scope leaks need a durable place to live until a human triages them.
2. **What can the subagent do inside its worktree?** AC promotions, finding allocation, code edits — which are subagent-driven, which are parent-driven?
3. **How are findings resolved?** Once flagged, what's the verb surface for the human triaging them?
4. **Where do findings get written, when, and by whom?**

The four forks below settle each of these.

---

## The four forks and their resolutions

### Fork 1: Findings storage

**Resolution: F-NNN as a seventh entity kind, with a uniform archive convention.**

Frontmatter arrays on the AC are simplest but fail at the moments that matter — cross-AC findings (one finding affects multiple ACs), long-form repro/triage prose, stable references from gaps and decisions (`Resolves: F-007`), escalation from `aiwf check` itself. Sibling files per AC give file-level isolation but introduce a new file pattern without solving cross-AC scope.

Adding a new entity kind is a kernel-level decision (amends principle #1 from "six kinds" to "seven"). The case for it rests on three points:

- The PoC is likely to graduate, and frontmatter arrays read as a stopgap once the full requirements stack is laid out.
- Findings sit naturally with the existing governance kinds (gap, decision, ADR) — they're flags on the planning tree, not execution units. The deliberate exclusion of `story`/`task` (CLAUDE.md, framework_entity_vocabulary memory) was specifically about execution units; findings are different in shape.
- One discipline for "things that need human attention" — TDD-cycle findings *and* `aiwf check` escalations live in the same surface. Without F-NNN, those would split into parallel mechanisms.

The directory-bloat concern (`work/findings/` would fill fastest of any kind) is real. The companion archive ADR generalizes the precedent already in the repo (`docs/pocv3/archive/gaps-pre-migration.md`) into a kernel convention: terminal-status promotion moves the file to `<kind>/archive/` in the same atomic commit. Applies uniformly across kinds; helps the existing 66-gap directory immediately as a side benefit.

Full rationale and alternatives in [ADR-0003](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) and [ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md).

### Fork 2: Subagent's edit surface

**Resolution: Hybrid. Subagent moves AC state in its worktree; parent allocates F-NNN post-merge.**

Three things move during a cycle: AC phase/status, F-NNN findings, and code/test files. Each can be subagent-driven or parent-driven independently.

- **Full-agency (subagent does everything)** is mental-model-clean but creates routine `ids-unique` collisions on every parallel cycle (resolved by `reallocate`, but it's friction every time).
- **Code-only subagent (parent does all aiwf verbs)** has the cleanest separation, but the AC's phase moves end up committed *after* the cycle, producing audit-ugly out-of-order phase history.
- **Hybrid** — AC moves with the cycle (the natural locus); F-NNN allocation centralized in the parent (dense ids, no race) — lines up cleanly with the data type of each thing being moved. AC promotions are imperative state moves; findings are typed structured data, perfectly suited to JSON return + parent-side recording.
- **Reservation-range allocation** (pre-allocate F-NNN ranges per subagent) avoids collisions but produces sparse ids; rejected as overengineering before [ADR-0001](../../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) (which solves the same problem more elegantly via inbox/mint).

The hybrid choice produces a **bounded edit-scope rule** for subagents that's a single sentence: "this AC's status + these declared filesets, nothing else." Easy to express in the agent definition's system prompt; easy to audit post-cycle.

This resolution settles Fork 4 implicitly: with parent owning F-NNN allocation, findings are recorded post-merge in the main checkout, not inside the worktree.

### Fork 3: Finding resolution UX

**Resolution: Generic `aiwf promote` with `--force --reason`; soft check on missing fix link.**

The kernel already invested in `aiwf promote --force --reason` as the universal terminal-transition pattern (M-017). Findings using the same pattern is consonant; not using it is novel-for-no-reason. F-NNN's whole point (per Fork 1) was to get the standard kernel treatment — status FSM, history, archive. Bare `aiwf promote` is what "standard kernel treatment" looks like.

A dedicated `aiwf finding {resolve,waive,invalidate}` verb family would duplicate `aiwf promote` and set a precedent for verb-family-per-kind that violates the kernel's minimum-verb-surface posture. The metadata-capture concern (how do humans see what fixed F-007?) is already solved by the trailer convention: `git log --grep "aiwf-entity: F-007"` shows the fix commit by definition; `aiwf history F-007` walks it for you.

A body-section validator (`aiwf check` requires `## Resolution` section on `resolved` transitions, analogous to M-066's `entity-body-empty` rule) is the rigorous version of Fork 3. It's deferred — wait for M-066's pattern to generalize across kinds before adding finding-specific body validation. The soft check on "resolved without an associated fix commit nearby" covers the most common discipline gap until then.

### Fork 4: Where findings get recorded

**Settled by Fork 2's hybrid resolution.** Subagent returns findings as JSON in its envelope; parent walks the JSON and calls `aiwf add finding` per finding (one commit each per kernel rule) after merging the worktree. F-NNN allocation stays serial in the parent → dense ids, no collisions, no race.

---

## End-to-end flow

```
┌─ Parent (main checkout) ───────────────────────────────────────────┐
│  1. Identify N independent ACs in the milestone (disjoint files)   │
│  2. Spawn N subagents in parallel, each with isolation:"worktree"  │
│     Each gets: AC id, declared fileset, system-prompt protocol     │
└────────────────────────────────────────────────────────────────────┘
            │                                            │
            ▼                                            ▼
┌─ Subagent (worktree-1) ──────┐         ┌─ Subagent (worktree-2) ──────┐
│  • aiwf promote AC-1 red     │         │  • aiwf promote AC-2 red     │
│  • write failing test        │         │  • write failing test        │
│  • aiwf promote AC-1 green   │         │  • aiwf promote AC-2 green   │
│  • write impl                │         │  • write impl                │
│  • run audit (branch-walk)   │         │  • run audit (branch-walk)   │
│  • aiwf promote AC-1 done    │         │  • aiwf promote AC-2 done    │
│  • return JSON: {            │         │  • return JSON: {            │
│      diff, tests, audit,     │         │      diff, tests, audit,     │
│      findings: [...]         │         │      findings: [...]         │
│    }                         │         │    }                         │
└──────────────────────────────┘         └──────────────────────────────┘
            │                                            │
            └────────────────────┬───────────────────────┘
                                 ▼
┌─ Parent (main checkout) ───────────────────────────────────────────┐
│  3. Merge worktrees serially to milestone branch                   │
│  4. Run aiwf check after each merge (catches ids-unique, etc.)     │
│  5. For each subagent's findings list: aiwf add finding F-NNN      │
│     with linked_acs frontmatter pointing back to the AC            │
│  6. Surface to human: "M-066 cycle done. AC-1: 2 findings (F-007,  │
│     F-008). AC-2: clean. Triage before closure."                   │
│  7. Stop. Wait for human.                                          │
└────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─ Human ──────────────────────────────────────────────────────────────┐
│  • Reads F-007, F-008 (or aiwf show / aiwf status)                  │
│  • For each: aiwf promote F-NNN resolved   (with fix commit)        │
│              aiwf promote F-NNN waived --force --reason "..."       │
│              aiwf promote F-NNN invalid --reason "..."              │
│  • Once all linked findings terminal:                                │
│    aiwf promote M-066/AC-1 met                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Data model summary

**New entity kind: `finding` (F-NNN)**

- Status FSM: `open → resolved | waived | invalid` — all terminal.
- Frontmatter:
  - `code` — stable code from a kernel-pinned set (initial: `branch-coverage-gap`, `weak-assertion`, `scope-leak`, `audit-skipped`, `convention-violation`, `discovery-gap`, `discovery-decision`, `ac-split-suggested`).
  - `linked_acs` — composite AC ids this finding blocks. Empty for non-AC-tied findings (e.g., escalated check findings on a milestone).
  - `linked_entities` — any other entity ids the finding pertains to.
  - `recorded_by` — provenance (`ai/claude`, `framework/aiwf-check`, `human/<email>`).
- Body: free-form prose. Optional `## Resolution` / `## Waiver` sections (soft-checked initially).
- Storage: `work/findings/F-NNN-<slug>.md`. Terminal entries archive to `work/findings/archive/`.

**Uniform archive convention (all kinds)**

- Terminal-status promotion moves the file to `<kind>/archive/` in the same atomic commit.
- Id-resolver scans both directories. References stay valid.
- `aiwf list` / `aiwf status` filter active by default; `--include-archived` shows everything.
- See [ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) for the full spec.

---

## Verb surface

| Verb | Change |
|---|---|
| `aiwf add finding` | New (subverb of existing `aiwf add`). Allocates F-NNN, sets `linked_acs`. Parent calls this post-merge, one finding at a time. |
| `aiwf promote F-NNN <terminal>` | New status set, but reuses existing `aiwf promote` machinery. Triggers archive move per the convention. |
| `aiwf promote M-NNN/AC-N met` | **Chokepoint**: refuses with `findings-block-met` if any `open` finding has this AC in `linked_acs`. Override via `--force` (human-only by existing rule). |
| `aiwf promote <any> <terminal>` | All kinds: terminal promotion now moves the file to `archive/`. |
| `aiwf check` | New finding code: `ac-has-open-findings` (informational; the actual block is at promote-time). |

No new verb families. The kernel's universal `aiwf promote`, `aiwf show`, `aiwf history`, `aiwf check` cover findings without per-kind specialization.

---

## Enforcement chokepoints

- **AC closure**: `aiwf promote AC met` reads linked findings, refuses on any `open`. Mechanical, kernel-side.
- **Subagent edit scope**: enforced two ways — (a) subagent's system prompt restricts allowed paths and verbs; (b) parent audits the worktree diff post-cycle, treats out-of-scope changes as a `scope-leak` finding (and may refuse the merge).
- **Sovereignty**: `aiwf promote F-NNN waived` and `aiwf promote F-NNN invalid` require `--force` + `--reason`, which per the existing kernel rule means human-actor only. Subagents structurally cannot waive their own findings.
- **Id collisions**: don't happen (parent allocates F-NNN serially). AC promotion collisions during parallel cycles are detected by `aiwf check` post-merge via existing rules. Under [ADR-0001](../../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) (proposed inbox/mint model), the collision surface vanishes structurally.

---

## Provenance

Falls out cleanly from the existing principal × agent × scope model in [`provenance-model.md`](provenance-model.md):

- Subagent commits inside its worktree carry `aiwf-actor: ai/claude` (or whatever scope the parent's `aiwf authorize` opened).
- Parent's `aiwf add finding` commits also `ai/claude` (parent is also an LLM agent).
- Human's `aiwf promote F-NNN waived --force` carries `aiwf-actor: human/<email>`. Trailers show the sovereign act.
- `--force` is human-only by the existing rule, which gives finding waiver/invalidation the same sovereignty treatment as cancellation, retitling, and other consequential acts.

---

## Open / deferred items

These are real design questions that don't have to be answered before E-19's dependent ADRs are accepted and the implementation epics start landing:

1. **Finding-code enumeration finalization.** Initial set is enumerated above; the full set settles when each producer (cycle protocol, escalated check rules) ships. New codes are kernel-pinned at the same time as their producing rules.
2. **Subagent agent definition (system prompt).** The protocol contract the subagent must follow. Lives under `.claude/agents/tdd-cycle.md` (or the host's equivalent). Detailed authoring deferred to E-19's first milestone.
3. **Parent's parallelization heuristic.** When does the parent choose to parallelize? "When ACs declare disjoint filesets" is the conservative answer. Whether the parent infers this or the milestone spec declares it explicitly is open until dogfooding shows what's needed.
4. **Body-section validator generalization.** Eventually F-NNN bodies should require `## Resolution` / `## Waiver` sections on terminal promotions (the rigorous version of Fork 3). Wait for M-066's body-section pattern to settle and generalize first.
5. **`aiwf reframe F-007 --as-gap` verb.** "This finding really wants to be a gap" — would resolve F-007 and pre-fill a G-NNN with linked context. Nice-to-have, not required. Cross-references already work via `linked_entities` without a dedicated verb.
6. **Cross-cycle findings.** A finding that pertains to no specific AC but to the milestone or epic as a whole. Data model supports this (empty `linked_acs` + non-empty `linked_entities`); no current producer emits them. Dogfooding will surface what's needed.

---

## Sequencing

Items 1-2 are pure design (ADRs already filed). Items 3-5 are kernel work, each ~1 milestone. Item 6 is the user-visible payoff and depends on the prior items.

1. **[ADR-0003](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md)** — `finding` as a 7th entity kind. **Filed.** Pending acceptance.
2. **[ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)** — Uniform archive convention. **Filed.** Pending acceptance.
3. **Implementation epic for archive convention** (filed once ADR-0004 is accepted) — kernel-wide change; lower-risk if landed before F-NNN, since findings ride the existing pattern.
4. **Implementation epic for F-NNN entity kind** (filed once ADR-0003 is accepted) — adds the kind enum, FSM, status set; `aiwf add finding` subverb; `aiwf show F-NNN` rendering; `aiwf history F-NNN` works for free.
5. **Implementation epic for findings-gated AC closure** (filed alongside item 4) — adds the `aiwf promote AC met` chokepoint; new check finding code `ac-has-open-findings`.
6. **[E-19](../../../work/epics/E-19-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md)** — Parallel TDD subagents with finding-gated AC closure. **Filed as draft.** The user-visible payoff; depends on items 3-5.

---

## References

- [ADR-0003](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — Add finding (F-NNN) as a seventh entity kind.
- [ADR-0004](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — Uniform archive convention for terminal-status entities.
- [ADR-0001](../../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) — Mint entity ids at trunk integration via per-kind inbox state. Compatible with this design; F-NNN inherits whichever id-allocation model the framework adopts.
- [E-19](../../../work/epics/E-19-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) — Parallel TDD subagents epic (depends on the ADRs above).
- [`design-decisions.md`](design-decisions.md) — kernel principles, including #1 (entity kinds) which ADR-0003 amends.
- [`provenance-model.md`](provenance-model.md) — principal × agent × scope; the sovereignty rules this design relies on.
- [`tree-discipline.md`](tree-discipline.md) — existing tree-shape rules; ADR-0004 adds a sub-rule.
- [`id-allocation.md`](id-allocation.md) — existing id allocation and lineage model; F-NNN slots into it directly.
- CLAUDE.md "What the PoC commits to" §1 (six entity kinds — amended by ADR-0003).
- M-066/AC-1 — the proximate trigger; cycle drift case study.
