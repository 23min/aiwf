---
id: G-0291
title: Relocate gitleaks secret-scan from pre-commit to pre-push
status: open
discovered_in: E-0045
---
## Problem

The gitleaks secret/path-leak scan (G-0103) runs as a **pre-commit** hook, firing on **every** commit — human edits and aiwf-verb commits alike. A secret or contributor-path is not actually exposed until the commit is **pushed**, so pre-commit is the wrong trust boundary: pre-push is where the content leaves the machine, and where the scan belongs. Running it pre-commit instead just taxes every commit's latency.

Combined with the pre-commit policy-lint `go test ./internal/policies/` (~70s — sibling gap `G-0280`, currently on the E-0044 branch) and the post-commit STATUS.md regen (G-0112), per-commit hook work makes a single mutating verb cost ~75–110s on this repo (measured 2026-06-27 while creating E-0045's entities on trunk).

## Relationship to existing work

- **ADR-0022 / M-0186 already decide "gitleaks relocates to pre-push"** — but only as a *consequence of the verb-commit plumbing migration* (`commit-tree` fires no pre-commit hooks, so verb commits need leak coverage relocated to pre-push). That decision is **verb-commit-scoped**. This gap is the **general hook-stage move**: gitleaks should be pre-push for *all* commits, so human commits stop paying it at pre-commit too. The implementation is likely shared (move the gitleaks hook stage once), but the general case deserves an explicit home so it is not lost if M-0186 is implemented narrowly.
- Sibling per-commit-overhead slice: the pre-commit policy-lint go-test (`G-0280`) — gate it on staged Go changes. Together these two remove the bulk of per-commit latency.
- G-0103 is the gitleaks chokepoint this relocates; G-0112 (STATUS.md post-commit regen) is the third per-commit cost.

## Direction

- Move the gitleaks invocation from the pre-commit hook to the **pre-push** hook, scanning the pushed commit range (`gitleaks git --log <range>`) rather than `--staged`.
- Preserve the guarantee: a leak is still blocked **before it leaves the machine** — pre-push is the real trust boundary.
- Reconcile with ADR-0022 / M-0186 so the verb-commit relocation and the general relocation are **one** mechanism, not two divergent ones.

## Why pre-push is the right boundary

A secret in a local, unpushed commit is not yet exposed; the push is the exposure event. Scanning at pre-push catches it at the actual trust boundary while removing per-commit latency — the same argument as ADR-0022's Option C, generalized from verb commits to every commit.

## Provenance

Surfaced 2026-06-27 while creating E-0045's planning entities on trunk: each `aiwf add` took ~75–110s, dominated by per-commit hook work. The gitleaks→pre-push remedy was discussed as ADR-0022's Option C but scoped to verb commits only; this gap captures the general case.
