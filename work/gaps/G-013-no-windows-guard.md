---
id: G-013
title: No Windows guard
status: addressed
---

Resolved in commit `dda370d` (fix(aiwf): G13 — refuse Windows up front with one clear message). Took both halves of the proposed fix: (a) `cmd/aiwf` gained `assertSupportedOS` called at the top of `main`, exiting 2 with a clear message on `runtime.GOOS == "windows"`; (b) `repolock` got a Windows stub (`repolock_windows.go`) so the package cross-compiles on Windows — without it, `syscall.Flock undefined` was exactly the deep-stack confusion the gap was filed against. Verified `GOOS=windows go build` produces a clean PE32+ binary that fires the assertSupportedOS message on first run. README's Known Limitations section (added in G10) already documents the Unix-only stance.

---
