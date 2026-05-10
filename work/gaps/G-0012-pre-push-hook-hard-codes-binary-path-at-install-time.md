---
id: G-0012
title: Pre-push hook hard-codes binary path at install time
status: addressed
addressed_by_commit:
  - 8ed5051
---

Resolved in commit `8ed5051` (fix(aiwf): G12 — aiwf doctor detects pre-push hook drift). Took option (b) from the proposed fix: hook content stays absolute-path (preserves the existing rationale that hooks shouldn't depend on the user's interactive PATH at push time), and `aiwf doctor` now reads `.git/hooks/pre-push` and reports drift. Five distinct states surface in the output (`ok`, `missing`, `stale path`, `not aiwf-managed`, `malformed`) and stale/missing/malformed increment the problem count so doctor exits non-zero. Re-running `aiwf init` is the documented remediation. Tests cover ok / stale / missing.

---
