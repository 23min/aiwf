---
id: G-010
title: macOS case-insensitive filesystem assumption
status: addressed
---

Resolved in commit `8950874` (fix(aiwf): G10 — surface case-equivalent paths and FS case-sensitivity). New `check.casePaths` validator flags any pair of entity paths that differ only in case (severity error), so a Linux-committed `E-01-foo` + `E-01-Foo` collision is caught at validation time before silently collapsing on macOS reviewer machines. `aiwf doctor` gains a "filesystem: case-sensitive | case-insensitive" line probed via temp-file + uppercased-stat. README's new "Known limitations" section documents the case-sensitivity contract alongside concurrent-invocation, validator-availability, and Unix-only scope.

---
