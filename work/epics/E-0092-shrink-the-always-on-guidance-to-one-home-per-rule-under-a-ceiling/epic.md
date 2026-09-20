---
id: E-0092
title: Shrink the always-on guidance to one home per rule under a ceiling
status: proposed
---

## Goal

Reduce the project instructions Claude and Codex read before a task to a ceiling for each host, while keeping each rule in one canonical home and preserving its effect. Measure task-loaded guidance separately so relocation cannot masquerade as a reduction.

## Context

E-0093 establishes the two supported hosts. Their entry points and loading mechanisms differ, so a filename-only measure cannot describe the instruction load of both. An instruction to read an entire document before any work belongs in the upfront load even when the host does not import that document automatically.

Repository development rules, aiwf operating guidance, and project-adopted language conventions have different owners. The delivery follow-up to E-0093 will establish tracked project guidance from an external source and migrate this repository off home-directory language imports. This epic reduces the resulting instructions; it does not implement that delivery system.

A check-backed rule can become a short pointer once its diagnostic states the remedy. Judgment rules need their reasoning retained and their effect observed. G-0676 identifies how root guidance grows during ordinary work; the fence addresses that growth before the reduction starts. Python scripts and TypeScript tests remain legitimate consumers of language guidance alongside Go code.

## Scope

- **Delivery prerequisite.** The separate follow-up to E-0093 must complete external guidance delivery and this repository's migration before implementation here. Its implementation and pre/post growth measurements belong to that epic.
- **The fence, M-0333.** Cover both host entry points and the canonical repository-development documents they route to. Preserve generated-block ownership and permit one guidance change to update its source and derived outputs together.
- **The baseline, M-0334.** Freeze a post-delivery, pre-reduction commit and guidance revision. Record both hosts' upfront load, task-loaded guidance, and behavior against the same tasks.
- **Relocation, M-0335.** Give task-specific development rules project-local homes reachable from either host, including a session started at the repository root. Re-aim or retire affected pins with reasons.
- **Delete copies, M-0336.** Remove duplicate operating rules, generic language conventions and provenance asides while preserving their canonical sources and routing.
- **Pointer cut, M-0337.** Audit diagnostics, correct messages that omit the remedy, and shorten check-backed rules. Reach the upfront ceiling for each host.
- **After observation, M-0338.** Compare each host against its own frozen baseline, including guidance loaded during the task.
- **Fragment, M-0339.** Shorten shared operating guidance only if both hosts show no lost effect; retain one source and verify both rendered forms.

## Out of scope

- External source selection, language detection, pack selection, upstream refresh and tracked-output delivery; the separate delivery epic owns them.
- Authoring language conventions inside aiwf or requiring VS Code/ai-dotfiles to supply project guidance.
- Personal collaboration preferences and machine/session configuration in ai-dotfiles.
- Entity templates and general changes to ritual or verb skill bodies, except references that must follow relocated guidance.
- Changing existing check conditions while improving their messages.
- New host adapters, including Copilot, or universal guarantees of model compliance.
- A ceiling imposed on consumer repositories; the reduction policies are internal to this repository.

## Constraints

- Delivery and migration finish first. M-0334's baseline uses the resulting project guidance, not a pre-delivery checkout. No upstream guidance refresh occurs between the before and after observations.
- The target is 3,500 whitespace-split words of upfront project instructions for each host. Count automatically loaded project text and documents required before any task. Report conditional task reads and personal/global instructions separately; do not exempt upfront text merely by moving it behind a reference.
- Each rule has one authored source. Generated copies in host entry points are delivery artifacts, not independent rule owners.
- The fence lands before the first reduction. Its commit scope allows related guidance sources and generated outputs together, but excludes unrelated implementation changes. Generated blocks are changed through their owner, never edited by hand.
- No rule is silently lost. Relocate or compress it; retain a short reason for judgment rules. Record each removed passage's disposition in the commit body.
- Language content stays externally owned; project-adopted copies are tracked and readable without aiwf or ai-dotfiles installed. D-0089 records the ownership boundary.
- Pins move or retire with a recorded reason. Extend D-0091's prohibition on prose-presence evidence to both host entry points and relocated development guidance.
- A check-backed section becomes a pointer only after its diagnostic states the remedy.
- Commit the observation rubric before its first run. Record command, expected result, observed result and environment, including host/model versions and guidance source revision.
- Record a growth-report row at each reduction boundary and derive the removal manifest from commits at wrap.
- Personal approval rules remain in effect: enumerated local reversible sequences may share approval; outward actions retain separate gates.
- Operational loading observations must distinguish readable files, observed reads, and behavioral compliance.

