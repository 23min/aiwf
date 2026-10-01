---
id: E-0092
title: Shrink the always-on guidance to one home per rule under a ceiling
status: active
---
## Goal

Give every guidance rule in this repository one home and one clear form. Rules that apply to nearly every task are primed in the root instructions under a ceiling for each host; everything else is read on demand through the project guidance router. Duplicates, near-duplicates and conflicts are resolved, verbose rules are tightened, and each rule keeps its effect.

## Context

E-0094 is done: selected language packs are delivered as tracked files under `.guidance/`, and both host entry points carry a managed routing block that sends a task to `.guidance/project.md`, then the index and the packs it needs. M-0348 observed both hosts following that route from a session started at the repository root, and observed Claude skipping it whenever `CLAUDE.md` already answered.

The primed load today, measured in whitespace-split words on `main` after E-0094:

| Host | Loaded before any task | Words |
|---|---|---|
| Claude | `CLAUDE.md`, plus the imported aiwf fragment | 9,543 + 2,235 |
| Codex | `AGENTS.md` with the fragment rendered inline, plus the required full read of `CLAUDE.md` its preamble demands | 2,503 + 9,543 |

`.guidance/project.md` currently says to read `CLAUDE.md` in full and that it takes precedence over the packs. The largest `CLAUDE.md` sections are Go conventions (4,274 words, mostly aiwf-specific development rules), the stress-test harness (790), what aiwf commits to (543), working with the user (459) and AC promotion (445). The Go pack itself is 285 words.

Repository development rules, aiwf operating guidance and project-adopted language conventions have different owners. A check-backed rule can become a short pointer once its diagnostic states the remedy. G-0676 identifies how root guidance grows during ordinary work; the fence addresses that growth before the reduction starts.

## Scope

- **The fence, M-0333.** Gate commits that change guidance, require a disposition for removed text, and fail CI when either host's primed load exceeds its ceiling.
- **The shipped fence, M-0350.** Move the commit rules into `aiwf check`, so every aiwf repository fences its handwritten instruction files by default (ADR-0053).
- **The inventory, M-0349.** Freeze the installed guidance. List every rule once — in this repository's guidance and in the shipped fragment — with its current homes and its audience, and give it a disposition. Decide every conflict with the maintainer. Name the on-demand documents and set the primed ceiling from the result.
- **The baseline, M-0334.** Record both hosts' loading and rule-following on tasks aimed at the rules the inventory changes, before any decision is carried out, using M-0348's observation harness.
- **Remove copies and move rules home, M-0336.** Delete text whose rule already lives in the aiwf fragment or a selected pack, and move each rule the inventory places in the other home, as the inventory records. From here on a test holds every removal to the inventory.
- **Route and tighten, M-0335.** Make `.guidance/project.md` a short task router, move on-demand rules into the documents it names, and apply the inventory's merge and tighten dispositions. The root `CLAUDE.md` keeps only primed rules.
- **Pointer cut, M-0337.** Audit diagnostics, correct messages that omit the remedy, shorten check-backed rules, and reach the ceiling for each host.
- **After observation, M-0338.** Compare each host against its own frozen baseline, including guidance read during the task, and confirm every inventory decision was carried out.

## Out of scope

- External source selection, language detection, pack selection, upstream refresh and tracked-output delivery; E-0094 owns them.
- Changing the content of external language packs; their wording belongs to the external source.
- Personal collaboration preferences and machine/session configuration, which live outside this repository.
- Entity templates and general changes to ritual or verb skill bodies, except references that must follow relocated guidance.
- Changing existing check conditions while improving their messages.
- New host adapters, including Copilot, or universal guarantees of model compliance.
- A size ceiling or size report for consumer repositories; the ceiling and the guidance-reader list are internal to this repository (ADR-0053).
- Fencing instruction files below a repository's root (ADR-0053).
- Rewording or shortening the shipped fragment's rules; the inventory records any such tightening as a suggestion for a later epic.

## Constraints

- **One instruction file per host, at the root.** Claude primes from the root `CLAUDE.md` and Codex from the root `AGENTS.md`. No instruction files in subdirectories; on-demand guidance is reached through `.guidance/project.md`.
- **Primed versus on demand.** A rule is primed when it applies to nearly every task, or when missing it causes harm before the agent would think to look. Everything else is on demand.
- **One home per rule, chosen by audience.** Rules for operating aiwf in any repository live in the shipped fragment; rules for developing aiwf live in this repository. Generated copies in host entry points are delivery outputs, not independent homes. A rule in the wrong home moves to the right one in a single commit, so it is never in both.
- **Effect is preserved, wording is not.** Rules may be merged, tightened or rewritten as the inventory records. The baseline and after-observations are the check that a changed rule kept its effect. No rule is dropped silently: each removed passage's disposition is recorded in its commit body with the inventory row it carries out, and a test holds every removal to the inventory.
- **Two ceilings.** Each host has a ceiling on its handwritten primed words, set by M-0349 and enforced by M-0333's policy. The shipped fragment's size is tracked separately; this epic moves rules into and out of it but does not shorten its wording. A reference that requires a full read before any task counts as primed.
- **Frozen guidance.** From M-0349's inventory until M-0338's after-observation, `guidance.enabled` is `false` in `aiwf.yaml`, so no `aiwf update` or `aiwf worktree add` refreshes the installed packs. Re-enable it at the epic's wrap.
- **The fence lands first.** Its commit scope allows related guidance sources and generated outputs together, but excludes unrelated implementation changes. Generated blocks change through their owner, never by hand.
- **Pins retire with a recorded reason.** D-0102 bars evidencing a criterion with a sentence pinned in either host entry point, the project router or the on-demand documents; a pin retires with its passage and leaves the reader list with it.
- A check-backed section becomes a pointer only after its diagnostic states the remedy.
- **Observations are records.** Commit the rubric before its first run. Record command, expected result, observed result and environment, including host/model versions and the installed guidance revision. Judge reads on tool events, not on the assistant's own account. Runs that invoke `aiwf update` point `HOME` at a scratch directory.
- Record a growth-report row at each reduction boundary and derive the removal manifest from commits at wrap.
- Personal approval rules remain in effect: enumerated local reversible sequences may share approval; outward actions retain separate gates.

