---
id: G-086
title: docs/pocv3/contracts.md still references non-existent aiwf list contracts (lines 98, 114-117); same drift class as G-061/G-085, different file
status: open
discovered_in: M-072
---

## What's missing

`docs/pocv3/contracts.md` retains five references to the dead `aiwf list contracts` form (lines 98, 114, 115, 116, 117). Sibling drift to G-061 (closed by E-20/M-072) and G-085 (closed by E-20/M-074), but in a third file that M-072's AC-8 named scope (`docs/pocv3/plans/contracts-plan.md` + `internal/skills/embedded/aiwf-contract/SKILL.md`) does not cover.

The references appear in a "5. Verbs" section of the design doc:

```
aiwf list contracts [--filter ...]   # registry view
...
aiwf list contracts --drifted               # live_source missing or stale
aiwf list contracts --verified-status fail  # last verify failed
aiwf list contracts --linked-adr ADR-0042   # everything that ADR created
aiwf list contracts --untouched-since 30d   # candidates for review
```

Two coupled issues:

1. **Verb shape:** `aiwf list contracts` (positional `contracts`) does not exist; the canonical form is `aiwf list --kind contract` (per M-072's V1).
2. **Speculative future flags:** `--filter`, `--drifted`, `--verified-status`, `--linked-adr`, `--untouched-since` are aspirational. M-072's V1 axes are `--kind`, `--status`, `--parent`, `--archived`, `--format`, `--pretty`; the rest are explicitly out-of-scope until concrete friction earns each one.

## Why it matters

Same kernel principle G-061 named: *"kernel functionality must be AI-discoverable"* fails when the documentation surface points at non-existent commands. A reader (human or AI) following `docs/pocv3/contracts.md` to learn the contract surface lands on `aiwf: unknown command` errors.

The fix shape splits cleanly:

- **Verb-shape sweep** — mechanical replace `aiwf list contracts` → `aiwf list --kind contract` for the bare-form usages (line 98 and the basic registry-view example). Drop-in.
- **Speculative-flag disposition** — the `--filter`, `--drifted`, etc. lines describe a future-axes wish list. Either move to a "Future considerations" subsection clearly marked as not-yet-implemented, delete (since the V1 surface is explicit), or file each as its own AC under a future "extend aiwf list V2" milestone. Each speculative flag deserves its own scoping decision rather than a silent inclusion in the V1 sweep.

Out of M-072 AC-8's named scope: filed as a follow-up so the discovery isn't lost. The kernel test `internal/policies/skill_coverage_test.go::TestNoReintroducedDeadVerbForms_ContractsAndSkill` covers M-072's two named files; extending its `sites` list to include `docs/pocv3/contracts.md` is the natural close-step once the speculative-flag disposition is decided.

## References

- G-061 — parent shape, closed by E-20.
- G-085 — sibling drift in CLAUDE.md + docs/pocv3 + a gap body, closed by E-20/M-074.
- M-072 — ships `aiwf list` V1; AC-8's scope was deliberately narrow.
- `internal/policies/skill_coverage_test.go::TestNoReintroducedDeadVerbForms_ContractsAndSkill` — drift guard whose `sites` list is the natural home for the third file once this gap closes.
