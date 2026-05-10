---
id: G-0086
title: docs/pocv3/contracts.md still references non-existent aiwf list contracts (lines 98, 114-117); same drift class as G-0061/G-0085, different file
status: addressed
discovered_in: M-0072
addressed_by_commit:
    - c3778ce
---

## What's missing

`docs/pocv3/contracts.md` retains five references to the dead `aiwf list contracts` form (lines 98, 114, 115, 116, 117). Sibling drift to G-0061 (closed by E-0020/M-0072) and G-0085 (closed by E-0020/M-0074), but in a third file that M-0072's AC-8 named scope (`docs/pocv3/plans/contracts-plan.md` + `internal/skills/embedded/aiwf-contract/SKILL.md`) does not cover.

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

1. **Verb shape:** `aiwf list contracts` (positional `contracts`) does not exist; the canonical form is `aiwf list --kind contract` (per M-0072's V1).
2. **Speculative future flags:** `--filter`, `--drifted`, `--verified-status`, `--linked-adr`, `--untouched-since` are aspirational. M-0072's V1 axes are `--kind`, `--status`, `--parent`, `--archived`, `--format`, `--pretty`; the rest are explicitly out-of-scope until concrete friction earns each one.

## Why it matters

Same kernel principle G-0061 named: *"kernel functionality must be AI-discoverable"* fails when the documentation surface points at non-existent commands. A reader (human or AI) following `docs/pocv3/contracts.md` to learn the contract surface lands on `aiwf: unknown command` errors.

The fix shape splits cleanly:

- **Verb-shape sweep** — mechanical replace `aiwf list contracts` → `aiwf list --kind contract` for the bare-form usages (line 98 and the basic registry-view example). Drop-in.
- **Speculative-flag disposition** — the `--filter`, `--drifted`, etc. lines describe a future-axes wish list. Either move to a "Future considerations" subsection clearly marked as not-yet-implemented, delete (since the V1 surface is explicit), or file each as its own AC under a future "extend aiwf list V2" milestone. Each speculative flag deserves its own scoping decision rather than a silent inclusion in the V1 sweep.

Out of M-0072 AC-8's named scope: filed as a follow-up so the discovery isn't lost. The kernel test `internal/policies/skill_coverage_test.go::TestNoReintroducedDeadVerbForms_ContractsAndSkill` covers M-0072's two named files; extending its `sites` list to include `docs/pocv3/contracts.md` is the natural close-step once the speculative-flag disposition is decided.

## References

- G-0061 — parent shape, closed by E-0020.
- G-0085 — sibling drift in CLAUDE.md + docs/pocv3 + a gap body, closed by E-0020/M-0074.
- M-0072 — ships `aiwf list` V1; AC-8's scope was deliberately narrow.
- `internal/policies/skill_coverage_test.go::TestNoReintroducedDeadVerbForms_ContractsAndSkill` — drift guard whose `sites` list is the natural home for the third file once this gap closes.
