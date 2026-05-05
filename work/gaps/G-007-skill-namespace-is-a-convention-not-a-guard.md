---
id: G-007
title: Skill namespace is a convention, not a guard
status: addressed
---

Resolved in commit `971fa88` (fix(aiwf): G7 — track skill ownership via on-disk manifest). Materialize now reads `.claude/skills/.aiwf-owned`, wipes only directories listed in the prior manifest that are no longer in the current embed, writes the embedded skills, and updates the manifest. Foreign directories — including any future `aiwf-rituals-*` plugin — are left alone, even when they share the prefix. The manifest path is added to `MaterializedPaths` so the existing `aiwf init` gitignore step covers it. Tests cover the load-bearing "third-party prefix-sharing dir survives update" scenario plus the regression that real cleanup still works when the prior manifest claims ownership. Manual smoke verified: `aiwf-rituals-tdd/` content survives `aiwf update` byte-for-byte.

---
