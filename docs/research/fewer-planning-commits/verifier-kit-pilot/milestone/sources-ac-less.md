# Milestones without acceptance criteria: sources and history

Collected 2026-11-16 for the milestone pilot's questions Q3 and Q4 (`spec/milestone/answers.md`).
Part 1 is measured from git history. Part 2 is every written claim a sweep of the repo's prose
found; quotes are verbatim. Spot-checked: D-0040:26, transition.go:202-206, E-0033 epic.md:18,
M-0123:109, G-0575:30-45, and D-0039 containing neither "legitimate" nor "end state".

## Part 1. What happened (git history and the tree, main at 2de605437)

Commands: `ms_acs.py` (reads every `work/epics/**/M-*.md` frontmatter) and
`m_hist.py` (replays every commit whose `aiwf-entity` trailer names a milestone or
AC, from `git log --reverse`, in order).

- 363 milestones: 319 done, 42 cancelled, 2 draft.
- 74 have no ACs at all:
  - 48 are M-0001 to M-0048, all done. They predate ACs: M-0014 to M-0023 built the AC feature.
  - 26 are cancelled milestones that were cancelled while still `draft`, before any AC was
    added.
  - No milestone after M-0048 has reached `done` with zero ACs.
- 31 milestones were promoted to `in_progress` with zero ACs (2026-05-08 to 2026-07-06). In every
  sample checked (M-0084, M-0124, M-0236) the ACs were added within a minute or two after the
  start. That was the ritual then: plan-milestones said "no front-loaded ACs", and
  start-milestone created them. D-0039 (2026-07-19) and M-0275 (ACs created at plan time) ended
  it. None since.
- `--force` starts of a zero-AC milestone: none. Every forced `in_progress` promote (7, on 5
  milestones: M-0072 twice, M-0073, M-0074 twice, M-0078, M-0106) was a `done → in_progress`
  reopening, made because the work turned out incomplete (reasons recorded in `aiwf-force`).
- 16 milestones have every AC `deferred` or `cancelled`. All 16 are cancelled milestones whose
  ACs were cancelled or deferred minutes before the milestone cancel, which D-0004 requires.
  None was ever started in that state.
- AC status changes made after the milestone was `done` or `cancelled`: none.
- `done` milestones with an `open` AC in the tree: none.

## Part 2. Written claims

