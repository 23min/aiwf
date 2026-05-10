---
id: G-0014
title: Parse failure cascades into refs-resolve findings
status: addressed
addressed_by_commit:
  - e2a39ee
  - d9a726c
---

Resolved in commit `e2a39ee` (fix(aiwf): G14 — register stub for unparseable entity to suppress refs-resolve cascade). Took the proposed approach: on parse (or read) failure the loader derives the entity's id from its path via the new `entity.IDFromPath` and registers a stub in `tree.Tree.Stubs`; `refsResolve` indexes Stubs alongside Entities so referrers resolve cleanly; `idsUnique` consults Stubs too so duplicate-id collisions involving stubs are still flagged. End-to-end `TestFixture_ProliminalCascadeEndToEnd` reproduces the wild proliminal.net case (E-0001 + 12 referrers) and confirms the 13→1 reduction. Verb-level `TestAdd_GapDiscoveredInStubbedEntity` confirms `Tree.Stubs` propagates through `projectAdd`'s shallow copy into the projection check, so verbs adding a referrer to a stubbed entity are not blocked. Coverage on changed code: 100% on `idsUnique`, `refsResolve`, `registerStub`; 89.5% on `IDFromPath`. Upstream skill fix in `ai-workflow-rituals` `d9a726c` removed the wrap-epic instruction that originally triggered this in the wild.

---
