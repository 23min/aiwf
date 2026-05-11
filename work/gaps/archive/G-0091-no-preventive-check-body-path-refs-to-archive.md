---
id: G-0091
title: No preventive check for body-prose path-form refs to archive-moved entities
status: addressed
addressed_by_commit:
    - abf788f
---
Body-prose markdown links that target entity files by path (e.g., `[ADR-0004](../adr/ADR-0004-uniform-archive-…md)` rather than the id-form `ADR-0004`) are brittle. When the target archives — moves from `work/<kind>/` to `work/<kind>/archive/` per ADR-0004 — the link breaks. There is no `aiwf check` rule that catches this preventively; the failure surfaces only when CI runs `link-check.yml` (lychee) on the next push, after the break has already shipped.

## Existing chokepoints

Two layers cover adjacent concerns but leave this gap uncovered:

- **`internal/check/check.go:377` — `refsResolve`** validates *id-form* references in frontmatter (`Resolves:`, `superseded_by:`, `linked_adrs:`, etc.) by id, regardless of file location. The forward-ref set is enumerated in `internal/entity/refs.go:19` (`ForwardRefs`). Per ADR-0004, the loader spans active+archive, so id-form refs are archive-safe — the kernel resolves them whether the target is in the active dir or under `archive/`. **Body-prose markdown link syntax is not part of the frontmatter and is not parsed by `refsResolve`.**

- **`.github/workflows/link-check.yml`** runs lychee against `**/*.md` on PRs and pushes that touch markdown, plus a weekly cron. Catches path-form breakage *post-archive* — a contributor archives an entity, the next CI run fails, and someone fixes the path-form refs after the fact. Reactive, not preventive.

## Why this matters

Path-form refs are not strictly broken at write-time — they resolve fine until the target archives. ADR-0004's archive convention turns this from "rare future risk" into "predictable failure mode every time `aiwf archive --apply` runs." The frequency rises proportionally with archive volume: gaps and the proposed F-NNN findings (companion ADR-0003) are the highest-volume kinds.

Three failure modes the post-hoc check leaves uncovered:

1. **Time-shifted blame.** The PR that introduced the path-form ref passes CI. The PR that runs `aiwf archive --apply` two months later breaks the link and looks like the cause. `git bisect` doesn't trivially identify the underlying issue (the introducing commit was clean at the time it landed).

2. **Link-check is a post-push catch, not a pre-push one.** `aiwf check` runs as a pre-push hook (CLAUDE.md "What aiwf commits to" §3); link-check.yml runs in GitHub Actions after push. Path-form drift escapes the local chokepoint that aiwf is designed to enforce its guarantees through. CLAUDE.md §5 — "framework correctness must not depend on the LLM's behavior" — implies the same about CI: the pre-push hook is the contract, not the post-push workflow.

3. **Existing path-form refs across `docs/` and entity bodies are unaudited.** Without a sweep, the codebase carries an unknown number of path-form refs to entities that haven't archived *yet*. ADR-0004's ratification + first `aiwf archive --apply` run is the trigger that surfaces all of them at once.

## Fix shape

Three layers, ordered roughly by cost:

1. **Codebase-wide sweep at ADR-0004 ratification.** Convert existing body-prose path-form refs to id-form (or to relative anchors that don't include the entity's slug). Mechanical; one or two PRs of work. Establishes the clean baseline before archive moves can break anything.

2. **A new check finding `body-ref-path-form: prefer id-form for entity references`.** Detection logic: parse markdown links in entity bodies, match path against `work/<kind>/(?:archive/)?<id>-…` and `docs/adr/(?:archive/)?ADR-NNNN-…` shapes; warn under default strictness; configurable to error via `aiwf.yaml`. Moderate impl cost — markdown-link parsing in body prose is non-trivial. The lint should respect the `acs-body-coherence` precedent of operating on entity-body markdown specifically, not all `*.md` files in the repo.

3. **Skill-level guidance in entity-authoring skills** (`aiwf-add`, `aiwf-edit-body`, plus the rituals plugin's wrap skills). When the operator is writing body prose that references another entity, prefer id-form. Advisory; doesn't satisfy CLAUDE.md §5 ("framework correctness must not depend on the LLM's behavior") but cheap to add as a defense-in-depth layer.

The right combination is probably (1) + (2): one-time sweep to clean up; a check rule going forward to prevent regressions. (3) is a free augmentation but not load-bearing.

## Out of scope

- **Generic markdown link discipline across `docs/` and external links.** lychee already covers external link rot and broken relative links between non-entity markdown files. This gap is specifically about *entity-targeting* refs where id-form is the kernel's canonical address.
- **Frontmatter-level `Refs:` rewriting.** Already handled by `refsResolve`; entities use id-form in frontmatter today.
- **The broader question of which markdown surfaces are authoritative.** A separate concern about doc-authority hierarchy (which `docs/` trees are normative vs exploratory vs archival) deserves its own gap; this one is scoped to the path-form-vs-id-form discipline only.

## References

- **ADR-0004** — Uniform archive convention for terminal-status entities; the convention this gap depends on. Its Negatives section already names path-form brittleness as a known cost; this gap captures the work item that follows.
- **`internal/check/check.go:377`** — `refsResolve`, the existing id-form ref-validity rule.
- **`internal/entity/refs.go:19`** — `ForwardRefs`, the canonical enumeration of frontmatter reference fields.
- **`.github/workflows/link-check.yml`** — the existing lychee-based post-hoc markdown link-check workflow.
- **CLAUDE.md** "What aiwf commits to" §3 — pre-push hook is the chokepoint; post-push CI is not equivalent for kernel guarantees.
- **CLAUDE.md** "What aiwf commits to" §5 — framework correctness must not depend on the LLM's behavior; skill-level guidance alone is not a kernel-level fix.