(The sweep's report, unchanged.)

### (a) May a milestone have zero ACs, or must it have at least one?

ACs optional:

1. `docs/design/design-decisions.md:152` — "**What's not a kernel rule.** No "milestone must have ≥1 AC" — ACs remain optional." Current design doc; line from 2026-05-01, unchanged since.
2. `docs/archive/pocv3/acs-and-tdd-plan.md:383` — "| Forcing a milestone to have ≥1 AC | ACs remain optional. |" Archived proposal.
3. `docs/design/legal-workflows-first-principles.md:425` (R-FP-0166) — "A milestone is **not** required to have ≥1 AC. ACs are optional." / "The kernel guards the AC outcome, not its existence." Pass B, 2026-05-18.
4. `docs/design/legal-workflows-audit.md:369` (R-AUDIT-0174), `:573` (R-RULE-027) — "NOT a kernel rule: "milestone must have ≥1 AC" — ACs remain optional". Pass A; banner says not the source of truth.
5. `internal/workflows/spec/antirules.go:23-24` (ANTI-0001) — "A milestone is NOT required to have >=1 AC. ACs are optional." Go spec anti-rule, 2026-05-19.
6. `internal/workflows/spec/spec.go:165` — "A milestone is NOT required to have ≥1 AC."
7. `internal/workflows/spec/evaluate.go:262-264` — "A milestone with zero ACs satisfies any all-ACs predicate vacuously (the empty universal); whether that matches the rule's intent is the rule-author's concern, not the evaluator's."
8. `internal/cellcoverage/fixture.go:163-165` — "The Milestone.done path takes the zero-AC vacuous-satisfaction route by default (spec AntiRule: a milestone is not required to have >= 1 AC)."
9. M-0123 (done) `:72` — "The milestone is `active`-with-zero-ACs through phase 1. That's intentional … The kernel does not forbid this; `acs-tdd-audit` only fires on ACs that exist with missing `tdd_phase`." (May 2026)
10. `docs/explorations/07-tdd-architecture-proposal.md:634` — "**Deferred without commitment to a stance.** The current implementation tolerates empty `acs[]` on milestones; doc-sweep and refactor milestones use this. Whether to add a strict-mode requirement that every milestone has at least one AC is a future call; not on this synthesis's path." (last edited 2026-07-21)

Zero ACs guarded (block start, warn at draft and done):

11. G-0334 (addressed by M-0268) `:11-16` — "A milestone can traverse its entire lifecycle — `draft → in_progress → done` — carrying zero acceptance criteria, tripping no finding. … So the AC-evidence discipline is vacuous for a zero-AC milestone: it can start *and* finish with nothing to work toward and nothing to substantiate completion."
12. G-0334 `:44-46`, `:51-56` — "**`draft → in_progress` on a zero-AC milestone is refused** by the promote verb (a hard block, not a warning)…" / "**`done` on a zero-AC milestone is not refused.** Instead, a new warning-severity check-time finding…"
13. G-0334 `:36-38` — "The `aiwfx-start-milestone` ritual guards AC presence advisorily — which, per "framework correctness must not depend on the LLM's behavior," is not a guarantee."
14. D-0039 (accepted, 2026-07-19) `:27-28` — "1. **`draft → in_progress` on a zero-AC milestone is refused** by the promote verb, with the standard `--force --reason "..."` override. 2. **`done` on a zero-AC milestone is not refused** — instead, a new warning-severity check-time finding surfaces it…"
15. D-0039 `:37` — "By the time a milestone reaches `done`, the meaningful decision was already made at the start-time gate (point 1): either the milestone was force-started with a recorded reason, or it was already `in_progress` before this rule shipped (grandfathered)."
16. D-0040 (accepted) `:26` — "D-0039 point 2 already establishes that a permanently zero-AC milestone is a legitimate end state, so a soft, force-overridable stance is correct there."
17. M-0268 (done) `:44` — "`aiwf promote M-NNN in_progress` on a milestone whose `acs[]` is empty exits non-zero… `--force --reason "..."` succeeds despite zero ACs."; `:52` "`aiwf promote M-NNN done` itself is **not** refused for a zero-AC milestone"; `:61` "it must never become a verb-time refusal at `done`"; `:68` "(D-0039 point 2: a permanently zero-AC milestone is a legitimate end state)".
18. E-0068 (done) `epic.md:27` — "Verb-time refusal on `draft → in_progress` when a milestone has zero ACs, with `--force --reason "..."` override"; `epic.md:42` "…the `done`-transition guard for a zero-AC milestone is check-time and warning-severity only — it must not become a second verb-time hard block."; `wrap.md:16` "a milestone can no longer start `in_progress` with zero ACs".
19. `internal/verb/promote.go:551-556` — "…would start a milestone with no acceptance criteria at all (D-0039 point 1 / M-0268/AC-1). Zero ACs means there is no contract yet for what "done" means; the operator either writes at least one AC first or overrides with --force."
20. `internal/verb/promote.go:561-564` — "D-0039 point 2 explicitly permits a milestone to reach done with zero ACs (only a warning fires), so "permanently AC-less" is a legitimate end state, not an inconsistency force would be papering over."
21. `internal/check/acs.go:437-439` — "a milestone is allowed to reach `done` permanently AC-less, but the warning keeps the state visible rather than silent."
22. `internal/skills/embedded/aiwf-check/SKILL.md:154` — "`milestone-done-zero-acs` | … advisory, not a refusal; a permanently AC-less milestone is a legitimate end state."
23. `aiwf-check/SKILL.md:155` — "subcode `zero-acs` when `acs:` is empty … Advisory, not a refusal: `draft` is a legitimate mid-planning state…"
24. `internal/check/hint.go:179` — "advisory only — a done milestone with no acceptance criteria is a legitimate end state; add one via `aiwf add ac <milestone-id> --title \"...\"` first if this was unintentional"
25. `internal/check/hint.go:181` — "a warning only — a draft milestone is legitimately mid-planning, so no action is needed if you are still scoping it"
26. `CHANGELOG.md:3129-3137` (0.28.0, 2026-07-22) — "`aiwf promote M-NNNN in_progress` now refuses a milestone with zero acceptance criteria … for the zero-AC case, accept `--force --reason "..."` to override … `milestone-done-zero-acs` (warning) for a milestone reaching `done` with no ACs at all"
27. `CHANGELOG.md:3087-3090` (0.29.0, 2026-07-24) — "warns on a `draft` milestone with zero acceptance criteria (`zero-acs`) … so an incomplete contract cannot quietly reach main; ACs are now created and body-filled at plan time."
28. D-0047 (accepted, 2026-07-22) `:56-61` — "surfaces a `draft` milestone with zero ACs or empty AC bodies … warn, don't block, per D-0039's own block-at-transition/warn-at-rest split, since `draft` is a legitimate mid-planning state"
29. G-0440 (addressed) `:20-22` — "a `draft` milestone visible on main can carry zero AC entities … invisible to any reader"
30. M-0275 (done) `:54-57` — "A new warning-severity check-time finding fires for a non-archived `draft` milestone whose `acs[]` is empty. Warning, not error — `draft` is a legitimate mid-planning state…"
31. `aiwfx-plan-milestones/SKILL.md:70` — "Creating and body-filling the ACs here … is what keeps a milestone from ever landing on main with zero ACs or empty AC bodies"
32. `aiwfx-start-milestone/SKILL.md:24` — "**ACs are expected to already exist.** … **Recovery fallback only** — a hand-written spec whose `acs[]` is empty: add them now and fill each body before proceeding"
33. `docs/initiatives/tdd-cycle-subagent-boundaries.md:115-118` — "a milestone can start and finish with zero ACs, no guard." (2026-07-16)

Audits on the conflict:

34. `docs/audits/gap-truth-audit-evidence.md:334-341` — "The kernel now *does* police AC existence, under different names. … ANTI-0001's Statement ("ACs are optional") is therefore narrower than it reads" (2026-08-23)
35. `docs/audits/entity-truth-audit.md:160`, `:1436`, `:1443` (2026-08-19) — "`draft → in_progress` needs ≥1 AC *with body prose*"; "the milestone must have at least one AC"; "since in_progress now requires at least one AC"
36. G-0161 (wontfix 2026-11-15) `:23` — "**ANTI-0001** (no `≥1 AC` requirement): create a milestone with 0 ACs, confirm `aiwf check` produces no `acs-required` finding."
37. `docs/reference/workflow-legality.md:75` (generated from the spec) — "| milestone | draft | promote | in\_progress | Legal |  | none | false |  |"

### (b) What kinds of work need a milestone but no ACs

1. G-0334 `:44-47` — "with the standard `--force --reason "..."` override for genuinely AC-less milestones (pure coordination / exploratory work)."
2. D-0039 `:33` — "`--force --reason` keeps genuinely AC-less (coordination-only, exploratory) milestones unblocked, at the cost of one flag and one recorded sentence — friction that lands only on the case where friction is the point, not on the common path of a milestone that was always going to carry ACs."
3. `docs/explorations/07-…:634` — "doc-sweep and refactor milestones use this."
4. M-0123 `:63`, `:65` — "promote `active` at start of phase 1 (no ACs yet…)"; "**Phase 1 — Reconciliation walk + schema design.** Research and authoring; not TDD-shaped."
5. D-0040 `:26`, M-0268 `:68` — "a broken promise, not a legitimate minimalist choice" (contrasting an empty AC body with having no ACs).
6. Records that untestable work still carries ACs: `aiwf-guidance.md:42-43` "**Promote an AC to met only with mechanical evidence — unless it claims an observation.**"; `aiwfx-start-milestone/SKILL.md:20` "Confirm every AC is concrete, and either testable or observational"; `.guidance/project/testing.md:3-5` "## Evidence for a doc-shaped acceptance criterion …"
7. Adjacent (about `tdd: none`, not AC count): `docs/explorations/07-…:243` "`none` — No formal TDD. … Used for documentation sweeps, exploratory work, refactors with no observable behavior change."; G-0055 `:47` "(e.g., "docs-only milestone", "config refresh, no behavior change")".

No prose names spike, release, wrap or migration milestones as AC-less, and none argues against coordination or exploratory AC-less milestones.

### (c) Do cancelled or deferred ACs count for completeness guards?

The zero-AC guards count entries in `acs[]`:

1. M-0268 `:44`, `:52` — "on a milestone whose `acs[]` is empty"; "a non-archived milestone has `status: done` and an empty `acs[]`"
2. `internal/check/acs.go:356`, `:436` — "subcode zero-acs when acs[] is empty"; "has status: done and an empty acs[]"
3. `aiwf-check/SKILL.md:154` — "with an empty `acs:` list"
4. D-0040 `:26` — "AC-1/AC-3 guard whether a milestone has any acceptance criteria at all"
5. `design-decisions.md:131`, `:134` — "Cancelled ACs stay in `acs[]` at their original position (status flip, not deletion)"; "(cancelled entries count toward position)"
6. R-FP-0062 (`:183`) — "Cancelled ACs stay in `acs[]` at their original position."; ANTI-0004 (`antirules.go:41`) "The position-stable position-in-acs[] retains the cancelled AC."

Terminal ACs are "off the contract" for the body-completeness guards:

7. `internal/entity/transition.go:208-210` — "Both terminals are removal-class: `deferred` and `cancelled` each mean "off the milestone's contract", and neither claims the criterion succeeded."
8. Repeated in `acs.go:485-487`, `check/entity_body.go:231-233`, `verb/ac.go:289-292`.
9. `promote.go:584-586` — "A terminal AC (entity.IsTerminalACStatus) is skipped: it is withdrawn from the milestone's contract, so it owes no prose"
10. `aiwf-check/SKILL.md:89` — "an AC still on its contract (one not at a terminal status: neither `deferred` nor `cancelled`)"
11. `CHANGELOG.md:622-635` (G-0464, 0.36.1) — "a deferred acceptance criterion is off the milestone's contract … `met` is not terminal there — a met criterion can still be rescoped — so its body is still held to the bar."
12. `CHANGELOG.md:377-382` (G-0695, 0.39.0) — "no longer refuses because a cancelled or deferred acceptance criterion has an empty body … under a criterion already withdrawn from the milestone"
13. G-0464 `:14-16` — "A `deferred` AC is equally out of the milestone's contract"
14. G-0695 `:47-49` — "the AC is terminal, so writing prose under it is work on a criterion already withdrawn."

Which AC statuses satisfy done or cancelled:

15. `design-decisions.md:144` — "`deferred` and `cancelled` are acceptable terminal AC states for a done milestone, with the body explanation as the documentation." (also `acs-and-tdd-plan.md:126`, `acs.go:341-343`, `acs.go:465-467`)
16. R-FP-0061 (`:182`) — "requires every AC to be in a terminal AC status (`met`, `deferred`, `cancelled`)"
17. `aiwfx-wrap-milestone/SKILL.md:21` — "confirm every AC's `status` is terminal (`met`, `deferred`, or `cancelled`)"
18. R-AUDIT-0081 (`legal-workflows-audit.md:198`) — "A milestone with `status: done` whose ACs include any non-terminal status fires `milestone-done-incomplete-acs`"
19. `transition.go:202` — "`met` is deliberately NOT terminal, unlike a kind's `done`." (also `.guidance/project/verb-design.md:41`)

What deferred and cancelled mean:

20. `acs-and-tdd-plan.md:105` (archived) — "`deferred` = "still want this, not now" (link to the receiving milestone in body); `cancelled` = "no longer in scope" (link to a `D-NNN` decision in body)."
21. `docs/explorations/07-…:166`, `:168` — "deferred (work moved to a gap or future milestone)"; "cancelled (no longer applies)"
22. `aiwfx-wrap-milestone/SKILL.md:87` — "Deferred ACs (status `deferred`) get a one-line note pointing at the receiving milestone or gap."
23. `aiwfx-wrap-milestone/SKILL.md:90` — "The kernel only guards the structural state (`status: cancelled`, position-stable in `acs[]`); the why is the human's narrative."

Progress rollups and the all-cancelled case:

24. `internal/cli/status/status.go:133-135` — "`Total` includes cancelled entries … `InScope` excludes them, so that's the denominator the renderers use"
25. `status.go:171-176` — "Returns "" when there are no ACs … "ACs all cancelled" — every AC was cancelled (in-scope = 0)"
26. `status.go:1466-1467` — "Cancelled-only milestones get no badge (the design's "all cancelled" case isn't useful at a glance)."
27. `htmlrender/pagedata.go:121-122` — "ACTotal is the sum of non-cancelled AC counts."
28. Same `met / (total - cancelled)` rollup in `README.md:401`, `aiwf-render/SKILL.md:14`, `docs/archive/pocv3/governance-html-plan.md:344` "AC met-ratio (`met / (total - cancelled)`) is the only milestone-level progress number."
29. G-0696 (open) `:9-11` — "the in-scope denominator as every criterion minus the cancelled ones, so a `deferred` criterion counts toward progress permanently"
30. G-0696 `:31-32` — "Excluding deferred from the denominator makes an all-deferred milestone reach `InScope` 0 too, where the wording would be false."
31. `docs/explorations/05-policy-model-design.md:178` (illustrative) — "Every milestone in `done` MUST have all acceptance criteria met."

No prose says whether a milestone whose ACs are all cancelled or deferred passes the zero-AC start refusal or the done-time zero-AC warning.

### (d) May ACs change status after the milestone is done or cancelled?

No parent-status condition:

1. `design-decisions.md:137` — "`deferred` and `cancelled` are terminal; `met` may move to `deferred`/`cancelled` if scope changes after the fact." (also `acs-and-tdd-plan.md:101`, `entity/entity.go:162-164`, `spec/rules.go:1197`, `:1206`, `docs/explorations/07-…:170`)
2. R-FP-0049 (`:158`) — "This admits the practical case where an AC was satisfied, then later re-evaluated as no-longer-applicable."
3. R-FP-0050 (`:159`) — "the model deliberately allows post-met scope changes."
4. `legal-workflows-audit.md:101-102`, `:569` — "(scope-change after the fact)"
5. `docs/reference/workflow-legality.md:162-163` (generated) — "| ac | met | promote | deferred | Legal |  |…"; "| ac | met | cancel | cancelled | Legal |  |…"

Rescoping tied to the milestone's run:

6. `transition.go:202-206` — "An AC is a claim inside a contract that can still be rescoped, so a met criterion may legitimately be descoped while its parent milestone runs; an epic is a closed unit of work."
7. M-0281 (done) `:282-284`, `:465` — "a met criterion can legitimately be descoped while its parent milestone runs"; "an AC is a claim inside a contract that can still be rescoped"
8. `design-decisions.md:129` — "their lifecycle is bounded by it."

ACs cannot outlive the milestone:

9. R-FP-0064 (`:185`) — "An AC's lifecycle is bounded by its parent milestone's lifecycle: cancelling the milestone implicitly cancels all of its ACs (or terminates them otherwise — the model doesn't pin the exact mechanism, but ACs cannot outlive the milestone). … The milestone's terminality is the AC's terminality." (conventional; Q6 left the mechanism open)
10. D-0004 (accepted) `:15` — "`aiwf cancel M-NNNN` refuses if any AC has status `open`"
11. D-0004 `:21` — "ACs cannot outlive the milestone; the operator commits to a disposition for each before the milestone terminalizes."
12. `entity-truth-audit.md:1267-1270` — flags R-FP-0064's cascade as a false claim.
13. E-0033 (done) `epic.md:18` — "**Cross-verb sequence legality** — e.g., "you can't promote an AC to met after the parent milestone is done." Currently implicit, never formally specified."
14. M-0123 `:109` — "Cross-verb sequence legality manifests as predicates on the later verb (e.g., "AC promote requires `parent.status != entity.StatusDone`")."
15. M-0268 `:67` — "a *terminal* status must never be reached while a child/AC is still non-terminal."

Adding ACs to a terminal milestone:

16. R-FP-0067 (`:193`) — "The verb refuses if the parent milestone does not exist or has a terminal status. … Adding an AC to a `done` milestone would create the contradiction R-FP-0061 polices."
17. R-AUDIT-0108 (`:243`), R-RULE-094 (`:665`) — "Parent milestone must be non-terminal (not `done`/`cancelled`)"
18. `entity-truth-audit.md:1923` — "whether `aiwf add ac` against a `done` or `cancelled` milestone is refused in practice … even though the verb itself carries no status guard." (left unsettled)

Remedies that change AC status after the milestone is terminal:

19. `design-decisions.md:144` — "a milestone that became `done` via `--force --reason` while ACs were still open keeps surfacing the inconsistency until the ACs reach a terminal state." (also `acs.go:345-349`, `aiwf-promote/SKILL.md:73`, `aiwfx-wrap-milestone/SKILL.md:274`)
20. `hint.go:177` (milestone-done-incomplete-acs) — "promote each open AC via `aiwf promote <milestone-id>/AC-N <met|deferred|cancelled>`"
21. `hint.go:180` (milestone-cancelled-incomplete-acs) — same.
22. G-0575 (open) `:10-11`, `:36-42` — "A clean merge … can close a milestone while adding criteria to it, leaving a `done` milestone with open acceptance criteria." Its table records the hint's first remedy "refused by the projection guard", and "`promote <m> in_progress --force` … **the only exit**".
23. G-0574 (open) `:25-27` — "From a `done` milestone with two open criteria, cancelling either one is refused, because the post-state message names a different remaining set."
24. M-0284 `:550-553`, M-0282 `:353-356` — decline `met → deferred/cancelled` as a correction, since "both meaning "off the contract" … would assert work remains where none does".

No prose explicitly permits changing an AC's status on a done or cancelled milestone as policy; permission is implied only by the unqualified "after the fact" wording (1-5) and the remedies (19-21).

### Contradictions

1. ≥1 AC: design-decisions.md:152, acs-and-tdd-plan.md:383, R-FP-0166, R-AUDIT-0174/R-RULE-027, antirules.go:23 (optional) vs D-0039:27, M-0268:44, promote.go:551-556, CHANGELOG.md:3129-3134, entity-truth-audit.md:1436, aiwfx-plan-milestones/SKILL.md:70 (start refused unless forced). gap-truth-audit-evidence.md:341 already flags ANTI-0001; the entity-truth audit's findings on design-decisions.md don't include line 152.
2. explorations/07:634 ("tolerates empty acs[] … a future call") vs D-0039:27.
3. M-0123:72 ("The kernel does not forbid this") vs D-0039:27 (M-0123 predates D-0039).
4. Attribution: D-0040:26, M-0268:68, promote.go:561-564 credit "permanently zero-AC milestone is a legitimate end state" to D-0039 point 2; D-0039 contains neither "legitimate" nor "end state", and its point-2 reasoning (:37) frames done-with-zero-ACs as the force-started or grandfathered case. The same wording, uncited, is in acs.go:438, aiwf-check/SKILL.md:154, hint.go:179.
5. workflow-legality.md:75 lists draft → in_progress Legal with no condition; promote refuses a zero-AC start.
6. Is met terminal? R-FP-0061:182, aiwfx-wrap-milestone/SKILL.md:21 (yes) vs transition.go:202, design-decisions.md:137, verb-design.md:41, CHANGELOG.md:633-635 (no).
7. Rollup denominator: status.go:133-135, README.md:401, governance-html-plan.md:344, aiwf-render/SKILL.md:14 count deferred as in scope vs transition.go:208-210, G-0464, G-0696 (off the contract).
8. ACs after the milestone ends: unqualified "after the fact" (design-decisions.md:137, R-FP-0049/0050, entity.go:162-164, rules.go:1197/1206) vs transition.go:204-205, M-0281:283-284 ("while its parent milestone runs"), R-FP-0064, D-0004:21 ("cannot outlive"), E-0033 epic.md:18 and M-0123:109 (examples forbidding an AC promote after done).
9. Remedies vs "cannot outlive": hint.go:177/180, design-decisions.md:144, aiwf-promote/SKILL.md:73, acs.go:345-349 tell the operator to promote ACs after done/cancelled, against D-0004:21 and R-FP-0064; G-0575:36-40 records those remedies refused or converging in the merge case.
10. Adding an AC to a done milestone: hint.go:179 suggests it; R-FP-0067, R-AUDIT-0108, R-RULE-094 say add ac refuses a terminal parent; entity-truth-audit.md:1923 left it unsettled.
11. Cancel cascade: R-FP-0064 (cascades) vs D-0004 (refuse with listing); already flagged by entity-truth-audit.md:1267.
12. explorations/05:178 ("all acceptance criteria met") vs design-decisions.md:144 (illustrative only).

### Entities that discuss this

G-0334 (addressed), G-0216 (addressed), G-0286 (addressed), G-0440 (addressed), G-0161 (wontfix), G-0335 (addressed), G-0464 (addressed), G-0695 (addressed), G-0696 (open), G-0574 (open), G-0575 (open), G-0055 (addressed); D-0004, D-0039, D-0040, D-0047 (accepted); E-0033, E-0068, E-0070 (done); M-0123, M-0268, M-0275, M-0281, M-0282, M-0284, M-0199 (done).