## Success criteria

- [ ] E-0094 is named as the delivery prerequisite and its completion is verified before implementation.
- [ ] Every guidance rule, in this repository's guidance and the shipped fragment, has one home chosen by its audience and a recorded disposition; no conflicting rules remain.
- [ ] Both hosts' handwritten primed words meet their ceilings; the policy catches regrowth and required-read indirection.
- [ ] Every aiwf repository fences edits to its handwritten instruction files by default (ADR-0053).
- [ ] No instruction file exists below the repository root; both hosts reach task-relevant guidance through `.guidance/project.md` from a root-started session, and no home-directory language import returns.
- [ ] Development guidance duplicates neither operating rules nor the selected language packs.
- [ ] Each moved pin is retired with its reason recorded.
- [ ] Guidance references and enforcement pointers resolve.
- [ ] Both hosts have before/after observations of the rules this epic changes, against the frozen baseline, with task-loaded text reported separately.
- [ ] Growth measurements are recorded, every inventory decision is carried out by a commit that names its row, and no removal falls outside the inventory.
- [ ] G-0676 and G-0436 are closed against evidence of their claims.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which on-demand documents exist and where they live | yes, before M-0335 | M-0349 names them from the inventory |
| The handwritten primed ceiling for each host | yes, before M-0337 | M-0349 sets it from the primed rules' tightened size |
| Which diagnostics already state the remedy | no | M-0337 produces the audit |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A thinned root no longer answers a task and the host does not follow the router | high | Observe root-started reads for both hosts in M-0335 and M-0338 |
| Tightening a rule changes its effect | high | Rubric tasks for each rule the inventory changes, compared against the frozen baseline |
| A rule is lost between its old and new home | high | Each removal names its inventory row; a test holds removals to the inventory from M-0336, and M-0338 requires every decision carried out |
| Upstream pack changes confound the comparison | high | `guidance.enabled: false` for the comparison window |
| A generated host copy is mistaken for a duplicate rule home | med | Judge ownership at the source, using the managed-block markers and `.guidance/.aiwf-owned` |
| New enforcement machinery outweighs the reduction | med | Reuse existing scans and report policy/test growth alongside prose savings |

## Milestones

- M-0333 — Fence repository guidance and measure primed load for both hosts.
- M-0350 — Ship the instruction-file fence in `aiwf check`; depends on M-0333.
- M-0349 — Freeze the guidance, inventory every rule in this repository and the shipped fragment, and decide its home and form; depends on M-0333.
- M-0334 — Record the baseline for both hosts on the rules the inventory changes; depends on M-0349 and M-0350.
- M-0336 — Remove copies and move rules to their audience's home; depends on M-0349 and M-0334.
- M-0335 — Route on-demand guidance through the project router and tighten what moves; depends on M-0336.
- M-0337 — Improve diagnostics and reach the ceiling for both hosts; depends on M-0335.
- M-0338 — Compare behaviour and load against the frozen baseline and confirm every inventory decision was carried out; depends on M-0337.
- M-0339 — Cancelled: shortening the shipped fragment's wording is left to a later epic.

## References

- E-0093 — supported Claude and Codex workflows.
- E-0094 — external guidance delivery and this repository's migration; M-0348 records the post-delivery measurements and host observations.
- ADR-0052 — project guidance independent of personal bootstrap.
- ADR-0053 — the instruction-file fence ships in `aiwf check`.
- ADR-0054 — external language-content ownership and tracked project delivery; applicable packs are adopted on update.
- D-0102 — the evidence rule over the development-guidance set, superseding D-0091.
- D-0070 — limits on pins over shipped prose.
- G-0676, G-0436 — the growth and stale-reference defects this epic addresses.
- G-0668 — measurement records need reproducible commands.
- `docs/design/growth.md`, `scripts/growth-report.py` — measurements and iteration log.
- `internal/policies/skill_edit_provenance_backstop.go`, `internal/policies/m0211_guidance_operating_anchors.go` — existing enforcement seams.
