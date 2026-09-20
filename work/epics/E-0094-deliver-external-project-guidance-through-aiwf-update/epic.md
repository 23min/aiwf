---
id: E-0094
title: Deliver external project guidance through aiwf update
status: proposed
---
## Goal

Let maintainers use `aiwf init` and `aiwf update` to select, refresh, and deliver externally authored engineering guidance as tracked project policy for Claude and Codex. Contributors need neither aiwf nor ai-dotfiles nor VS Code synchronization to read that policy.

## Context

E-0093 provides the existing host-delivery foundation. D-0089 records the proposed external content-ownership boundary. This epic delivers and migrates project guidance before E-0092 freezes its baseline and reduces instruction load.

Language conventions change independently of workflow machinery. Keep their content and detection patterns outside aiwf; aiwf owns selection, retrieval, and project delivery. Adapt the existing ai-dotfiles detector without retaining a dependency on a local dotfiles checkout.

## Scope

### External source and initial content

- Use `23min/engineering-guidance` as the default canonical source. Establish that repository as part of delivery; its availability is not yet verified. It contains Markdown and a small declarative catalogue, without requiring aiwf planning or runtime machinery.
- Seed it with the existing engineering and language guidance from ai-dotfiles, including guidance needed by aiwf itself. Preserve wording except for necessary packaging, naming, and reference changes. Personal collaboration rules, approvals, machine setup, and session tooling stay in ai-dotfiles.
- Catalogue entries carry pack ids, document paths, applicability descriptions, and simple detection patterns. Opinionated names disclose tooling choices. A new language requires external content and patterns, not an aiwf release.

### Configuration and selection

- Guidance maintenance defaults on for aiwf projects, including existing projects without a guidance-source setting. One upstream serves each project; maintainers may override the default source or disable maintenance.
- Store selected pack ids in `guidance.packs` and suppressed suggestions in `guidance.ignored` in `aiwf.yaml`. Merge these settings with existing guidance configuration without discarding unrelated fields.
- During interactive initialization, detect applicability and ask for the initial selection. Detection does not adopt policy. During interactive update, offer newly applicable packs with descriptions; accepted selections are recorded and installed in that invocation.
- Offer all matching catalogue choices without designating a default pack. Neither a package manager nor a base pack is implicitly selected. There are no pack dependencies or automatic inclusion rules; each pack is explicitly selected. Explicit selection also supports projects whose source files do not yet exist.
- Each suggestion permits select, not now, or ignore. Select records the id in `packs`; ignore records it in `ignored`; not now records neither. Leaving a choice unchecked does not silently ignore it. Selected and ignored ids are excluded from further suggestions.
- To remove a selected pack persistently, move its id from `packs` to `ignored` and run update. Remove an ignored id to make it eligible again. Removing only a selection can cause it to be suggested again. No dedicated removal command is needed.
- Noninteractive runs report suggestions and refresh existing selections without adopting new policy. A selected pack with no current matching files is reported and retained.

### Detection and upstream refresh

- Adapt marker-file and extension/path matching, nested-project scanning, exclusions, and explanations from the existing detector. Keep language patterns in the external catalogue. Exclude gitignored, vendor, and generated material; do not add framework dependency parsing.
- Each enabled update checks the latest default branch through a fresh temporary Git clone using existing Git credentials. Remove the temporary checkout afterward. No persistent cache, tag/version selection mechanism, or separate lockfile.
- Validate the catalogue, all selected documents, output paths, and ownership conflicts before replacing any installed guidance. An unavailable source, missing selected pack, invalid input, or locally edited generated output leaves the installed guidance set and its recorded revision unchanged. Report the reason and continue unrelated aiwf update work. An incomplete first installation is visibly incomplete.
- A successful refresh updates tracked outputs and records the exact installed source commit in the index. Unchanged inputs produce no guidance diff. Updates prepare working-tree changes; they never commit or push automatically.

### Project files and host routing

