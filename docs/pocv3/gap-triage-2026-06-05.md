# aiwf Open-Gap Triage

**Date:** 2026-06-05
**Branch:** epic/E-0030-branch-model-chokepoint
**Companion to:** [`health-scorecard-2026-06-04.md`](health-scorecard-2026-06-04.md)
**Total open gaps reviewed:** 74

## Summary

The open-gap pile has grown to 74 entries — enough to trigger maintainer anxiety ("we'll never get done"). This triage is the honest read: what's actually in the pile, where the real signal is, what should close as `wontfix`, what should promote to epics, and which apparent gaps are really one mini-epic with multiple symptoms.

**Headline:** the pile is **~75% real signal, ~20% scorecard-noise (filed in the last week), ~5% genuine `wontfix`**. The kernel is not getting worse — the gap-filing discipline is working faster than the gap-closing throughput. Of 74 open gaps, **4 are real bugs that should land soon, 9 are critical hardening, ~10 should close as `wontfix`, and ~25 cluster into 5 mini-epics that should be folded.** After the recommended sweep, the standing backlog drops from 74 to ~25.

This doc is a snapshot. The durable record is the gaps themselves; this is what we saw on this date.

## Age distribution

Sample of 30 open gap creation dates spans 2026-05-05 to 2026-06-05.

- **Median age:** ≈ 8 days
- **Oldest open gap:** 31 days (G-0022, G-0023)
- **<2 weeks old:** 60%
- **<3 days old:** 30%

**The pile is almost entirely fresh.** Nothing is stale in the >3-month sense. The age distribution alone tells the real story: the maintainer's friction is from *filing velocity*, not from accumulated debt.

## Tier rubric

| Tier | Definition | Action implication |
|:--|:--|:--|
| **T0** | Real defect — silent failure, broken policy, security/correctness bug, data-loss risk | Should land soon; not optional |
| **T1** | Critical hardening — a chokepoint missing for a property the kernel already commits to | High-value; address in next milestone window |
| **T2** | Quality-of-life — improves operator UX, developer ergonomics, or kernel completeness | Worth doing; not urgent |
| **T3** | Polish — refactor, naming, prose tweaks, documentation hygiene | Pick up when in the file; survivable indefinitely |
| **T4** | Speculative / future-epic parking — described as "future work" or "noticed for a future epic" | Probably belongs as an epic or a deferred note, not a gap |
| **WF** | `wontfix` candidate — honest call that the cost > value, or scope/intent has drifted past relevance | Close as `wontfix` with a one-line rationale |
| **DUP** | Overlaps substantively with another open gap or with already-addressed work | Merge into the other or close as duplicate |

## At-a-glance distribution

| Tier | Count | % of pile |
|:--|--:|--:|
| T0 (real defects) | 4 | 5% |
| T1 (critical hardening) | 9 | 12% |
| T2 (quality-of-life) | ~20 | 27% |
| T3 (polish) | ~20 | 27% |
| T4 (epic-shaped parking) | 8 | 11% |
| WF (wontfix candidates) | 10 | 14% |
| DUP / mis-listed | 3 | 4% |

## T0 — Real bugs landing soon (4)

These are not aspirational chokepoints; they are broken things.

| ID | Bug | Cost to fix |
|:--|:--|:--|
| **G-0226** | `aiwf acknowledge-illegal` hard-requires SHA reachable from HEAD; force-push orphans have no override path. Documented use case returns exit 2. | ~1 day |
| **G-0067** | TDD red-first / branch-coverage chokepoint missing. Every `tdd: required` claim is honor-system today. Augmented 2026-06-04 with the diff-coverage gate proposal. | Medium milestone |
| **G-0231** | `PolicyTrailerKeysViaConstants` regex is structurally broken; CI is green on a check producing zero violations against known violations. Silent green is worse than no check. | ~2 days |
| **G-0221** | Disk-level atomic writes; OS-crash-mid-write leaves half-written entity files. Two callsites already do it correctly; consolidating into one helper is straightforward. | ~3 days |

## T1 — Critical hardening (9)

Properties the kernel already commits to but doesn't yet mechanize.