## Success criteria

- [ ] The delivery prerequisite is identified by its allocated entity and its completion is verified before implementation.
- [ ] Both hosts' upfront project instructions meet the ceiling; the policy catches regrowth and required-read indirection.
- [ ] Both hosts reach task-relevant project guidance from a root-started session, without home-directory language imports.
- [ ] Repository development guidance duplicates neither operating rules nor the selected language conventions.
- [ ] Each moved pin is re-aimed or retired with its reason recorded.
- [ ] Guidance references and enforcement pointers resolve.
- [ ] Both hosts have before/after observations against the frozen post-delivery baseline, with task-loaded text reported separately.
- [ ] Growth measurements and the commit-derived removal manifest are recorded.
- [ ] G-0676 and G-0436 are closed against evidence of their claims.
- [ ] If the fragment stage runs, both host renderings retain their routing and operating obligations without a second authored copy.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which entity owns delivery and migration | yes, before implementation | Allocate the follow-up to E-0093 next and replace the dependency description with its real id |
| Which generated paths and blocks belong to each updater | yes, before M-0333 | Read the completed delivery implementation; use its ownership records |
| Which diagnostics already state the remedy | no | M-0337 produces the audit |
| Whether moving guidance preserves delivery and behavior | yes, before wrap | M-0335 and M-0338 record both hosts' observations |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A root-started host misses nested guidance | high | Explicit routing and observed reads before relevant edits for both hosts |
| Upstream changes confound the reduction comparison | high | Freeze source revision and installed bytes across baseline and after runs |
| A generated host copy is mistaken for a duplicate rule source | med | Judge ownership at source and rendered load per host |
| New enforcement machinery outweighs the reduction | med | Reuse existing scans and report policy/test growth alongside prose savings |
| A moved judgment rule loses effect | high | Keep its reasoning and compare behavior against the fixed rubric |

## Milestones

- M-0333 — Fence repository guidance and measure upfront load for both hosts; delivery prerequisite required.
- M-0334 — Freeze the post-delivery baseline and observe both hosts; delivery prerequisite required.
- M-0335 — Relocate development guidance with verified routing for both hosts; depends on M-0333 and M-0334.
- M-0336 — Remove copies while preserving project-local sources; depends on M-0335.
- M-0337 — Improve diagnostics and reach the ceiling for both hosts; depends on M-0336.
- M-0338 — Compare behavior and load against the frozen baseline; depends on M-0337.
- M-0339 — Shorten the shared fragment only if both hosts retain its effect; depends on M-0338.

## References

- E-0093 — supported Claude and Codex workflows; its delivery follow-up is the prerequisite to be allocated.
- D-0089 — external language-content ownership and project-local delivery.
- D-0091 — prose-presence evidence restriction; apply it to both hosts here.
- D-0070 — limits on pins over shipped prose.
- G-0676, G-0436 — the growth and stale-reference defects this epic addresses.
- G-0668 — measurement records need reproducible commands.
- `docs/design/growth.md`, `scripts/growth-report.py` — measurements and iteration log.
- `internal/policies/skill_edit_provenance_backstop.go`, `internal/policies/m0211_guidance_operating_anchors.go` — existing enforcement seams.