- Deliver selected documents and a small generated `.guidance/index.md` under tracked `.guidance/`. Keep handwritten project exceptions in `.guidance/project.md`, outside generated ownership.
- Maintain concise routing blocks in `AGENTS.md` and `CLAUDE.md` through the existing host selection and wiring controls. Both route to the same local content: read project overrides first, then the index and task-relevant packs. Routing explicitly states that project overrides take precedence over pack guidance.
- Keep the index and routing small. Do not load the complete selected language bundle upfront or create a skill for each language. Shared engineering guidance and task-specific packs have explicit applicability.
- Preserve handwritten files and surrounding host instructions. Refuse to overwrite locally modified generated outputs. Removing a pack removes its unmodified owned output and index entry; a conflict preserves the installed set for review.
- `guidance.enabled: false` disables upstream checks, detection suggestions, and external guidance refreshes. Leave installed files and routing untouched: this stops maintenance, not use of the existing project policy. Existing aiwf operating guidance remains independently maintained.
- Local diagnostics distinguish selections, installed revision, missing artifacts, and modified owned content. Only a successful upstream check establishes freshness; installed files do not prove that a model read them.

### Migration and measurement

- Deliver an ai-dotfiles compatibility update before migrating repositories. Its global host entry points retain personal rules and a short routing instruction, with engineering guidance, including code-health, outside unconditional global content. Keep the legacy guidance files available. Retain a compatible distribution from the canonical external source rather than independently editing two corpora.
- Route by installed project ownership: an aiwf-managed `.guidance/index.md` selects project guidance exclusively; without it, use legacy ai-dotfiles guidance. An explicitly empty installed selection still owns project policy and must not trigger fallback. Disabled maintenance does not change installed ownership. This routing is an assistant instruction to read files, not a native conditional import; verify both hosts in fresh sessions.
- Teach `dotfiles-sync` the same ownership check so it does not recreate legacy imports in migrated repositories. Preserve its existing delivery for unmigrated repositories, including old-aiwf and non-aiwf consumers. Changing global loading to explicit legacy reads must preserve access and be observed; do not claim identical loading behavior.
- Before project handover, check for incompatible ai-dotfiles delivery, including the unconditional global Codex bundle. If present, report that ai-dotfiles needs updating and leave guidance migration incomplete with legacy delivery intact. The aiwf binary upgrade and unrelated update work may proceed. Do not silently rewrite personal configuration or install a second active corpus.
- After validating replacements and compatibility, install project guidance and host routing and remove recognized legacy managed imports. The installed index records ownership; no separate migration registry or launcher wrapper. Define interruption recovery together with materialization so failed handover does not strand a repository between owners. Existing sessions containing legacy instructions need restarting before verifying exclusive project delivery.
- Verify Claude and Codex from a checkout without ai-dotfiles, including root-started tasks on nested files, new files, and unrelated prose. Record observed reads and behavior separately from installation checks.
- Measure growth before and after delivery/migration. Link the allocated prerequisite into E-0092 so its reduction baseline follows this work. Broad deduplication and compression remain E-0092's responsibility.

## Out of scope

- Language conventions or a fixed supported-language list embedded in aiwf.
- Registries, marketplaces, dependency solvers, transitive includes, executable provider hooks, or per-language plugins.
- Background updates, session-start policy writes, a separate guidance-upgrade command, persistent downloads, or a lockfile.
- Framework detection through dependency parsing; additional host adapters such as Copilot.
- Rewriting the imported guidance corpus, authoring new language guidance merely to expand coverage, or replacing personal preferences with project policy.
- E-0092's instruction ceiling and broad reduction work; universal guarantees of model compliance.

## Constraints

- Prefer the existing ownership, rendering, and safe-write machinery where it fits. External tracked policy must coexist with aiwf's existing generated artifacts and user-authored host files.
- Reject unsafe paths and malformed external inputs; never execute provider content as installation code. Keep configuration and catalogue validation at their boundaries.
- Complete validation before writes and use atomic file replacement. Specify recovery for interrupted multi-file writes without claiming a filesystem transaction that the implementation cannot provide.
- Document the settings, selection actions, failure reporting, and removal procedure alongside implementation. Preserve existing host opt-outs and unrelated configuration.
- Use behavior tests with local Git and filesystem fixtures for retrieval, selection, conflicts, removal, and repeatability. Do not depend on live upstream availability or freeze imported prose wording in tests.
- Live host observations record command/task, expected result, observed result, and environment. Network actions, commits, publication, and other outward actions retain the operator's approval boundaries.

