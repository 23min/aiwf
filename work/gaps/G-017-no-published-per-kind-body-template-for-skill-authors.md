---
id: G-017
title: No published per-kind body template for skill authors
status: addressed
addressed_by_commit:
  - f4a0fae
---

Resolved in commit `f4a0fae` (fix(aiwf): G17 — add 'aiwf template' verb, completes the per-kind contract surface). Took the proposed approach: a read-only `aiwf template [kind]` verb mirrors `aiwf schema`. With no kind, emits every kind separated by `KIND: <kind>` headers. With a kind, emits just that template raw and unprefixed, so `aiwf template epic > new_epic_body.md` works as a one-liner. Standard `--format=text|json [--pretty]` envelope. JSON shape: `{result: {templates: [{kind, body}]}}`. Reads from `entity.BodyTemplate` (already exported); no internal data move required. Together with `aiwf schema`, this completes the published per-kind contract that AI scaffolders need to author files outside the `aiwf add` path. Coverage: 85.3% on `runTemplate`, 80% on `writeTemplateText`.

Resolved in commit `9486046` (fix(aiwf): G16 — add id-path-consistent check to catch silent path/id drift). Took the proposed approach: a new `idPathConsistent` check iterates `tree.Entities`, derives the expected id from each path via `entity.IDFromPath`, and emits an error finding on disagreement. Stubs are skipped (constructed from path-derived id by construction). Defensive: if `IDFromPath` returns false for an entity PathKind accepted (impossible by construction), the entity is skipped rather than panicked on. Hint table entry points the user at `aiwf reallocate` for renumbering (rewrites both sides + updates references atomically), `aiwf rename` for slug-only drift, or hand-correction when the user knows which side is right. Pinned by a new fixture file at `internal/check/testdata/messy/work/epics/E-01-orig/M-099-path-id-mismatch.md` (path encodes M-099, frontmatter says M-100) — `TestFixture_Messy` now asserts the new code appears alongside the existing ten. Coverage: 100% on `idPathConsistent`. Completes the path-vs-frontmatter story G14's stub mechanism implicitly relied on.

---
