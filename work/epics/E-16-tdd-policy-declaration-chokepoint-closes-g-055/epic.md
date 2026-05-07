---
id: E-16
title: TDD policy declaration chokepoint (closes G-055)
status: proposed
---

## Goal

Make every milestone's TDD policy an explicit, recorded choice at creation time. Today, `aiwf add milestone` has no `--tdd` flag and absence of the field silently maps to `tdd: none`, so an LLM (or human) following the `aiwf-add` skill faithfully produces a code milestone with no TDD tracking. This violates the kernel's "framework correctness must not depend on LLM behavior" principle. See [G-055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) for the empirical evidence (E-14's M-049..M-055 all created with no TDD, M-061 reproducing the pattern this week).

End state: `aiwf add milestone --tdd required|advisory|none` is the chokepoint; `aiwf.yaml: tdd.default: required` is the project-level fallback shipped by `aiwf init`; `aiwf update` migrates existing consumer repos with loud output so the policy shift is visible exactly when it lands.

## Scope

- `--tdd` flag on `aiwf add milestone` with closed-set values, tab-completable, resolved at creation time and written to milestone frontmatter as the per-milestone source of truth.
- `aiwf.yaml: tdd.default` schema field (closed set: `required | advisory | none`).
- `aiwf init` seeds `tdd.default: required` into freshly-created `aiwf.yaml` with an explanatory comment.
- `aiwf update` idempotently migrates existing `aiwf.yaml` files lacking the field, preserving comments and key order, with loud text + JSON envelope output.
- `aiwf check` finding `milestone-tdd-undeclared` (warning by default; error under `aiwf.yaml: tdd.strict: true`) as defense-in-depth against hand-edits and import paths that bypass the verb chokepoint.
- `aiwf-add` and `aiwf-check` skills updated to document the new flag, the resolution order, the project default, and the new finding code.

## Out of scope

- **Promote-time guard.** Refusing `draft -> in_progress` transitions when `tdd:` is absent is overkill once the creation chokepoint and the check-finding backstop are in place. YAGNI; revisit if drift shows up empirically.
- **Retroactive audit of historical milestones.** Existing milestones with no `tdd:` field stay grandfathered as `tdd: none`. The new check finding surfaces them for visibility but does not retroactively engage `acs-tdd-audit` against their already-met ACs. The grandfather rule is load-bearing for not lighting up E-14's tree on upgrade.
- **`--tdd-reason` flag** for explaining `--tdd none` choices. Mentioned in G-055 as a deferrable refinement; not in this epic.
- **Changes to the AC TDD-phase FSM** (`red | green | refactor | done`) or the `acs-tdd-audit` rule. Both already work correctly when the parent milestone's `tdd:` field is set; this epic just makes sure the field gets set.