## Success criteria

- [ ] The external source contains the existing engineering/language corpus, separated from personal settings, with usable descriptions and detection patterns.
- [ ] Default-enabled init/update suggest applicable packs while explicit selection, not now, ignore, and noninteractive operation preserve maintainer control.
- [ ] Adding an external language pack and detection patterns requires no aiwf code change.
- [ ] Enabled update checks upstream on demand without a persistent cache and records the installed commit in the tracked index.
- [ ] Fetch, validation, missing-pack, and local-edit failures preserve the installed guidance set and report the failure while unrelated update work continues.
- [ ] Persistent removal and re-enabling suggestions work through `packs` and `ignored`; absent language files never silently remove a selection.
- [ ] Disabled maintenance leaves installed policy and routing usable without network checks or suggestions.
- [ ] Both hosts reach shared project-local guidance and respect project overrides in the exercised root/nested/new-file tasks; prose-only tasks avoid irrelevant language reads.
- [ ] Repeated unchanged updates preserve bytes, handwritten overrides, and unrelated configuration and instructions.
- [ ] aiwf and its personal-bootstrap integration use the new delivery boundary, with growth measurements and E-0092 prerequisite linkage recorded.
- [ ] Updating ai-dotfiles preserves guidance for old-aiwf and non-aiwf consumers; migration of one repository on a shared machine does not withdraw guidance from another. Failed installation leaves legacy delivery usable.
- [ ] Successful migration leaves one active engineering-guidance path per repository. Both hosts exercise legacy fallback and exclusive project routing; empty installed selections and disabled maintenance do not re-enable legacy guidance.
- [ ] Incompatible personal delivery blocks guidance handover with an actionable diagnostic while allowing the binary upgrade and unrelated update work; personal files are not silently rewritten.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact catalogue schema, source-override field, and pack-to-file mapping | Before implementation | Choose the smallest validated representation against current config and ownership code |
| Which ai-dotfiles source files and bootstrap paths move or change | Before migration | Inventory maintained sources; map engineering content to packs and preserve personal settings |
| Recovery and reporting for interrupted writes | Before materialization ships | Define behavior against the existing writer and exercise interruption/retry fixtures |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Default-enabled maintenance is mistaken for automatic policy adoption | High | Explicit selections; noninteractive runs only suggest |
| An upstream failure or conflict causes mixed revisions | High | Preflight the complete selection; preserve installed state on validation failures |
| Both old and new paths load the same rules | High | Ship compatibility routing first, preflight handover, and verify exclusive reads in fresh sessions |
| Hosts miss relevant guidance or ignore overrides | High | Shared routing with explicit precedence and observed task reads |

## Milestones

- M-0344 — Publish the external engineering guidance corpus. No milestone prerequisites.
- M-0345 — Preserve legacy guidance through repository-aware routing. Depends on M-0344.
- M-0346 — Deliver explicitly selected project guidance through update. Depends on M-0344, M-0345.
- M-0347 — Suggest applicable guidance during init and update. Depends on M-0346.
- M-0348 — Verify migration and establish the reduction prerequisite. Depends on M-0344, M-0345, M-0346, M-0347.

## References

- E-0093 — supported hosts and existing delivery foundation.
- E-0092 — post-delivery guidance reduction and frozen baseline.
- D-0089 — proposed external content-ownership boundary.
- ADR-0014 — existing managed host delivery; reconcile this external tracked-policy scope with its distribution boundary.
- `internal/skills/render.go`, `internal/skills/ownership.go`, `internal/initrepo/agents_guidance.go` — existing delivery seams to assess for reuse.
- `docs/design/growth.md`, `scripts/growth-report.py` — growth measurements.
