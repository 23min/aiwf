---
id: G-0102
title: Entity titles uncapped; long titles break filesystem paths, status HTML layout, and CLI table columns
status: open
---
## What's missing

`aiwf add <kind>` accepts a `--title` of arbitrary length and slugifies it directly into the entity's filename. There is no cap at either layer:

- **No title-length cap** at verb-level — `aiwf add gap --title "..."` accepts any string the shell will pass.
- **No slug-length cap** at slugification — the full title (with punctuation, accents, and spaces collapsed to hyphens) becomes the on-disk slug between the canonical `<KIND>-NNNN-` prefix and the `.md` suffix.

Real measurements from this repo (active tree, `work/gaps/` only):

```
250  G-0099-orchestration-design-s-worktree-isolation-depends-on-agent-kwarg-honor-...
228  G-0067-wf-tdd-cycle-is-llm-honor-system-advisory-under-load-the-llm-bypasses-...
219  G-0069-aiwf-init-s-printritualssuggestion-hardcodes-the-cli-install-form-...
217  G-0088-skill-coverage-policy-walks-internal-skills-embedded-only-plugin-...
202  G-0100-provenance-untrailered-entity-commit-fires-on-git-merge-commits-...
```

Five gap filenames already over 200 chars; G-0099 at 250 chars is approaching the **255-byte filename limit** on macOS APFS and Linux ext4/xfs. Add an archive move (`work/gaps/archive/...`) and a typical repo root, and absolute paths cross 350 chars — past where some Windows tools, zip utilities, and CI cache layers stop coping.

Surfaces that degrade in proportion to title length:

1. **Filesystem.** Filenames approach OS limits; `tar`, `zip`, some Docker volume mounts, and Windows cross-platform tooling start failing silently or noisily. Tab-completion of bare paths becomes unusable.
2. **`aiwf status` CLI output.** Long titles cause table rows to wrap mid-cell (per G-0080), breaking column alignment so the eye can't scan down the STATUS column or the PARENT column.
3. **`aiwf list` CLI output.** Same wrapping pathology — and worse for wider terminal users who expect one row per entity.
4. **`aiwf render` status HTML.** Long titles overflow narrow columns, force horizontal scroll, or stretch the page wider than typical browser viewports. Renderer mitigations (CSS truncation, hover-to-expand) work around the symptom but don't address that the title was unbounded to start with.
5. **`git log` rendering.** Entity-touching commits put the title (or a derivative) in the commit subject; long titles violate the Conventional-Commits ~72-char rule and wrap awkwardly in `git log --oneline`.
6. **`aiwf history <id>` output.** Per-event lines truncate or wrap on long titles.

The current title shape encourages this — many existing gaps put the *problem statement* in the title (which is the natural place to land it during a fast `aiwf add gap` flow). That pattern is the source of the length explosion: title carries content that belongs in the body.

**Possible policy directions** (not prescribing — that's for the implementing milestone):

- **A. Cap the title** at verb-level — `aiwf add` refuses if `--title` exceeds N chars (e.g., 80 or 100), with a hint that elaboration belongs in `--body-file` or post-add `aiwf edit-body`. Forces concise titles; some friction for fast capture.
- **B. Cap the slug only** — title can stay arbitrarily long; slug derivation truncates to N chars (e.g., 80). Title remains expressive in frontmatter and rendered output; filesystem and CLI-table symptoms go away. Doesn't address the `git log` / commit-subject side.
- **C. Both.** Soft warning on title length (`title-too-long` finding, advisory); hard cap on slug length. Matches the kernel's existing pattern (errors are findings; verbs pre-flight; opt-out via `--allow-long-title`).

Lean is C — pairs with the existing `slug-dropped-chars` warning class, and matches the verb hygiene contract pattern from ADR-0005 (pre-flight + atomic completeness + opt-out flag).

## Why it matters

The title is on the read path for every channel an operator or AI assistant uses to inspect the planning tree: `aiwf list`, `aiwf status`, `aiwf show`, `aiwf render`, `aiwf history`, file completion, and git log. When titles are uncapped, every channel inherits the same readability degradation — and the degradation compounds with tree size. A repo with 100 entities at 200-char average titles is unreadable in any tabular surface; a repo at 50-char average titles isn't.

The pattern is the same shape as G-0080 (wide-table verbs wrap mid-row): the symptom appears in the rendering surfaces, but the root cause is upstream in the data. G-0080 mitigates by making the *renderer* TTY-aware; this gap proposes preventing the data from getting that wide in the first place. The two are complementary — render mitigations help users on narrow terminals even when titles are short; title caps help every surface even when the renderer is dumb.

Tying this to the existing kernel surface:

- **`slug-dropped-chars` already exists** as a slug-quality warning at `aiwf add` time (e.g., when a title contains a `→`). A `slug-too-long` or `title-too-long` warning is the same family of finding — a natural extension, not a new mechanism.
- **ADR-0005 (verb hygiene contract, proposed)** specifies that verbs pre-flight against known finding rules; a title-length pre-flight in `aiwf add` is exactly the shape ADR-0005 names.
- **Renderer-side mitigation** (G-0080) is in scope for the wide-table surface even after this gap closes; the two layers are belt + suspenders.

This is a small, well-scoped fix. The judgement question is which policy (A / B / C) and what the cap value is; both can be settled in a `wf-patch`-sized milestone or a single `wf-patch` ritual without an epic wrapper.

## Related

- G-0080 — wide-table verbs wrap mid-row; renderer-side companion to this gap.
- ADR-0005 — verb hygiene contract; the pre-flight obligation this gap's fix would implement.
- `slug-dropped-chars` finding (existing) — same finding family this would extend.
