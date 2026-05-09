---
id: G-092
title: No documented hierarchy of doc authority across docs/; LLMs and humans cannot tell normative from exploratory from archival without reading every file
status: open
---
The `docs/` tree carries qualitatively different artefacts: kernel-normative design records (CLAUDE.md sister docs, `design-decisions.md`, ADRs); exploratory / proposal / synthesis documents (`docs/explorations/`, the various `*-plan.md` files); and historical-archive content (`docs/pocv3/archive/`, the migration text records). The `docs/` path alone does not signal which class a given file belongs to, and **there is no documented hierarchy of doc authority** that an LLM or a new contributor can consult before building their working model of the project.

## Existing state

- **CLAUDE.md** is the single canonically-normative document for the repo — it carries the engineering principles, what aiwf commits to, the Go conventions, and the enforcement table. Its authority is implicit (every Claude Code session loads it as a system instruction) but never declared in prose anyone can cite.
- **ADRs** under `docs/adr/` are normative architectural records. ADR-0004 (accepted; this gap's filing parent) is an example of one whose reasoning binds future kernel work. Authority is structural — they're ADRs — but not declared elsewhere as authoritative.
- **`docs/pocv3/design/*.md`** — `design-decisions.md`, `provenance-model.md`, `tree-discipline.md`, etc. — are deeply load-bearing for understanding aiwf's principles. CLAUDE.md cites them but doesn't formally rank them as second-tier-after-CLAUDE.md.
- **`docs/pocv3/plans/*.md`** are forward-looking design documents. Some have shipped (their plans are now baked into the kernel); others remain proposals. There is no marker indicating "this plan has shipped — read the kernel for current truth" vs "this plan is still active design intent."
- **`docs/explorations/*.md`** are explicitly non-normative thinking documents — but only a reader who notices the directory name and infers its meaning will treat them that way. The recently-added `06-tdd-diagnostic.md` and `07-tdd-architecture-proposal.md` are a diagnostic + proposal pair, not approved direction; their non-normative status is captured only in their own preambles.
- **`docs/pocv3/archive/`** is a text-record archive (pre-migration session notes, `gaps-pre-migration.md`). Marked as archive by its directory name but not documented as such in any index.

ADR-0004 names this issue inline in its Decision section: *"The broader question of which `docs/` trees are normative vs. exploratory vs. archival deserves its own treatment in a separate gap."* This is that gap.

## Why it matters

Three failure modes:

1. **LLM context inflation and drift weight.** A Claude Code session reading `docs/` to understand a problem domain has no signal about which files to weight as authoritative. An exploratory diagnostic in `docs/explorations/` may carry equal interpretive weight as a load-bearing design doc in `docs/pocv3/design/`, because both render the same when read. ADR-0004's forget-by-default principle applies to *archived entities* under `work/` — but not to docs that *function* as exploratory or archival but live in non-archive directories.

2. **New-contributor onboarding.** A human contributor cloning the repo and skimming `docs/` builds a mental model that may double-count exploratory content or miss normative content. The `pocv3/` directory naming is itself an artefact of the framework's "PoC v3" history; G-074 and G-075 already track its renaming and prose-framing drift. Without a documented hierarchy, even an updated naming scheme doesn't tell readers which docs to treat as truth.

3. **Drift-checking discipline is per-file, not per-tier.** Some docs are kept in lockstep with code changes (the lychee link-check workflow verifies references; `aiwf check` validates entity bodies). Other docs aren't drift-checked at all (`docs/explorations/`, planning docs that have shipped). Today the only signal that a doc is drift-checked is whether someone notices its references break — there's no upfront declaration "this doc is normative; if it drifts from the code, that's a bug." G-061, G-085, and G-086 are concrete instances of this drift class.

ADR-0004 addresses entity-archive forget-by-default. The `docs/` analogue — declaring which trees are normative, which are exploratory, which are archival — would extend the same forget-by-default discipline to documentation, giving LLMs and humans an upfront reading map rather than a per-file scavenger hunt.

## Fix shape

Three layers, candidate combinations:

1. **CLAUDE.md gains a "Documentation hierarchy" section.** A short table or list naming the trees by authority tier:
   - **Normative:** CLAUDE.md, `docs/adr/`, `docs/pocv3/design/`.
   - **Forward-looking design / plans:** `docs/pocv3/plans/` (with caveats about which have shipped).
   - **Exploratory / non-normative:** `docs/explorations/`, `docs/notes/`.
   - **Archival:** `docs/pocv3/archive/`.
   The list is small enough to fit in a paragraph; CLAUDE.md is already the canonical "what the project commits to" doc.

2. **Per-tree marker file (`_AUTHORITY.md` or similar).** Each `docs/` subtree carries a one-paragraph file declaring its tier and drift-check expectations. Cheap to add; survives directory renames better than a single CLAUDE.md table; can be parsed mechanically if a future drift-check rule wants to know which trees to enforce.

3. **`aiwf check` rule for normative-tree drift.** Once the hierarchy is declared, the kernel can mechanically check that normative-tree files don't reference removed code paths or stale entities. Builds on the existing G-061 / G-085 / G-086 drift-class findings. Out of scope for the immediate fix but the natural follow-on.

(1) is the minimum — write the table, ratify, done. (2) is a structural improvement that makes the hierarchy survive prose churn. (3) is the kernel-level enforcement that earns CLAUDE.md's "framework correctness must not depend on LLM behavior" standard.

## Out of scope

- **Renaming `docs/pocv3/`.** Tracked separately by G-075.
- **Sweeping pocv3 prose framing.** Tracked by G-074.
- **Drift-checking of currently-stale `docs/pocv3/contracts.md`.** Tracked by G-086.
- **Per-doc deprecation markers** ("this plan has shipped — see kernel for current truth"). Useful but a different concern; could fold into (2) above or get its own gap if pursued separately.

## References

- **ADR-0004** — Uniform archive convention for terminal-status entities. Inline-flagged the doc-authority question as needing its own gap; this is that gap.
- **CLAUDE.md** "What aiwf commits to" §6 (layered location-of-truth) — describes the engine/policy/state separation but not the documentation-tier separation.
- **G-074** — `docs/pocv3/` body prose still uses PoC framing; needs sweep.
- **G-075** — `docs/pocv3/` directory naming is now historical; rename or accept.
- **G-086** — `docs/pocv3/contracts.md` still references non-existent `aiwf list contracts`.
- `docs/explorations/06-tdd-diagnostic.md`, `docs/explorations/07-tdd-architecture-proposal.md` — recently-added exploratory docs whose non-normative status is signaled only by their own preambles.