| ID | Title | Why now |
|:--|:--|:--|
| **G-0179** | Enforce full local CI gate (golangci-lint) at wrap on unpushed branches | 9 latent lint failures rode through three milestone wraps on E-0038 |
| **G-0218** | Operator-typed commit messages bypass aiwf-verb registry at composition | Fabricated `aiwf-verb: merge` trailers shipped twice in one wrap cycle |
| **G-0216** | Empty AC body blocks milestone draft→in_progress promote | Concrete failure mode of contract-first TDD discipline |
| **G-0184** | aiwf check misses invented id-shaped tokens; no rule against fabricating ids | Direct chokepoint against the most common LLM failure mode |
| **G-0140** | Implement `--evidence` flag on `aiwf promote AC met` per D-0005 | Decision already landed; this is the implementation |
| **G-0163** | ADR/Decision accepted cancel routes through FSM-illegal target | FSM forbids `accepted→rejected` but the verb allows it; real bad-state bug |
| **G-0166** | RejectionLayerCheckTime cells rejected at verb-time | Spec/impl drift on two M-0123 cells |
| **G-0220** | Ritual SKILL.md edits without structural AC pins have no mechanical backstop | Sister of G-0067 — discipline holds because maintainer remembers |
| **G-0195** | canonicalTrailerKeys drifts from trailerOrder; no mirror-validity guard | Drift-prevention test that itself has drift |

## WF — `wontfix` candidates (10)

Each is a gap whose body either offers "accept as-is" as a valid resolution path, or whose premise is speculative/YAGNI per the author's own words.

| ID | Reason it can close |
|:--|:--|
| **G-0074** | Body explicitly offers "Accept" as resolution. docs/pocv3/ PoC framing is cosmetic; nobody confused. |
| **G-0075** | Body says "reads correctly enough." docs/pocv3/ directory name signals vintage, works fine. |
| **G-0077** | Post-promotion working paper. Body says "start as a gap; shape into work when ready." It's a TODO note, not a kernel concern. |
| **G-0104** | Test-parallelism shipping mechanism. "About whether to prescribe for all potential future consumers." There are no consumers yet. YAGNI. |
| **G-0023** | Delegated --force. Body literally says "YAGNI for the PoC." |
| **G-0060** | Patch ritual loosely defined. The gap *is* the question, not a fix. Belongs as an ADR/decision; close gap once decision lands. |
| **G-0068** | Dynamic finding subcodes discoverability. Operator workaround exists; nobody bitten twice; theoretical and narrow. |
| **G-0178** | Prove non-Claude agent target (Codex). Speculative until a real second consumer asks. |
| **G-0116** | aiwfx-start-epic worktree-before-promote on trunk-based. Body says fix is "advisory until G-0059 resolves." Dependent on unresolved upstream gap; redundant filing. |
| **G-0213** | cellcoverage fixture writes fictional aiwf-branch values. Duplicate of G-0197 with different framing. Merge. |

## T4 — Epic-shaped parking (8)

These are described as "future epic" or carry kernel-wide scope. They should be promoted to epics and removed from the gap pile.

| ID | What it really is |
|:--|:--|
| **G-0121** | Legal workflows and verb composition aren't pinned mechanically. Body explicitly defers to "a milestone, not part of this gap" — this *is* a major epic about a missing spec layer. |
| **G-0078** | No priority field on entities. Kernel-wide enum + verb + render + status. Clear epic. |
| **G-0212** | Data-loss audit for verb composition. Title literally contains "future epic." |
| **G-0117** | aiwf render html: SPA instead of N files. ADR + JS dependency + permalink-shape change + cross-ref sweep. Epic. |
| **G-0227** | Layering & cohesion refactor (cliutil split + 4 Options-struct migrations + policy). Multi-week epic. *Filed by me today as a single gap; should have been an epic.* |
| **G-0073** | depends_on restricted to milestone→milestone; cross-kind blocking via body prose. Schema + FSM + render + check. Epic. |
| **G-0092** | No documented hierarchy of doc authority across docs/. Three-layer fix (table + per-tree markers + check rule). Epic. |
| **G-0113** | Rendered HTML site has no publish path. CI workflow + Pages + cadence decision. Epic. |

Honorable mentions (not promoted but epic-flavoured): **G-0022** (provenance model extension surface — six speculative extensions in one parking-lot gap; should be split or epic-ified).

## Hidden mini-epics — clusters of gaps that are really one thing

The most important finding: **what looks like 74 gaps is actually closer to 6 themed clusters + ~25 standalone items.** Each cluster is one mini-epic with multiple symptoms filed as separate gaps.

### Cluster 1: BranchOracle hardening (13 gaps)

`G-0197, G-0198, G-0201, G-0203, G-0204, G-0205, G-0206, G-0207, G-0211, G-0213, G-0215, G-0224, G-0225`

All children of E-0030's M-0158 / M-0161 surface. The cluster's existence is itself the evidence for **G-0222** (conformance-suites-at-seams) — one chokepoint there would have caught most of these in one pass.

Recommended: one mini-epic *"BranchOracle conformance + edge-case hardening."* Members become milestone-scoped ACs or close as superseded.

### Cluster 2: Statusline hardening (4 gaps)

`G-0183, G-0187, G-0188, G-0189`

All small, all converge on the same surface. Recommended: one statusline-hardening milestone.

