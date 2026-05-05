---
id: G-040
title: '`work/` is mechanically unprotected — `aiwf check` silently ignores stray files'
status: addressed
addressed_by_commit:
  - bdd43c2
---

Resolved in commit `(this commit)` (feat(aiwf): G40 — tree-discipline check + tree.allow_paths config + skill rule). The tree loader at `internal/tree/tree.go` walks `work/*` subdirectories and registers everything `entity.PathKind` recognizes; until this fix, files at *any* other path were silently skipped — no finding, no warning, no log line. An LLM-written `work/scratch.md`, an accidental `work/epics/E-01-foo/notes.md`, or a leftover `work/old-stuff/` directory was invisible to `aiwf check` and therefore invisible to the pre-push hook. The chokepoint that was supposed to make tree-shape guarantees real had a blind spot the size of the whole `work/` tree.

The fix has three pieces, all in this commit:

1. **Mechanical layer** — `tree.Tree` gains a `Strays []string` field populated during the walk for `work/*` subtrees (docs/adr/ stays permissive). New `check.TreeDiscipline(t, allow, strict) []Finding` filters strays through (a) auto-exempt for files inside any recognized contract directory, (b) user allow-list via `aiwf.yaml: tree.allow_paths` glob list, and emits `unexpected-tree-file` findings for the rest. Severity is **warning** by default; **error** when `aiwf.yaml: tree.strict: true` (the pre-push hook then blocks the push). Wired into `runCheck` after the standard rule chain so render/status callers don't get tree-discipline noise on every read.
2. **AI-discoverable layer** — `aiwf-add` SKILL gains a "Tree discipline" subsection naming the rule and the failure mode; `aiwf-check` SKILL gains the new finding code in its warnings table. **No new skill** — folded into existing skills to avoid skill sprawl per the user's directive.
3. **Doctrine layer** — new `docs/pocv3/design/tree-discipline.md` records the canonical path shapes, the "verbs own tree shape; body prose can be edited directly" rule, the `aiwf.yaml: tree.*` configuration surface, and the explicit decision *not* to add an `aiwf edit` verb (YAGNI; revisit if untrailered-body-edit audit warnings become noisy).

What's allowed without a verb: **body-prose edits to existing entity files**, with the resulting untrailered commit reconciled by `aiwf adopt` (the existing G24 surface). What's not allowed: any *new* file under `work/` outside the six recognized path shapes (epic/milestone/gap/decision/contract/ADR), and any change to an existing entity file's frontmatter without going through the appropriate verb. The mechanical check is the guarantee per the kernel principle "framework correctness must not depend on the LLM's behavior"; the skill is the convenient version, not the load-bearing one.

The check enforces this in the consumer repo. The kernel's own `CLAUDE.md` is *not* aiwf's responsibility — aiwf ships the embedded skill (gitignored, refreshed on `aiwf update`) and the check; the consumer's hand-written `CLAUDE.md` is theirs alone. This matches the existing kernel principle "marker-managed framework artifacts" — aiwf does not write to consumer-authored files.

The discoverability test policies (G21 / G26 lineage) caught both the new finding code and the new yaml field at first run, exactly as designed — the implementation could not land without the doctrine + skill updates. That's the policy framework working: if you add a kernel surface and forget to make it discoverable, the lint refuses to pass.

Severity: **High**. The blind spot was load-bearing (the pre-push hook is the chokepoint, and the chokepoint was leaking). No reported wedge, but the user discovered it via a real LLM-mistake under `work/` in a consumer repo before any kernel guarantee fired.

---
