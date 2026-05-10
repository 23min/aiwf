---
id: G-0021
title: Kernel surface is partially undocumented for AI assistants
status: addressed
addressed_by_commit:
  - 5a7df46
  - 351e694
---

Resolved across commits `5a7df46` (docs(aiwf): document case-paths and load-error in aiwf-check skill) and `351e694` (feat(aiwf): extend discoverability policy from provenance-* to all codes).

A six-axis audit (verbs, flags, finding codes, trailer keys, body-section names, YAML fields) against the four CLAUDE.md-named documentation channels (`aiwf <verb> --help`, embedded skills under `.claude/skills/aiwf-*`, CLAUDE.md / CLAUDE.md, and any markdown under `docs/pocv3/`) found:

- **Verbs (22), flags (40+), body sections (18), YAML fields (20+):** every item documented in at least one channel; zero gaps.
- **Trailer keys (15):** zero gaps. `aiwf-prior-parent` is mentioned in `design-lessons.md` (reachable via `CLAUDE.md`); the rest are in printHelp or `provenance-model.md`.
- **Finding codes (18 active across `check/` and `contractcheck/`):** two genuinely undocumented — `case-paths` and `load-error`. Both inline string literals (not named constants), invisible to the prior `provenance-*`-scoped policy. Added to the `aiwf-check` skill's errors table.

The `PolicyFindingCodesAreDiscoverable` policy in `internal/policies/discoverability.go` was extended in two ways so this gap can't reopen unnoticed:

1. *Code enumeration* expanded from "named `provenance-*` constants" to "every kebab-case finding code anywhere," via a new `loadCheckCodeLiterals` AST walk over `Finding{Code: "..."}` literals across `check/` and `contractcheck/`.
2. *Channel set* expanded from "`aiwf-check` skill + `main.go`" to the full CLAUDE.md set: every embedded skill, `main.go`, both `CLAUDE.md` files, and every markdown under `docs/pocv3/`.

`go test ./internal/policies/...` is the CI-enforced safety net: any new finding code added without a documentation mention fails `TestPolicy_FindingCodesAreDiscoverable` before merge.

The audit's other-axes verification was a one-shot pass; if a future iteration adds a new axis (e.g., a JSON envelope field schema), the audit-and-policy pattern from this gap is the template.

---

<a id="g24"></a>