### Cluster 3: Wrap-ritual chokepoints (4 gaps)

`G-0179, G-0218, G-0219, G-0220`

All about *"wrap-time should mechanically enforce X."* Recommended: one wrap-hardening milestone closes all four.

### Cluster 4: Scorecard follow-ups (9 gaps, all filed 2026-06-05)

`G-0227, G-0228, G-0229, G-0230, G-0231, G-0232, G-0233, G-0234, G-0235`

**Honest call:** I filed these today from the 2026-06-04 health scorecard. Each is real, but they should land as **1–2 epics** (e.g. *"Kernel hygiene: types + drift-prevention policies"* and *"Refactor: cliutil split + Options-struct + naming polish"*), not 9 parallel gaps. The triage's blunt word for the cluster was "noise" — not because the items aren't real, but because parallel-filing inflated the apparent count of independent concerns.

Recommended: fold into 1–2 epics; cancel the 9 gaps as superseded; carry the work as milestones.

### Cluster 5: Ritual-skill drift (3 gaps)

`G-0175, G-0219, G-0224`

All about skill-content correctness against verb-registry truth. Recommended: one milestone.

### Cluster 6: Legacy/PoC framing sweep (3 gaps)

`G-0074, G-0075, G-0077`

All three are WF candidates per the list above. Not really a cluster to address — a cluster to close.

## The concrete sweep that drops 74 → ~25

In order of leverage:

1. **Close the 10 WF candidates** with one-line rationales pulled from their bodies (~30 min, 10 commits).
2. **Promote the 8 T4 epic-shaped items** to actual epics. The gaps become superseded; the epics become tracked in `aiwf status`.
3. **Fold the 5 mini-epic clusters** (BranchOracle, Statusline, Wrap-ritual, Scorecard, Ritual-skill) into 5 epics. Member gaps either become milestone-scoped ACs or close as "tracked under E-NNNN."
4. **Live with the ~25 remaining standalone gaps.** Most are T2/T3 backlog where "address when in the file" is a defensible policy.

After the sweep:
- **Real T0/T1 work** in motion as milestones under the appropriate epics (~4 + ~9 = 13 items)
- **Cluster work** carried as 5 themed epics
- **Standing backlog** of ~25 standalone T2/T3 gaps — sane size to look at without anxiety

## Honest reflection on filing velocity

The discipline that filled the pile is genuinely good. Gap-filing is how the kernel learns from itself — each one captures a moment of friction or insight that would otherwise vanish. The 2026-06-04 health scorecard alone surfaced 43 atomic items that were quietly true about the codebase; without the audit they would not be findable now.

What's missing is a parallel **gap closing / triage cadence**. Filing is unbounded; closing is bounded by attention. Without a periodic sweep like this one, the pile only grows. The size of the open list is then a signal of *audit thoroughness*, not of *kernel decay* — but it lands on the maintainer's eye as the latter.

A periodic gap-triage ritual (this doc formalized as a quarterly or per-epic-wrap exercise) is the missing piece. The triage is cheap once the practice exists; the pile becomes self-bounding when WF/promote/fold decisions happen alongside filing decisions.

## Specific self-critique

Of the 12 gaps filed today (2026-06-05):

- **G-0221, G-0222, G-0223, ADR-0017** — well-shaped; specific bugs/decisions with concrete bodies. Keep.
- **G-0227–G-0235** — over-filed. The 9 cluster gaps came from the health scorecard's "recommended moves" per principle. Each is real, but filing them as 9 parallel concerns produced apparent fragmentation. They should fold into 1–2 epics retrospectively.

The right shape today would have been: file the 3 scorecard-named gaps (atomic, conformance, slog), and propose 1–2 epics carrying the rest as planned milestones. Instead I filed 9 gaps after explicitly considering the alternative and choosing wrong.

This is the kind of self-correction that periodic triage catches. Filed, observed within 24 hours, called out, fixable.

## Methodology

This triage was produced by:

1. Running `aiwf list --kind gap --status open` to enumerate (74 entries).
2. A general-purpose subagent reading each gap's body, classifying into one of the tiers above, flagging WF / DUP / epic-shaped, and grouping by area.
3. The agent returned a structured markdown table + an honest read of the pile.
4. Synthesis in this document.

Subagent token cost: ~140k. Read coverage: all 74 open gap bodies. Today's date: 2026-06-05.

## How this section ages

When a future triage runs, the comparison point is:

- Which of these 74 closed (as `addressed`, `wontfix`, or `superseded`)?
- Which T0/T1 items still appear in the pile? (If so: why didn't they land?)
- Did any of the WF candidates regrow? (If so: WF was the wrong call.)
- Are the cluster-epics in progress or still parked?
- What new items showed up — and what's the closing/filing ratio since last triage?
