# `docs/pocv3/` triage

Per-file disposition for every file under `docs/pocv3/` at the time of M-0126. This table is the
contract M-0127 (Relocate) executes against verbatim — no re-classification during the sweep; a
wrong call here is a new gap, not a silent revision mid-sweep.

## Disposition semantics

- **relocate** — content is normative and currently true; moves to the live `docs/<subdir>/` path
  named in Target.
- **archive** — content is historical, done, or superseded; moves to `docs/archive/pocv3/<name>`
  (a sibling namespace to the existing `docs/archive/`, per the Open Question #1 resolution below).
- **supersede-with-entity** — forward-looking intent worth tracking; Target names the entity id
  that carries it forward. The source file is *also* archived to `docs/archive/pocv3/<name>` as
  background reading — this disposition answers "where does the intent go," not "where does the
  file go."
- **delete** — no row in this table uses this disposition. Nothing under `docs/pocv3/` cleared the
  explicit-justification bar for deletion; the forget-by-default default (archive) covers every
  ambiguous case.

Rationale for the Open Question and non-obvious calls lives in M-0126's "Triage rationale" section,
not repeated per-row here.

## Table

| File | Disposition | Target | Rationale |
|---|---|---|---|
| `docs/pocv3/architecture.md` | relocate | `docs/architecture.md` | Foundational, currently-accurate system reference; cross-referenced from CLAUDE.md and every other pocv3 doc. |
| `docs/pocv3/archive/gaps-pre-migration.md` | archive | `docs/archive/pocv3/gaps-pre-migration.md` | Already explicitly frozen pre-migration record; header says so. |
| `docs/pocv3/archive/poc-plan-pre-migration.md` | archive | `docs/archive/pocv3/poc-plan-pre-migration.md` | Already explicitly frozen session/iteration index. |
| `docs/pocv3/archive/rituals-plugin-plan.md` | archive | `docs/archive/pocv3/rituals-plugin-plan.md` | Already explicitly superseded by ADR-0014; first line says so. |
| `docs/pocv3/contracts.md` | archive | `docs/archive/pocv3/contracts.md` | Superseded design proposal — the shipped contract system (`internal/contractcheck`/`contractverify`) diverged completely from this doc's `live_source`/`verifier`/`drift_guard` model. |
| `docs/pocv3/design/agent-orchestration.md` | archive | `docs/archive/pocv3/agent-orchestration.md` | ADR-0009 (written to ratify this doc's substrate) is `rejected`; the failure mode it targeted (G-0099) was closed by a simpler shipped mechanism instead. Zero implementation exists. |
| `docs/pocv3/design/design-decisions.md` | relocate | `docs/design/design-decisions.md` | Load-bearing kernel-commitments record; cited by dozens of Go doc-comments and CLAUDE.md itself. |
| `docs/pocv3/design/design-lessons.md` | relocate | `docs/design/design-lessons.md` | Durable architectural discipline, cited from CLAUDE.md's "Designing a new verb" section. |
| `docs/pocv3/design/healthy-codebase-principles.md` | archive | `docs/archive/pocv3/healthy-codebase-principles.md` | Its 24-principle rubric is now duplicated and canonical in the shipped `wf-codebase-health` embedded skill; this copy is a stale fork. |
| `docs/pocv3/design/id-allocation.md` | relocate | `docs/design/id-allocation.md` | Matches the current `internal/trunk` / `aiwf reallocate` implementation exactly; kept current. |
| `docs/pocv3/design/legal-workflows-audit.md` | relocate | `docs/design/legal-workflows-audit.md` | Load-bearing test fixture — `internal/policies/m0121_audit_catalog_test.go` reads this exact path structurally. M-0127 must update that path constant. |
| `docs/pocv3/design/legal-workflows-audit-r1.md` | archive | `docs/archive/pocv3/legal-workflows-audit-r1.md` | Frozen near-duplicate subset of `legal-workflows-audit.md`; its sole purpose (freezing state for Pass B) is moot now that Pass B/C are done. Zero code references. |
| `docs/pocv3/design/legal-workflows-first-principles.md` | relocate | `docs/design/legal-workflows-first-principles.md` | Load-bearing test fixture — `internal/policies/m0122_first_principles_catalog_test.go` reads this exact path structurally. M-0127 must update that path constant. |
| `docs/pocv3/design/parallel-tdd-subagents.md` | archive | `docs/archive/pocv3/parallel-tdd-subagents.md` | Self-superseded by `agent-orchestration.md` (itself now archived); depends on the rejected ADR-0009 substrate. |
| `docs/pocv3/design/performance.md` | relocate | `docs/design/performance.md` | Explicitly self-described "living document"; actively updated with real, recent measurements. |
| `docs/pocv3/design/policy-model.md` | relocate | `docs/explorations/05-policy-model-design.md` | **Overwrites**, not just moves alongside — diffed against the existing file at that path and this is a later, more refined draft of the same design (better bundle-naming section). No rejection ADR exists for this idea, unlike agent-orchestration.md; it belongs with its sibling exploration docs (01–04, 06–09), not in archive. |
| `docs/pocv3/design/provenance-model.md` | relocate | `docs/design/provenance-model.md` | Fully implemented (`internal/scope`, `internal/verb/authorize.go`); cited directly by CLAUDE.md. |
| `docs/pocv3/design/_scratch-subagents-research.md` | archive | `docs/archive/pocv3/_scratch-subagents-research.md` | Self-describes its own eventual archive path once its synthesis (`agent-orchestration.md`) settled; that synthesis is now itself archived. |
| `docs/pocv3/design/tree-discipline.md` | relocate | `docs/design/tree-discipline.md` | Overall doctrine current, matches `internal/check/tree_discipline.go`. One stale line (pre-`aiwf edit-body`) to fix during Relocate — doesn't change the disposition. |
| `docs/pocv3/gap-triage-2026-06-05.md` | archive | `docs/archive/pocv3/gap-triage-2026-06-05.md` | Self-declared point-in-time snapshot; durable record is the gap entities, not this doc. |
| `docs/pocv3/gap-triage-2026-06-16.md` | archive | `docs/archive/pocv3/gap-triage-2026-06-16.md` | Same snapshot pattern. Its "Candidate B" (version-source split) is now tracked as **G-0432**; Candidates A and C were recommended by the doc itself to fold into the existing G-0235 rather than get their own filing. |
| `docs/pocv3/handoff/release-prep-prompt.md` | archive | `docs/archive/pocv3/release-prep-prompt.md` | One-off, dated session handoff prompt; the task it describes is long done. |
| `docs/pocv3/health-scorecard-2026-06-04.md` | archive | `docs/archive/pocv3/health-scorecard-2026-06-04.md` | Dated snapshot, explicitly superseded by the 2026-06-16 scorecard's own delta section. |
| `docs/pocv3/health-scorecard-2026-06-16.md` | archive | `docs/archive/pocv3/health-scorecard-2026-06-16.md` | Same snapshot pattern; built against a "Prior audit:" field by design, not a living reference. |
| `docs/pocv3/m0168-mutate-hunt-survivor-disposition.md` | archive | `docs/archive/pocv3/m0168-mutate-hunt-survivor-disposition.md` | Milestone-deliverable evidence for M-0168, `done`/archived under archived epic E-0042. Update the relative link in that epic's `wrap.md` when moved. |
| `docs/pocv3/m0169-vacuity-audit.md` | archive | `docs/archive/pocv3/m0169-vacuity-audit.md` | Same pattern, M-0169's deliverable evidence, also `done`/archived under E-0042. |
| `docs/pocv3/migration/from-prior-systems.md` | relocate | `docs/migration/from-prior-systems.md` | Describes the live, currently-supported `aiwf import` producer-side workflow — not historical. |
| `docs/pocv3/migration/import-format.md` | relocate | `docs/migration/import-format.md` | Normative spec for the currently-shipped `aiwf import` manifest format. |
| `docs/pocv3/overview.md` | relocate | `docs/overview.md` | Root `README.md` explicitly delegates here for FSM diagrams; verified against `internal/entity/entity.go`'s live status tables. |
| `docs/pocv3/plans/acs-and-tdd-plan.md` | archive | `docs/archive/pocv3/acs-and-tdd-plan.md` | Fully shipped (`AcceptanceCriterion`/`TDDPhase`/`acs[]` all live); no residual scope. |
| `docs/pocv3/plans/contracts-plan.md` | archive | `docs/archive/pocv3/contracts-plan.md` | I1 (the floor) shipped in full. I2's residual scope (import-manifest `contracts:` block) was considered and declined — no adopter currently migrates via `aiwf import` with pre-existing contracts. No entity paired; the whole plan archives as a unit. I3/I4 stay YAGNI-deferred per the plan's own text. |
| `docs/pocv3/plans/governance-html-plan.md` | archive | `docs/archive/pocv3/governance-html-plan.md` | Fully shipped (`internal/htmlrender`, `aiwf render --format=html`); no residual scope. |
| `docs/pocv3/plans/loom-by-example.md` | relocate | `docs/explorations/loom/loom-by-example.md` | Live-but-not-yet-committed formal-verification research, preserved for future pickup. No entity filed — companion doc to `loom-light-plan.md`; see that row for the full rationale. |
| `docs/pocv3/plans/loom-light-plan.md` | relocate | `docs/explorations/loom/loom-light-plan.md` | Zero implementation, and its own §5 lists multiple genuinely-unresolved design forks (standalone vs bundled engine, which verifier, `.lm` now-or-later) — not yet concrete enough for an epic. Relocated (not archived) to a new `docs/explorations/loom/` topic subfolder, matching the `policy-model.md` / `explorations/surveys/` precedent for open, not-yet-committed research, so it stays discoverable for whoever picks the idea back up. |
| `docs/pocv3/plans/observability-surfaces-plan.md` | supersede-with-entity | **G-0433** | Phase 1's `depends_on`-surfacing and readiness-marker items are tracked as G-0433. The plan's third Phase-1 item (local-vs-origin delta, explicitly the larger "small epic" of the three) is deferred, not filed. Phases 2–3 stay unfiled, gated on shown need per the plan's own text. Source file also archives to `docs/archive/pocv3/observability-surfaces-plan.md`. |
| `docs/pocv3/plans/provenance-model-plan.md` | archive | `docs/archive/pocv3/provenance-model-plan.md` | Fully shipped (all 10 build steps done per its own status table); matches the live `internal/scope`/`internal/verb/authorize.go` implementation. |
| `docs/pocv3/plans/status-report-plan.md` | archive | `docs/archive/pocv3/status-report-plan.md` | Self-declared shipped; no residual scope. |
| `docs/pocv3/plans/update-broaden-plan.md` | archive | `docs/archive/pocv3/update-broaden-plan.md` | Self-declared shipped; no residual scope. |
| `docs/pocv3/plans/upgrade-flow-plan.md` | archive | `docs/archive/pocv3/upgrade-flow-plan.md` | Self-declared shipped; no residual scope. |
| `docs/pocv3/README.md` | archive | `docs/archive/pocv3/README.md` | Its entire subject — the `docs/pocv3/` directory layout — ceases to exist once the directory retires. No standalone value once relocated; its reusable charter language folds directly into M-0128's new hierarchy section. Archived rather than deleted, per the milestone's forget-by-default default. |
| `docs/pocv3/skill-author-guide.md` | relocate | `docs/skill-author-guide.md` | Resolves epic Open Question #4. Genuinely unique content (no overlap found in `.claude/skills/` or `internal/skills/embedded-rituals/`); audience is skill authors generally, not just ritual authors, so it stays top-level rather than absorbed into the rituals plugin's own docs. |
| `docs/pocv3/workflows.md` | relocate | `docs/workflows.md` | Root `README.md` explicitly delegates here; verb sequences verified against the live CLI. |
