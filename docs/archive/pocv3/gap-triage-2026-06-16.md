# Gap Triage — 2026-06-16

Pairs with [`health-scorecard-2026-06-16.md`](health-scorecard-2026-06-16.md). Live tree at triage time: **71 open gaps**, 4 proposed ADRs (ADR-0001, ADR-0009, ADR-0015, ADR-0017), 10 proposed `D-NNNN` decisions. The tree is under active concurrent work, so this is a point-in-time snapshot — navigate by `aiwf list`, not by this doc, once it ages.

## How to read this against the scorecard

The fresh scorecard returned **12 Strong / 11 Weak / 1 Missing** — much harsher than 2026-06-04's 20 Strong / 4 Weak. Per the scorecard's own delta, only **one** change is a real code improvement (C3 atomic-writes Weak→Strong via G-0221). The A3/C1/C2/C4 downgrades are the adversarial verifiers reclassifying "the property holds by inspection but has *no mechanical chokepoint*" as Weak — a stricter rubric, not regressions. Read the Weak set as **"where the chokepoints aren't yet,"** not "where the code broke."

## 1. Scorecard verdict → backlog cross-reference

Every non-Strong verdict, mapped to the gap that already tracks it (or flagged as uncovered). **8 of 12 are already tracked**; 3 areas are uncovered candidates; 1 is a partial.

| Verdict | Tracking gap | Coverage |
|:--|:--|:--|
| A3 Layered (Weak) | **G-0227** ("+ policy" item) | partial — confirm the policy item is the import-direction/depguard test, not just the cliutil split |
| B2 Schemas (Weak) | **G-0229** (schema-versioning hygiene at config/manifest/recipe) | tracked |
| B3 Pre/post (Weak) | — | **uncovered → candidate A** |
| C1 Single source of truth (Weak) | — | **uncovered → candidate B** (verified real) |
| C2 Idempotence (Weak) | **G-0230** (NoOp on same-state + dry-run) | tracked |
| C4 Versioned schemas (Weak) | **G-0229** (entity frontmatter version field) | tracked (partial) |
| D2 Equivalence at seams (Weak) | **G-0222** (conformance suites at seams) | tracked |
| D3 Branch coverage (**Missing**) | **G-0067** (mechanical RED-first / coverage gate) | tracked — the lone Missing, top priority |
| D4 Altitude (Weak) | **G-0233** (DOM-structural htmlrender, e2e widening) | tracked |
| E1 Structured logs (Weak) | **G-0223** + **ADR-0017** | tracked; decision pending ratification |
| F1 Names (Weak) | overlaps candidate A; **G-0235** (partial) | mostly uncovered → fold into candidate A |
| F3 Decision records (Weak) | — | **uncovered → candidate C** |

**Takeaway:** the audit did *not* surface a pile of new work. The Weak set is overwhelmingly the **2026-06-04 cluster gaps (G-0227…G-0235) still open** plus G-0067/G-0222/G-0223. The audit's value this round is confirmation that those clusters are the right durable backlog, and that C3 closed.

## 2. Uncovered findings → candidate gaps (NOT filed)

Three genuinely-uncovered findings. Listed as candidates only — **not filed**, per the lesson from 2026-06-05's 9-cluster over-filing. Verify each at filing time; decide whether each is its own gap or folds into an existing cluster.

- **Candidate A — `no_silent_fallback` bare-identifier blind spot [B3, F1].** The policy's `exprTypeIdent` doesn't resolve bare-identifier `switch` tags, so `initialStatus` (`internal/verb/common.go:47`) and `BodyTemplate` (`internal/entity/serialize.go:177`) escape the default-clause requirement; the policy's doc comment also overstates what it checks (default-existence only). Fix is small: resolve bare idents or add explicit `default:` clauses, and reconcile the doc. *Likely folds into G-0235 (guardrail-policy sweep) rather than standing alone.*
- **Candidate B — version-source split [C1] (verified real).** `aiwf version`/`doctor` use `ResolvedVersion()` (`internal/cli/root.go:83-88`, prefers the ldflags stamp); the JSON envelope uses `version.Current()` (buildinfo). For a `make install`-stamped binary these report **different strings**. `PolicyEnvelopeVersionSource` pins the envelope but not the human print site. This is *not* the already-fixed `"dev"`-fallback bug — it's the residual ldflags-stamped case. Fix: route both through one resolver, widen the policy to pin the human site. *Adjacent to G-0232 (envelope) but distinct; narrow standalone candidate.*
- **Candidate C — decision-record drift [F3].** ADR-0016 carries a live frontmatter-vs-body `status:` drift (a recurrence of the ADR-0003 wart noted at 06-04), and decision-to-decision supersession is prose-only / unmechanized. The drift is a micro-fix (one `aiwf` retitle/edit); the supersession-mechanization is the gap-worthy part. *Could fold into G-0235; the ADR-0016 drift should just be fixed in-context.*

## 3. Tiered open-gap landscape (71)

Leverage-tiered, not severity-tiered. T1 is the audit backlog; T2 is actionable correctness; T3–T5 are clusters that want a mini-epic each; T6 defers.

### T1 — Audit-backlog cluster (scorecard-mapped, highest leverage) — 11

| Gap | Maps to | Note |
|:--|:--|:--|
| G-0067 | D3 (Missing) | diff-scoped branch-coverage gate + policy — **the lone Missing; #1 priority** |
| G-0223 | E1 | opt-in slog; gated on ratifying **ADR-0017** |
| G-0222 | D2 | PageDataResolver + BranchOracle conformance suites |
| G-0227 | A3, A1 | cliutil split + Options-struct + **layering import-direction policy** |
| G-0228 | B1 | typed `Status` / `FindingCode` / `codes.Code` coverage |
| G-0229 | B2, C4 | strict-decode config/manifest + schema-version field |
| G-0230 | C2 | NoOp on same-state + dry-run on wide-blast verbs |
| G-0232 | G3, B2 | `correlation_id` wiring + mutating-verb metadata |
| G-0233 | D4, D1 | DOM-structural htmlrender tests + fault harness + e2e widening |
| G-0234 | E4 | allowed-set-inline / typed-Coded / remediation polish |
| G-0235 | conventions | CLAUDE.md conventions sweep + guardrail policies (cited-ids, no-time-now) |

### T2 — Correctness bugs & small fixes (actionable, low-risk) — 10

| Gap | What |
|:--|:--|
| G-0247 | `add ac` duplicate AC body headings; check collapses them (confirmed bug, two-sided fix) |
| G-0246 | ADRs lack a general `relates_to` cross-reference field |
| G-0168 | no post-create mutation verbs for set-at-create frontmatter fields |
| G-0216 | empty AC body blocks draft→in_progress promote |
| G-0217 | `aiwf status` WRAP PENDING conflates wrap-ritual and trunk-merge pending |
| G-0249 | add `milestone-active-under-nonactive-epic` kernel finding |
| G-0248 | `aiwfx-plan-milestones` next-step skips `aiwfx-start-epic` for proposed epics |
| G-0199 | finding hints must name the exact remediation command |
| G-0198 | branchparse regex accepts prefix-id mismatch |
| G-0090 | M-0079 AC-8 drift-check has untested branches |

### T3 — BranchOracle / kernel-mechanization cluster (mini-epic) — 16

G-0121, G-0160, G-0161, G-0166, G-0197, G-0200, G-0201, G-0202, G-0203, G-0204, G-0205, G-0206, G-0207, G-0211, G-0213, G-0215. *BranchOracle robustness (shallow clones, force-push, renames, detached HEAD), cell-coverage fixtures, FSM-edge coverage, verb-composition E2E. Several touch the same D2-seam G-0222 wants conformance over.*

### T4 — Ritual / workflow discipline — 10

G-0060, G-0099, G-0111, G-0116, G-0175, G-0209, G-0219, G-0220, G-0224, G-0225. *Patch-ritual rules, worktree-isolation precondition, wrap-side ritual, SKILL.md structural pins, retired-code citations in skills.*

### T5 — Statusline cluster (one milestone) — 4

G-0183, G-0187, G-0188, G-0189. *Install path, e2e behavioral test, non-ritual-branch epics, stale-after-push.*

### T6 — Defer / YAGNI / gated / doc-housekeeping — 20

G-0022, G-0023, G-0068, G-0070, G-0073, G-0074, G-0075, G-0077, G-0078, G-0092, G-0104, G-0110, G-0113, G-0117, G-0140 (gated on **D-0005**), G-0157, G-0169, G-0178, G-0181, G-0212. *Speculative-until-forcing-function, doc-authority/`docs/pocv3` housekeeping (now partly overtaken by the shipped E-0040 / consumer-CLAUDE.md work), `--format=json` widening, future-epic-shaped.*

## 4. Velocity & how the pile is moving

Since the 2026-06-05 triage: **20 gaps closed, 16 filed** — net pile roughly flat (~71), but the *composition* improved: the cheap T0/T1 bugs (silent-green CI, acknowledge-illegal cluster, body-prose-id, FSM accepted-cancel, atomic writes, wrap-time lint, ritual drift) all closed, and an entire epic (E-0040, consumer-CLAUDE.md guidance) shipped to **v0.14.0**. The pile isn't growing because quality is decaying; it's flat because filing velocity ≈ closing velocity, and the remaining backlog is now concentrated in the audit clusters (T1) and the BranchOracle mini-epic (T3) rather than scattered one-off bugs.

## 5. Recommended next

1. **The T1 audit cluster is the durable backlog.** G-0227…G-0235 are not 9 parallel gaps to grind individually — they cluster into ~2 epics (a "mechanical-chokepoint hardening" epic covering A3 layering policy / C2 NoOp policy / B1 typed-codes / G-0235 guardrails, and a "test-architecture" epic covering D2 conformance / D3 coverage gate / D4 DOM-structural). Consider promoting them rather than wf-patching each.
2. **Highest single-gap leverage: G-0067** (D3, the lone Missing) — but it's blocked on **E-0016** declaring the TDD policy first. **E-0016 → G-0067** is the critical path for the worst scorecard verdict.
3. **Ratify ADR-0017** (conversation-only) to unblock G-0223 (E1), the next-worst verdict.
4. **Candidates A/B/C** from §2: decide fold-vs-file. Lean: A and C fold into G-0235; B is a narrow standalone worth filing; the ADR-0016 status drift is a fix-in-context micro, not a gap.

## How this ages

The durable record is the gaps, not this doc. Re-run the pairing (scorecard + triage) at the next real boundary — after E-0016 closes, or before a major release — not on a calendar. The comparison point next time: which T1 cluster gaps closed, did the candidates get filed/folded, and did any Strong verdict regress on real code (vs rubric drift).
